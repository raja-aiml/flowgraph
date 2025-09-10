# Parameterized RAG (Retrieval-Augmented Generation)

The `prebuilt/rag` package provides parameterized, configurable RAG pipeline builders for the FlowGraph framework. Instead of hardcoded values, you can now create flexible RAG configurations tailored to your specific use cases.

## Features

- **Configurable Nodes and Edges**: Define your own RAG pipeline structure
- **Validation**: Built-in validation ensures your configurations are correct
- **Pre-built Templates**: Common RAG patterns available out-of-the-box
- **Extensible**: Easy to customize for different RAG architectures

## Configuration Structure

### SimpleRAGConfig

```go
type SimpleRAGConfig struct {
    ID         string       // Graph ID
    Name       string       // Graph name  
    EntryPoint string       // Entry node ID
    Nodes      []NodeConfig // Node definitions
    Edges      []EdgeConfig // Edge definitions
}
```

### NodeConfig

```go
type NodeConfig struct {
    ID   string // Unique node identifier
    Name string // Human-readable name
    Type string // Node type (processor, retriever, generator, etc.)
    Func string // Function identifier for runtime binding
}
```

### EdgeConfig

```go
type EdgeConfig struct {
    ID     string // Unique edge identifier
    Source string // Source node ID
    Target string // Target node ID
}
```

## Usage Examples

### 1. Default Configuration

```go
config := rag.DefaultSimpleRAGConfig()
graph, err := buildGraph(config)
```

This creates a standard RAG pipeline with:
- Query processing
- Document retrieval  
- Context formation
- Response generation

### 2. Custom Configuration

```go
nodes := []rag.NodeConfig{
    {
        ID:   "input",
        Name: "User Input", 
        Type: "input",
        Func: "user_input_processor",
    },
    {
        ID:   "search",
        Name: "Vector Search",
        Type: "retriever", 
        Func: "pinecone_search",
    },
    // ... more nodes
}

edges := []rag.EdgeConfig{
    {ID: "input-to-search", Source: "input", Target: "search"},
    // ... more edges
}

config := rag.CustomSimpleRAGConfig(
    "my-rag", 
    "My Custom RAG Pipeline",
    "input",
    nodes, 
    edges,
)
```

### 3. Q&A Optimized Configuration

```go
config := rag.QARAGConfig("chromadb_search", "gpt4_generator")
graph, err := buildGraph(config)
```

This creates a pipeline optimized for question-answering with:
- Question embedding
- Vector search
- Result reranking  
- Answer generation

## Validation

All configurations are automatically validated to ensure:

- Non-empty IDs and required fields
- Unique node IDs
- Valid edge references
- Proper entry point specification

```go
if err := rag.ValidateConfig(config); err != nil {
    log.Fatal("Invalid configuration:", err)
}
```

## Building Graphs

```go
builder := rag.NewSimpleRAG()
graph, err := builder.Build(context.Background(), config)
if err != nil {
    return fmt.Errorf("failed to build graph: %w", err)
}
```

## Node Types

Common node types used in RAG pipelines:

- `input`: Entry points for user queries/data
- `processor`: Text processing, embedding, formatting
- `retriever`: Document/data retrieval operations  
- `generator`: Response generation (LLM calls)
- `output`: Final result formatting/output

## Function Identifiers

The `Func` field in `NodeConfig` specifies the runtime function to bind to each node. These should correspond to your actual implementation functions:

- `query_processor`: Process and normalize input queries
- `text_embedder`: Convert text to embeddings
- `document_retriever`: Search document stores
- `context_formatter`: Format retrieved context
- `response_generator`: Generate final responses
- `openai_embedder`: OpenAI embedding service
- `pinecone_search`: Pinecone vector database search
- `gpt4_generator`: GPT-4 response generation

## Error Handling

The package provides comprehensive error handling for:

- Configuration validation errors
- Graph construction errors  
- Runtime binding errors

All errors include detailed context to help with debugging.

## Integration Example

See `examples/rag-demo/main.go` for a complete example showing:

- Default configuration usage
- Custom configuration creation
- Q&A configuration setup
- Validation demonstrations
- Graph building process

## Next Steps

1. Implement your node processors with the function identifiers
2. Set up runtime bindings for your specific LLM/vector DB integrations
3. Use the configured graphs in your FlowGraph runtime
4. Monitor and optimize your RAG pipeline performance

The parameterized approach allows you to easily experiment with different RAG architectures while maintaining clean, validated configurations.
