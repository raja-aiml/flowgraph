// Package channel provides in-memory channel implementation
package channel

import (
    "context"
    "sync"
    "time"
)
 
import imetrics "github.com/flowgraph/flowgraph/internal/infrastructure/metrics"

// InMemoryChannel provides fast in-memory message passing
// PRINCIPLES:
// - KISS: Simple channel with basic operations
// - SRP: Single responsibility - in-memory message passing
// - Thread-safe: Uses proper synchronization
type InMemoryChannel struct {
	messages chan Message
	closed   bool
	mu       sync.RWMutex
	timeout  time.Duration
}

// InMemoryChannelConfig holds configuration for InMemoryChannel
type InMemoryChannelConfig struct {
	BufferSize int           // Buffer size for the channel
	Timeout    time.Duration // Default timeout for operations
}

// NewInMemoryChannel creates a new in-memory channel
func NewInMemoryChannel(config InMemoryChannelConfig) *InMemoryChannel {
	if config.BufferSize <= 0 {
		config.BufferSize = 100 // Default buffer size
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second // Default timeout
	}

	return &InMemoryChannel{
		messages: make(chan Message, config.BufferSize),
		timeout:  config.Timeout,
	}
}

// DefaultInMemoryChannel creates an in-memory channel with default settings
func DefaultInMemoryChannel() *InMemoryChannel {
    cfg := InMemoryChannelConfig{BufferSize: 100, Timeout: 30 * time.Second}
    if defaultRuntimeConfig.InMemoryBufferSize > 0 {
        cfg.BufferSize = defaultRuntimeConfig.InMemoryBufferSize
    }
    if defaultRuntimeConfig.InMemoryTimeout > 0 {
        cfg.Timeout = defaultRuntimeConfig.InMemoryTimeout
    }
    return NewInMemoryChannel(cfg)
}

// Send sends a message to the channel
func (c *InMemoryChannel) Send(ctx context.Context, message Message) error {
	// Validate message first
	if err := message.Validate(); err != nil {
		return err
	}

	// Check if channel is closed
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrChannelClosed
	}
	c.mu.RUnlock()

	// Send with timeout
    select {
    case c.messages <- message:
        imetrics.ChannelSent("inmemory", 1)
        return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(c.timeout):
		return ErrTimeout
	}
}

// Receive receives a message from the channel
func (c *InMemoryChannel) Receive(ctx context.Context) (Message, error) {
	// Check if channel is closed
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		// Try to receive any remaining messages
    select {
    case msg := <-c.messages:
        imetrics.ChannelReceived("inmemory", 1)
        return msg, nil
		default:
			return Message{}, ErrChannelClosed
		}
	}
	c.mu.RUnlock()

	// Receive with timeout
	select {
	case msg := <-c.messages:
		return msg, nil
	case <-ctx.Done():
		return Message{}, ctx.Err()
	case <-time.After(c.timeout):
		return Message{}, ErrTimeout
	}
}

// Close closes the channel
func (c *InMemoryChannel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil // Already closed
	}

	c.closed = true
	close(c.messages)
	return nil
}

// Len returns the number of messages currently in the channel
func (c *InMemoryChannel) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.messages)
}

// Cap returns the capacity of the channel
func (c *InMemoryChannel) Cap() int {
	return cap(c.messages)
}

// IsClosed returns whether the channel is closed
func (c *InMemoryChannel) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Stats returns channel statistics
func (c *InMemoryChannel) Stats() InMemoryChannelStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return InMemoryChannelStats{
		Length:   len(c.messages),
		Capacity: cap(c.messages),
		Closed:   c.closed,
	}
}

// InMemoryChannelStats provides channel statistics
type InMemoryChannelStats struct {
	Length   int  `json:"length"`
	Capacity int  `json:"capacity"`
	Closed   bool `json:"closed"`
}
