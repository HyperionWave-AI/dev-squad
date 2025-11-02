package review

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	aiservice "hyper/internal/ai-service"
)

// CompactionEngine handles intelligent summarization of knowledge entries
type CompactionEngine struct {
	llmClient     *ClaudeAPIClient
	targetWords   int
	temperature   float64
	model         string
	maxTokens     int
	dryRunEnabled bool
}

// CriticalElements holds important elements that must be preserved during compaction
type CriticalElements struct {
	FilePaths    []string
	Functions    []string
	Errors       []string
	Commands     []string
	GitCommits   []string
	LineRefs     []string
}

// CompactionResult holds the result of compaction
type CompactionResult struct {
	OriginalText    string
	CompactedText   string
	OriginalWords   int
	CompactedWords  int
	PreservedAll    bool
	MissingElements []string
	DryRun          bool
}

// NewCompactionEngine creates a new compaction engine using AIConfig
func NewCompactionEngine(aiConfig *aiservice.AIConfig) (*CompactionEngine, error) {
	if aiConfig == nil {
		return nil, fmt.Errorf("AIConfig is required for compaction engine")
	}

	llmClient, err := NewClaudeAPIClient(aiConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	targetWords := 300
	if tw := os.Getenv("COMPACTION_TARGET_WORDS"); tw != "" {
		if parsed, err := strconv.Atoi(tw); err == nil {
			targetWords = parsed
		}
	}

	// Use configured temperature from AIConfig, with optional override
	temperature := aiConfig.Temperature
	if temp := os.Getenv("COMPACTION_TEMPERATURE"); temp != "" {
		if parsed, err := strconv.ParseFloat(temp, 64); err == nil {
			temperature = parsed
		}
	}

	// Use configured model from AIConfig
	model := aiConfig.Model
	if model == "" {
		model = "claude-sonnet-4-5-20250929" // fallback
	}

	return &CompactionEngine{
		llmClient:   llmClient,
		targetWords: targetWords,
		temperature: temperature,
		model:       model,
		maxTokens:   1500,
	}, nil
}

// extractCriticalElements identifies important elements that must be preserved
func (ce *CompactionEngine) extractCriticalElements(text string) *CriticalElements {
	elements := &CriticalElements{
		FilePaths:  []string{},
		Functions:  []string{},
		Errors:     []string{},
		Commands:   []string{},
		GitCommits: []string{},
		LineRefs:   []string{},
	}

	// Extract file paths (e.g., "path/to/file.go", "file.js:123")
	filePathRegex := regexp.MustCompile(`(?:^|\s)([a-zA-Z0-9_\-./]+\.[a-zA-Z0-9]+(?::[0-9]+)?)(?:\s|$|,|;|\))`)
	fileMatches := filePathRegex.FindAllStringSubmatch(text, -1)
	for _, match := range fileMatches {
		if len(match) > 1 {
			elements.FilePaths = append(elements.FilePaths, match[1])
		}
	}

	// Extract line references (e.g., "file.go:123", "line 456")
	lineRefRegex := regexp.MustCompile(`(?:[a-zA-Z0-9_\-./]+\.go|line)\s*:?\s*([0-9]+)`)
	lineMatches := lineRefRegex.FindAllStringSubmatch(text, -1)
	for _, match := range lineMatches {
		if len(match) > 1 {
			elements.LineRefs = append(elements.LineRefs, match[0])
		}
	}

	// Extract function names (e.g., "functionName()", "method.Call()")
	functionRegex := regexp.MustCompile(`(?:func\s+)?([a-zA-Z_][a-zA-Z0-9_]*(?:\.[a-zA-Z_][a-zA-Z0-9_]*)?)\s*\(`)
	funcMatches := functionRegex.FindAllStringSubmatch(text, -1)
	for _, match := range funcMatches {
		if len(match) > 1 {
			elements.Functions = append(elements.Functions, match[1])
		}
	}

	// Extract error messages (quoted strings with "error", "failed", "panic")
	errorRegex := regexp.MustCompile(`"[^"]*(?:error|failed|panic|fatal)[^"]*"`)
	errorMatches := errorRegex.FindAllString(text, -1)
	elements.Errors = append(elements.Errors, errorMatches...)

	// Extract shell commands (commands in backticks or "Run: command" or "Run command")
	commandRegex := regexp.MustCompile("`([^`]+)`")
	commandMatches := commandRegex.FindAllStringSubmatch(text, -1)
	for _, match := range commandMatches {
		if len(match) > 1 {
			elements.Commands = append(elements.Commands, match[1])
		}
	}

	// Also extract commands after "Run:" or "Run " (case insensitive) - stops at common word boundaries
	runCommandRegex := regexp.MustCompile(`(?i)(?:run:\s*|run\s+)([a-zA-Z0-9_\-./]+(?:\s+[a-zA-Z0-9_\-./=]+)*)(?:\s+(?:to|and|or|for|with|in|on|at|from)\s|\s*$|\.|\n)`)
	runMatches := runCommandRegex.FindAllStringSubmatch(text, -1)
	for _, match := range runMatches {
		if len(match) > 1 {
			elements.Commands = append(elements.Commands, strings.TrimSpace(match[1]))
		}
	}

	// Extract git commits (SHA hashes)
	gitCommitRegex := regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)
	gitMatches := gitCommitRegex.FindAllString(text, -1)
	elements.GitCommits = append(elements.GitCommits, gitMatches...)

	return elements
}

