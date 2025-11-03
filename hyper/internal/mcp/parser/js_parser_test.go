package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSParser_Parse_ValidCode(t *testing.T) {
	parser := NewJSParser()

	tests := []struct {
		name          string
		code          string
		expectedNodes int
		expectedTypes []NodeType
		expectedNames []string
	}{
		{
			name: "simple function",
			code: `function add(a, b) {
	return a + b;
}`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeFunction},
			expectedNames: []string{"add"},
		},
		{
			name: "arrow function",
			code: `const multiply = (a, b) => {
	return a * b;
}`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeArrowFunction},
			expectedNames: []string{"multiply"},
		},
		{
			name: "async function",
			code: `async function fetchData() {
	const response = await fetch('/api');
	return response.json();
}`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeFunction},
			expectedNames: []string{"fetchData"},
		},
		{
			name: "class with methods",
			code: `class Calculator {
	add(a, b) {
		return a + b;
	}

	subtract(a, b) {
		return a - b;
	}
}`,
			expectedNodes: 3, // class + 2 methods
			expectedTypes: []NodeType{NodeTypeClass, NodeTypeMethod, NodeTypeMethod},
			expectedNames: []string{"Calculator", "Calculator.add", "Calculator.subtract"},
		},
		{
			name: "exported functions",
			code: `export function publicFunc() {}
export const arrowFunc = () => {}`,
			expectedNodes: 2,
			expectedTypes: []NodeType{NodeTypeFunction, NodeTypeArrowFunction},
			expectedNames: []string{"publicFunc", "arrowFunc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse("test.js", []byte(tt.code))
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedNodes, len(nodes), "unexpected number of nodes")

			for i, node := range nodes {
				if i < len(tt.expectedTypes) {
					assert.Equal(t, tt.expectedTypes[i], node.Type, "unexpected node type at index %d", i)
					assert.Equal(t, tt.expectedNames[i], node.Name, "unexpected node name at index %d", i)
					assert.NotEmpty(t, node.Content, "node content should not be empty")
					assert.NotEmpty(t, node.Signature, "node signature should not be empty")
					assert.Greater(t, node.EndLine, 0, "end line should be > 0")
				}
			}
		})
	}
}

func TestJSParser_Parse_AsyncMetadata(t *testing.T) {
	parser := NewJSParser()

	code := `async function asyncFunc() {}
function normalFunc() {}
const asyncArrow = async () => {}
const normalArrow = () => {}`

	nodes, err := parser.Parse("test.js", []byte(code))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes), 4)

	// Check async metadata
	asyncFuncNode := nodes[0]
	assert.True(t, asyncFuncNode.Metadata["async"].(bool))

	normalFuncNode := nodes[1]
	assert.False(t, normalFuncNode.Metadata["async"].(bool))
}

func TestJSParser_Parse_ExportedMetadata(t *testing.T) {
	parser := NewJSParser()

	code := `export function exportedFunc() {}
function privateFunc() {}
export class ExportedClass {}`

	nodes, err := parser.Parse("test.js", []byte(code))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes), 3)

	exportedFunc := nodes[0]
	assert.True(t, exportedFunc.Metadata["exported"].(bool))

	privateFunc := nodes[1]
	assert.False(t, privateFunc.Metadata["exported"].(bool))

	exportedClass := nodes[2]
	assert.True(t, exportedClass.Metadata["exported"].(bool))
}

func TestJSParser_SupportsLanguage(t *testing.T) {
	parser := NewJSParser()

	tests := []struct {
		language string
		expected bool
	}{
		{"javascript", true},
		{"typescript", true},
		{"js", true},
		{"ts", true},
		{"jsx", true},
		{"tsx", true},
		{"go", false},
		{"python", false},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := parser.SupportsLanguage(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSParser_Parse_ClassMethods(t *testing.T) {
	parser := NewJSParser()

	code := `class MyClass {
	constructor() {}

	async fetchData() {
		return await getData();
	}

	processData() {
		return this.data;
	}
}`

	nodes, err := parser.Parse("test.js", []byte(code))
	assert.NoError(t, err)

	// Should have class + methods (constructor might be filtered)
	var classNode *CodeNode
	var methodNodes []CodeNode

	for _, node := range nodes {
		if node.Type == NodeTypeClass {
			classNode = &node
		} else if node.Type == NodeTypeMethod {
			methodNodes = append(methodNodes, node)
		}
	}

	assert.NotNil(t, classNode, "should have class node")
	assert.Equal(t, "MyClass", classNode.Name)
	assert.GreaterOrEqual(t, len(methodNodes), 1, "should have at least one method")

	// Check method metadata includes class name
	for _, method := range methodNodes {
		assert.Equal(t, "MyClass", method.Metadata["class"])
	}
}
