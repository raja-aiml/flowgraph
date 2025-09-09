package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/flowgraph/flowgraph/internal/app/dto"
)

// StateService implements the StateManager interface
// PRINCIPLES:
// - SRP: Manages execution state lifecycle
// - KISS: Simple in-memory state storage
// - OCP: Extensible for persistent state storage
type StateService struct {
	states map[string]*dto.ExecutionContext
	mu     sync.RWMutex
}

// NewStateService creates a new state service
func NewStateService() *StateService {
	return &StateService{
		states: make(map[string]*dto.ExecutionContext),
	}
}

// SaveState saves the current execution state
func (s *StateService) SaveState(ctx context.Context, executionCtx *dto.ExecutionContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a copy to avoid concurrent modification
	stateCopy := &dto.ExecutionContext{
		ExecutionID: executionCtx.ExecutionID,
		GraphID:     executionCtx.GraphID,
		ThreadID:    executionCtx.ThreadID,
		CurrentStep: executionCtx.CurrentStep,
		State:       make(map[string]interface{}),
		Config:      executionCtx.Config,
		StartTime:   executionCtx.StartTime,
	}

	// Deep copy the state map
	for k, v := range executionCtx.State {
		stateCopy.State[k] = v
	}

	s.states[executionCtx.ExecutionID] = stateCopy
	return nil
}

// LoadState loads execution state for resuming
func (s *StateService) LoadState(ctx context.Context, executionID string) (*dto.ExecutionContext, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, exists := s.states[executionID]
	if !exists {
		return nil, fmt.Errorf("execution state not found: %s", executionID)
	}

	// Return a copy to avoid concurrent modification
	stateCopy := &dto.ExecutionContext{
		ExecutionID: state.ExecutionID,
		GraphID:     state.GraphID,
		ThreadID:    state.ThreadID,
		CurrentStep: state.CurrentStep,
		State:       make(map[string]interface{}),
		Config:      state.Config,
		StartTime:   state.StartTime,
	}

	// Deep copy the state map
	for k, v := range state.State {
		stateCopy.State[k] = v
	}

	return stateCopy, nil
}

// UpdateState updates the execution state with new data
func (s *StateService) UpdateState(ctx context.Context, executionID string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.states[executionID]
	if !exists {
		return fmt.Errorf("execution state not found: %s", executionID)
	}

	// Apply updates to the state
	for k, v := range updates {
		state.State[k] = v
	}

	return nil
}

// CleanupState removes execution state after completion
func (s *StateService) CleanupState(ctx context.Context, executionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, executionID)
	return nil
}

// GetActiveStates returns the number of active execution states (for monitoring)
func (s *StateService) GetActiveStates() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.states)
}
