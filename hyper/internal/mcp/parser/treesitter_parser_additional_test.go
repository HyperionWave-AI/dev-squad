package parser

import (
	"context"
	"strings"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

func parseTreeForTest(t *testing.T, language *sitter.Language, content []byte) (*sitter.Tree, *sitter.Node) {
	t.Helper()

	p := sitter.NewParser()
	if p == nil {
		t.Fatal("expected tree-sitter parser")
	}
	p.SetLanguage(language)

	tree, err := p.ParseCtx(context.Background(), nil, content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if tree == nil {
		t.Fatal("expected tree")
	}

	return tree, tree.RootNode()
}

func findFirstNodeByType(node *sitter.Node, nodeType string) *sitter.Node {
	if node == nil {
		return nil
	}
	if node.Type() == nodeType {
		return node
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if found := findFirstNodeByType(child, nodeType); found != nil {
			return found
		}
	}
	return nil
}

func hasNodeType(nodes []CodeNode, nodeType NodeType) bool {
	for _, n := range nodes {
		if n.Type == nodeType {
			return true
		}
	}
	return false
}

func TestNewTreeSitterParser_SupportedAliases(t *testing.T) {
	langs := []string{
		"javascript", "js", "jsx",
		"typescript", "ts", "tsx",
		"python", "py",
		"java",
		"cpp", "c++", "cc", "cxx",
		"rust", "rs",
	}

	for _, lang := range langs {
		t.Run(lang, func(t *testing.T) {
			p, err := NewTreeSitterParser(lang)
			if err != nil {
				t.Fatalf("NewTreeSitterParser(%q) error: %v", lang, err)
			}
			if p == nil || p.language == nil {
				t.Fatalf("expected non-nil parser for %q", lang)
			}
			if p.languageName != lang {
				t.Fatalf("expected languageName %q, got %q", lang, p.languageName)
			}
		})
	}
}

func TestTreeSitterParser_ParseAndWalkExtractsCoreNodes(t *testing.T) {
	p, err := NewTreeSitterParser("javascript")
	if err != nil {
		t.Fatalf("NewTreeSitterParser error: %v", err)
	}

	source := []byte("import pkg from 'mod';\nclass Box { move() {} }\nfunction calc(a, b) { return a + b }\n")
	nodes, err := p.Parse("sample.js", source)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected nodes")
	}

	if !hasNodeType(nodes, NodeTypeImport) {
		t.Fatal("expected import node")
	}
	if !hasNodeType(nodes, NodeTypeClass) {
		t.Fatal("expected class node")
	}
	if !hasNodeType(nodes, NodeTypeMethod) {
		t.Fatal("expected method node")
	}
	if !hasNodeType(nodes, NodeTypeFunction) {
		t.Fatal("expected function node")
	}
}

func TestTreeSitterParser_ParseRecoversFromNilLanguagePanic(t *testing.T) {
	p := &TreeSitterParser{}
	nodes, err := p.Parse("panic.js", []byte("function x() {}"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "tree-sitter panic") {
		t.Fatalf("expected panic error message, got: %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("expected empty nodes after panic recovery, got %d", len(nodes))
	}
}

func TestTreeSitterParser_DirectExtractors(t *testing.T) {
	p := &TreeSitterParser{}
	var nodes []CodeNode

	javaSource := []byte("import java.util.List;\nclass Service { void run() {} }\ninterface Shape {}\n")
	javaTree, javaRoot := parseTreeForTest(t, java.GetLanguage(), javaSource)
	defer javaTree.Close()

	methodNode := findFirstNodeByType(javaRoot, "method_declaration")
	if methodNode == nil {
		t.Fatal("expected method_declaration node")
	}
	p.extractMethod(methodNode, javaSource, &nodes, "Service")

	interfaceNode := findFirstNodeByType(javaRoot, "interface_declaration")
	if interfaceNode == nil {
		t.Fatal("expected interface_declaration node")
	}
	p.extractInterface(interfaceNode, javaSource, &nodes)

	importNode := findFirstNodeByType(javaRoot, "import_declaration")
	if importNode == nil {
		t.Fatal("expected import_declaration node")
	}
	p.extractImport(importNode, javaSource, &nodes)

	cppSource := []byte("struct Item { int value; };")
	cppTree, cppRoot := parseTreeForTest(t, cpp.GetLanguage(), cppSource)
	defer cppTree.Close()

	structNode := findFirstNodeByType(cppRoot, "struct_specifier")
	if structNode == nil {
		t.Fatal("expected struct_specifier node")
	}
	p.extractStruct(structNode, cppSource, &nodes)

	jsSource := []byte("class Holder { run(){} }")
	jsTree, jsRoot := parseTreeForTest(t, javascript.GetLanguage(), jsSource)
	defer jsTree.Close()

	classNode := findFirstNodeByType(jsRoot, "class_declaration")
	if classNode == nil {
		t.Fatal("expected class_declaration node")
	}
	p.extractClass(classNode, jsSource, &nodes)

	jsMethod := findFirstNodeByType(jsRoot, "method_definition")
	if jsMethod == nil {
		t.Fatal("expected method_definition node")
	}
	p.extractMethod(jsMethod, jsSource, &nodes, "Holder")

	if !hasNodeType(nodes, NodeTypeMethod) {
		t.Fatal("expected extracted method nodes")
	}
	if !hasNodeType(nodes, NodeTypeInterface) {
		t.Fatal("expected extracted interface nodes")
	}
	if !hasNodeType(nodes, NodeTypeImport) {
		t.Fatal("expected extracted import nodes")
	}
	if !hasNodeType(nodes, NodeTypeStruct) {
		t.Fatal("expected extracted struct nodes")
	}
	if !hasNodeType(nodes, NodeTypeClass) {
		t.Fatal("expected extracted class nodes")
	}

	foundParent := false
	for _, n := range nodes {
		if n.Type != NodeTypeMethod {
			continue
		}
		parentName, ok := n.Metadata["parentName"].(string)
		if ok && parentName == "Service" {
			foundParent = true
			break
		}
	}
	if !foundParent {
		t.Fatal("expected method metadata with parentName=Service")
	}
}

func TestTreeSitterParser_FindDocstringVariants(t *testing.T) {
	p := &TreeSitterParser{}

	jsLineCommentSource := []byte("// function docs\nfunction alpha() {}\n")
	jsLineTree, jsLineRoot := parseTreeForTest(t, javascript.GetLanguage(), jsLineCommentSource)
	defer jsLineTree.Close()
	jsLineFunc := findFirstNodeByType(jsLineRoot, "function_declaration")
	if jsLineFunc == nil {
		t.Fatal("expected function_declaration node")
	}
	if got := p.findDocstring(jsLineFunc, jsLineCommentSource); !strings.Contains(got, "function docs") {
		t.Fatalf("expected line comment docstring, got: %q", got)
	}

	jsBlockCommentSource := []byte("/* block docs */\nfunction beta() {}\n")
	jsBlockTree, jsBlockRoot := parseTreeForTest(t, javascript.GetLanguage(), jsBlockCommentSource)
	defer jsBlockTree.Close()
	jsBlockFunc := findFirstNodeByType(jsBlockRoot, "function_declaration")
	if jsBlockFunc == nil {
		t.Fatal("expected function_declaration node")
	}
	if got := p.findDocstring(jsBlockFunc, jsBlockCommentSource); !strings.Contains(got, "block docs") {
		t.Fatalf("expected block comment docstring, got: %q", got)
	}

	pySource := []byte("'''\npython docs\n'''\ndef gamma():\n    return 1\n")
	pyTree, pyRoot := parseTreeForTest(t, python.GetLanguage(), pySource)
	defer pyTree.Close()
	pyFunc := findFirstNodeByType(pyRoot, "function_definition")
	if pyFunc == nil {
		t.Fatal("expected function_definition node")
	}
	if got := p.findDocstring(pyFunc, pySource); !strings.Contains(got, "python docs") {
		t.Fatalf("expected python docstring, got: %q", got)
	}

	noDocSource := []byte("function firstLine() {}\n")
	noDocTree, noDocRoot := parseTreeForTest(t, javascript.GetLanguage(), noDocSource)
	defer noDocTree.Close()
	noDocFunc := findFirstNodeByType(noDocRoot, "function_declaration")
	if noDocFunc == nil {
		t.Fatal("expected function_declaration node")
	}
	if got := p.findDocstring(noDocFunc, noDocSource); got != "" {
		t.Fatalf("expected empty docstring for first-line function, got: %q", got)
	}
}

func TestTreeSitterParser_UtilityHelpers(t *testing.T) {
	p := &TreeSitterParser{}

	tsSource := []byte("interface I { x: number }\n")
	tsTree, tsRoot := parseTreeForTest(t, typescript.GetLanguage(), tsSource)
	defer tsTree.Close()
	ifaceNode := findFirstNodeByType(tsRoot, "interface_declaration")
	if ifaceNode == nil {
		t.Fatal("expected interface_declaration node")
	}

	if name := p.findNodeName(ifaceNode, tsSource); name != "I" {
		t.Fatalf("expected interface name I, got %q", name)
	}

	signature := p.extractSignature(ifaceNode, tsSource)
	if signature == "" || !strings.HasPrefix(signature, "interface I") {
		t.Fatalf("expected extracted signature, got %q", signature)
	}

	importSource := []byte("import z from 'pkg';\n")
	importTree, importRoot := parseTreeForTest(t, javascript.GetLanguage(), importSource)
	defer importTree.Close()
	importNode := findFirstNodeByType(importRoot, "import_statement")
	if importNode == nil {
		t.Fatal("expected import_statement node")
	}

	if name := p.findNodeName(importNode, importSource); name != "" {
		t.Fatalf("expected empty name for import node, got %q", name)
	}
}

func TestGetLanguageFromExtension_AllMappings(t *testing.T) {
	tests := map[string]string{
		"index.js":    "javascript",
		"index.jsx":   "javascript",
		"index.ts":    "typescript",
		"index.tsx":   "tsx",
		"main.py":     "python",
		"Main.java":   "java",
		"native.cpp":  "cpp",
		"native.cc":   "cpp",
		"native.cxx":  "cpp",
		"native.c++":  "cpp",
		"header.hpp":  "cpp",
		"header.h":    "cpp",
		"lib.rs":      "rust",
		"UNKNOWN.TS":  "typescript",
		"noext":       "",
		"unknown.txt": "",
	}

	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			if got := GetLanguageFromExtension(input); got != expected {
				t.Fatalf("GetLanguageFromExtension(%q) = %q, want %q", input, got, expected)
			}
		})
	}
}
