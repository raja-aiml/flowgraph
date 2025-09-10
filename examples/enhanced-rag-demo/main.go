package main

import (
	"context"
	"fmt"
	"log"

	"github.com/flowgraph/flowgraph/pkg/flowgraph"
	"github.com/flowgraph/flowgraph/pkg/prebuilt/rag"
)

func main() {
	fmt.Println("Enhanced RAG Demo - With Real Function Implementations")

	// Create enhanced RAG runtime with function registry
	ragRuntime := rag.NewEnhancedRAGRuntime()

	// Register custom RAG functions
	setupCustomRAGFunctions(ragRuntime)

	// Example 1: Default RAG with real implementations
	fmt.Println("\n=== Example 1: Default RAG with Function Implementations ===")
	runDefaultRAGExample(ragRuntime)

	// Example 2: Custom RAG with OpenAI + Pinecone implementations
	fmt.Println("\n=== Example 2: Custom RAG with Real API Implementations ===")
	runCustomRAGExample(ragRuntime)

	// Example 3: Q&A RAG with different providers
	fmt.Println("\n=== Example 3: Q&A RAG with ChromaDB + Claude ===")
	runQARAGExample(ragRuntime)

	fmt.Println("\n=== Demo Complete ===")
}

func setupCustomRAGFunctions(ragRuntime *rag.EnhancedRAGRuntime) {
	// Register OpenAI embedder with API key
	openaiEmbedder := &rag.OpenAIEmbedder{
		APIKey: "sk-your-openai-key-here", // In real usage, use environment variables
		Model:  "text-embedding-ada-002",
	}
	ragRuntime.RegisterRAGProcessor(openaiEmbedder)

	// Register Pinecone search with configuration
	pineconeSearch := &rag.PineconeSearch{
		APIKey:      "your-pinecone-key-here",
		Environment: "us-west1-gcp",
		IndexName:   "rag-index",
	}
	ragRuntime.RegisterRAGProcessor(pineconeSearch)

	// Register GPT-4 generator
	gpt4Generator := &rag.GPT4Generator{
		APIKey:      "sk-your-openai-key-here",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
	}
	ragRuntime.RegisterRAGProcessor(gpt4Generator)

	// Register ChromaDB search function
	ragRuntime.RegisterRAGFunction("chromadb_search", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		embedding, exists := input["embedding"]
		if !exists {
			return nil, fmt.Errorf("embedding required for ChromaDB search")
		}

		// Simulate ChromaDB search
		results := []map[string]interface{}{
			{
				"id":       "chroma_doc1",
				"distance": 0.15, // ChromaDB uses distance instead of score
				"metadata": map[string]interface{}{"source": "knowledge_base.pdf", "page": 1},
				"content":  "ChromaDB retrieved content about the query...",
			},
			{
				"id":       "chroma_doc2",
				"distance": 0.23,
				"metadata": map[string]interface{}{"source": "manual.txt", "section": "FAQ"},
				"content":  "Additional relevant information from ChromaDB...",
			},
		}

		return map[string]interface{}{
			"results":         results,
			"query_embedding": embedding,
			"provider":        "chromadb",
			"collection":      "main_collection",
		}, nil
	})

	// Register Claude 3 generator function
	ragRuntime.RegisterRAGFunction("claude_3_generator", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		query, queryExists := input["query"]
		results, resultsExists := input["results"]

		if !queryExists {
			return nil, fmt.Errorf("query required for Claude 3 generation")
		}

		var contextDocs []string
		if resultsExists {
			if resultsList, ok := results.([]map[string]interface{}); ok {
				for _, result := range resultsList {
					if content, exists := result["content"]; exists {
						if contentStr, ok := content.(string); ok {
							contextDocs = append(contextDocs, contentStr)
						}
					}
				}
			}
		}

		// Simulate Claude 3 API call
		response := fmt.Sprintf(`Based on your query "%v", I've analyzed %d relevant documents. 

Claude 3's analysis suggests that this is a complex question that requires careful consideration of multiple factors. The retrieved context provides valuable insights that help formulate a comprehensive response.

Key findings from the documents:
%s

This response demonstrates Claude 3's reasoning capabilities and contextual understanding.`,
			query, len(contextDocs), fmt.Sprintf("- Found %d relevant pieces of information", len(contextDocs)))

		return map[string]interface{}{
			"response":     response,
			"query":        query,
			"context_docs": len(contextDocs),
			"model":        "claude-3-sonnet-20240229",
			"provider":     "anthropic",
			"reasoning":    "multi-step analysis with context integration",
		}, nil
	})

	// Register enhanced processors with custom logic
	ragRuntime.RegisterRAGFunction("text_embedder", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		text, exists := input["text"]
		if !exists {
			return nil, fmt.Errorf("text required for embedding")
		}

		// Simulate a more sophisticated embedding process
		textStr := fmt.Sprintf("%v", text)
		embedding := make([]float64, 384) // Different embedding size for demonstration

		// Simple hash-based simulation (in real usage, call actual embedding service)
		hash := 0
		for _, char := range textStr {
			hash = hash*31 + int(char)
		}

		for i := range embedding {
			embedding[i] = float64((hash+i)%1000) / 1000.0
		}

		return map[string]interface{}{
			"embedding":  embedding,
			"text":       textStr,
			"model":      "sentence-transformers/all-MiniLM-L6-v2",
			"provider":   "huggingface",
			"dimensions": len(embedding),
		}, nil
	})

	ragRuntime.RegisterRAGFunction("result_reranker", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		results, exists := input["results"]
		if !exists {
			return input, nil // No results to rerank
		}

		resultsList, ok := results.([]map[string]interface{})
		if !ok {
			return input, nil
		}

		// Simulate reranking logic - in real usage, use cross-encoder models
		for i, result := range resultsList {
			// Add rerank score (simulate cross-encoder evaluation)
			originalScore := 0.5
			if score, exists := result["score"]; exists {
				if scoreFloat, ok := score.(float64); ok {
					originalScore = scoreFloat
				}
			}

			// Boost score based on some criteria (e.g., content length, metadata)
			rerankBoost := float64(len(fmt.Sprintf("%v", result["content"]))) / 1000.0
			newScore := originalScore + rerankBoost
			if newScore > 1.0 {
				newScore = 1.0
			}

			resultsList[i]["rerank_score"] = newScore
			resultsList[i]["original_score"] = originalScore
		}

		return map[string]interface{}{
			"results":        resultsList,
			"rerank_method":  "cross_encoder_simulation",
			"reranked_count": len(resultsList),
		}, nil
	})
}

