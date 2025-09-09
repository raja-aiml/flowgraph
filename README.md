# FlowGraph 🌊

**A high-performance Go implementation of graph-based workflows with 100% feature parity**

FlowGraph is graph-based workflow execution engine, 

## 🎯 Project Goals

- **100% Feature Parity**: Complete graph-based workflow execution capabilities
- **10x Performance**: Leverage Go's concurrency for superior execution speed
- **Clean Architecture**: Strict separation of concerns and dependency management
- **Quality First**: >95% test coverage with automated principle enforcement

## 🏗️ Architecture

FlowGraph follows Clean Architecture principles with clear separation of layers:

```
flowgraph/
├── cmd/                    # 🚀 Application entry points
│   ├── flowgraph/         # CLI application
│   └── flowgraph-server/  # Server application
├── internal/              # 🔒 Private packages
│   ├── core/             # 🎯 Domain logic (zero dependencies)
│   │   ├── graph/        # Graph entities and business rules
│   │   ├── checkpoint/   # Checkpoint domain logic
│   │   ├── pregel/       # Pregel engine core
│   │   └── channel/      # Channel abstractions
│   ├── usecases/         # 📋 Application business logic
│   │   ├── graph/        # Graph use cases
│   │   └── checkpoint/   # Checkpoint use cases
│   ├── adapters/         # 🔌 External interfaces
│   │   ├── repository/   # Data persistence
│   │   └── api/          # HTTP/gRPC handlers
│   └── infrastructure/   # ⚙️ Technical details
│       ├── config/       # Configuration management
│       ├── logger/       # Logging infrastructure
│       └── metrics/      # Metrics and monitoring
├── pkg/                  # 📦 Public packages
│   ├── flowgraph/       # Public API
│   └── serialization/   # Serialization utilities
└── test/                # 🧪 Testing infrastructure
    ├── integration/     # Integration tests
    └── fixtures/        # Test data and utilities
```

## 🔧 Development Setup

### Prerequisites

- Go 1.21+
- Make
- Git

### Quick Start

```bash
# Clone the repository
git clone https://github.com/flowgraph/flowgraph.git
cd flowgraph

# Install development tools
make install-tools

# Run quality checks
make pre-commit

# Start development server
make dev
```

## 🎯 Architectural Principles

FlowGraph enforces strict architectural principles through automated validation:

### KISS (Keep It Simple, Stupid)
- Functions ≤50 lines
- Cyclomatic complexity ≤10
- Nesting ≤3 levels
- Self-documenting code

### YAGNI (You Aren't Gonna Need It)
- No unused code
- No speculative features
- Delete immediately when not needed

### SOLID Principles
- **Single Responsibility**: One reason to change
- **Open/Closed**: Open for extension, closed for modification
- **Liskov Substitution**: Interface implementations are interchangeable
- **Interface Segregation**: Interfaces ≤5 methods
- **Dependency Inversion**: Core domain has zero external dependencies

### DRY (Don't Repeat Yourself)
- Extract after 3rd duplication
- Centralized constants and validation
- Shared utilities and helpers

## 🚀 Available Commands

```bash
# Quality Assurance
make principle-check    # Validate architectural principles
make lint              # Run linter with architectural rules
make test              # Run all tests
make coverage-check    # Ensure >95% coverage

# Development
make dev               # Start development with hot reload
make build             # Build binary
make clean             # Clean build artifacts

# Pre-commit checks (must pass)
make pre-commit        # Run all quality checks
```

## 📊 Quality Gates

Every commit must pass these automated checks:

- ✅ **Architectural Principles**: `make principle-check`
- ✅ **Code Quality**: `make lint` (zero violations)
- ✅ **Test Coverage**: `make coverage-check` (>95%)
- ✅ **All Tests**: `make test` (100% pass rate)

## 🧪 Testing Strategy

- **Unit Tests**: >95% coverage for all business logic
- **Integration Tests**: Database and external service integration
- **Property-Based Testing**: Automated test case generation
- **Benchmarks**: Performance regression detection

## 🔍 Code Quality Enforcement

FlowGraph uses automated tools to enforce architectural principles:

- **golangci-lint**: 20+ linters with custom rules
- **principle-check.sh**: Custom validation script
- **CI/CD Pipeline**: Blocks merges on violations
- **Pre-commit Hooks**: Catch issues early

## 📈 Performance Goals

- **Throughput**: 10,000+ concurrent graph executions
- **Latency**: Sub-millisecond message passing
- **Memory**: Efficient garbage collection and resource usage
- **Scalability**: Horizontal scaling with distributed state

## 🛠️ Core Components

### Graph Processing Engine
- Pregel-inspired architecture
- Parallel execution with goroutines
- State management with consistency guarantees
- Channel-based communication

### Checkpoint System
- Multiple storage backends (PostgreSQL, SQLite, Memory)
- Cross-language compatibility
- Efficient serialization (MessagePack, JSON)
- Transaction support

### Prebuilt Components
- ReAct agents
- Tool execution nodes
- Validation frameworks
- Human-in-the-loop patterns

## 🚦 Project Status

- ✅ **Phase 1**: Project structure and foundation
- 🔄 **Phase 2**: Base checkpoint library (In Progress)
- ⏳ **Phase 3**: Database checkpointers
- ⏳ **Phase 4**: Core FlowGraph engine
- ⏳ **Phase 5**: Prebuilt components
- ⏳ **Phase 6**: SDK and client library
- ⏳ **Phase 7**: CLI tool

