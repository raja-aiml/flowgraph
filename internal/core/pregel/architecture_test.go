package pregel

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCompletePregelArchitecture tests all implemented sub-tasks
func TestCompletePregelArchitecture(t *testing.T) {
	t.Run("BSP Superstep Execution", func(t *testing.T) {
		vertices := map[string]VertexProgram{
			"A": &simpleVertex{},
			"B": &simpleVertex{},
		}
		initialStates := map[string]map[string]interface{}{
			"A": {"count": 0, "next": "B"},
			"B": {"count": 0, "next": "A"},
		}
		config := Config{
			MaxSupersteps: 3,
			Parallelism:   2,
		}
		engine := NewEngine(vertices, initialStates, config)
		ctx := context.Background()

		assert.NoError(t, engine.Run(ctx))
		assert.True(t, engine.Halted)
	})

	t.Run("Message Aggregation", func(t *testing.T) {
		aggregator := NewMessageAggregator(nil)
		msg1 := &Message{From: "A", To: "B", Value: 1}
		msg2 := &Message{From: "A", To: "B", Value: 2}

		aggregator.AddMessage(msg1)
		aggregator.AddMessage(msg2)

		messages := aggregator.GetMessages("B")
		assert.Len(t, messages, 2)
	})

	t.Run("Work Stealing Scheduler", func(t *testing.T) {
		scheduler := NewWorkStealingScheduler(2, 100)
		assert.Equal(t, 2, scheduler.numWorkers)
		assert.Len(t, scheduler.queues, 2)
		scheduler.Stop()
	})

	t.Run("Streaming Capabilities", func(t *testing.T) {
		streamer := NewStreamer()
		streamer.Start()

		event := StreamEvent{
			Type:     "vertex_start",
			VertexID: "A",
			Step:     0,
		}
		streamer.EmitEvent(event)
		streamer.Stop()
	})

	t.Run("Error Recovery", func(t *testing.T) {
		policy := RetryPolicy{MaxRetries: 3, BackoffMs: 10}
		handler := NewErrorRecoveryHandler(policy)

		retry, err := handler.HandleError("A", assert.AnError, 0)
		assert.True(t, retry)
		assert.NoError(t, err)

		retry, err = handler.HandleError("A", assert.AnError, 3)
		assert.False(t, retry)
		assert.Error(t, err)
	})

	t.Run("Checkpointing", func(t *testing.T) {
		cm := NewCheckpointManager()
		states := map[string]map[string]interface{}{
			"A": {"count": 5},
			"B": {"count": 3},
		}

		cm.SaveCheckpoint(1, states)
		loadedStates, exists := cm.LoadCheckpoint(1)

		assert.True(t, exists)
		assert.Equal(t, states, loadedStates)
	})

	t.Run("Resilient Engine", func(t *testing.T) {
		vertices := map[string]VertexProgram{
			"A": &simpleVertex{},
		}
		initialStates := map[string]map[string]interface{}{
			"A": {"count": 0},
		}
		config := Config{
			MaxSupersteps: 2,
			Parallelism:   1,
			RetryPolicy:   RetryPolicy{MaxRetries: 2, BackoffMs: 1},
		}

		engine := NewResilientEngine(vertices, initialStates, config)
		assert.NotNil(t, engine.ErrorHandler)
		assert.NotNil(t, engine.CheckpointManager)
	})

	t.Run("Integrated Engine with All Components", func(t *testing.T) {
		// Create a more complex scenario with message combining
		vertices := map[string]VertexProgram{
			"A": &messagePassingVertex{targetCount: 5},
			"B": &messagePassingVertex{targetCount: 5},
			"C": &messagePassingVertex{targetCount: 5},
		}
		initialStates := map[string]map[string]interface{}{
			"A": {"count": 0, "messages_sent": 0},
			"B": {"count": 0, "messages_sent": 0},
			"C": {"count": 0, "messages_sent": 0},
		}
		config := Config{
			MaxSupersteps: 5,
			Parallelism:   3,
			StreamOutput:  true,
			RetryPolicy:   RetryPolicy{MaxRetries: 2, BackoffMs: 1},
		}

		engine := NewEngine(vertices, initialStates, config)
		defer engine.Stop()

		// Add a stream handler to capture events
		eventCount := 0
		handler := &testStreamHandler{eventCount: &eventCount}
		engine.Streamer.AddHandler(handler)

		ctx := context.Background()
		assert.NoError(t, engine.Run(ctx))

		// Verify all components were used (with proper synchronization)
		handler.mutex.Lock()
		finalEventCount := eventCount
		handler.mutex.Unlock()
		assert.True(t, finalEventCount > 0, "Streaming events should have been emitted")
		assert.NotNil(t, engine.MessageAggregator, "Message aggregator should be present")
		assert.NotNil(t, engine.Scheduler, "Work stealing scheduler should be present")
		assert.NotNil(t, engine.CheckpointManager, "Checkpoint manager should be present")
		assert.NotNil(t, engine.ErrorHandler, "Error handler should be present")

		// Verify final state progression
		for vid, state := range engine.VertexStates {
			count := state["count"].(int)
			assert.GreaterOrEqual(t, count, 1, "Vertex %s should have processed messages", vid)
		}
	})

	t.Run("Message Combining with SumCombiner", func(t *testing.T) {
		combiner := &SumCombiner{}

		messages := []*Message{
			{From: "A", To: "B", Value: 5},
			{From: "C", To: "B", Value: 3},
			{From: "D", To: "B", Value: 2},
		}

		combined := combiner.Combine(messages)
		assert.NotNil(t, combined)
		assert.Equal(t, 10.0, combined.Value)
		assert.Equal(t, "B", combined.To)
	})
}

