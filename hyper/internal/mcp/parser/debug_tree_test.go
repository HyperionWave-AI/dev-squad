package parser

import (
	"context"
	"fmt"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
)

func TestTreeStructure(t *testing.T) {
	jsCode := []byte(`
// This is a function comment
function calculateSum(a, b) {
    const result = a + b;
    return result;
}
`)

	parser := sitter.NewParser()
	if parser == nil {
		t.Fatal("Failed to create parser")
	}

	lang := javascript.GetLanguage()
	if lang == nil {
		t.Fatal("Failed to get JavaScript language")
	}

	parser.SetLanguage(lang)

	tree, err := parser.ParseCtx(context.Background(), nil, jsCode)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	defer tree.Close()

	root := tree.RootNode()

	fmt.Println("\n=== TREE STRUCTURE ===")
	printTree(root, jsCode, 0)
}

func printTree(node *sitter.Node, content []byte, depth int) {
	indent := strings.Repeat("  ", depth)

	nodeText := string(content[node.StartByte():node.EndByte()])
	if len(nodeText) > 40 {
		nodeText = nodeText[:40] + "..."
	}
	nodeText = strings.ReplaceAll(nodeText, "\n", "\\n")

	fmt.Printf("%sType: %-30s Text: %s\n", indent, node.Type(), nodeText)

	// Recursively print children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			printTree(child, content, depth+1)
		}
	}
}
