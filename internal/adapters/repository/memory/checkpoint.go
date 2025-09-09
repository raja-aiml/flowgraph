// Package adapters provides concrete implementations of checkpoint storage
package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/flowgraph/flowgraph/pkg/serialization"
)

// InMemorySaver implements checkpoint.Saver with thread-safe in-memory storage
// PRINCIPLES:
// - KISS: Simple in-memory map with proper concurrency
// - SRP: Single responsibility for in-memory checkpoint storage
// - DIP: Implements checkpoint.Saver interface
type InMemorySaver struct {
	// Use sync.Map for concurrent access
	checkpoints sync.Map
	// TTL management
	ttlMap     sync.Map
	defaultTTL time.Duration
	// Memory management
	maxMemoryMB int64
	currentSize int64
	sizeMutex   sync.RWMutex
	// LRU eviction
	accessOrder map[string]time.Time
	orderMutex  sync.RWMutex
	// Serialization
	serializer *serialization.Serializer
	// Cleanup
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	cleanupOnce   sync.Once
}

// InMemoryConfig holds configuration for InMemorySaver
type InMemoryConfig struct {
	DefaultTTL      time.Duration             // Default TTL for checkpoints
	MaxMemoryMB     int64                     // Maximum memory usage in MB
	CleanupInterval time.Duration             // Cleanup interval for expired items
	Serializer      *serialization.Serializer // Custom serializer (optional)
}

// checkpointEntry holds checkpoint data with metadata
type checkpointEntry struct {
	checkpoint *checkpoint.Checkpoint
	data       []byte // Serialized data
	size       int64  // Size in bytes
	createdAt  time.Time
	accessedAt time.Time
}

// NewInMemorySaver creates a new in-memory checkpoint saver
// PRINCIPLES:
// - KISS: Simple constructor with sensible defaults
// - YAGNI: Only required configuration
func NewInMemorySaver(config InMemoryConfig) *InMemorySaver {
	// Set defaults
	if config.DefaultTTL == 0 {
		config.DefaultTTL = 24 * time.Hour
	}
	if config.MaxMemoryMB == 0 {
		config.MaxMemoryMB = 1024 // 1GB default
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 5 * time.Minute
	}
	if config.Serializer == nil {
		config.Serializer = serialization.DefaultSerializer()
	}

	saver := &InMemorySaver{
		defaultTTL:  config.DefaultTTL,
		maxMemoryMB: config.MaxMemoryMB,
		serializer:  config.Serializer,
		accessOrder: make(map[string]time.Time),
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	saver.startCleanup(config.CleanupInterval)

	return saver
}

// Save stores a checkpoint in memory
// PRINCIPLES:
// - KISS: Simple save operation with validation
// - SRP: Only saves, doesn't validate business logic
func (s *InMemorySaver) Save(_ context.Context, cp *checkpoint.Checkpoint) error {
	if err := cp.Validate(); err != nil {
		return fmt.Errorf("checkpoint validation failed: %w", err)
	}

	// Serialize checkpoint
	data, err := s.serializer.Serialize(cp)
	if err != nil {
		return fmt.Errorf("checkpoint serialization failed: %w", err)
	}

	size := int64(len(data))

	// Check memory limits before storing
	if err := s.checkMemoryLimit(size); err != nil {
		return err
	}

	now := time.Now()
	entry := &checkpointEntry{
		checkpoint: cp,
		data:       data,
		size:       size,
		createdAt:  now,
		accessedAt: now,
	}

	// Store checkpoint
	s.checkpoints.Store(cp.ID, entry)

	// Set TTL
	s.ttlMap.Store(cp.ID, now.Add(s.defaultTTL))

	// Update access order and memory tracking
	s.orderMutex.Lock()
	s.accessOrder[cp.ID] = now
	s.orderMutex.Unlock()

	s.sizeMutex.Lock()
	s.currentSize += size
	s.sizeMutex.Unlock()

	return nil
}

// Load retrieves a checkpoint from memory
func (s *InMemorySaver) Load(_ context.Context, id string) (*checkpoint.Checkpoint, error) {
	value, exists := s.checkpoints.Load(id)
	if !exists {
		return nil, checkpoint.ErrCheckpointNotFound
	}

	entry, ok := value.(*checkpointEntry)
	if !ok {
		return nil, fmt.Errorf("invalid checkpoint entry type")
	}

	// Check TTL
	if s.isExpired(id) {
		s.delete(id)
		return nil, checkpoint.ErrCheckpointNotFound
	}

	// Update access time for LRU
	now := time.Now()
	entry.accessedAt = now

	s.orderMutex.Lock()
	s.accessOrder[id] = now
	s.orderMutex.Unlock()

	// Deserialize checkpoint
	var cp checkpoint.Checkpoint
	if err := s.serializer.Deserialize(entry.data, &cp); err != nil {
		return nil, fmt.Errorf("checkpoint deserialization failed: %w", err)
	}

	return &cp, nil
}

// List returns checkpoints matching the filter
func (s *InMemorySaver) List(ctx context.Context, filter checkpoint.Filter) ([]*checkpoint.Checkpoint, error) {
	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("filter validation failed: %w", err)
	}

	var results []*checkpoint.Checkpoint
	matchCount := 0
	resultCount := 0

	s.checkpoints.Range(func(key, value interface{}) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}

		// Check TTL
		if s.isExpired(id) {
			s.delete(id)
			return true
		}

		entry, ok := value.(*checkpointEntry)
		if !ok {
			return true
		}

		cp := entry.checkpoint

		// Apply filters
		if filter.GraphID != "" && cp.GraphID != filter.GraphID {
			return true
		}
		if filter.ThreadID != "" && cp.ThreadID != filter.ThreadID {
			return true
		}
		if filter.Since != nil && cp.Timestamp.Before(*filter.Since) {
			return true
		}

		// Apply offset
		if filter.Offset > 0 && matchCount < filter.Offset {
			matchCount++
			return true
		}

		// Apply limit
		if filter.Limit > 0 && resultCount >= filter.Limit {
			return false
		}

		results = append(results, cp)
		matchCount++
		resultCount++
		return true
	})

	return results, nil
}

