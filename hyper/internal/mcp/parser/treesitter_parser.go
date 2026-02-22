package parser

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// TreeSitterParser implements the ASTParser interface using Tree-sitter
type TreeSitterParser struct {
	language     *sitter.Language
	languageName string
}

// NewTreeSitterParser creates a new Tree-sitter parser for the specified language
func NewTreeSitterParser(lang string) (*TreeSitterParser, error) {
	var tsLang *sitter.Language

	switch lang {
	case "javascript", "js", "jsx":
		tsLang = javascript.GetLanguage()
	case "typescript", "ts":
		tsLang = typescript.GetLanguage()
	case "tsx":
		tsLang = tsx.GetLanguage()
	case "python", "py":
		tsLang = python.GetLanguage()
	case "java":
		tsLang = java.GetLanguage()
	case "cpp", "c++", "cc", "cxx":
		tsLang = cpp.GetLanguage()
	case "rust", "rs":
		tsLang = rust.GetLanguage()
	default:
		return nil, fmt.Errorf("unsupported language: %s", lang)
	}

	// Validate language was loaded
	if tsLang == nil {
		return nil, fmt.Errorf("failed to load tree-sitter language for: %s", lang)
	}

	return &TreeSitterParser{
		language:     tsLang,
		languageName: lang,
	}, nil
}

// Parse implements the ASTParser interface
func (p *TreeSitterParser) Parse(filePath string, content []byte) (nodes []CodeNode, err error) {
	// Recover from panics (Tree-sitter C library issues)
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tree-sitter panic (C library not available): %v", r)
			nodes = []CodeNode{}
		}
	}()

	// Create parser
	parser := sitter.NewParser()
	if parser == nil {
		return nil, fmt.Errorf("failed to create tree-sitter parser")
	}

	// Set language
	parser.SetLanguage(p.language)

	// Parse the source code
	tree, parseErr := parser.ParseCtx(context.TODO(), nil, content)
	if parseErr != nil {
		return nil, fmt.Errorf("failed to parse: %w", parseErr)
	}
	if tree == nil {
		return nil, fmt.Errorf("tree-sitter returned nil tree (C library not available)")
	}
	defer tree.Close()

	// Extract nodes from the tree
	rootNode := tree.RootNode()

	// Walk the AST and extract relevant nodes
	p.walkNode(rootNode, content, &nodes, "")

	return nodes, nil
}

// SupportsLanguage implements the ASTParser interface
func (p *TreeSitterParser) SupportsLanguage(language string) bool {
	supportedLangs := map[string]bool{
		"javascript": true,
		"js":         true,
		"jsx":        true,
		"typescript": true,
		"ts":         true,
		"tsx":        true,
		"python":     true,
		"py":         true,
		"java":       true,
		"cpp":        true,
		"c++":        true,
		"cc":         true,
		"cxx":        true,
		"rust":       true,
		"rs":         true,
	}
	return supportedLangs[language]
}

// walkNode recursively walks the AST tree and extracts relevant nodes
func (p *TreeSitterParser) walkNode(node *sitter.Node, content []byte, nodes *[]CodeNode, parentName string) {
	nodeType := node.Type()

	// Extract based on node type
	switch nodeType {
	case "function_declaration", "function", "function_definition":
		p.extractFunction(node, content, nodes, parentName)
	case "method_declaration", "method_definition":
		p.extractMethod(node, content, nodes, parentName)
	case "class_declaration", "class_definition":
		p.extractClass(node, content, nodes)
	case "struct_specifier":
		p.extractStruct(node, content, nodes)
	case "interface_declaration":
		p.extractInterface(node, content, nodes)
	case "import_statement", "import_from_statement", "import_declaration":
		p.extractImport(node, content, nodes)
	}

	// Recursively process child nodes
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			p.walkNode(child, content, nodes, parentName)
		}
	}
}

