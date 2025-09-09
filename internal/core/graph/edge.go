// Package graph provides edge definitions
package graph

// EdgeType represents the type of edge
type EdgeType string

const (
	// EdgeTypeDefault represents a default edge
	EdgeTypeDefault EdgeType = "default"
	// EdgeTypeConditional represents a conditional edge
	EdgeTypeConditional EdgeType = "conditional"
	// EdgeTypeError represents an error handling edge
	EdgeTypeError EdgeType = "error"
)

// Edge represents a connection between nodes
// PRINCIPLES:
// - KISS: Simple edge representation
// - SRP: Only responsible for edge data
type Edge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"` // Source node ID
	Target     string                 `json:"target"` // Target node ID
	Type       EdgeType               `json:"type"`
	Condition  string                 `json:"condition,omitempty"`  // Condition for conditional edges
	Weight     float64                `json:"weight,omitempty"`     // Edge weight for algorithms
	Properties map[string]interface{} `json:"properties,omitempty"` // Additional properties
}

// Validate ensures edge integrity
// PRINCIPLES:
// - SRP: Single responsibility - validation only
// - KISS: Simple validation, <10 lines
func (e *Edge) Validate() error {
	if e.Source == "" {
		return ErrInvalidSource
	}
	if e.Target == "" {
		return ErrInvalidTarget
	}
	if e.Source == e.Target {
		return ErrSelfLoop
	}
	if e.Type == "" {
		e.Type = EdgeTypeDefault
	}
	return nil
}

// IsConditional checks if edge is conditional
func (e *Edge) IsConditional() bool {
	return e.Type == EdgeTypeConditional
}
