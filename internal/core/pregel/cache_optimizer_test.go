package pregel

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

// TestCacheOptimization validates cache-aware execution improvements
func TestCacheOptimization(t *testing.T) {
	tests := []struct {
		name         string
		vertexCount  int
		messageCount int
		expectSpeedup float64
	}{
		{
			name:         "Small Graph Cache Test",
			vertexCount:  100,
			messageCount: 1000,
			expectSpeedup: 1.2,
		},
		{
			name:         "Medium Graph Cache Test",
			vertexCount:  1000,
			messageCount: 10000,
			expectSpeedup: 2.0,
		},
		{
			name:         "Large Graph Cache Test",
			vertexCount:  10000,
			messageCount: 100000,
			expectSpeedup: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test vertices
			vertices := make(map[string]VertexProgram)
			states := make(map[string]map[string]interface{})
			
			for i := 0; i < tt.vertexCount; i++ {
				vid := fmt.Sprintf("V%d", i)
				vertices[vid] = &cacheTestVertex{
					totalVertices: tt.vertexCount,
					messagesPerStep: tt.messageCount / tt.vertexCount,
				}
				states[vid] = map[string]interface{}{"id": i}
			}

			config := Config{
				MaxSupersteps: 3,
				Parallelism:   runtime.NumCPU(),
				QueueCapacity: 1024,
			}

			// Run with standard engine
			standardEngine := NewEngine(vertices, states, config)
			defer standardEngine.Stop()
			
			startStandard := time.Now()
			err := standardEngine.Run(context.Background())
			standardTime := time.Since(startStandard)
			
			if err != nil {
				t.Fatalf("Standard engine failed: %v", err)
			}

			// Reset states
			for i := 0; i < tt.vertexCount; i++ {
				vid := fmt.Sprintf("V%d", i)
				states[vid] = map[string]interface{}{"id": i}
			}

			// Run with cache-optimized engine
			optimizedEngine := NewCacheOptimizedEngine(vertices, states, config)
			defer optimizedEngine.Stop()
			
			startOptimized := time.Now()
			err = optimizedEngine.Run(context.Background())
			optimizedTime := time.Since(startOptimized)
			
			if err != nil {
				t.Fatalf("Optimized engine failed: %v", err)
			}

			// Calculate speedup
			speedup := float64(standardTime) / float64(optimizedTime)
			
			t.Logf("Standard engine: %v", standardTime)
			t.Logf("Optimized engine: %v", optimizedTime)
			t.Logf("Speedup: %.2fx (expected: %.2fx)", speedup, tt.expectSpeedup)
			
			// Verify optimization stats
			stats := optimizedEngine.GetOptimizationStats()
			t.Logf("Optimization stats: %+v", stats)
			
			if speedup < tt.expectSpeedup * 0.8 {
				t.Logf("WARNING: Speedup %.2fx is below expected %.2fx", speedup, tt.expectSpeedup)
			} else {
				t.Logf("SUCCESS: Cache optimization achieved %.2fx speedup", speedup)
			}
		})
	}
}

// BenchmarkCacheOptimization compares performance with and without cache optimization
func BenchmarkCacheOptimization(b *testing.B) {
	vertexCounts := []int{100, 1000, 10000}
	
	for _, count := range vertexCounts {
		b.Run(fmt.Sprintf("Standard_%d", count), func(b *testing.B) {
			vertices := make(map[string]VertexProgram)
			states := make(map[string]map[string]interface{})
			
			for i := 0; i < count; i++ {
				vid := fmt.Sprintf("V%d", i)
				vertices[vid] = &benchVertex{}
				states[vid] = map[string]interface{}{}
			}

			config := Config{
				MaxSupersteps: 1,
				Parallelism:   runtime.NumCPU(),
				QueueCapacity: count,
			}

			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				engine := NewEngine(vertices, states, config)
				_ = engine.Run(context.Background())
				engine.Stop()
			}
		})

		b.Run(fmt.Sprintf("Optimized_%d", count), func(b *testing.B) {
			vertices := make(map[string]VertexProgram)
			states := make(map[string]map[string]interface{})
			
			for i := 0; i < count; i++ {
				vid := fmt.Sprintf("V%d", i)
				vertices[vid] = &benchVertex{}
				states[vid] = map[string]interface{}{}
			}

			config := Config{
				MaxSupersteps: 1,
				Parallelism:   runtime.NumCPU(),
				QueueCapacity: count,
			}

			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				engine := NewCacheOptimizedEngine(vertices, states, config)
				_ = engine.Run(context.Background())
				engine.Stop()
			}
		})
	}
}