## 🤝 Contributing

We welcome contributions! Please ensure all code follows our architectural principles:

1. **Read**: [Architecture Guide](./GO_ARCHITECTURE_GUIDE.md)
2. **Check**: Run `make pre-commit` before submitting
3. **Test**: Maintain >95% test coverage
4. **Document**: Update docs for new features

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Graph Workflow Community**: For inspiration and design patterns
- **Go Community**: For the excellent ecosystem and tools
- **Clean Architecture**: Robert C. Martin's architectural principles

---

**Built with ❤️ and Go's performance in mind**

## ⚡ Parallelism & Metrics

- Parallelism defaults: If `Parallelism <= 0`, the engine uses `runtime.NumCPU()` workers for high-throughput goroutines.
- CPU scaling: Set `ParallelismFactor` to scale workers relative to CPU cores (e.g., `1.5` → 1.5× CPUs). If `Parallelism > 0`, it takes precedence.
- Scheduler stats: Introspect worker count and queue depths via `Scheduler.Stats()`.
- Engine stats: Observe `Workers`, `QueuesQueued`, `ActiveCount`, and `Halted` via `Engine.Stats()`.

Example:

```go
package main

import (
    "context"
    "fmt"
    pregel "github.com/flowgraph/flowgraph/internal/core/pregel"
)

type noopVertex struct{}

func (n *noopVertex) Compute(id string, state map[string]interface{}, msgs []*pregel.Message) (map[string]interface{}, []*pregel.Message, bool, error) {
    return state, nil, true, nil // halt immediately
}

func main() {
    verts := map[string]pregel.VertexProgram{"A": &noopVertex{}}
    states := map[string]map[string]interface{}{"A": {}}
    cfg := pregel.Config{MaxSupersteps: 1, ParallelismFactor: 1.5}
    eng := pregel.NewEngine(verts, states, cfg)

    fmt.Printf("Workers: %d\n", eng.Stats().Workers)
    _ = eng.Run(context.Background())
    fmt.Printf("After run — Active: %d, Halted: %v\n", eng.Stats().ActiveCount, eng.Stats().Halted)
}
```

## 🔧 Tuning Scheduler & Channels

- Scheduler queue capacity: set `pregel.Config{QueueCapacity: 1000}` to increase per-worker queue size.
- Channel defaults:
  - Set defaults at startup:
    ```go
    channel.SetDefaultRuntimeConfig(channel.RuntimeConfig{
        InMemoryBufferSize:  1024,
        InMemoryTimeout:     5 * time.Second,
        BufferedMaxSize:     4096,
        BufferedTimeout:     5 * time.Second,
        PersistentTimeout:   10 * time.Second,
    })
    ```
  - `DefaultInMemoryChannel/DefaultBufferedChannel/DefaultPersistentChannel` now respect these overrides.

## 📈 Benchmarking & Profiling

- Channels:
  - `go test ./internal/core/channel -run ^$ -bench . -benchmem`
- Pregel:
  - `go test ./internal/core/pregel -run ^$ -bench . -benchmem`
- With CPU/memory profiles:
  - `bash scripts/bench.sh`
  - Generates `channel_cpu.out`, `channel_mem.out`, `pregel_cpu.out`, `pregel_mem.out` in `flowgraph/`

## 🧳 PersistentChannel Retention

- Configure retention to prevent disk bloat:
  - `MaxMessages`: keep at most N messages (0 = unlimited)
  - `MaxAge`: expire messages older than duration
  - `OnFull`: when exceeding `MaxSizeMB`, choose `"reject"` (default) or `"evict_oldest"`
  - `CleanupInterval`: run periodic compaction if > 0

Example:

```go
cfg := channel.PersistentChannelConfig{
    DataDir:         "./data/queue",
    MaxSizeMB:       100,
    MaxMessages:     10000,
    MaxAge:          24 * time.Hour,
    OnFull:          "evict_oldest",
    CleanupInterval: time.Minute,
}
ch, _ := channel.NewPersistentChannel(cfg)
```

## 📊 Metrics (expvar-powered)

- FlowGraph exposes basic counters/gauges via expvar:
  - Channels: `flowgraph_channel_sent_total`, `flowgraph_channel_received_total`, `flowgraph_channel_evicted_total`, `flowgraph_channel_size_bytes`
  - Engine/Scheduler: `flowgraph_supersteps_total`, `flowgraph_vertex_executions_total`, `flowgraph_scheduler_workers`, `flowgraph_scheduler_queued_total`
- Access by importing `net/http/pprof` or serving `/debug/vars`:
  ```go
  import (
      _ "net/http/pprof"
      _ "expvar"
      "net/http"
  )
  func main(){ go http.ListenAndServe(":8080", nil) }
  // Then GET http://localhost:8080/debug/vars
  ```

## 📦 Public API (pkg/flowgraph)

- Use a simple façade to build and run graphs without importing internals directly.

```go
rt := flowgraph.NewRuntime()
g := &flowgraph.Graph{ID: "g1", Name: "Demo", EntryPoint: "start", Nodes: map[string]*flowgraph.Node{
    "start": {ID: "start", Name: "Start", Type: flowgraph.NodeType("function"),
}}
resp, err := rt.RunSimple(context.Background(), g, "thread-1", map[string]interface{}{"msg": "hello"})
```
