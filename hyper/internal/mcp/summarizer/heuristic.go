package summarizer

import (
	"fmt"
	"regexp"
	"strings"

	"go.uber.org/zap"
)

// HeuristicEngine provides fallback code summarization using pattern matching
// when LLM-based summarization is unavailable or fails
type HeuristicEngine struct {
	logger *zap.Logger
}

// NewHeuristicEngine creates a new heuristic summarization engine
func NewHeuristicEngine(logger *zap.Logger) *HeuristicEngine {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &HeuristicEngine{
		logger: logger,
	}
}

// Summarize generates a summary using heuristic strategies
func (h *HeuristicEngine) Summarize(code string, metadata CodeMetadata) string {
	var parts []string

	// Strategy 1: Use existing documentation
	if metadata.DocContent != "" {
		docSummary := h.extractDocumentation(metadata.DocContent)
		if docSummary != "" {
			parts = append(parts, docSummary)
		}
	}

	// Strategy 2: Extract node type and name
	if metadata.NodeType != "" && metadata.NodeName != "" {
		nodeSummary := h.extractNodeInfo(metadata.NodeType, metadata.NodeName, metadata.Signature)
		if nodeSummary != "" {
			parts = append(parts, nodeSummary)
		}
	}

	// Strategy 3: Extract first comment from code
	if comment := h.extractFirstComment(code); comment != "" {
		parts = append(parts, comment)
	}

	// Strategy 4: Identify key symbols
	if symbols := h.extractKeySymbols(code, metadata.Language); len(symbols) > 0 {
		symbolSummary := fmt.Sprintf("Uses: %s", strings.Join(symbols, ", "))
		parts = append(parts, symbolSummary)
	}

	// Build final summary
	summary := strings.Join(parts, ". ")

	// Trim to reasonable length
	if len(summary) > 200 {
		summary = summary[:200] + "..."
	}

	// Ensure we have at least some summary
	if summary == "" {
		summary = fmt.Sprintf("Code snippet in %s", metadata.Language)
	}

	h.logger.Debug("Generated heuristic summary",
		zap.String("file", metadata.FilePath),
		zap.String("node_type", metadata.NodeType),
		zap.Int("summary_length", len(summary)),
	)

	return summary
}

// extractDocumentation extracts meaningful summary from documentation
func (h *HeuristicEngine) extractDocumentation(docContent string) string {
	if docContent == "" {
		return ""
	}

	// Clean up documentation
	lines := strings.Split(docContent, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Remove comment markers
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "//")
		line = strings.TrimPrefix(line, "/*")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSuffix(line, "*/")
		line = strings.TrimSpace(line)

		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	if len(cleanLines) == 0 {
		return ""
	}

	// Take first meaningful line
	summary := cleanLines[0]
	if len(summary) > 100 {
		summary = summary[:100] + "..."
	}

	return summary
}

// extractNodeInfo extracts information about the node (function, class, etc.)
func (h *HeuristicEngine) extractNodeInfo(nodeType, nodeName, signature string) string {
	var parts []string

	// Determine article based on node type
	article := "a"
	if strings.HasPrefix(strings.ToLower(nodeType), "i") {
		article = "an"
	}

	parts = append(parts, fmt.Sprintf("Defines %s %s '%s'", article, nodeType, nodeName))

	// Add signature info if available
	if signature != "" {
		// Extract parameters from signature
		if params := h.extractParameters(signature); params != "" {
			parts = append(parts, fmt.Sprintf("Parameters: %s", params))
		}
	}

	return strings.Join(parts, ". ")
}

// extractParameters extracts parameter information from a function signature
func (h *HeuristicEngine) extractParameters(signature string) string {
	// Look for parameter patterns like (param1, param2, ...)
	re := regexp.MustCompile(`\((.*?)\)`)
	matches := re.FindStringSubmatch(signature)

	if len(matches) < 2 {
		return ""
	}

	params := matches[1]
	if params == "" {
		return "none"
	}

	// Limit to first 50 chars
	if len(params) > 50 {
		params = params[:50] + "..."
	}

	return params
}

// extractFirstComment extracts the first meaningful comment from code
func (h *HeuristicEngine) extractFirstComment(code string) string {
	lines := strings.Split(code, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Look for single-line comments
		if strings.HasPrefix(trimmed, "//") {
			comment := strings.TrimPrefix(trimmed, "//")
			comment = strings.TrimSpace(comment)
			if comment != "" && len(comment) > 5 {
				if len(comment) > 100 {
					comment = comment[:100] + "..."
				}
				return comment
			}
		}

		// Look for multi-line comment start
		if strings.HasPrefix(trimmed, "/*") {
			comment := strings.TrimPrefix(trimmed, "/*")
			comment = strings.TrimSuffix(comment, "*/")
			comment = strings.TrimSpace(comment)
			if comment != "" && len(comment) > 5 {
				if len(comment) > 100 {
					comment = comment[:100] + "..."
				}
				return comment
			}
		}

		// Look for docstring patterns (Python, Go)
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, "'''") {
			comment := strings.TrimPrefix(trimmed, `"""`)
			comment = strings.TrimPrefix(comment, "'''")
			comment = strings.TrimSuffix(comment, `"""`)
			comment = strings.TrimSuffix(comment, "'''")
			comment = strings.TrimSpace(comment)
			if comment != "" && len(comment) > 5 {
				if len(comment) > 100 {
					comment = comment[:100] + "..."
				}
				return comment
			}
		}

		// Stop after first non-comment, non-empty line
		if !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			break
		}
	}

	return ""
}

