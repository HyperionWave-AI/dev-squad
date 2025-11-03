package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoParser implements AST parsing for Go files
type GoParser struct{}

// NewGoParser creates a new Go AST parser
func NewGoParser() *GoParser {
	return &GoParser{}
}

// Parse parses a Go file and extracts semantic code nodes
func (p *GoParser) Parse(filePath string, content []byte) ([]CodeNode, error) {
	fset := token.NewFileSet()

	// Parse with AllErrors mode to get as much information as possible
	file, err := parser.ParseFile(fset, filePath, content, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	var nodes []CodeNode

	// Walk the AST and extract nodes
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			// Extract function or method declaration
			codeNode := p.extractFuncDecl(node, fset, content)
			if codeNode != nil {
				nodes = append(nodes, *codeNode)
			}

		case *ast.GenDecl:
			// Extract type, const, var, or import declarations
			codeNodes := p.extractGenDecl(node, fset, content)
			nodes = append(nodes, codeNodes...)
		}
		return true
	})

	return nodes, nil
}

// SupportsLanguage returns true if this parser supports the given language
func (p *GoParser) SupportsLanguage(language string) bool {
	return language == "go" || language == "golang"
}

// extractFuncDecl extracts a function or method declaration
func (p *GoParser) extractFuncDecl(fn *ast.FuncDecl, fset *token.FileSet, content []byte) *CodeNode {
	startPos := fset.Position(fn.Pos())
	endPos := fset.Position(fn.End())

	// Determine if this is a method or function
	nodeType := NodeTypeFunction
	nodeName := fn.Name.Name
	signature := p.buildFunctionSignature(fn)

	metadata := make(map[string]interface{})

	// Check if this is a method (has a receiver)
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		nodeType = NodeTypeMethod
		receiverType := p.extractReceiverType(fn.Recv.List[0])
		metadata["receiver"] = receiverType
		nodeName = fmt.Sprintf("%s.%s", receiverType, fn.Name.Name)
	}

	// Check if function is exported
	metadata["exported"] = fn.Name.IsExported()

	// Extract function content
	nodeContent := p.extractContent(content, startPos.Line, endPos.Line)

	return &CodeNode{
		Type:      nodeType,
		Name:      nodeName,
		StartLine: startPos.Line,
		EndLine:   endPos.Line,
		Content:   nodeContent,
		Signature: signature,
		Metadata:  metadata,
	}
}

// extractGenDecl extracts general declarations (type, const, var, import)
func (p *GoParser) extractGenDecl(gen *ast.GenDecl, fset *token.FileSet, content []byte) []CodeNode {
	var nodes []CodeNode

	for _, spec := range gen.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			// Extract type declaration (struct, interface, type alias)
			node := p.extractTypeSpec(s, gen, fset, content)
			if node != nil {
				nodes = append(nodes, *node)
			}

		case *ast.ValueSpec:
			// Extract var or const declaration
			node := p.extractValueSpec(s, gen, fset, content)
			if node != nil {
				nodes = append(nodes, *node)
			}

		case *ast.ImportSpec:
			// We typically don't index imports as separate chunks
			// But we could if needed for completeness
			continue
		}
	}

	return nodes
}

// extractTypeSpec extracts type declarations
func (p *GoParser) extractTypeSpec(spec *ast.TypeSpec, gen *ast.GenDecl, fset *token.FileSet, content []byte) *CodeNode {
	startPos := fset.Position(gen.Pos())
	endPos := fset.Position(gen.End())

	nodeType := NodeTypeType
	metadata := make(map[string]interface{})
	metadata["exported"] = spec.Name.IsExported()

	// Determine specific type
	switch spec.Type.(type) {
	case *ast.StructType:
		nodeType = NodeTypeStruct
	case *ast.InterfaceType:
		nodeType = NodeTypeInterface
	}

	signature := p.buildTypeSignature(spec, gen)
	nodeContent := p.extractContent(content, startPos.Line, endPos.Line)

	return &CodeNode{
		Type:      nodeType,
		Name:      spec.Name.Name,
		StartLine: startPos.Line,
		EndLine:   endPos.Line,
		Content:   nodeContent,
		Signature: signature,
		Metadata:  metadata,
	}
}

// extractValueSpec extracts variable or constant declarations
func (p *GoParser) extractValueSpec(spec *ast.ValueSpec, gen *ast.GenDecl, fset *token.FileSet, content []byte) *CodeNode {
	// Only index exported package-level vars/consts
	if len(spec.Names) == 0 || !spec.Names[0].IsExported() {
		return nil
	}

	startPos := fset.Position(gen.Pos())
	endPos := fset.Position(gen.End())

	nodeType := NodeTypeVariable
	if gen.Tok == token.CONST {
		nodeType = NodeTypeConstant
	}

	// Join multiple names if declaring multiple vars/consts
	names := make([]string, len(spec.Names))
	for i, name := range spec.Names {
		names[i] = name.Name
	}
	nodeName := strings.Join(names, ", ")

	signature := p.buildValueSignature(spec, gen)
	nodeContent := p.extractContent(content, startPos.Line, endPos.Line)

	metadata := make(map[string]interface{})
	metadata["exported"] = true

	return &CodeNode{
		Type:      nodeType,
		Name:      nodeName,
		StartLine: startPos.Line,
		EndLine:   endPos.Line,
		Content:   nodeContent,
		Signature: signature,
		Metadata:  metadata,
	}
}

// buildFunctionSignature builds a readable function signature
func (p *GoParser) buildFunctionSignature(fn *ast.FuncDecl) string {
	var sig strings.Builder

	sig.WriteString("func ")

	// Add receiver if method
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		sig.WriteString("(")
		sig.WriteString(p.extractReceiverType(fn.Recv.List[0]))
		sig.WriteString(") ")
	}

	sig.WriteString(fn.Name.Name)
	sig.WriteString("(...)")

	return sig.String()
}

// buildTypeSignature builds a readable type signature
func (p *GoParser) buildTypeSignature(spec *ast.TypeSpec, gen *ast.GenDecl) string {
	var sig strings.Builder

	sig.WriteString("type ")
	sig.WriteString(spec.Name.Name)

	switch spec.Type.(type) {
	case *ast.StructType:
		sig.WriteString(" struct")
	case *ast.InterfaceType:
		sig.WriteString(" interface")
	default:
		sig.WriteString(" ...")
	}

	return sig.String()
}

// buildValueSignature builds a readable var/const signature
func (p *GoParser) buildValueSignature(spec *ast.ValueSpec, gen *ast.GenDecl) string {
	var sig strings.Builder

	if gen.Tok == token.CONST {
		sig.WriteString("const ")
	} else {
		sig.WriteString("var ")
	}

	names := make([]string, len(spec.Names))
	for i, name := range spec.Names {
		names[i] = name.Name
	}
	sig.WriteString(strings.Join(names, ", "))

	return sig.String()
}

// extractReceiverType extracts the receiver type from a method
func (p *GoParser) extractReceiverType(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	}
	return "unknown"
}

// extractContent extracts the content between start and end lines
func (p *GoParser) extractContent(content []byte, startLine, endLine int) string {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	currentLine := 1

	for scanner.Scan() {
		if currentLine >= startLine && currentLine <= endLine {
			lines = append(lines, scanner.Text())
		}
		if currentLine > endLine {
			break
		}
		currentLine++
	}

	return strings.Join(lines, "\n")
}
