package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArrayParameter(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
		wantErr  bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "JSON string array - the critical case",
			input:    `["item1", "item2", "item3"]`,
			expected: []string{"item1", "item2", "item3"},
			wantErr:  false,
		},
		{
			name:     "JSON string array with spaces",
			input:    `["task 1", "task 2"]`,
			expected: []string{"task 1", "task 2"},
			wantErr:  false,
		},
		{
			name:     "native []interface{} array",
			input:    []interface{}{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
			wantErr:  false,
		},
		{
			name:     "native []string array",
			input:    []string{"x", "y", "z"},
			expected: []string{"x", "y", "z"},
			wantErr:  false,
		},
		{
			name:     "[]interface{} with mixed types - converts to strings",
			input:    []interface{}{"text", 123, true},
			expected: []string{"text", "123", "true"},
			wantErr:  false,
		},
		{
			name:     "single string value - becomes single-element array",
			input:    "single-item",
			expected: []string{"single-item"},
			wantErr:  false,
		},
		{
			name:     "single number value - becomes single-element array",
			input:    42,
			expected: []string{"42"},
			wantErr:  false,
		},
		{
			name:     "empty JSON array",
			input:    "[]",
			expected: []string{},
			wantErr:  false,
		},
		{
			name:     "empty native array",
			input:    []interface{}{},
			expected: []string{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseArrayParameter(tt.input, "testParam")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestParseArrayParameter_RealWorldAIInput tests the exact scenario from AI
func TestParseArrayParameter_RealWorldAIInput(t *testing.T) {
	// Simulate what AI sends: todos as a JSON string instead of native array
	input := map[string]interface{}{
		"todos": `["Create user interface", "Add validation", "Write tests"]`,
	}

	todosRaw := input["todos"]
	result, err := parseArrayParameter(todosRaw, "todos")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"Create user interface",
		"Add validation",
		"Write tests",
	}, result)
}

// TestParseArrayParameter_FilesModified tests filesModified parameter parsing
func TestParseArrayParameter_FilesModified(t *testing.T) {
	// AI sends filesModified as JSON string array
	input := map[string]interface{}{
		"filesModified": `["/path/to/file1.go", "/path/to/file2.go"]`,
	}

	filesModifiedRaw := input["filesModified"]
	result, err := parseArrayParameter(filesModifiedRaw, "filesModified")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"/path/to/file1.go",
		"/path/to/file2.go",
	}, result)
}

// TestParseArrayParameter_QdrantCollections tests qdrantCollections parameter parsing
func TestParseArrayParameter_QdrantCollections(t *testing.T) {
	// AI sends qdrantCollections as JSON string array
	input := map[string]interface{}{
		"qdrantCollections": `["collection1", "collection2"]`,
	}

	collectionsRaw := input["qdrantCollections"]
	result, err := parseArrayParameter(collectionsRaw, "qdrantCollections")

	require.NoError(t, err)
	assert.Equal(t, []string{
		"collection1",
		"collection2",
	}, result)
}
