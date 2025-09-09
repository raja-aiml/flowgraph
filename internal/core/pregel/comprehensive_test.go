package pregel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedVertexPrograms tests the vertex programs we added
func TestAdvancedVertexPrograms(t *testing.T) {
	t.Run("SimpleCounterVertex", func(t *testing.T) {
		vertex := &SimpleCounterVertex{MaxCount: 3}

		state := map[string]interface{}{
			"count":     0,
			"neighbors": []string{"B", "C"},
		}

		// First computation
		newState, messages, halt, err := vertex.Compute("A", state, []*Message{})
		require.NoError(t, err)
		assert.False(t, halt)
		assert.Equal(t, 1, newState["count"])
		assert.Len(t, messages, 2) // Messages to B and C

		// Continue until max count
		for i := 2; i <= 3; i++ {
			newState, messages, halt, err = vertex.Compute("A", newState, []*Message{})
			require.NoError(t, err)
			assert.Equal(t, i, newState["count"])
			if i == 3 {
				assert.True(t, halt)
			} else {
				assert.False(t, halt)
				assert.Len(t, messages, 2)
			}
		}
	})

	t.Run("PageRankVertex", func(t *testing.T) {
		vertex := &PageRankVertex{
			DampingFactor: 0.85,
			Convergence:   0.01,
		}

		state := map[string]interface{}{
			"rank":      1.0,
			"iteration": 0,
			"edges":     []string{"B", "C"},
		}

		// Simulate incoming PageRank from neighbors
		messages := []*Message{
			{From: "B", Value: 0.425},
			{From: "C", Value: 0.425},
		}

		newState, outMessages, halt, err := vertex.Compute("A", state, messages)
		require.NoError(t, err)

		assert.NotEqual(t, 1.0, newState["rank"]) // Rank should change
		assert.Equal(t, 1, newState["iteration"])
		assert.NotNil(t, newState["delta"])

		if !halt {
			assert.Len(t, outMessages, 2) // Messages to B and C
			for _, msg := range outMessages {
				assert.IsType(t, float64(0), msg.Value)
			}
		}
	})

	t.Run("ShortestPathVertex", func(t *testing.T) {
		vertex := &ShortestPathVertex{SourceVertex: "A"}

		// Source vertex
		sourceState := map[string]interface{}{
			"distance": float64(1000000), // Infinity
			"edges":    map[string]float64{"B": 2.0, "C": 5.0},
		}

		newState, messages, halt, err := vertex.Compute("A", sourceState, []*Message{})
		require.NoError(t, err)
		assert.Equal(t, 0.0, newState["distance"]) // Source gets distance 0
		assert.False(t, halt)                      // Should send messages
		assert.Len(t, messages, 2)                 // To B and C

		// Non-source vertex receiving distance update
		nonSourceState := map[string]interface{}{
			"distance": float64(1000000),
			"edges":    map[string]float64{"D": 3.0},
		}

		distanceMessage := []*Message{
			{From: "A", Value: 2.0}, // Distance 2 from A
		}

		newState, messages, halt, err = vertex.Compute("B", nonSourceState, distanceMessage)
		require.NoError(t, err)
		assert.Equal(t, 2.0, newState["distance"])
		assert.False(t, halt)      // Changed, so should propagate
		assert.Len(t, messages, 1) // To D with distance 5.0 (2+3)
		assert.Equal(t, 5.0, messages[0].Value)
	})
}

// TestAdvancedMessageCombiners tests all the message combiners
func TestAdvancedMessageCombiners(t *testing.T) {
	messages := []*Message{
		{From: "A", To: "Target", Value: 5},
		{From: "B", To: "Target", Value: 3},
		{From: "C", To: "Target", Value: 8},
		{From: "D", To: "Target", Value: 1},
	}

	t.Run("SumCombiner", func(t *testing.T) {
		combiner := &SumCombiner{}
		result := combiner.Combine(messages)
		require.NotNil(t, result)
		assert.Equal(t, 17.0, result.Value) // 5+3+8+1
	})

	t.Run("MinCombiner", func(t *testing.T) {
		combiner := &MinCombiner{}
		result := combiner.Combine(messages)
		require.NotNil(t, result)
		assert.Equal(t, 1.0, result.Value) // Min of 5,3,8,1
	})

	t.Run("MaxCombiner", func(t *testing.T) {
		combiner := &MaxCombiner{}
		result := combiner.Combine(messages)
		require.NotNil(t, result)
		assert.Equal(t, 8.0, result.Value) // Max of 5,3,8,1
	})

	t.Run("ListCombiner", func(t *testing.T) {
		combiner := &ListCombiner{}
		result := combiner.Combine(messages)
		require.NotNil(t, result)

		valueList, ok := result.Value.([]interface{})
		require.True(t, ok)
		assert.Len(t, valueList, 4)
		assert.Contains(t, valueList, 5)
		assert.Contains(t, valueList, 3)
		assert.Contains(t, valueList, 8)
		assert.Contains(t, valueList, 1)
	})

	t.Run("NoCombiner", func(t *testing.T) {
		combiner := &NoCombiner{}
		result := combiner.Combine(messages)
		require.NotNil(t, result)
		assert.Equal(t, 5, result.Value) // Should return first message
	})
}

