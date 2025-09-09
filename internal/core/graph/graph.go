// Package graph provides the core graph domain entities
// following Clean Architecture principles with zero external dependencies.
package graph

import (
	"time"
)

// Graph represents the core graph entity
// PRINCIPLES:
// - KISS: Simple struct, no complex hierarchies
// - SRP: Only responsible for graph structure, not execution
// - YAGNI: No unused fields or methods
type Graph struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Nodes      map[string]*Node       `json:"nodes"`
	Edges      []*Edge                `json:"edges"`
	EntryPoint string                 `json:"entry_point"`
	State      map[string]interface{} `json:"state,omitempty"`
	Config     GraphConfig            `json:"config"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// GraphConfig holds graph configuration
type GraphConfig struct {
	Interrupts    []string               `json:"interrupts,omitempty"`
	MaxIterations int                    `json:"max_iterations,omitempty"`
	Timeout       time.Duration          `json:"timeout,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// Validate ensures graph integrity
// PRINCIPLES:
// - SRP: Single responsibility - validation only
// - KISS: Simple validation rules, easy to understand
// - Max 10 lines (complexity rule)
func (g *Graph) Validate() error {
	if g.Name == "" {
		return ErrInvalidGraphName
	}
	if g.EntryPoint == "" {
		return ErrNoEntryPoint
	}
	if _, exists := g.Nodes[g.EntryPoint]; !exists {
		return ErrInvalidEntryPoint
	}
	return nil
}

// AddNode adds a node to the graph
// PRINCIPLES:
// - KISS: Direct and simple implementation
// - SRP: Only adds node, doesn't validate graph
// - No nesting beyond 2 levels
func (g *Graph) AddNode(node *Node) error {
    if node == nil {
        return ErrNilNode
    }
    if err := node.Validate(); err != nil {
        return err
    }
    if g.Nodes == nil {
        g.Nodes = make(map[string]*Node)
    }
    // Prevent duplicate node IDs
    if _, exists := g.Nodes[node.ID]; exists {
        return ErrDuplicateNode
    }
    g.Nodes[node.ID] = node
    g.UpdatedAt = time.Now()
    return nil
}

// AddEdge adds an edge to the graph
func (g *Graph) AddEdge(edge *Edge) error {
    if edge == nil {
        return ErrNilEdge
    }
    if err := edge.Validate(); err != nil {
        return err
    }
    // Verify source and target nodes exist
    if _, exists := g.Nodes[edge.Source]; !exists {
        return ErrSourceNodeNotFound
    }
    if _, exists := g.Nodes[edge.Target]; !exists {
        return ErrTargetNodeNotFound
    }
    // Prevent duplicate edges (same source, target, type, and condition)
    for _, e := range g.Edges {
        if e.Source == edge.Source && e.Target == edge.Target && e.Type == edge.Type && e.Condition == edge.Condition {
            return ErrDuplicateEdge
        }
    }
    g.Edges = append(g.Edges, edge)
    g.UpdatedAt = time.Now()
    return nil
}
