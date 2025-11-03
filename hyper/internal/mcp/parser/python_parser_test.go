package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPythonParser_Parse_ValidCode(t *testing.T) {
	parser := NewPythonParser()

	tests := []struct {
		name          string
		code          string
		expectedNodes int
		expectedTypes []NodeType
		expectedNames []string
	}{
		{
			name: "simple function",
			code: `def add(a, b):
    return a + b`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeFunction},
			expectedNames: []string{"add"},
		},
		{
			name: "async function",
			code: `async def fetch_data():
    response = await get_data()
    return response`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeFunction},
			expectedNames: []string{"fetch_data"},
		},
		{
			name: "class with methods",
			code: `class Calculator:
    def add(self, a, b):
        return a + b

    def subtract(self, a, b):
        return a - b`,
			expectedNodes: 3, // class + 2 methods
			expectedTypes: []NodeType{NodeTypeClass, NodeTypeMethod, NodeTypeMethod},
			expectedNames: []string{"Calculator", "Calculator.add", "Calculator.subtract"},
		},
		{
			name: "function with decorator",
			code: `@property
def value(self):
    return self._value`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeFunction},
			expectedNames: []string{"value"},
		},
		{
			name: "class with decorator",
			code: `@dataclass
class Point:
    x: int
    y: int`,
			expectedNodes: 1,
			expectedTypes: []NodeType{NodeTypeClass},
			expectedNames: []string{"Point"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := parser.Parse("test.py", []byte(tt.code))
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

func TestPythonParser_Parse_AsyncMetadata(t *testing.T) {
	parser := NewPythonParser()

	code := `async def async_func():
    pass

def normal_func():
    pass

class MyClass:
    async def async_method(self):
        pass

    def normal_method(self):
        pass`

	nodes, err := parser.Parse("test.py", []byte(code))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes), 5) // 2 funcs + class + 2 methods

	// Find async function
	for _, node := range nodes {
		if node.Name == "async_func" {
			assert.True(t, node.Metadata["async"].(bool))
		} else if node.Name == "normal_func" {
			assert.False(t, node.Metadata["async"].(bool))
		} else if node.Name == "MyClass.async_method" {
			assert.True(t, node.Metadata["async"].(bool))
		} else if node.Name == "MyClass.normal_method" {
			assert.False(t, node.Metadata["async"].(bool))
		}
	}
}

func TestPythonParser_Parse_Decorators(t *testing.T) {
	parser := NewPythonParser()

	code := `@staticmethod
@cache
def cached_func():
    pass

@dataclass
@frozen
class Config:
    value: str`

	nodes, err := parser.Parse("test.py", []byte(code))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(nodes), 2)

	// Check function decorators
	funcNode := nodes[0]
	assert.Equal(t, "cached_func", funcNode.Name)
	decorators := funcNode.Metadata["decorators"].([]string)
	assert.Contains(t, decorators, "staticmethod")
	assert.Contains(t, decorators, "cache")

	// Check class decorators
	classNode := nodes[1]
	assert.Equal(t, "Config", classNode.Name)
	classDecorators := classNode.Metadata["decorators"].([]string)
	assert.Contains(t, classDecorators, "dataclass")
	assert.Contains(t, classDecorators, "frozen")
}

func TestPythonParser_SupportsLanguage(t *testing.T) {
	parser := NewPythonParser()

	tests := []struct {
		language string
		expected bool
	}{
		{"python", true},
		{"py", true},
		{"Python", false}, // case sensitive
		{"javascript", false},
		{"go", false},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := parser.SupportsLanguage(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPythonParser_Parse_IndentationTracking(t *testing.T) {
	parser := NewPythonParser()

	code := `class Outer:
    def method1(self):
        x = 1
        y = 2
        return x + y

    def method2(self):
        if True:
            return 1
        return 0

def top_level():
    pass`

	nodes, err := parser.Parse("test.py", []byte(code))
	assert.NoError(t, err)

	// Should have class + 2 methods + 1 top-level function
	assert.GreaterOrEqual(t, len(nodes), 4)

	var classNode *CodeNode
	var methodNodes []CodeNode
	var topLevelFunc *CodeNode

	for _, node := range nodes {
		if node.Type == NodeTypeClass {
			classNode = &node
		} else if node.Type == NodeTypeMethod {
			methodNodes = append(methodNodes, node)
		} else if node.Name == "top_level" {
			topLevelFunc = &node
		}
	}

	assert.NotNil(t, classNode, "should have class node")
	assert.Equal(t, 2, len(methodNodes), "should have 2 methods")
	assert.NotNil(t, topLevelFunc, "should have top-level function")

	// Verify methods are within class range
	for _, method := range methodNodes {
		assert.GreaterOrEqual(t, method.StartLine, classNode.StartLine)
		assert.LessOrEqual(t, method.EndLine, classNode.EndLine)
		assert.Equal(t, "Outer", method.Metadata["class"])
	}

	// Verify top-level function is outside class
	assert.Greater(t, topLevelFunc.StartLine, classNode.EndLine)
}

func TestPythonParser_Parse_EmptyFile(t *testing.T) {
	parser := NewPythonParser()

	code := `# Just a comment
# Nothing to parse`

	nodes, err := parser.Parse("test.py", []byte(code))
	assert.NoError(t, err)
	assert.Equal(t, 0, len(nodes), "should return empty nodes for file with no definitions")
}
