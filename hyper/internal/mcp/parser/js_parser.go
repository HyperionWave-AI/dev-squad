package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// JSParser implements AST parsing for JavaScript and TypeScript files
type JSParser struct {
	functionRegex      *regexp.Regexp
	arrowFunctionRegex *regexp.Regexp
	classRegex         *regexp.Regexp
	methodRegex        *regexp.Regexp
}

// NewJSParser creates a new JavaScript/TypeScript AST parser
func NewJSParser() *JSParser {
	return &JSParser{
		// Match function declarations: function name(...) or async function name(...)
		functionRegex: regexp.MustCompile(`(?m)^\s*(export\s+)?(async\s+)?function\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*\(`),

		// Match arrow functions: const/let/var name = (...) => or const name = async (...) =>
		arrowFunctionRegex: regexp.MustCompile(`(?m)^\s*(export\s+)?(const|let|var)\s+([a-zA-Z_$][a-zA-Z0-9_$]*)\s*=\s*(async\s+)?\([^)]*\)\s*=>`),

		// Match class declarations: class Name or export class Name
		classRegex: regexp.MustCompile(`(?m)^\s*(export\s+)?(default\s+)?class\s+([a-zA-Z_$][a-zA-Z0-9_$]*)`),

		// Match class methods: methodName(...) or async methodName(...)
		methodRegex: regexp.MustCompile(`(?m)^\s*(async\s+)?([a-zA-Z_$][a-zA-Z0-9_$]*)\s*\([^)]*\)\s*\{`),
	}
}

// Parse parses a JavaScript/TypeScript file and extracts semantic code nodes
func (p *JSParser) Parse(filePath string, content []byte) ([]CodeNode, error) {
	lines := p.splitIntoLines(content)
	var nodes []CodeNode

	// Track classes for method extraction
	classRanges := p.findClassRanges(lines)

	// Extract function declarations
	functionNodes := p.extractFunctions(lines)
	nodes = append(nodes, functionNodes...)

	// Extract arrow functions
	arrowNodes := p.extractArrowFunctions(lines)
	nodes = append(nodes, arrowNodes...)

	// Extract classes and their methods
	classNodes := p.extractClasses(lines, classRanges)
	nodes = append(nodes, classNodes...)

	return nodes, nil
}

// SupportsLanguage returns true if this parser supports the given language
func (p *JSParser) SupportsLanguage(language string) bool {
	return language == "javascript" || language == "typescript" ||
		   language == "js" || language == "ts" || language == "jsx" || language == "tsx"
}

// splitIntoLines splits content into lines while preserving line numbers
func (p *JSParser) splitIntoLines(content []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// extractFunctions extracts function declarations
func (p *JSParser) extractFunctions(lines []string) []CodeNode {
	var nodes []CodeNode

	for i, line := range lines {
		matches := p.functionRegex.FindStringSubmatch(line)
		if len(matches) > 3 {
			functionName := matches[3]
			isAsync := strings.TrimSpace(matches[2]) == "async"
			isExported := strings.TrimSpace(matches[1]) == "export"

			// Find the end of the function by tracking braces
			endLine := p.findBlockEnd(lines, i)
			if endLine == -1 {
				endLine = i // Fallback to single line
			}

			content := p.extractContent(lines, i+1, endLine+1) // Convert to 1-based

			signature := p.buildFunctionSignature(functionName, isAsync, isExported)

			metadata := make(map[string]interface{})
			metadata["async"] = isAsync
			metadata["exported"] = isExported

			nodes = append(nodes, CodeNode{
				Type:      NodeTypeFunction,
				Name:      functionName,
				StartLine: i + 1, // 1-based line numbers
				EndLine:   endLine + 1,
				Content:   content,
				Signature: signature,
				Metadata:  metadata,
			})
		}
	}

	return nodes
}

// extractArrowFunctions extracts arrow function declarations
func (p *JSParser) extractArrowFunctions(lines []string) []CodeNode {
	var nodes []CodeNode

	for i, line := range lines {
		matches := p.arrowFunctionRegex.FindStringSubmatch(line)
		if len(matches) > 3 {
			functionName := matches[3]
			isAsync := strings.TrimSpace(matches[4]) == "async"
			isExported := strings.TrimSpace(matches[1]) == "export"

			// Find the end of the arrow function
			endLine := p.findArrowFunctionEnd(lines, i)
			if endLine == -1 {
				endLine = i
			}

			content := p.extractContent(lines, i+1, endLine+1)

			signature := p.buildArrowFunctionSignature(functionName, isAsync, isExported)

			metadata := make(map[string]interface{})
			metadata["async"] = isAsync
			metadata["exported"] = isExported
			metadata["arrow_function"] = true

			nodes = append(nodes, CodeNode{
				Type:      NodeTypeArrowFunction,
				Name:      functionName,
				StartLine: i + 1,
				EndLine:   endLine + 1,
				Content:   content,
				Signature: signature,
				Metadata:  metadata,
			})
		}
	}

	return nodes
}

// extractClasses extracts class declarations and their methods
func (p *JSParser) extractClasses(lines []string, classRanges []classRange) []CodeNode {
	var nodes []CodeNode

	for _, classRange := range classRanges {
		className := classRange.name
		isExported := classRange.exported

		content := p.extractContent(lines, classRange.startLine, classRange.endLine)

		signature := p.buildClassSignature(className, isExported)

		metadata := make(map[string]interface{})
		metadata["exported"] = isExported

		nodes = append(nodes, CodeNode{
			Type:      NodeTypeClass,
			Name:      className,
			StartLine: classRange.startLine,
			EndLine:   classRange.endLine,
			Content:   content,
			Signature: signature,
			Metadata:  metadata,
		})

		// Extract methods within the class
		methodNodes := p.extractMethodsInRange(lines, className, classRange.startLine-1, classRange.endLine-1)
		nodes = append(nodes, methodNodes...)
	}

	return nodes
}

// extractMethodsInRange extracts methods within a class range
func (p *JSParser) extractMethodsInRange(lines []string, className string, startIdx, endIdx int) []CodeNode {
	var nodes []CodeNode

	for i := startIdx; i <= endIdx && i < len(lines); i++ {
		line := lines[i]
		matches := p.methodRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			methodName := matches[2]

			// Skip constructor and common non-methods
			if methodName == "if" || methodName == "for" || methodName == "while" ||
			   methodName == "switch" || methodName == "catch" {
				continue
			}

			isAsync := strings.TrimSpace(matches[1]) == "async"

			// Find the end of the method
			endLine := p.findBlockEnd(lines, i)
			if endLine == -1 || endLine > endIdx {
				endLine = endIdx
			}

			content := p.extractContent(lines, i+1, endLine+1)

			signature := p.buildMethodSignature(className, methodName, isAsync)

			metadata := make(map[string]interface{})
			metadata["async"] = isAsync
			metadata["class"] = className

			nodes = append(nodes, CodeNode{
				Type:      NodeTypeMethod,
				Name:      fmt.Sprintf("%s.%s", className, methodName),
				StartLine: i + 1,
				EndLine:   endLine + 1,
				Content:   content,
				Signature: signature,
				Metadata:  metadata,
			})

			// Skip to end of method to avoid nested matches
			i = endLine
		}
	}

	return nodes
}

