# Enhanced RAG with Real Function Implementations

This example demonstrates how to move beyond hardcoded function names to actual function implementations in your RAG pipelines. The enhanced system provides a registry-based approach for binding real API calls and custom logic to RAG nodes.

## 🚀 Key Features

- **Function Registry**: Register actual implementations instead of just string names
- **Type-Safe Processing**: Structured interfaces for RAG processors
- **Real API Integration**: Examples with OpenAI, Pinecone, ChromaDB, Claude 3
- **Custom Processors**: Easy to create your own RAG components
- **Runtime Binding**: Functions are resolved and executed at runtime

## 📋 Architecture Overview

### Before: Hardcoded Function Names
```go
// Old way - just string identifiers
{
    ID:   "embedder",
    Name: "Embedding Node",
    Type: "processor", 
    Func: "openai_embedder", // Just a string!
}
```

### After: Real Function Implementations
```go
// New way - actual implementations
openaiEmbedder := &rag.OpenAIEmbedder{
    APIKey: "sk-your-key",
    Model:  "text-embedding-ada-002",
}
ragRuntime.RegisterRAGProcessor(openaiEmbedder)

// Now "openai_embedder" calls real OpenAI API
```

## 🛠 Implementation Guide

### 1. Create Enhanced RAG Runtime

```go
// Create runtime with function registry
ragRuntime := rag.NewEnhancedRAGRuntime()
```

### 2. Register Function Implementations

#### A. Built-in Processors
```go
// OpenAI Embeddings
openaiEmbedder := &rag.OpenAIEmbedder{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "text-embedding-ada-002",
}
ragRuntime.RegisterRAGProcessor(openaiEmbedder)

// Pinecone Vector Search  
pineconeSearch := &rag.PineconeSearch{
    APIKey:      os.Getenv("PINECONE_API_KEY"),
    Environment: "us-west1-gcp",
    IndexName:   "my-rag-index",
}
ragRuntime.RegisterRAGProcessor(pineconeSearch)

// GPT-4 Generation
gpt4Generator := &rag.GPT4Generator{
    APIKey:      os.Getenv("OPENAI_API_KEY"),
    Model:       "gpt-4",
    Temperature: 0.7,
    MaxTokens:   1000,
}
ragRuntime.RegisterRAGProcessor(gpt4Generator)
```

#### B. Custom Function Implementations
```go
// Register a custom function
ragRuntime.RegisterRAGFunction("chromadb_search", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    embedding := input["embedding"]
    
    // Real ChromaDB API call here
    client := chromadb.NewClient("http://localhost:8000")
    results, err := client.Query(
        collection="my_collection",
        query_embeddings=[embedding],
        n_results=5,
    )
    if err != nil {
        return nil, err
    }
    
    return map[string]interface{}{
        "results":   results,
        "provider":  "chromadb",
        "timestamp": time.Now(),
    }, nil
})
```

#### C. Custom Processor Structs
```go
type CustomDocumentProcessor struct {
    DatabaseURL string
    APIKey      string
    Client      *http.Client
}

func (c *CustomDocumentProcessor) Name() string {
    return "custom_document_processor"
}

func (c *CustomDocumentProcessor) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Your custom processing logic
    query := input["query"].(string)
    
    // Call your custom API
    response, err := c.Client.Post(c.DatabaseURL, "application/json", /* payload */)
    if err != nil {
        return nil, err
    }
    
    // Process response
    return map[string]interface{}{
        "processed_query": processedData,
        "source": "custom_api",
    }, nil
}

// Register it
customProcessor := &CustomDocumentProcessor{
    DatabaseURL: "https://api.myservice.com/process",
    APIKey: os.Getenv("CUSTOM_API_KEY"),
    Client: &http.Client{Timeout: 30 * time.Second},
}
ragRuntime.RegisterRAGProcessor(customProcessor)
```

### 3. Create RAG Configuration

```go
// Same configuration as before - but now functions are real!
config := rag.CustomSimpleRAGConfig(
    "production-rag",
    "Production RAG with Real APIs",
    "input",
    []rag.NodeConfig{
        {ID: "input", Type: "processor", Func: "query_processor"},
        {ID: "embed", Type: "processor", Func: "openai_embedder"},     // Real OpenAI
        {ID: "search", Type: "retriever", Func: "pinecone_search"},    // Real Pinecone  
        {ID: "generate", Type: "generator", Func: "gpt4_generator"},   // Real GPT-4
    },
    []rag.EdgeConfig{
        {ID: "input-embed", Source: "input", Target: "embed"},
        {ID: "embed-search", Source: "embed", Target: "search"},
        {ID: "search-generate", Source: "search", Target: "generate"},
    },
)
```

### 4. Execute RAG Pipeline

```go
// Build the graph
graph, err := rag.NewSimpleRAG().Build(context.Background(), config)
if err != nil {
    log.Fatal(err)
}

// Execute with real function calls
result, err := ragRuntime.ExecuteRAGGraph(
    context.Background(),
    graph,
    "user_session_123",
    map[string]interface{}{
        "query": "What are the benefits of vector databases for RAG?",
    },
)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("RAG Response: %v\n", result.Output)
```

## 🔌 Integration Examples

