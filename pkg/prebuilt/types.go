package prebuilt

import (
    "context"
    "fmt"

    "github.com/flowgraph/flowgraph/pkg/flowgraph"
)

// Builder constructs a FlowGraph from a typed configuration.
// Implementations should be pure (no side effects) and return
// a valid graph that passes core validation.
type Builder interface {
    Name() string
    Build(ctx context.Context, cfg any) (*flowgraph.Graph, error)
}

// BuildFunc is a convenience adapter to implement Builder via functions.
type BuildFunc struct {
    NameStr string
    Fn      func(ctx context.Context, cfg any) (*flowgraph.Graph, error)
}

func (b BuildFunc) Name() string { return b.NameStr }
func (b BuildFunc) Build(ctx context.Context, cfg any) (*flowgraph.Graph, error) {
    return b.Fn(ctx, cfg)
}

// NewBuildFunc creates a Builder from a function.
func NewBuildFunc(name string, fn func(ctx context.Context, cfg any) (*flowgraph.Graph, error)) BuildFunc {
    return BuildFunc{NameStr: name, Fn: fn}
}

// Registry holds named prebuilts and their default config constructors.
type Registry struct {
    builders map[string]Builder
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
    return &Registry{builders: make(map[string]Builder)}
}

// Register adds or replaces a prebuilt builder.
func (r *Registry) Register(b Builder) {
    r.builders[b.Name()] = b
}

// MustRegister panics on duplicate names; useful during init() setup.
func (r *Registry) MustRegister(b Builder) {
    if _, exists := r.builders[b.Name()]; exists {
        panic(fmt.Sprintf("prebuilt already registered: %s", b.Name()))
    }
    r.builders[b.Name()] = b
}

// Get retrieves a named prebuilt.
func (r *Registry) Get(name string) (Builder, bool) {
    b, ok := r.builders[name]
    return b, ok
}

// DefaultRegistry is a singleton for convenience. Projects can also
// construct their own Registry if they want isolation.
var DefaultRegistry = NewRegistry()