// classRange represents the range of a class in the source file
type classRange struct {
	name      string
	startLine int // 1-based
	endLine   int // 1-based
	exported  bool
}

// findClassRanges finds all class declarations and their ranges
func (p *JSParser) findClassRanges(lines []string) []classRange {
	var ranges []classRange

	for i, line := range lines {
		matches := p.classRegex.FindStringSubmatch(line)
		if len(matches) > 3 {
			className := matches[3]
			isExported := strings.TrimSpace(matches[1]) == "export"

			endLine := p.findBlockEnd(lines, i)
			if endLine == -1 {
				endLine = i
			}

			ranges = append(ranges, classRange{
				name:      className,
				startLine: i + 1,
				endLine:   endLine + 1,
				exported:  isExported,
			})
		}
	}

	return ranges
}

// findBlockEnd finds the end of a code block by tracking braces
func (p *JSParser) findBlockEnd(lines []string, startIdx int) int {
	braceCount := 0
	foundOpenBrace := false

	for i := startIdx; i < len(lines); i++ {
		line := lines[i]

		for _, ch := range line {
			if ch == '{' {
				braceCount++
				foundOpenBrace = true
			} else if ch == '}' {
				braceCount--
				if foundOpenBrace && braceCount == 0 {
					return i
				}
			}
		}
	}

	return -1
}

// findArrowFunctionEnd finds the end of an arrow function
func (p *JSParser) findArrowFunctionEnd(lines []string, startIdx int) int {
	line := lines[startIdx]

	// Check if arrow function has a block body
	if strings.Contains(line, "=> {") || strings.Contains(line, "=>{") {
		return p.findBlockEnd(lines, startIdx)
	}

	// Single expression arrow function - look for semicolon or end of statement
	for i := startIdx; i < len(lines) && i < startIdx+5; i++ {
		if strings.Contains(lines[i], ";") ||
		   (i > startIdx && !strings.HasSuffix(strings.TrimSpace(lines[i-1]), ",")) {
			return i
		}
	}

	return startIdx
}

// extractContent extracts content from lines (1-based line numbers)
func (p *JSParser) extractContent(lines []string, startLine, endLine int) string {
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}

	var result []string
	for i := startLine - 1; i < endLine && i < len(lines); i++ {
		result = append(result, lines[i])
	}

	return strings.Join(result, "\n")
}

// buildFunctionSignature builds a function signature
func (p *JSParser) buildFunctionSignature(name string, isAsync, isExported bool) string {
	var parts []string
	if isExported {
		parts = append(parts, "export")
	}
	if isAsync {
		parts = append(parts, "async")
	}
	parts = append(parts, "function", name+"(...)")
	return strings.Join(parts, " ")
}

// buildArrowFunctionSignature builds an arrow function signature
func (p *JSParser) buildArrowFunctionSignature(name string, isAsync, isExported bool) string {
	var parts []string
	if isExported {
		parts = append(parts, "export")
	}
	parts = append(parts, "const", name, "=")
	if isAsync {
		parts = append(parts, "async")
	}
	parts = append(parts, "(...) =>")
	return strings.Join(parts, " ")
}

// buildClassSignature builds a class signature
func (p *JSParser) buildClassSignature(name string, isExported bool) string {
	if isExported {
		return fmt.Sprintf("export class %s", name)
	}
	return fmt.Sprintf("class %s", name)
}

// buildMethodSignature builds a method signature
func (p *JSParser) buildMethodSignature(className, methodName string, isAsync bool) string {
	if isAsync {
		return fmt.Sprintf("async %s.%s(...)", className, methodName)
	}
	return fmt.Sprintf("%s.%s(...)", className, methodName)
}
