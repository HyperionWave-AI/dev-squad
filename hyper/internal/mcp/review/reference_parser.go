package review

import (
	"fmt"
	"regexp"
	"strings"
)

// Regular expression patterns for different reference types
var (
	// File:line patterns: file.go:123 or file.go (line 123) or file.go (lines 50-75)
	fileLinePattern = regexp.MustCompile(`([a-zA-Z0-9_\-./]+\.(go|ts|tsx|js|jsx|py|java|rs|md|yaml|yml|json)):(\d+)`)
	fileLinesPattern = regexp.MustCompile(`([a-zA-Z0-9_\-./]+\.(go|ts|tsx|js|jsx|py|java|rs|md|yaml|yml|json))\s*\(lines?\s+(\d+)(?:-(\d+))?\)`)

	// Function patterns: FunctionName() or Type.Method() or package.FunctionName() or handleSearch()
	// Matches both uppercase and lowercase starting function names
	functionPattern = regexp.MustCompile(`\b([a-zA-Z][a-zA-Z0-9]*(?:\.[a-zA-Z][a-zA-Z0-9]*)?)\s*\(`)

	// Git commit patterns: 7-40 character hex strings (allow up to 41 for edge cases)
	// No trailing word boundary to match hashes at end of string
	commitPattern = regexp.MustCompile(`\b([0-9a-f]{7,})`)

	// API endpoint patterns: HTTP method + path
	apiPattern = regexp.MustCompile(`\b(GET|POST|PUT|DELETE|PATCH)\s+(/[a-zA-Z0-9/_:\-]+)`)

	// File path patterns: relative or absolute paths (without line numbers)
	filePathPattern = regexp.MustCompile(`\b([a-zA-Z0-9_\-]+/[a-zA-Z0-9_\-./]+\.(go|ts|tsx|js|jsx|py|java|rs|md|yaml|yml|json))\b`)
)

// parseReferences extracts all references from knowledge entry text
// Returns a slice of Reference objects with Type, Value, and Context populated
func parseReferences(text string) []Reference {
	var refs []Reference
	seen := make(map[string]bool) // Deduplicate references

	// 1. Extract file:line references (highest priority)
	matches := fileLinePattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		filePath := text[match[2]:match[3]]
		lineNum := text[match[6]:match[7]]
		value := fmt.Sprintf("%s:%s", filePath, lineNum)

		if !seen[value] {
			refs = append(refs, Reference{
				Type:    ReferenceTypeFileLine,
				Value:   value,
				Context: extractContext(text, match[0], match[1]),
			})
			seen[value] = true
		}
	}

	// 2. Extract file (lines X-Y) references
	matches = fileLinesPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		filePath := text[match[2]:match[3]]
		startLine := text[match[6]:match[7]]
		var endLine string
		if match[8] != -1 && match[9] != -1 {
			endLine = text[match[8]:match[9]]
		} else {
			endLine = startLine
		}
		value := fmt.Sprintf("%s:%s-%s", filePath, startLine, endLine)

		if !seen[value] {
			refs = append(refs, Reference{
				Type:    ReferenceTypeFileLine,
				Value:   value,
				Context: extractContext(text, match[0], match[1]),
			})
			seen[value] = true
		}
	}

	// 3. Extract function references
	matches = functionPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		funcName := text[match[2]:match[3]]

		// Filter out file extensions (e.g., "mcp.go" should not be treated as a function)
		if strings.HasSuffix(funcName, ".go") || strings.HasSuffix(funcName, ".js") ||
		   strings.HasSuffix(funcName, ".ts") || strings.HasSuffix(funcName, ".py") ||
		   strings.HasSuffix(funcName, ".tsx") || strings.HasSuffix(funcName, ".jsx") {
			continue
		}

		// Filter out common false positives (keywords, common words)
		if isLikelyFunctionName(funcName) {
			// Use funcName with method call syntax as key to avoid duplicates
			key := "func:" + funcName
			if !seen[key] {
				refs = append(refs, Reference{
					Type:    ReferenceTypeFunction,
					Value:   funcName,
					Context: extractContext(text, match[0], match[1]),
				})
				seen[key] = true
			}
		}
	}

	// 4. Extract commit hashes (with context validation)
	matches = commitPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		commitHash := text[match[2]:match[3]]

		// Only consider as commit if it appears in commit reference patterns
		// Look for patterns like "commit <hash>", "SHA <hash>", "hash <hash>"
		// Get context before the hash (up to 30 chars)
		contextStart := match[0] - 30
		if contextStart < 0 {
			contextStart = 0
		}
		beforeContext := strings.ToLower(text[contextStart:match[0]])

		// Check if there's a commit keyword immediately before the hash (with optional whitespace)
		// Patterns: "commit <hash>", "SHA: <hash>", "hash: <hash>", etc.
		hasCommitKeyword := regexp.MustCompile(`(?:commit|sha|hash)\s*:?\s*$`).MatchString(beforeContext)

		if hasCommitKeyword && !seen[commitHash] {
			refs = append(refs, Reference{
				Type:    ReferenceTypeCommit,
				Value:   commitHash,
				Context: extractContext(text, match[0], match[1]),
			})
			seen[commitHash] = true
		}
	}

	// 5. Extract API endpoints
	matches = apiPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		method := text[match[2]:match[3]]
		path := text[match[4]:match[5]]
		value := fmt.Sprintf("%s %s", method, path)

		if !seen[value] {
			refs = append(refs, Reference{
				Type:    ReferenceTypeAPI,
				Value:   value,
				Context: extractContext(text, match[0], match[1]),
			})
			seen[value] = true
		}
	}

	// 6. Extract file paths (lowest priority, exclude those already found as file:line)
	matches = filePathPattern.FindAllStringSubmatchIndex(text, -1)
	for _, match := range matches {
		filePath := text[match[2]:match[3]]

		// Skip if already found as file:line reference
		alreadyFound := false
		for _, ref := range refs {
			if ref.Type == ReferenceTypeFileLine && strings.HasPrefix(ref.Value, filePath+":") {
				alreadyFound = true
				break
			}
		}

		if !alreadyFound && !seen[filePath] {
			refs = append(refs, Reference{
				Type:    ReferenceTypeFile,
				Value:   filePath,
				Context: extractContext(text, match[0], match[1]),
			})
			seen[filePath] = true
		}
	}

	return refs
}