// TestMessageAggregatorAdvanced tests advanced aggregator features
func TestMessageAggregatorAdvanced(t *testing.T) {
	t.Run("MessageAggregator with different combiners", func(t *testing.T) {
		// Test with MinCombiner
		aggregator := NewMessageAggregator(&MinCombiner{})

		aggregator.AddMessage(&Message{From: "A", To: "Target", Value: 10})
		aggregator.AddMessage(&Message{From: "B", To: "Target", Value: 5})
		aggregator.AddMessage(&Message{From: "C", To: "Target", Value: 15})

		messages := aggregator.GetMessages("Target")
		require.Len(t, messages, 1)             // Combined into one
		assert.Equal(t, 5.0, messages[0].Value) // Minimum value

		// Test aggregator utility methods
		assert.Equal(t, 3, aggregator.GetMessageCount())
		assert.True(t, aggregator.HasMessages())

		allMessages := aggregator.GetAllMessages()
		assert.Len(t, allMessages, 1)           // One vertex
		assert.Len(t, allMessages["Target"], 3) // Three original messages

		aggregator.Clear()
		assert.False(t, aggregator.HasMessages())
		assert.Equal(t, 0, aggregator.GetMessageCount())
	})
}

// TestStreamingHandlers tests the advanced streaming handlers
func TestStreamingHandlers(t *testing.T) {
	t.Run("MetricsStreamHandler", func(t *testing.T) {
		handler := NewMetricsStreamHandler()

		// Simulate events
		events := []StreamEvent{
			{Type: "superstep_start", Step: 0},
			{Type: "vertex_start", VertexID: "A", Step: 0},
			{Type: "vertex_end", VertexID: "A", Step: 0, Data: map[string]interface{}{"message_count": 2}},
			{Type: "vertex_start", VertexID: "B", Step: 0},
			{Type: "vertex_end", VertexID: "B", Step: 0, Data: map[string]interface{}{"message_count": 1}},
			{Type: "superstep_end", Step: 0},
		}

		for _, event := range events {
			err := handler.HandleEvent(event)
			assert.NoError(t, err)
		}

		metrics := handler.GetMetrics()
		assert.Equal(t, 1, metrics["vertex_executions"].(map[string]int)["A"])
		assert.Equal(t, 1, metrics["vertex_executions"].(map[string]int)["B"])
		assert.Equal(t, 3, metrics["total_messages"])
		assert.Equal(t, 1, metrics["superstep_count"])

		eventCounts := metrics["event_counts"].(map[string]int)
		assert.Equal(t, 2, eventCounts["vertex_start"])
		assert.Equal(t, 2, eventCounts["vertex_end"])
		assert.Equal(t, 1, eventCounts["superstep_start"])
		assert.Equal(t, 1, eventCounts["superstep_end"])
	})

	t.Run("CallbackStreamHandler", func(t *testing.T) {
		handler := NewCallbackStreamHandler()

		vertexStartCount := 0
		handler.AddCallback("vertex_start", func(event StreamEvent) error {
			vertexStartCount++
			return nil
		})

		// Send some events
		handler.HandleEvent(StreamEvent{Type: "vertex_start", VertexID: "A"})
		handler.HandleEvent(StreamEvent{Type: "vertex_start", VertexID: "B"})
		handler.HandleEvent(StreamEvent{Type: "vertex_end", VertexID: "A"}) // Should not increment

		assert.Equal(t, 2, vertexStartCount)
	})
}

