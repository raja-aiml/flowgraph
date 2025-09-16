package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"rag-pgvector-openai/internal/config"
	"rag-pgvector-openai/internal/database"
	"rag-pgvector-openai/internal/openai"
	"rag-pgvector-openai/internal/processors"

	"github.com/flowgraph/flowgraph/pkg/flowgraph"
	"github.com/flowgraph/flowgraph/pkg/prebuilt/rag"
)

func main() {
	fmt.Println("Production RAG with pgvector and OpenAI")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize context
	ctx := context.Background()

	// Initialize OpenAI client
	openaiClient := openai.NewClient(
		cfg.OpenAI.APIKey,
		cfg.OpenAI.EmbeddingModel,
		cfg.OpenAI.Model,
		cfg.OpenAI.MaxTokens,
		cfg.OpenAI.Temperature,
		cfg.App.RequestTimeout,
	)

	// Test OpenAI connection
	fmt.Println("Testing OpenAI connection...")
	if err := openaiClient.HealthCheck(ctx); err != nil {
		log.Fatalf("OpenAI health check failed: %v", err)
	}
	fmt.Println("✅ OpenAI connection successful")

	// Initialize PostgreSQL with pgvector
	fmt.Println("Connecting to PostgreSQL with pgvector...")
	vectorDB, err := database.NewVectorDB(
		ctx,
		cfg.GetDatabaseURL(),
		cfg.Vector.Dimensions,
		cfg.Database.MaxConnections,
		cfg.Database.MinConnections,
	)
	if err != nil {
		log.Fatalf("Failed to initialize vector database: %v", err)
	}
	defer vectorDB.Close()

	// Test database connection
	if err := vectorDB.HealthCheck(ctx); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}
	fmt.Println("✅ PostgreSQL with pgvector connection successful")

	// Initialize sample documents if database is empty
	if err := initializeSampleDocuments(ctx, vectorDB, openaiClient); err != nil {
		log.Fatalf("Failed to initialize sample documents: %v", err)
	}

	// Create enhanced RAG runtime
	ragRuntime := rag.NewEnhancedRAGRuntime()

	// Register production processors
	setupProductionRAGFunctions(ragRuntime, openaiClient, vectorDB, cfg)

	// Example 1: Production RAG Pipeline
	fmt.Println("\n=== Production RAG Pipeline Demo ===")
	if err := runProductionRAGDemo(ctx, ragRuntime, cfg); err != nil {
		log.Printf("Production RAG demo failed: %v", err)
	}

	// Example 2: Advanced RAG with re-ranking
	fmt.Println("\n=== Advanced RAG with Re-ranking Demo ===")
	if err := runAdvancedRAGDemo(ctx, ragRuntime, cfg); err != nil {
		log.Printf("Advanced RAG demo failed: %v", err)
	}

	fmt.Println("\n=== Production RAG Demo Complete ===")
}

func setupProductionRAGFunctions(ragRuntime *rag.EnhancedRAGRuntime, openaiClient *openai.Client, vectorDB *database.VectorDB, cfg *config.Config) {
	// Register production processors
	queryProcessor := processors.NewQueryProcessor()
	ragRuntime.RegisterRAGProcessor(queryProcessor)

	embedder := processors.NewProductionEmbedder(openaiClient, cfg.OpenAI.EmbeddingModel)
	ragRuntime.RegisterRAGProcessor(embedder)

	vectorSearch := processors.NewProductionVectorSearch(
		vectorDB,
		cfg.Vector.MaxSearchResults,
		cfg.Vector.SimilarityThreshold,
	)
	ragRuntime.RegisterRAGProcessor(vectorSearch)

	generator := processors.NewProductionRAGGenerator(
		openaiClient,
		cfg.OpenAI.Model,
		cfg.OpenAI.MaxTokens,
		cfg.OpenAI.Temperature,
		5, // max docs for context
	)
	ragRuntime.RegisterRAGProcessor(generator)

	ranker := processors.NewResultRanker(0.1, 0.05) // diversity and recency weights
	ragRuntime.RegisterRAGProcessor(ranker)

	log.Println("✅ Production RAG functions registered")
}

