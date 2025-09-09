package graphrepo

import (
    "context"
    "fmt"
    "sync"

    "github.com/flowgraph/flowgraph/internal/core/graph"
    "github.com/flowgraph/flowgraph/pkg/validation"
)

// InMemoryGraphRepository provides an in-memory implementation of a graph repository
// PRINCIPLES:
// - KISS: Simple map-based storage
// - SRP: Only responsible for graph persistence
// - Thread-safe

type InMemoryGraphRepository struct {
	mu     sync.RWMutex
	graphs map[string]*graph.Graph
}

func NewInMemoryGraphRepository() *InMemoryGraphRepository {
	return &InMemoryGraphRepository{
		graphs: make(map[string]*graph.Graph),
	}
}

func (r *InMemoryGraphRepository) Save(ctx context.Context, g *graph.Graph) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    // Validate graph structure before saving (no cycle check by default)
    if err := validation.ValidateCoreGraph(g); err != nil {
        return fmt.Errorf("invalid graph: %w", err)
    }
    r.graphs[g.ID] = g
    return nil
}

func (r *InMemoryGraphRepository) Get(ctx context.Context, id string) (*graph.Graph, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    g, ok := r.graphs[id]
    if !ok {
        return nil, graph.ErrGraphNotFound
    }
    return g, nil
}

func (r *InMemoryGraphRepository) List(ctx context.Context) ([]*graph.Graph, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []*graph.Graph
	for _, g := range r.graphs {
		out = append(out, g)
	}
	return out, nil
}
