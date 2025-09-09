package usecases

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/flowgraph/flowgraph/internal/app/dto"
    "github.com/flowgraph/flowgraph/pkg/validation"
    cgraph "github.com/flowgraph/flowgraph/internal/core/graph"
)

// DefaultGraphExecutor implements the GraphExecutor interface
// PRINCIPLES:
// - KISS: Simple, straightforward execution logic
// - SRP: Focuses only on graph execution orchestration
// - DRY: Reuses common execution patterns
type DefaultGraphExecutor struct {
	nodeProcessor     NodeProcessor
	edgeEvaluator     EdgeEvaluator
	stateManager      StateManager
	checkpointManager CheckpointManager
	graphRepository   GraphRepository
	executions        map[string]*dto.ExecutionContext
	mu                sync.RWMutex
}

// NewDefaultGraphExecutor creates a new graph executor with dependencies
func NewDefaultGraphExecutor(
	nodeProcessor NodeProcessor,
	edgeEvaluator EdgeEvaluator,
	stateManager StateManager,
	checkpointManager CheckpointManager,
	graphRepository GraphRepository,
) *DefaultGraphExecutor {
	return &DefaultGraphExecutor{
		nodeProcessor:     nodeProcessor,
		edgeEvaluator:     edgeEvaluator,
		stateManager:      stateManager,
		checkpointManager: checkpointManager,
		graphRepository:   graphRepository,
		executions:        make(map[string]*dto.ExecutionContext),
	}
}

// Execute runs a graph with the given request
func (e *DefaultGraphExecutor) Execute(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
    if err := req.Validate(); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }

    // Optional: Validate graph before creating execution context to fail fast
    if req.Config.ValidateGraph {
        var opts []validation.GraphValidationOptions
        if req.Config.ValidateCycles {
            opts = append(opts, validation.GraphValidationOptions{CheckCycles: true})
        }
        graphDef, gerr := e.graphRepository.Get(ctx, req.GraphID)
        if gerr != nil {
            return nil, gerr
        }
        if verr := validation.ValidateCoreGraph(graphDef, opts...); verr != nil {
            return nil, fmt.Errorf("graph validation failed: %w", verr)
        }
    }

	// Create execution context
	executionID := e.generateExecutionID(req.GraphID, req.ThreadID)
	execCtx := &dto.ExecutionContext{
		ExecutionID: executionID,
		GraphID:     req.GraphID,
		ThreadID:    req.ThreadID,
		CurrentStep: 0,
		State:       req.Input,
		Config:      req.Config,
		StartTime:   time.Now(),
	}

	// Register execution
	e.mu.Lock()
	e.executions[executionID] = execCtx
	e.mu.Unlock()

	// Initialize response
	response := &dto.ExecutionResponse{
		ExecutionID: executionID,
		GraphID:     req.GraphID,
		ThreadID:    req.ThreadID,
		Status:      dto.ExecutionStatusRunning,
		StartTime:   execCtx.StartTime,
		Steps:       make([]dto.StepResult, 0),
	}

	// Execute the graph
	err := e.executeGraph(ctx, execCtx, response)

	// Complete execution
	response.EndTime = time.Now()
	response.Duration = response.EndTime.Sub(response.StartTime)

	if err != nil {
		response.Status = dto.ExecutionStatusFailed
		response.Error = err.Error()
	} else {
		response.Status = dto.ExecutionStatusCompleted
		response.Output = execCtx.State
	}

	// Cleanup
	e.mu.Lock()
	delete(e.executions, executionID)
	e.mu.Unlock()

	return response, err
}