// extractFunction extracts function declarations
func (p *TreeSitterParser) extractFunction(node *sitter.Node, content []byte, nodes *[]CodeNode, parentName string) {
	name := p.findNodeName(node, content)
	if name == "" {
		name = "anonymous"
	}

	signature := p.extractSignature(node, content)
	docContent := p.findDocstring(node, content)
	symbols := p.extractSymbols(node, content)

	metadata := map[string]interface{}{
		"parentName": parentName,
	}
	if docContent != "" {
		metadata["hasDocstring"] = true
		metadata["docContent"] = docContent
	}
	if len(symbols) > 0 {
		metadata["symbols"] = symbols
	}

	codeNode := CodeNode{
		Type:      NodeTypeFunction,
		Name:      name,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   string(content[node.StartByte():node.EndByte()]),
		Signature: signature,
		Metadata:  metadata,
	}

	*nodes = append(*nodes, codeNode)
}

// extractMethod extracts method declarations
func (p *TreeSitterParser) extractMethod(node *sitter.Node, content []byte, nodes *[]CodeNode, parentName string) {
	name := p.findNodeName(node, content)
	if name == "" {
		name = "anonymous"
	}

	signature := p.extractSignature(node, content)
	docContent := p.findDocstring(node, content)
	symbols := p.extractSymbols(node, content)

	metadata := map[string]interface{}{
		"parentName": parentName,
	}
	if docContent != "" {
		metadata["hasDocstring"] = true
		metadata["docContent"] = docContent
	}
	if len(symbols) > 0 {
		metadata["symbols"] = symbols
	}

	codeNode := CodeNode{
		Type:      NodeTypeMethod,
		Name:      name,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   string(content[node.StartByte():node.EndByte()]),
		Signature: signature,
		Metadata:  metadata,
	}

	*nodes = append(*nodes, codeNode)
}

// extractClass extracts class declarations
func (p *TreeSitterParser) extractClass(node *sitter.Node, content []byte, nodes *[]CodeNode) {
	name := p.findNodeName(node, content)
	if name == "" {
		return
	}

	signature := p.extractSignature(node, content)
	docContent := p.findDocstring(node, content)
	symbols := p.extractSymbols(node, content)

	metadata := map[string]interface{}{}
	if docContent != "" {
		metadata["hasDocstring"] = true
		metadata["docContent"] = docContent
	}
	if len(symbols) > 0 {
		metadata["symbols"] = symbols
	}

	codeNode := CodeNode{
		Type:      NodeTypeClass,
		Name:      name,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   string(content[node.StartByte():node.EndByte()]),
		Signature: signature,
		Metadata:  metadata,
	}

	*nodes = append(*nodes, codeNode)

	// Recursively extract methods from the class
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil && (child.Type() == "method_declaration" || child.Type() == "method_definition") {
			p.extractMethod(child, content, nodes, name)
		}
	}
}

// extractStruct extracts struct declarations
func (p *TreeSitterParser) extractStruct(node *sitter.Node, content []byte, nodes *[]CodeNode) {
	name := p.findNodeName(node, content)
	if name == "" {
		return
	}

	signature := p.extractSignature(node, content)
	docContent := p.findDocstring(node, content)
	symbols := p.extractSymbols(node, content)

	metadata := map[string]interface{}{}
	if docContent != "" {
		metadata["hasDocstring"] = true
		metadata["docContent"] = docContent
	}
	if len(symbols) > 0 {
		metadata["symbols"] = symbols
	}

	codeNode := CodeNode{
		Type:      NodeTypeStruct,
		Name:      name,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   string(content[node.StartByte():node.EndByte()]),
		Signature: signature,
		Metadata:  metadata,
	}

	*nodes = append(*nodes, codeNode)
}

// extractInterface extracts interface declarations
func (p *TreeSitterParser) extractInterface(node *sitter.Node, content []byte, nodes *[]CodeNode) {
	name := p.findNodeName(node, content)
	if name == "" {
		return
	}

	signature := p.extractSignature(node, content)
	docContent := p.findDocstring(node, content)
	symbols := p.extractSymbols(node, content)

	metadata := map[string]interface{}{}
	if docContent != "" {
		metadata["hasDocstring"] = true
		metadata["docContent"] = docContent
	}
	if len(symbols) > 0 {
		metadata["symbols"] = symbols
	}

	codeNode := CodeNode{
		Type:      NodeTypeInterface,
		Name:      name,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   string(content[node.StartByte():node.EndByte()]),
		Signature: signature,
		Metadata:  metadata,
	}

	*nodes = append(*nodes, codeNode)
}

