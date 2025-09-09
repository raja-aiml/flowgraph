package usecases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/channel"
)

// StreamingExecutor provides real-time streaming execution capabilities
// PRINCIPLES:
// - Real-time event streaming during execution
// - Node-finished hooks and event pipeline
// - Streaming checkpoint integration
type StreamingExecutor struct {
	baseExecutor     GraphExecutor
	eventBus         *EventBus
	streamingChannel channel.Channel
	nodeHooks        []NodeFinishedHook
	executionHooks   []ExecutionHook
	mu               sync.RWMutex
}

// NodeFinishedHook is called when a node completes execution
type NodeFinishedHook func(ctx context.Context, nodeID string, result *dto.StepResult) error

// ExecutionHook is called at various points during execution
type ExecutionHook func(ctx context.Context, event ExecutionEvent) error

// ExecutionEvent represents events during graph execution
type ExecutionEvent struct {
	Type        string                 `json:"type"` // "started", "node_started", "node_finished", "completed", "failed"
	ExecutionID string                 `json:"execution_id"`
	GraphID     string                 `json:"graph_id"`
	ThreadID    string                 `json:"thread_id"`
	NodeID      string                 `json:"node_id,omitempty"`
	Step        int                    `json:"step"`
	Timestamp   time.Time              `json:"timestamp"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// EventBus manages event distribution
type EventBus struct {
	subscribers map[string][]EventSubscriber
	mu          sync.RWMutex
}

// EventSubscriber handles specific event types
type EventSubscriber interface {
	HandleEvent(ctx context.Context, event ExecutionEvent) error
	GetEventTypes() []string
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]EventSubscriber),
	}
}

// Subscribe adds an event subscriber for specific event types
func (eb *EventBus) Subscribe(subscriber EventSubscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, eventType := range subscriber.GetEventTypes() {
		eb.subscribers[eventType] = append(eb.subscribers[eventType], subscriber)
	}
}

// Publish publishes an event to all subscribers
func (eb *EventBus) Publish(ctx context.Context, event ExecutionEvent) {
	eb.mu.RLock()
	subscribers := eb.subscribers[event.Type]
	eb.mu.RUnlock()

	for _, subscriber := range subscribers {
		go func(sub EventSubscriber) {
			if err := sub.HandleEvent(ctx, event); err != nil {
				fmt.Printf("Event subscriber error: %v\n", err)
			}
		}(subscriber)
	}
}

// NewStreamingExecutor creates a streaming-enabled graph executor
func NewStreamingExecutor(
	baseExecutor GraphExecutor,
	streamingChannel channel.Channel,
) *StreamingExecutor {
	return &StreamingExecutor{
		baseExecutor:     baseExecutor,
		eventBus:         NewEventBus(),
		streamingChannel: streamingChannel,
		nodeHooks:        make([]NodeFinishedHook, 0),
		executionHooks:   make([]ExecutionHook, 0),
	}
}

// AddNodeFinishedHook registers a callback for node completion
func (se *StreamingExecutor) AddNodeFinishedHook(hook NodeFinishedHook) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.nodeHooks = append(se.nodeHooks, hook)
}

// AddExecutionHook registers a callback for execution events
func (se *StreamingExecutor) AddExecutionHook(hook ExecutionHook) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.executionHooks = append(se.executionHooks, hook)
}

// Subscribe adds an event subscriber to the event bus
func (se *StreamingExecutor) Subscribe(subscriber EventSubscriber) {
	se.eventBus.Subscribe(subscriber)
}

// Execute runs graph execution with real-time streaming
func (se *StreamingExecutor) Execute(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Emit execution started event
	startEvent := ExecutionEvent{
		Type:        "started",
		ExecutionID: "",
		GraphID:     req.GraphID,
		ThreadID:    req.ThreadID,
		Timestamp:   time.Now(),
		Data: map[string]interface{}{
			"input":  req.Input,
			"config": req.Config,
		},
	}
	se.publishEvent(ctx, startEvent)

	// Execute using base executor with streaming wrapper
	response, err := se.executeWithStreaming(ctx, req)

	if err != nil {
		// Emit execution failed event
		failEvent := ExecutionEvent{
			Type:        "failed",
			ExecutionID: response.ExecutionID,
			GraphID:     req.GraphID,
			ThreadID:    req.ThreadID,
			Timestamp:   time.Now(),
			Error:       err.Error(),
		}
		se.publishEvent(ctx, failEvent)
	} else {
		// Emit execution completed event
		completeEvent := ExecutionEvent{
			Type:        "completed",
			ExecutionID: response.ExecutionID,
			GraphID:     req.GraphID,
			ThreadID:    req.ThreadID,
			Timestamp:   time.Now(),
			Data: map[string]interface{}{
				"output":   response.Output,
				"steps":    len(response.Steps),
				"duration": response.Duration,
			},
		}
		se.publishEvent(ctx, completeEvent)
	}

	return response, err
}

// executeWithStreaming wraps base execution with streaming capabilities
func (se *StreamingExecutor) executeWithStreaming(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Create a streaming-aware context
	streamCtx := se.createStreamingContext(ctx, req)

	// Execute with base executor
	response, err := se.baseExecutor.Execute(streamCtx, req)
	if err != nil {
		return response, err
	}

	// Post-process response with streaming enhancements
	se.enhanceResponseWithStreaming(response)

	return response, nil
}

// streamingDataKey is the context key for streaming data
type streamingDataKey string

const streamingDataContextKey streamingDataKey = "streaming_data"

// createStreamingContext creates a context with streaming capabilities
func (se *StreamingExecutor) createStreamingContext(ctx context.Context, req *dto.ExecutionRequest) context.Context {
	// Add streaming metadata to context
	streamData := map[string]interface{}{
		"streaming_enabled": true,
		"graph_id":          req.GraphID,
		"thread_id":         req.ThreadID,
		"start_time":        time.Now(),
	}

	// Create context with streaming data (this would be implementation-specific)
	return context.WithValue(ctx, streamingDataContextKey, streamData)
}

// enhanceResponseWithStreaming adds streaming metadata to response
func (se *StreamingExecutor) enhanceResponseWithStreaming(response *dto.ExecutionResponse) {
	// Add streaming metadata to each step
	for i := range response.Steps {
		step := &response.Steps[i]

		// Emit node started event
		nodeStartEvent := ExecutionEvent{
			Type:        "node_started",
			ExecutionID: response.ExecutionID,
			GraphID:     response.GraphID,
			ThreadID:    response.ThreadID,
			NodeID:      step.NodeID,
			Step:        step.StepNumber,
			Timestamp:   step.StartTime,
		}
		se.publishEvent(context.Background(), nodeStartEvent)

		// Emit node finished event
		nodeFinishedEvent := ExecutionEvent{
			Type:        "node_finished",
			ExecutionID: response.ExecutionID,
			GraphID:     response.GraphID,
			ThreadID:    response.ThreadID,
			NodeID:      step.NodeID,
			Step:        step.StepNumber,
			Timestamp:   step.EndTime,
			Data: map[string]interface{}{
				"duration": step.Duration,
				"status":   step.Status,
				"output":   step.Output,
			},
		}
		if step.Error != "" {
			nodeFinishedEvent.Error = step.Error
		}
		se.publishEvent(context.Background(), nodeFinishedEvent)

		// Call node finished hooks
		se.callNodeFinishedHooks(context.Background(), step.NodeID, step)
	}
}

// callNodeFinishedHooks invokes all registered node finished hooks
func (se *StreamingExecutor) callNodeFinishedHooks(ctx context.Context, nodeID string, result *dto.StepResult) {
	se.mu.RLock()
	hooks := se.nodeHooks
	se.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx, nodeID, result); err != nil {
			fmt.Printf("Node finished hook error: %v\n", err)
		}
	}
}

// publishEvent publishes an event to the event bus and streaming channel
func (se *StreamingExecutor) publishEvent(ctx context.Context, event ExecutionEvent) {
	// Publish to event bus
	se.eventBus.Publish(ctx, event)

	// Call execution hooks
	se.mu.RLock()
	hooks := se.executionHooks
	se.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx, event); err != nil {
			fmt.Printf("Execution hook error: %v\n", err)
		}
	}

	// Stream to channel
	se.streamEvent(ctx, event)
}

// streamEvent sends an event to the streaming channel
func (se *StreamingExecutor) streamEvent(ctx context.Context, event ExecutionEvent) {
	if se.streamingChannel == nil {
		return
	}

	message := channel.Message{
		ID:      fmt.Sprintf("exec-event-%d", time.Now().UnixNano()),
		Type:    channel.MessageTypeData,
		Source:  "streaming-executor",
		Target:  "event-monitor",
		Payload: event,
		Headers: map[string]string{
			"event-type":   event.Type,
			"execution-id": event.ExecutionID,
			"timestamp":    event.Timestamp.Format(time.RFC3339),
		},
	}

	if err := se.streamingChannel.Send(ctx, message); err != nil {
		fmt.Printf("Failed to stream execution event: %v\n", err)
	}
}

// Resume continues execution from a checkpoint with streaming
func (se *StreamingExecutor) Resume(ctx context.Context, checkpointID string, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Emit resume started event
	resumeEvent := ExecutionEvent{
		Type:      "resume_started",
		GraphID:   req.GraphID,
		ThreadID:  req.ThreadID,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"checkpoint_id": checkpointID,
		},
	}
	se.publishEvent(ctx, resumeEvent)

	// Resume with base executor
	return se.baseExecutor.Resume(ctx, checkpointID, req)
}

// Stop halts execution with streaming notification
func (se *StreamingExecutor) Stop(ctx context.Context, executionID string) error {
	// Emit stop event
	stopEvent := ExecutionEvent{
		Type:        "stopped",
		ExecutionID: executionID,
		Timestamp:   time.Now(),
	}
	se.publishEvent(ctx, stopEvent)

	return se.baseExecutor.Stop(ctx, executionID)
}

// GetStatus returns status with streaming enhancements
func (se *StreamingExecutor) GetStatus(ctx context.Context, executionID string) (*dto.ExecutionResponse, error) {
	return se.baseExecutor.GetStatus(ctx, executionID)
}

// Built-in Event Subscribers

// LoggingSubscriber logs all execution events
type LoggingSubscriber struct {
	Verbose bool
}

func (ls *LoggingSubscriber) HandleEvent(ctx context.Context, event ExecutionEvent) error {
	if ls.Verbose {
		fmt.Printf("[%s] %s: %s (Step %d) - %v\n",
			event.Timestamp.Format("15:04:05"),
			event.Type,
			event.ExecutionID,
			event.Step,
			event.Data)
	} else {
		fmt.Printf("[%s] %s: %s\n",
			event.Timestamp.Format("15:04:05"),
			event.Type,
			event.ExecutionID)
	}
	return nil
}

func (ls *LoggingSubscriber) GetEventTypes() []string {
	return []string{"started", "node_started", "node_finished", "completed", "failed", "stopped"}
}

// MetricsSubscriber collects execution metrics
type MetricsSubscriber struct {
	mu              sync.RWMutex
	ExecutionCounts map[string]int             // execution status -> count
	NodeDurations   map[string][]time.Duration // node -> durations
	ExecutionTimes  map[string]time.Duration   // execution_id -> total duration
	EventCounts     map[string]int             // event type -> count
}

func NewMetricsSubscriber() *MetricsSubscriber {
	return &MetricsSubscriber{
		ExecutionCounts: make(map[string]int),
		NodeDurations:   make(map[string][]time.Duration),
		ExecutionTimes:  make(map[string]time.Duration),
		EventCounts:     make(map[string]int),
	}
}

func (ms *MetricsSubscriber) HandleEvent(ctx context.Context, event ExecutionEvent) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.EventCounts[event.Type]++

	switch event.Type {
	case "completed", "failed":
		ms.ExecutionCounts[event.Type]++
		if duration, ok := event.Data["duration"].(time.Duration); ok {
			ms.ExecutionTimes[event.ExecutionID] = duration
		}
	case "node_finished":
		if duration, ok := event.Data["duration"].(time.Duration); ok {
			ms.NodeDurations[event.NodeID] = append(ms.NodeDurations[event.NodeID], duration)
		}
	}

	return nil
}

func (ms *MetricsSubscriber) GetEventTypes() []string {
	return []string{"started", "node_finished", "completed", "failed"}
}

func (ms *MetricsSubscriber) GetMetrics() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	avgNodeDurations := make(map[string]time.Duration)
	for node, durations := range ms.NodeDurations {
		if len(durations) > 0 {
			total := time.Duration(0)
			for _, d := range durations {
				total += d
			}
			avgNodeDurations[node] = total / time.Duration(len(durations))
		}
	}

	return map[string]interface{}{
		"execution_counts":   ms.ExecutionCounts,
		"avg_node_durations": avgNodeDurations,
		"execution_times":    ms.ExecutionTimes,
		"event_counts":       ms.EventCounts,
	}
}
