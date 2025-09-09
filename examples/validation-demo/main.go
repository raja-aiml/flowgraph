// Example demonstrating the complete validation framework
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/flowgraph/flowgraph/pkg/validation"
)

func main() {
	fmt.Println("=== FlowGraph Validation Framework Demo ===")

	// Demo 1: Basic struct validation
	fmt.Println("\n1. Basic Struct Validation:")
	demoBasicValidation()

	// Demo 2: Enhanced validation with custom rules
	fmt.Println("\n2. Enhanced Validation with Custom Rules:")
	demoEnhancedValidation()

	// Demo 3: Model validation
	fmt.Println("\n3. Model Validation:")
	demoModelValidation()

	// Demo 4: HTTP middleware validation
	fmt.Println("\n4. HTTP Middleware Validation:")
	demoHTTPMiddleware()
}

func demoBasicValidation() {
	type User struct {
		Name  string `validate:"required,min=2,max=50"`
		Email string `validate:"required,email"`
		Age   int    `validate:"min=0,max=120"`
	}

	// Valid user
	validUser := User{
		Name:  "John Doe",
		Email: "john@example.com",
		Age:   25,
	}

	if err := validation.ValidateStruct(validUser); err != nil {
		fmt.Printf("❌ Validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Valid user passed validation")
	}

	// Invalid user
	invalidUser := User{
		Name:  "", // Required field missing
		Email: "invalid-email",
		Age:   -5,
	}

	if err := validation.ValidateStruct(invalidUser); err != nil {
		fmt.Printf("❌ Invalid user failed validation (expected): %v\n", err)
	} else {
		fmt.Println("✅ Invalid user unexpectedly passed")
	}
}

func demoEnhancedValidation() {
	type GraphElement struct {
		NodeID      string `validate:"required,node_id"`
		EdgeID      string `validate:"required,edge_id"`
		State       string `validate:"required,graph_state"`
		Channel     string `validate:"required,channel_name"`
		MessageType string `validate:"required,message_type"`
	}

	// Valid element
	validElement := GraphElement{
		NodeID:      "processing_node_1",
		EdgeID:      "input->processing",
		State:       "running",
		Channel:     "data_channel",
		MessageType: "data",
	}

	if err := validation.ValidateWithPlayground(validElement); err != nil {
		fmt.Printf("❌ Enhanced validation failed: %v\n", err)
	} else {
		fmt.Println("✅ Valid graph element passed enhanced validation")
	}

	// Invalid element
	invalidElement := GraphElement{
		NodeID:      "node with spaces!", // Invalid characters
		EdgeID:      "invalid_edge_format",
		State:       "invalid_state",
		Channel:     "INVALID_CHANNEL", // Uppercase not allowed
		MessageType: "invalid_type",
	}

	if err := validation.ValidateWithPlayground(invalidElement); err != nil {
		fmt.Printf("❌ Invalid element failed enhanced validation (expected):\n")
		if validationErrors, ok := err.(validation.ValidationErrors); ok {
			for i, ve := range validationErrors {
				fmt.Printf("   %d. Field '%s': %s\n", i+1, ve.Field, ve.Message)
			}
		}
	} else {
		fmt.Println("✅ Invalid element unexpectedly passed")
	}
}

func demoModelValidation() {
	// Create a sample graph configuration
	timeout := 30 * time.Second

	graphConfig := validation.GraphConfig{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Name:        "Sample Processing Graph",
		Description: stringPtr("A demonstration graph for processing data"),
		Version:     "1.0.0",
		State:       "pending",
		Nodes: []validation.NodeConfig{
			{
				ID:        "input_node",
				Name:      "Input Processor",
				Type:      "function",
				Enabled:   true,
				Timeout:   &timeout,
				Retries:   3,
				Tags:      []string{"input", "processor"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:        "output_node",
				Name:      "Output Processor",
				Type:      "function",
				Enabled:   true,
				Timeout:   &timeout,
				Retries:   2,
				Tags:      []string{"output", "processor"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		Edges: []validation.EdgeConfig{
			{
				ID:        "input_node->output_node",
				Source:    "input_node",
				Target:    "output_node",
				CreatedAt: time.Now(),
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Validate with go-playground/validator
	if err := validation.ValidateWithPlayground(graphConfig); err != nil {
		fmt.Printf("❌ Graph config failed validation: %v\n", err)
	} else {
		fmt.Println("✅ Graph config passed go-playground validation")
	}

	// Validate with custom business rules
	if err := graphConfig.Validate(); err != nil {
		fmt.Printf("❌ Graph config failed custom validation: %v\n", err)
	} else {
		fmt.Println("✅ Graph config passed custom validation")
	}
}

func demoHTTPMiddleware() {
	type CreateNodeRequest struct {
		Name    string                 `json:"name" validate:"required,min=1,max=100"`
		Type    string                 `json:"type" validate:"required,oneof=function tool condition"`
		Enabled bool                   `json:"enabled"`
		Config  map[string]interface{} `json:"config,omitempty"`
	}

	// Create a simple HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Node created successfully",
		})
	})

	// Create validation middleware
	validator := validation.NewRequestValidator(validation.DefaultValidationConfig()).
		JSON(CreateNodeRequest{}).
		QueryParams(map[string]string{"graph_id": "required"}).
		Headers(map[string]string{"Authorization": "required"})

	// Wrap handler with validation
	validatedHandler := validator.Build()(handler)

	// Create HTTP server (for demonstration - not actually starting)
	server := &http.Server{
		Addr:    ":8080",
		Handler: validatedHandler,
	}

	fmt.Printf("✅ HTTP server configured with validation middleware at %s\n", server.Addr)
	fmt.Println("   - JSON body validation for CreateNodeRequest")
	fmt.Println("   - Query parameter validation for 'graph_id'")
	fmt.Println("   - Header validation for 'Authorization'")

	// Demonstrate error serialization
	errors := validation.ValidationErrors{
		{Field: "name", Value: "", Message: "field is required"},
		{Field: "type", Value: "invalid", Message: "must be one of: function, tool, condition"},
	}

	errorJSON, err := validation.MarshalValidationErrors(errors)
	if err != nil {
		log.Printf("Failed to marshal errors: %v", err)
	} else {
		fmt.Printf("✅ Validation errors JSON:\n%s\n", string(errorJSON))
	}
}

func stringPtr(s string) *string {
	return &s
}
