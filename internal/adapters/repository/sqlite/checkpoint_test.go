package sqlite

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/flowgraph/flowgraph/pkg/serialization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteCheckpointSaver(t *testing.T) {
	// Create in-memory database
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Create serializer
	serializer := serialization.DefaultSerializer()

	// Create checkpoint saver
	saver := NewCheckpointSaver(db, serializer)

	// Create tables
	err = saver.CreateTables(ctx)
	require.NoError(t, err)

	// Test Save and Load
	cp := &checkpoint.Checkpoint{
		ID:       "test-1",
		GraphID:  "graph-1",
		ThreadID: "thread-1",
		State: map[string]interface{}{
			"step":  1,
			"value": "test",
		},
		Metadata: checkpoint.Metadata{
			Step:   1,
			Source: "test",
		},
		Timestamp: time.Now(),
		Version:   "1.0",
	}

	// Save checkpoint
	err = saver.Save(ctx, cp)
	require.NoError(t, err)

	// Load checkpoint
	loaded, err := saver.Load(ctx, "test-1")
	require.NoError(t, err)
	assert.Equal(t, cp.ID, loaded.ID)
	assert.Equal(t, cp.GraphID, loaded.GraphID)
	assert.Equal(t, cp.ThreadID, loaded.ThreadID)
	assert.Equal(t, cp.State["value"], loaded.State["value"])
	assert.Equal(t, cp.Metadata.Step, loaded.Metadata.Step)

	// Test List
	filter := checkpoint.Filter{
		GraphID: "graph-1",
		Limit:   10,
	}
	checkpoints, err := saver.List(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, checkpoints, 1)
	assert.Equal(t, "test-1", checkpoints[0].ID)

	// Test Delete
	err = saver.Delete(ctx, "test-1")
	require.NoError(t, err)

	// Verify deleted
	_, err = saver.Load(ctx, "test-1")
	assert.Equal(t, checkpoint.ErrCheckpointNotFound, err)
}

func TestSQLiteCheckpointSaver_Errors(t *testing.T) {
	ctx := context.Background()
	serializer := serialization.DefaultSerializer()

	// Create saver with nil database
	saver := &CheckpointSaver{
		db:         nil,
		serializer: serializer,
		tableName:  "checkpoints",
	}

	// Test Save with nil checkpoint
	err := saver.Save(ctx, nil)
	assert.Equal(t, checkpoint.ErrInvalidCheckpointID, err)

	// Test Load with empty ID
	_, err = saver.Load(ctx, "")
	assert.Equal(t, checkpoint.ErrInvalidCheckpointID, err)

	// Test Delete with empty ID
	err = saver.Delete(ctx, "")
	assert.Equal(t, checkpoint.ErrInvalidCheckpointID, err)
}
