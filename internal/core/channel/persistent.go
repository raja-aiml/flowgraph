// Package channel provides persistent channel implementation
package channel

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "sync"
    "time"
)
import imetrics "github.com/flowgraph/flowgraph/internal/infrastructure/metrics"

// PersistentChannel provides disk-backed message passing
// PRINCIPLES:
// - KISS: Simple file-based persistence
// - SRP: Single responsibility - persistent message passing
// - Thread-safe: Uses proper synchronization
type PersistentChannel struct {
    dataDir     string
    indexFile   string
    messages    []persistentMessage
    closed      bool
    mu          sync.RWMutex
    timeout     time.Duration
    nextID      int64
    maxSize     int64
    currentSize int64
    syncWrites  bool
    // retention
    maxMessages int
    maxAge      time.Duration
    onFull      string
    stopCleanup chan struct{}
    cleanupWG   sync.WaitGroup
}

// persistentMessage represents a message stored on disk
type persistentMessage struct {
	ID       int64     `json:"id"`
	Message  Message   `json:"message"`
	Filename string    `json:"filename"`
	Created  time.Time `json:"created"`
}

// PersistentChannelConfig holds configuration for PersistentChannel
type PersistentChannelConfig struct {
    DataDir    string        // Directory to store message files
    MaxSizeMB  int64         // Maximum total size in MB
    Timeout    time.Duration // Default timeout for operations
    SyncWrites bool          // Whether to sync writes to disk
    // Retention policies
    MaxMessages     int           // Keep at most this many messages (0 = unlimited)
    MaxAge          time.Duration // Expire messages older than this (0 = unlimited)
    OnFull          string        // Behavior when exceeding MaxSizeMB: "reject" (default) or "evict_oldest"
    CleanupInterval time.Duration // If >0, run periodic compaction to enforce retention
}

// NewPersistentChannel creates a new persistent channel
func NewPersistentChannel(config PersistentChannelConfig) (*PersistentChannel, error) {
	if config.DataDir == "" {
		return nil, fmt.Errorf("data directory is required")
	}
	if config.MaxSizeMB <= 0 {
		config.MaxSizeMB = 100 // Default 100MB
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second // Default timeout
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

    pc := &PersistentChannel{
        dataDir:     config.DataDir,
        indexFile:   filepath.Join(config.DataDir, "index.json"),
        messages:    make([]persistentMessage, 0),
        timeout:     config.Timeout,
        maxSize:     config.MaxSizeMB * 1024 * 1024, // Convert to bytes
        syncWrites:  config.SyncWrites,
        maxMessages: config.MaxMessages,
        maxAge:      config.MaxAge,
        onFull:      config.OnFull,
        stopCleanup: make(chan struct{}),
    }

	// Load existing index
	if err := pc.loadIndex(); err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

    // Start background cleanup if requested
    if config.CleanupInterval > 0 {
        pc.cleanupWG.Add(1)
        go pc.cleanupLoop(config.CleanupInterval)
    }

    return pc, nil
}

// DefaultPersistentChannel creates a persistent channel with default settings
func DefaultPersistentChannel(dataDir string) (*PersistentChannel, error) {
    cfg := PersistentChannelConfig{DataDir: dataDir, MaxSizeMB: 100, Timeout: 30 * time.Second}
    if defaultRuntimeConfig.PersistentTimeout > 0 {
        cfg.Timeout = defaultRuntimeConfig.PersistentTimeout
    }
    return NewPersistentChannel(cfg)
}

// Send sends a message to the channel
func (c *PersistentChannel) Send(ctx context.Context, message Message) error {
	// Validate message first
	if err := message.Validate(); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if channel is closed
	if c.closed {
		return ErrChannelClosed
	}

    // Check size limits
    if c.currentSize >= c.maxSize {
        if c.onFull == "evict_oldest" {
            c.evictUntilUnderSize()
        } else {
            return fmt.Errorf("channel size limit exceeded")
        }
    }

    // Create persistent message
    c.nextID++
    filename := fmt.Sprintf("msg_%d.json", c.nextID)
    fullpath := filepath.Join(c.dataDir, filename)

	pm := persistentMessage{
		ID:       c.nextID,
		Message:  message,
		Filename: filename,
		Created:  time.Now(),
	}

    // Write message to file (durably if configured)
    if err := c.writeMessage(fullpath, message); err != nil {
        return fmt.Errorf("failed to write message: %w", err)
    }

	// Calculate file size
    if info, err := os.Stat(fullpath); err == nil {
        c.currentSize += info.Size()
    }

	// Add to index
	c.messages = append(c.messages, pm)

    // Enforce retention
    c.enforceMaxMessages()
    if c.onFull == "evict_oldest" && c.currentSize > c.maxSize {
        c.evictUntilUnderSize()
    }
    c.enforceMaxAge()

    // Save index
    if err := c.saveIndex(); err != nil {
        // Try to clean up the message file
        os.Remove(fullpath)
        return fmt.Errorf("failed to save index: %w", err)
    }

    imetrics.ChannelSent("persistent", 1)
    imetrics.ChannelSizeBytes("persistent", c.currentSize)

	return nil
}

// Receive receives a message from the channel
func (c *PersistentChannel) Receive(ctx context.Context) (Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if channel is closed
	if c.closed {
		return Message{}, ErrChannelClosed
	}

	// Check if we have messages
	if len(c.messages) == 0 {
		return Message{}, ErrChannelEmpty
	}

	// Get the first message (FIFO)
    pm := c.messages[0]
    fullpath := filepath.Join(c.dataDir, pm.Filename)

	// Read message from file
    message, err := c.readMessage(fullpath)
    if err != nil {
        return Message{}, fmt.Errorf("failed to read message: %w", err)
    }

	// Remove from index
	c.messages = c.messages[1:]

	// Delete file and update size
    if info, err := os.Stat(fullpath); err == nil {
        c.currentSize -= info.Size()
    }
    os.Remove(fullpath)

    // Save updated index
    if err := c.saveIndex(); err != nil {
        return Message{}, fmt.Errorf("failed to save index: %w", err)
    }

    imetrics.ChannelReceived("persistent", 1)
    imetrics.ChannelSizeBytes("persistent", c.currentSize)
    return message, nil
}

// Close closes the channel
func (c *PersistentChannel) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

	if c.closed {
		return nil // Already closed
	}

    c.closed = true
    close(c.stopCleanup)
    return c.saveIndex()
}

