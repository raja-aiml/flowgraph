package usecases

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/flowgraph/flowgraph/internal/core/graph"
)

func TestDefaultNodeProcessor(t *testing.T) {
	processor := NewDefaultNodeProcessor()

	t.Run("Process function node", func(t *testing.T) {
		ctx := context.Background()

		node := &graph.Node{
			ID:   "test-func",
			Type: graph.NodeTypeFunction,
			Name: "Test Function",
		}

		input := map[string]interface{}{
			"message": "Hello",
		}

		output, err := processor.Process(ctx, node, input)
		require.NoError(t, err)
		assert.Equal(t, "Hello", output["message"])
		assert.Equal(t, "test-func", output["processed_by"])
		assert.Equal(t, string(graph.NodeTypeFunction), output["node_type"])
	})

	t.Run("Process conditional node", func(t *testing.T) {
		ctx := context.Background()

		node := &graph.Node{
			ID:   "test-condition",
			Type: graph.NodeTypeConditional,
			Name: "Test Condition",
		}

		input := map[string]interface{}{
			"value": 42,
		}

		output, err := processor.Process(ctx, node, input)
		require.NoError(t, err)
		assert.Equal(t, 42, output["value"])
		assert.Equal(t, true, output["condition_result"])
		assert.Equal(t, "test-condition", output["evaluated_by"])
	})

	t.Run("Process tool node", func(t *testing.T) {
		ctx := context.Background()

		node := &graph.Node{
			ID:   "test-tool",
			Type: graph.NodeTypeTool,
			Name: "Test Tool",
		}

		input := map[string]interface{}{
			"data": "test",
		}

		output, err := processor.Process(ctx, node, input)
		require.NoError(t, err)
		assert.Equal(t, "test", output["data"])
		assert.Equal(t, "test-tool", output["tool_executed"])
		assert.Equal(t, string(graph.NodeTypeTool), output["tool_type"])
	})

	t.Run("Process agent node", func(t *testing.T) {
		ctx := context.Background()

		node := &graph.Node{
			ID:   "test-agent",
			Type: graph.NodeTypeAgent,
			Name: "Test Agent",
		}

		input := map[string]interface{}{
			"query": "What is the weather?",
		}

		output, err := processor.Process(ctx, node, input)
		require.NoError(t, err)
		assert.Equal(t, "What is the weather?", output["query"])
		assert.Equal(t, "test-agent", output["agent_processed"])
		assert.Equal(t, string(graph.NodeTypeAgent), output["agent_type"])
	})

	t.Run("Process unsupported node type", func(t *testing.T) {
		ctx := context.Background()

		node := &graph.Node{
			ID:   "test-unknown",
			Type: graph.NodeType("unknown"),
			Name: "Unknown Node",
		}

		input := map[string]interface{}{}

		_, err := processor.Process(ctx, node, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no processor found")
	})

	t.Run("CanProcess", func(t *testing.T) {
		assert.True(t, processor.CanProcess(graph.NodeTypeFunction))
		assert.True(t, processor.CanProcess(graph.NodeTypeConditional))
		assert.True(t, processor.CanProcess(graph.NodeTypeTool))
		assert.True(t, processor.CanProcess(graph.NodeTypeAgent))
		assert.False(t, processor.CanProcess(graph.NodeType("unknown")))
	})
}
