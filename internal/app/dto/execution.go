package dto

import (
	"time"

	"github.com/flowgraph/flowgraph/internal/core/graph"
)

// ExecutionRequest represents a request to execute a graph
type ExecutionRequest struct {
	GraphID    string                 `json:"graph_id"`
	ThreadID   string                 `json:"thread_id"`
	Input      map[string]interface{} `json:"input"`
	Config     ExecutionConfig        `json:"config"`
	ResumeFrom string                 `json:"resume_from,omitempty"` // Checkpoint ID to resume from
}

// ExecutionConfig contains configuration for graph execution
type ExecutionConfig struct {
    MaxSteps        int           `json:"max_steps"`        // Maximum number of steps to execute
    Timeout         time.Duration `json:"timeout"`          // Execution timeout
    CheckpointEvery int           `json:"checkpoint_every"` // Save checkpoint every N steps
    Parallel        bool          `json:"parallel"`         // Enable parallel execution where possible
    DebugMode       bool          `json:"debug_mode"`       // Enable debug logging
    ValidateGraph   bool          `json:"validate_graph"`   // Validate graph structure before execute
    ValidateCycles  bool          `json:"validate_cycles"`  // Enable cycle detection during validation
}

// ExecutionResponse represents the response from graph execution
type ExecutionResponse struct {
	ExecutionID string                 `json:"execution_id"`
	GraphID     string                 `json:"graph_id"`
	ThreadID    string                 `json:"thread_id"`
	Status      ExecutionStatus        `json:"status"`
	Output      map[string]interface{} `json:"output"`
	Steps       []StepResult           `json:"steps"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Error       string                 `json:"error,omitempty"`
}

// ExecutionStatus represents the status of graph execution
type ExecutionStatus string

const (
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusStopped   ExecutionStatus = "stopped"
)

// StepResult represents the result of executing a single step
type StepResult struct {
	StepNumber   int                    `json:"step_number"`
	NodeID       string                 `json:"node_id"`
	NodeType     graph.NodeType         `json:"node_type"`
	Input        map[string]interface{} `json:"input"`
	Output       map[string]interface{} `json:"output"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Duration     time.Duration          `json:"duration"`
	Status       StepStatus             `json:"status"`
	Error        string                 `json:"error,omitempty"`
	CheckpointID string                 `json:"checkpoint_id,omitempty"`
}

// StepStatus represents the status of a single step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
)

// ExecutionContext holds the context for graph execution
type ExecutionContext struct {
	ExecutionID string
	GraphID     string
	ThreadID    string
	CurrentStep int
	State       map[string]interface{}
	Config      ExecutionConfig
	StartTime   time.Time
}

// Validate validates the execution request
func (req *ExecutionRequest) Validate() error {
	if req.GraphID == "" {
		return ErrMissingGraphID
	}
	if req.ThreadID == "" {
		return ErrMissingThreadID
	}
	if req.Config.MaxSteps <= 0 {
		req.Config.MaxSteps = 100 // Default value
	}
	if req.Config.Timeout <= 0 {
		req.Config.Timeout = 5 * time.Minute // Default timeout
	}
    if req.Config.CheckpointEvery <= 0 {
        req.Config.CheckpointEvery = 10 // Default checkpoint frequency
    }
    // Default: validate graph structure before executing
    if !req.Config.ValidateGraph {
        // If caller didn't set it explicitly, enable by default
        req.Config.ValidateGraph = true
    }
    return nil
}
