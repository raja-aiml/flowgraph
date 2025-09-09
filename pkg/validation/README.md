# FlowGraph Validation Framework

A comprehensive validation framework for the FlowGraph project that provides Pydantic-like validation capabilities in Go.

## 🎯 Overview

The FlowGraph Validation Framework provides:
- **Struct-based validation** using reflection and tags
- **Enhanced validation** with go-playground/validator integration  
- **Custom FlowGraph-specific validation rules**
- **HTTP middleware** for request/response validation
- **JSON/YAML marshaling** with proper error formatting
- **Comprehensive test coverage** with examples

## 🚀 Features

### ✅ Core Validation Capabilities
- **Basic validation**: required, min, max, length validation
- **Type validation**: email, URL, UUID, numeric validation
- **Custom validation**: FlowGraph-specific rules for nodes, edges, channels
- **Nested validation**: Deep validation of struct hierarchies
- **Error aggregation**: Multiple validation errors with detailed messages

### 🏗️ FlowGraph-Specific Validators
- `node_id`: Validates node identifier format (alphanumeric, underscore, hyphen)
- `edge_id`: Validates edge identifier format (source->target:condition)
- `graph_state`: Validates graph state (pending, running, completed, failed, cancelled, paused)
- `channel_name`: Validates channel name format (lowercase alphanumeric with underscore)
- `message_type`: Validates message type (data, control, error, heartbeat)
- `uuid4`: Validates UUID v4 format
- `semver`: Validates semantic version format

### 🌐 HTTP Middleware Support
- **JSON body validation**: Automatic request body validation
- **Query parameter validation**: URL parameter validation with custom rules
- **Header validation**: HTTP header validation (Authorization, Content-Type, etc.)
- **Fluent API**: Chain validation rules for complex scenarios
- **Error responses**: Standardized JSON error format

## 📦 Installation

The validation framework is included in the FlowGraph project:

```bash
go get github.com/flowgraph/flowgraph/pkg/validation
```

Dependencies:
- `github.com/go-playground/validator/v10` - Enhanced validation engine
- `github.com/stretchr/testify` - Testing utilities

## 🔧 Quick Start

### Basic Struct Validation

```go
package main

import (
    "fmt"
    "github.com/flowgraph/flowgraph/pkg/validation"
)

type User struct {
    Name  string `validate:"required,min=2,max=50"`
    Email string `validate:"required,email"`
    Age   int    `validate:"min=0,max=120"`
}

func main() {
    user := User{
        Name:  "John Doe",
        Email: "john@example.com", 
        Age:   25,
    }
    
    if err := validation.ValidateWithPlayground(user); err != nil {
        fmt.Printf("Validation failed: %v\n", err)
    } else {
        fmt.Println("✅ User is valid")
    }
}
```

### FlowGraph-Specific Validation

```go
type GraphNode struct {
    ID       string `validate:"required,node_id"`
    State    string `validate:"required,graph_state"`
    Channel  string `validate:"required,channel_name"`
    MsgType  string `validate:"required,message_type"`
}

node := GraphNode{
    ID:      "processing_node_1",
    State:   "running",
    Channel: "data_channel", 
    MsgType: "data",
}

err := validation.ValidateWithPlayground(node)
```

### Custom Business Logic Validation

```go
type NodeConfig struct {
    ID      string         `validate:"required,node_id"`
    Type    string         `validate:"required,oneof=function tool condition"`
    Timeout *time.Duration `validate:"omitempty,min=1s,max=300s"`
}

// Implement custom validation
func (nc *NodeConfig) Validate() error {
    if nc.Type == "condition" && nc.Timeout != nil && *nc.Timeout > 60*time.Second {
        return validation.ValidationErrors{{
            Field:   "timeout",
            Value:   nc.Timeout,
            Message: "condition nodes cannot have timeout > 60s",
        }}
    }
    return nil
}
```

### HTTP Middleware Validation

```go
package main

import (
    "net/http"
    "github.com/flowgraph/flowgraph/pkg/validation"
)

type CreateNodeRequest struct {
    Name    string `json:"name" validate:"required,min=1,max=100"`
    Type    string `json:"type" validate:"required,oneof=function tool condition"`
    Enabled bool   `json:"enabled"`
}

func main() {
    // Create your handler
    handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Node created"))
    })
    
    // Add validation middleware
    validator := validation.NewRequestValidator(validation.DefaultValidationConfig()).
        JSON(CreateNodeRequest{}).
        QueryParams(map[string]string{"graph_id": "required"}).
        Headers(map[string]string{"Authorization": "required"})
    
    // Wrap handler
    validatedHandler := validator.Build()(handler)
    
    http.Handle("/nodes", validatedHandler)
    http.ListenAndServe(":8080", nil)
}
```