// Delete removes a checkpoint from memory
func (s *InMemorySaver) Delete(_ context.Context, id string) error {
	if _, exists := s.checkpoints.Load(id); !exists {
		return checkpoint.ErrCheckpointNotFound
	}

	s.delete(id)
	return nil
}

// GetStats returns memory usage statistics
type MemoryStats struct {
	Count              int64   `json:"count"`
	SizeMB             int64   `json:"size_mb"`
	MaxSizeMB          int64   `json:"max_size_mb"`
	UtilizationPercent float64 `json:"utilization_percent"`
}

func (s *InMemorySaver) GetStats() MemoryStats {
	s.sizeMutex.RLock()
	currentSize := s.currentSize
	maxSize := s.maxMemoryMB
	s.sizeMutex.RUnlock()

	count := int64(0)
	s.checkpoints.Range(func(_, value interface{}) bool {
		count++
		return true
	})

	// Convert bytes to MB (keep precision by using float64 calculation first)
	sizeMBFloat := float64(currentSize) / (1024 * 1024)
	sizeMB := int64(sizeMBFloat)

	// For small sizes, use a minimum of 1 if there's any data
	if sizeMB == 0 && currentSize > 0 {
		sizeMB = 1
	}

	maxSizeMB := maxSize
	var utilization float64
	if maxSizeMB > 0 {
		utilization = sizeMBFloat / float64(maxSizeMB) * 100
	}

	return MemoryStats{
		Count:              count,
		SizeMB:             sizeMB,
		MaxSizeMB:          maxSizeMB,
		UtilizationPercent: utilization,
	}
}

// Close stops the cleanup goroutine and releases resources
func (s *InMemorySaver) Close() error {
	s.cleanupOnce.Do(func() {
		close(s.stopCleanup)
		if s.cleanupTicker != nil {
			s.cleanupTicker.Stop()
		}
	})
	return nil
}

// Private helper methods

// startCleanup starts the cleanup goroutine for expired items
func (s *InMemorySaver) startCleanup(interval time.Duration) {
	s.cleanupTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-s.cleanupTicker.C:
				s.cleanupExpired()
			case <-s.stopCleanup:
				return
			}
		}
	}()
}

// cleanupExpired removes expired checkpoints
func (s *InMemorySaver) cleanupExpired() {
	var expiredKeys []string

	s.ttlMap.Range(func(key, value interface{}) bool {
		id, ok := key.(string)
		if !ok {
			return true
		}

		expiry, ok := value.(time.Time)
		if !ok {
			return true
		}

		if time.Now().After(expiry) {
			expiredKeys = append(expiredKeys, id)
		}

		return true
	})

	for _, id := range expiredKeys {
		s.delete(id)
	}
}

// isExpired checks if a checkpoint has expired
func (s *InMemorySaver) isExpired(id string) bool {
	value, exists := s.ttlMap.Load(id)
	if !exists {
		return false
	}

	expiry, ok := value.(time.Time)
	if !ok {
		return false
	}

	return time.Now().After(expiry)
}

// delete removes a checkpoint and updates memory tracking
func (s *InMemorySaver) delete(id string) {
	if value, exists := s.checkpoints.LoadAndDelete(id); exists {
		if entry, ok := value.(*checkpointEntry); ok {
			s.sizeMutex.Lock()
			s.currentSize -= entry.size
			s.sizeMutex.Unlock()
		}
	}

	s.ttlMap.Delete(id)

	s.orderMutex.Lock()
	delete(s.accessOrder, id)
	s.orderMutex.Unlock()
}

// checkMemoryLimit checks if adding new data would exceed memory limits
func (s *InMemorySaver) checkMemoryLimit(newSize int64) error {
	s.sizeMutex.RLock()
	currentSize := s.currentSize
	maxSize := s.maxMemoryMB * 1024 * 1024 // Convert MB to bytes
	s.sizeMutex.RUnlock()

	if currentSize+newSize > maxSize {
		// Try to free memory using LRU eviction
		freed := s.evictLRU((currentSize + newSize) - maxSize)
		if freed < (currentSize+newSize)-maxSize {
			return fmt.Errorf("memory limit exceeded: current=%dMB, max=%dMB",
				currentSize/(1024*1024), s.maxMemoryMB)
		}
	}

	return nil
}

// evictLRU evicts least recently used items to free the target amount of memory
func (s *InMemorySaver) evictLRU(targetBytes int64) int64 {
	s.orderMutex.Lock()
	defer s.orderMutex.Unlock()

	// Sort by access time (oldest first)
	type accessInfo struct {
		id         string
		accessTime time.Time
	}

	var items []accessInfo
	for id, accessTime := range s.accessOrder {
		items = append(items, accessInfo{id: id, accessTime: accessTime})
	}

	// Simple bubble sort for oldest items (KISS principle)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].accessTime.After(items[j].accessTime) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	freedBytes := int64(0)
	for _, item := range items {
		if freedBytes >= targetBytes {
			break
		}

		if value, exists := s.checkpoints.Load(item.id); exists {
			if entry, ok := value.(*checkpointEntry); ok {
				freedBytes += entry.size
				s.delete(item.id)
			}
		}
	}

	return freedBytes
}

// DefaultInMemorySaver creates an InMemorySaver with default configuration
func DefaultInMemorySaver() *InMemorySaver {
	return NewInMemorySaver(InMemoryConfig{})
}
