// Package validation provides enhanced validation with go-playground/validator integration
package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Enhanced validator instance with custom validations
var (
	// Validate is the main validator instance
	Validate *validator.Validate
)

func init() {
	Validate = validator.New()

	// Register custom validation functions
	Validate.RegisterValidation("node_id", validateNodeID)
	Validate.RegisterValidation("edge_id", validateEdgeID)
	Validate.RegisterValidation("graph_state", validateGraphState)
	Validate.RegisterValidation("channel_name", validateChannelName)
	Validate.RegisterValidation("message_type", validateMessageType)
	Validate.RegisterValidation("uuid4", validateUUID4)
	Validate.RegisterValidation("semver", validateSemVer)

	// Register tag name function to use JSON tags for field names
	Validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		if name == "" {
			return fld.Name
		}
		return name
	})
}

// ValidateWithPlayground validates using go-playground/validator
func ValidateWithPlayground(s interface{}) error {
	err := Validate.Struct(s)
	if err != nil {
		return formatValidationErrors(err)
	}
	return nil
}

// formatValidationErrors converts validator errors to our custom format
func formatValidationErrors(err error) ValidationErrors {
	var errors ValidationErrors

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, fieldError := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   fieldError.Field(),
				Value:   fieldError.Value(),
				Message: getErrorMessage(fieldError),
			})
		}
	}

	return errors
}

// getErrorMessage returns a human-readable error message
func getErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "min":
		return fmt.Sprintf("minimum value/length is %s", fe.Param())
	case "max":
		return fmt.Sprintf("maximum value/length is %s", fe.Param())
	case "len":
		return fmt.Sprintf("length must be exactly %s", fe.Param())
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "uuid":
		return "must be a valid UUID"
	case "node_id":
		return "must be a valid node identifier (alphanumeric, underscore, hyphen)"
	case "edge_id":
		return "must be a valid edge identifier"
	case "graph_state":
		return "must be a valid graph state value"
	case "channel_name":
		return "must be a valid channel name"
	case "message_type":
		return "must be a valid message type (data, control, error, heartbeat)"
	default:
		return fmt.Sprintf("validation failed: %s", fe.Tag())
	}
}

// Custom validation functions for FlowGraph-specific rules

// validateNodeID validates node identifier format
func validateNodeID(fl validator.FieldLevel) bool {
	nodeID := fl.Field().String()
	if nodeID == "" {
		return false
	}

	// Node ID must be alphanumeric with underscores and hyphens
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, nodeID)
	return matched && len(nodeID) >= 1 && len(nodeID) <= 100
}

// validateEdgeID validates edge identifier format
func validateEdgeID(fl validator.FieldLevel) bool {
	edgeID := fl.Field().String()
	if edgeID == "" {
		return false
	}

	// Edge ID format: source->target or source->target:condition
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+->[a-zA-Z0-9_-]+(:[\w-]+)?$`, edgeID)
	return matched
}

// validateGraphState validates graph state values
func validateGraphState(fl validator.FieldLevel) bool {
	state := fl.Field().String()
	validStates := []string{"pending", "running", "completed", "failed", "cancelled", "paused"}

	for _, validState := range validStates {
		if state == validState {
			return true
		}
	}
	return false
}

// validateChannelName validates channel name format
func validateChannelName(fl validator.FieldLevel) bool {
	channelName := fl.Field().String()
	if channelName == "" {
		return false
	}

	// Channel name must be lowercase alphanumeric with underscores
	matched, _ := regexp.MatchString(`^[a-z0-9_]+$`, channelName)
	return matched && len(channelName) >= 1 && len(channelName) <= 50
}

// validateMessageType validates message type values
func validateMessageType(fl validator.FieldLevel) bool {
	msgType := fl.Field().String()
	validTypes := []string{"data", "control", "error", "heartbeat"}

	for _, validType := range validTypes {
		if msgType == validType {
			return true
		}
	}
	return false
}

// validateUUID4 validates UUID v4 format
func validateUUID4(fl validator.FieldLevel) bool {
	uuid := fl.Field().String()
	if uuid == "" {
		return false
	}

	// UUID v4 regex pattern
	matched, _ := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, uuid)
	return matched
}

// validateSemVer validates semantic version format
func validateSemVer(fl validator.FieldLevel) bool {
	version := fl.Field().String()
	if version == "" {
		return false
	}

	// Semantic version regex pattern (simplified)
	matched, _ := regexp.MatchString(`^(\d+)\.(\d+)\.(\d+)(-[\w\.-]+)?(\+[\w\.-]+)?$`, version)
	return matched
}

// ValidationConfig holds validation configuration
type ValidationConfig struct {
	StrictMode  bool `json:"strict_mode"`
	SkipMissing bool `json:"skip_missing"`
	MaxErrors   int  `json:"max_errors"`
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		StrictMode:  true,
		SkipMissing: false,
		MaxErrors:   10,
	}
}

// ValidateWithConfig validates with specific configuration
func ValidateWithConfig(s interface{}, config *ValidationConfig) error {
	if config == nil {
		config = DefaultValidationConfig()
	}

	err := ValidateWithPlayground(s)
	if err != nil {
		if validationErrors, ok := err.(ValidationErrors); ok {
			if config.MaxErrors > 0 && len(validationErrors) > config.MaxErrors {
				return ValidationErrors(validationErrors[:config.MaxErrors])
			}
		}
		return err
	}

	return nil
}

// MarshalValidationErrors marshals validation errors to JSON
func MarshalValidationErrors(errors ValidationErrors) ([]byte, error) {
	type ErrorResponse struct {
		Errors []ValidationError `json:"errors"`
		Count  int               `json:"count"`
	}

	response := ErrorResponse{
		Errors: errors,
		Count:  len(errors),
	}

	return json.Marshal(response)
}

// UnmarshalValidationErrors unmarshals validation errors from JSON
func UnmarshalValidationErrors(data []byte) (ValidationErrors, error) {
	type ErrorResponse struct {
		Errors []ValidationError `json:"errors"`
		Count  int               `json:"count"`
	}

	var response ErrorResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return ValidationErrors(response.Errors), nil
}
