package rag

import (
	"context"
	"fmt"

	"github.com/flowgraph/flowgraph/pkg/flowgraph"
	"github.com/flowgraph/flowgraph/pkg/prebuilt"
)

// SimpleRAGConfig defines inputs to the Simple RAG builder.
// These are structural switches; business logic (LLM/vector DB) should be
// implemented by your node processors/tooling at runtime.
type NodeConfig struct {
	ID   string
	Name string
	Type string
	Func string
}

type EdgeConfig struct {
	ID     string
	Source string
	Target string
}

type SimpleRAGConfig struct {
	ID         string // Graph ID
	Name       string // Graph name
	EntryPoint string // Entry node ID
	Nodes      []NodeConfig
	Edges      []EdgeConfig
}

// NewSimpleRAG registers a prebuilt under the given name into a registry.
func NewSimpleRAG() prebuilt.Builder {
	return prebuilt.NewBuildFunc("simple_rag", func(ctx context.Context, cfg any) (*flowgraph.Graph, error) {
		c, ok := cfg.(SimpleRAGConfig)
		if !ok {
			return nil, fmt.Errorf("invalid config type for simple_rag, expected SimpleRAGConfig")
		}

		// Validate configuration before proceeding
		if err := ValidateConfig(c); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		id := c.ID
		if id == "" {
			id = "prebuilt-simple-rag"
		}
		name := c.Name
		if name == "" {
			name = "Prebuilt: Simple RAG"
		}
		entry := c.EntryPoint
		if entry == "" && len(c.Nodes) > 0 {
			entry = c.Nodes[0].ID
		}

		g := &flowgraph.Graph{
			ID:         id,
			Name:       name,
			EntryPoint: entry,
			Nodes:      map[string]*flowgraph.Node{},
			Edges:      []*flowgraph.Edge{},
		}

		for _, n := range c.Nodes {
			if n.ID == "" {
				return nil, fmt.Errorf("node ID cannot be empty")
			}
			if n.Type == "" {
				return nil, fmt.Errorf("node type cannot be empty for node %s", n.ID)
			}

			g.Nodes[n.ID] = &flowgraph.Node{
				ID:     n.ID,
				Name:   n.Name,
				Type:   flowgraph.NodeType(n.Type),
				Config: map[string]any{"func": n.Func},
			}
		}

		for _, e := range c.Edges {
			if e.ID == "" {
				return nil, fmt.Errorf("edge ID cannot be empty")
			}
			if e.Source == "" || e.Target == "" {
				return nil, fmt.Errorf("edge source and target cannot be empty for edge %s", e.ID)
			}

			g.Edges = append(g.Edges, &flowgraph.Edge{
				ID:     e.ID,
				Source: e.Source,
				Target: e.Target,
			})
		}

		if err := g.Validate(); err != nil {
			return nil, fmt.Errorf("graph validation failed: %w", err)
		}
		return g, nil
	})
}

// DefaultSimpleRAGConfig creates a default configuration for a simple RAG pipeline
// with common nodes: retrieval, context, generation, and response formatting.
func DefaultSimpleRAGConfig() SimpleRAGConfig {
	return SimpleRAGConfig{
		ID:         "simple-rag-default",
		Name:       "Default Simple RAG Pipeline",
		EntryPoint: "query",
		Nodes: []NodeConfig{
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
		Edges: []EdgeConfig{
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

// CustomSimpleRAGConfig creates a customizable RAG configuration with the provided parameters.
func CustomSimpleRAGConfig(id, name, entryPoint string, nodes []NodeConfig, edges []EdgeConfig) SimpleRAGConfig {
	config := SimpleRAGConfig{
		ID:         id,
		Name:       name,
		EntryPoint: entryPoint,
		Nodes:      nodes,
		Edges:      edges,
	}

	// Apply defaults if not provided
	if config.ID == "" {
		config.ID = "custom-simple-rag"
	}
	if config.Name == "" {
		config.Name = "Custom Simple RAG Pipeline"
	}
	if config.EntryPoint == "" && len(nodes) > 0 {
		config.EntryPoint = nodes[0].ID
	}

	return config
}

// QARAGConfig creates a configuration optimized for Question-Answering RAG.
func QARAGConfig(vectorDBFunc, llmFunc string) SimpleRAGConfig {
	return SimpleRAGConfig{
		ID:         "qa-rag",
		Name:       "Question-Answer RAG Pipeline",
		EntryPoint: "question",
		Nodes: []NodeConfig{
			{
				ID:   "question",
				Name: "Question Input Node",
				Type: "input",
				Func: "question_input",
			},
			{
				ID:   "embedding",
				Name: "Question Embedding Node",
				Type: "processor",
				Func: "text_embedder",
			},
			{
				ID:   "search",
				Name: "Vector Search Node",
				Type: "retriever",
				Func: vectorDBFunc,
			},
			{
				ID:   "rerank",
				Name: "Result Reranking Node",
				Type: "processor",
				Func: "result_reranker",
			},
			{
				ID:   "answer",
				Name: "Answer Generation Node",
				Type: "generator",
				Func: llmFunc,
			},
		},
		Edges: []EdgeConfig{
			{
				ID:     "question-to-embedding",
				Source: "question",
				Target: "embedding",
			},
			{
				ID:     "embedding-to-search",
				Source: "embedding",
				Target: "search",
			},
			{
				ID:     "search-to-rerank",
				Source: "search",
				Target: "rerank",
			},
			{
				ID:     "rerank-to-answer",
				Source: "rerank",
				Target: "answer",
			},
		},
	}
}

// ValidateConfig validates the SimpleRAGConfig for common issues.
func ValidateConfig(cfg SimpleRAGConfig) error {
	if cfg.ID == "" {
		return fmt.Errorf("config ID cannot be empty")
	}

	if len(cfg.Nodes) == 0 {
		return fmt.Errorf("config must have at least one node")
	}

	// Check if entry point exists in nodes
	if cfg.EntryPoint != "" {
		found := false
		for _, node := range cfg.Nodes {
			if node.ID == cfg.EntryPoint {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("entry point '%s' not found in nodes", cfg.EntryPoint)
		}
	}

	// Validate node IDs are unique
	nodeIDs := make(map[string]bool)
	for _, node := range cfg.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node ID: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// Validate edges reference existing nodes
	for _, edge := range cfg.Edges {
		if !nodeIDs[edge.Source] {
			return fmt.Errorf("edge source '%s' references non-existent node", edge.Source)
		}
		if !nodeIDs[edge.Target] {
			return fmt.Errorf("edge target '%s' references non-existent node", edge.Target)
		}
	}

	return nil
}

// init registers simple_rag into the default registry.
func init() {
	prebuilt.DefaultRegistry.MustRegister(NewSimpleRAG())
}
