package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoParser_Parse_ValidCode(t *testing.T) {
	parser := NewGoParser()

	tests := []struct {
		name          string
		code          string
		expectedNodes int
		expectedTypes []NodeType
		expectedNames []string
	}{
		{
			name: "simple function",
			code: `package main

func Add(a, b int) int {
	return a + b
}`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeFunction},
			expectedNames: []string{"Add"},
		},
		{
			name: "struct and method",
			code: `package main

type Calculator struct {
	value int
}

func (c *Calculator) Add(n int) {
	c.value += n
}`,
			expectedNodes: 2,
			expectedTypes: []NodeType{NodeTypeStruct, NodeTypeMethod},
			expectedNames: []string{"Calculator", "*Calculator.Add"},
		},
		{
			name: "interface",
			code: `package main

type Reader interface {
	Read(p []byte) (n int, err error)
}`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeInterface},
			expectedNames: []string{"Reader"},
		},
		{
			name: "exported constant",
			code: `package main

const MaxValue = 100`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeConstant},
			expectedNames: []string{"MaxValue"},
		},
		{
			name: "multiple functions",
			code: `package main

func Foo() {}
func Bar() {}
func Baz() {}`,
			expectedNodes: 3,
			expectedTypes: []NodeType{NodeTypeFunction, NodeTypeFunction, NodeTypeFunction},
			expectedNames: []string{"Foo", "Bar", "Baz"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse("test.go", []byte(tt.code))
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedNodes, len(nodes), "unexpected number of nodes")

			for i, node := range nodes {
				assert.Equal(t, tt.expectedTypes[i], node.Type, "unexpected node type at index %d", i)
				assert.Equal(t, tt.expectedNames[i], node.Name, "unexpected node name at index %d", i)
				assert.NotEmpty(t, node.Content, "node content should not be empty")
				assert.NotEmpty(t, node.Signature, "node signature should not be empty")
				assert.Greater(t, node.EndLine, node.StartLine-1, "end line should be >= start line")
			}
		})
	}
}

func TestGoParser_Parse_InvalidSyntax(t *testing.T) {
	parser := NewGoParser()

	invalidCode := `package main

func BrokenFunction( {
	// Missing closing paren and brace
`

	nodes, err := parser.Parse("test.go", []byte(invalidCode))
	assert.Error(t, err, "should return error for invalid syntax")
	assert.Nil(t, nodes, "should return nil nodes on error")
}

func TestGoParser_SupportsLanguage(t *testing.T) {
	parser := NewGoParser()

	tests := []struct {
		language string
		expected bool
	}{
		{"go", true},
		{"golang", true},
		{"Go", false}, // case sensitive
		{"javascript", false},
		{"python", false},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := parser.SupportsLanguage(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGoParser_ExtractMetadata(t *testing.T) {
	parser := NewGoParser()

	code := `package main

type Handler struct{}

func (h *Handler) Handle() {}

func PublicFunc() {}

func privateFunc() {}`

	nodes, err := parser.Parse("test.go", []byte(code))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes), 3)

	// Check struct
	structNode := nodes[0]
	assert.Equal(t, NodeTypeStruct, structNode.Type)
	assert.True(t, structNode.Metadata["exported"].(bool))

	// Check method has receiver metadata
	methodNode := nodes[1]
	assert.Equal(t, NodeTypeMethod, methodNode.Type)
	assert.Equal(t, "*Handler", methodNode.Metadata["receiver"])

	// Check exported vs private functions
	for _, node := range nodes {
		if node.Type == NodeTypeFunction {
			exported := node.Metadata["exported"].(bool)
			if node.Name == "PublicFunc" {
				assert.True(t, exported)
			} else if node.Name == "privateFunc" {
				assert.False(t, exported)
			}
		}
	}
}
