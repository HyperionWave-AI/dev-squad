package parser

import (
	"fmt"
	"testing"
)

func TestExtractionDebug(t *testing.T) {
	// Test with a simple JavaScript function
	jsCode := `
// This is a function comment
// It explains what the function does
function calculateSum(a, b) {
    const result = a + b;
    const message = "Sum calculated";
    return result;
}
`

	p, err := NewTreeSitterParser("javascript")
	if err != nil {
		t.Fatalf("Error creating parser: %v", err)
	}

	nodes, err := p.Parse("test.js", []byte(jsCode))
	if err != nil {
		t.Fatalf("Error parsing: %v", err)
	}

	fmt.Printf("\n=== EXTRACTION DEBUG ===\n")
	fmt.Printf("Found %d nodes\n\n", len(nodes))

	for i, node := range nodes {
		fmt.Printf("Node %d:\n", i+1)
		fmt.Printf("  Type: %s\n", node.Type)
		fmt.Printf("  Name: %s\n", node.Name)
		fmt.Printf("  Signature: %s\n", node.Signature)
		fmt.Printf("  Lines: %d-%d\n", node.StartLine, node.EndLine)

		// Check metadata
		if node.Metadata != nil {
			fmt.Printf("  Metadata keys: ")
			for key := range node.Metadata {
				fmt.Printf("%s, ", key)
			}
			fmt.Println()

			if symbols, ok := node.Metadata["symbols"].([]string); ok {
				fmt.Printf("  Symbols (%d): %v\n", len(symbols), symbols)
			} else {
				fmt.Printf("  Symbols: NOT FOUND\n")
			}

			if hasDoc, ok := node.Metadata["hasDocstring"].(bool); ok && hasDoc {
				if doc, ok := node.Metadata["docContent"].(string); ok {
					fmt.Printf("  Docstring: %s\n", doc)
				}
			} else {
				fmt.Printf("  Docstring: NOT FOUND\n")
			}
		}
		fmt.Println()
	}
}
