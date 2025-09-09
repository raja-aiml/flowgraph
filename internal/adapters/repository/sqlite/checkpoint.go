package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/checkpoint"
	"github.com/flowgraph/flowgraph/pkg/serialization"
	_ "modernc.org/sqlite"
)

// CheckpointSaver implements checkpoint.Saver interface for SQLite
type CheckpointSaver struct {
    db         *sql.DB
    serializer *serialization.Serializer
    tableName  string
}

// NewCheckpointSaver creates a new SQLite checkpoint saver
func NewCheckpointSaver(db *sql.DB, serializer *serialization.Serializer) *CheckpointSaver {
    return &CheckpointSaver{
        db:         db,
        serializer: serializer,
        tableName:  "checkpoints",
    }
}

// WithTableName allows overriding the default table name with validation.
// Only alphanumeric and underscore are permitted to prevent SQL injection via identifiers.
func (s *CheckpointSaver) WithTableName(name string) *CheckpointSaver {
    if isSafeIdent(name) {
        s.tableName = name
    }
    return s
}

func isSafeIdent(s string) bool {
    if s == "" { return false }
    for i := 0; i < len(s); i++ {
        c := s[i]
        if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
            continue
        }
        return false
    }
    return true
}

// Save stores a checkpoint in SQLite
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
		INSERT OR REPLACE INTO %s (id, graph_id, thread_id, state, metadata, timestamp, version)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, s.tableName)

	_, err = s.db.ExecContext(ctx, query,
		cp.ID, cp.GraphID, cp.ThreadID, data, string(metadataJSON), cp.Timestamp.Unix(), cp.Version)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return nil
}

// Load retrieves a checkpoint by ID
func (s *CheckpointSaver) Load(ctx context.Context, id string) (*checkpoint.Checkpoint, error) {
	if id == "" {
		return nil, checkpoint.ErrInvalidCheckpointID
	}

	query := fmt.Sprintf(`
		SELECT id, graph_id, thread_id, state, metadata, timestamp, version
		FROM %s 
		WHERE id = ?
	`, s.tableName)

	var cp checkpoint.Checkpoint
	var data []byte
	var metadataJSON string
	var timestamp int64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&cp.ID, &cp.GraphID, &cp.ThreadID, &data, &metadataJSON, &timestamp, &cp.Version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, checkpoint.ErrCheckpointNotFound
		}
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Convert timestamp
	cp.Timestamp = time.Unix(timestamp, 0)

	// Deserialize state
	cp.State = make(map[string]interface{})
	if err := s.serializer.Deserialize(data, &cp.State); err != nil {
		return nil, fmt.Errorf("failed to deserialize checkpoint state: %w", err)
	}

	// Deserialize metadata
	if err := json.Unmarshal([]byte(metadataJSON), &cp.Metadata); err != nil {
		return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	return &cp, nil
}

// List retrieves checkpoints based on filter criteria
func (s *CheckpointSaver) List(ctx context.Context, filter checkpoint.Filter) ([]*checkpoint.Checkpoint, error) {
	query, args := s.buildListQuery(filter)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	var checkpoints []*checkpoint.Checkpoint
	for rows.Next() {
		var cp checkpoint.Checkpoint
		var data []byte
		var metadataJSON string
		var timestamp int64

		err := rows.Scan(
			&cp.ID, &cp.GraphID, &cp.ThreadID, &data, &metadataJSON, &timestamp, &cp.Version,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint row: %w", err)
		}

		// Convert timestamp
		cp.Timestamp = time.Unix(timestamp, 0)

		// Deserialize state
		cp.State = make(map[string]interface{})
		if err := s.serializer.Deserialize(data, &cp.State); err != nil {
			return nil, fmt.Errorf("failed to deserialize checkpoint state: %w", err)
		}

		// Deserialize metadata
		if err := json.Unmarshal([]byte(metadataJSON), &cp.Metadata); err != nil {
			return nil, fmt.Errorf("failed to deserialize metadata: %w", err)
		}

		checkpoints = append(checkpoints, &cp)
	}

	return checkpoints, nil
}

// Delete removes a checkpoint by ID
func (s *CheckpointSaver) Delete(ctx context.Context, id string) error {
	if id == "" {
		return checkpoint.ErrInvalidCheckpointID
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", s.tableName)
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete checkpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return checkpoint.ErrCheckpointNotFound
	}

	return nil
}

// CreateTables creates the necessary database tables
func (s *CheckpointSaver) CreateTables(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			graph_id TEXT NOT NULL,
			thread_id TEXT NOT NULL,
			state BLOB NOT NULL,
			metadata TEXT,
			timestamp INTEGER NOT NULL,
			version TEXT NOT NULL DEFAULT '1.0'
		);
		
		CREATE INDEX IF NOT EXISTS idx_%s_graph_id ON %s (graph_id);
		CREATE INDEX IF NOT EXISTS idx_%s_thread_id ON %s (thread_id);
		CREATE INDEX IF NOT EXISTS idx_%s_timestamp ON %s (timestamp);
	`, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName, s.tableName)

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

// buildListQuery constructs the SQL query for listing checkpoints
func (s *CheckpointSaver) buildListQuery(filter checkpoint.Filter) (string, []interface{}) {
	query := fmt.Sprintf("SELECT id, graph_id, thread_id, state, metadata, timestamp, version FROM %s WHERE 1=1", s.tableName)
	args := make([]interface{}, 0)

	if filter.GraphID != "" {
		query += " AND graph_id = ?"
		args = append(args, filter.GraphID)
	}

	if filter.ThreadID != "" {
		query += " AND thread_id = ?"
		args = append(args, filter.ThreadID)
	}

	if filter.Since != nil {
		query += " AND timestamp > ?"
		args = append(args, filter.Since.Unix())
	}

	if filter.Before != nil {
		query += " AND timestamp < ?"
		args = append(args, filter.Before.Unix())
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	return query, args
}

// Close closes the database connection
func (s *CheckpointSaver) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}
