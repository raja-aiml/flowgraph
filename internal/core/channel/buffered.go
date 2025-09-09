// Package channel provides buffered channel implementation
package channel

import (
    "context"
    "time"
    "sync"
)
 
import imetrics "github.com/flowgraph/flowgraph/internal/infrastructure/metrics"

// BufferedChannel provides queued message passing with configurable buffering
// PRINCIPLES:
// - KISS: Simple queue-based implementation
// - SRP: Single responsibility - buffered message passing
// - Thread-safe: Uses proper synchronization
type BufferedChannel struct {
    buffer   []Message
    maxSize  int
    closed   bool
    mu       sync.RWMutex
    notify   chan struct{}
    timeout  time.Duration
}

// BufferedChannelConfig holds configuration for BufferedChannel
type BufferedChannelConfig struct {
	MaxSize int           // Maximum buffer size
	Timeout time.Duration // Default timeout for operations
}

// NewBufferedChannel creates a new buffered channel
func NewBufferedChannel(config BufferedChannelConfig) *BufferedChannel {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000 // Default max size
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second // Default timeout
	}

    bc := &BufferedChannel{
        buffer:  make([]Message, 0, config.MaxSize),
        maxSize: config.MaxSize,
        timeout: config.Timeout,
        notify:  make(chan struct{}),
    }

    return bc
}

// DefaultBufferedChannel creates a buffered channel with default settings
func DefaultBufferedChannel() *BufferedChannel {
    cfg := BufferedChannelConfig{MaxSize: 1000, Timeout: 30 * time.Second}
    if defaultRuntimeConfig.BufferedMaxSize > 0 {
        cfg.MaxSize = defaultRuntimeConfig.BufferedMaxSize
    }
    if defaultRuntimeConfig.BufferedTimeout > 0 {
        cfg.Timeout = defaultRuntimeConfig.BufferedTimeout
    }
    return NewBufferedChannel(cfg)
}

// Send sends a message to the channel
// nolint:funlen,gocognit,gocyclo // Kept as a single logical unit for clarity and performance
func (c *BufferedChannel) Send(ctx context.Context, message Message) error {
	// Validate message first
	if err := message.Validate(); err != nil {
		return err
	}

    deadline := time.Now().Add(c.timeout)
    for {
        c.mu.Lock()
        if c.closed {
            c.mu.Unlock()
            return ErrChannelClosed
        }
        if len(c.buffer) < c.maxSize {
            // Add message and notify waiters
            c.buffer = append(c.buffer, message)
            imetrics.ChannelSent("buffered", 1)
            c.signal()
            c.mu.Unlock()
            return nil
        }
        // Need to wait for space
        ch := c.notify
        c.mu.Unlock()

        // Check cancellation/timeout
        if err := ctx.Err(); err != nil {
            return err
        }
        if time.Now().After(deadline) {
            return ErrTimeout
        }

        // Wait for a state change or cancellation/timeout
        remaining := time.Until(deadline)
        select {
        case <-ch:
            // state changed; loop and recheck
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(remaining):
            return ErrTimeout
        }
    }
}

// Receive receives a message from the channel
// nolint:funlen,gocognit,gocyclo // Kept as a single logical unit for clarity and performance
func (c *BufferedChannel) Receive(ctx context.Context) (Message, error) {
    deadline := time.Now().Add(c.timeout)
    for {
        c.mu.Lock()
        if len(c.buffer) > 0 {
            // Pop FIFO and notify waiters
            message := c.buffer[0]
            c.buffer = c.buffer[1:]
            imetrics.ChannelReceived("buffered", 1)
            c.signal()
            c.mu.Unlock()
            return message, nil
        }
        if c.closed {
            c.mu.Unlock()
            return Message{}, ErrChannelClosed
        }
        ch := c.notify
        c.mu.Unlock()

        if err := ctx.Err(); err != nil {
            return Message{}, err
        }
        if time.Now().After(deadline) {
            return Message{}, ErrTimeout
        }

        remaining := time.Until(deadline)
        select {
        case <-ch:
        case <-ctx.Done():
            return Message{}, ctx.Err()
        case <-time.After(remaining):
            return Message{}, ErrTimeout
        }
    }
}

// Close closes the channel
func (c *BufferedChannel) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return nil // Already closed
    }

    c.closed = true
    c.signal()
    return nil
}

// Len returns the number of messages currently in the buffer
func (c *BufferedChannel) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.buffer)
}

// Cap returns the capacity of the channel
func (c *BufferedChannel) Cap() int {
	return c.maxSize
}

// IsClosed returns whether the channel is closed
func (c *BufferedChannel) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// Clear removes all messages from the buffer
func (c *BufferedChannel) Clear() {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.buffer = c.buffer[:0]
    c.signal()
}

// Stats returns channel statistics
func (c *BufferedChannel) Stats() BufferedChannelStats {
    c.mu.RLock()
    defer c.mu.RUnlock()

	return BufferedChannelStats{
		Length:   len(c.buffer),
		Capacity: c.maxSize,
		Closed:   c.closed,
	}
}

// signal notifies all current waiters of a state change.
// Must be called with c.mu held.
func (c *BufferedChannel) signal() {
    // Close the current notify channel to broadcast to all listeners,
    // then create a new one for future waits.
    old := c.notify
    // Replace notify first to avoid races with late listeners.
    c.notify = make(chan struct{})
    // Closing old will wake all waiters selecting on it.
    close(old)
}

// BufferedChannelStats provides channel statistics
type BufferedChannelStats struct {
	Length   int  `json:"length"`
	Capacity int  `json:"capacity"`
	Closed   bool `json:"closed"`
}