// extractImport extracts import statements
func (p *TreeSitterParser) extractImport(node *sitter.Node, content []byte, nodes *[]CodeNode) {
	importPath := string(content[node.StartByte():node.EndByte()])

	codeNode := CodeNode{
		Type:      NodeTypeImport,
		Name:      importPath,
		StartLine: int(node.StartPoint().Row) + 1,
		EndLine:   int(node.EndPoint().Row) + 1,
		Content:   importPath,
		Signature: importPath,
		Metadata:  map[string]interface{}{},
	}

	*nodes = append(*nodes, codeNode)
}

// findNodeName finds the identifier/name of a node
func (p *TreeSitterParser) findNodeName(node *sitter.Node, content []byte) string {
	// Look for identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		childType := child.Type()
		if childType == "identifier" || childType == "name" || childType == "type_identifier" {
			return string(content[child.StartByte():child.EndByte()])
		}
	}

	return ""
}

// extractSignature extracts the function/method signature
func (p *TreeSitterParser) extractSignature(node *sitter.Node, content []byte) string {
	// For now, use the first line as signature
	// This could be enhanced to extract just the declaration line
	lines := strings.Split(string(content[node.StartByte():node.EndByte()]), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}
	return ""
}

// findDocstring finds documentation comment before a node
func (p *TreeSitterParser) findDocstring(node *sitter.Node, content []byte) string {
	// Look for comment node immediately before this node
	startLine := int(node.StartPoint().Row)
	if startLine == 0 {
		return ""
	}

	// Simple heuristic: look at the previous few lines for comments
	lines := strings.Split(string(content), "\n")
	docLines := []string{}

	// Check up to 10 lines before the node
	for i := startLine - 1; i >= 0 && i >= startLine-10; i-- {
		line := strings.TrimSpace(lines[i])

		// Python docstrings
		if strings.HasPrefix(line, `"""`) || strings.HasPrefix(line, `'''`) {
			docLines = append([]string{line}, docLines...)
			// Keep looking backwards for the opening quote
			for j := i - 1; j >= 0 && j >= startLine-20; j-- {
				prevLine := strings.TrimSpace(lines[j])
				docLines = append([]string{prevLine}, docLines...)
				if strings.HasPrefix(prevLine, `"""`) || strings.HasPrefix(prevLine, `'''`) {
					break
				}
			}
			break
		}

		// JSDoc, JavaDoc, etc.
		if strings.HasPrefix(line, "/**") || strings.HasPrefix(line, "/*") {
			docLines = append([]string{line}, docLines...)
			// Keep looking backwards until we find the start
			for j := i - 1; j >= 0 && j >= startLine-20; j-- {
				prevLine := strings.TrimSpace(lines[j])
				docLines = append([]string{prevLine}, docLines...)
				if strings.Contains(prevLine, "*/") {
					break
				}
			}
			break
		}

		// Single-line comments
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			docLines = append([]string{line}, docLines...)
		} else if line != "" {
			// Stop at first non-comment, non-empty line
			break
		}
	}

	if len(docLines) > 0 {
		return strings.Join(docLines, "\n")
	}

	return ""
}

// extractSymbols extracts identifiers defined or used in a node
func (p *TreeSitterParser) extractSymbols(node *sitter.Node, content []byte) []string {
	symbols := make(map[string]bool) // Use map to deduplicate
	p.collectIdentifiers(node, content, symbols)

	// Convert to slice
	result := make([]string, 0, len(symbols))
	for symbol := range symbols {
		result = append(result, symbol)
	}

	return result
}

// collectIdentifiers recursively collects identifiers
func (p *TreeSitterParser) collectIdentifiers(node *sitter.Node, content []byte, symbols map[string]bool) {
	nodeType := node.Type()

	if nodeType == "identifier" || nodeType == "variable_name" {
		name := string(content[node.StartByte():node.EndByte()])
		if name != "" {
			symbols[name] = true
		}
	}

	// Recursively check children
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child != nil {
			p.collectIdentifiers(child, content, symbols)
		}
	}
}

// GetLanguageFromExtension returns the language name for a file extension
func GetLanguageFromExtension(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".js", ".jsx":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "tsx"
	case ".py":
		return "python"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx", ".c++", ".hpp", ".h":
		return "cpp"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}
