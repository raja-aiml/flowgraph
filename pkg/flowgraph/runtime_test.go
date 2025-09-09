package flowgraph

import (
    "context"
    "testing"

    "github.com/flowgraph/flowgraph/internal/app/dto"
    coregraph "github.com/flowgraph/flowgraph/internal/core/graph"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestRuntime_RunSimple(t *testing.T) {
    rt := NewRuntime()
    ctx := context.Background()

    g := &coregraph.Graph{
        ID:         "rt-graph",
        Name:       "Runtime Graph",
        EntryPoint: "start",
        Nodes: map[string]*coregraph.Node{
            "start": {ID: "start", Name: "Start", Type: coregraph.NodeTypeFunction},
        },
        Edges: []*coregraph.Edge{},
    }

    resp, err := rt.RunSimple(ctx, g, "thread-1", map[string]interface{}{"msg": "hi"})
    require.NoError(t, err)
    require.NotNil(t, resp)
    assert.Equal(t, "rt-graph", resp.GraphID)
    assert.Equal(t, dto.ExecutionStatusCompleted, resp.Status)
}

