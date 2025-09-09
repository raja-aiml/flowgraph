package pregel

// Engine implements the Pregel vertex-centric graph computation model with BSP synchronization.
// PRINCIPLES:
// - Vertex-centric parallel computation
// - Bulk Synchronous Parallel (BSP) superstep barriers
// - Message-passing between vertices
// - Error recovery and streaming support (to be added)

import (
    "context"
    "fmt"
    "math"
    "runtime"
    "sync"
    "time"
)
import imetrics "github.com/flowgraph/flowgraph/internal/infrastructure/metrics"

// Config holds engine configuration
type Config struct {
    MaxSupersteps int
    Parallelism   int
    // ParallelismFactor scales CPU count when Parallelism is not set.
    // Example: 1.5 => 1.5x NumCPU workers. Ignored if Parallelism > 0.
    ParallelismFactor float64
    // QueueCapacity sets the per-worker queue capacity for the scheduler.
    // Defaults to 100 when <= 0.
    QueueCapacity int
    Timeout       time.Duration
    RetryPolicy   RetryPolicy
    StreamOutput  bool
}

// RetryPolicy defines retry behavior for failed vertices
type RetryPolicy struct {
	MaxRetries int
	BackoffMs  int
}

type Engine struct {
	Vertices          map[string]VertexProgram
	VertexStates      map[string]map[string]interface{}
	Config            Config
	Scheduler         *WorkStealingScheduler
	MessageAggregator *MessageAggregator
	Streamer          *Streamer
	CheckpointManager *CheckpointManager
	ErrorHandler      *ErrorRecoveryHandler
	ActiveVertices    map[string]bool
	Halted            bool
	retryAttempts     map[string]int
	mu                sync.RWMutex
}

func NewEngine(vertices map[string]VertexProgram, initialStates map[string]map[string]interface{}, config Config) *Engine {
    // Ensure a sensible default for parallelism to maximize goroutine throughput.
    effParallelism := config.Parallelism
    if effParallelism <= 0 {
        if config.ParallelismFactor > 0 {
            effParallelism = int(math.Ceil(config.ParallelismFactor * float64(runtime.NumCPU())))
        } else {
            effParallelism = runtime.NumCPU()
        }
        if effParallelism < 1 {
            effParallelism = 1
        }
    }

    qcap := config.QueueCapacity
    if qcap <= 0 { qcap = 100 }
    scheduler := NewWorkStealingScheduler(effParallelism, qcap)
    scheduler.StartWorkers()

    return &Engine{
		Vertices:          vertices,
		VertexStates:      initialStates,
        Config:            Config{MaxSupersteps: config.MaxSupersteps, Parallelism: effParallelism, ParallelismFactor: config.ParallelismFactor, QueueCapacity: qcap, Timeout: config.Timeout, RetryPolicy: config.RetryPolicy, StreamOutput: config.StreamOutput},
		Scheduler:         scheduler,
		MessageAggregator: NewMessageAggregator(&SumCombiner{}),
		Streamer:          NewStreamer(),
		CheckpointManager: NewCheckpointManager(),
		ErrorHandler:      NewErrorRecoveryHandler(config.RetryPolicy),
		ActiveVertices:    make(map[string]bool),
		Halted:            false,
		retryAttempts:     make(map[string]int),
	}
}

// Run executes the Pregel algorithm with full integration of all components
func (e *Engine) Run(ctx context.Context) error {
    e.startStreamingIfEnabled()
    e.initActiveVertices()
    e.emitEvent(StreamEvent{Type: "execution_start", Step: 0, Data: map[string]interface{}{"total_vertices": len(e.Vertices)}})

    for step := 0; step < e.Config.MaxSupersteps && !e.Halted; step++ {
        if err := e.runStep(ctx, step); err != nil {
            return err
        }
    }

    e.emitEvent(StreamEvent{Type: "execution_end", Step: -1, Data: map[string]interface{}{"final_states": e.VertexStates}})
    return nil
}

// startStreamingIfEnabled starts the streamer and ensures it is stopped when Run returns.
func (e *Engine) startStreamingIfEnabled() {
    if e.Config.StreamOutput {
        e.Streamer.Start()
        // Stop streamer when engine is stopped via Stop() or when Run completes.
        // Run defers cannot be used here; Stop() also calls Streamer.Stop().
    }
}

// initActiveVertices marks all vertices as active at the beginning of execution.
func (e *Engine) initActiveVertices() {
    for vid := range e.Vertices {
        e.ActiveVertices[vid] = true
    }
}

// runStep executes a single superstep including checkpointing, events, and cleanup.
func (e *Engine) runStep(ctx context.Context, step int) error {
    stepCtx, cancel := e.stepContext(ctx)
    if cancel != nil {
        defer cancel()
    }

    e.CheckpointManager.SaveCheckpoint(step, e.VertexStates)
    imetrics.IncSupersteps()
    e.emitEvent(StreamEvent{Type: "superstep_start", Step: step, Data: map[string]interface{}{"active_vertices": len(e.getActiveVertices())}})

    if err := e.executeSuperstep(stepCtx, step); err != nil {
        if recoveredStates, exists := e.CheckpointManager.LoadCheckpoint(step); exists {
            e.VertexStates = recoveredStates
            e.emitEvent(StreamEvent{Type: "checkpoint_recovery", Step: step, Data: map[string]interface{}{"error": err.Error()}})
        }
        return fmt.Errorf("superstep %d failed: %w", step, err)
    }

    e.emitEvent(StreamEvent{Type: "superstep_end", Step: step, Data: map[string]interface{}{"active_vertices": len(e.getActiveVertices())}})
    if !e.hasActiveVertices() {
        e.Halted = true
    }
    e.MessageAggregator.Clear()
    return nil
}

