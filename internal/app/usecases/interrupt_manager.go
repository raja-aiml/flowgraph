package usecases

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/google/uuid"
)

// InterruptManager handles human-in-the-loop interrupts and workflow control
type InterruptManager struct {
	interrupts      map[string]*InterruptContext
	interruptPoints map[string]bool // node_id -> can_interrupt
	checkpointSaver checkpoint.Saver
	mu              sync.RWMutex
	defaultTimeout  time.Duration
}

// InterruptContext represents an active interrupt session
type InterruptContext struct {
	ID              string                 `json:"id"`
	ExecutionID     string                 `json:"execution_id"`
	GraphID         string                 `json:"graph_id"`
	ThreadID        string                 `json:"thread_id"`
	NodeID          string                 `json:"node_id"`
	InterruptType   string                 `json:"interrupt_type"`
	Status          string                 `json:"status"`
	RequestedAt     time.Time              `json:"requested_at"`
	RespondedAt     *time.Time             `json:"responded_at,omitempty"`
	Timeout         time.Duration          `json:"timeout"`
	UserID          string                 `json:"user_id,omitempty"`
	Message         string                 `json:"message"`
	RequiredInput   map[string]interface{} `json:"required_input,omitempty"`
	UserResponse    map[string]interface{} `json:"user_response,omitempty"`
	CheckpointID    string                 `json:"checkpoint_id"`
	ResumeData      map[string]interface{} `json:"resume_data,omitempty"`
	ResponseChannel chan InterruptResponse `json:"-"`
	CancelFunc      context.CancelFunc     `json:"-"`
}

// InterruptResponse represents a user's response to an interrupt
type InterruptResponse struct {
	Approved bool                   `json:"approved"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Message  string                 `json:"message,omitempty"`
	UserID   string                 `json:"user_id,omitempty"`
}

// InterruptRequest represents a request for user intervention
type InterruptRequest struct {
	NodeID        string                 `json:"node_id"`
	InterruptType string                 `json:"interrupt_type"`
	Message       string                 `json:"message"`
	RequiredInput map[string]interface{} `json:"required_input,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	UserID        string                 `json:"user_id,omitempty"`
}

// NewInterruptManager creates a new interrupt manager
func NewInterruptManager(checkpointSaver checkpoint.Saver) *InterruptManager {
	return &InterruptManager{
		interrupts:      make(map[string]*InterruptContext),
		interruptPoints: make(map[string]bool),
		checkpointSaver: checkpointSaver,
		defaultTimeout:  5 * time.Minute,
	}
}

// RequestInterrupt creates an interrupt request for human intervention
func (im *InterruptManager) RequestInterrupt(
	ctx context.Context,
	executionID string,
	graphID string,
	threadID string,
	req *InterruptRequest,
) (*InterruptContext, error) {
	// Create checkpoint before interrupting
	checkpointID := uuid.New().String()
	checkpointData := &checkpoint.Checkpoint{
		ID:       checkpointID,
		GraphID:  graphID,
		ThreadID: threadID,
		State: map[string]interface{}{
			"interrupt_requested": true,
			"interrupt_type":      req.InterruptType,
			"execution_id":        executionID,
			"node_id":             req.NodeID,
		},
		Metadata: checkpoint.Metadata{
			Source:    "interrupt_manager",
			CreatedBy: "system",
		},
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	err := im.checkpointSaver.Save(ctx, checkpointData)
	if err != nil {
		return nil, err
	}

	// Create interrupt context
	interruptCtx := &InterruptContext{
		ID:              uuid.New().String(),
		ExecutionID:     executionID,
		GraphID:         graphID,
		ThreadID:        threadID,
		NodeID:          req.NodeID,
		InterruptType:   req.InterruptType,
		Status:          "pending",
		RequestedAt:     time.Now(),
		Timeout:         req.Timeout,
		UserID:          req.UserID,
		Message:         req.Message,
		RequiredInput:   req.RequiredInput,
		CheckpointID:    checkpointID,
		ResponseChannel: make(chan InterruptResponse, 1),
	}

	im.mu.Lock()
	im.interrupts[interruptCtx.ID] = interruptCtx
	im.mu.Unlock()

	return interruptCtx, nil
}

// RespondToInterrupt provides a response to an interrupt request
func (im *InterruptManager) RespondToInterrupt(
	interruptID string,
	response *InterruptResponse,
) error {
	im.mu.RLock()
	interruptCtx, exists := im.interrupts[interruptID]
	im.mu.RUnlock()

	if !exists {
		return errors.New("interrupt not found")
	}

	if interruptCtx.Status != "pending" {
		return errors.New("interrupt already resolved")
	}

	// Update interrupt context
	now := time.Now()
	interruptCtx.RespondedAt = &now
	interruptCtx.UserResponse = response.Data

	if response.Approved {
		interruptCtx.Status = "approved"
	} else {
		interruptCtx.Status = "rejected"
	}

	// Send response through channel
	select {
	case interruptCtx.ResponseChannel <- *response:
	default:
		// Channel might be closed or full
	}

	return nil
}

// InterruptAwareExecutor wraps an executor with interrupt capabilities
type InterruptAwareExecutor struct {
	baseExecutor     GraphExecutor
	interruptManager *InterruptManager
}

// NewInterruptAwareExecutor creates an interrupt-aware executor
func NewInterruptAwareExecutor(
	baseExecutor GraphExecutor,
	interruptManager *InterruptManager,
) *InterruptAwareExecutor {
	return &InterruptAwareExecutor{
		baseExecutor:     baseExecutor,
		interruptManager: interruptManager,
	}
}

type interruptManagerKey string

// Execute runs graph execution with interrupt support
func (ie *InterruptAwareExecutor) Execute(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	ctxWithInterrupts := context.WithValue(ctx, interruptManagerKey("interrupt_manager"), ie.interruptManager)
	return ie.baseExecutor.Execute(ctxWithInterrupts, req)
}

// Resume continues execution from a checkpoint
func (ie *InterruptAwareExecutor) Resume(ctx context.Context, checkpointID string, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	ctxWithInterrupts := context.WithValue(ctx, interruptManagerKey("interrupt_manager"), ie.interruptManager)
	return ie.baseExecutor.Resume(ctxWithInterrupts, checkpointID, req)
}

// Stop halts execution
func (ie *InterruptAwareExecutor) Stop(ctx context.Context, executionID string) error {
	return ie.baseExecutor.Stop(ctx, executionID)
}

// GetStatus returns execution status
func (ie *InterruptAwareExecutor) GetStatus(ctx context.Context, executionID string) (*dto.ExecutionResponse, error) {
	return ie.baseExecutor.GetStatus(ctx, executionID)
}
