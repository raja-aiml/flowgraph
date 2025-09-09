// +build integration

// Package integration contains integration tests for FlowGraph
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCheckpointSaver implements checkpoint.Saver for testing
type MockCheckpointSaver struct {
	checkpoints map[string]*checkpoint.Checkpoint
}

// NewMockCheckpointSaver creates a new mock checkpoint saver
func NewMockCheckpointSaver() *MockCheckpointSaver {
	return &MockCheckpointSaver{
		checkpoints: make(map[string]*checkpoint.Checkpoint),
	}
}

func (m *MockCheckpointSaver) Save(ctx context.Context, cp *checkpoint.Checkpoint) error {
	if err := cp.Validate(); err != nil {
		return err
	}
	m.checkpoints[cp.ID] = cp
	return nil
}

func (m *MockCheckpointSaver) Load(ctx context.Context, id string) (*checkpoint.Checkpoint, error) {
	cp, exists := m.checkpoints[id]
	if !exists {
		return nil, checkpoint.ErrCheckpointNotFound
	}
	return cp, nil
}

func (m *MockCheckpointSaver) List(ctx context.Context, filter checkpoint.Filter) ([]*checkpoint.Checkpoint, error) {
	if err := filter.Validate(); err != nil {
		return nil, err
	}
	
	var result []*checkpoint.Checkpoint
	for _, cp := range m.checkpoints {
		// Apply filter
		if filter.GraphID != "" && cp.GraphID != filter.GraphID {
			continue
		}
		if filter.ThreadID != "" && cp.ThreadID != filter.ThreadID {
			continue
		}
		result = append(result, cp)
	}
	
	// Apply limit
	if filter.Limit > 0 && len(result) > filter.Limit {
		result = result[:filter.Limit]
	}
	
	return result, nil
}

func (m *MockCheckpointSaver) Delete(ctx context.Context, id string) error {
	if _, exists := m.checkpoints[id]; !exists {
		return checkpoint.ErrCheckpointNotFound
	}
	delete(m.checkpoints, id)
	return nil
}

func TestCheckpointSaver_Integration(t *testing.T) {
	ctx := context.Background()
	saver := NewMockCheckpointSaver()
	
	t.Run("Save and Load checkpoint", func(t *testing.T) {
		cp := &checkpoint.Checkpoint{
			ID:       "test-1",
			GraphID:  "graph-1",
			ThreadID: "thread-1",
			State: map[string]interface{}{
				"key": "value",
			},
			Timestamp: time.Now(),
		}
		
		// Save
		err := saver.Save(ctx, cp)
		require.NoError(t, err)
		
		// Load
		loaded, err := saver.Load(ctx, "test-1")
		require.NoError(t, err)
		assert.Equal(t, cp.ID, loaded.ID)
		assert.Equal(t, cp.GraphID, loaded.GraphID)
	})
	
	t.Run("List checkpoints with filter", func(t *testing.T) {
		// Add more checkpoints
		for i := 2; i <= 5; i++ {
			cp := &checkpoint.Checkpoint{
				ID:       fmt.Sprintf("test-%d", i),
				GraphID:  "graph-1",
				ThreadID: fmt.Sprintf("thread-%d", i),
				State:    map[string]interface{}{"index": i},
				Timestamp: time.Now(),
			}
			err := saver.Save(ctx, cp)
			require.NoError(t, err)
		}
		
		// List all for graph-1
		filter := checkpoint.Filter{
			GraphID: "graph-1",
		}
		list, err := saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 5)
		
		// List with limit
		filter.Limit = 3
		list, err = saver.List(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, list, 3)
	})
	
	t.Run("Delete checkpoint", func(t *testing.T) {
		err := saver.Delete(ctx, "test-1")
		require.NoError(t, err)
		
		// Try to load deleted checkpoint
		_, err = saver.Load(ctx, "test-1")
		assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
		
		// Try to delete non-existent checkpoint
		err = saver.Delete(ctx, "non-existent")
		assert.ErrorIs(t, err, checkpoint.ErrCheckpointNotFound)
	})
	
	t.Run("Validation errors", func(t *testing.T) {
		// Invalid checkpoint
		cp := &checkpoint.Checkpoint{
			ID: "", // Invalid: empty ID
		}
		err := saver.Save(ctx, cp)
		assert.Error(t, err)
		
		// Invalid filter
		filter := checkpoint.Filter{
			Limit: -1, // Invalid: negative limit
		}
		_, err = saver.List(ctx, filter)
		assert.Error(t, err)
	})
}

func TestCheckpointValidation(t *testing.T) {
	tests := []struct {
		name      string
		checkpoint *checkpoint.Checkpoint
		wantErr   error
	}{
		{
			name: "valid checkpoint",
			checkpoint: &checkpoint.Checkpoint{
				ID:       "test-1",
				GraphID:  "graph-1",
				ThreadID: "thread-1",
				State:    map[string]interface{}{"key": "value"},
			},
			wantErr: nil,
		},
		{
			name: "missing ID",
			checkpoint: &checkpoint.Checkpoint{
				GraphID:  "graph-1",
				ThreadID: "thread-1",
				State:    map[string]interface{}{},
			},
			wantErr: checkpoint.ErrInvalidCheckpointID,
		},
		{
			name: "missing GraphID",
			checkpoint: &checkpoint.Checkpoint{
				ID:       "test-1",
				ThreadID: "thread-1",
				State:    map[string]interface{}{},
			},
			wantErr: checkpoint.ErrInvalidGraphID,
		},
		{
			name: "nil state",
			checkpoint: &checkpoint.Checkpoint{
				ID:       "test-1",
				GraphID:  "graph-1",
				ThreadID: "thread-1",
				State:    nil,
			},
			wantErr: checkpoint.ErrNilState,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.checkpoint.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}