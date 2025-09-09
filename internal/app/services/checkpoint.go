package services

import (
	"context"
	"fmt"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
)

// CheckpointService implements the CheckpointManager interface
// PRINCIPLES:
// - SRP: Manages checkpoint operations for graph execution
// - DIP: Depends on checkpoint.Saver abstraction
// - OCP: Extensible for different checkpoint strategies
type CheckpointService struct {
	saver checkpoint.Saver
}

// NewCheckpointService creates a new checkpoint service
func NewCheckpointService(saver checkpoint.Saver) *CheckpointService {
	return &CheckpointService{
		saver: saver,
	}
}

// CreateCheckpoint creates a checkpoint at the current execution state
func (s *CheckpointService) CreateCheckpoint(ctx context.Context, executionCtx *dto.ExecutionContext) (string, error) {
	checkpointID := s.generateCheckpointID(executionCtx)

	cp := &checkpoint.Checkpoint{
		ID:       checkpointID,
		GraphID:  executionCtx.GraphID,
		ThreadID: executionCtx.ThreadID,
		State:    executionCtx.State,
		Metadata: checkpoint.Metadata{
			Step:      executionCtx.CurrentStep,
			Source:    "graph_executor",
			CreatedBy: "flowgraph",
			Writes: map[string]interface{}{
				"execution_id": executionCtx.ExecutionID,
				"config":       executionCtx.Config,
				"start_time":   executionCtx.StartTime,
			},
		},
		Timestamp: time.Now(),
	}

	err := s.saver.Save(ctx, cp)
	if err != nil {
		return "", fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return checkpointID, nil
}

// LoadCheckpoint loads execution state from a checkpoint
func (s *CheckpointService) LoadCheckpoint(ctx context.Context, checkpointID string) (*dto.ExecutionContext, error) {
	cp, err := s.saver.Load(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Extract execution context from checkpoint
	execCtx := &dto.ExecutionContext{
		GraphID:     cp.GraphID,
		ThreadID:    cp.ThreadID,
		State:       cp.State,
		CurrentStep: cp.Metadata.Step,
	}

	// Extract additional metadata from Writes field
	if executionID, ok := cp.Metadata.Writes["execution_id"].(string); ok {
		execCtx.ExecutionID = executionID
	}
	if config, ok := cp.Metadata.Writes["config"].(dto.ExecutionConfig); ok {
		execCtx.Config = config
	}
	if startTime, ok := cp.Metadata.Writes["start_time"].(time.Time); ok {
		execCtx.StartTime = startTime
	}

	return execCtx, nil
}

// ListCheckpoints returns available checkpoints for a thread
func (s *CheckpointService) ListCheckpoints(ctx context.Context, threadID string) ([]string, error) {
	filter := checkpoint.Filter{
		ThreadID: threadID,
		Limit:    100, // Reasonable limit
	}

	checkpoints, err := s.saver.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	var checkpointIDs []string
	for _, cp := range checkpoints {
		checkpointIDs = append(checkpointIDs, cp.ID)
	}

	return checkpointIDs, nil
}

// generateCheckpointID creates a unique checkpoint ID
func (s *CheckpointService) generateCheckpointID(execCtx *dto.ExecutionContext) string {
	return fmt.Sprintf("checkpoint-%s-%s-%d-%d",
		execCtx.GraphID,
		execCtx.ThreadID,
		execCtx.CurrentStep,
		time.Now().UnixNano())
}
