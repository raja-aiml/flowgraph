// Package channel provides state reduction functionality
package channel

import (
	"fmt"
	"reflect"
)

// StateReducer interface for reducing state updates
// PRINCIPLES:
// - ISP: Interface segregation with single method
// - SRP: Single responsibility - state reduction
type StateReducer interface {
	// Reduce combines current state with update
	Reduce(current, update map[string]interface{}) map[string]interface{}
}

// AppendReducer appends values to lists/arrays
// PRINCIPLES:
// - KISS: Simple append operation
// - SRP: Only handles list appending
type AppendReducer struct{}

// NewAppendReducer creates a new append reducer
func NewAppendReducer() *AppendReducer {
	return &AppendReducer{}
}

// Reduce appends update values to current lists
func (r *AppendReducer) Reduce(current, update map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy current state
	for k, v := range current {
		result[k] = v
	}

	// Apply updates by appending
	for key, updateVal := range update {
		if currentVal, exists := result[key]; exists {
			// Try to append to existing value
			result[key] = r.appendValues(currentVal, updateVal)
		} else {
			// No existing value, just set it
			result[key] = updateVal
		}
	}

	return result
}

// appendValues appends two values together
func (r *AppendReducer) appendValues(current, update interface{}) interface{} {
	currentV := reflect.ValueOf(current)
	updateV := reflect.ValueOf(update)

	// Handle different types
	switch {
	case currentV.Kind() == reflect.Slice && updateV.Kind() == reflect.Slice:
		// Both are slices, concatenate them
		return reflect.AppendSlice(currentV, updateV).Interface()
	case currentV.Kind() == reflect.Slice:
		// Current is slice, append update as element
		return reflect.Append(currentV, updateV).Interface()
	case updateV.Kind() == reflect.Slice:
		// Update is slice, prepend current as element
		newSlice := reflect.MakeSlice(updateV.Type(), 0, updateV.Len()+1)
		newSlice = reflect.Append(newSlice, reflect.ValueOf(current))
		return reflect.AppendSlice(newSlice, updateV).Interface()
	default:
		// Neither is slice, create a new slice with both values
		return []interface{}{current, update}
	}
}

// MergeReducer merges map values
// PRINCIPLES:
// - KISS: Simple map merging
// - SRP: Only handles map merging
type MergeReducer struct{}

// NewMergeReducer creates a new merge reducer
func NewMergeReducer() *MergeReducer {
	return &MergeReducer{}
}

// Reduce merges update maps into current maps
func (r *MergeReducer) Reduce(current, update map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy current state
	for k, v := range current {
		result[k] = v
	}

	// Apply updates by merging
	for key, updateVal := range update {
		if currentVal, exists := result[key]; exists {
			// Try to merge existing value
			result[key] = r.mergeValues(currentVal, updateVal)
		} else {
			// No existing value, just set it
			result[key] = updateVal
		}
	}

	return result
}

// mergeValues merges two values together
func (r *MergeReducer) mergeValues(current, update interface{}) interface{} {
	// Try to merge as maps
	if currentMap, ok := current.(map[string]interface{}); ok {
		if updateMap, ok := update.(map[string]interface{}); ok {
			merged := make(map[string]interface{})
			// Copy current map
			for k, v := range currentMap {
				merged[k] = v
			}
			// Merge update map
			for k, v := range updateMap {
				if existing, exists := merged[k]; exists {
					// Recursively merge nested maps
					merged[k] = r.mergeValues(existing, v)
				} else {
					merged[k] = v
				}
			}
			return merged
		}
	}

	// If not maps, update replaces current
	return update
}

// ReplaceReducer replaces values completely
// PRINCIPLES:
// - KISS: Simple replacement
// - SRP: Only handles value replacement
type ReplaceReducer struct{}

// NewReplaceReducer creates a new replace reducer
func NewReplaceReducer() *ReplaceReducer {
	return &ReplaceReducer{}
}

// Reduce replaces current values with update values
func (r *ReplaceReducer) Reduce(current, update map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy current state
	for k, v := range current {
		result[k] = v
	}

	// Apply updates by replacement
	for key, updateVal := range update {
		result[key] = updateVal
	}

	return result
}

// MaxReducer keeps the maximum value
type MaxReducer struct{}

// NewMaxReducer creates a new max reducer
func NewMaxReducer() *MaxReducer {
	return &MaxReducer{}
}

// Reduce keeps the maximum value between current and update
func (r *MaxReducer) Reduce(current, update map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy current state
	for k, v := range current {
		result[k] = v
	}

	// Apply updates by keeping maximum
	for key, updateVal := range update {
		if currentVal, exists := result[key]; exists {
			result[key] = r.maxValue(currentVal, updateVal)
		} else {
			result[key] = updateVal
		}
	}

	return result
}

// maxValue returns the maximum of two values
func (r *MaxReducer) maxValue(current, update interface{}) interface{} {
	switch c := current.(type) {
	case int:
		if u, ok := update.(int); ok && u > c {
			return u
		}
		return c
	case int64:
		if u, ok := update.(int64); ok && u > c {
			return u
		}
		return c
	case float64:
		if u, ok := update.(float64); ok && u > c {
			return u
		}
		return c
	case string:
		if u, ok := update.(string); ok && u > c {
			return u
		}
		return c
	default:
		// For non-comparable types, return update
		return update
	}
}

// MinReducer keeps the minimum value
type MinReducer struct{}

// NewMinReducer creates a new min reducer
func NewMinReducer() *MinReducer {
	return &MinReducer{}
}

// Reduce keeps the minimum value between current and update
func (r *MinReducer) Reduce(current, update map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy current state
	for k, v := range current {
		result[k] = v
	}

	// Apply updates by keeping minimum
	for key, updateVal := range update {
		if currentVal, exists := result[key]; exists {
			result[key] = r.minValue(currentVal, updateVal)
		} else {
			result[key] = updateVal
		}
	}

	return result
}

// minValue returns the minimum of two values
func (r *MinReducer) minValue(current, update interface{}) interface{} {
	switch c := current.(type) {
	case int:
		if u, ok := update.(int); ok && u < c {
			return u
		}
		return c
	case int64:
		if u, ok := update.(int64); ok && u < c {
			return u
		}
		return c
	case float64:
		if u, ok := update.(float64); ok && u < c {
			return u
		}
		return c
	case string:
		if u, ok := update.(string); ok && u < c {
			return u
		}
		return c
	default:
		// For non-comparable types, return update
		return update
	}
}

// ReducerType represents the type of reducer
type ReducerType string

const (
	// ReducerTypeAppend appends values to lists
	ReducerTypeAppend ReducerType = "append"
	// ReducerTypeMerge merges map values
	ReducerTypeMerge ReducerType = "merge"
	// ReducerTypeReplace replaces values completely
	ReducerTypeReplace ReducerType = "replace"
	// ReducerTypeMax keeps maximum values
	ReducerTypeMax ReducerType = "max"
	// ReducerTypeMin keeps minimum values
	ReducerTypeMin ReducerType = "min"
)

// NewStateReducer creates a new state reducer by type
func NewStateReducer(reducerType ReducerType) (StateReducer, error) {
	switch reducerType {
	case ReducerTypeAppend:
		return NewAppendReducer(), nil
	case ReducerTypeMerge:
		return NewMergeReducer(), nil
	case ReducerTypeReplace:
		return NewReplaceReducer(), nil
	case ReducerTypeMax:
		return NewMaxReducer(), nil
	case ReducerTypeMin:
		return NewMinReducer(), nil
	default:
		return nil, fmt.Errorf("unknown reducer type: %s", reducerType)
	}
}