// Cleanup removes all message files and resets the channel
func (c *PersistentChannel) Cleanup() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove all message files
	for _, pm := range c.messages {
		filepath := filepath.Join(c.dataDir, pm.Filename)
		os.Remove(filepath)
	}

	// Reset state
	c.messages = c.messages[:0]
	c.currentSize = 0
	c.nextID = 0

	// Save empty index
	return c.saveIndex()
}

// Len returns the number of messages currently in the channel
func (c *PersistentChannel) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.messages)
}

// IsClosed returns whether the channel is closed
func (c *PersistentChannel) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Stats returns channel statistics
func (c *PersistentChannel) Stats() PersistentChannelStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return PersistentChannelStats{
		Length:       len(c.messages),
		SizeBytes:    c.currentSize,
		MaxSizeBytes: c.maxSize,
		DataDir:      c.dataDir,
		Closed:       c.closed,
	}
}

// writeMessage writes a message to a file
func (c *PersistentChannel) writeMessage(filepath string, message Message) error {
    data, err := json.Marshal(message)
    if err != nil {
        return err
    }
    // Write atomically: to temp then rename
    tmp := filepath + ".tmp"
    f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
    if err != nil {
        return err
    }
    if _, err := f.Write(data); err != nil {
        f.Close()
        return err
    }
    if c.syncWrites {
        if err := f.Sync(); err != nil {
            f.Close()
            return err
        }
    }
    if err := f.Close(); err != nil {
        return err
    }
    if err := os.Rename(tmp, filepath); err != nil {
        return err
    }
    if c.syncWrites {
        // Sync the directory entry after rename
        _ = syncDir(filepath)
    }
    return nil
}

// readMessage reads a message from a file
func (c *PersistentChannel) readMessage(filepath string) (Message, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return Message{}, err
	}

	var message Message
	if err := json.Unmarshal(data, &message); err != nil {
		return Message{}, err
	}

	return message, nil
}

// loadIndex loads the message index from disk
func (c *PersistentChannel) loadIndex() error {
    if _, err := os.Stat(c.indexFile); os.IsNotExist(err) {
        // Index doesn't exist, start fresh
        c.messages = make([]persistentMessage, 0)
        c.nextID = 0
        c.currentSize = 0
        return nil
    }

    data, err := os.ReadFile(c.indexFile)
    if err != nil {
        // Attempt recovery by scanning data directory
        return c.recoverIndex()
    }

	var index struct {
		Messages []persistentMessage `json:"messages"`
		NextID   int64               `json:"next_id"`
		Closed   bool                `json:"closed"`
	}

    if err := json.Unmarshal(data, &index); err != nil {
        // Attempt recovery by scanning data directory
        return c.recoverIndex()
    }

	c.messages = index.Messages
	c.nextID = index.NextID
	c.closed = index.Closed

    // Calculate current size
    c.currentSize = 0
    for _, pm := range c.messages {
        fullpath := filepath.Join(c.dataDir, pm.Filename)
        if info, err := os.Stat(fullpath); err == nil {
            c.currentSize += info.Size()
        }
    }

    return nil
}

