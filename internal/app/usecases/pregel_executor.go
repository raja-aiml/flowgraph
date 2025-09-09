package usecases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/graph"
	"github.com/flowgraph/flowgraph/internal/core/pregel"
	"github.com/flowgraph/flowgraph/pkg/validation"
)

// PregelGraphExecutor integrates the Pregel engine with graph execution
// PRINCIPLES:
// - BSP: Bulk Synchronous Parallel coordination
// - Parallelism: Concurrent execution of independent nodes
// - Streaming: Real-time execution monitoring
type PregelGraphExecutor struct {
	nodeProcessor     NodeProcessor
	edgeEvaluator     EdgeEvaluator
	stateManager      StateManager
	checkpointManager CheckpointManager
	graphRepository   GraphRepository
	executions        map[string]*dto.ExecutionContext
	mu                sync.RWMutex
}

// NewPregelGraphExecutor creates a new Pregel-based graph executor
func NewPregelGraphExecutor(
	nodeProcessor NodeProcessor,
	edgeEvaluator EdgeEvaluator,
	stateManager StateManager,
	checkpointManager CheckpointManager,
	graphRepository GraphRepository,
) *PregelGraphExecutor {
	return &PregelGraphExecutor{
		nodeProcessor:     nodeProcessor,
		edgeEvaluator:     edgeEvaluator,
		stateManager:      stateManager,
		checkpointManager: checkpointManager,
		graphRepository:   graphRepository,
		executions:        make(map[string]*dto.ExecutionContext),
	}
}

// GraphVertexProgram adapts FlowGraph nodes to Pregel vertex programs
type GraphVertexProgram struct {
	NodeID        string
	Processor     NodeProcessor
	EdgeEvaluator EdgeEvaluator
	GraphDef      *graph.Graph
}

// Compute implements the Pregel vertex program interface
func (v *GraphVertexProgram) Compute(vertexID string, state map[string]interface{}, messages []*pregel.Message) (map[string]interface{}, []*pregel.Message, bool, error) {
	// Find the node in graph definition
	currentNode, exists := v.GraphDef.Nodes[vertexID]
	if !exists {
		return state, nil, true, fmt.Errorf("node %s not found", vertexID)
	}

	// Process messages to update state
	for _, msg := range messages {
		// Merge message value into state
		if msg.Value != nil {
			// Assuming the value is a map[string]interface{}
			if msgData, ok := msg.Value.(map[string]interface{}); ok {
				for key, value := range msgData {
					state[key] = value
				}
			}
		}
	}

	// Execute the node
	ctx := context.Background() // TODO: pass proper context
	output, err := v.Processor.Process(ctx, currentNode, state)
	if err != nil {
		return state, nil, true, err
	}

	// Update state with output
	newState := make(map[string]interface{})
	for key, value := range state {
		newState[key] = value
	}
	for key, value := range output {
		newState[key] = value
	}

	// Find next nodes and send messages
	var outgoingMessages []*pregel.Message
	nextNodes, err := v.EdgeEvaluator.GetNextNodes(ctx, currentNode, v.GraphDef.Edges, newState)
	if err != nil {
		return newState, nil, true, err
	}

	// If no next nodes, halt this vertex
	if len(nextNodes) == 0 {
		return newState, nil, true, nil
	}

	// Send messages to next nodes
	for _, nextNode := range nextNodes {
		msg := &pregel.Message{
			From:  vertexID,
			To:    nextNode.ID,
			Value: newState,
			Step:  0, // Will be set by engine
		}
		outgoingMessages = append(outgoingMessages, msg)
	}

	return newState, outgoingMessages, false, nil // Continue execution
}

