// Package validation provides model definitions with validation tags
package validation

import (
	"time"
)

// NodeConfig represents a graph node configuration with validation
// PRINCIPLES:
// - Single Responsibility: Node configuration only
// - Validation: Comprehensive validation tags
type NodeConfig struct {
	ID        string                 `json:"id" validate:"required,node_id" yaml:"id"`
	Name      string                 `json:"name" validate:"required,min=1,max=100" yaml:"name"`
	Type      string                 `json:"type" validate:"required,oneof=function tool condition" yaml:"type"`
	Enabled   bool                   `json:"enabled" yaml:"enabled"`
	Timeout   *time.Duration         `json:"timeout,omitempty" validate:"omitempty,min=1s,max=300s" yaml:"timeout,omitempty"`
	Retries   int                    `json:"retries" validate:"min=0,max=10" yaml:"retries"`
	Config    map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Tags      []string               `json:"tags,omitempty" validate:"dive,alphanum" yaml:"tags,omitempty"`
	CreatedAt time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" yaml:"updated_at"`
}

// Validate implements custom validation for NodeConfig
func (nc *NodeConfig) Validate() error {
	// Custom business logic validation
	if nc.Type == "condition" && nc.Timeout != nil && *nc.Timeout > 60*time.Second {
		return ValidationErrors{{
			Field:   "timeout",
			Value:   nc.Timeout,
			Message: "condition nodes cannot have timeout > 60s",
		}}
	}

	return nil
}

// EdgeConfig represents a graph edge configuration with validation
type EdgeConfig struct {
	ID        string                 `json:"id" validate:"required,edge_id" yaml:"id"`
	Source    string                 `json:"source" validate:"required,node_id" yaml:"source"`
	Target    string                 `json:"target" validate:"required,node_id" yaml:"target"`
	Condition *string                `json:"condition,omitempty" validate:"omitempty,min=1" yaml:"condition,omitempty"`
	Weight    *float64               `json:"weight,omitempty" validate:"omitempty,min=0,max=1" yaml:"weight,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at" yaml:"created_at"`
}

// GraphConfig represents a complete graph configuration
type GraphConfig struct {
	ID          string              `json:"id" validate:"required,uuid4" yaml:"id"`
	Name        string              `json:"name" validate:"required,min=1,max=200" yaml:"name"`
	Description *string             `json:"description,omitempty" validate:"omitempty,max=1000" yaml:"description,omitempty"`
	Version     string              `json:"version" validate:"required,semver" yaml:"version"`
	State       string              `json:"state" validate:"required,graph_state" yaml:"state"`
	Nodes       []NodeConfig        `json:"nodes" validate:"required,min=1,dive,required" yaml:"nodes"`
	Edges       []EdgeConfig        `json:"edges" validate:"dive,required" yaml:"edges"`
	Config      *GraphRuntimeConfig `json:"config,omitempty" validate:"omitempty" yaml:"config,omitempty"`
	CreatedAt   time.Time           `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" yaml:"updated_at"`
}

