package serialization

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaDeserializer_StringField(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "string from string",
			input: json.RawMessage(`{"name": "test"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			expected: map[string]interface{}{
				"name": "test",
			},
		},
		{
			name:  "string from number",
			input: json.RawMessage(`{"count": 123}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{"type": "string"},
				},
			},
			expected: map[string]interface{}{
				"count": "123",
			},
		},
		{
			name:  "string from float",
			input: json.RawMessage(`{"value": 123.456}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"value": map[string]interface{}{"type": "string"},
				},
			},
			expected: map[string]interface{}{
				"value": "123.456",
			},
		},
		{
			name:  "string from boolean",
			input: json.RawMessage(`{"flag": true}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"flag": map[string]interface{}{"type": "string"},
				},
			},
			expected: map[string]interface{}{
				"flag": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArguments(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaDeserializer_NumberField(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "number from number",
			input: json.RawMessage(`{"value": 123.456}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"value": map[string]interface{}{"type": "number"},
				},
			},
			expected: map[string]interface{}{
				"value": 123.456,
			},
		},
		{
			name:  "number from string",
			input: json.RawMessage(`{"value": "456.789"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"value": map[string]interface{}{"type": "number"},
				},
			},
			expected: map[string]interface{}{
				"value": 456.789,
			},
		},
		{
			name:  "integer from float",
			input: json.RawMessage(`{"count": 123.99}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{"type": "integer"},
				},
			},
			expected: map[string]interface{}{
				"count": 123,
			},
		},
		{
			name:  "number from boolean true",
			input: json.RawMessage(`{"flag": true}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"flag": map[string]interface{}{"type": "number"},
				},
			},
			expected: map[string]interface{}{
				"flag": 1.0,
			},
		},
		{
			name:  "integer from boolean false",
			input: json.RawMessage(`{"flag": false}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"flag": map[string]interface{}{"type": "integer"},
				},
			},
			expected: map[string]interface{}{
				"flag": 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArguments(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaDeserializer_BooleanField(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "boolean from boolean",
			input: json.RawMessage(`{"active": true}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{"type": "boolean"},
				},
			},
			expected: map[string]interface{}{
				"active": true,
			},
		},
		{
			name:  "boolean from string true",
			input: json.RawMessage(`{"active": "true"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{"type": "boolean"},
				},
			},
			expected: map[string]interface{}{
				"active": true,
			},
		},
		{
			name:  "boolean from string yes",
			input: json.RawMessage(`{"active": "yes"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{"type": "boolean"},
				},
			},
			expected: map[string]interface{}{
				"active": true,
			},
		},
		{
			name:  "boolean from number 1",
			input: json.RawMessage(`{"active": 1}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{"type": "boolean"},
				},
			},
			expected: map[string]interface{}{
				"active": true,
			},
		},
		{
			name:  "boolean from number 0",
			input: json.RawMessage(`{"active": 0}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{"type": "boolean"},
				},
			},
			expected: map[string]interface{}{
				"active": false,
			},
		},
		{
			name:  "boolean from empty string",
			input: json.RawMessage(`{"active": ""}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"active": map[string]interface{}{"type": "boolean"},
				},
			},
			expected: map[string]interface{}{
				"active": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArguments(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaDeserializer_ArrayField(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "array from array",
			input: json.RawMessage(`{"items": ["a", "b", "c"]}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{"type": "string"},
					},
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{"a", "b", "c"},
			},
		},
		{
			name:  "array with type coercion",
			input: json.RawMessage(`{"numbers": ["1", 2, "3.5"]}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"numbers": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{"type": "number"},
					},
				},
			},
			expected: map[string]interface{}{
				"numbers": []interface{}{1.0, 2.0, 3.5},
			},
		},
		{
			name:  "array from single value",
			input: json.RawMessage(`{"item": "single"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"item": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{"type": "string"},
					},
				},
			},
			expected: map[string]interface{}{
				"item": []interface{}{"single"},
			},
		},
		{
			name:  "array from JSON string",
			input: json.RawMessage(`{"items": "[\"x\", \"y\", \"z\"]"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{"type": "string"},
					},
				},
			},
			expected: map[string]interface{}{
				"items": []interface{}{"x", "y", "z"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArguments(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaDeserializer_ObjectField(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "nested object",
			input: json.RawMessage(`{"user": {"name": "John", "age": 30}}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name": map[string]interface{}{"type": "string"},
							"age":  map[string]interface{}{"type": "integer"},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"user": map[string]interface{}{
					"name": "John",
					"age":  30,
				},
			},
		},
		{
			name:  "nested object with type coercion",
			input: json.RawMessage(`{"config": {"enabled": "true", "count": "42"}}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"config": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"enabled": map[string]interface{}{"type": "boolean"},
							"count":   map[string]interface{}{"type": "integer"},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"enabled": true,
					"count":   42,
				},
			},
		},
		{
			name:  "object from JSON string",
			input: json.RawMessage(`{"data": "{\"x\": 1, \"y\": 2}"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"data": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"x": map[string]interface{}{"type": "integer"},
							"y": map[string]interface{}{"type": "integer"},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"data": map[string]interface{}{
					"x": 1,
					"y": 2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArguments(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaDeserializer_RequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		wantError bool
		errorMsg  string
	}{
		{
			name:  "missing required field",
			input: json.RawMessage(`{"optional": "value"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"required": map[string]interface{}{"type": "string"},
					"optional": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"required"},
			},
			wantError: true,
			errorMsg:  "required field",
		},
		{
			name:  "all required fields present",
			input: json.RawMessage(`{"required1": "val1", "required2": "val2"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"required1": map[string]interface{}{"type": "string"},
					"required2": map[string]interface{}{"type": "string"},
				},
				"required": []interface{}{"required1", "required2"},
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeserializeToolArgumentsStrict(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSchemaDeserializer_DefaultValues(t *testing.T) {
	tests := []struct {
		name     string
		input    json.RawMessage
		schema   interface{}
		expected map[string]interface{}
	}{
		{
			name:  "uses default value when field missing",
			input: json.RawMessage(`{"other": "value"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{
						"type":    "integer",
						"default": 10,
					},
					"other": map[string]interface{}{"type": "string"},
				},
			},
			expected: map[string]interface{}{
				"count": 10,
				"other": "value",
			},
		},
		{
			name:  "ignores default when field present",
			input: json.RawMessage(`{"count": 20}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"count": map[string]interface{}{
						"type":    "integer",
						"default": 10,
					},
				},
			},
			expected: map[string]interface{}{
				"count": 20,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArguments(tt.input, tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaDeserializer_AdditionalProperties(t *testing.T) {
	tests := []struct {
		name      string
		input     json.RawMessage
		schema    interface{}
		expected  map[string]interface{}
		wantError bool
	}{
		{
			name:  "allows additional properties by default",
			input: json.RawMessage(`{"known": "value", "unknown": "extra"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"known": map[string]interface{}{"type": "string"},
				},
			},
			expected: map[string]interface{}{
				"known":   "value",
				"unknown": "extra",
			},
		},
		{
			name:  "rejects additional properties when false",
			input: json.RawMessage(`{"known": "value", "unknown": "extra"}`),
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"known": map[string]interface{}{"type": "string"},
				},
				"additionalProperties": false,
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DeserializeToolArgumentsStrict(tt.input, tt.schema)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaDeserializer_ComplexExample(t *testing.T) {
	// Test a real-world example with mixed types and nested structures
	input := json.RawMessage(`{
		"taskId": 12345,
		"assignedTo": "user123",
		"priority": "3",
		"tags": "bug,urgent",
		"metadata": "{\"reporter\": \"john\", \"version\": \"1.2.3\"}",
		"isActive": 1,
		"dueDate": "2024-01-15"
	}`)

	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"taskId": map[string]interface{}{"type": "string"},
			"assignedTo": map[string]interface{}{"type": "string"},
			"priority": map[string]interface{}{"type": "integer"},
			"tags": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{"type": "string"},
			},
			"metadata": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"reporter": map[string]interface{}{"type": "string"},
					"version":  map[string]interface{}{"type": "string"},
				},
			},
			"isActive": map[string]interface{}{"type": "boolean"},
			"dueDate": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"taskId", "assignedTo"},
	}

	expected := map[string]interface{}{
		"taskId":     "12345",
		"assignedTo": "user123",
		"priority":   3,
		"tags":       []interface{}{"bug,urgent"}, // Single string becomes array
		"metadata": map[string]interface{}{
			"reporter": "john",
			"version":  "1.2.3",
		},
		"isActive": true,
		"dueDate":  "2024-01-15",
	}

	result, err := DeserializeToolArguments(input, schema)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestNormalizeToolArguments(t *testing.T) {
	input := json.RawMessage(`{"count": "42", "enabled": 1}`)
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count":   map[string]interface{}{"type": "integer"},
			"enabled": map[string]interface{}{"type": "boolean"},
		},
	}

	normalized, err := NormalizeToolArguments(input, schema)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(normalized, &result)
	require.NoError(t, err)

	assert.Equal(t, float64(42), result["count"]) // JSON unmarshals numbers as float64
	assert.Equal(t, true, result["enabled"])
}

func TestValidateAgainstSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"required": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"required"},
	}

	// Valid input
	validInput := json.RawMessage(`{"required": "value"}`)
	err := ValidateAgainstSchema(validInput, schema)
	assert.NoError(t, err)

	// Invalid input (missing required field)
	invalidInput := json.RawMessage(`{"optional": "value"}`)
	err = ValidateAgainstSchema(invalidInput, schema)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}
