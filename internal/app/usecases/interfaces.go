package usecases

import (
	"context"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/graph"
)

// GraphRepository defines the interface for graph storage and retrieval
// PRINCIPLES:
// - SRP: Only responsible for graph persistence
// - DIP: Used for dependency injection
type GraphRepository interface {
	Save(ctx context.Context, g *graph.Graph) error
	Get(ctx context.Context, id string) (*graph.Graph, error)
	List(ctx context.Context) ([]*graph.Graph, error)
}

// GraphExecutor defines the interface for executing graphs
// PRINCIPLES:
// - SRP: Single responsibility for graph execution orchestration
// - OCP: Open for extension with different execution strategies
// - DIP: Depends on abstractions, not concretions
type GraphExecutor interface {
	// Execute runs a graph with the given request
	Execute(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error)

	// Resume continues execution from a checkpoint
	Resume(ctx context.Context, checkpointID string, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error)

	// Stop halts a running execution
	Stop(ctx context.Context, executionID string) error

	// GetStatus returns the current status of an execution
	GetStatus(ctx context.Context, executionID string) (*dto.ExecutionResponse, error)
}

// NodeProcessor defines the interface for processing individual nodes
type NodeProcessor interface {
	// Process executes a single node with the given context and input
	Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error)

	// CanProcess returns true if this processor can handle the given node type
	CanProcess(nodeType graph.NodeType) bool
}

// EdgeEvaluator defines the interface for evaluating edge conditions
type EdgeEvaluator interface {
	// Evaluate returns true if the edge condition is satisfied
	Evaluate(ctx context.Context, edge *graph.Edge, state map[string]interface{}) (bool, error)

	// GetNextNodes returns the next nodes to execute based on edge evaluation
	GetNextNodes(ctx context.Context, currentNode *graph.Node, edges []*graph.Edge, state map[string]interface{}) ([]*graph.Node, error)
}

// StateManager defines the interface for managing execution state
type StateManager interface {
	// SaveState saves the current execution state
	SaveState(ctx context.Context, executionCtx *dto.ExecutionContext) error

	// LoadState loads execution state for resuming
	LoadState(ctx context.Context, executionID string) (*dto.ExecutionContext, error)

	// UpdateState updates the execution state with new data
	UpdateState(ctx context.Context, executionID string, updates map[string]interface{}) error

	// CleanupState removes execution state after completion
	CleanupState(ctx context.Context, executionID string) error
}

// CheckpointManager defines the interface for checkpoint operations during execution
type CheckpointManager interface {
	// CreateCheckpoint creates a checkpoint at the current execution state
	CreateCheckpoint(ctx context.Context, executionCtx *dto.ExecutionContext) (string, error)

	// LoadCheckpoint loads execution state from a checkpoint
	LoadCheckpoint(ctx context.Context, checkpointID string) (*dto.ExecutionContext, error)

	// ListCheckpoints returns available checkpoints for a thread
	ListCheckpoints(ctx context.Context, threadID string) ([]string, error)
}
