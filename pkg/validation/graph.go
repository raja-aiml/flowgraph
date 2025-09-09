package validation

import (
    "fmt"

    coregraph "github.com/flowgraph/flowgraph/internal/core/graph"
)

// GraphValidationOptions controls optional validation checks.
type GraphValidationOptions struct {
    // CheckCycles enables detection of directed cycles.
    CheckCycles bool
}

// ValidateCoreGraph performs structural validation on the core graph entity.
// It is intended for graphs loaded from external sources where in-method guards
// (e.g., AddNode/AddEdge) may have been bypassed.
func ValidateCoreGraph(g *coregraph.Graph, opts ...GraphValidationOptions) error {
    if g == nil {
        return fmt.Errorf("graph is nil")
    }

    // Basic graph fields
    if err := g.Validate(); err != nil {
        return err
    }

    // Validate all nodes
    for _, n := range g.Nodes {
        if n == nil {
            return fmt.Errorf("nil node encountered")
        }
        if err := n.Validate(); err != nil {
            return err
        }
    }

    // Track seen edges to detect duplicates
    type edgeKey struct{ s, t, ty, cond string }
    seenEdges := make(map[edgeKey]struct{})

    // Validate edges and endpoints
    for _, e := range g.Edges {
        if e == nil {
            return fmt.Errorf("nil edge encountered")
        }
        if err := e.Validate(); err != nil {
            return err
        }
        if _, ok := g.Nodes[e.Source]; !ok {
            return coregraph.ErrSourceNodeNotFound
        }
        if _, ok := g.Nodes[e.Target]; !ok {
            return coregraph.ErrTargetNodeNotFound
        }
        k := edgeKey{e.Source, e.Target, string(e.Type), e.Condition}
        if _, dup := seenEdges[k]; dup {
            return coregraph.ErrDuplicateEdge
        }
        seenEdges[k] = struct{}{}
    }

    // Optional cycle detection
    var cfg GraphValidationOptions
    if len(opts) > 0 {
        cfg = opts[0]
    }
    if cfg.CheckCycles {
        if hasCycle(g) {
            return coregraph.ErrCyclicGraph
        }
    }

    return nil
}

// hasCycle detects any cycle in a directed graph using DFS with coloring.
func hasCycle(g *coregraph.Graph) bool {
    const (
        white = 0 // unvisited
        gray  = 1 // visiting
        black = 2 // visited
    )
    color := make(map[string]int, len(g.Nodes))
    adj := make(map[string][]string, len(g.Nodes))
    for _, e := range g.Edges {
        adj[e.Source] = append(adj[e.Source], e.Target)
    }
    var dfs func(string) bool
    dfs = func(u string) bool {
        color[u] = gray
        for _, v := range adj[u] {
            if color[v] == gray {
                return true // back-edge
            }
            if color[v] == white {
                if dfs(v) {
                    return true
                }
            }
        }
        color[u] = black
        return false
    }
    for id := range g.Nodes {
        if color[id] == white {
            if dfs(id) {
                return true
            }
        }
    }
    return false
}

