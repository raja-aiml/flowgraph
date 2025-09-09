package main

import (
    "context"
    "fmt"

    pregel "github.com/flowgraph/flowgraph/internal/core/pregel"
)

// simpleVertex halts immediately; used to demonstrate parallelism config and stats.
type simpleVertex struct{}

func (s *simpleVertex) Compute(vertexID string, state map[string]interface{}, messages []*pregel.Message) (map[string]interface{}, []*pregel.Message, bool, error) {
    return state, nil, true, nil
}

func main() {
    // Single vertex program; real graphs would have many vertices.
    vertices := map[string]pregel.VertexProgram{
        "A": &simpleVertex{},
    }
    initial := map[string]map[string]interface{}{
        "A": {},
    }

    // Use CPU-scaled parallelism for high throughput
    cfg := pregel.Config{
        MaxSupersteps:      1,
        ParallelismFactor:  1.5, // 1.5x NumCPU workers when Parallelism is unset
    }

    engine := pregel.NewEngine(vertices, initial, cfg)
    stats := engine.Stats()
    fmt.Printf("Workers: %d, Queued: %d, Active: %d, Halted: %v\n", stats.Workers, stats.QueuesQueued, stats.ActiveCount, stats.Halted)

    // Run the engine
    _ = engine.Run(context.Background())

    // Print stats after completion
    stats = engine.Stats()
    fmt.Printf("After run — Workers: %d, Queued: %d, Active: %d, Halted: %v\n", stats.Workers, stats.QueuesQueued, stats.ActiveCount, stats.Halted)
}