// TestCheckpointManagerAdvanced tests advanced checkpoint features
func TestCheckpointManagerAdvanced(t *testing.T) {
	cm := NewCheckpointManager()

	// Save multiple checkpoints
	for i := 0; i < 5; i++ {
		states := map[string]map[string]interface{}{
			"A": {"count": i},
			"B": {"count": i * 2},
		}
		cm.SaveCheckpoint(i, states)
	}

	// Test listing checkpoints
	checkpoints := cm.ListCheckpoints()
	assert.Len(t, checkpoints, 5)

	// Test loading specific checkpoint
	states, exists := cm.LoadCheckpoint(2)
	assert.True(t, exists)
	assert.Equal(t, 2, states["A"]["count"])
	assert.Equal(t, 4, states["B"]["count"])

	// Test deletion
	cm.DeleteCheckpoint(2)
	_, exists = cm.LoadCheckpoint(2)
	assert.False(t, exists)

	// Test clear
	cm.Clear()
	checkpoints = cm.ListCheckpoints()
	assert.Len(t, checkpoints, 0)
}

// TestSuperstepAdvanced tests advanced superstep features
func TestSuperstepAdvanced(t *testing.T) {
	superstep := NewSuperstep(0, &SumCombiner{})

	// Add vertices
	superstep.AddVertex("A")
	superstep.AddVertex("B")
	superstep.AddVertex("C")

	assert.Equal(t, 3, superstep.GetActiveVertexCount())
	assert.True(t, superstep.HasActiveVertices())

	activeVertices := superstep.GetActiveVertices()
	assert.Len(t, activeVertices, 3)
	assert.Contains(t, activeVertices, "A")
	assert.Contains(t, activeVertices, "B")
	assert.Contains(t, activeVertices, "C")

	// Add messages to aggregator
	superstep.Aggregator.AddMessage(&Message{From: "A", To: "B", Value: 1})
	superstep.Aggregator.AddMessage(&Message{From: "C", To: "B", Value: 2})

	assert.Equal(t, 2, superstep.GetMessageCount())

	// Complete vertices
	superstep.CompleteVertex("A", true) // A halts
	assert.Equal(t, 2, superstep.GetActiveVertexCount())

	superstep.CompleteVertex("B", false)
	superstep.CompleteVertex("C", false)

	// Check if superstep is complete
	assert.False(t, superstep.IsComplete()) // EndTime not set yet

	// Reset superstep
	superstep.Reset()
	assert.Equal(t, 0, superstep.GetActiveVertexCount())
	assert.Equal(t, 0, superstep.GetMessageCount())
	assert.False(t, superstep.IsComplete())
}

// TestIntegratedPregelEngine tests the fully integrated engine
func TestIntegratedPregelEngine(t *testing.T) {
	t.Run("PageRank Algorithm Integration", func(t *testing.T) {
		// Create a simple graph: A -> B -> C -> A
		vertices := map[string]VertexProgram{
			"A": &PageRankVertex{DampingFactor: 0.85, Convergence: 0.01},
			"B": &PageRankVertex{DampingFactor: 0.85, Convergence: 0.01},
			"C": &PageRankVertex{DampingFactor: 0.85, Convergence: 0.01},
		}

		initialStates := map[string]map[string]interface{}{
			"A": {"rank": 1.0, "iteration": 0, "edges": []string{"B"}},
			"B": {"rank": 1.0, "iteration": 0, "edges": []string{"C"}},
			"C": {"rank": 1.0, "iteration": 0, "edges": []string{"A"}},
		}

		config := Config{
			MaxSupersteps: 10,
			Parallelism:   3,
			StreamOutput:  true,
			RetryPolicy:   RetryPolicy{MaxRetries: 2, BackoffMs: 1},
		}

		engine := NewEngine(vertices, initialStates, config)
		defer engine.Stop()

		// Add metrics handler
		metricsHandler := NewMetricsStreamHandler()
		engine.Streamer.AddHandler(metricsHandler)

		ctx := context.Background()
		err := engine.Run(ctx)
		assert.NoError(t, err)

		// Verify PageRank converged
		for vid, state := range engine.VertexStates {
			rank := state["rank"].(float64)
			assert.Greater(t, rank, 0.0, "Vertex %s should have positive PageRank", vid)
			iteration := state["iteration"].(int)
			assert.Greater(t, iteration, 0, "Vertex %s should have completed iterations", vid)
		}

		// Check metrics
		metrics := metricsHandler.GetMetrics()
		assert.Greater(t, metrics["total_messages"].(int), 0)
		assert.Greater(t, len(metrics["vertex_executions"].(map[string]int)), 0)
	})
}
