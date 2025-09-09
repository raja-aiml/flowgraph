// Package checkpoint provides the core checkpoint domain entities and interfaces
// following Clean Architecture principles with zero external dependencies.
package checkpoint

import (
	"time"
)

// Checkpoint represents a saved state in the graph execution
// PRINCIPLES:
// - KISS: Simple struct with clear fields
// - SRP: Only responsible for checkpoint data structure
type Checkpoint struct {
	ID        string                 `json:"id"`
	GraphID   string                 `json:"graph_id"`
	ThreadID  string                 `json:"thread_id"`
	State     map[string]interface{} `json:"state"`
	Metadata  Metadata               `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
}

// Metadata contains additional information about a checkpoint
type Metadata struct {
	Step      int                    `json:"step"`
	Source    string                 `json:"source"`
	Writes    map[string]interface{} `json:"writes,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	CreatedBy string                 `json:"created_by,omitempty"`
	Tags      []string               `json:"tags,omitempty"`
}

// Validate ensures checkpoint integrity
// PRINCIPLES:
// - SRP: Single responsibility - validation only
// - KISS: Simple validation rules, easy to understand
func (c *Checkpoint) Validate() error {
	if c.ID == "" {
		return ErrInvalidCheckpointID
	}
	if c.GraphID == "" {
		return ErrInvalidGraphID
	}
	if c.ThreadID == "" {
		return ErrInvalidThreadID
	}
	if c.State == nil {
		return ErrNilState
	}
	return nil
}
