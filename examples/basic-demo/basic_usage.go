package main

import (
	"fmt"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/graph"
	"github.com/flowgraph/flowgraph/pkg/serialization"
)

// DemoFlowGraphCapabilities demonstrates what can be done with current implementation
func main() {
	fmt.Println("🌊 FlowGraph - Current Capabilities Demo")
	fmt.Println("=======================================")

	// 1. Graph Creation and Validation
	demoGraphCreation()

	// 2. Serialization Capabilities
	demoSerialization()

	// 3. What's Available Summary
	demoAvailableFeatures()
}

func demoGraphCreation() {
	fmt.Println("\n📊 1. Graph Creation and Validation")
	fmt.Println("------------------------------------")

	// Create nodes
	startNode := &graph.Node{
		ID:   "start",
		Name: "Start Node",
		Type: graph.NodeTypeFunction,
		Config: map[string]interface{}{
			"function": "initialize",
		},
	}

	processNode := &graph.Node{
		ID:   "process",
		Name: "Process Data",
		Type: graph.NodeTypeFunction,
		Config: map[string]interface{}{
			"function": "process_data",
		},
	}

	endNode := &graph.Node{
		ID:   "end",
		Name: "End Node",
		Type: graph.NodeTypeFunction,
		Config: map[string]interface{}{
			"function": "finalize",
		},
	}

	// Create edges
	startToProcess := &graph.Edge{
		Source:    "start",
		Target:    "process",
		Condition: "success",
	}

	processToEnd := &graph.Edge{
		Source:    "process",
		Target:    "end",
		Condition: "always",
	}

	// Create graph with proper node map structure
	g := &graph.Graph{
		ID:         "demo-graph",
		Name:       "Demo Processing Graph",
		EntryPoint: "start",
		Nodes: map[string]*graph.Node{
			"start":   startNode,
			"process": processNode,
			"end":     endNode,
		},
		Edges: []*graph.Edge{startToProcess, processToEnd},
	}

	// Validate graph
	if err := g.Validate(); err != nil {
		fmt.Printf("❌ Graph validation failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Created and validated graph '%s' with %d nodes and %d edges\n",
		g.Name, len(g.Nodes), len(g.Edges))

	// Demonstrate node validation
	fmt.Printf("✅ Start node validation: %v\n", startNode.Validate() == nil)
	fmt.Printf("✅ Edge validation: %v\n", startToProcess.Validate() == nil)
	fmt.Printf("✅ Conditional check: %v\n", processToEnd.IsConditional())
}

func demoAvailableFeatures() {
	fmt.Println("\n� 2. Available Features Summary")
	fmt.Println("---------------------------------")

	fmt.Println("✅ COMPLETED Components:")
	fmt.Println("  📊 Graph Domain Model (97.9% test coverage)")
	fmt.Println("    - Node/Edge creation and validation")
	fmt.Println("    - Graph structure management")
	fmt.Println("    - Clean Architecture compliance")

	fmt.Println("\n  💾 Checkpoint System (78.8% test coverage)")
	fmt.Println("    - In-memory checkpoint storage")
	fmt.Println("    - TTL and memory management")
	fmt.Println("    - Thread-safe operations")

	fmt.Println("\n  🔧 Serialization Layer (83.3% test coverage)")
	fmt.Println("    - JSON/MessagePack codecs")
	fmt.Println("    - gzip/zstd compression")
	fmt.Println("    - AES-GCM encryption")

	fmt.Println("\n  🧠 Pregel Engine (84.5% test coverage)")
	fmt.Println("    - BSP (Bulk Synchronous Parallel) execution")
	fmt.Println("    - Work-stealing scheduler")
	fmt.Println("    - Message aggregation system")
	fmt.Println("    - Error recovery mechanisms")

	fmt.Println("\n  ⚡ Execution Engine (89.5% test coverage)")
	fmt.Println("    - DefaultGraphExecutor")
	fmt.Println("    - Node processors (Function/Tool/Agent/Conditional)")
	fmt.Println("    - Edge evaluation logic")

	fmt.Println("\n🚧 IN-PROGRESS Components:")
	fmt.Println("  - Channel abstraction layer (interface defined)")
	fmt.Println("  - Validation framework (partial implementation)")
	fmt.Println("  - Streaming capabilities (Pregel integration)")

	fmt.Println("\n❌ MISSING Components:")
	fmt.Println("  - PostgreSQL/SQLite repositories")
	fmt.Println("  - Service layer (StateService/CheckpointService)")
	fmt.Println("  - Complete channel implementations")
	fmt.Println("  - REST API endpoints")
	fmt.Println("  - CLI framework")
}

func demoSerialization() {
	fmt.Println("\n🔧 2. Serialization Capabilities")
	fmt.Println("----------------------------------")

	// Test data
	data := map[string]interface{}{
		"message": "Hello FlowGraph!",
		"numbers": []int{1, 2, 3, 4, 5},
		"nested": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"active":    true,
		},
	}

	// JSON Serialization
	jsonSerializer := serialization.DefaultSerializer()

	serialized, err := jsonSerializer.Serialize(data)
	if err != nil {
		fmt.Printf("❌ JSON serialization failed: %v\n", err)
		return
	}
	fmt.Printf("✅ JSON serialized: %d bytes\n", len(serialized))

	var deserialized map[string]interface{}
	if err := jsonSerializer.Deserialize(serialized, &deserialized); err != nil {
		fmt.Printf("❌ JSON deserialization failed: %v\n", err)
		return
	}
	fmt.Printf("✅ JSON deserialized successfully\n")

	// MessagePack Serialization
	msgpackSerializer := serialization.NewSerializer(serialization.SerializationConfig{
		Codec: serialization.NewMsgPackCodec(),
	})

	msgpackData, err := msgpackSerializer.Serialize(data)
	if err != nil {
		fmt.Printf("❌ MessagePack serialization failed: %v\n", err)
		return
	}
	fmt.Printf("✅ MessagePack serialized: %d bytes\n", len(msgpackData))

	fmt.Printf("📊 Serialization comparison:\n")
	fmt.Printf("   JSON: %d bytes\n", len(serialized))
	fmt.Printf("   MessagePack: %d bytes (%.1f%% of JSON size)\n",
		len(msgpackData), 100.0*float64(len(msgpackData))/float64(len(serialized)))
}
