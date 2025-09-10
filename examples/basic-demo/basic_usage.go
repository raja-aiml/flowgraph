package main

import (
	"fmt"
	"time"

	"github.com/flowgraph/flowgraph/internal/core/graph"
	"github.com/flowgraph/flowgraph/pkg/serialization"
)

// Document approval workflow: Submission → Review → Approval
func main() {
	fmt.Println("📄 Document Approval Workflow (FlowGraph Real-World Example)")
	fmt.Println("=========================================================")

	// 1. Build workflow graph
	workflow := buildApprovalGraph()

	// 2. Validate graph structure
	if err := workflow.Validate(); err != nil {
		fmt.Printf("❌ Workflow graph validation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Workflow graph '%s' validated (%d nodes, %d edges)\n", workflow.Name, len(workflow.Nodes), len(workflow.Edges))

	// 3. Simulate document submission
	doc := map[string]any{
		"id":           "doc-123",
		"title":        "Quarterly Report",
		"author":       "Alice",
		"content":      "Q1 results...",
		"status":       "submitted",
		"submitted_at": time.Now().Format(time.RFC3339),
	}
	fmt.Printf("\n� Document submitted: %v\n", doc)

	// 4. Validate document data (simulate business rule)
	if doc["title"] == "" || doc["author"] == "" {
		fmt.Println("❌ Document validation failed: missing title or author")
		return
	}
	fmt.Println("✅ Document validation passed")

	// 5. Simulate review step
	doc["status"] = "reviewed"
	doc["reviewed_by"] = "Bob"
	doc["reviewed_at"] = time.Now().Format(time.RFC3339)
	fmt.Printf("\n🔎 Document reviewed: %v\n", doc)

	// 6. Simulate approval step
	doc["status"] = "approved"
	doc["approved_by"] = "Carol"
	doc["approved_at"] = time.Now().Format(time.RFC3339)
	fmt.Printf("\n✅ Document approved: %v\n", doc)

	// 7. Serialize document for storage
	serializer := serialization.DefaultSerializer()
	serialized, err := serializer.Serialize(doc)
	if err != nil {
		fmt.Printf("❌ Document serialization failed: %v\n", err)
		return
	}
	fmt.Printf("\n💾 Document serialized (%d bytes)\n", len(serialized))

	// 8. Deserialize and verify
	var restored map[string]interface{}
	if err := serializer.Deserialize(serialized, &restored); err != nil {
		fmt.Printf("❌ Document deserialization failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Document deserialized: %v\n", restored)
}

func buildApprovalGraph() *graph.Graph {
	// Nodes: submission, review, approval
	submission := &graph.Node{
		ID:     "submission",
		Name:   "Document Submission",
		Type:   graph.NodeTypeFunction,
		Config: map[string]interface{}{"function": "submit_document"},
	}
	review := &graph.Node{
		ID:     "review",
		Name:   "Document Review",
		Type:   graph.NodeTypeFunction,
		Config: map[string]interface{}{"function": "review_document"},
	}
	approval := &graph.Node{
		ID:     "approval",
		Name:   "Document Approval",
		Type:   graph.NodeTypeFunction,
		Config: map[string]interface{}{"function": "approve_document"},
	}

	// Edges: submission → review → approval
	subToRev := &graph.Edge{Source: "submission", Target: "review", Condition: "submitted"}
	revToApp := &graph.Edge{Source: "review", Target: "approval", Condition: "reviewed"}

	return &graph.Graph{
		ID:         "doc-approval",
		Name:       "Document Approval Workflow",
		EntryPoint: "submission",
		Nodes: map[string]*graph.Node{
			"submission": submission,
			"review":     review,
			"approval":   approval,
		},
		Edges: []*graph.Edge{subToRev, revToApp},
	}
}
