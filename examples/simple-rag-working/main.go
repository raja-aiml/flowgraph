package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/flowgraph/flowgraph/pkg/prebuilt/rag"
)

// SimpleRAGDemo demonstrates how to create a working RAG system with real function implementations
func main() {
	fmt.Println("🚀 Simple Working RAG Example")
	fmt.Println("=====================================")

	// Step 1: Create the enhanced runtime
	ragRuntime := rag.NewEnhancedRAGRuntime()

	// Step 2: Register simple but functional implementations
	registerSimpleFunctions(ragRuntime)

	// Step 3: Create a basic RAG configuration
	config := createSimpleRAGConfig()

	// Step 4: Build the graph
	graph, err := rag.NewSimpleRAG().Build(context.Background(), config)
	if err != nil {
		log.Fatalf("Failed to build graph: %v", err)
	}

	fmt.Printf("✅ Built RAG graph: %s\n", graph.Name)
	fmt.Printf("   - Nodes: %d\n", len(graph.Nodes))
	fmt.Printf("   - Edges: %d\n", len(graph.Edges))

	// Step 5: Show what would happen in execution
	demonstrateExecution()

	fmt.Println("\n🎯 Key Takeaway:")
	fmt.Println("Instead of hardcoded strings like 'query_processor', you now have:")
	fmt.Println("- Real functions that can call APIs")
	fmt.Println("- Custom processors with business logic")
	fmt.Println("- Type-safe interfaces")
	fmt.Println("- Runtime function resolution")
}

func registerSimpleFunctions(ragRuntime *rag.EnhancedRAGRuntime) {
	fmt.Println("\n📝 Registering RAG Functions...")

	// Register a simple query processor
	ragRuntime.RegisterRAGFunction("query_processor", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		query, exists := input["query"]
		if !exists {
			return nil, fmt.Errorf("query is required")
		}

		queryStr := fmt.Sprintf("%v", query)

		// Simple query preprocessing
		processedQuery := strings.TrimSpace(strings.ToLower(queryStr))

		fmt.Printf("   🔍 Query Processor: '%s' -> '%s'\n", queryStr, processedQuery)

		return map[string]interface{}{
			"query":           queryStr,
			"processed_query": processedQuery,
			"query_length":    len(queryStr),
			"processed_by":    "query_processor",
		}, nil
	})

	// Register a simple document retriever
	ragRuntime.RegisterRAGFunction("document_retriever", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		processedQuery, exists := input["processed_query"]
		if !exists {
			processedQuery = input["query"]
		}

		queryStr := fmt.Sprintf("%v", processedQuery)

		// Simulate document retrieval based on query
		var documents []map[string]interface{}
		if strings.Contains(queryStr, "machine learning") || strings.Contains(queryStr, "ai") {
			documents = []map[string]interface{}{
				{"id": "doc1", "title": "Introduction to Machine Learning", "content": "Machine learning is a subset of AI that enables computers to learn without explicit programming..."},
				{"id": "doc2", "title": "AI in Modern Applications", "content": "Artificial intelligence is transforming various industries by providing intelligent automation..."},
			}
		} else if strings.Contains(queryStr, "database") || strings.Contains(queryStr, "vector") {
			documents = []map[string]interface{}{
				{"id": "doc3", "title": "Vector Databases Explained", "content": "Vector databases are specialized databases for storing and querying high-dimensional vectors..."},
				{"id": "doc4", "title": "Database Optimization", "content": "Modern databases use various optimization techniques to improve query performance..."},
			}
		} else {
			documents = []map[string]interface{}{
				{"id": "doc5", "title": "General Information", "content": "This is general information that might be relevant to your query..."},
			}
		}

		fmt.Printf("   📚 Document Retriever: Found %d documents for '%s'\n", len(documents), queryStr)

		return map[string]interface{}{
			"query":          input["query"],
			"documents":      documents,
			"document_count": len(documents),
			"retrieved_by":   "document_retriever",
		}, nil
	})

	// Register a context formatter
	ragRuntime.RegisterRAGFunction("context_formatter", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		documents, exists := input["documents"]
		if !exists {
			return input, nil
		}

		docsList, ok := documents.([]map[string]interface{})
		if !ok {
			return input, nil
		}

		var contextParts []string
		for i, doc := range docsList {
			title := doc["title"]
			content := doc["content"]
			contextParts = append(contextParts, fmt.Sprintf("[%d] %s: %s", i+1, title, content))
		}

		formattedContext := strings.Join(contextParts, "\n\n")

		fmt.Printf("   📝 Context Formatter: Created context with %d documents\n", len(docsList))

		return map[string]interface{}{
			"query":             input["query"],
			"documents":         documents,
			"formatted_context": formattedContext,
			"context_length":    len(formattedContext),
			"formatted_by":      "context_formatter",
		}, nil
	})

	// Register a response generator
	ragRuntime.RegisterRAGFunction("response_generator", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		query := input["query"]
		formattedContext, hasContext := input["formatted_context"]

		var response string
		if hasContext {
			contextStr := fmt.Sprintf("%v", formattedContext)
			response = fmt.Sprintf(`Based on the retrieved information, here's what I can tell you about "%v":

%s

This response was generated by analyzing the relevant documents and providing context-aware information.`, query, contextStr)
		} else {
			response = fmt.Sprintf(`I apologize, but I don't have specific information about "%v" in my current knowledge base. Please try rephrasing your question or asking about a different topic.`, query)
		}

		fmt.Printf("   🤖 Response Generator: Created %d character response\n", len(response))

		return map[string]interface{}{
			"query":           input["query"],
			"response":        response,
			"response_length": len(response),
			"generated_by":    "response_generator",
		}, nil
	})

	fmt.Println("✅ All functions registered!")
}