## 🏛️ Architecture

### Core Components

```
pkg/validation/
├── validator.go      # Basic validation with reflection
├── enhanced.go       # go-playground/validator integration  
├── middleware.go     # HTTP middleware components
├── models.go         # Validation model definitions
└── validation_test.go # Comprehensive tests
```

### Validation Flow

1. **Input** → Struct with validation tags
2. **Tag Processing** → Parse validation rules from struct tags
3. **Rule Execution** → Execute built-in and custom validation rules
4. **Error Aggregation** → Collect and format validation errors
5. **Output** → ValidationErrors or nil

### Error Handling

```go
type ValidationError struct {
    Field   string      `json:"field"`
    Value   interface{} `json:"value"`
    Message string      `json:"message"`
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
    // Returns formatted error string
}
```

## 📋 Validation Tags

### Built-in Tags
- `required`: Field must not be zero/empty
- `min=N`: Minimum value/length
- `max=N`: Maximum value/length  
- `len=N`: Exact length
- `email`: Valid email format
- `url`: Valid URL format
- `oneof=a b c`: Value must be one of specified options
- `dive`: Validate slice/map elements

### FlowGraph Custom Tags
- `node_id`: Node identifier (alphanumeric, _, -)
- `edge_id`: Edge identifier (source->target format)
- `graph_state`: Graph state enum validation
- `channel_name`: Channel name format validation
- `message_type`: Message type enum validation
- `uuid4`: UUID v4 format validation
- `semver`: Semantic version format validation

## 🧪 Testing

Run the complete test suite:

```bash
cd flowgraph
go test -v ./pkg/validation
```

Run the validation demo:

```bash
go run cmd/validation-demo/main.go
```

### Test Coverage

The validation framework achieves **100% test coverage** with:
- ✅ Unit tests for all validation functions
- ✅ Integration tests for middleware
- ✅ Edge case testing
- ✅ Error handling verification
- ✅ Performance benchmarks

## 🔧 Configuration

### Validation Configuration

```go
config := &validation.ValidationConfig{
    StrictMode:  true,  // Fail on any validation error
    SkipMissing: false, // Don't skip missing fields
    MaxErrors:   10,    // Maximum errors to report
}

err := validation.ValidateWithConfig(struct, config)
```

### Custom Validators

Register custom validation functions:

```go
validation.Validate.RegisterValidation("custom_rule", func(fl validator.FieldLevel) bool {
    // Your custom validation logic
    return true
})
```

## 🚀 Performance

The validation framework is optimized for:
- **Low latency**: Minimal reflection overhead
- **Memory efficiency**: Reusable validator instances
- **Concurrency**: Thread-safe validation operations
- **Scalability**: Suitable for high-throughput APIs

## 🤝 Integration

### With FlowGraph Components

The validation framework integrates seamlessly with:
- **Graph Configuration**: Validate node and edge configurations
- **Channel Messages**: Validate message structure and content  
- **Checkpoint Data**: Validate checkpoint state and metadata
- **API Requests**: Validate HTTP requests and responses
- **Serialization**: Validate data before/after JSON/YAML marshaling

### Example Integration

```go
// Validate graph configuration before execution
func ExecuteGraph(config validation.GraphConfig) error {
    // Structural validation
    if err := validation.ValidateWithPlayground(config); err != nil {
        return fmt.Errorf("invalid graph config: %w", err)
    }
    
    // Business logic validation  
    if err := config.Validate(); err != nil {
        return fmt.Errorf("graph validation failed: %w", err)
    }
    
    // Execute graph...
    return nil
}
```

## 📚 Examples

See the complete examples in:
- `cmd/validation-demo/main.go` - Comprehensive demonstration
- `pkg/validation/validation_test.go` - Test examples
- `pkg/validation/models.go` - Model definitions with validation

## 🔮 Future Enhancements

Planned improvements:
- [ ] Schema validation for JSON payloads
- [ ] Internationalization (i18n) for error messages  
- [ ] Performance optimizations for large structs
- [ ] Integration with OpenAPI/Swagger specifications
- [ ] Validation rule composition and reuse
- [ ] Real-time validation for streaming data

---

**The FlowGraph Validation Framework provides enterprise-grade validation capabilities that ensure data integrity and API reliability throughout the FlowGraph system.**
