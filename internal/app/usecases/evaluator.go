package usecases

import (
	"context"
	"fmt"

	"github.com/flowgraph/flowgraph/internal/core/graph"
)

// DefaultEdgeEvaluator implements the EdgeEvaluator interface
// PRINCIPLES:
// - SRP: Only responsible for edge condition evaluation
// - OCP: Extensible for different condition types
// - DRY: Reuses common evaluation patterns
type DefaultEdgeEvaluator struct{}

// NewDefaultEdgeEvaluator creates a new edge evaluator
func NewDefaultEdgeEvaluator() *DefaultEdgeEvaluator {
	return &DefaultEdgeEvaluator{}
}

// Evaluate returns true if the edge condition is satisfied
func (e *DefaultEdgeEvaluator) Evaluate(ctx context.Context, edge *graph.Edge, state map[string]interface{}) (bool, error) {
	// If no condition is specified, always evaluate to true
	if edge.Condition == "" {
		return true, nil
	}

	// In a full implementation, this would:
	// 1. Parse the condition expression
	// 2. Evaluate it against the state
	// 3. Return the boolean result

	// For now, we'll implement simple condition evaluation
	return e.evaluateSimpleCondition(edge.Condition, state)
}

// GetNextNodes returns the next nodes to execute based on edge evaluation
func (e *DefaultEdgeEvaluator) GetNextNodes(ctx context.Context, currentNode *graph.Node, edges []*graph.Edge, state map[string]interface{}) ([]*graph.Node, error) {
	var nextNodes []*graph.Node

	for _, edge := range edges {
		// Skip edges that don't start from the current node
		if edge.Source != currentNode.ID {
			continue
		}

		// Evaluate edge condition
		shouldTraverse, err := e.Evaluate(ctx, edge, state)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate edge %s: %w", edge.ID, err)
		}

		if shouldTraverse {
			// In a full implementation, we'd load the target node from a repository
			// For now, we'll create a placeholder node
			targetNode := &graph.Node{
				ID:   edge.Target,
				Type: graph.NodeTypeFunction,
				Name: fmt.Sprintf("Node-%s", edge.Target),
			}
			nextNodes = append(nextNodes, targetNode)
		}
	}

	return nextNodes, nil
}

// evaluateSimpleCondition evaluates simple conditions
func (e *DefaultEdgeEvaluator) evaluateSimpleCondition(condition string, state map[string]interface{}) (bool, error) {
	// Simple condition evaluation examples:
	// "status == 'success'" -> check if state["status"] == "success"
	// "count > 10" -> check if state["count"] > 10
	// "has_error" -> check if state["has_error"] is truthy

	switch condition {
	case "always":
		return true, nil
	case "never":
		return false, nil
	case "has_error":
		if errVal, exists := state["error"]; exists {
			return errVal != nil, nil
		}
		return false, nil
	case "no_error":
		if errVal, exists := state["error"]; exists {
			return errVal == nil, nil
		}
		return true, nil
	case "success":
		if status, exists := state["status"]; exists {
			return status == "success", nil
		}
		return false, nil
	case "failure":
		if status, exists := state["status"]; exists {
			return status == "failure" || status == "failed", nil
		}
		return false, nil
	default:
		// For more complex conditions, we'd need a proper expression evaluator
		// For now, default to true for unknown conditions
		return true, nil
	}
}
