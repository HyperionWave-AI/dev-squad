package summarizer

import (
	"fmt"
	"strings"
	"time"
)

// PromptVersion represents the version of prompts being used
type PromptVersion string

const (
	PromptVersionV1 PromptVersion = "v1"
	PromptVersionV2 PromptVersion = "v2"
)

// PromptTemplate represents a single prompt template
type PromptTemplate struct {
	Name       string
	Version    PromptVersion
	Template   string
	MaxTokens  int
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Description string
}

// PromptBuilder handles building prompts for different purposes
type PromptBuilder struct {
	version PromptVersion
}

// NewPromptBuilder creates a new prompt builder with the specified version
func NewPromptBuilder(version PromptVersion) *PromptBuilder {
	if version == "" {
		version = PromptVersionV1
	}
	return &PromptBuilder{
		version: version,
	}
}

// CodeSummaryPromptV1 is the v1 prompt for code summarization
const CodeSummaryPromptV1 = `You are a code summarization expert. Your task is to create a concise, technical summary of the following code snippet.

REQUIREMENTS:
- Maximum 100 tokens
- Focus on WHAT the code does, not HOW
- Include key functions/classes mentioned
- Mention important patterns or libraries used
- Be specific and technical
- Use clear, professional language

CODE:
{code}

METADATA:
- File: {filePath}
- Type: {nodeType}
- Name: {nodeName}
- Language: {language}
{signature}

SUMMARY:`

// CodeSummaryPromptV2 is the v2 prompt for code summarization with improved structure
const CodeSummaryPromptV2 = `You are a code summarization expert. Create a concise, technical summary of this code.

INSTRUCTIONS:
1. Maximum 100 tokens
2. Start with the main purpose
3. List key functions/classes
4. Mention important patterns
5. Be specific and technical

CODE:
{code}

CONTEXT:
- File: {filePath}
- Type: {nodeType}
- Name: {nodeName}
- Language: {language}
{signature}

SUMMARY:`

// LanguageSpecificPrompts contains language-specific prompt variations
var LanguageSpecificPrompts = map[string]string{
	"go": `You are a Go code summarization expert. Summarize this Go code focusing on:
- Goroutines and concurrency patterns
- Error handling approach
- Package dependencies
- Interface implementations

CODE:
{code}

METADATA:
- File: {filePath}
- Type: {nodeType}
- Name: {nodeName}
{signature}

SUMMARY (max 100 tokens):`,

	"python": `You are a Python code summarization expert. Summarize this Python code focusing on:
- Classes and inheritance
- Decorators and metaclasses
- Async/await patterns
- Library dependencies

CODE:
{code}

METADATA:
- File: {filePath}
- Type: {nodeType}
- Name: {nodeName}
{signature}

SUMMARY (max 100 tokens):`,

	"typescript": `You are a TypeScript code summarization expert. Summarize this TypeScript code focusing on:
- Types and interfaces
- Async operations
- React patterns (if applicable)
- Library dependencies

CODE:
{code}

METADATA:
- File: {filePath}
- Type: {nodeType}
- Name: {nodeName}
{signature}

SUMMARY (max 100 tokens):`,

	"java": `You are a Java code summarization expert. Summarize this Java code focusing on:
- Classes and inheritance
- Design patterns
- Exception handling
- Library dependencies

CODE:
{code}

METADATA:
- File: {filePath}
- Type: {nodeType}
- Name: {nodeName}
{signature}

SUMMARY (max 100 tokens):`,
}

// MetadataExtractionPrompt is used to extract metadata from code
const MetadataExtractionPrompt = `Extract key metadata from this code snippet.

REQUIREMENTS:
- Main function/class name
- Primary purpose (1-2 sentences)
- Key dependencies (libraries, imports)
- Important patterns used

CODE:
{code}

METADATA:
NAME:
PURPOSE:
DEPENDENCIES:
PATTERNS:`

// QualityScoringPrompt is used to score the quality of a summary
const QualityScoringPrompt = `Rate the quality of this summary on a scale of 1-10.

CRITERIA:
- Accuracy: Does it correctly represent the code?
- Completeness: Does it include all key information?
- Clarity: Is it easy to understand?
- Conciseness: Is it appropriately brief?

SUMMARY:
{summary}

CODE:
{code}

SCORE (1-10):`

// promptRegistry stores all available prompts
var promptRegistry = map[string]*PromptTemplate{
	"code_summary_v1": {
		Name:        "code_summary_v1",
		Version:     PromptVersionV1,
		Template:    CodeSummaryPromptV1,
		MaxTokens:   100,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "V1 code summarization prompt - basic structure",
	},
	"code_summary_v2": {
		Name:        "code_summary_v2",
		Version:     PromptVersionV2,
		Template:    CodeSummaryPromptV2,
		MaxTokens:   100,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "V2 code summarization prompt - improved structure",
	},
	"metadata_extraction": {
		Name:        "metadata_extraction",
		Version:     PromptVersionV1,
		Template:    MetadataExtractionPrompt,
		MaxTokens:   200,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "Extract metadata from code",
	},
	"quality_scoring": {
		Name:        "quality_scoring",
		Version:     PromptVersionV1,
		Template:    QualityScoringPrompt,
		MaxTokens:   50,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Description: "Score the quality of a summary",
	},
}