### OpenAI + Pinecone Stack
```go
// Production-ready OpenAI + Pinecone integration
ragRuntime.RegisterRAGProcessor(&rag.OpenAIEmbedder{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model: "text-embedding-ada-002",
})

ragRuntime.RegisterRAGProcessor(&rag.PineconeSearch{
    APIKey: os.Getenv("PINECONE_API_KEY"),
    Environment: os.Getenv("PINECONE_ENVIRONMENT"),
    IndexName: "production-rag-index",
})

ragRuntime.RegisterRAGProcessor(&rag.GPT4Generator{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model: "gpt-4-turbo-preview",
    Temperature: 0.3, // Lower for more focused responses
    MaxTokens: 2000,
})
```

### HuggingFace + ChromaDB + Claude Stack
```go
// Alternative stack with different providers
ragRuntime.RegisterRAGFunction("hf_embedder", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // HuggingFace Inference API call
    client := huggingface.NewInferenceClient(os.Getenv("HF_API_KEY"))
    embeddings, err := client.FeatureExtraction(
        "sentence-transformers/all-MiniLM-L6-v2",
        input["text"].(string),
    )
    return map[string]interface{}{
        "embedding": embeddings,
        "provider": "huggingface",
    }, err
})

ragRuntime.RegisterRAGFunction("chromadb_search", chromaDBSearchFunc)
ragRuntime.RegisterRAGFunction("claude_generator", claudeGeneratorFunc)
```

### Local/Self-Hosted Stack  
```go
// Fully local stack for privacy/cost optimization
ragRuntime.RegisterRAGFunction("local_embedder", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Call local sentence-transformers model
    return callLocalEmbeddingModel(input["text"].(string))
})

ragRuntime.RegisterRAGFunction("local_llm", func(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Call local Ollama or similar
    return callOllamaAPI("llama2", input)
})
```

## 🎯 Best Practices

### 1. Environment Configuration
```go
// Use environment variables for API keys
type Config struct {
    OpenAIKey     string `env:"OPENAI_API_KEY"`
    PineconeKey   string `env:"PINECONE_API_KEY"`
    PineconeEnv   string `env:"PINECONE_ENV" envDefault:"us-west1-gcp"`
    ChromaDBURL   string `env:"CHROMADB_URL" envDefault:"http://localhost:8000"`
}
```

### 2. Error Handling
```go
func (p *ProductionEmbedder) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Validate input
    text, exists := input["text"]
    if !exists {
        return nil, fmt.Errorf("text field required")
    }
    
    // Retry logic for API calls
    var result map[string]interface{}
    var err error
    for attempt := 0; attempt < 3; attempt++ {
        result, err = p.callAPI(ctx, text)
        if err == nil {
            break
        }
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }
    
    return result, err
}
```

### 3. Monitoring and Logging
```go
func (p *MonitoredProcessor) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    start := time.Now()
    
    // Log input (be careful with sensitive data)
    log.Printf("Processing with %s: input_keys=%v", p.Name(), getKeys(input))
    
    result, err := p.actualProcessor.Process(ctx, input)
    
    // Log metrics
    duration := time.Since(start)
    log.Printf("Completed %s: duration=%v, error=%v", p.Name(), duration, err != nil)
    
    return result, err
}
```

### 4. Testing with Mock Implementations
```go
// Create mock implementations for testing
type MockEmbedder struct{}

func (m *MockEmbedder) Name() string { return "openai_embedder" }

func (m *MockEmbedder) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    // Return deterministic test data
    return map[string]interface{}{
        "embedding": []float64{0.1, 0.2, 0.3}, // Mock embedding
        "text": input["text"],
    }, nil
}

// In tests
testRuntime := rag.NewEnhancedRAGRuntime()
testRuntime.RegisterRAGProcessor(&MockEmbedder{})
```

## 🔍 Debugging

### Function Registry Inspection
```go
// Check what functions are registered
registry := ragRuntime.GetFunctionRegistry()
// List available functions for debugging

// Validate configuration before execution
if err := rag.ValidateConfig(config); err != nil {
    log.Fatalf("Invalid RAG config: %v", err)
}
```

### Execution Tracing
```go
// Add execution tracing to your processors
func (p *TracedProcessor) Process(ctx context.Context, input map[string]interface{}) (map[string]interface{}, error) {
    trace := map[string]interface{}{
        "processor": p.Name(),
        "timestamp": time.Now(),
        "input_keys": getKeys(input),
    }
    
    result, err := p.baseProcessor.Process(ctx, input)
    
    trace["output_keys"] = getKeys(result)
    trace["error"] = err
    
    // Log or store trace
    logExecution(trace)
    
    return result, err
}
```

## 📊 Performance Considerations

- **Connection Pooling**: Reuse HTTP clients for API calls
- **Caching**: Cache embeddings and search results when appropriate
- **Batching**: Process multiple inputs together when APIs support it
- **Async Processing**: Use goroutines for independent operations
- **Circuit Breakers**: Implement circuit breakers for external API resilience

## 🚀 Running the Demo

```bash
# Set environment variables
export OPENAI_API_KEY="your-openai-key"
export PINECONE_API_KEY="your-pinecone-key"
export PINECONE_ENVIRONMENT="us-west1-gcp"

# Run the enhanced demo
go run examples/enhanced-rag-demo/main.go
```

This enhanced approach transforms your RAG pipeline from a configuration-only system into a fully functional, production-ready implementation with real API integrations!