func runDefaultRAGExample(ragRuntime *rag.EnhancedRAGRuntime) {
	config := rag.DefaultSimpleRAGConfig()
	graph, err := buildRAGGraph(config)
	if err != nil {
		log.Printf("Failed to build default graph: %v", err)
		return
	}

	// Test with actual function execution (simulated)
	fmt.Printf("Built graph: %s\n", graph.Name)
	fmt.Println("Functions registered and ready for execution")

	// In a real implementation, you would execute:
	// result, err := ragRuntime.ExecuteRAGGraph(context.Background(), graph, "thread1", map[string]interface{}{"query": "What is machine learning?"})
	fmt.Println("✅ Default RAG configuration ready for execution")
}

func runCustomRAGExample(ragRuntime *rag.EnhancedRAGRuntime) {
	nodes := []rag.NodeConfig{
		{
			ID:   "input",
			Name: "User Input Node",
			Type: "processor",
			Func: "user_input_processor",
		},
		{
			ID:   "embedder",
			Name: "OpenAI Embedder",
			Type: "processor",
			Func: "openai_embedder", // This will use the registered OpenAIEmbedder
		},
		{
			ID:   "search",
			Name: "Pinecone Vector Search",
			Type: "retriever",
			Func: "pinecone_search", // This will use the registered PineconeSearch
		},
		{
			ID:   "generator",
			Name: "GPT-4 Response Generator",
			Type: "generator",
			Func: "gpt4_generator", // This will use the registered GPT4Generator
		},
	}

	edges := []rag.EdgeConfig{
		{ID: "input-to-embedder", Source: "input", Target: "embedder"},
		{ID: "embedder-to-search", Source: "embedder", Target: "search"},
		{ID: "search-to-generator", Source: "search", Target: "generator"},
	}

	config := rag.CustomSimpleRAGConfig(
		"openai-pinecone-rag",
		"OpenAI + Pinecone RAG Pipeline",
		"input",
		nodes,
		edges,
	)

	graph, err := buildRAGGraph(config)
	if err != nil {
		log.Printf("Failed to build custom graph: %v", err)
		return
	}

	fmt.Printf("Built graph: %s\n", graph.Name)
	fmt.Println("✅ Custom RAG with real API implementations ready")

	// Example of what the execution would look like:
	fmt.Println("Example execution flow:")
	fmt.Println("  1. Input: 'What are the benefits of vector databases?'")
	fmt.Println("  2. OpenAI Embedder: Convert query to 1536-dim vector")
	fmt.Println("  3. Pinecone Search: Find similar vectors in 'rag-index'")
	fmt.Println("  4. GPT-4 Generator: Generate response with context")
}

func runQARAGExample(ragRuntime *rag.EnhancedRAGRuntime) {
	config := rag.QARAGConfig("chromadb_search", "claude_3_generator")
	graph, err := buildRAGGraph(config)
	if err != nil {
		log.Printf("Failed to build Q&A graph: %v", err)
		return
	}

	fmt.Printf("Built graph: %s\n", graph.Name)
	fmt.Println("✅ Q&A RAG with ChromaDB + Claude 3 ready")

	// Example execution trace
	fmt.Println("Example execution with reranking:")
	fmt.Println("  1. Question: 'How does RAG improve LLM accuracy?'")
	fmt.Println("  2. Embedding: Convert to 384-dim vector (HuggingFace)")
	fmt.Println("  3. ChromaDB Search: Retrieve documents with distance scores")
	fmt.Println("  4. Reranker: Apply cross-encoder for relevance boost")
	fmt.Println("  5. Claude 3: Generate analytical response with reasoning")
}

func buildRAGGraph(config rag.SimpleRAGConfig) (*flowgraph.Graph, error) {
	builder := rag.NewSimpleRAG()
	return builder.Build(context.Background(), config)
}

// Example of how to create your own custom RAG processor
type CustomDocumentProcessor struct {
	DatabaseURL string
	APIKey      string
}

func (c *CustomDocumentProcessor) Name() string {
	return "custom_document_processor"
}

func (c *CustomDocumentProcessor) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	// Your custom logic here
	query := input["query"]

	// Simulate custom processing
	processedQuery := fmt.Sprintf("PROCESSED: %v", query)

	return map[string]interface{}{
		"processed_query": processedQuery,
		"processor_type":  "custom",
		"database_used":   c.DatabaseURL,
		"timestamp":       "2025-09-09T12:00:00Z",
	}, nil
}