func createSimpleRAGConfig() rag.SimpleRAGConfig {
	return rag.SimpleRAGConfig{
		ID:         "simple-working-rag",
		Name:       "Simple Working RAG Pipeline",
		EntryPoint: "query",
		Nodes: []rag.NodeConfig{
			{
				ID:   "query",
				Name: "Query Processing Node",
				Type: "processor",
				Func: "query_processor",
			},
			{
				ID:   "retrieval",
				Name: "Document Retrieval Node",
				Type: "retriever",
				Func: "document_retriever",
			},
			{
				ID:   "context",
				Name: "Context Formation Node",
				Type: "processor",
				Func: "context_formatter",
			},
			{
				ID:   "generation",
				Name: "Response Generation Node",
				Type: "generator",
				Func: "response_generator",
			},
		},
		Edges: []rag.EdgeConfig{
			{
				ID:     "query-to-retrieval",
				Source: "query",
				Target: "retrieval",
			},
			{
				ID:     "retrieval-to-context",
				Source: "retrieval",
				Target: "context",
			},
			{
				ID:     "context-to-generation",
				Source: "context",
				Target: "generation",
			},
		},
	}
}

func demonstrateExecution() {
	fmt.Println("\n🎬 Execution Demonstration:")
	fmt.Println("If we executed this RAG pipeline with query 'What is machine learning?', here's what would happen:")

	// Simulate the execution flow
	ctx := context.Background()

	// Step 1: Query Processing
	fmt.Println("\n1️⃣ Query Processing:")
	queryProcessor := createQueryProcessor()
	queryResult, _ := queryProcessor.Process(ctx, map[string]interface{}{
		"query": "What is machine learning?",
	})

	// Step 2: Document Retrieval
	fmt.Println("\n2️⃣ Document Retrieval:")
	docRetriever := createDocRetriever()
	docResult, _ := docRetriever.Process(ctx, queryResult)

	// Step 3: Context Formation
	fmt.Println("\n3️⃣ Context Formation:")
	contextFormatter := createContextFormatter()
	contextResult, _ := contextFormatter.Process(ctx, docResult)

	// Step 4: Response Generation
	fmt.Println("\n4️⃣ Response Generation:")
	responseGen := createResponseGenerator()
	finalResult, _ := responseGen.Process(ctx, contextResult)

	fmt.Printf("\n🎯 Final Response:\n%s\n", finalResult["response"])
}

// Helper functions to create processors for demonstration
func createQueryProcessor() rag.RAGProcessor {
	return rag.NewProcessorFunc("query_processor", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		query := input["query"].(string)
		processed := strings.TrimSpace(strings.ToLower(query))
		fmt.Printf("   Input: '%s' -> Output: '%s'\n", query, processed)
		return map[string]interface{}{
			"query":           query,
			"processed_query": processed,
		}, nil
	})
}

func createDocRetriever() rag.RAGProcessor {
	return rag.NewProcessorFunc("document_retriever", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		documents := []map[string]interface{}{
			{"title": "ML Basics", "content": "Machine learning enables computers to learn patterns from data automatically."},
			{"title": "AI Overview", "content": "Artificial intelligence encompasses machine learning and other computational approaches."},
		}
		fmt.Printf("   Retrieved %d relevant documents\n", len(documents))
		input["documents"] = documents
		return input, nil
	})
}

func createContextFormatter() rag.RAGProcessor {
	return rag.NewProcessorFunc("context_formatter", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		docs := input["documents"].([]map[string]interface{})
		context := ""
		for i, doc := range docs {
			context += fmt.Sprintf("[%d] %s: %s\n", i+1, doc["title"], doc["content"])
		}
		fmt.Printf("   Formatted context: %d characters\n", len(context))
		input["formatted_context"] = context
		return input, nil
	})
}

func createResponseGenerator() rag.RAGProcessor {
	return rag.NewProcessorFunc("response_generator", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		query := input["query"]
		context := input["formatted_context"]
		response := fmt.Sprintf("Based on the context about '%v':\n\n%s\n\nThis provides a comprehensive answer to your question.", query, context)
		fmt.Printf("   Generated response: %d characters\n", len(response))
		input["response"] = response
		return input, nil
	})
}
