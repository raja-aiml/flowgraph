package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/flowgraph/flowgraph/pkg/serialization"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CheckpointSaver implements checkpoint.Saver interface for PostgreSQL
type CheckpointSaver struct {
	pool       *pgxpool.Pool
	serializer *serialization.Serializer
	tableName  string
}

// NewCheckpointSaver creates a new PostgreSQL checkpoint saver
func NewCheckpointSaver(pool *pgxpool.Pool, serializer *serialization.Serializer) *CheckpointSaver {
	return &CheckpointSaver{
		pool:       pool,
		serializer: serializer,
		tableName:  "checkpoints",
	}
}

// Save stores a checkpoint in PostgreSQL
func (s *CheckpointSaver) Save(ctx context.Context, cp *checkpoint.Checkpoint) error {
	if cp == nil {
		return checkpoint.ErrInvalidCheckpointID
	}

	// Serialize the checkpoint state
	data, err := s.serializer.Serialize(cp.State)
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint state: %w", err)
	}

	// Serialize metadata
	metadataJSON, err := json.Marshal(cp.Metadata)
	if err != nil {
		return fmt.Errorf("failed to serialize metadata: %w", err)
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, graph_id, thread_id, state, metadata, timestamp, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			state = EXCLUDED.state,
			metadata = EXCLUDED.metadata,
			timestamp = EXCLUDED.timestamp
	`, s.tableName)

	_, err = s.pool.Exec(ctx, query,
		cp.ID, cp.GraphID, cp.ThreadID, data, metadataJSON, cp.Timestamp, cp.Version)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
} // Load retrieves a checkpoint by ID
func (s *CheckpointSaver) Load(ctx context.Context, id string) (*checkpoint.Checkpoint, error) {
	if id == "" {
		return nil, checkpoint.ErrInvalidCheckpointID
	}

	query := fmt.Sprintf(`
		SELECT id, graph_id, thread_id, state, metadata, timestamp, version
		FROM %s 
		WHERE id = $1
	`, s.tableName)

	var cp checkpoint.Checkpoint
	var data []byte
	var metadataJSON []byte

	err := s.pool.QueryRow(ctx, query, id).Scan(
		&cp.ID, &cp.GraphID, &cp.ThreadID, &data, &metadataJSON, &cp.Timestamp, &cp.Version,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, checkpoint.ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Deserialize state
	cp.State = make(map[string]interface{})
	if err := s.serializer.Deserialize(data, &cp.State); err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint state: %w", err)
	}

	// Deserialize metadata
	if err := json.Unmarshal(metadataJSON, &cp.Metadata); err != nil {
		return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	return &cp, nil
}

// List retrieves checkpoints based on filter criteria
func (s *CheckpointSaver) List(ctx context.Context, filter checkpoint.Filter) ([]*checkpoint.Checkpoint, error) {
	query, args := s.buildListQuery(filter)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*checkpoint.Checkpoint
	for rows.Next() {
		var cp checkpoint.Checkpoint
		var data []byte
		var metadataJSON []byte

		err := rows.Scan(
			&cp.ID, &cp.GraphID, &cp.ThreadID, &data, &metadataJSON, &cp.Timestamp, &cp.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint row: %w", err)
		}

		// Deserialize state
		cp.State = make(map[string]interface{})
		if err := s.serializer.Deserialize(data, &cp.State); err != nil {
			return nil, fmt.Errorf("failed to deserialize checkpoint state: %w", err)
		}

		// Deserialize metadata
		if err := json.Unmarshal(metadataJSON, &cp.Metadata); err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}

		checkpoints = append(checkpoints, &cp)
	}

	return checkpoints, nil
} // Delete removes a checkpoint by ID
func (s *CheckpointSaver) Delete(ctx context.Context, id string) error {
	if id == "" {
		return checkpoint.ErrInvalidCheckpointID
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", s.tableName)
	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	if result.RowsAffected() == 0 {
		return checkpoint.ErrCheckpointNotFound
	}

	return nil
}

// CreateTables creates the necessary database tables
func (s *CheckpointSaver) CreateTables(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) PRIMARY KEY,
			graph_id VARCHAR(255) NOT NULL,
			thread_id VARCHAR(255) NOT NULL,
			state BYTEA NOT NULL,
			metadata JSONB,
			timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
			version VARCHAR(50) NOT NULL DEFAULT '1.0'
		);
		
		CREATE INDEX IF NOT EXISTS idx_%s_graph_id ON %s (graph_id);
		CREATE INDEX IF NOT EXISTS idx_%s_thread_id ON %s (thread_id);
		CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s (timestamp);
	`, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName)

	_, err := s.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// buildListQuery constructs the SQL query for listing checkpoints
func (s *CheckpointSaver) buildListQuery(filter checkpoint.Filter) (string, []interface{}) {
	query := fmt.Sprintf("SELECT id, graph_id, thread_id, state, metadata, timestamp, version FROM %s WHERE 1=1", s.tableName)
	args := make([]interface{}, 0)
	argCount := 0

	if filter.GraphID != "" {
		argCount++
		query += fmt.Sprintf(" AND graph_id = $%d", argCount)
		args = append(args, filter.GraphID)
	}

	if filter.ThreadID != "" {
		argCount++
		query += fmt.Sprintf(" AND thread_id = $%d", argCount)
		args = append(args, filter.ThreadID)
	}

	if filter.Since != nil {
		argCount++
		query += fmt.Sprintf(" AND timestamp > $%d", argCount)
		args = append(args, *filter.Since)
	}

	if filter.Before != nil {
		argCount++
		query += fmt.Sprintf(" AND timestamp < $%d", argCount)
		args = append(args, *filter.Before)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	return query, args
}

// Close closes the database connection pool
func (s *CheckpointSaver) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}
