package usecases

import (
	"context"
	"fmt"

	"github.com/flowgraph/flowgraph/internal/core/graph"
)

// DefaultNodeProcessor implements the NodeProcessor interface
// PRINCIPLES:
// - SRP: Handles only node processing logic
// - OCP: Extensible for different node types
// - LSP: Substitutable for any NodeProcessor implementation
type DefaultNodeProcessor struct {
	// processors maps node types to their specific processors
	processors map[graph.NodeType]NodeTypeProcessor
}

// NodeTypeProcessor defines the interface for type-specific node processors
type NodeTypeProcessor interface {
	Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error)
}

// NewDefaultNodeProcessor creates a new node processor
func NewDefaultNodeProcessor() *DefaultNodeProcessor {
	processor := &DefaultNodeProcessor{
		processors: make(map[graph.NodeType]NodeTypeProcessor),
	}

	// Register default processors
	processor.RegisterProcessor(graph.NodeTypeFunction, &FunctionNodeProcessor{})
	processor.RegisterProcessor(graph.NodeTypeConditional, &ConditionalNodeProcessor{})
	processor.RegisterProcessor(graph.NodeTypeTool, &ToolNodeProcessor{})
	processor.RegisterProcessor(graph.NodeTypeAgent, &AgentNodeProcessor{})

	return processor
}

// RegisterProcessor registers a processor for a specific node type
func (p *DefaultNodeProcessor) RegisterProcessor(nodeType graph.NodeType, processor NodeTypeProcessor) {
	p.processors[nodeType] = processor
}

// Process executes a single node with the given context and input
func (p *DefaultNodeProcessor) Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error) {
	processor, exists := p.processors[node.Type]
	if !exists {
		return nil, fmt.Errorf("no processor found for node type: %s", node.Type)
	}

	return processor.Process(ctx, node, input)
}

// CanProcess returns true if this processor can handle the given node type
func (p *DefaultNodeProcessor) CanProcess(nodeType graph.NodeType) bool {
	_, exists := p.processors[nodeType]
	return exists
}

// FunctionNodeProcessor handles function nodes
type FunctionNodeProcessor struct{}

func (p *FunctionNodeProcessor) Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error) {
	// In a full implementation, this would:
	// 1. Extract function definition from node configuration
	// 2. Execute the function with the input
	// 3. Return the function output

	// For now, we'll simulate function execution
	output := make(map[string]interface{})
	for k, v := range input {
		output[k] = v
	}

	// Add a processed marker
	output["processed_by"] = node.ID
	output["node_type"] = string(node.Type)

	return output, nil
}

// ConditionalNodeProcessor handles conditional nodes
type ConditionalNodeProcessor struct{}

func (p *ConditionalNodeProcessor) Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error) {
	// In a full implementation, this would:
	// 1. Extract condition logic from node configuration
	// 2. Evaluate the condition against the input
	// 3. Return the input with condition result

	output := make(map[string]interface{})
	for k, v := range input {
		output[k] = v
	}

	// Add condition evaluation result
	output["condition_result"] = true // Simplified
	output["evaluated_by"] = node.ID

	return output, nil
}

// StartNodeProcessor handles start nodes
type ToolNodeProcessor struct{}

func (p *ToolNodeProcessor) Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error) {
	// Tool nodes execute external tools or APIs
	output := make(map[string]interface{})
	for k, v := range input {
		output[k] = v
	}

	output["tool_executed"] = node.ID
	output["tool_type"] = string(node.Type)
	return output, nil
}

// AgentNodeProcessor handles agent nodes
type AgentNodeProcessor struct{}

func (p *AgentNodeProcessor) Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error) {
	// Agent nodes handle AI agent interactions
	output := make(map[string]interface{})
	for k, v := range input {
		output[k] = v
	}

	output["agent_processed"] = node.ID
	output["agent_type"] = string(node.Type)
	return output, nil
}
