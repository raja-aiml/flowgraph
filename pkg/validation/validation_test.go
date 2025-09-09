package validation

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "name",
		Value:   "",
		Message: "field is required",
	}

	expected := "validation error on field 'name': field is required (got: )"
	assert.Equal(t, expected, err.Error())
}

func TestValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		{Field: "name", Value: "", Message: "field is required"},
		{Field: "age", Value: -1, Message: "must be positive"},
	}

	expected := "validation error on field 'name': field is required (got: ); validation error on field 'age': must be positive (got: -1)"
	assert.Equal(t, expected, errors.Error())
}

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required"`
		Age   int    `validate:"min=0,max=120"`
		Email string `validate:"required"`
	}

	t.Run("Valid struct", func(t *testing.T) {
		ts := TestStruct{
			Name:  "John Doe",
			Age:   25,
			Email: "john@example.com",
		}

		err := ValidateStruct(ts)
		assert.NoError(t, err)
	})

	t.Run("Invalid struct - required field missing", func(t *testing.T) {
		ts := TestStruct{
			Age:   25,
			Email: "john@example.com",
		}

		err := ValidateStruct(ts)
		assert.Error(t, err)

		validationErrors, ok := err.(ValidationErrors)
		require.True(t, ok)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Name", validationErrors[0].Field)
		assert.Contains(t, validationErrors[0].Message, "required")
	})

	t.Run("Invalid struct - min validation", func(t *testing.T) {
		ts := TestStruct{
			Name:  "John Doe",
			Age:   -5,
			Email: "john@example.com",
		}

		err := ValidateStruct(ts)
		assert.Error(t, err)

		validationErrors, ok := err.(ValidationErrors)
		require.True(t, ok)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "Age", validationErrors[0].Field)
		assert.Contains(t, validationErrors[0].Message, ">=")
	})
}

func TestEnhancedValidation(t *testing.T) {
	type TestModel struct {
		NodeID      string `validate:"required,node_id"`
		EdgeID      string `validate:"required,edge_id"`
		State       string `validate:"required,graph_state"`
		Channel     string `validate:"required,channel_name"`
		MessageType string `validate:"required,message_type"`
	}

	t.Run("Valid enhanced validation", func(t *testing.T) {
		tm := TestModel{
			NodeID:      "node_1",
			EdgeID:      "source->target",
			State:       "running",
			Channel:     "test_channel",
			MessageType: "data",
		}

		err := ValidateWithPlayground(tm)
		assert.NoError(t, err)
	})

	t.Run("Invalid node ID", func(t *testing.T) {
		tm := TestModel{
			NodeID:      "node with spaces",
			EdgeID:      "source->target",
			State:       "running",
			Channel:     "test_channel",
			MessageType: "data",
		}

		err := ValidateWithPlayground(tm)
		assert.Error(t, err)

		validationErrors, ok := err.(ValidationErrors)
		require.True(t, ok)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "NodeID", validationErrors[0].Field)
	})

	t.Run("Invalid message type", func(t *testing.T) {
		tm := TestModel{
			NodeID:      "node_1",
			EdgeID:      "source->target",
			State:       "running",
			Channel:     "test_channel",
			MessageType: "invalid_type",
		}

		err := ValidateWithPlayground(tm)
		assert.Error(t, err)

		validationErrors, ok := err.(ValidationErrors)
		require.True(t, ok)
		assert.Len(t, validationErrors, 1)
		assert.Equal(t, "MessageType", validationErrors[0].Field)
	})
}

func TestCustomValidationFunctions(t *testing.T) {
	t.Run("validateNodeID", func(t *testing.T) {
		tests := []struct {
			input    string
			expected bool
		}{
			{"node_1", true},
			{"node-1", true},
			{"Node123", true},
			{"", false},
			{"node with spaces", false},
			{"node@invalid", false},
			{string(make([]byte, 101)), false}, // Too long
		}

		for _, test := range tests {
			// We can't test validateNodeID directly as it's internal,
			// but we can test through the validator
			type TestStruct struct {
				ID string `validate:"node_id"`
			}

			ts := TestStruct{ID: test.input}
			err := ValidateWithPlayground(ts)

			if test.expected {
				assert.NoError(t, err, "Input: %s", test.input)
			} else {
				assert.Error(t, err, "Input: %s", test.input)
			}
		}
	})
}

