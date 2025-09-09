// Package graph defines domain-specific errors
package graph

import "errors"

// Domain errors - DRY principle: defined once, used everywhere
var (
	// Graph errors
	ErrInvalidGraphName  = errors.New("invalid graph name")
	ErrNoEntryPoint      = errors.New("no entry point specified")
	ErrInvalidEntryPoint = errors.New("entry point node not found")
	ErrGraphNotFound     = errors.New("graph not found")
	ErrCyclicGraph       = errors.New("cyclic dependency detected")

	// Node errors
	ErrNilNode            = errors.New("node cannot be nil")
	ErrInvalidNodeID      = errors.New("invalid node ID")
	ErrInvalidNodeName    = errors.New("invalid node name")
	ErrInvalidNodeType    = errors.New("invalid node type")
	ErrNodeNotFound       = errors.New("node not found")
	ErrDuplicateNode      = errors.New("duplicate node ID")
	ErrMissingConditional = errors.New("conditional node missing conditions")

	// Edge errors
	ErrNilEdge            = errors.New("edge cannot be nil")
	ErrInvalidSource      = errors.New("invalid source node")
	ErrInvalidTarget      = errors.New("invalid target node")
	ErrSourceNodeNotFound = errors.New("source node not found")
	ErrTargetNodeNotFound = errors.New("target node not found")
	ErrDuplicateEdge      = errors.New("duplicate edge")
	ErrSelfLoop           = errors.New("self-loops are not allowed")
)
