// Package graph provides node definitions
package graph

import "time"

// NodeType represents the type of node
type NodeType string

const (
	// NodeTypeFunction represents a function node
	NodeTypeFunction NodeType = "function"
	// NodeTypeTool represents a tool node
	NodeTypeTool NodeType = "tool"
	// NodeTypeAgent represents an agent node
	NodeTypeAgent NodeType = "agent"
	// NodeTypeConditional represents a conditional routing node
	NodeTypeConditional NodeType = "conditional"
	// NodeTypeSubgraph represents a subgraph node
	NodeTypeSubgraph NodeType = "subgraph"
)

// Node represents a vertex in the graph
// PRINCIPLES:
// - KISS: Simple node representation
// - SRP: Only responsible for node data
type Node struct {
	ID          string                 `json:"id"`
	Type        NodeType               `json:"type"`
	Name        string                 `json:"name"`
	Function    string                 `json:"function,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Retries     int                    `json:"retries,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
	Conditional *ConditionalBranch     `json:"conditional,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// ConditionalBranch defines conditional routing logic
type ConditionalBranch struct {
	Conditions map[string]string `json:"conditions"` // condition_name -> target_node_id
	Default    string            `json:"default"`    // default target if no conditions match
}

// Validate ensures node integrity
// PRINCIPLES:
// - SRP: Single responsibility - validation only
// - KISS: Simple validation, <10 lines
func (n *Node) Validate() error {
	if n.ID == "" {
		return ErrInvalidNodeID
	}
	if n.Name == "" {
		return ErrInvalidNodeName
	}
	if n.Type == "" {
		return ErrInvalidNodeType
	}
	if n.Type == NodeTypeConditional && n.Conditional == nil {
		return ErrMissingConditional
	}
	return nil
}

// IsConditional checks if node is conditional
func (n *Node) IsConditional() bool {
	return n.Type == NodeTypeConditional
}
