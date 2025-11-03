package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

// PythonParser implements AST parsing for Python files
type PythonParser struct {
	functionRegex  *regexp.Regexp
	classRegex     *regexp.Regexp
	decoratorRegex *regexp.Regexp
	methodRegex    *regexp.Regexp
}

// NewPythonParser creates a new Python AST parser
func NewPythonParser() *PythonParser {
	return &PythonParser{
		// Match function definitions: def function_name(...): or async def function_name(...):
		functionRegex: regexp.MustCompile(`(?m)^(\s*)(async\s+)?def\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`),

		// Match class definitions: class ClassName:
		classRegex: regexp.MustCompile(`(?m)^(\s*)class\s+([a-zA-Z_][a-zA-Z0-9_]*)`),

		// Match decorators: @decorator or @decorator(...)
		decoratorRegex: regexp.MustCompile(`(?m)^(\s*)@([a-zA-Z_][a-zA-Z0-9_.]*)`),

		// Match async functions: async def function_name(...):
		methodRegex: regexp.MustCompile(`(?m)^(\s*)(async\s+)?def\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`),
	}
}

// Parse parses a Python file and extracts semantic code nodes
func (p *PythonParser) Parse(filePath string, content []byte) ([]CodeNode, error) {
	lines := p.splitIntoLines(content)
	var nodes []CodeNode

	// Track class ranges for method extraction
	classRanges := p.findClassRanges(lines)

	// Extract top-level functions (not methods)
	functionNodes := p.extractTopLevelFunctions(lines, classRanges)
	nodes = append(nodes, functionNodes...)

	// Extract classes and their methods
	classNodes := p.extractClasses(lines, classRanges)
	nodes = append(nodes, classNodes...)

	return nodes, nil
}

// SupportsLanguage returns true if this parser supports the given language
func (p *PythonParser) SupportsLanguage(language string) bool {
	return language == "python" || language == "py"
}