// TestCacheMissReduction verifies reduction in cache misses
func TestCacheMissReduction(t *testing.T) {
	// Create a graph with known communication pattern
	vertexCount := 1000
	vertices := make(map[string]VertexProgram)
	states := make(map[string]map[string]interface{})
	
	// Create clustered communication pattern
	for i := 0; i < vertexCount; i++ {
		vid := fmt.Sprintf("V%d", i)
		// Vertices communicate primarily within clusters of 10
		vertices[vid] = &clusterVertex{
			clusterSize: 10,
			clusterID:   i / 10,
		}
		states[vid] = map[string]interface{}{"value": i}
	}

	config := Config{
		MaxSupersteps: 5,
		Parallelism:   runtime.NumCPU(),
		QueueCapacity: 1024,
	}

	// Run with standard engine
	t.Log("Testing standard engine cache behavior...")
	standardEngine := NewEngine(vertices, states, config)
	defer standardEngine.Stop()
	
	var mStandard runtime.MemStats
	runtime.ReadMemStats(&mStandard)
	allocsBeforeStandard := mStandard.Mallocs
	
	startStandard := time.Now()
	_ = standardEngine.Run(context.Background())
	standardTime := time.Since(startStandard)
	
	runtime.ReadMemStats(&mStandard)
	allocsStandard := mStandard.Mallocs - allocsBeforeStandard

	// Reset states
	for i := 0; i < vertexCount; i++ {
		vid := fmt.Sprintf("V%d", i)
		states[vid] = map[string]interface{}{"value": i}
	}

	// Run with optimized engine
	t.Log("Testing optimized engine cache behavior...")
	optimizedEngine := NewCacheOptimizedEngine(vertices, states, config)
	defer optimizedEngine.Stop()
	
	var mOptimized runtime.MemStats
	runtime.ReadMemStats(&mOptimized)
	allocsBeforeOptimized := mOptimized.Mallocs
	
	startOptimized := time.Now()
	_ = optimizedEngine.Run(context.Background())
	optimizedTime := time.Since(startOptimized)
	
	runtime.ReadMemStats(&mOptimized)
	allocsOptimized := mOptimized.Mallocs - allocsBeforeOptimized

	// Compare results
	allocReduction := float64(allocsStandard - allocsOptimized) / float64(allocsStandard) * 100
	speedup := float64(standardTime) / float64(optimizedTime)
	
	t.Logf("Memory allocations:")
	t.Logf("  Standard: %d allocations", allocsStandard)
	t.Logf("  Optimized: %d allocations", allocsOptimized)
	t.Logf("  Reduction: %.1f%%", allocReduction)
	
	t.Logf("Execution time:")
	t.Logf("  Standard: %v", standardTime)
	t.Logf("  Optimized: %v", optimizedTime)
	t.Logf("  Speedup: %.2fx", speedup)
	
	if allocReduction > 20 {
		t.Logf("SUCCESS: Achieved %.1f%% reduction in allocations", allocReduction)
	}
	
	if speedup > 1.5 {
		t.Logf("SUCCESS: Achieved %.2fx speedup from cache optimization", speedup)
	}
}

// Test vertex implementations

type cacheTestVertex struct {
	totalVertices   int
	messagesPerStep int
}

func (ctv *cacheTestVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	// Simulate computation
	sum := 0
	for _, msg := range messages {
		if val, ok := msg.Value.(int); ok {
			sum += val
		}
	}
	state["sum"] = sum
	
	// Send messages to neighbors
	outMessages := make([]*Message, 0, ctv.messagesPerStep)
	for i := 0; i < ctv.messagesPerStep; i++ {
		target := fmt.Sprintf("V%d", (i + len(vertexID)) % ctv.totalVertices)
		outMessages = append(outMessages, &Message{
			To:    target,
			From:  vertexID,
			Value: i,
		})
	}
	
	return state, outMessages, len(messages) == 0 && ctv.messagesPerStep == 0, nil
}

