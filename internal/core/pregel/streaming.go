package pregel

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StreamEvent represents events during graph execution
type StreamEvent struct {
	Type      string      // "vertex_start", "vertex_end", "message", "superstep_start", "superstep_end"
	VertexID  string      // Relevant vertex ID
	Step      int         // Superstep number
	Data      interface{} // Event-specific data
	Timestamp int64       // Unix timestamp
}

// StreamHandler processes streaming events
type StreamHandler interface {
	HandleEvent(event StreamEvent) error
}

// Streamer manages event streaming during execution
type Streamer struct {
	handlers []StreamHandler
	events   chan StreamEvent
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
}

func NewStreamer() *Streamer {
	ctx, cancel := context.WithCancel(context.Background())
	return &Streamer{
		handlers: make([]StreamHandler, 0),
		events:   make(chan StreamEvent, 1000),
		ctx:      ctx,
		cancel:   cancel,
		running:  false,
	}
}

func (s *Streamer) AddHandler(handler StreamHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
}

func (s *Streamer) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}
	s.running = true

	go func() {
		for {
			select {
			case event := <-s.events:
				s.mu.RLock()
				for _, handler := range s.handlers {
					_ = handler.HandleEvent(event)
				}
				s.mu.RUnlock()
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *Streamer) EmitEvent(event StreamEvent) {
	select {
	case s.events <- event:
	default:
		// Drop event if buffer is full
	}
}

func (s *Streamer) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	s.running = false

	s.cancel()
	close(s.events)
}

// NodeFinishedHook is called when a vertex completes execution
type NodeFinishedHook func(vertexID string, result VertexResult)

// StreamingEngine extends Engine with streaming capabilities
type StreamingEngine struct {
	*Engine
	Streamer         *Streamer
	NodeFinishedHook NodeFinishedHook
}

func NewStreamingEngine(vertices map[string]VertexProgram, initialStates map[string]map[string]interface{}, config Config) *StreamingEngine {
	engine := NewEngine(vertices, initialStates, config)
	return &StreamingEngine{
		Engine:   engine,
		Streamer: NewStreamer(),
	}
}

func (se *StreamingEngine) Run(ctx context.Context) error {
	if se.Config.StreamOutput {
		se.Streamer.Start()
		defer se.Streamer.Stop()
	}
	return se.Engine.Run(ctx)
}

// Built-in stream handlers

// LogStreamHandler logs events to standard output
type LogStreamHandler struct {
	Verbose bool
}

func (lsh *LogStreamHandler) HandleEvent(event StreamEvent) error {
	if lsh.Verbose {
		fmt.Printf("[%d] %s: %s (Step %d) - %v\n",
			event.Timestamp, event.Type, event.VertexID, event.Step, event.Data)
	} else {
		fmt.Printf("%s: %s\n", event.Type, event.VertexID)
	}
	return nil
}

// MetricsStreamHandler collects execution metrics
type MetricsStreamHandler struct {
	mu             sync.RWMutex
	VertexCounts   map[string]int        // vertex -> execution count
	SuperstepTimes map[int]time.Duration // superstep -> duration
	EventCounts    map[string]int        // event type -> count
	TotalMessages  int
	startTimes     map[int]time.Time // superstep -> start time
}

func NewMetricsStreamHandler() *MetricsStreamHandler {
	return &MetricsStreamHandler{
		VertexCounts:   make(map[string]int),
		SuperstepTimes: make(map[int]time.Duration),
		EventCounts:    make(map[string]int),
		startTimes:     make(map[int]time.Time),
	}
}

func (msh *MetricsStreamHandler) HandleEvent(event StreamEvent) error {
	msh.mu.Lock()
	defer msh.mu.Unlock()

	msh.EventCounts[event.Type]++

	switch event.Type {
	case "vertex_start":
		msh.VertexCounts[event.VertexID]++
	case "superstep_start":
		msh.startTimes[event.Step] = time.Now()
	case "superstep_end":
		if startTime, exists := msh.startTimes[event.Step]; exists {
			msh.SuperstepTimes[event.Step] = time.Since(startTime)
		}
	case "vertex_end":
		if data, ok := event.Data.(map[string]interface{}); ok {
			if msgCount, ok := data["message_count"].(int); ok {
				msh.TotalMessages += msgCount
			}
		}
	}

	return nil
}

func (msh *MetricsStreamHandler) GetMetrics() map[string]interface{} {
	msh.mu.RLock()
	defer msh.mu.RUnlock()

	totalDuration := time.Duration(0)
	for _, duration := range msh.SuperstepTimes {
		totalDuration += duration
	}

	return map[string]interface{}{
		"vertex_executions": msh.VertexCounts,
		"superstep_times":   msh.SuperstepTimes,
		"event_counts":      msh.EventCounts,
		"total_messages":    msh.TotalMessages,
		"total_duration":    totalDuration,
		"superstep_count":   len(msh.SuperstepTimes),
	}
}

// CallbackStreamHandler executes custom callbacks for specific events
type CallbackStreamHandler struct {
	callbacks map[string]func(StreamEvent) error
	mu        sync.RWMutex
}

func NewCallbackStreamHandler() *CallbackStreamHandler {
	return &CallbackStreamHandler{
		callbacks: make(map[string]func(StreamEvent) error),
	}
}

func (csh *CallbackStreamHandler) AddCallback(eventType string, callback func(StreamEvent) error) {
	csh.mu.Lock()
	defer csh.mu.Unlock()
	csh.callbacks[eventType] = callback
}

func (csh *CallbackStreamHandler) HandleEvent(event StreamEvent) error {
	csh.mu.RLock()
	callback, exists := csh.callbacks[event.Type]
	csh.mu.RUnlock()

	if exists {
		return callback(event)
	}
	return nil
}
