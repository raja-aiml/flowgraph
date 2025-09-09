package services

import (
	"context"
	"fmt"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/channel"
	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
)

// StreamingCheckpointService provides checkpoint management with real-time streaming
// PRINCIPLES:
// - Streaming checkpoints for real-time monitoring
// - Event-driven checkpoint notifications
// - Multi-backend checkpoint storage
type StreamingCheckpointService struct {
	saver        checkpoint.Saver
	eventChannel channel.Channel
	hooks        []CheckpointHook
	autoSave     bool
	saveInterval time.Duration
}

// CheckpointHook is called when checkpoint operations occur
type CheckpointHook func(operation string, checkpointID string, execCtx *dto.ExecutionContext) error

// CheckpointEvent represents a checkpoint-related event
type CheckpointEvent struct {
	Type         string                 `json:"type"` // "created", "loaded", "deleted"
	CheckpointID string                 `json:"checkpoint_id"`
	ExecutionID  string                 `json:"execution_id"`
	ThreadID     string                 `json:"thread_id"`
	Timestamp    time.Time              `json:"timestamp"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// NewStreamingCheckpointService creates a streaming checkpoint service
func NewStreamingCheckpointService(saver checkpoint.Saver, eventChannel channel.Channel) *StreamingCheckpointService {
	return &StreamingCheckpointService{
		saver:        saver,
		eventChannel: eventChannel,
		hooks:        make([]CheckpointHook, 0),
		autoSave:     false,
		saveInterval: 30 * time.Second,
	}
}

// EnableAutoSave enables automatic checkpoint saving at intervals
func (s *StreamingCheckpointService) EnableAutoSave(interval time.Duration) {
	s.autoSave = true
	s.saveInterval = interval
}

// AddCheckpointHook registers a callback for checkpoint operations
func (s *StreamingCheckpointService) AddCheckpointHook(hook CheckpointHook) {
	s.hooks = append(s.hooks, hook)
}

// CreateCheckpoint creates a checkpoint with streaming notification
func (s *StreamingCheckpointService) CreateCheckpoint(ctx context.Context, executionCtx *dto.ExecutionContext) (string, error) {
	checkpointID := s.generateCheckpointID(executionCtx)

	cp := &checkpoint.Checkpoint{
		ID:       checkpointID,
		GraphID:  executionCtx.GraphID,
		ThreadID: executionCtx.ThreadID,
		State:    executionCtx.State,
		Metadata: checkpoint.Metadata{
			Step:      executionCtx.CurrentStep,
			Source:    "streaming_checkpoint_service",
			CreatedBy: "flowgraph",
			Tags:      []string{"streaming", "auto-generated"},
			Writes: map[string]interface{}{
				"execution_id": executionCtx.ExecutionID,
				"config":       executionCtx.Config,
				"start_time":   executionCtx.StartTime,
				"stream_id":    fmt.Sprintf("stream-%d", time.Now().UnixNano()),
			},
		},
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	// Save checkpoint
	err := s.saver.Save(ctx, cp)
	if err != nil {
		return "", fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Notify hooks
	for _, hook := range s.hooks {
		if err := hook("created", checkpointID, executionCtx); err != nil {
			fmt.Printf("Checkpoint hook error: %v\n", err)
		}
	}

	// Stream checkpoint event
	s.streamCheckpointEvent(ctx, "created", checkpointID, executionCtx)

	return checkpointID, nil
}

// CreateStreamingCheckpoint creates a checkpoint optimized for streaming
func (s *StreamingCheckpointService) CreateStreamingCheckpoint(ctx context.Context, executionCtx *dto.ExecutionContext, streamData map[string]interface{}) (string, error) {
	checkpointID := s.generateStreamingCheckpointID(executionCtx)

	cp := &checkpoint.Checkpoint{
		ID:       checkpointID,
		GraphID:  executionCtx.GraphID,
		ThreadID: executionCtx.ThreadID,
		State:    executionCtx.State,
		Metadata: checkpoint.Metadata{
			Step:      executionCtx.CurrentStep,
			Source:    "streaming_executor",
			CreatedBy: "flowgraph-streaming",
			Tags:      []string{"streaming", "real-time", "node-finished"},
			Writes: map[string]interface{}{
				"execution_id": executionCtx.ExecutionID,
				"config":       executionCtx.Config,
				"start_time":   executionCtx.StartTime,
				"stream_data":  streamData,
				"streaming_id": fmt.Sprintf("stream-%d", time.Now().UnixNano()),
			},
		},
		Timestamp: time.Now(),
		Version:   "1.1",
	}

	err := s.saver.Save(ctx, cp)
	if err != nil {
		return "", fmt.Errorf("failed to save streaming checkpoint: %w", err)
	}

	// Stream with enhanced data
	s.streamCheckpointEvent(ctx, "streaming_created", checkpointID, executionCtx)

	return checkpointID, nil
}

// LoadCheckpoint loads checkpoint with streaming notification
func (s *StreamingCheckpointService) LoadCheckpoint(ctx context.Context, checkpointID string) (*dto.ExecutionContext, error) {
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

	// Extract additional metadata
	if writes := cp.Metadata.Writes; writes != nil {
		if executionID, ok := writes["execution_id"].(string); ok {
			execCtx.ExecutionID = executionID
		}
		if config, ok := writes["config"].(dto.ExecutionConfig); ok {
			execCtx.Config = config
		}
		if startTime, ok := writes["start_time"].(time.Time); ok {
			execCtx.StartTime = startTime
		}
	}

	// Notify hooks
	for _, hook := range s.hooks {
		if err := hook("loaded", checkpointID, execCtx); err != nil {
			fmt.Printf("Checkpoint hook error: %v\n", err)
		}
	}

	// Stream checkpoint event
	s.streamCheckpointEvent(ctx, "loaded", checkpointID, execCtx)

	return execCtx, nil
}

// ListCheckpoints returns checkpoints with streaming metadata
func (s *StreamingCheckpointService) ListCheckpoints(ctx context.Context, threadID string) ([]string, error) {
	filter := checkpoint.Filter{
		ThreadID: threadID,
		Limit:    100,
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

// DeleteCheckpoint deletes checkpoint with streaming notification
func (s *StreamingCheckpointService) DeleteCheckpoint(ctx context.Context, checkpointID string) error {
	err := s.saver.Delete(ctx, checkpointID)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	// Stream deletion event
	event := CheckpointEvent{
		Type:         "deleted",
		CheckpointID: checkpointID,
		Timestamp:    time.Now(),
	}

	s.streamEvent(ctx, event)
	return nil
}

// StartAutoSave starts automatic checkpoint saving in background
func (s *StreamingCheckpointService) StartAutoSave(ctx context.Context, execCtx *dto.ExecutionContext) {
	if !s.autoSave {
		return
	}

	go func() {
		ticker := time.NewTicker(s.saveInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if checkpointID, err := s.CreateCheckpoint(ctx, execCtx); err == nil {
					fmt.Printf("Auto-saved checkpoint: %s\n", checkpointID)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// streamCheckpointEvent streams a checkpoint-related event
func (s *StreamingCheckpointService) streamCheckpointEvent(ctx context.Context, eventType, checkpointID string, execCtx *dto.ExecutionContext) {
	event := CheckpointEvent{
		Type:         eventType,
		CheckpointID: checkpointID,
		ExecutionID:  execCtx.ExecutionID,
		ThreadID:     execCtx.ThreadID,
		Timestamp:    time.Now(),
		Metadata: map[string]interface{}{
			"step":     execCtx.CurrentStep,
			"graph_id": execCtx.GraphID,
		},
	}

	s.streamEvent(ctx, event)
}

// streamEvent sends an event to the streaming channel
func (s *StreamingCheckpointService) streamEvent(ctx context.Context, event CheckpointEvent) {
	if s.eventChannel == nil {
		return
	}

	message := channel.Message{
		ID:      fmt.Sprintf("checkpoint-event-%d", time.Now().UnixNano()),
		Type:    channel.MessageTypeData,
		Source:  "checkpoint-service",
		Target:  "streaming-monitor",
		Payload: event,
		Headers: map[string]string{
			"event-type": event.Type,
			"timestamp":  event.Timestamp.Format(time.RFC3339),
		},
	}

	if err := s.eventChannel.Send(ctx, message); err != nil {
		fmt.Printf("Failed to stream checkpoint event: %v\n", err)
	}
}

// generateCheckpointID creates a standard checkpoint ID
func (s *StreamingCheckpointService) generateCheckpointID(execCtx *dto.ExecutionContext) string {
	return fmt.Sprintf("checkpoint-%s-%s-%d-%d",
		execCtx.GraphID,
		execCtx.ThreadID,
		execCtx.CurrentStep,
		time.Now().UnixNano())
}

// generateStreamingCheckpointID creates a streaming-optimized checkpoint ID
func (s *StreamingCheckpointService) generateStreamingCheckpointID(execCtx *dto.ExecutionContext) string {
	return fmt.Sprintf("stream-checkpoint-%s-%s-%d-%d",
		execCtx.GraphID,
		execCtx.ThreadID,
		execCtx.CurrentStep,
		time.Now().UnixNano())
}
