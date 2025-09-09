// Package checkpoint defines domain-specific errors
package checkpoint

import "errors"

// Domain errors - DRY principle: defined once, used everywhere
var (
	// Checkpoint validation errors
	ErrInvalidCheckpointID = errors.New("invalid checkpoint ID")
	ErrInvalidGraphID      = errors.New("invalid graph ID")
	ErrInvalidThreadID     = errors.New("invalid thread ID")
	ErrNilState            = errors.New("checkpoint state cannot be nil")
	ErrCheckpointNotFound  = errors.New("checkpoint not found")

	// Filter validation errors
	ErrInvalidLimit     = errors.New("limit cannot be negative")
	ErrInvalidOffset    = errors.New("offset cannot be negative")
	ErrInvalidTimeRange = errors.New("invalid time range: since is after before")

	// Persistence errors
	ErrSaveFailed   = errors.New("failed to save checkpoint")
	ErrLoadFailed   = errors.New("failed to load checkpoint")
	ErrDeleteFailed = errors.New("failed to delete checkpoint")
)
