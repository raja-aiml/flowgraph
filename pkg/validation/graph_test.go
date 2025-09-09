package validation

import (
    "testing"

    coregraph "github.com/flowgraph/flowgraph/internal/core/graph"
    "github.com/stretchr/testify/assert"
)

func TestValidateCoreGraph_Endpoints(t *testing.T) {
    g := &coregraph.Graph{
        ID:         "g1",
        Name:       "test",
        EntryPoint: "n1",
        Nodes: map[string]*coregraph.Node{
            "n1": {ID: "n1", Name: "Node 1", Type: coregraph.NodeTypeFunction},
        },
        Edges: []*coregraph.Edge{
            {Source: "n1", Target: "missing"},
        },
    }
    err := ValidateCoreGraph(g)
    assert.Error(t, err)
}

func TestValidateCoreGraph_DuplicateEdges(t *testing.T) {
    g := &coregraph.Graph{
        ID:         "g1",
        Name:       "test",
        EntryPoint: "n1",
        Nodes: map[string]*coregraph.Node{
            "n1": {ID: "n1", Name: "Node 1", Type: coregraph.NodeTypeFunction},
            "n2": {ID: "n2", Name: "Node 2", Type: coregraph.NodeTypeFunction},
        },
        Edges: []*coregraph.Edge{
            {Source: "n1", Target: "n2", Type: coregraph.EdgeTypeDefault},
            {Source: "n1", Target: "n2", Type: coregraph.EdgeTypeDefault},
        },
    }
    err := ValidateCoreGraph(g)
    assert.Error(t, err)
}

func TestValidateCoreGraph_CycleDetection(t *testing.T) {
    g := &coregraph.Graph{
        ID:         "g1",
        Name:       "test",
        EntryPoint: "n1",
        Nodes: map[string]*coregraph.Node{
            "n1": {ID: "n1", Name: "Node 1", Type: coregraph.NodeTypeFunction},
            "n2": {ID: "n2", Name: "Node 2", Type: coregraph.NodeTypeFunction},
        },
        Edges: []*coregraph.Edge{
            {Source: "n1", Target: "n2"},
            {Source: "n2", Target: "n1"},
        },
    }
    // Default does not check cycles
    assert.NoError(t, ValidateCoreGraph(g))
    // Enabling cycle check should error
    err := ValidateCoreGraph(g, GraphValidationOptions{CheckCycles: true})
    assert.Error(t, err)
}

