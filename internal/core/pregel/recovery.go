package pregel

import (
	"fmt"
	"sync"
	"time"
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	OriginalError error
	VertexID      string
	Attempt       int
}

func (re *RetryableError) Error() string {
	return fmt.Sprintf("vertex %s failed on attempt %d: %v", re.VertexID, re.Attempt, re.OriginalError)
}

// ErrorRecoveryHandler handles vertex execution errors
type ErrorRecoveryHandler struct {
	policy RetryPolicy
}

func NewErrorRecoveryHandler(policy RetryPolicy) *ErrorRecoveryHandler {
	return &ErrorRecoveryHandler{policy: policy}
}

func (erh *ErrorRecoveryHandler) HandleError(vertexID string, err error, attempt int) (bool, error) {
	if attempt >= erh.policy.MaxRetries {
		return false, &RetryableError{
			OriginalError: err,
			VertexID:      vertexID,
			Attempt:       attempt,
		}
	}

	// Exponential backoff
	if erh.policy.BackoffMs > 0 {
		backoffTime := time.Duration(erh.policy.BackoffMs*(1<<attempt)) * time.Millisecond
		time.Sleep(backoffTime)
	}

	return true, nil
}

// CheckpointManager handles state persistence for recovery
type CheckpointManager struct {
	checkpoints    map[string]map[string]interface{}
	mu             sync.RWMutex
	maxCheckpoints int
}

func NewCheckpointManager() *CheckpointManager {
	return &CheckpointManager{
		checkpoints:    make(map[string]map[string]interface{}),
		maxCheckpoints: 10, // Keep last 10 checkpoints
	}
}

func (cm *CheckpointManager) SaveCheckpoint(superstep int, states map[string]map[string]interface{}) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	checkpointKey := fmt.Sprintf("step_%d", superstep)
	cm.checkpoints[checkpointKey] = make(map[string]interface{})
	for vertexID, state := range states {
		// Deep copy the state
		stateCopy := make(map[string]interface{})
		for k, v := range state {
			stateCopy[k] = v
		}
		cm.checkpoints[checkpointKey][vertexID] = stateCopy
	}

	// Clean up old checkpoints if we exceed max
	if len(cm.checkpoints) > cm.maxCheckpoints {
		cm.cleanupOldCheckpoints(superstep)
	}
}

func (cm *CheckpointManager) LoadCheckpoint(superstep int) (map[string]map[string]interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	checkpointKey := fmt.Sprintf("step_%d", superstep)
	checkpoint, exists := cm.checkpoints[checkpointKey]
	if !exists {
		return nil, false
	}

	states := make(map[string]map[string]interface{})
	for vertexID, state := range checkpoint {
		if stateMap, ok := state.(map[string]interface{}); ok {
			// Deep copy the state
			stateCopy := make(map[string]interface{})
			for k, v := range stateMap {
				stateCopy[k] = v
			}
			states[vertexID] = stateCopy
		}
	}
	return states, true
}

func (cm *CheckpointManager) cleanupOldCheckpoints(currentStep int) {
	// Remove checkpoints that are too old
	for key := range cm.checkpoints {
		var step int
		if _, err := fmt.Sscanf(key, "step_%d", &step); err == nil {
			if currentStep-step > cm.maxCheckpoints {
				delete(cm.checkpoints, key)
			}
		}
	}
}

func (cm *CheckpointManager) ListCheckpoints() []int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var steps []int
	for key := range cm.checkpoints {
		var step int
		if _, err := fmt.Sscanf(key, "step_%d", &step); err == nil {
			steps = append(steps, step)
		}
	}
	return steps
}

func (cm *CheckpointManager) DeleteCheckpoint(superstep int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	checkpointKey := fmt.Sprintf("step_%d", superstep)
	delete(cm.checkpoints, checkpointKey)
}

func (cm *CheckpointManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.checkpoints = make(map[string]map[string]interface{})
}

// ResilientEngine extends Engine with error recovery capabilities
type ResilientEngine struct {
	*Engine
	ErrorHandler      *ErrorRecoveryHandler
	CheckpointManager *CheckpointManager
	retryAttempts     map[string]int
}

func NewResilientEngine(vertices map[string]VertexProgram, initialStates map[string]map[string]interface{}, config Config) *ResilientEngine {
	engine := NewEngine(vertices, initialStates, config)
	return &ResilientEngine{
		Engine:            engine,
		ErrorHandler:      NewErrorRecoveryHandler(config.RetryPolicy),
		CheckpointManager: NewCheckpointManager(),
		retryAttempts:     make(map[string]int),
	}
}