// Validate implements custom validation for GraphConfig
func (gc *GraphConfig) Validate() error {
	var errors ValidationErrors

	// Validate node ID uniqueness
	nodeIDs := make(map[string]bool)
	for _, node := range gc.Nodes {
		if nodeIDs[node.ID] {
			errors = append(errors, ValidationError{
				Field:   "nodes",
				Value:   node.ID,
				Message: "duplicate node ID",
			})
		}
		nodeIDs[node.ID] = true
	}

	// Validate edge references
	for _, edge := range gc.Edges {
		if !nodeIDs[edge.Source] {
			errors = append(errors, ValidationError{
				Field:   "edges.source",
				Value:   edge.Source,
				Message: "source node does not exist",
			})
		}
		if !nodeIDs[edge.Target] {
			errors = append(errors, ValidationError{
				Field:   "edges.target",
				Value:   edge.Target,
				Message: "target node does not exist",
			})
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// GraphRuntimeConfig represents runtime configuration for graph execution
type GraphRuntimeConfig struct {
	MaxConcurrency int                `json:"max_concurrency" validate:"min=1,max=100" yaml:"max_concurrency"`
	Timeout        time.Duration      `json:"timeout" validate:"min=1s,max=3600s" yaml:"timeout"`
	RetryPolicy    *RetryPolicyConfig `json:"retry_policy,omitempty" yaml:"retry_policy,omitempty"`
	CheckpointMode string             `json:"checkpoint_mode" validate:"oneof=none manual automatic" yaml:"checkpoint_mode"`
	ChannelConfig  *ChannelConfig     `json:"channel_config,omitempty" yaml:"channel_config,omitempty"`
}

// RetryPolicyConfig represents retry configuration
type RetryPolicyConfig struct {
	MaxRetries    int           `json:"max_retries" validate:"min=0,max=10" yaml:"max_retries"`
	BackoffFactor float64       `json:"backoff_factor" validate:"min=1,max=10" yaml:"backoff_factor"`
	BaseDelay     time.Duration `json:"base_delay" validate:"min=100ms,max=60s" yaml:"base_delay"`
	MaxDelay      time.Duration `json:"max_delay" validate:"min=1s,max=300s" yaml:"max_delay"`
}

// ChannelConfig represents channel configuration
type ChannelConfig struct {
	Type        string  `json:"type" validate:"required,oneof=inmemory buffered persistent" yaml:"type"`
	BufferSize  *int    `json:"buffer_size,omitempty" validate:"omitempty,min=1,max=10000" yaml:"buffer_size,omitempty"`
	MaxSizeMB   *int64  `json:"max_size_mb,omitempty" validate:"omitempty,min=1,max=1000" yaml:"max_size_mb,omitempty"`
	PersistPath *string `json:"persist_path,omitempty" validate:"omitempty,min=1" yaml:"persist_path,omitempty"`
}

// MessageConfig represents message configuration with validation
type MessageConfig struct {
	ID       string                 `json:"id" validate:"required,uuid4" yaml:"id"`
	Type     string                 `json:"type" validate:"required,message_type" yaml:"type"`
	Source   string                 `json:"source" validate:"required,node_id" yaml:"source"`
	Target   string                 `json:"target" validate:"required,node_id" yaml:"target"`
	Payload  interface{}            `json:"payload" yaml:"payload"`
	Headers  map[string]string      `json:"headers,omitempty" yaml:"headers,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	TTL      *time.Duration         `json:"ttl,omitempty" validate:"omitempty,min=1s,max=86400s" yaml:"ttl,omitempty"`
}

// CheckpointConfig represents checkpoint configuration
type CheckpointConfig struct {
	ID        string                 `json:"id" validate:"required,uuid4" yaml:"id"`
	GraphID   string                 `json:"graph_id" validate:"required,uuid4" yaml:"graph_id"`
	State     map[string]interface{} `json:"state" validate:"required" yaml:"state"`
	Version   int64                  `json:"version" validate:"min=1" yaml:"version"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at" yaml:"created_at"`
	ExpiresAt *time.Time             `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
}

// APIRequest represents a generic API request with validation
type APIRequest struct {
	RequestID string            `json:"request_id" validate:"required,uuid4" yaml:"request_id"`
	Method    string            `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH" yaml:"method"`
	Path      string            `json:"path" validate:"required,uri" yaml:"path"`
	Headers   map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body      interface{}       `json:"body,omitempty" yaml:"body,omitempty"`
	Timestamp time.Time         `json:"timestamp" yaml:"timestamp"`
}

// APIResponse represents a generic API response with validation
type APIResponse struct {
	RequestID  string            `json:"request_id" validate:"required,uuid4" yaml:"request_id"`
	StatusCode int               `json:"status_code" validate:"min=100,max=599" yaml:"status_code"`
	Headers    map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Body       interface{}       `json:"body,omitempty" yaml:"body,omitempty"`
	Errors     []ValidationError `json:"errors,omitempty" yaml:"errors,omitempty"`
	Timestamp  time.Time         `json:"timestamp" yaml:"timestamp"`
}
