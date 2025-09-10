package rag

import (
	"context"
	"fmt"

	"github.com/flowgraph/flowgraph/internal/app/dto"
	"github.com/flowgraph/flowgraph/internal/core/graph"
	"github.com/flowgraph/flowgraph/pkg/flowgraph"
)

// RAGProcessor defines the interface for RAG-specific node processing functions
type RAGProcessor interface {
	Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
	Name() string
}

// ProcessorFunc is a function type that implements RAGProcessor
type ProcessorFunc struct {
	name string
	fn   func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)
}

func (p ProcessorFunc) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	return p.fn(ctx, input)
}

func (p ProcessorFunc) Name() string {
	return p.name
}

// NewProcessorFunc creates a RAGProcessor from a function
func NewProcessorFunc(name string, fn func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)) RAGProcessor {
	return ProcessorFunc{name: name, fn: fn}
}

// RAGFunctionRegistry holds the actual implementations of RAG functions
type RAGFunctionRegistry struct {
	processors map[string]RAGProcessor
}

// NewRAGFunctionRegistry creates a new function registry
func NewRAGFunctionRegistry() *RAGFunctionRegistry {
	return &RAGFunctionRegistry{
		processors: make(map[string]RAGProcessor),
	}
}

// Register adds a processor to the registry
func (r *RAGFunctionRegistry) Register(processor RAGProcessor) {
	r.processors[processor.Name()] = processor
}

// RegisterFunc adds a function as a processor to the registry
func (r *RAGFunctionRegistry) RegisterFunc(name string, fn func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)) {
	r.processors[name] = NewProcessorFunc(name, fn)
}

// Get retrieves a processor by name
func (r *RAGFunctionRegistry) Get(name string) (RAGProcessor, bool) {
	processor, exists := r.processors[name]
	return processor, exists
}

// RAGNodeProcessor implements the NodeTypeProcessor interface for RAG nodes
type RAGNodeProcessor struct {
	functionRegistry *RAGFunctionRegistry
}

// NewRAGNodeProcessor creates a new RAG node processor
func NewRAGNodeProcessor(functionRegistry *RAGFunctionRegistry) *RAGNodeProcessor {
	return &RAGNodeProcessor{
		functionRegistry: functionRegistry,
	}
}

// Process executes a RAG node by looking up the function in the registry
func (p *RAGNodeProcessor) Process(ctx context.Context, node *graph.Node, input map[string]interface{}) (map[string]interface{}, error) {
	// Get function name from node config
	funcName, exists := node.Config["func"]
	if !exists {
		return nil, fmt.Errorf("no function specified in node config for node %s", node.ID)
	}

	funcNameStr, ok := funcName.(string)
	if !ok {
		return nil, fmt.Errorf("function name must be a string for node %s", node.ID)
	}

	// Look up the actual processor function
	processor, exists := p.functionRegistry.Get(funcNameStr)
	if !exists {
		return nil, fmt.Errorf("function '%s' not found in registry for node %s", funcNameStr, node.ID)
	}

	// Execute the function
	output, err := processor.Process(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("error executing function '%s' in node %s: %w", funcNameStr, node.ID, err)
	}

	// Add execution metadata
	output["_node_id"] = node.ID
	output["_function_name"] = funcNameStr
	output["_node_type"] = string(node.Type)

	return output, nil
}

// EnhancedRAGRuntime combines the graph runtime with RAG function registry
type EnhancedRAGRuntime struct {
	runtime          *flowgraph.Runtime
	functionRegistry *RAGFunctionRegistry
	ragNodeProcessor *RAGNodeProcessor
}

// NewEnhancedRAGRuntime creates a runtime with RAG function support
func NewEnhancedRAGRuntime() *EnhancedRAGRuntime {
	runtime := flowgraph.NewRuntime()
	functionRegistry := NewRAGFunctionRegistry()
	ragNodeProcessor := NewRAGNodeProcessor(functionRegistry)

	return &EnhancedRAGRuntime{
		runtime:          runtime,
		functionRegistry: functionRegistry,
		ragNodeProcessor: ragNodeProcessor,
	}
}

// RegisterRAGFunction adds a RAG function to the registry
func (r *EnhancedRAGRuntime) RegisterRAGFunction(name string, fn func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error)) {
	r.functionRegistry.RegisterFunc(name, fn)
}

// RegisterRAGProcessor adds a RAG processor to the registry
func (r *EnhancedRAGRuntime) RegisterRAGProcessor(processor RAGProcessor) {
	r.functionRegistry.Register(processor)
}

// GetRuntime returns the underlying FlowGraph runtime
func (r *EnhancedRAGRuntime) GetRuntime() *flowgraph.Runtime {
	return r.runtime
}