// splitIntoLines splits content into lines while preserving line numbers
func (p *PythonParser) splitIntoLines(content []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

// extractTopLevelFunctions extracts top-level function definitions (not methods)
func (p *PythonParser) extractTopLevelFunctions(lines []string, classRanges []pythonClassRange) []CodeNode {
	var nodes []CodeNode

	for i, line := range lines {
		matches := p.functionRegex.FindStringSubmatch(line)
		if len(matches) > 3 {
			indent := matches[1]
			functionName := matches[3]

			// Skip if this is a method inside a class (has indentation or is within class range)
			if len(indent) > 0 || p.isWithinClassRange(i+1, classRanges) {
				continue
			}

			// Check for async
			isAsync := strings.Contains(line, "async def")

			// Look for decorators before the function
			decorators := p.findDecorators(lines, i)

			// Find the end of the function by tracking indentation
			endLine := p.findPythonBlockEnd(lines, i)
			if endLine == -1 {
				endLine = i
			}

			content := p.extractContent(lines, i+1, endLine+1)

			signature := p.buildFunctionSignature(functionName, isAsync, decorators)

			metadata := make(map[string]interface{})
			metadata["async"] = isAsync
			if len(decorators) > 0 {
				metadata["decorators"] = decorators
			}

			nodes = append(nodes, CodeNode{
				Type:      NodeTypeFunction,
				Name:      functionName,
				StartLine: i + 1 - len(decorators), // Include decorators in range
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
func (p *PythonParser) extractClasses(lines []string, classRanges []pythonClassRange) []CodeNode {
	var nodes []CodeNode

	for _, classRange := range classRanges {
		className := classRange.name

		// Look for decorators before the class
		decorators := p.findDecorators(lines, classRange.startLine-1)

		content := p.extractContent(lines, classRange.startLine-len(decorators), classRange.endLine)

		signature := p.buildClassSignature(className, decorators)

		metadata := make(map[string]interface{})
		if len(decorators) > 0 {
			metadata["decorators"] = decorators
		}

		nodes = append(nodes, CodeNode{
			Type:      NodeTypeClass,
			Name:      className,
			StartLine: classRange.startLine - len(decorators),
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
func (p *PythonParser) extractMethodsInRange(lines []string, className string, startIdx, endIdx int) []CodeNode {
	var nodes []CodeNode

	for i := startIdx; i <= endIdx && i < len(lines); i++ {
		line := lines[i]
		matches := p.methodRegex.FindStringSubmatch(line)
		if len(matches) > 3 {
			indent := matches[1]
			isAsync := strings.TrimSpace(matches[2]) == "async"
			methodName := matches[3]

			// Must have indentation to be a method
			if len(indent) == 0 {
				continue
			}

			// Look for decorators before the method
			decorators := p.findDecorators(lines, i)

			// Find the end of the method
			endLine := p.findPythonBlockEnd(lines, i)
			if endLine == -1 || endLine > endIdx {
				endLine = endIdx
			}

			content := p.extractContent(lines, i+1-len(decorators), endLine+1)

			signature := p.buildMethodSignature(className, methodName, isAsync, decorators)

			metadata := make(map[string]interface{})
			metadata["async"] = isAsync
			metadata["class"] = className
			if len(decorators) > 0 {
				metadata["decorators"] = decorators
			}

			nodes = append(nodes, CodeNode{
				Type:      NodeTypeMethod,
				Name:      fmt.Sprintf("%s.%s", className, methodName),
				StartLine: i + 1 - len(decorators),
				EndLine:   endLine + 1,
				Content:   content,
				Signature: signature,
				Metadata:  metadata,
			})

			// Skip to end of method
			i = endLine
		}
	}

	return nodes
}

// pythonClassRange represents the range of a class in the source file
type pythonClassRange struct {
	name      string
	startLine int // 1-based
	endLine   int // 1-based
}

// findClassRanges finds all class declarations and their ranges
func (p *PythonParser) findClassRanges(lines []string) []pythonClassRange {
	var ranges []pythonClassRange

	for i, line := range lines {
		matches := p.classRegex.FindStringSubmatch(line)
		if len(matches) > 2 {
			indent := matches[1]
			className := matches[2]

			// Only top-level classes (no indentation)
			if len(indent) > 0 {
				continue
			}

			endLine := p.findPythonBlockEnd(lines, i)
			if endLine == -1 {
				endLine = i
			}

			ranges = append(ranges, pythonClassRange{
				name:      className,
				startLine: i + 1,
				endLine:   endLine + 1,
			})
		}
	}

	return ranges
}

// isWithinClassRange checks if a line is within any class range
func (p *PythonParser) isWithinClassRange(lineNum int, classRanges []pythonClassRange) bool {
	for _, cr := range classRanges {
		if lineNum >= cr.startLine && lineNum <= cr.endLine {
			return true
		}
	}
	return false
}

// findPythonBlockEnd finds the end of a Python block by tracking indentation
func (p *PythonParser) findPythonBlockEnd(lines []string, startIdx int) int {
	if startIdx >= len(lines) {
		return -1
	}

	// Get the indentation of the definition line
	defLine := lines[startIdx]
	defIndent := p.getIndentLevel(defLine)

	// The block starts at the next line
	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Check indentation
		currentIndent := p.getIndentLevel(line)

		// If we find a line with same or less indentation, the block ended at previous line
		if currentIndent <= defIndent {
			return i - 1
		}
	}

	// Block extends to end of file
	return len(lines) - 1
}

// getIndentLevel returns the indentation level (number of leading spaces/tabs)
func (p *PythonParser) getIndentLevel(line string) int {
	indent := 0
	for _, ch := range line {
		if ch == ' ' {
			indent++
		} else if ch == '\t' {
			indent += 4 // Count tab as 4 spaces
		} else {
			break
		}
	}
	return indent
}

// findDecorators finds decorators immediately before a function/class (0-indexed line number)
func (p *PythonParser) findDecorators(lines []string, defIdx int) []string {
	var decorators []string

	// Look backwards for decorators
	for i := defIdx - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Stop at empty lines or non-decorator lines
		if line == "" {
			continue
		}

		matches := p.decoratorRegex.FindStringSubmatch(lines[i])
		if len(matches) > 2 {
			decoratorName := matches[2]
			decorators = append([]string{decoratorName}, decorators...) // Prepend to maintain order
		} else {
			// Found a non-decorator, stop looking
			break
		}
	}

	return decorators
}

// extractContent extracts content from lines (1-based line numbers)
func (p *PythonParser) extractContent(lines []string, startLine, endLine int) string {
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
func (p *PythonParser) buildFunctionSignature(name string, isAsync bool, decorators []string) string {
	var parts []string

	if len(decorators) > 0 {
		for _, dec := range decorators {
			parts = append(parts, "@"+dec)
		}
	}

	if isAsync {
		parts = append(parts, fmt.Sprintf("async def %s(...)", name))
	} else {
		parts = append(parts, fmt.Sprintf("def %s(...)", name))
	}

	return strings.Join(parts, " ")
}

// buildClassSignature builds a class signature
func (p *PythonParser) buildClassSignature(name string, decorators []string) string {
	var parts []string

	if len(decorators) > 0 {
		for _, dec := range decorators {
			parts = append(parts, "@"+dec)
		}
	}

	parts = append(parts, fmt.Sprintf("class %s", name))

	return strings.Join(parts, " ")
}

// buildMethodSignature builds a method signature
func (p *PythonParser) buildMethodSignature(className, methodName string, isAsync bool, decorators []string) string {
	var parts []string

	if len(decorators) > 0 {
		for _, dec := range decorators {
			parts = append(parts, "@"+dec)
		}
	}

	if isAsync {
		parts = append(parts, fmt.Sprintf("async def %s.%s(...)", className, methodName))
	} else {
		parts = append(parts, fmt.Sprintf("def %s.%s(...)", className, methodName))
	}

	return strings.Join(parts, " ")
}
