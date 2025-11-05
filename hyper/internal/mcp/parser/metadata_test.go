package parser

import (
	"testing"
)

func TestGoParser_EnhancedMetadata(t *testing.T) {
	code := []byte(`package main

// Calculate computes the sum of two integers.
// This is a test function with documentation.
func Calculate(a, b int) int {
	result := a + b
	message := "calculating"
	fmt.Println(message)
	return result
}
`)

	parser := NewGoParser()
	nodes, err := parser.Parse("test.go", code)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("Expected at least 1 node")
	}

	node := nodes[0]
	t.Logf("\n=== Enhanced Metadata Test ===")
	t.Logf("Node: %s", node.Name)
	t.Logf("Type: %s", node.Type)

	// Check docstring
	if hasDoc, ok := node.Metadata["hasDocstring"].(bool); !ok || !hasDoc {
		t.Error("Expected hasDocstring to be true")
	} else {
		t.Logf("✓ Has docstring: true")
	}

	if doc, ok := node.Metadata["docContent"].(string); ok {
		t.Logf("✓ Docstring content: %s", doc)
		if len(doc) == 0 {
			t.Error("Expected non-empty docstring content")
		}
	} else {
		t.Error("Expected docContent in metadata")
	}

	// Check symbols
	if symbols, ok := node.Metadata["symbols"].([]string); ok {
		t.Logf("✓ Symbols (%d): %v", len(symbols), symbols)
		if len(symbols) == 0 {
			t.Error("Expected some symbols to be extracted")
		}

		// Check for expected symbols
		expectedSymbols := map[string]bool{
			"a": true, "b": true, "result": true, "message": true, "fmt": true, "Println": true,
		}
		for _, sym := range symbols {
			if expectedSymbols[sym] {
				t.Logf("  Found expected symbol: %s", sym)
			}
		}
	} else {
		t.Error("Expected symbols in metadata")
	}
}
