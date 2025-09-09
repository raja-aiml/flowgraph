package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/channel"
)

// EnhancedStateService provides advanced state management with channel integration
// PRINCIPLES:
// - Integration with channel-based communication
// - Support for distributed state management
// - Real-time state synchronization
type EnhancedStateService struct {
	states           map[string]*dto.ExecutionContext
	channels         map[string]channel.Channel
	stateChangeHooks []StateChangeHook
	mu               sync.RWMutex
}

// StateChangeHook is called when execution state changes
type StateChangeHook func(executionID string, oldState, newState *dto.ExecutionContext) error

// NewEnhancedStateService creates an enhanced state service
func NewEnhancedStateService() *EnhancedStateService {
	return &EnhancedStateService{
		states:           make(map[string]*dto.ExecutionContext),
		channels:         make(map[string]channel.Channel),
		stateChangeHooks: make([]StateChangeHook, 0),
	}
}

// AddStateChangeHook registers a callback for state changes
func (s *EnhancedStateService) AddStateChangeHook(hook StateChangeHook) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stateChangeHooks = append(s.stateChangeHooks, hook)
}

// CreateChannel creates a communication channel for state updates
func (s *EnhancedStateService) CreateChannel(executionID string, channelType string) (channel.Channel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var ch channel.Channel

	switch channelType {
	case "inmemory":
		ch = channel.DefaultInMemoryChannel()
	case "buffered":
		ch = channel.DefaultBufferedChannel()
	case "persistent":
		persistentCh, err := channel.DefaultPersistentChannel("state-" + executionID)
		if err != nil {
			return nil, fmt.Errorf("failed to create persistent channel: %w", err)
		}
		ch = persistentCh
	default:
		return nil, fmt.Errorf("unsupported channel type: %s", channelType)
	}

	s.channels[executionID] = ch
	return ch, nil
}

// SaveStateWithChannel saves state and broadcasts changes via channel
func (s *EnhancedStateService) SaveStateWithChannel(ctx context.Context, executionCtx *dto.ExecutionContext) error {
	s.mu.Lock()
	oldState := s.states[executionCtx.ExecutionID]
	s.mu.Unlock()

	// Create a copy to avoid concurrent modification
	stateCopy := s.copyExecutionContext(executionCtx)

	s.mu.Lock()
	s.states[executionCtx.ExecutionID] = stateCopy
	s.mu.Unlock()

	// Notify hooks about state change
	for _, hook := range s.stateChangeHooks {
		if err := hook(executionCtx.ExecutionID, oldState, stateCopy); err != nil {
			// Log error but continue
			fmt.Printf("State change hook error: %v\n", err)
		}
	}

	// Broadcast state change via channel if available
	if ch, exists := s.channels[executionCtx.ExecutionID]; exists {
		stateMessage := channel.Message{
			ID:     fmt.Sprintf("state-update-%d", time.Now().UnixNano()),
			Type:   channel.MessageTypeData,
			Source: "state-service",
			Target: executionCtx.ExecutionID,
			Payload: map[string]interface{}{
				"execution_id": executionCtx.ExecutionID,
				"step":         executionCtx.CurrentStep,
				"state":        executionCtx.State,
				"timestamp":    time.Now(),
			},
		}

		if err := ch.Send(ctx, stateMessage); err != nil {
			return fmt.Errorf("failed to broadcast state change: %w", err)
		}
	}

	return nil
}

// LoadStateWithChannel loads state and sets up channel subscription
func (s *EnhancedStateService) LoadStateWithChannel(ctx context.Context, executionID string) (*dto.ExecutionContext, error) {
	s.mu.RLock()
	state, exists := s.states[executionID]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("execution state not found: %s", executionID)
	}

	// Return a copy to avoid concurrent modification
	return s.copyExecutionContext(state), nil
}

// UpdateStateViaChannel updates state by receiving from channel
func (s *EnhancedStateService) UpdateStateViaChannel(ctx context.Context, executionID string) error {
	s.mu.RLock()
	ch, exists := s.channels[executionID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no channel found for execution: %s", executionID)
	}

	// Receive state update from channel
	update, err := ch.Receive(ctx)
	if err != nil {
		return fmt.Errorf("failed to receive state update: %w", err)
	}

	// Apply update to state
	if updateMap, ok := update.Payload.(map[string]interface{}); ok {
		if stateData, ok := updateMap["state"].(map[string]interface{}); ok {
			return s.UpdateState(ctx, executionID, stateData)
		}
	}

	return fmt.Errorf("invalid state update format")
}

// SaveState implements the basic StateManager interface
func (s *EnhancedStateService) SaveState(ctx context.Context, executionCtx *dto.ExecutionContext) error {
	return s.SaveStateWithChannel(ctx, executionCtx)
}

// LoadState implements the basic StateManager interface
func (s *EnhancedStateService) LoadState(ctx context.Context, executionID string) (*dto.ExecutionContext, error) {
	return s.LoadStateWithChannel(ctx, executionID)
}

// UpdateState updates the execution state with new data
func (s *EnhancedStateService) UpdateState(ctx context.Context, executionID string, updates map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, exists := s.states[executionID]
	if !exists {
		return fmt.Errorf("execution state not found: %s", executionID)
	}

	oldState := s.copyExecutionContext(state)

	// Apply updates to the state
	for k, v := range updates {
		state.State[k] = v
	}

	// Notify hooks about state change
	for _, hook := range s.stateChangeHooks {
		if err := hook(executionID, oldState, state); err != nil {
			// Log error but continue
			fmt.Printf("State change hook error: %v\n", err)
		}
	}

	return nil
}

// CleanupState removes execution state and closes channels
func (s *EnhancedStateService) CleanupState(ctx context.Context, executionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close channel if exists
	if ch, exists := s.channels[executionID]; exists {
		if err := ch.Close(); err != nil {
			// Log error but continue cleanup
			fmt.Printf("Error closing channel: %v\n", err)
		}
		delete(s.channels, executionID)
	}

	delete(s.states, executionID)
	return nil
}

// GetActiveStates returns the number of active execution states
func (s *EnhancedStateService) GetActiveStates() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.states)
}

// GetActiveChannels returns the number of active channels
func (s *EnhancedStateService) GetActiveChannels() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.channels)
}

// copyExecutionContext creates a deep copy of ExecutionContext
func (s *EnhancedStateService) copyExecutionContext(execCtx *dto.ExecutionContext) *dto.ExecutionContext {
	if execCtx == nil {
		return nil
	}

	copyCtx := &dto.ExecutionContext{
		ExecutionID: execCtx.ExecutionID,
		GraphID:     execCtx.GraphID,
		ThreadID:    execCtx.ThreadID,
		CurrentStep: execCtx.CurrentStep,
		State:       make(map[string]interface{}),
		Config:      execCtx.Config,
		StartTime:   execCtx.StartTime,
	}

	// Deep copy the state map
	for k, v := range execCtx.State {
		copyCtx.State[k] = v
	}

	return copyCtx
}