// buildCompactionPrompt creates a structured prompt for the LLM
func (ce *CompactionEngine) buildCompactionPrompt(text string, targetWords int) string {
	// Extract critical elements to emphasize preservation
	elements := ce.extractCriticalElements(text)

	preservationList := []string{}
	if len(elements.FilePaths) > 0 {
		preservationList = append(preservationList, fmt.Sprintf("File paths: %s", strings.Join(elements.FilePaths[:min(5, len(elements.FilePaths))], ", ")))
	}
	if len(elements.Functions) > 0 {
		preservationList = append(preservationList, fmt.Sprintf("Functions: %s", strings.Join(elements.Functions[:min(5, len(elements.Functions))], ", ")))
	}
	if len(elements.Commands) > 0 {
		preservationList = append(preservationList, fmt.Sprintf("Commands: %s", strings.Join(elements.Commands[:min(3, len(elements.Commands))], ", ")))
	}

	preservationNote := ""
	if len(preservationList) > 0 {
		preservationNote = fmt.Sprintf("\n\nCritical elements detected (MUST preserve exactly):\n%s", strings.Join(preservationList, "\n"))
	}

	prompt := fmt.Sprintf(`You are a technical knowledge compaction expert. Your task is to compress the following technical knowledge entry into EXACTLY %d words (±10 words acceptable) while preserving ALL critical information.

MANDATORY STRUCTURE (use these exact headings):
1. PROBLEM (50 words): What issue/question was being addressed?
2. SOLUTION (100 words): How was it solved? Key implementation details.
3. KEY FILES (50 words): Which files/functions were modified? Be specific.
4. GOTCHAS (50 words): Edge cases, errors encountered, important warnings.
5. TESTING (50 words): How to verify/test this? Commands, expected results.

PRESERVATION RULES (CRITICAL):
- Preserve ALL file paths exactly (e.g., "hyper/internal/mcp/storage/knowledge.go")
- Preserve ALL function names exactly (e.g., "CompactEntry()", "extractCriticalElements")
- Preserve ALL line references exactly (e.g., "line 123", "file.go:456")
- Preserve ALL error messages exactly (in quotes)
- Preserve ALL commands exactly (e.g., "make test", "kubectl apply")
- Preserve ALL git commit hashes exactly
- Use technical terminology precisely%s

ORIGINAL TEXT:
%s

Provide ONLY the compacted text following the 5-section structure. No preamble, no meta-commentary.`, targetWords, preservationNote, text)

	return prompt
}

