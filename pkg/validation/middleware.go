// Package validation provides middleware for HTTP request validation
package validation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// Middleware provides validation middleware for HTTP handlers
type Middleware struct {
	config *ValidationConfig
}

// NewMiddleware creates a new validation middleware
func NewMiddleware(config *ValidationConfig) *Middleware {
	if config == nil {
		config = DefaultValidationConfig()
	}

	return &Middleware{
		config: config,
	}
}

// ValidateJSON validates JSON request body against a struct type
func (m *Middleware) ValidateJSON(structType interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create new instance of the struct type
			val := reflect.New(reflect.TypeOf(structType)).Interface()

			// Decode JSON body
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(val); err != nil {
				m.writeErrorResponse(w, http.StatusBadRequest,
					ValidationErrors{{
						Field:   "request_body",
						Value:   nil,
						Message: fmt.Sprintf("invalid JSON: %v", err),
					}})
				return
			}

			// Validate the decoded struct
			if err := ValidateWithConfig(val, m.config); err != nil {
				if validationErrors, ok := err.(ValidationErrors); ok {
					m.writeErrorResponse(w, http.StatusBadRequest, validationErrors)
					return
				}
				m.writeErrorResponse(w, http.StatusInternalServerError,
					ValidationErrors{{
						Field:   "validation",
						Value:   nil,
						Message: "validation failed",
					}})
				return
			}

			// Store validated struct in request context if needed
			// You could add context.WithValue here if you want to pass the validated struct

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateQueryParams validates URL query parameters
func (m *Middleware) ValidateQueryParams(paramRules map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			var errors ValidationErrors

			for param, rule := range paramRules {
				value := query.Get(param)

				// Basic validation rules
				switch rule {
				case "required":
					if value == "" {
						errors = append(errors, ValidationError{
							Field:   param,
							Value:   value,
							Message: "parameter is required",
						})
					}
				case "node_id":
					if value != "" && !isValidNodeID(value) {
						errors = append(errors, ValidationError{
							Field:   param,
							Value:   value,
							Message: "must be a valid node identifier",
						})
					}
				case "numeric":
					if value != "" && !isNumeric(value) {
						errors = append(errors, ValidationError{
							Field:   param,
							Value:   value,
							Message: "must be numeric",
						})
					}
				}
			}

			if len(errors) > 0 {
				m.writeErrorResponse(w, http.StatusBadRequest, errors)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ValidateHeaders validates HTTP headers
func (m *Middleware) ValidateHeaders(headerRules map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var errors ValidationErrors

			for header, rule := range headerRules {
				value := r.Header.Get(header)

				switch rule {
				case "required":
					if value == "" {
						errors = append(errors, ValidationError{
							Field:   header,
							Value:   value,
							Message: "header is required",
						})
					}
				case "bearer_token":
					if value != "" && !isBearerToken(value) {
						errors = append(errors, ValidationError{
							Field:   header,
							Value:   value,
							Message: "must be a valid bearer token",
						})
					}
				}
			}

			if len(errors) > 0 {
				m.writeErrorResponse(w, http.StatusUnauthorized, errors)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// writeErrorResponse writes validation errors as JSON response
func (m *Middleware) writeErrorResponse(w http.ResponseWriter, statusCode int, errors ValidationErrors) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorData, err := MarshalValidationErrors(errors)
	if err != nil {
		// Fallback error response
		w.Write([]byte(`{"error":"validation failed","message":"internal validation error"}`))
		return
	}

	w.Write(errorData)
}

// Helper validation functions

func isValidNodeID(s string) bool {
	if len(s) == 0 || len(s) > 100 {
		return false
	}

	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isBearerToken(s string) bool {
	return len(s) > 7 && s[:7] == "Bearer " && len(s[7:]) > 0
}

// RequestValidator provides fluent API for request validation
type RequestValidator struct {
	middleware *Middleware
	handlers   []func(http.Handler) http.Handler
}

// NewRequestValidator creates a new request validator
func NewRequestValidator(config *ValidationConfig) *RequestValidator {
	return &RequestValidator{
		middleware: NewMiddleware(config),
		handlers:   make([]func(http.Handler) http.Handler, 0),
	}
}

// JSON adds JSON body validation
func (rv *RequestValidator) JSON(structType interface{}) *RequestValidator {
	rv.handlers = append(rv.handlers, rv.middleware.ValidateJSON(structType))
	return rv
}

// QueryParams adds query parameter validation
func (rv *RequestValidator) QueryParams(rules map[string]string) *RequestValidator {
	rv.handlers = append(rv.handlers, rv.middleware.ValidateQueryParams(rules))
	return rv
}

// Headers adds header validation
func (rv *RequestValidator) Headers(rules map[string]string) *RequestValidator {
	rv.handlers = append(rv.handlers, rv.middleware.ValidateHeaders(rules))
	return rv
}

// Build creates the final middleware handler
func (rv *RequestValidator) Build() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := next
		// Apply middleware in reverse order
		for i := len(rv.handlers) - 1; i >= 0; i-- {
			handler = rv.handlers[i](handler)
		}
		return handler
	}
}
