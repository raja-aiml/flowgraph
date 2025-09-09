package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowgraph/flowgraph/internal/core/graph"
)

func TestDefaultEdgeEvaluator(t *testing.T) {
	evaluator := NewDefaultEdgeEvaluator()

	t.Run("Evaluate edge without condition", func(t *testing.T) {
		ctx := context.Background()

		edge := &graph.Edge{
			ID:        "edge-1",
			Source:    "node-1",
			Target:    "node-2",
			Condition: "", // No condition
		}

		state := map[string]interface{}{
			"status": "success",
		}

		result, err := evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.True(t, result) // Should always be true when no condition
	})

	t.Run("Evaluate edge with 'always' condition", func(t *testing.T) {
		ctx := context.Background()

		edge := &graph.Edge{
			ID:        "edge-2",
			Source:    "node-1",
			Target:    "node-2",
			Condition: "always",
		}

		state := map[string]interface{}{}

		result, err := evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("Evaluate edge with 'never' condition", func(t *testing.T) {
		ctx := context.Background()

		edge := &graph.Edge{
			ID:        "edge-3",
			Source:    "node-1",
			Target:    "node-2",
			Condition: "never",
		}

		state := map[string]interface{}{}

		result, err := evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("Evaluate edge with 'success' condition", func(t *testing.T) {
		ctx := context.Background()

		edge := &graph.Edge{
			ID:        "edge-4",
			Source:    "node-1",
			Target:    "node-2",
			Condition: "success",
		}

		// Test with success status
		state := map[string]interface{}{
			"status": "success",
		}

		result, err := evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.True(t, result)

		// Test with failure status
		state["status"] = "failure"
		result, err = evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("Evaluate edge with 'has_error' condition", func(t *testing.T) {
		ctx := context.Background()

		edge := &graph.Edge{
			ID:        "edge-5",
			Source:    "node-1",
			Target:    "node-2",
			Condition: "has_error",
		}

		// Test with error
		state := map[string]interface{}{
			"error": "Something went wrong",
		}

		result, err := evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.True(t, result)

		// Test without error
		state = map[string]interface{}{
			"error": nil,
		}

		result, err = evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.False(t, result)

		// Test with no error field
		state = map[string]interface{}{}

		result, err = evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("Evaluate edge with unknown condition", func(t *testing.T) {
		ctx := context.Background()

		edge := &graph.Edge{
			ID:        "edge-6",
			Source:    "node-1",
			Target:    "node-2",
			Condition: "custom_condition",
		}

		state := map[string]interface{}{}

		result, err := evaluator.Evaluate(ctx, edge, state)
		require.NoError(t, err)
		assert.True(t, result) // Default to true for unknown conditions
	})

	t.Run("GetNextNodes", func(t *testing.T) {
		ctx := context.Background()

		currentNode := &graph.Node{
			ID:   "node-1",
			Type: graph.NodeTypeFunction,
			Name: "Current Node",
		}

		edges := []*graph.Edge{
			{
				ID:        "edge-1",
				Source:    "node-1",
				Target:    "node-2",
				Condition: "success",
			},
			{
				ID:        "edge-2",
				Source:    "node-1",
				Target:    "node-3",
				Condition: "failure",
			},
			{
				ID:        "edge-3",
				Source:    "other-node", // Different source
				Target:    "node-4",
				Condition: "always",
			},
		}

		// Test with success status
		state := map[string]interface{}{
			"status": "success",
		}

		nextNodes, err := evaluator.GetNextNodes(ctx, currentNode, edges, state)
		require.NoError(t, err)
		assert.Len(t, nextNodes, 1)
		assert.Equal(t, "node-2", nextNodes[0].ID)

		// Test with failure status
		state["status"] = "failure"
		nextNodes, err = evaluator.GetNextNodes(ctx, currentNode, edges, state)
		require.NoError(t, err)
		assert.Len(t, nextNodes, 1)
		assert.Equal(t, "node-3", nextNodes[0].ID)
	})
}