// BuildCodeSummaryPrompt builds a code summarization prompt
func (pb *PromptBuilder) BuildCodeSummaryPrompt(code string, metadata CodeMetadata) string {
	template := pb.selectTemplate("code_summary")
	template = pb.selectLanguageSpecificTemplate(metadata.Language)
	
	values := map[string]string{
		"code":       code,
		"filePath":   metadata.FilePath,
		"nodeType":   metadata.NodeType,
		"nodeName":   metadata.NodeName,
		"language":   metadata.Language,
		"signature":  formatSignature(metadata.Signature),
	}
	
	return pb.buildPrompt(template, values)
}

// BuildMetadataExtractionPrompt builds a metadata extraction prompt
func (pb *PromptBuilder) BuildMetadataExtractionPrompt(code string) string {
	values := map[string]string{
		"code": code,
	}
	
	return pb.buildPrompt(MetadataExtractionPrompt, values)
}

// BuildQualityScoringPrompt builds a quality scoring prompt
func (pb *PromptBuilder) BuildQualityScoringPrompt(summary, code string) string {
	values := map[string]string{
		"summary": summary,
		"code":    code,
	}
	
	return pb.buildPrompt(QualityScoringPrompt, values)
}

// selectTemplate selects the appropriate template based on version
func (pb *PromptBuilder) selectTemplate(templateType string) string {
	switch templateType {
	case "code_summary":
		if pb.version == PromptVersionV2 {
			return CodeSummaryPromptV2
		}
		return CodeSummaryPromptV1
	case "metadata_extraction":
		return MetadataExtractionPrompt
	case "quality_scoring":
		return QualityScoringPrompt
	default:
		return CodeSummaryPromptV1
	}
}

// selectLanguageSpecificTemplate selects language-specific prompt if available
func (pb *PromptBuilder) selectLanguageSpecificTemplate(language string) string {
	if prompt, exists := LanguageSpecificPrompts[strings.ToLower(language)]; exists {
		return prompt
	}
	// Return default template with generic REQUIREMENTS
	return pb.selectTemplate("code_summary")
}

// buildPrompt replaces placeholders in a template with actual values
func (pb *PromptBuilder) buildPrompt(template string, values map[string]string) string {
	result := template
	for key, value := range values {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	// Clean up any remaining placeholders
	result = cleanupPlaceholders(result)
	
	return result
}

// cleanupPlaceholders removes any remaining unresolved placeholders
func cleanupPlaceholders(text string) string {
	lines := strings.Split(text, "\n")
	var result []string
	
	for _, line := range lines {
		// Skip lines that are just placeholders
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
			continue
		}
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// formatSignature formats a function/class signature for display
func formatSignature(signature string) string {
	if signature == "" {
		return ""
	}
	
	if len(signature) > 100 {
		return "- Signature: " + signature[:100] + "..."
	}
	
	return "- Signature: " + signature
}

// GetPromptTemplate retrieves a prompt template by name
func GetPromptTemplate(name string) (*PromptTemplate, error) {
	template, exists := promptRegistry[name]
	if !exists {
		return nil, fmt.Errorf("prompt template not found: %s", name)
	}
	return template, nil
}

// ListPromptTemplates returns all available prompt templates
func ListPromptTemplates() []*PromptTemplate {
	var templates []*PromptTemplate
	for _, template := range promptRegistry {
		templates = append(templates, template)
	}
	return templates
}

// GetPromptsByVersion returns all prompts for a specific version
func GetPromptsByVersion(version PromptVersion) []*PromptTemplate {
	var templates []*PromptTemplate
	for _, template := range promptRegistry {
		if template.Version == version {
			templates = append(templates, template)
		}
	}
	return templates
}

// ValidatePrompt validates that a prompt is well-formed
func ValidatePrompt(prompt string) error {
	if prompt == "" {
		return fmt.Errorf("prompt is empty")
	}
	
	if len(prompt) < 50 {
		return fmt.Errorf("prompt is too short (< 50 chars)")
	}
	
	// Check for unresolved placeholders
	if strings.Contains(prompt, "{") && strings.Contains(prompt, "}") {
		// Check if there are actual unresolved placeholders
		lines := strings.Split(prompt, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
				return fmt.Errorf("unresolved placeholder in prompt: %s", trimmed)
			}
		}
	}
	
	return nil
}
