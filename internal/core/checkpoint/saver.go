// Package checkpoint provides checkpoint persistence interfaces
package checkpoint

import (
	"context"
	"time"
)

// Saver interface for checkpoint persistence (DIP - Dependency Inversion)
// PRINCIPLES:
// - ISP: Interface segregation with ≤5 methods
// - DIP: Core domain depends on interface, not implementations
// - SRP: Single responsibility - checkpoint persistence
type Saver interface {
	// Save persists a checkpoint
	Save(ctx context.Context, checkpoint *Checkpoint) error

	// Load retrieves a checkpoint by ID
	Load(ctx context.Context, id string) (*Checkpoint, error)

	// List returns checkpoints matching the filter
	List(ctx context.Context, filter Filter) ([]*Checkpoint, error)

	// Delete removes a checkpoint by ID
	Delete(ctx context.Context, id string) error
}

// Filter for checkpoint queries (ISP - segregated interface)
type Filter struct {
	GraphID  string     `json:"graph_id,omitempty"`
	ThreadID string     `json:"thread_id,omitempty"`
	Limit    int        `json:"limit,omitempty"`
	Offset   int        `json:"offset,omitempty"`
	Since    *time.Time `json:"since,omitempty"`
	Before   *time.Time `json:"before,omitempty"`
	Tags     []string   `json:"tags,omitempty"`
}

// Validate ensures filter parameters are valid
func (f *Filter) Validate() error {
	if f.Limit < 0 {
		return ErrInvalidLimit
	}
	if f.Offset < 0 {
		return ErrInvalidOffset
	}
	if f.Since != nil && f.Before != nil && f.Since.After(*f.Before) {
		return ErrInvalidTimeRange
	}
	return nil
}