// Resume continues execution from a checkpoint
func (e *DefaultGraphExecutor) Resume(ctx context.Context, checkpointID string, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Load execution context from checkpoint
	execCtx, err := e.checkpointManager.LoadCheckpoint(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Update context with new request data if provided
	if req.Config.MaxSteps > 0 {
		execCtx.Config.MaxSteps = req.Config.MaxSteps
	}
	if req.Config.Timeout > 0 {
		execCtx.Config.Timeout = req.Config.Timeout
	}
	// Fix: Ensure CheckpointEvery is set to a non-zero default value
	if execCtx.Config.CheckpointEvery == 0 {
		execCtx.Config.CheckpointEvery = 1 // Default to every step
	}

	// Create new execution ID for resumed execution
	newExecutionID := e.generateExecutionID(execCtx.GraphID, execCtx.ThreadID)
	execCtx.ExecutionID = newExecutionID

	// Register execution
	e.mu.Lock()
	e.executions[newExecutionID] = execCtx
	e.mu.Unlock()

	// Initialize response
	response := &dto.ExecutionResponse{
		ExecutionID: newExecutionID,
		GraphID:     execCtx.GraphID,
		ThreadID:    execCtx.ThreadID,
		Status:      dto.ExecutionStatusRunning,
		StartTime:   time.Now(),
		Steps:       make([]dto.StepResult, 0),
	}

    // Continue execution
    // Validate graph before resuming if requested
    if execCtx.Config.ValidateGraph {
        var opts []validation.GraphValidationOptions
        if execCtx.Config.ValidateCycles {
            opts = append(opts, validation.GraphValidationOptions{CheckCycles: true})
        }
        graphDef, gerr := e.graphRepository.Get(ctx, execCtx.GraphID)
        if gerr != nil {
            return nil, fmt.Errorf("failed to load graph: %w", gerr)
        }
        if verr := validation.ValidateCoreGraph(graphDef, opts...); verr != nil {
            return nil, fmt.Errorf("graph validation failed: %w", verr)
        }
    }

    err = e.executeGraph(ctx, execCtx, response)

	// Complete execution
	response.EndTime = time.Now()
	response.Duration = response.EndTime.Sub(response.StartTime)

	if err != nil {
		response.Status = dto.ExecutionStatusFailed
		response.Error = err.Error()
	} else {
		response.Status = dto.ExecutionStatusCompleted
		response.Output = execCtx.State
	}

	// Cleanup
	e.mu.Lock()
	delete(e.executions, newExecutionID)
	e.mu.Unlock()

	return response, err
}

// Stop halts a running execution
func (e *DefaultGraphExecutor) Stop(ctx context.Context, executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.executions[executionID]; !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	// In a full implementation, this would cancel the context
	// For now, we'll mark it as stopped
	delete(e.executions, executionID)
	return nil
}

// GetStatus returns the current status of an execution
func (e *DefaultGraphExecutor) GetStatus(ctx context.Context, executionID string) (*dto.ExecutionResponse, error) {
	e.mu.RLock()
	execCtx, exists := e.executions[executionID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("execution %s not found", executionID)
	}

	return &dto.ExecutionResponse{
		ExecutionID: executionID,
		GraphID:     execCtx.GraphID,
		ThreadID:    execCtx.ThreadID,
		Status:      dto.ExecutionStatusRunning,
		StartTime:   execCtx.StartTime,
	}, nil
}

// executeGraph orchestrates the execution of a graph
func (e *DefaultGraphExecutor) executeGraph(ctx context.Context, execCtx *dto.ExecutionContext, response *dto.ExecutionResponse) error {
    graphDef, err := e.graphRepository.Get(ctx, execCtx.GraphID)
    if err != nil {
        return err
    }

    currentNode := graphDef.Nodes[graphDef.EntryPoint]
    maxSteps := e.maxStepsOrDefault(execCtx)

    for step := 0; step < maxSteps && currentNode != nil; step++ {
        var next *dto.StepResult
        currentNode, next, err = e.runSequentialStep(ctx, execCtx, graphDef, currentNode, step)
        if err != nil {
            return err
        }
        if next != nil {
            response.Steps = append(response.Steps, *next)
        }
        execCtx.CurrentStep++
    }
    return nil
}

func (e *DefaultGraphExecutor) maxStepsOrDefault(execCtx *dto.ExecutionContext) int {
    if execCtx.Config.MaxSteps > 0 {
        return execCtx.Config.MaxSteps
    }
    return 100
}

// runSequentialStep executes a single node step and returns the next node and the recorded step result.
func (e *DefaultGraphExecutor) runSequentialStep(
    ctx context.Context,
    execCtx *dto.ExecutionContext,
    graphDef *cgraph.Graph,
    current *cgraph.Node,
    step int,
) (*cgraph.Node, *dto.StepResult, error) {
    // Convert adapters to internal graph types without importing graph here by using shims.
    // However, we can avoid shims by defining minimal interfaces: we already did above.
    // But here we'll downcast to maintain original behavior using helper wrappers.
    // Build a small wrapper struct to keep this function isolated.
    // To avoid over-engineering, use the actual types via type assertions.

    // Get real types
    // Note: We rely on the exact internal types used when calling this function.
    // This preserves behavior without adding cross-package imports in this section.

    // Execute node
    stepStart := time.Now()
    output, err := e.nodeProcessor.Process(ctx, current, execCtx.State)
    if err != nil {
        return nil, nil, fmt.Errorf("node %s execution failed: %w", current.ID, err)
    }
    execCtx.State = output

    // Checkpoint policy
    if e.checkpointManager != nil && (step%execCtx.Config.CheckpointEvery == 0) {
        _, _ = e.checkpointManager.CreateCheckpoint(ctx, execCtx)
    }

    // Record step
    rec := &dto.StepResult{
        StepNumber: step + 1,
        NodeID:     current.ID,
        NodeType:   current.Type,
        Input:      execCtx.State,
        Output:     output,
        StartTime:  stepStart,
        EndTime:    time.Now(),
        Duration:   time.Since(stepStart),
        Status:     dto.StepStatusCompleted,
    }

    // Next node
    nextNodes, err := e.edgeEvaluator.GetNextNodes(ctx, current, graphDef.Edges, execCtx.State)
    if err != nil || len(nextNodes) == 0 {
        return nil, rec, nil
    }
    return nextNodes[0], rec, nil
}

// generateExecutionID creates a unique execution ID
func (e *DefaultGraphExecutor) generateExecutionID(graphID, threadID string) string {
	return fmt.Sprintf("%s-%s-%d", graphID, threadID, time.Now().UnixNano())
}