// Execute runs a graph using the Pregel engine with BSP coordination
func (e *PregelGraphExecutor) Execute(ctx context.Context, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Load graph definition
	graphDef, err := e.graphRepository.Get(ctx, req.GraphID)
	if err != nil {
		return nil, err
	}

	// Optional graph validation
	if req.Config.ValidateGraph {
		var opts []validation.GraphValidationOptions
		if req.Config.ValidateCycles {
			opts = append(opts, validation.GraphValidationOptions{CheckCycles: true})
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

	// Execute using Pregel engine
	err = e.executeWithPregelEngine(ctx, execCtx, response, graphDef)

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

// executeWithPregelEngine runs graph execution using Pregel BSP coordination
func (e *PregelGraphExecutor) executeWithPregelEngine(
	ctx context.Context,
	execCtx *dto.ExecutionContext,
	response *dto.ExecutionResponse,
	graphDef *graph.Graph,
) error {
	// Create vertex programs for each node
	vertices := make(map[string]pregel.VertexProgram)
	initialStates := make(map[string]map[string]interface{})

	// Initialize all nodes as vertices
	for _, node := range graphDef.Nodes {
		vertexProgram := &GraphVertexProgram{
			NodeID:        node.ID,
			Processor:     e.nodeProcessor,
			EdgeEvaluator: e.edgeEvaluator,
			GraphDef:      graphDef,
		}
		vertices[node.ID] = vertexProgram

		// Initialize state for each vertex
		nodeState := make(map[string]interface{})
		for key, value := range execCtx.State {
			nodeState[key] = value
		}
		initialStates[node.ID] = nodeState
	}

	// Configure Pregel engine
	parallelism := 4 // Default parallelism
	if execCtx.Config.Parallel {
		parallelism = 8 // Higher parallelism for parallel mode
	}

	pregelConfig := pregel.Config{
		MaxSupersteps: execCtx.Config.MaxSteps,
		Parallelism:   parallelism,
		Timeout:       execCtx.Config.Timeout,
		StreamOutput:  true,
	}

	if pregelConfig.MaxSupersteps <= 0 {
		pregelConfig.MaxSupersteps = 100
	}

	// Create and run Pregel engine
	engine := pregel.NewEngine(vertices, initialStates, pregelConfig)

	// Set up streaming to capture step results
	if engine.Streamer != nil {
		stepHandler := &ExecutionStepHandler{response: response}
		engine.Streamer.AddHandler(stepHandler)
	}

	// Execute the graph
	err := engine.Run(ctx)
	if err != nil {
		return fmt.Errorf("Pregel execution failed: %w", err)
	}

	// Extract final state from completed vertices
	finalState := make(map[string]interface{})
	for _, vertexState := range engine.VertexStates {
		for key, value := range vertexState {
			finalState[key] = value
		}
	}
	execCtx.State = finalState

	return nil
}

// ExecutionStepHandler implements StreamHandler to capture execution steps
type ExecutionStepHandler struct {
	response *dto.ExecutionResponse
	mu       sync.RWMutex
}

func (h *ExecutionStepHandler) HandleEvent(event pregel.StreamEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	stepResult := dto.StepResult{
		StepNumber: event.Step,
		NodeID:     fmt.Sprintf("superstep-%d", event.Step),
		NodeType:   graph.NodeType("pregel-superstep"),
		StartTime:  time.Unix(event.Timestamp, 0),
		EndTime:    time.Now(),
		Duration:   0,
		Status:     dto.StepStatusCompleted,
	}

	if event.Type == "superstep_start" {
		stepResult.Status = dto.StepStatusRunning
	} else if event.Type == "superstep_end" {
		stepResult.Status = dto.StepStatusCompleted
	}

	h.response.Steps = append(h.response.Steps, stepResult)
	return nil
}

// Resume continues execution from a checkpoint using Pregel engine
func (e *PregelGraphExecutor) Resume(ctx context.Context, checkpointID string, req *dto.ExecutionRequest) (*dto.ExecutionResponse, error) {
	// Load execution context from checkpoint
	execCtx, err := e.checkpointManager.LoadCheckpoint(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Update context with new request data
	if req.Config.MaxSteps > 0 {
		execCtx.Config.MaxSteps = req.Config.MaxSteps
	}
	if req.Config.Timeout > 0 {
		execCtx.Config.Timeout = req.Config.Timeout
	}

	// Create new execution ID for resumed execution
	newExecutionID := e.generateExecutionID(execCtx.GraphID, execCtx.ThreadID)
	execCtx.ExecutionID = newExecutionID

	// Register execution
	e.mu.Lock()
	e.executions[newExecutionID] = execCtx
	e.mu.Unlock()

	// Load graph definition
	graphDef, err := e.graphRepository.Get(ctx, execCtx.GraphID)
	if err != nil {
		return nil, err
	}

	// Initialize response
	response := &dto.ExecutionResponse{
		ExecutionID: newExecutionID,
		GraphID:     execCtx.GraphID,
		ThreadID:    execCtx.ThreadID,
		Status:      dto.ExecutionStatusRunning,
		StartTime:   time.Now(),
		Steps:       make([]dto.StepResult, 0),
	}

	// Continue execution using Pregel
	err = e.executeWithPregelEngine(ctx, execCtx, response, graphDef)

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

// Stop halts a running Pregel execution
func (e *PregelGraphExecutor) Stop(ctx context.Context, executionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.executions[executionID]; !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	// TODO: Implement proper cancellation of Pregel engine
	delete(e.executions, executionID)
	return nil
}

// GetStatus returns the current status of a Pregel execution
func (e *PregelGraphExecutor) GetStatus(ctx context.Context, executionID string) (*dto.ExecutionResponse, error) {
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

// generateExecutionID creates a unique execution ID
func (e *PregelGraphExecutor) generateExecutionID(graphID, threadID string) string {
	return fmt.Sprintf("%s-%s-%d", graphID, threadID, time.Now().UnixNano())
}
