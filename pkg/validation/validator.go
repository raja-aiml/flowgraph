// Package validation provides validation utilities for FlowGraph
// This replaces Pydantic-like validation from Python
package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Validator interface for custom validation
// PRINCIPLES:
// - ISP: Simple interface with single method
// - DIP: Depend on interface, not concrete types
type Validator interface {
	Validate() error
}

// ValidationError represents a validation error with details
type ValidationError struct {
	Field   string      `json:"field"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s (got: %v)", e.Field, e.Message, e.Value)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// ValidateStruct validates a struct using reflection and tags
// PRINCIPLES:
// - KISS: Simple validation based on struct tags
// - DRY: Reusable for all structs
func ValidateStruct(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("ValidateStruct only accepts structs; got %T", v)
	}

	var errors ValidationErrors
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Check for required tag
		if tag := field.Tag.Get("validate"); tag != "" {
			if err := validateField(field.Name, fieldValue, tag); err != nil {
				errors = append(errors, err...)
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	// Check if type implements Validator interface
	if validator, ok := v.(Validator); ok {
		return validator.Validate()
	}

	return nil
}

// validateField validates a single field based on tag rules
func validateField(fieldName string, value reflect.Value, tag string) ValidationErrors {
	var errors ValidationErrors
	rules := strings.Split(tag, ",")

	for _, rule := range rules {
		rule = strings.TrimSpace(rule)

		switch {
		case rule == "required":
			if isZero(value) {
				errors = append(errors, ValidationError{
					Field:   fieldName,
					Value:   value.Interface(),
					Message: "field is required",
				})
			}

		case strings.HasPrefix(rule, "min="):
			minStr := strings.TrimPrefix(rule, "min=")
			if err := validateMin(fieldName, value, minStr); err != nil {
				errors = append(errors, *err)
			}

		case strings.HasPrefix(rule, "max="):
			maxStr := strings.TrimPrefix(rule, "max=")
			if err := validateMax(fieldName, value, maxStr); err != nil {
				errors = append(errors, *err)
			}
		}
	}

	return errors
}

// isZero checks if a value is zero/empty
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
		return v.IsNil()
	default:
		return false
	}
}

// compareValue validates value/length against a limit
// isMin determines if this is a minimum (true) or maximum (false) validation
func compareValue(fieldName string, value reflect.Value, limitStr string, isMin bool) *ValidationError {
	limit, err := strconv.ParseFloat(limitStr, 64)
	if err != nil {
		validationType := "min"
		if !isMin {
			validationType = "max"
		}
		return &ValidationError{
			Field:   fieldName,
			Value:   value.Interface(),
			Message: fmt.Sprintf("invalid %s value: %s", validationType, limitStr),
		}
	}

	var actualValue float64
	var isLength bool

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		actualValue = float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		actualValue = float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		actualValue = value.Float()
	case reflect.String:
		actualValue = float64(len(value.String()))
		isLength = true
	case reflect.Slice, reflect.Array, reflect.Map:
		actualValue = float64(value.Len())
		isLength = true
	default:
		return nil
	}

	var violated bool
	var operator, lengthPrefix string

	if isMin {
		violated = actualValue < limit
		operator = ">="
	} else {
		violated = actualValue > limit
		operator = "<="
	}

	if violated {
		if isLength {
			lengthPrefix = "length must be "
		} else {
			lengthPrefix = "value must be "
		}
		return &ValidationError{
			Field:   fieldName,
			Value:   value.Interface(),
			Message: fmt.Sprintf("%s%s %s", lengthPrefix, operator, limitStr),
		}
	}

	return nil
}

// validateMin validates minimum value/length
func validateMin(fieldName string, value reflect.Value, minStr string) *ValidationError {
	return compareValue(fieldName, value, minStr, true)
}

// validateMax validates maximum value/length
func validateMax(fieldName string, value reflect.Value, maxStr string) *ValidationError {
	return compareValue(fieldName, value, maxStr, false)
}