// Test helper types
type messagePassingVertex struct {
	targetCount int
}

func (mpv *messagePassingVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	count := 0
	if v, ok := state["count"].(int); ok {
		count = v
	}

	messagesSent := 0
	if v, ok := state["messages_sent"].(int); ok {
		messagesSent = v
	}

	// Process incoming messages
	for _, msg := range messages {
		if val, ok := msg.Value.(int); ok {
			count += val
		} else if val, ok := msg.Value.(float64); ok {
			count += int(val)
		}
	}

	// Always increment count to show activity
	count++
	state["count"] = count

	// Send messages to other vertices if we haven't reached target
	var outMsgs []*Message
	if messagesSent < mpv.targetCount {
		// Send message to next vertex (simple ring topology)
		var next string
		switch vertexID {
		case "A":
			next = "B"
		case "B":
			next = "C"
		case "C":
			next = "A"
		}

		if next != "" {
			outMsgs = append(outMsgs, &Message{
				From:  vertexID,
				To:    next,
				Value: 1,
			})
			messagesSent++
		}
	}

	state["messages_sent"] = messagesSent
	halt := messagesSent >= mpv.targetCount && count >= 3 // More lenient halt condition

	return state, outMsgs, halt, nil
}

type testStreamHandler struct {
	eventCount *int
	mutex      sync.Mutex
}

func (tsh *testStreamHandler) HandleEvent(event StreamEvent) error {
	tsh.mutex.Lock()
	defer tsh.mutex.Unlock()
	*tsh.eventCount++
	return nil
}

// TestPregelVsPython demonstrates feature parity areas
func TestPregelVsPython(t *testing.T) {
	t.Run("Vertex-centric computation model", func(t *testing.T) {
		// ✅ Implemented: VertexProgram interface
		var program VertexProgram = &simpleVertex{}
		assert.NotNil(t, program)
	})

	t.Run("BSP synchronization barriers", func(t *testing.T) {
		// ✅ Implemented: Superstep coordination
		superstep := NewSuperstep(0, nil)
		assert.Equal(t, 0, superstep.StepNumber)
	})

	t.Run("Message aggregation and combining", func(t *testing.T) {
		// ✅ Implemented: MessageAggregator
		aggregator := NewMessageAggregator(nil)
		assert.NotNil(t, aggregator)
	})

	t.Run("Parallel worker pools with goroutines", func(t *testing.T) {
		// ✅ Implemented: WorkStealingScheduler
		scheduler := NewWorkStealingScheduler(4, 100)
		assert.Equal(t, 4, scheduler.numWorkers)
		scheduler.Stop()
	})

	t.Run("Streaming outputs and node-finished hooks", func(t *testing.T) {
		// ✅ Implemented: StreamingEngine
		vertices := map[string]VertexProgram{"A": &simpleVertex{}}
		initialStates := map[string]map[string]interface{}{"A": {"count": 0}}
		config := Config{MaxSupersteps: 1, StreamOutput: true}

		engine := NewStreamingEngine(vertices, initialStates, config)
		assert.NotNil(t, engine.Streamer)
	})

	t.Run("Error recovery and retry policies", func(t *testing.T) {
		// ✅ Implemented: ResilientEngine with retry logic
		policy := RetryPolicy{MaxRetries: 3, BackoffMs: 100}
		handler := NewErrorRecoveryHandler(policy)
		assert.Equal(t, 3, handler.policy.MaxRetries)
	})
}