// GetFunctionRegistry returns the RAG function registry
func (r *EnhancedRAGRuntime) GetFunctionRegistry() *RAGFunctionRegistry {
	return r.functionRegistry
}

// ExecuteRAGGraph runs a RAG graph with function resolution
func (r *EnhancedRAGRuntime) ExecuteRAGGraph(ctx context.Context, graph *flowgraph.Graph, threadID string, input map[string]interface{}) (*dto.ExecutionResponse, error) {
	// Note: In a real implementation, you would need to integrate the RAGNodeProcessor
	// with the runtime's node processor. For now, this assumes the runtime can handle
	// the function resolution through the existing processor chain.

	return r.runtime.RunSimple(ctx, graph, threadID, input)
}

// Common RAG processor implementations

// OpenAIEmbedder creates embeddings using OpenAI API
type OpenAIEmbedder struct {
	APIKey string
	Model  string
}

func (o *OpenAIEmbedder) Name() string {
	return "openai_embedder"
}

func (o *OpenAIEmbedder) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	text, exists := input["text"]
	if !exists {
		return nil, fmt.Errorf("text field required for embedding")
	}

	textStr, ok := text.(string)
	if !ok {
		return nil, fmt.Errorf("text must be a string")
	}

	// In a real implementation, this would call the OpenAI API
	// For demo purposes, we'll simulate the embedding
	embedding := make([]float64, 1536) // OpenAI embedding dimensions
	for i := range embedding {
		embedding[i] = float64(len(textStr)) / float64(1536) // Simple simulation
	}

	return map[string]interface{}{
		"embedding": embedding,
		"text":      textStr,
		"model":     o.Model,
		"provider":  "openai",
	}, nil
}

// PineconeSearch performs vector search using Pinecone
type PineconeSearch struct {
	APIKey      string
	Environment string
	IndexName   string
}

func (p *PineconeSearch) Name() string {
	return "pinecone_search"
}

func (p *PineconeSearch) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	embedding, exists := input["embedding"]
	if !exists {
		return nil, fmt.Errorf("embedding field required for search")
	}

	// In a real implementation, this would query Pinecone
	// For demo purposes, we'll simulate search results
	results := []map[string]interface{}{
		{
			"id":       "doc1",
			"score":    0.95,
			"metadata": map[string]interface{}{"title": "Relevant Document 1"},
			"content":  "This is relevant content from document 1...",
		},
		{
			"id":       "doc2",
			"score":    0.88,
			"metadata": map[string]interface{}{"title": "Relevant Document 2"},
			"content":  "This is relevant content from document 2...",
		},
	}

	return map[string]interface{}{
		"results":         results,
		"query_embedding": embedding,
		"provider":        "pinecone",
		"index":           p.IndexName,
	}, nil
}

// GPT4Generator generates responses using GPT-4
type GPT4Generator struct {
	APIKey      string
	Model       string
	Temperature float64
	MaxTokens   int
}

func (g *GPT4Generator) Name() string {
	return "gpt4_generator"
}

func (g *GPT4Generator) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
	query, queryExists := input["query"]
	results, resultsExists := input["results"]

	if !queryExists {
		return nil, fmt.Errorf("query field required for generation")
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

	// In a real implementation, this would call GPT-4 API
	// For demo purposes, we'll simulate the response
	response := fmt.Sprintf("Based on the query '%v' and %d context documents, here is a generated response using GPT-4.",
		query, len(contextDocs))

	return map[string]interface{}{
		"response":     response,
		"query":        query,
		"context_docs": len(contextDocs),
		"model":        g.Model,
		"provider":     "openai",
		"temperature":  g.Temperature,
	}, nil
}

// DefaultRAGFunctionRegistry creates a registry with common RAG functions
func DefaultRAGFunctionRegistry() *RAGFunctionRegistry {
	registry := NewRAGFunctionRegistry()

	// Register default implementations
	registry.Register(&OpenAIEmbedder{Model: "text-embedding-ada-002"})
	registry.Register(&PineconeSearch{})
	registry.Register(&GPT4Generator{Model: "gpt-4", Temperature: 0.7, MaxTokens: 1000})

	// Register simple function implementations
	registry.RegisterFunc("query_processor", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"processed_query": input["query"],
			"timestamp":       "2025-09-09T12:00:00Z",
		}, nil
	})

	registry.RegisterFunc("document_retriever", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"documents": []string{"doc1", "doc2", "doc3"},
			"count":     3,
		}, nil
	})

	registry.RegisterFunc("context_formatter", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"formatted_context": "Context: " + fmt.Sprintf("%v", input),
		}, nil
	})

	registry.RegisterFunc("response_generator", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
		return map[string]interface{}{
			"response": "Generated response based on: " + fmt.Sprintf("%v", input),
		}, nil
	})

	return registry
}
