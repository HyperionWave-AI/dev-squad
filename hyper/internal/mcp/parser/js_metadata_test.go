package parser

import (
	"testing"
)

func TestJSParser_EnhancedMetadata(t *testing.T) {
	code := []byte(`
// This function calculates the sum
// It takes two parameters
function calculateSum(a, b) {
    const result = a + b;
    const message = "calculating";
    console.log(message);
    return result;
}
`)

	parser := NewJSParser()
	nodes, err := parser.Parse("test.js", code)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least 1 node")
	}

	node := nodes[0]
	t.Logf("\n=== JS Enhanced Metadata Test ===")
	t.Logf("Node: %s", node.Name)

	// Check docstring
	if hasDoc, ok := node.Metadata["hasDocstring"].(bool); ok && hasDoc {
		if doc, ok := node.Metadata["docContent"].(string); ok {
			t.Logf("✓ Docstring: %s", doc)
		}
	} else {
		t.Error("Expected docstring")
	}

	// Check symbols
	if symbols, ok := node.Metadata["symbols"].([]string); ok {
		t.Logf("✓ Symbols (%d): %v", len(symbols), symbols)
		if len(symbols) == 0 {
			t.Error("Expected symbols")
		}
	} else {
		t.Error("Expected symbols in metadata")
	}
}
