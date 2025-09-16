package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/lib/pq"
	"github.com/pgvector/pgvector-go"
)

// VectorDB handles PostgreSQL operations with pgvector
type VectorDB struct {
	pool       *pgxpool.Pool
	dimensions int
}

// Document represents a document with vector embedding
type Document struct {
	ID        string                 `json:"id"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata"`
	Embedding []float32              `json:"embedding"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Document   Document `json:"document"`
	Similarity float64  `json:"similarity"`
}

// NewVectorDB creates a new VectorDB instance
func NewVectorDB(ctx context.Context, databaseURL string, dimensions int, maxConns, minConns int) (*VectorDB, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 30
	config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	vdb := &VectorDB{
		pool:       pool,
		dimensions: dimensions,
	}

	// Initialize database schema
	if err := vdb.InitializeSchema(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return vdb, nil
}

// InitializeSchema creates necessary tables and extensions
func (vdb *VectorDB) InitializeSchema(ctx context.Context) error {
	queries := []string{
		// Enable pgvector extension
		"CREATE EXTENSION IF NOT EXISTS vector;",

		// Create documents table
		fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS documents (
				id TEXT PRIMARY KEY,
				content TEXT NOT NULL,
				metadata JSONB DEFAULT '{}',
				embedding vector(%d),
				created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
				updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
			);
		`, vdb.dimensions),

		// Create index for vector similarity search
		"CREATE INDEX IF NOT EXISTS documents_embedding_idx ON documents USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);",

		// Create index for metadata searches
		"CREATE INDEX IF NOT EXISTS documents_metadata_idx ON documents USING GIN (metadata);",

		// Create updated_at trigger
		`
			CREATE OR REPLACE FUNCTION update_updated_at_column()
			RETURNS TRIGGER AS $$
			BEGIN
				NEW.updated_at = NOW();
				RETURN NEW;
			END;
			$$ language 'plpgsql';
		`,

		`
			DO $$
			BEGIN
				IF NOT EXISTS (
					SELECT 1 FROM pg_trigger 
					WHERE tgname = 'update_documents_updated_at'
				) THEN
					CREATE TRIGGER update_documents_updated_at
					BEFORE UPDATE ON documents
					FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
				END IF;
			END $$;
		`,
	}

	for _, query := range queries {
		if _, err := vdb.pool.Exec(ctx, query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

// InsertDocument inserts a new document with its embedding
func (vdb *VectorDB) InsertDocument(ctx context.Context, doc Document) error {
	query := `
		INSERT INTO documents (id, content, metadata, embedding, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata,
			embedding = EXCLUDED.embedding,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = now
	}
	doc.UpdatedAt = now

	embedding := pgvector.NewVector(doc.Embedding)

	_, err := vdb.pool.Exec(ctx, query,
		doc.ID,
		doc.Content,
		doc.Metadata,
		embedding,
		doc.CreatedAt,
		doc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	return nil
}

// SearchSimilarDocuments performs vector similarity search
func (vdb *VectorDB) SearchSimilarDocuments(ctx context.Context, embedding []float32, limit int, threshold float64) ([]SearchResult, error) {
	query := `
		SELECT 
			id, content, metadata, embedding, created_at, updated_at,
			1 - (embedding <=> $1) as similarity
		FROM documents
		WHERE 1 - (embedding <=> $1) > $2
		ORDER BY embedding <=> $1
		LIMIT $3
	`

	queryVector := pgvector.NewVector(embedding)

	rows, err := vdb.pool.Query(ctx, query, queryVector, threshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute similarity search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var doc Document
		var similarity float64
		var embedding pgvector.Vector

		err := rows.Scan(
			&doc.ID,
			&doc.Content,
			&doc.Metadata,
			&embedding,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		doc.Embedding = embedding.Slice()

		results = append(results, SearchResult{
			Document:   doc,
			Similarity: similarity,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return results, nil
}

// GetDocumentByID retrieves a document by its ID
func (vdb *VectorDB) GetDocumentByID(ctx context.Context, id string) (*Document, error) {
	query := `
		SELECT id, content, metadata, embedding, created_at, updated_at
		FROM documents
		WHERE id = $1
	`

	var doc Document
	var embedding pgvector.Vector

	err := vdb.pool.QueryRow(ctx, query, id).Scan(
		&doc.ID,
		&doc.Content,
		&doc.Metadata,
		&embedding,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	doc.Embedding = embedding.Slice()
	return &doc, nil
}

// DeleteDocument deletes a document by ID
func (vdb *VectorDB) DeleteDocument(ctx context.Context, id string) error {
	query := "DELETE FROM documents WHERE id = $1"

	result, err := vdb.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("document with id %s not found", id)
	}

	return nil
}

// GetDocumentCount returns the total number of documents
func (vdb *VectorDB) GetDocumentCount(ctx context.Context) (int64, error) {
	var count int64
	err := vdb.pool.QueryRow(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get document count: %w", err)
	}
	return count, nil
}

// SearchWithFilters performs similarity search with metadata filters
func (vdb *VectorDB) SearchWithFilters(ctx context.Context, embedding []float32, limit int, threshold float64, filters map[string]interface{}) ([]SearchResult, error) {
	baseQuery := `
		SELECT 
			id, content, metadata, embedding, created_at, updated_at,
			1 - (embedding <=> $1) as similarity
		FROM documents
		WHERE 1 - (embedding <=> $1) > $2
	`

	args := []interface{}{pgvector.NewVector(embedding), threshold}
	argIndex := 3

	// Add metadata filters
	for key, value := range filters {
		baseQuery += fmt.Sprintf(" AND metadata->>'%s' = $%d", key, argIndex)
		args = append(args, value)
		argIndex++
	}

	baseQuery += fmt.Sprintf(" ORDER BY embedding <=> $1 LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := vdb.pool.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute filtered search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var doc Document
		var similarity float64
		var embedding pgvector.Vector

		err := rows.Scan(
			&doc.ID,
			&doc.Content,
			&doc.Metadata,
			&embedding,
			&doc.CreatedAt,
			&doc.UpdatedAt,
			&similarity,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		doc.Embedding = embedding.Slice()

		results = append(results, SearchResult{
			Document:   doc,
			Similarity: similarity,
		})
	}

	return results, nil
}

// Close closes the database connection pool
func (vdb *VectorDB) Close() {
	vdb.pool.Close()
}

// HealthCheck performs a health check on the database
func (vdb *VectorDB) HealthCheck(ctx context.Context) error {
	return vdb.pool.Ping(ctx)
}
