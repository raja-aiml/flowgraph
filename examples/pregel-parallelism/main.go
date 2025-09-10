package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	pregel "github.com/flowgraph/flowgraph/internal/core/pregel"
)

// workerVertex simulates a distributed computation node
type workerVertex struct{}

func (w *workerVertex) Compute(vertexID string, state map[string]interface{}, messages []*pregel.Message) (map[string]interface{}, []*pregel.Message, bool, error) {
	// Simulate computation
	result := rand.Intn(1000)
	state["result"] = result
	time.Sleep(time.Millisecond * time.Duration(rand.Intn(50))) // Simulate work
	return state, nil, true, nil
}

func main() {
	fmt.Println("🖥️ Distributed Computation (FlowGraph Real-World Example)")
	fmt.Println("======================================================")

	// Multiple worker nodes
	vertices := map[string]pregel.VertexProgram{}
	initial := map[string]map[string]interface{}{}
	for i := 1; i <= 8; i++ {
		id := fmt.Sprintf("worker-%d", i)
		vertices[id] = &workerVertex{}
		initial[id] = map[string]interface{}{}
	}

	// Parallel execution config
	cfg := pregel.Config{
		MaxSupersteps:     2,
		ParallelismFactor: 2.0, // 2x NumCPU workers
	}

	engine := pregel.NewEngine(vertices, initial, cfg)
	fmt.Printf("Starting distributed computation with %d workers...\n", len(vertices))

	// Run the engine
	_ = engine.Run(context.Background())

	// Aggregate results
	results := make([]int, 0, len(vertices))
	for id, state := range initial {
		if v, ok := state["result"].(int); ok {
			results = append(results, v)
			fmt.Printf("Worker %s result: %d\n", id, v)
		}
	}
	fmt.Printf("\n✅ Computation complete. Aggregated results: %v\n", results)
}