func TestValidationMiddleware(t *testing.T) {
	type RequestBody struct {
		Name string `json:"name" validate:"required"`
		Age  int    `json:"age" validate:"min=0"`
	}

	middleware := NewMiddleware(DefaultValidationConfig())

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("Valid JSON request", func(t *testing.T) {
		body := RequestBody{Name: "John", Age: 25}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		validatedHandler := middleware.ValidateJSON(RequestBody{})(handler)
		validatedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "OK", rr.Body.String())
	})

	t.Run("Invalid JSON request", func(t *testing.T) {
		body := RequestBody{Name: "", Age: -5} // Invalid: empty name, negative age
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		validatedHandler := middleware.ValidateJSON(RequestBody{})(handler)
		validatedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "errors")
	})

	t.Run("Malformed JSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/test", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		validatedHandler := middleware.ValidateJSON(RequestBody{})(handler)
		validatedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

func TestRequestValidator(t *testing.T) {
	type RequestBody struct {
		NodeID string `json:"node_id" validate:"required,node_id"`
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	validator := NewRequestValidator(DefaultValidationConfig()).
		JSON(RequestBody{}).
		QueryParams(map[string]string{"id": "required"}).
		Headers(map[string]string{"Authorization": "required"})

	validatedHandler := validator.Build()(handler)

	t.Run("Valid request", func(t *testing.T) {
		body := RequestBody{NodeID: "valid_node"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/test?id=123", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")

		rr := httptest.NewRecorder()
		validatedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Missing query parameter", func(t *testing.T) {
		body := RequestBody{NodeID: "valid_node"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token123")

		rr := httptest.NewRecorder()
		validatedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})

	t.Run("Missing authorization header", func(t *testing.T) {
		body := RequestBody{NodeID: "valid_node"}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/test?id=123", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		validatedHandler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})
}

func TestModelValidation(t *testing.T) {
	t.Run("NodeConfig validation", func(t *testing.T) {
		timeout := 30 * time.Second

		config := NodeConfig{
			ID:        "test_node",
			Name:      "Test Node",
			Type:      "function",
			Enabled:   true,
			Timeout:   &timeout,
			Retries:   3,
			Tags:      []string{"test", "function"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := ValidateWithPlayground(config)
		assert.NoError(t, err)
	})

	t.Run("NodeConfig custom validation", func(t *testing.T) {
		timeout := 120 * time.Second // Too long for condition type

		config := NodeConfig{
			ID:        "test_condition",
			Name:      "Test Condition",
			Type:      "condition",
			Enabled:   true,
			Timeout:   &timeout,
			Retries:   3,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := ValidateWithPlayground(config)
		assert.NoError(t, err) // Playground validation passes

		// But custom validation should fail
		err = config.Validate()
		assert.Error(t, err)
	})

	t.Run("GraphConfig validation", func(t *testing.T) {
		config := GraphConfig{
			ID:      "550e8400-e29b-41d4-a716-446655440000",
			Name:    "Test Graph",
			Version: "1.0.0",
			State:   "pending",
			Nodes: []NodeConfig{
				{
					ID:        "node1",
					Name:      "Node 1",
					Type:      "function",
					Enabled:   true,
					Retries:   0,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					ID:        "node2",
					Name:      "Node 2",
					Type:      "function",
					Enabled:   true,
					Retries:   0,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			Edges: []EdgeConfig{
				{
					ID:        "node1->node2",
					Source:    "node1",
					Target:    "node2",
					CreatedAt: time.Now(),
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := ValidateWithPlayground(config)
		assert.NoError(t, err)

		err = config.Validate()
		assert.NoError(t, err)
	})

	t.Run("GraphConfig custom validation - duplicate nodes", func(t *testing.T) {
		config := GraphConfig{
			ID:      "550e8400-e29b-41d4-a716-446655440000",
			Name:    "Test Graph",
			Version: "1.0.0",
			State:   "pending",
			Nodes: []NodeConfig{
				{
					ID:        "node1",
					Name:      "Node 1",
					Type:      "function",
					Enabled:   true,
					Retries:   0,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				{
					ID:        "node1", // Duplicate ID
					Name:      "Node 1 Duplicate",
					Type:      "function",
					Enabled:   true,
					Retries:   0,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := config.Validate()
		assert.Error(t, err)

		validationErrors, ok := err.(ValidationErrors)
		require.True(t, ok)
		assert.Len(t, validationErrors, 1)
		assert.Contains(t, validationErrors[0].Message, "duplicate")
	})
}

func TestValidationConfig(t *testing.T) {
	config := DefaultValidationConfig()
	assert.True(t, config.StrictMode)
	assert.False(t, config.SkipMissing)
	assert.Equal(t, 10, config.MaxErrors)
}

func TestMarshalUnmarshalValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		{Field: "name", Value: "", Message: "field is required"},
		{Field: "age", Value: -1, Message: "must be positive"},
	}

	data, err := MarshalValidationErrors(errors)
	assert.NoError(t, err)

	unmarshaled, err := UnmarshalValidationErrors(data)
	assert.NoError(t, err)

	assert.Len(t, unmarshaled, 2)
	assert.Equal(t, errors[0].Field, unmarshaled[0].Field)
	assert.Equal(t, errors[1].Field, unmarshaled[1].Field)
}