// saveIndex saves the message index to disk
func (c *PersistentChannel) saveIndex() error {
    index := struct {
        Messages []persistentMessage `json:"messages"`
        NextID   int64               `json:"next_id"`
        Closed   bool                `json:"closed"`
    }{
        Messages: c.messages,
        NextID:   c.nextID,
        Closed:   c.closed,
    }

    data, err := json.Marshal(index)
    if err != nil {
        return err
    }
    // Write atomically to temp then rename
    tmp := c.indexFile + ".tmp"
    f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
    if err != nil {
        return err
    }
    if _, err := f.Write(data); err != nil {
        f.Close()
        return err
    }
    if c.syncWrites {
        if err := f.Sync(); err != nil {
            f.Close()
            return err
        }
    }
    if err := f.Close(); err != nil {
        return err
    }
    if err := os.Rename(tmp, c.indexFile); err != nil {
        return err
    }
    if c.syncWrites {
        // Sync directory to persist rename
        _ = syncDir(c.indexFile)
    }
    return nil
}

// PersistentChannelStats provides channel statistics
type PersistentChannelStats struct {
	Length       int    `json:"length"`
	SizeBytes    int64  `json:"size_bytes"`
	MaxSizeBytes int64  `json:"max_size_bytes"`
	DataDir      string `json:"data_dir"`
	Closed       bool   `json:"closed"`
}

// enforceMaxMessages evicts oldest messages to satisfy MaxMessages.
func (c *PersistentChannel) enforceMaxMessages() {
    if c.maxMessages <= 0 {
        return
    }
    for len(c.messages) > c.maxMessages {
        c.removeOldest()
    }
}

// enforceMaxAge evicts messages older than MaxAge.
func (c *PersistentChannel) enforceMaxAge() {
    if c.maxAge <= 0 {
        return
    }
    cutoff := time.Now().Add(-c.maxAge)
    for len(c.messages) > 0 {
        if c.messages[0].Created.After(cutoff) {
            break
        }
        c.removeOldest()
    }
}

// evictUntilUnderSize removes oldest messages until size <= maxSize.
func (c *PersistentChannel) evictUntilUnderSize() {
    for len(c.messages) > 0 && c.currentSize > c.maxSize {
        c.removeOldest()
    }
}

// removeOldest deletes the oldest message file and updates index/size.
func (c *PersistentChannel) removeOldest() {
    pm := c.messages[0]
    fullpath := filepath.Join(c.dataDir, pm.Filename)
    if info, err := os.Stat(fullpath); err == nil {
        c.currentSize -= info.Size()
    }
    _ = os.Remove(fullpath)
    c.messages = c.messages[1:]
    _ = c.saveIndex()
    imetrics.ChannelEvicted("persistent", 1)
    imetrics.ChannelSizeBytes("persistent", c.currentSize)
}

// cleanupLoop periodically enforces retention
func (c *PersistentChannel) cleanupLoop(interval time.Duration) {
    defer c.cleanupWG.Done()
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for {
        select {
        case <-c.stopCleanup:
            return
        case <-ticker.C:
            c.mu.Lock()
            c.enforceMaxAge()
            c.enforceMaxMessages()
            if c.onFull == "evict_oldest" && c.currentSize > c.maxSize {
                c.evictUntilUnderSize()
            }
            c.mu.Unlock()
        }
    }
}

// recoverIndex attempts to rebuild the index by scanning message files.
func (c *PersistentChannel) recoverIndex() error {
    entries, err := os.ReadDir(c.dataDir)
    if err != nil {
        return err
    }
    msgs := make([]persistentMessage, 0)
    var maxID int64
    var totalSize int64
    for _, e := range entries {
        name := e.Name()
        if !e.IsDir() && len(name) > 4 && name[:4] == "msg_" && filepath.Ext(name) == ".json" {
            fullpath := filepath.Join(c.dataDir, name)
            // Read message to get content
            m, err := c.readMessage(fullpath)
            if err != nil {
                // Skip unreadable files
                continue
            }
            // Extract numeric id
            var id int64
            _, _ = fmt.Sscanf(name, "msg_%d.json", &id)
            if id > maxID {
                maxID = id
            }
            fi, _ := os.Stat(fullpath)
            if fi != nil {
                totalSize += fi.Size()
            }
            msgs = append(msgs, persistentMessage{ID: id, Message: m, Filename: name, Created: time.Now()})
        }
    }
    c.messages = msgs
    c.nextID = maxID
    c.currentSize = totalSize
    c.closed = false
    return nil
}

// syncDir fsyncs the parent directory of the given file path.
func syncDir(path string) error {
    dir := filepath.Dir(path)
    df, err := os.Open(dir)
    if err != nil {
        return err
    }
    defer df.Close()
    return df.Sync()
}
