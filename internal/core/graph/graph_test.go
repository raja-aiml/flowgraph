package graph

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraph_Validate(t *testing.T) {
	tests := []struct {
		name    string
		graph   *Graph
		wantErr error
	}{
		{
			name: "valid graph",
			graph: &Graph{
				Name:       "test-graph",
				EntryPoint: "node1",
				Nodes: map[string]*Node{
					"node1": {ID: "node1", Name: "Node 1", Type: NodeTypeFunction},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing name",
			graph: &Graph{
				EntryPoint: "node1",
				Nodes: map[string]*Node{
					"node1": {ID: "node1"},
				},
			},
			wantErr: ErrInvalidGraphName,
		},
		{
			name: "missing entry point",
			graph: &Graph{
				Name: "test-graph",
			},
			wantErr: ErrNoEntryPoint,
		},
		{
			name: "invalid entry point",
			graph: &Graph{
				Name:       "test-graph",
				EntryPoint: "nonexistent",
				Nodes:      map[string]*Node{},
			},
			wantErr: ErrInvalidEntryPoint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.graph.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGraph_AddNode(t *testing.T) {
    g := &Graph{
        Name: "test-graph",
    }

	t.Run("add valid node", func(t *testing.T) {
		node := &Node{
			ID:   "node1",
			Name: "Test Node",
			Type: NodeTypeFunction,
		}
		err := g.AddNode(node)
		require.NoError(t, err)
		assert.NotNil(t, g.Nodes)
		assert.Equal(t, node, g.Nodes["node1"])
	})

	t.Run("add nil node", func(t *testing.T) {
		err := g.AddNode(nil)
		assert.ErrorIs(t, err, ErrNilNode)
	})

    t.Run("add invalid node", func(t *testing.T) {
        node := &Node{
            ID: "", // Invalid
        }
        err := g.AddNode(node)
        assert.Error(t, err)
    })

    t.Run("add duplicate node", func(t *testing.T) {
        node := &Node{
            ID:   "node-dup",
            Name: "Duplicate Node",
            Type: NodeTypeFunction,
        }
        require.NoError(t, g.AddNode(node))
        err := g.AddNode(node)
        assert.ErrorIs(t, err, ErrDuplicateNode)
    })
}

func TestGraph_AddEdge(t *testing.T) {
    g := &Graph{
        Name: "test-graph",
        Nodes: map[string]*Node{
            "node1": {ID: "node1", Name: "Node 1", Type: NodeTypeFunction},
            "node2": {ID: "node2", Name: "Node 2", Type: NodeTypeFunction},
        },
    }

	t.Run("add valid edge", func(t *testing.T) {
		edge := &Edge{
			Source: "node1",
			Target: "node2",
		}
		err := g.AddEdge(edge)
		require.NoError(t, err)
		assert.Len(t, g.Edges, 1)
		assert.Equal(t, edge, g.Edges[0])
		assert.False(t, g.UpdatedAt.IsZero())
	})

	t.Run("add nil edge", func(t *testing.T) {
		err := g.AddEdge(nil)
		assert.ErrorIs(t, err, ErrNilEdge)
	})

	t.Run("add edge with non-existent source", func(t *testing.T) {
		edge := &Edge{
			Source: "nonexistent",
			Target: "node2",
		}
		err := g.AddEdge(edge)
		assert.ErrorIs(t, err, ErrSourceNodeNotFound)
	})

    t.Run("add edge with non-existent target", func(t *testing.T) {
        edge := &Edge{
            Source: "node1",
            Target: "nonexistent",
        }
        err := g.AddEdge(edge)
        assert.ErrorIs(t, err, ErrTargetNodeNotFound)
    })

    t.Run("add duplicate edge", func(t *testing.T) {
        // Use a fresh graph context for this subtest
        g2 := &Graph{
            Name: "test-graph",
            Nodes: map[string]*Node{
                "node1": {ID: "node1", Name: "Node 1", Type: NodeTypeFunction},
                "node2": {ID: "node2", Name: "Node 2", Type: NodeTypeFunction},
            },
        }
        edge := &Edge{
            Source:    "node1",
            Target:    "node2",
            Type:      EdgeTypeDefault,
            Condition: "", // same condition means duplicate
        }
        require.NoError(t, g2.AddEdge(edge))
        err := g2.AddEdge(edge)
        assert.ErrorIs(t, err, ErrDuplicateEdge)
    })
}

func TestNode_Validate(t *testing.T) {
	tests := []struct {
		name    string
		node    *Node
		wantErr error
	}{
		{
			name: "valid node",
			node: &Node{
				ID:   "node1",
				Name: "Test Node",
				Type: NodeTypeFunction,
			},
			wantErr: nil,
		},
		{
			name: "missing ID",
			node: &Node{
				Name: "Test Node",
				Type: NodeTypeFunction,
			},
			wantErr: ErrInvalidNodeID,
		},
		{
			name: "missing name",
			node: &Node{
				ID:   "node1",
				Type: NodeTypeFunction,
			},
			wantErr: ErrInvalidNodeName,
		},
		{
			name: "missing type",
			node: &Node{
				ID:   "node1",
				Name: "Test Node",
			},
			wantErr: ErrInvalidNodeType,
		},
		{
			name: "conditional without conditions",
			node: &Node{
				ID:   "node1",
				Name: "Test Node",
				Type: NodeTypeConditional,
			},
			wantErr: ErrMissingConditional,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEdge_Validate(t *testing.T) {
	tests := []struct {
		name    string
		edge    *Edge
		wantErr error
	}{
		{
			name: "valid edge",
			edge: &Edge{
				Source: "node1",
				Target: "node2",
			},
			wantErr: nil,
		},
		{
			name: "missing source",
			edge: &Edge{
				Target: "node2",
			},
			wantErr: ErrInvalidSource,
		},
		{
			name: "missing target",
			edge: &Edge{
				Source: "node1",
			},
			wantErr: ErrInvalidTarget,
		},
		{
			name: "self loop",
			edge: &Edge{
				Source: "node1",
				Target: "node1",
			},
			wantErr: ErrSelfLoop,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNode_IsConditional(t *testing.T) {
	tests := []struct {
		name string
		node *Node
		want bool
	}{
		{
			name: "conditional node",
			node: &Node{Type: NodeTypeConditional},
			want: true,
		},
		{
			name: "function node",
			node: &Node{Type: NodeTypeFunction},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.node.IsConditional())
		})
	}
}

func TestEdge_IsConditional(t *testing.T) {
	tests := []struct {
		name string
		edge *Edge
		want bool
	}{
		{
			name: "conditional edge",
			edge: &Edge{Type: EdgeTypeConditional},
			want: true,
		},
		{
			name: "default edge",
			edge: &Edge{Type: EdgeTypeDefault},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.edge.IsConditional())
		})
	}
}

func TestGraphConfig(t *testing.T) {
	config := GraphConfig{
		Interrupts:    []string{"node1", "node2"},
		MaxIterations: 100,
		Timeout:       10 * time.Second,
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	assert.Equal(t, 2, len(config.Interrupts))
	assert.Equal(t, 100, config.MaxIterations)
	assert.Equal(t, 10*time.Second, config.Timeout)
	assert.Equal(t, "value", config.Metadata["key"])
}
