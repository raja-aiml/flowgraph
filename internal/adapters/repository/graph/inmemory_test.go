package graphrepo

import (
    "context"
    "testing"

    coregraph "github.com/flowgraph/flowgraph/internal/core/graph"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestInMemoryGraphRepository_Get_NotFound(t *testing.T) {
    repo := NewInMemoryGraphRepository()

    g, err := repo.Get(context.Background(), "does-not-exist")
    assert.Nil(t, g)
    assert.ErrorIs(t, err, coregraph.ErrGraphNotFound)
}

func TestInMemoryGraphRepository_SaveAndGet(t *testing.T) {
    repo := NewInMemoryGraphRepository()

    g := &coregraph.Graph{
        ID:         "g1",
        Name:       "test-graph",
        EntryPoint: "n1",
        Nodes: map[string]*coregraph.Node{
            "n1": {ID: "n1", Name: "Node 1", Type: coregraph.NodeTypeFunction},
        },
    }

    // Basic validation succeeds
    require.NoError(t, g.Validate())

    // Save and retrieve
    require.NoError(t, repo.Save(context.Background(), g))

    loaded, err := repo.Get(context.Background(), "g1")
    require.NoError(t, err)
    require.NotNil(t, loaded)
    assert.Equal(t, g, loaded)
}

func TestInMemoryGraphRepository_SaveInvalid(t *testing.T) {
    repo := NewInMemoryGraphRepository()

    // Graph with invalid edge target
    g := &coregraph.Graph{
        ID:         "g-bad",
        Name:       "bad-graph",
        EntryPoint: "n1",
        Nodes: map[string]*coregraph.Node{
            "n1": {ID: "n1", Name: "Node 1", Type: coregraph.NodeTypeFunction},
        },
        Edges: []*coregraph.Edge{{Source: "n1", Target: "missing"}},
    }

    err := repo.Save(context.Background(), g)
    require.Error(t, err)
}
