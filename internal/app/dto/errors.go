package dto

import "errors"

// Execution errors
var (
	ErrMissingGraphID   = errors.New("graph ID is required")
	ErrMissingThreadID  = errors.New("thread ID is required")
	ErrInvalidConfig    = errors.New("invalid execution configuration")
	ErrExecutionFailed  = errors.New("graph execution failed")
	ErrExecutionTimeout = errors.New("graph execution timeout")
	ErrStepFailed       = errors.New("step execution failed")
	ErrInvalidInput     = errors.New("invalid input provided")
	ErrInvalidOutput    = errors.New("invalid output produced")
)