// stepContext returns a per-step context with timeout if configured.
func (e *Engine) stepContext(ctx context.Context) (context.Context, context.CancelFunc) {
    if e.Config.Timeout > 0 {
        return context.WithTimeout(ctx, e.Config.Timeout)
    }
    return ctx, nil
}

func (e *Engine) executeSuperstep(ctx context.Context, step int) error {
    activeVertices := e.getActiveVertices()
    if len(activeVertices) == 0 {
        return nil
    }

    // Create result channel for collecting vertex execution results
    results := make(chan VertexResult, len(activeVertices))

    e.scheduleVertexTasks(step, activeVertices, results)

    // Collect results with timeout support
    var errs []error
    collected := 0
    for collected < len(activeVertices) {
        select {
        case result := <-results:
            collected++
            imetrics.IncVertexExecs(1)
            if retry, herr := e.handleVertexResult(result, step, results); retry {
                collected--
            } else if herr != nil {
                errs = append(errs, herr)
            }
        case <-ctx.Done():
            return fmt.Errorf("superstep %d timed out: %w", step, ctx.Err())
        }
    }

    if len(errs) > 0 {
        return errs[0]
    }
    return nil
}

// scheduleVertexTasks submits tasks for the given active vertices and emits vertex_start events.
func (e *Engine) scheduleVertexTasks(step int, vertices []string, results chan VertexResult) {
    for _, vid := range vertices {
        messages := e.MessageAggregator.GetMessages(vid)
        e.emitEvent(StreamEvent{Type: "vertex_start", VertexID: vid, Step: step, Data: map[string]interface{}{"message_count": len(messages)}})
        task := VertexTask{VertexID: vid, Program: e.Vertices[vid], State: e.VertexStates[vid], Messages: messages, Result: results}
        e.Scheduler.Schedule(task)
    }
}

// handleVertexResult updates state, schedules retries if needed, and emits vertex_end.
// It returns (retryScheduled, handledError).
func (e *Engine) handleVertexResult(result VertexResult, step int, results chan VertexResult) (bool, error) {
    if result.Error != nil {
        retry, handledErr := e.ErrorHandler.HandleError(result.VertexID, result.Error, e.retryAttempts[result.VertexID])
        if retry {
            e.retryAttempts[result.VertexID]++
            task := VertexTask{
                VertexID: result.VertexID,
                Program:  e.Vertices[result.VertexID],
                State:    e.VertexStates[result.VertexID],
                Messages: e.MessageAggregator.GetMessages(result.VertexID),
                Result:   results,
            }
            e.Scheduler.Schedule(task)
            // Emit vertex end with error
            e.emitEvent(StreamEvent{Type: "vertex_end", VertexID: result.VertexID, Step: step, Data: map[string]interface{}{"halt": result.Halt, "error": result.Error, "message_count": len(result.Messages)}})
            return true, nil
        }
        e.emitEvent(StreamEvent{Type: "vertex_end", VertexID: result.VertexID, Step: step, Data: map[string]interface{}{"halt": result.Halt, "error": result.Error, "message_count": len(result.Messages)}})
        return false, handledErr
    }

    // Success path
    e.mu.Lock()
    e.VertexStates[result.VertexID] = result.State
    e.ActiveVertices[result.VertexID] = !result.Halt
    e.retryAttempts[result.VertexID] = 0
    e.mu.Unlock()

    for _, msg := range result.Messages {
        msg.Step = step + 1
        e.MessageAggregator.AddMessage(msg)
    }
    e.emitEvent(StreamEvent{Type: "vertex_end", VertexID: result.VertexID, Step: step, Data: map[string]interface{}{"halt": result.Halt, "error": result.Error, "message_count": len(result.Messages)}})
    return false, nil
}

// Helper methods
func (e *Engine) emitEvent(event StreamEvent) {
	if e.Streamer != nil {
		event.Timestamp = time.Now().Unix()
		e.Streamer.EmitEvent(event)
	}
}

func (e *Engine) getActiveVertices() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var active []string
	for vid, isActive := range e.ActiveVertices {
		if isActive {
			active = append(active, vid)
		}
	}
	return active
}

func (e *Engine) hasActiveVertices() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, active := range e.ActiveVertices {
		if active {
			return true
		}
	}
	return false
}

// Stop cleans up engine resources
func (e *Engine) Stop() {
    if e.Scheduler != nil {
        e.Scheduler.Stop()
    }
    if e.Streamer != nil {
        e.Streamer.Stop()
    }
}

// EngineStats provides runtime metrics for observability.
type EngineStats struct {
    Workers      int
    QueuesQueued int
    ActiveCount  int
    Halted       bool
}

// Stats returns a snapshot of engine metrics.
func (e *Engine) Stats() EngineStats {
    e.mu.RLock()
    defer e.mu.RUnlock()
    sched := e.Scheduler.Stats()
    return EngineStats{
        Workers:      sched.NumWorkers,
        QueuesQueued: sched.TotalQueued,
        ActiveCount:  len(e.getActiveVertices()),
        Halted:       e.Halted,
    }
}
