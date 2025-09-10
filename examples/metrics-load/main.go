package main

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"os"
)

// Simulate a batch data processing pipeline with checkpointing and recovery
func main() {
	fmt.Println("🗃️ Batch Data Processing Pipeline (FlowGraph Real-World Example)")
	fmt.Println("============================================================")

	data := make([]int, 0, 100)
	checkpointFile := "/tmp/flowgraph-batch-checkpoint"

	// 1. Load checkpoint if exists
	if _, err := os.Stat(checkpointFile); err == nil {
		f, err := os.Open(checkpointFile)
		if err == nil {
			defer f.Close()
			dec := gob.NewDecoder(f)
			if err := dec.Decode(&data); err == nil {
				fmt.Printf("✅ Restored data from checkpoint (%d items)\n", len(data))
			}
		}
	}

	// 2. Simulate batch processing with periodic checkpointing
	for i := len(data); i < cap(data); i++ {
		data = append(data, rand.Intn(1000))
		if i%20 == 0 && i > 0 {
			f, err := os.Create(checkpointFile)
			if err == nil {
				enc := gob.NewEncoder(f)
				if err := enc.Encode(data); err == nil {
					fmt.Printf("💾 Checkpoint saved at item %d\n", i)
				}
				f.Close()
			}
		}
		// Simulate crash/recovery
		if i == 60 {
			fmt.Println("⚡ Simulating crash! (Demo only, continuing instead of exiting)")
			// os.Exit(1) // Commented out for demo
		}
	}

	fmt.Printf("✅ Batch processing complete (%d items processed)\n", len(data))

	// 3. Clean up checkpoint
	_ = os.Remove(checkpointFile)
	fmt.Println("🧹 Checkpoint file cleaned up")
}