// extractKeySymbols identifies important symbols used in the code
func (h *HeuristicEngine) extractKeySymbols(code string, language string) []string {
	var symbols []string
	symbolMap := make(map[string]bool)

	// Language-specific patterns
	switch strings.ToLower(language) {
	case "go":
		symbols = h.extractGoSymbols(code, symbolMap)
	case "python":
		symbols = h.extractPythonSymbols(code, symbolMap)
	case "typescript", "javascript":
		symbols = h.extractTypeScriptSymbols(code, symbolMap)
	case "java":
		symbols = h.extractJavaSymbols(code, symbolMap)
	default:
		symbols = h.extractGenericSymbols(code, symbolMap)
	}

	// Limit to top 3 symbols
	if len(symbols) > 3 {
		symbols = symbols[:3]
	}

	return symbols
}

// extractGoSymbols extracts Go-specific symbols
func (h *HeuristicEngine) extractGoSymbols(code string, symbolMap map[string]bool) []string {
	var symbols []string

	// Look for common Go patterns
	patterns := []string{
		`\b(fmt|json|http|io|os|sync|context|errors)\b`,
		`\b(make|append|copy|len|cap|range)\b`,
		`\b(goroutine|channel|mutex|lock)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			if !symbolMap[match] {
				symbols = append(symbols, match)
				symbolMap[match] = true
			}
		}
	}

	return symbols
}

// extractPythonSymbols extracts Python-specific symbols
func (h *HeuristicEngine) extractPythonSymbols(code string, symbolMap map[string]bool) []string {
	var symbols []string

	// Look for common Python patterns
	patterns := []string{
		`\b(import|from|class|def|async|await)\b`,
		`\b(list|dict|set|tuple|str|int|float)\b`,
		`\b(try|except|finally|raise|assert)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			if !symbolMap[match] {
				symbols = append(symbols, match)
				symbolMap[match] = true
			}
		}
	}

	return symbols
}

// extractTypeScriptSymbols extracts TypeScript/JavaScript-specific symbols
func (h *HeuristicEngine) extractTypeScriptSymbols(code string, symbolMap map[string]bool) []string {
	var symbols []string

	// Look for common TypeScript/JavaScript patterns
	patterns := []string{
		`\b(import|export|async|await|Promise|async)\b`,
		`\b(class|interface|type|enum|const|let|var)\b`,
		`\b(try|catch|finally|throw|Error)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			if !symbolMap[match] {
				symbols = append(symbols, match)
				symbolMap[match] = true
			}
		}
	}

	return symbols
}

// extractJavaSymbols extracts Java-specific symbols
func (h *HeuristicEngine) extractJavaSymbols(code string, symbolMap map[string]bool) []string {
	var symbols []string

	// Look for common Java patterns
	patterns := []string{
		`\b(public|private|protected|static|final|abstract)\b`,
		`\b(class|interface|enum|extends|implements)\b`,
		`\b(try|catch|finally|throw|throws)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			if !symbolMap[match] {
				symbols = append(symbols, match)
				symbolMap[match] = true
			}
		}
	}

	return symbols
}

// extractGenericSymbols extracts generic symbols for unknown languages
func (h *HeuristicEngine) extractGenericSymbols(code string, symbolMap map[string]bool) []string {
	var symbols []string

	// Look for generic patterns
	patterns := []string{
		`\b(function|class|def|struct|interface)\b`,
		`\b(import|include|require|use)\b`,
		`\b(try|catch|error|exception)\b`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(code, -1)
		for _, match := range matches {
			if !symbolMap[match] {
				symbols = append(symbols, match)
				symbolMap[match] = true
			}
		}
	}

	return symbols
}

// QualityScore returns a quality score for a heuristic summary (0-1)
// Higher scores indicate better quality summaries
func (h *HeuristicEngine) QualityScore(summary string, code string, metadata CodeMetadata) float64 {
	score := 0.5 // Base score

	// Bonus for length (longer summaries tend to be more informative)
	if len(summary) > 50 {
		score += 0.1
	}
	if len(summary) > 100 {
		score += 0.1
	}

	// Bonus for having documentation
	if metadata.DocContent != "" {
		score += 0.1
	}

	// Bonus for having node name
	if metadata.NodeName != "" {
		score += 0.1
	}

	// Bonus for having signature
	if metadata.Signature != "" {
		score += 0.1
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}