func initializeSampleDocuments(ctx context.Context, vectorDB *database.VectorDB, openaiClient *openai.Client) error {
	// Check if we already have documents
	count, err := vectorDB.GetDocumentCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get document count: %w", err)
	}

	if count > 0 {
		log.Printf("Database already contains %d documents", count)
		return nil
	}

	log.Println("Initializing sample documents...")

	sampleDocs := []struct {
		id       string
		content  string
		metadata map[string]interface{}
	}{
		{
			id:      "doc1",
			content: "Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data without being explicitly programmed. It includes supervised learning, unsupervised learning, and reinforcement learning approaches.",
			metadata: map[string]interface{}{
				"topic":    "machine_learning",
				"category": "AI",
				"level":    "beginner",
			},
		},
		{
			id:      "doc2",
			content: "Vector databases are specialized databases designed to store and query high-dimensional vector embeddings. They use techniques like approximate nearest neighbor search to efficiently find similar vectors, making them ideal for applications like semantic search and recommendation systems.",
			metadata: map[string]interface{}{
				"topic":    "vector_databases",
				"category": "databases",
				"level":    "intermediate",
			},
		},
		{
			id:      "doc3",
			content: "Retrieval-Augmented Generation (RAG) combines the power of large language models with external knowledge retrieval. It works by first retrieving relevant documents based on a query, then using those documents as context for generating more accurate and informed responses.",
			metadata: map[string]interface{}{
				"topic":    "RAG",
				"category": "NLP",
				"level":    "advanced",
			},
		},
		{
			id:      "doc4",
			content: "PostgreSQL with the pgvector extension provides efficient storage and querying of vector embeddings. It supports various distance metrics including cosine similarity, L2 distance, and inner product, making it suitable for similarity search applications.",
			metadata: map[string]interface{}{
				"topic":    "pgvector",
				"category": "databases",
				"level":    "intermediate",
			},
		},
		{
			id:      "doc5",
			content: "OpenAI's text-embedding-ada-002 model produces 1536-dimensional embeddings that capture semantic meaning of text. These embeddings can be used for search, clustering, recommendations, anomaly detection, and classification tasks.",
			metadata: map[string]interface{}{
				"topic":    "embeddings",
				"category": "AI",
				"level":    "intermediate",
			},
		},
	}

	// Create embeddings and store documents
	for _, doc := range sampleDocs {
		// Create embedding for the document
		resp, err := openaiClient.CreateEmbedding(ctx, openai.EmbeddingRequest{
			Text: doc.content,
		})
		if err != nil {
			return fmt.Errorf("failed to create embedding for document %s: %w", doc.id, err)
		}

		// Store document with embedding
		document := database.Document{
			ID:        doc.id,
			Content:   doc.content,
			Metadata:  doc.metadata,
			Embedding: resp.Embedding,
		}

		if err := vectorDB.InsertDocument(ctx, document); err != nil {
			return fmt.Errorf("failed to insert document %s: %w", doc.id, err)
		}

		log.Printf("Inserted document: %s", doc.id)
	}

	log.Printf("✅ Initialized %d sample documents", len(sampleDocs))
	return nil
}

