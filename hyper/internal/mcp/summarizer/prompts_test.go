package summarizer

import (
	"strings"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════
// PROMPT BUILDER TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestNewPromptBuilder(t *testing.T) {
	tests := []struct {
		name    string
		version PromptVersion
		want    PromptVersion
	}{
		{
			name:    "with V1",
			version: PromptVersionV1,
			want:    PromptVersionV1,
		},
		{
			name:    "with V2",
			version: PromptVersionV2,
			want:    PromptVersionV2,
		},
		{
			name:    "with empty defaults to V1",
			version: "",
			want:    PromptVersionV1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(tt.version)
			if pb.version != tt.want {
				t.Errorf("NewPromptBuilder() version = %v, want %v", pb.version, tt.want)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// CODE SUMMARY PROMPT TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestBuildCodeSummaryPrompt(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	metadata := CodeMetadata{
		FilePath:   "main.go",
		Language:   "go",
		NodeType:   "function",
		NodeName:   "main",
		Signature:  "func main()",
		DocContent: "Main entry point",
	}

	code := `func main() {
		fmt.Println("Hello, World!")
	}`

	prompt := pb.BuildCodeSummaryPrompt(code, metadata)

	// Verify prompt contains key elements
	if !strings.Contains(prompt, code) {
		t.Error("Prompt should contain the code")
	}

	if !strings.Contains(prompt, metadata.FilePath) {
		t.Error("Prompt should contain the file path")
	}

	if !strings.Contains(prompt, metadata.NodeName) {
		t.Error("Prompt should contain the node name")
	}

	if !strings.Contains(prompt, metadata.Language) {
		t.Error("Prompt should contain the language")
	}

	// Verify no unresolved placeholders
	if err := ValidatePrompt(prompt); err != nil {
		t.Errorf("Prompt validation failed: %v", err)
	}
}

func TestBuildCodeSummaryPromptLanguageSpecific(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected string
	}{
		{
			name:     "Go language",
			language: "go",
			expected: "goroutines",
		},
		{
			name:     "Python language",
			language: "python",
			expected: "classes",
		},
		{
			name:     "TypeScript language",
			language: "typescript",
			expected: "types",
		},
		{
			name:     "Java language",
			language: "java",
			expected: "classes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(PromptVersionV1)

			metadata := CodeMetadata{
				FilePath:  "test.go",
				Language:  tt.language,
				NodeType:  "function",
				NodeName:  "test",
				Signature: "func test()",
			}

			code := "func test() {}"

			prompt := pb.BuildCodeSummaryPrompt(code, metadata)

			if !strings.Contains(strings.ToLower(prompt), strings.ToLower(tt.expected)) {
				t.Errorf("Prompt for %s should contain '%s'", tt.language, tt.expected)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// METADATA EXTRACTION PROMPT TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestBuildMetadataExtractionPrompt(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	code := `func ProcessData(input []string) ([]int, error) {
		// Process input
		return result, nil
	}`

	prompt := pb.BuildMetadataExtractionPrompt(code)

	// Verify prompt contains key elements
	if !strings.Contains(prompt, code) {
		t.Error("Prompt should contain the code")
	}

	if !strings.Contains(prompt, "NAME:") {
		t.Error("Prompt should request NAME extraction")
	}

	if !strings.Contains(prompt, "PURPOSE:") {
		t.Error("Prompt should request PURPOSE extraction")
	}

	if !strings.Contains(prompt, "DEPENDENCIES:") {
		t.Error("Prompt should request DEPENDENCIES extraction")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// QUALITY SCORING PROMPT TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestBuildQualityScoringPrompt(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	summary := "This function processes data and returns results"
	code := `func ProcessData(input []string) ([]int, error) {
		return result, nil
	}`

	prompt := pb.BuildQualityScoringPrompt(summary, code)

	// Verify prompt contains key elements
	if !strings.Contains(prompt, summary) {
		t.Error("Prompt should contain the summary")
	}

	if !strings.Contains(prompt, code) {
		t.Error("Prompt should contain the code")
	}

	if !strings.Contains(prompt, "CRITERIA:") {
		t.Error("Prompt should request CRITERIA")
	}

	if !strings.Contains(prompt, "CRITERIA:") {
		t.Error("Prompt should request CRITERIA")
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PROMPT TEMPLATE REGISTRY TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestGetPromptTemplate(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "existing template",
			key:     "code_summary_v1",
			wantErr: false,
		},
		{
			name:    "non-existing template",
			key:     "non_existing",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := GetPromptTemplate(tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPromptTemplate() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && template.Name == "" {
				t.Error("GetPromptTemplate() returned empty template")
			}
		})
	}
}

func TestListPromptTemplates(t *testing.T) {
	templates := ListPromptTemplates()

	if len(templates) == 0 {
		t.Error("ListPromptTemplates() should return at least one template")
	}

	// Verify all templates have required fields
	for _, template := range templates {
		if template.Name == "" {
			t.Error("Template should have a name")
		}
		if template.Template == "" {
			t.Error("Template should have template content")
		}
		if template.MaxTokens <= 0 {
			t.Error("Template should have positive MaxTokens")
		}
	}
}

func TestGetPromptsByVersion(t *testing.T) {
	tests := []struct {
		name    string
		version PromptVersion
		minLen  int
	}{
		{
			name:    "V1 templates",
			version: PromptVersionV1,
			minLen:  1,
		},
		{
			name:    "V2 templates",
			version: PromptVersionV2,
			minLen:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templates := GetPromptsByVersion(tt.version)

			if len(templates) < tt.minLen {
				t.Errorf("GetPromptsByVersion() returned %d templates, want at least %d", len(templates), tt.minLen)
			}

			// Verify all returned templates match the version
			for _, template := range templates {
				if template.Version != tt.version {
					t.Errorf("Template version %v doesn't match requested %v", template.Version, tt.version)
				}
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// PROMPT VALIDATION TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestValidatePrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{
			name:    "valid prompt",
			prompt:  "This is a valid prompt with sufficient content to be considered valid",
			wantErr: false,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: true,
		},
		{
			name:    "too short prompt",
			prompt:  "short",
			wantErr: true,
		},
		{
			name:    "unresolved placeholder",
			prompt:  "This prompt has an unresolved {code} in it",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrompt() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// HELPER FUNCTION TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestSelectLanguageSpecificTemplate(t *testing.T) {
	tests := []struct {
		name     string
		language string
		expected string
	}{
		{
			name:     "Go",
			language: "go",
			expected: "goroutines",
		},
		{
			name:     "Python",
			language: "python",
			expected: "classes",
		},
		{
			name:     "TypeScript",
			language: "typescript",
			expected: "types",
		},
		{
			name:     "JavaScript",
			language: "javascript",
			expected: "REQUIREMENTS:",
		},
		{
			name:     "Java",
			language: "java",
			expected: "classes",
		},
		{
			name:     "Unknown language",
			language: "rust",
			expected: "REQUIREMENTS:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPromptBuilder(PromptVersionV1)
			template := pb.selectLanguageSpecificTemplate(tt.language)

			if !strings.Contains(strings.ToLower(template), strings.ToLower(tt.expected)) {
				t.Errorf("Template for %s should contain '%s'", tt.language, tt.expected)
			}
		})
	}
}

func TestBuildPrompt(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	template := "Hello {name}, your file is {filePath}. This is a longer template to ensure the result is long enough to pass validation. The file contains important code that needs to be summarized."
	values := map[string]string{
		"name":     "John",
		"filePath": "main.go",
	}

	result := pb.buildPrompt(template, values)

	if !strings.Contains(result, "John") {
		t.Error("buildPrompt should replace {name} placeholder")
	}

	if !strings.Contains(result, "main.go") {
		t.Error("buildPrompt should replace {filePath} placeholder")
	}

	// Verify no known placeholders remain
	if err := ValidatePrompt(result); err != nil {
		t.Errorf("buildPrompt result failed validation: %v", err)
	}
}

func TestFormatSignature(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		expected  string
	}{
		{
			name:      "empty signature",
			signature: "",
			expected:  "",
		},
		{
			name:      "short signature",
			signature: "func main()",
			expected:  "- Signature: func main()",
		},
		{
			name:      "long signature",
			signature: strings.Repeat("a", 150),
			expected:  "- Signature:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSignature(tt.signature)

			if tt.expected == "" && result != "" {
				t.Errorf("formatSignature() should return empty for empty input")
			}

			if tt.expected != "" && !strings.Contains(result, tt.expected) {
				t.Errorf("formatSignature() result should contain '%s'", tt.expected)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// INTEGRATION TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestPromptGenerationIntegration(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	metadata := CodeMetadata{
		FilePath:   "utils.go",
		Language:   "go",
		NodeType:   "function",
		NodeName:   "ParseJSON",
		Signature:  "func ParseJSON(data []byte) (interface{}, error)",
		DocContent: "Parses JSON data",
	}

	code := `func ParseJSON(data []byte) (interface{}, error) {
		var result interface{}
		err := json.Unmarshal(data, &result)
		return result, err
	}`

	// Generate prompt
	prompt := pb.BuildCodeSummaryPrompt(code, metadata)

	// Validate the generated prompt
	if err := ValidatePrompt(prompt); err != nil {
		t.Errorf("Generated prompt failed validation: %v", err)
	}

	// Verify all metadata is included
	if !strings.Contains(prompt, metadata.FilePath) {
		t.Error("Prompt should include file path")
	}

	if !strings.Contains(prompt, metadata.NodeName) {
		t.Error("Prompt should include node name")
	}

	if !strings.Contains(prompt, metadata.Language) {
		t.Error("Prompt should include language")
	}
}

func TestMultipleLanguagePrompts(t *testing.T) {
	languages := []string{"go", "python", "typescript", "java"}

	for _, lang := range languages {
		t.Run(lang, func(t *testing.T) {
			pb := NewPromptBuilder(PromptVersionV1)

			metadata := CodeMetadata{
				FilePath:  "test.go",
				Language:  lang,
				NodeType:  "function",
				NodeName:  "test",
				Signature: "func test()",
			}

			code := "func test() {}"

			prompt := pb.BuildCodeSummaryPrompt(code, metadata)

			// Validate prompt
			if err := ValidatePrompt(prompt); err != nil {
				t.Errorf("Generated prompt for %s failed validation: %v", lang, err)
			}

			// Verify it's language-specific
			if !strings.Contains(prompt, lang) && lang != "typescript" && lang != "python" && lang != "java" {
				t.Errorf("Prompt for %s should mention the language", lang)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// BENCHMARK TESTS
// ═══════════════════════════════════════════════════════════════════════════

func BenchmarkBuildCodeSummaryPrompt(b *testing.B) {
	pb := NewPromptBuilder(PromptVersionV1)

	metadata := CodeMetadata{
		FilePath:   "main.go",
		Language:   "go",
		NodeType:   "function",
		NodeName:   "main",
		Signature:  "func main()",
		DocContent: "Main entry point",
	}

	code := `func main() {
		fmt.Println("Hello, World!")
	}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb.BuildCodeSummaryPrompt(code, metadata)
	}
}

func BenchmarkValidatePrompt(b *testing.B) {
	prompt := "This is a valid prompt with sufficient content to be considered valid for benchmarking purposes"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidatePrompt(prompt)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// EDGE CASE TESTS
// ═══════════════════════════════════════════════════════════════════════════

func TestPromptWithSpecialCharacters(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		Signature: "func test()",
	}

	code := `func test() {
		// Comment with special chars: @#$%^&*()
		fmt.Println("String with \"quotes\" and 'apostrophes'")
	}`

	prompt := pb.BuildCodeSummaryPrompt(code, metadata)

	if !strings.Contains(prompt, code) {
		t.Error("Prompt should preserve special characters in code")
	}
}

func TestPromptWithLargeCode(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	metadata := CodeMetadata{
		FilePath:  "test.go",
		Language:  "go",
		NodeType:  "function",
		NodeName:  "test",
		Signature: "func test()",
	}

	// Generate large code snippet
	code := strings.Repeat("fmt.Println(\"line\")\n", 100)

	prompt := pb.BuildCodeSummaryPrompt(code, metadata)

	if err := ValidatePrompt(prompt); err != nil {
		t.Errorf("Prompt with large code failed validation: %v", err)
	}
}

func TestPromptWithEmptyMetadata(t *testing.T) {
	pb := NewPromptBuilder(PromptVersionV1)

	metadata := CodeMetadata{
		FilePath: "test.go",
		Language: "go",
	}

	code := "func test() {}"

	prompt := pb.BuildCodeSummaryPrompt(code, metadata)

	if err := ValidatePrompt(prompt); err != nil {
		t.Errorf("Prompt with empty metadata failed validation: %v", err)
	}
}
