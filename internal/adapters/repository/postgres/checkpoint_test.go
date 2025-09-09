package postgres

import (
	"context"
	"testing"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/flowgraph/flowgraph/pkg/serialization"
	"github.com/stretchr/testify/assert"
)

func TestPostgresCheckpointSaver(t *testing.T) {
	t.Skip("Integration test requires PostgreSQL database")

	// This test would require actual PostgreSQL instance
	// For CI/CD, this should be run with docker-compose or testcontainers
}

func TestPostgresCheckpointSaver_Errors(t *testing.T) {
	ctx := context.Background()
	serializer := serialization.DefaultSerializer()

	// Create saver with invalid pool (will fail on operations)
	saver := &CheckpointSaver{
		pool:       nil,
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