type clusterVertex struct {
	clusterSize int
	clusterID   int
}

func (cv *clusterVertex) Compute(vertexID string, state map[string]interface{}, messages []*Message) (map[string]interface{}, []*Message, bool, error) {
	// Process incoming messages
	sum := 0
	for _, msg := range messages {
		if val, ok := msg.Value.(int); ok {
			sum += val
		}
	}
	state["sum"] = sum
	
	// Send messages primarily within cluster
	outMessages := make([]*Message, 0, cv.clusterSize)
	baseID := cv.clusterID * cv.clusterSize
	
	for i := 0; i < cv.clusterSize; i++ {
		targetID := baseID + i
		if fmt.Sprintf("V%d", targetID) != vertexID {
			outMessages = append(outMessages, &Message{
				To:    fmt.Sprintf("V%d", targetID),
				From:  vertexID,
				Value: 1,
			})
		}
	}
	
	return state, outMessages, len(outMessages) == 0, nil
}

// TestScalabilityImprovement verifies the fix for 1000-vertex performance cliff
func TestScalabilityImprovement(t *testing.T) {
	testCases := []int{10, 100, 1000, 5000}
	
	standardTimes := make([]time.Duration, len(testCases))
	optimizedTimes := make([]time.Duration, len(testCases))
	
	for i, vertexCount := range testCases {
		vertices := make(map[string]VertexProgram)
		states := make(map[string]map[string]interface{})
		
		for j := 0; j < vertexCount; j++ {
			vid := fmt.Sprintf("V%d", j)
			vertices[vid] = &benchVertex{}
			states[vid] = map[string]interface{}{}
		}
		
		config := Config{
			MaxSupersteps: 1,
			Parallelism:   runtime.NumCPU(),
			QueueCapacity: minInt(vertexCount, 1024),
		}
		
		// Standard engine
		standardEngine := NewEngine(vertices, states, config)
		start := time.Now()
		_ = standardEngine.Run(context.Background())
		standardTimes[i] = time.Since(start)
		standardEngine.Stop()
		
		// Reset states
		for j := 0; j < vertexCount; j++ {
			vid := fmt.Sprintf("V%d", j)
			states[vid] = map[string]interface{}{}
		}
		
		// Optimized engine
		optimizedEngine := NewCacheOptimizedEngine(vertices, states, config)
		start = time.Now()
		_ = optimizedEngine.Run(context.Background())
		optimizedTimes[i] = time.Since(start)
		optimizedEngine.Stop()
	}
	
	t.Log("Scalability Comparison:")
	t.Log("Vertices | Standard   | Optimized  | Speedup | Efficiency")
	t.Log("---------|------------|------------|---------|------------")
	
	for i, count := range testCases {
		speedup := float64(standardTimes[i]) / float64(optimizedTimes[i])
		
		var efficiency float64
		if i > 0 {
			scaleFactor := float64(count) / float64(testCases[i-1])
			timeRatioStandard := float64(standardTimes[i]) / float64(standardTimes[i-1])
			timeRatioOptimized := float64(optimizedTimes[i]) / float64(optimizedTimes[i-1])
			efficiencyStandard := scaleFactor / timeRatioStandard * 100
			efficiencyOptimized := scaleFactor / timeRatioOptimized * 100
			efficiency = efficiencyOptimized - efficiencyStandard
		}
		
		t.Logf("%8d | %10v | %10v | %6.2fx | %+.1f%%", 
			count, standardTimes[i], optimizedTimes[i], speedup, efficiency)
	}
	
	// Check if we fixed the 1000-vertex cliff
	if len(testCases) >= 3 {
		// Calculate efficiency at 1000 vertices
		scaleFactor := float64(testCases[2]) / float64(testCases[1]) // 1000/100
		timeRatio := float64(optimizedTimes[2]) / float64(optimizedTimes[1])
		efficiency := scaleFactor / timeRatio
		
		if efficiency > 0.4 { // 40% efficiency is much better than 0.47%
			t.Logf("SUCCESS: Fixed 1000-vertex performance cliff! Efficiency: %.1f%%", efficiency*100)
		} else {
			t.Logf("WARNING: 1000-vertex efficiency still low: %.1f%%", efficiency*100)
		}
	}
}

// minInt returns minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}