// verifyPreservedElements checks that critical elements were preserved in compacted text
func (ce *CompactionEngine) verifyPreservedElements(original string, compacted string) (bool, []string) {
	originalElements := ce.extractCriticalElements(original)
	compactedElements := ce.extractCriticalElements(compacted)

	missing := []string{}

	// Check file paths
	compactedFilePaths := make(map[string]bool)
	for _, fp := range compactedElements.FilePaths {
		compactedFilePaths[fp] = true
	}
	for _, fp := range originalElements.FilePaths {
		if !compactedFilePaths[fp] {
			missing = append(missing, fmt.Sprintf("file path: %s", fp))
		}
	}

	// Check functions (allow some loss if there are many)
	if len(originalElements.Functions) <= 10 {
		compactedFunctions := make(map[string]bool)
		for _, fn := range compactedElements.Functions {
			compactedFunctions[fn] = true
		}
		for _, fn := range originalElements.Functions {
			if !compactedFunctions[fn] {
				missing = append(missing, fmt.Sprintf("function: %s", fn))
			}
		}
	}

	// Check error messages
	compactedErrors := make(map[string]bool)
	for _, err := range compactedElements.Errors {
		compactedErrors[err] = true
	}
	for _, err := range originalElements.Errors {
		if !compactedErrors[err] {
			missing = append(missing, fmt.Sprintf("error: %s", err))
		}
	}

	// Check commands
	compactedCommands := make(map[string]bool)
	for _, cmd := range compactedElements.Commands {
		compactedCommands[cmd] = true
	}
	for _, cmd := range originalElements.Commands {
		if !compactedCommands[cmd] {
			missing = append(missing, fmt.Sprintf("command: %s", cmd))
		}
	}

	return len(missing) == 0, missing
}

// countWords counts the number of words in a text
func (ce *CompactionEngine) countWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

// CompactEntry compresses a knowledge entry using the LLM
func (ce *CompactionEngine) CompactEntry(ctx context.Context, entryText string, dryRun bool) (*CompactionResult, error) {
	result := &CompactionResult{
		OriginalText:  entryText,
		OriginalWords: ce.countWords(entryText),
		DryRun:        dryRun,
	}

	// Check if compaction is needed
	if result.OriginalWords <= ce.targetWords {
		result.CompactedText = entryText
		result.CompactedWords = result.OriginalWords
		result.PreservedAll = true
		return result, nil
	}

	// Build compaction prompt
	prompt := ce.buildCompactionPrompt(entryText, ce.targetWords)

	if dryRun {
		result.CompactedText = fmt.Sprintf("[DRY RUN] Would compact %d words to ~%d words using Claude API", result.OriginalWords, ce.targetWords)
		result.CompactedWords = ce.targetWords
		result.PreservedAll = true
		return result, nil
	}

	// Call LLM
	compactedText, err := ce.llmClient.SendMessage(ctx, prompt, ce.model, ce.temperature, ce.maxTokens)
	if err != nil {
		return nil, fmt.Errorf("LLM compaction failed: %w", err)
	}

	result.CompactedText = strings.TrimSpace(compactedText)
	result.CompactedWords = ce.countWords(result.CompactedText)

	// Validate word count (250-350 words acceptable for 300 target)
	minWords := ce.targetWords - 50
	maxWords := ce.targetWords + 50
	if result.CompactedWords < minWords || result.CompactedWords > maxWords {
		return nil, fmt.Errorf("compacted text has %d words, expected %d±50", result.CompactedWords, ce.targetWords)
	}

	// Verify critical elements preserved
	preserved, missing := ce.verifyPreservedElements(entryText, result.CompactedText)
	result.PreservedAll = preserved
	result.MissingElements = missing

	if !preserved {
		return nil, fmt.Errorf("compaction lost critical elements: %s", strings.Join(missing, ", "))
	}

	return result, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
