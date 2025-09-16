package main

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"rag-pgvector-openai/internal/config"
	"rag-pgvector-openai/internal/database"
	"rag-pgvector-openai/internal/openai"
)

func TestProductionRAGComponents(t *testing.T) {
	// This is an example test - in production you'd use proper test doubles
	// and not make real API calls in unit tests

	t.Run("ConfigLoading", func(t *testing.T) {
		// Test configuration loading
		// This will fail without proper .env setup
		cfg, err := config.LoadConfig()
		if err != nil {
			t.Logf("Config loading failed (expected without .env): %v", err)
			return
		}

		if cfg.Vector.Dimensions <= 0 {
			t.Error("Vector dimensions should be positive")
		}

		if cfg.App.RequestTimeout <= 0 {
			t.Error("Request timeout should be positive")
		}
	})

	t.Run("DatabaseSchema", func(t *testing.T) {
		// Test database schema creation
		// This would require a test database
		t.Skip("Requires test database setup")
	})

	t.Run("OpenAIClientInitialization", func(t *testing.T) {
		// Test OpenAI client initialization
		client := openai.NewClient(
			"test-key",
			"text-embedding-ada-002",
			"gpt-4",
			1500,
			0.7,
			30*time.Second,
		)

		if client == nil {
			t.Error("OpenAI client should not be nil")
		}
	})
}

func ExampleMain() {
	// Example of how to use the production RAG system
	// This is for documentation purposes

	fmt.Println("Production RAG Example")

	// 1. Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Failed to load config: %v", err)
		return
	}

	// 2. Initialize OpenAI client
	openaiClient := openai.NewClient(
		cfg.OpenAI.APIKey,
		cfg.OpenAI.EmbeddingModel,
		cfg.OpenAI.Model,
		cfg.OpenAI.MaxTokens,
		cfg.OpenAI.Temperature,
		cfg.App.RequestTimeout,
	)

	// 3. Initialize database
	ctx := context.Background()
	vectorDB, err := database.NewVectorDB(
		ctx,
		cfg.GetDatabaseURL(),
		cfg.Vector.Dimensions,
		cfg.Database.MaxConnections,
		cfg.Database.MinConnections,
	)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return
	}
	defer vectorDB.Close()

	// 4. Example document insertion
	sampleDoc := database.Document{
		ID:      "example-doc",
		Content: "This is an example document for testing the RAG system.",
		Metadata: map[string]interface{}{
			"category": "example",
			"source":   "test",
		},
	}

	// Create embedding (in real usage)
	embeddingResp, err := openaiClient.CreateEmbedding(ctx, openai.EmbeddingRequest{
		Text: sampleDoc.Content,
	})
	if err == nil {
		sampleDoc.Embedding = embeddingResp.Embedding

		// Insert document
		if err := vectorDB.InsertDocument(ctx, sampleDoc); err != nil {
			log.Printf("Failed to insert document: %v", err)
		}
	}

	// 5. Example query
	query := "Tell me about the RAG system"

	// Create query embedding
	queryEmbeddingResp, err := openaiClient.CreateEmbedding(ctx, openai.EmbeddingRequest{
		Text: query,
	})
	if err == nil {
		// Search for similar documents
		results, err := vectorDB.SearchSimilarDocuments(
			ctx,
			queryEmbeddingResp.Embedding,
			cfg.Vector.MaxSearchResults,
			cfg.Vector.SimilarityThreshold,
		)

		if err == nil && len(results) > 0 {
			// Extract document contents
			var docs []string
			for _, result := range results {
				docs = append(docs, result.Document.Content)
			}

			// Generate RAG response
			response, err := openaiClient.CreateRAGResponse(ctx, query, docs, 5)
			if err == nil {
				fmt.Printf("Response: %s\n", response.Content)
			}
		}
	}

	fmt.Println("Example completed")
}