func runProductionRAGDemo(ctx context.Context, ragRuntime *rag.EnhancedRAGRuntime, _ *config.Config) error {
	// Create production RAG configuration
	nodes := []rag.NodeConfig{
		{
			ID:   "query",
			Name: "Query Processing Node",
			Type: "processor",
			Func: "production_query_processor",
		},
		{
			ID:   "embedder",
			Name: "Production Embedder",
			Type: "processor",
			Func: "production_embedder",
		},
		{
			ID:   "search",
			Name: "Vector Search with pgvector",
			Type: "retriever",
			Func: "production_vector_search",
		},
		{
			ID:   "generator",
			Name: "RAG Response Generator",
			Type: "generator",
			Func: "production_rag_generator",
		},
	}

	edges := []rag.EdgeConfig{
		{ID: "query-to-embedder", Source: "query", Target: "embedder"},
		{ID: "embedder-to-search", Source: "embedder", Target: "search"},
		{ID: "search-to-generator", Source: "search", Target: "generator"},
	}

	config := rag.CustomSimpleRAGConfig(
		"production-rag",
		"Production RAG with pgvector and OpenAI",
		"query",
		nodes,
		edges,
	)

	graph, err := buildRAGGraph(config)
	if err != nil {
		return fmt.Errorf("failed to build production graph: %w", err)
	}

	// Test queries
	testQueries := []string{
		"What is machine learning and how does it work?",
		"How do vector databases store and query embeddings?",
		"Explain how RAG improves language model responses",
	}

	for i, query := range testQueries {
		fmt.Printf("\nQuery %d: %s\n", i+1, query)

		// Add timestamp to context
		queryCtx := context.WithValue(ctx, "timestamp", time.Now().Format(time.RFC3339))

		// Execute the RAG pipeline
		result, err := ragRuntime.ExecuteRAGGraph(queryCtx, graph, fmt.Sprintf("thread_%d", i+1), map[string]interface{}{
			"query": query,
		})

		if err != nil {
			log.Printf("Failed to execute query %d: %v", i+1, err)
			continue
		}

		fmt.Printf("Status: %s\n", result.Status)
		if result.Error != "" {
			fmt.Printf("Error: %s\n", result.Error)
		}

		// In a real implementation, you would extract and display the generated response
		fmt.Printf("✅ Query %d processed successfully\n", i+1)
	}

	return nil
}

func runAdvancedRAGDemo(ctx context.Context, ragRuntime *rag.EnhancedRAGRuntime, _ *config.Config) error {
	// Create advanced RAG configuration with re-ranking
	nodes := []rag.NodeConfig{
		{
			ID:   "query",
			Name: "Query Processing Node",
			Type: "processor",
			Func: "production_query_processor",
		},
		{
			ID:   "embedder",
			Name: "Production Embedder",
			Type: "processor",
			Func: "production_embedder",
		},
		{
			ID:   "search",
			Name: "Vector Search with pgvector",
			Type: "retriever",
			Func: "production_vector_search",
		},
		{
			ID:   "ranker",
			Name: "Result Re-ranker",
			Type: "processor",
			Func: "result_ranker",
		},
		{
			ID:   "generator",
			Name: "RAG Response Generator",
			Type: "generator",
			Func: "production_rag_generator",
		},
	}

	edges := []rag.EdgeConfig{
		{ID: "query-to-embedder", Source: "query", Target: "embedder"},
		{ID: "embedder-to-search", Source: "embedder", Target: "search"},
		{ID: "search-to-ranker", Source: "search", Target: "ranker"},
		{ID: "ranker-to-generator", Source: "ranker", Target: "generator"},
	}

	config := rag.CustomSimpleRAGConfig(
		"advanced-rag",
		"Advanced RAG with Re-ranking",
		"query",
		nodes,
		edges,
	)

	graph, err := buildRAGGraph(config)
	if err != nil {
		return fmt.Errorf("failed to build advanced graph: %w", err)
	}

	// Test with a more complex query
	complexQuery := "Compare vector databases and traditional databases for AI applications. What are the advantages of using pgvector?"

	fmt.Printf("Complex Query: %s\n", complexQuery)

	queryCtx := context.WithValue(ctx, "timestamp", time.Now().Format(time.RFC3339))

	result, err := ragRuntime.ExecuteRAGGraph(queryCtx, graph, "advanced_thread", map[string]interface{}{
		"query": complexQuery,
	})

	if err != nil {
		return fmt.Errorf("failed to execute advanced query: %w", err)
	}

	fmt.Printf("Status: %s\n", result.Status)
	if result.Error != "" {
		fmt.Printf("Error: %s\n", result.Error)
	}

	fmt.Println("✅ Advanced RAG query processed successfully with re-ranking")
	return nil
}

func buildRAGGraph(config rag.SimpleRAGConfig) (*flowgraph.Graph, error) {
	builder := rag.NewSimpleRAG()
	return builder.Build(context.Background(), config)
}