// extractContext extracts surrounding text (up to 50 chars before and after)
func extractContext(text string, start, end int) string {
	contextStart := start - 50
	if contextStart < 0 {
		contextStart = 0
	}

	contextEnd := end + 50
	if contextEnd > len(text) {
		contextEnd = len(text)
	}

	return strings.TrimSpace(text[contextStart:contextEnd])
}

// isLikelyFunctionName filters out common false positives
func isLikelyFunctionName(name string) bool {
	// Filter out common false positives (both upper and lowercase)
	falsePositives := map[string]bool{
		"If": true, "For": true, "While": true, "Switch": true,
		"Type": true, "This": true, "The": true, "When": true,
		"Error": true, "String": true, "Int": true, "Bool": true,
		"Map": true, "Array": true, "List": true, "Set": true,
		"if": true, "for": true, "while": true, "switch": true,
		"new": true, "return": true, "case": true, "break": true,
	}

	// Check if it's a false positive
	if falsePositives[name] {
		return false
	}

	// Must be at least 2 characters
	if len(name) < 2 {
		return false
	}

	// Check if it has mixed case or is a reasonable length lowercase name
	// This handles both HandleSearch (uppercase) and handleSearch (lowercase)
	hasUppercase := false
	hasLowercase := false
	for _, ch := range name {
		if ch >= 'A' && ch <= 'Z' {
			hasUppercase = true
		}
		if ch >= 'a' && ch <= 'z' {
			hasLowercase = true
		}
	}

	// Accept if:
	// 1. Has both upper and lowercase (e.g., HandleSearch, handleSearch)
	// 2. Is all lowercase but longer than 3 chars (e.g., handleSearch is reasonable, but "if" isn't)
	// 3. Starts with uppercase and has lowercase (traditional Go naming)
	if (hasUppercase && hasLowercase) || (hasLowercase && !hasUppercase && len(name) > 3) {
		return true
	}

	// Reject all-uppercase acronyms like "HTTP"
	return false
}
