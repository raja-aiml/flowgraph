// Package channel provides communication abstractions for graph execution
package channel

import (
	"context"
)

// Channel interface for message passing between nodes
// PRINCIPLES:
// - ISP: Interface segregation with ≤5 methods
// - DIP: Core domain depends on interface, not implementations
// - SRP: Single responsibility - message passing
type Channel interface {
	// Send sends a message to the channel
	Send(ctx context.Context, message Message) error

	// Receive receives a message from the channel
	Receive(ctx context.Context) (Message, error)

	// Close closes the channel
	Close() error
}

// Message represents a message passed between nodes
type Message struct {
	ID       string                 `json:"id"`
	Type     MessageType            `json:"type"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Payload  interface{}            `json:"payload"`
	Headers  map[string]string      `json:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MessageType represents the type of message
type MessageType string

const (
	// MessageTypeData represents a data message
	MessageTypeData MessageType = "data"
	// MessageTypeControl represents a control message
	MessageTypeControl MessageType = "control"
	// MessageTypeError represents an error message
	MessageTypeError MessageType = "error"
	// MessageTypeHeartbeat represents a heartbeat message
	MessageTypeHeartbeat MessageType = "heartbeat"
)

// Validate ensures message integrity
func (m *Message) Validate() error {
	if m.ID == "" {
		return ErrInvalidMessageID
	}
	if m.Type == "" {
		return ErrInvalidMessageType
	}
	if m.Source == "" {
		return ErrInvalidSource
	}
	if m.Target == "" {
		return ErrInvalidTarget
	}
	return nil
}
