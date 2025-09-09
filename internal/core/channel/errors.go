// Package channel defines domain-specific errors
package channel

import "errors"

// Domain errors - DRY principle: defined once, used everywhere
var (
	// Message errors
	ErrInvalidMessageID   = errors.New("invalid message ID")
	ErrInvalidMessageType = errors.New("invalid message type")
	ErrInvalidSource      = errors.New("invalid message source")
	ErrInvalidTarget      = errors.New("invalid message target")

	// Channel errors
	ErrChannelClosed = errors.New("channel is closed")
	ErrChannelFull   = errors.New("channel is full")
	ErrChannelEmpty  = errors.New("channel is empty")
	ErrTimeout       = errors.New("operation timed out")
)
