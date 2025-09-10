package main

import (
	"context"
	"fmt"
	"log"

	"github.com/flowgraph/flowgraph/pkg/flowgraph"
	"github.com/flowgraph/flowgraph/pkg/prebuilt/rag"
)

func main() {
	fmt.Println("RAG Configuration Demo")

	// Example 1: Using default configuration
	fmt.Println("\n=== Example 1: Default RAG Configuration ===")
	defaultConfig := rag.DefaultSimpleRAGConfig()
	printConfig("Default Config", defaultConfig)

	// Build graph from default config
	graph1, err := buildGraph(defaultConfig)
	if err != nil {
		log.Fatalf("Failed to build default graph: %v", err)
	}
	fmt.Printf("Built graph: %s with %d nodes and %d edges\n",
		graph1.Name, len(graph1.Nodes), len(graph1.Edges))

	// Example 2: Custom configuration
	fmt.Println("\n=== Example 2: Custom RAG Configuration ===")
	customNodes := []rag.NodeConfig{
		{
			ID:   "input",
			Name: "User Input Node",
			Type: "input",
			Func: "user_input_processor",
		},
		{
			ID:   "vectorizer",
			Name: "Text Vectorization Node",
			Type: "processor",
			Func: "openai_embedder",
		},
		{
			ID:   "search_db",
			Name: "Vector Database Search",
			Type: "retriever",
			Func: "pinecone_search",
		},
		{
			ID:   "llm_response",
			Name: "LLM Response Generator",
			Type: "generator",
			Func: "gpt4_generator",
		},
	}

	customEdges := []rag.EdgeConfig{
		{ID: "input-to-vectorizer", Source: "input", Target: "vectorizer"},
		{ID: "vectorizer-to-search", Source: "vectorizer", Target: "search_db"},
		{ID: "search-to-llm", Source: "search_db", Target: "llm_response"},
	}

	customConfig := rag.CustomSimpleRAGConfig(
		"custom-rag-pipeline",
		"Custom RAG with OpenAI + Pinecone",
		"input",
		customNodes,
		customEdges,
	)
	printConfig("Custom Config", customConfig)

	// Build graph from custom config
	graph2, err := buildGraph(customConfig)
	if err != nil {
		log.Fatalf("Failed to build custom graph: %v", err)
	}
	fmt.Printf("Built graph: %s with %d nodes and %d edges\n",
		graph2.Name, len(graph2.Nodes), len(graph2.Edges))

	// Example 3: Q&A optimized configuration
	fmt.Println("\n=== Example 3: Q&A RAG Configuration ===")
	qaConfig := rag.QARAGConfig("chromadb_search", "claude_3_generator")
	printConfig("Q&A Config", qaConfig)

	// Build graph from Q&A config
	graph3, err := buildGraph(qaConfig)
	if err != nil {
		log.Fatalf("Failed to build Q&A graph: %v", err)
	}
	fmt.Printf("Built graph: %s with %d nodes and %d edges\n",
		graph3.Name, len(graph3.Nodes), len(graph3.Edges))

	// Example 4: Validation demo
	fmt.Println("\n=== Example 4: Configuration Validation ===")
	invalidConfig := rag.SimpleRAGConfig{
		ID:   "", // Invalid: empty ID
		Name: "Invalid Config Test",
		Nodes: []rag.NodeConfig{
			{ID: "node1", Name: "Node 1", Type: "processor", Func: "func1"},
			{ID: "node1", Name: "Node 2", Type: "processor", Func: "func2"}, // Invalid: duplicate ID
		},
		Edges: []rag.EdgeConfig{
			{ID: "edge1", Source: "node1", Target: "nonexistent"}, // Invalid: target doesn't exist
		},
	}

	if err := rag.ValidateConfig(invalidConfig); err != nil {
		fmt.Printf("Validation correctly caught error: %v\n", err)
	}
}

func printConfig(title string, config rag.SimpleRAGConfig) {
	fmt.Printf("%s:\n", title)
	fmt.Printf("  ID: %s\n", config.ID)
	fmt.Printf("  Name: %s\n", config.Name)
	fmt.Printf("  Entry Point: %s\n", config.EntryPoint)
	fmt.Printf("  Nodes (%d):\n", len(config.Nodes))
	for _, node := range config.Nodes {
		fmt.Printf("    - %s (%s) -> %s [%s]\n", node.ID, node.Name, node.Type, node.Func)
	}
	fmt.Printf("  Edges (%d):\n", len(config.Edges))
	for _, edge := range config.Edges {
		fmt.Printf("    - %s: %s -> %s\n", edge.ID, edge.Source, edge.Target)
	}
}

func buildGraph(config rag.SimpleRAGConfig) (*flowgraph.Graph, error) {
	builder := rag.NewSimpleRAG()
	return builder.Build(context.Background(), config)
}
