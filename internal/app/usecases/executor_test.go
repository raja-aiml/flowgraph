// ...existing code...
package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	graphrepo "github.com/flowgraph/flowgraph/internal/adapters/repository/graph"
	memory "github.com/flowgraph/flowgraph/internal/adapters/repository/memory"
	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/app/services"
	"github.com/flowgraph/flowgraph/internal/core/graph"
)

func TestDefaultGraphExecutor_Execute(t *testing.T) {
	// Setup dependencies
	nodeProcessor := NewDefaultNodeProcessor()
	edgeEvaluator := NewDefaultEdgeEvaluator()
	stateManager := services.NewStateService()
	checkpointSaver := memory.DefaultInMemorySaver()
	defer checkpointSaver.Close()
	checkpointManager := services.NewCheckpointService(checkpointSaver)
	graphRepo := graphrepo.NewInMemoryGraphRepository()
	executor := NewDefaultGraphExecutor(nodeProcessor, edgeEvaluator, stateManager, checkpointManager, graphRepo)

	// Save test graph to repository
	testGraph := &graph.Graph{
		ID:   "test-graph",
		Name: "Test Graph",
		Nodes: map[string]*graph.Node{
			"start": {ID: "start", Type: graph.NodeTypeFunction, Name: "Start"},
			"step1": {ID: "step1", Type: graph.NodeTypeFunction, Name: "Step 1"},
			"step2": {ID: "step2", Type: graph.NodeTypeFunction, Name: "Step 2"},
			"end":   {ID: "end", Type: graph.NodeTypeFunction, Name: "End"},
		},
		Edges: []*graph.Edge{
			{ID: "e1", Source: "start", Target: "step1", Condition: "always"},
			{ID: "e2", Source: "step1", Target: "step2", Condition: "always"},
			{ID: "e3", Source: "step2", Target: "end", Condition: "always"},
		},
		EntryPoint: "start",
	}
	_ = graphRepo.Save(context.Background(), testGraph)

	t.Run("Execute simple graph", func(t *testing.T) {
		ctx := context.Background()

		req := &dto.ExecutionRequest{
			GraphID:  "test-graph",
			ThreadID: "test-thread",
			Input: map[string]interface{}{
				"message": "Hello, World!",
			},
			Config: dto.ExecutionConfig{
				MaxSteps:        10,
				Timeout:         time.Minute,
				CheckpointEvery: 5,
			},
		}

		response, err := executor.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "test-graph", response.GraphID)
		assert.Equal(t, "test-thread", response.ThreadID)
		assert.Equal(t, dto.ExecutionStatusCompleted, response.Status)
		assert.NotEmpty(t, response.ExecutionID)
		assert.True(t, response.Duration > 0)
		assert.Len(t, response.Steps, 4) // Now executes all steps: start, step1, step2, end
	})

	t.Run("Execute multi-step graph", func(t *testing.T) {
		ctx := context.Background()
		multiStepGraph := &graph.Graph{
			ID:   "multi-step-graph",
			Name: "Multi Step Graph",
			Nodes: map[string]*graph.Node{
				"start": {ID: "start", Type: graph.NodeTypeFunction, Name: "Start"},
				"step1": {ID: "step1", Type: graph.NodeTypeFunction, Name: "Step 1"},
				"step2": {ID: "step2", Type: graph.NodeTypeFunction, Name: "Step 2"},
				"end":   {ID: "end", Type: graph.NodeTypeFunction, Name: "End"},
			},
			Edges: []*graph.Edge{
				{ID: "e1", Source: "start", Target: "step1", Condition: "always"},
				{ID: "e2", Source: "step1", Target: "step2", Condition: "always"},
				{ID: "e3", Source: "step2", Target: "end", Condition: "always"},
			},
			EntryPoint: "start",
		}
		_ = graphRepo.Save(ctx, multiStepGraph)

		req := &dto.ExecutionRequest{
			GraphID:  "multi-step-graph",
			ThreadID: "thread-multi",
			Input: map[string]interface{}{
				"counter": 0,
			},
			Config: dto.ExecutionConfig{
				MaxSteps:        5,
				Timeout:         time.Minute,
				CheckpointEvery: 1,
			},
		}

		response, err := executor.Execute(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "multi-step-graph", response.GraphID)
		assert.Equal(t, "thread-multi", response.ThreadID)
		assert.Equal(t, dto.ExecutionStatusCompleted, response.Status)
		assert.True(t, response.Duration > 0)
		assert.GreaterOrEqual(t, len(response.Steps), 3) // Should traverse at least start, step1, step2, end
		for _, step := range response.Steps {
			assert.NotEmpty(t, step.NodeID)
			assert.Equal(t, dto.StepStatusCompleted, step.Status)
		}
	})

	t.Run("Execute with invalid request", func(t *testing.T) {
		ctx := context.Background()

		req := &dto.ExecutionRequest{
			GraphID:  "", // Invalid: empty graph ID
			ThreadID: "test-thread",
			Input:    map[string]interface{}{},
		}

		response, err := executor.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid request")
	})

	t.Run("Execute with invalid request", func(t *testing.T) {
		ctx := context.Background()

		req := &dto.ExecutionRequest{
			GraphID:  "", // Invalid: empty graph ID
			ThreadID: "test-thread",
			Input:    map[string]interface{}{},
		}

		response, err := executor.Execute(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "invalid request")
	})

	t.Run("Get status of running execution", func(t *testing.T) {
		ctx := context.Background()

		// For this test, we'd need to modify the executor to support async execution
		// For now, we'll test the error case
		response, err := executor.GetStatus(ctx, "non-existent-execution")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Stop execution", func(t *testing.T) {
		ctx := context.Background()

		err := executor.Stop(ctx, "non-existent-execution")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestDefaultGraphExecutor_Execute_ValidationCycles(t *testing.T) {
    // Setup dependencies
    nodeProcessor := NewDefaultNodeProcessor()
    edgeEvaluator := NewDefaultEdgeEvaluator()
    stateManager := services.NewStateService()
    checkpointSaver := memory.DefaultInMemorySaver()
    defer checkpointSaver.Close()
    checkpointManager := services.NewCheckpointService(checkpointSaver)
    graphRepo := graphrepo.NewInMemoryGraphRepository()
    executor := NewDefaultGraphExecutor(nodeProcessor, edgeEvaluator, stateManager, checkpointManager, graphRepo)

    // Create a cyclic graph (allowed by repo save; cycles are validated at execute-time)
    cyclic := &graph.Graph{
        ID:   "cyclic-graph",
        Name: "Cyclic Graph",
        Nodes: map[string]*graph.Node{
            "A": {ID: "A", Type: graph.NodeTypeFunction, Name: "A"},
            "B": {ID: "B", Type: graph.NodeTypeFunction, Name: "B"},
        },
        Edges: []*graph.Edge{
            {ID: "e1", Source: "A", Target: "B"},
            {ID: "e2", Source: "B", Target: "A"},
        },
        EntryPoint: "A",
    }
    // Repository save will validate endpoints and duplicates but not cycles
    _ = graphRepo.Save(context.Background(), cyclic)

    ctx := context.Background()
    req := &dto.ExecutionRequest{
        GraphID:  "cyclic-graph",
        ThreadID: "thread-cycle",
        Input:    map[string]interface{}{},
        Config: dto.ExecutionConfig{
            MaxSteps:       5,
            Timeout:        time.Minute,
            ValidateGraph:  true,
            ValidateCycles: true, // enable cycle detection
        },
    }

    resp, err := executor.Execute(ctx, req)
    assert.Error(t, err)
    assert.Nil(t, resp)
}

func TestDefaultGraphExecutor_Resume(t *testing.T) {
	// Setup dependencies
	nodeProcessor := NewDefaultNodeProcessor()
	edgeEvaluator := NewDefaultEdgeEvaluator()
	stateManager := services.NewStateService()
	checkpointSaver := memory.DefaultInMemorySaver()
	defer checkpointSaver.Close()
	checkpointManager := services.NewCheckpointService(checkpointSaver)

	graphRepo := graphrepo.NewInMemoryGraphRepository()
	executor := NewDefaultGraphExecutor(nodeProcessor, edgeEvaluator, stateManager, checkpointManager, graphRepo)

	t.Run("Resume from checkpoint", func(t *testing.T) {
		ctx := context.Background()
		testGraph := &graph.Graph{
			ID:   "test-graph",
			Name: "Test Graph",
			Nodes: map[string]*graph.Node{
				"start": {ID: "start", Type: graph.NodeTypeFunction, Name: "Start"},
				"step1": {ID: "step1", Type: graph.NodeTypeFunction, Name: "Step 1"},
				"step2": {ID: "step2", Type: graph.NodeTypeFunction, Name: "Step 2"},
				"end":   {ID: "end", Type: graph.NodeTypeFunction, Name: "End"},
			},
			Edges: []*graph.Edge{
				{ID: "e1", Source: "start", Target: "step1", Condition: "always"},
				{ID: "e2", Source: "step1", Target: "step2", Condition: "always"},
				{ID: "e3", Source: "step2", Target: "end", Condition: "always"},
			},
			EntryPoint: "start",
		}
		_ = graphRepo.Save(ctx, testGraph)

		// Create a checkpoint first
		executionCtx := &dto.ExecutionContext{
			ExecutionID: "test-execution",
			GraphID:     "test-graph",
			ThreadID:    "test-thread",
			CurrentStep: 5,
			State: map[string]interface{}{
				"progress": "halfway",
			},
			Config: dto.ExecutionConfig{
				MaxSteps: 10,
				Timeout:  time.Minute,
			},
			StartTime: time.Now().Add(-time.Minute),
		}

		checkpointID, err := checkpointManager.CreateCheckpoint(ctx, executionCtx)
		require.NoError(t, err)

		// Resume execution
		req := &dto.ExecutionRequest{
			GraphID:  "test-graph",
			ThreadID: "test-thread",
			Input:    map[string]interface{}{},
			Config: dto.ExecutionConfig{
				MaxSteps: 20,
				Timeout:  time.Minute,
			},
		}

		response, err := executor.Resume(ctx, checkpointID, req)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.Equal(t, "test-graph", response.GraphID)
		assert.Equal(t, "test-thread", response.ThreadID)
		assert.Equal(t, dto.ExecutionStatusCompleted, response.Status)
	})

	t.Run("Resume with invalid checkpoint", func(t *testing.T) {
		ctx := context.Background()

		req := &dto.ExecutionRequest{
			GraphID:  "test-graph",
			ThreadID: "test-thread",
			Input:    map[string]interface{}{},
		}

		response, err := executor.Resume(ctx, "invalid-checkpoint", req)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "failed to load checkpoint")
	})
}
