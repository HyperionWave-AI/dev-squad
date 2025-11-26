package handlers

import (
	"testing"

	"go.uber.org/zap"

	"hyper/internal/mcp/storage"
)

// TestApplyResponseModeSummary tests that summary mode removes content
func TestApplyResponseModeSummary(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	handler := &CodeToolsHandler{logger: logger}

	results := []storage.SearchResult{
		{
			FilePath:  "test.go",
			StartLine: 10,
			EndLine:   20,
			Content:   "func HelloWorld() {\n\tfmt.Println(\"Hello\")\n}",
			Score:     0.95,
		},
	}

	processed := handler.applyResponseMode(results, "summary")

	if len(processed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(processed))
	}

	if processed[0].Content != "" {
		t.Errorf("Summary mode should remove content, got: %s", processed[0].Content)
	}

	if processed[0].FilePath != "test.go" {
		t.Errorf("Summary mode should preserve metadata, got: %s", processed[0].FilePath)
	}

	if processed[0].StartLine != 10 || processed[0].EndLine != 20 {
		t.Errorf("Summary mode should preserve line numbers")
	}
}

// TestApplyResponseModePreview tests that preview mode keeps first 20 lines
func TestApplyResponseModePreview(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	handler := &CodeToolsHandler{logger: logger}

	// Create content with 30 lines
	content := ""
	for i := 1; i <= 30; i++ {
		content += "line " + string(rune(i)) + "\n"
	}

	results := []storage.SearchResult{
		{
			FilePath:  "test.go",
			StartLine: 1,
			EndLine:   30,
			Content:   content,
			Score:     0.95,
		},
	}

	processed := handler.applyResponseMode(results, "preview")

	if len(processed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(processed))
	}

	if processed[0].Content == "" {
		t.Errorf("Preview mode should keep content")
	}

	if len(processed[0].Content) > len(content) {
		t.Errorf("Preview mode should truncate content")
	}

	if !contains(processed[0].Content, "truncated") {
		t.Errorf("Preview mode should indicate truncation")
	}
}

// TestApplyResponseModeFull tests that full mode keeps all content
func TestApplyResponseModeFull(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	handler := &CodeToolsHandler{logger: logger}

	originalContent := "func HelloWorld() {\n\tfmt.Println(\"Hello\")\n}"

	results := []storage.SearchResult{
		{
			FilePath:  "test.go",
			StartLine: 1,
			EndLine:   3,
			Content:   originalContent,
			Score:     0.95,
		},
	}

	processed := handler.applyResponseMode(results, "full")

	if len(processed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(processed))
	}

	if processed[0].Content != originalContent {
		t.Errorf("Full mode should preserve all content")
	}
}

// TestEstimateResponseModeTokens tests token estimation for each mode
func TestEstimateResponseModeTokens(t *testing.T) {
	result := storage.SearchResult{
		Content: "func HelloWorld() {\n\tfmt.Println(\"Hello\")\n}",
	}

	summaryTokens := estimateResponseModeTokens(result, "summary")
	if summaryTokens < 100 || summaryTokens > 200 {
		t.Errorf("Summary mode tokens should be 100-200, got %d", summaryTokens)
	}

	previewTokens := estimateResponseModeTokens(result, "preview")
	if previewTokens < 300 || previewTokens > 500 {
		t.Errorf("Preview mode tokens should be 300-500, got %d", previewTokens)
	}

	fullTokens := estimateResponseModeTokens(result, "full")
	if fullTokens <= 0 {
		t.Errorf("Full mode tokens should be > 0, got %d", fullTokens)
	}
}

// TestApplyResponseModeMultipleResults tests response mode with multiple results
func TestApplyResponseModeMultipleResults(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	handler := &CodeToolsHandler{logger: logger}

	results := []storage.SearchResult{
		{
			FilePath:  "file1.go",
			StartLine: 1,
			EndLine:   10,
			Content:   "func Func1() {}",
			Score:     0.95,
		},
		{
			FilePath:  "file2.go",
			StartLine: 20,
			EndLine:   30,
			Content:   "func Func2() {}",
			Score:     0.90,
		},
		{
			FilePath:  "file3.go",
			StartLine: 40,
			EndLine:   50,
			Content:   "func Func3() {}",
			Score:     0.85,
		},
	}

	processed := handler.applyResponseMode(results, "summary")

	if len(processed) != 3 {
		t.Errorf("Expected 3 results, got %d", len(processed))
	}

	for i, result := range processed {
		if result.Content != "" {
			t.Errorf("Result %d should have empty content in summary mode", i)
		}
		if result.FilePath == "" {
			t.Errorf("Result %d should preserve FilePath", i)
		}
	}
}

// TestResponseModePreservesMetadata tests that all modes preserve metadata
func TestResponseModePreservesMetadata(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	handler := &CodeToolsHandler{logger: logger}

	result := storage.SearchResult{
		FileID:       "file123",
		FilePath:     "test.go",
		RelativePath: "src/test.go",
		Language:     "go",
		StartLine:    10,
		EndLine:      20,
		Content:      "func Test() {}",
		Score:        0.95,
		NodeType:     "function",
		NodeName:     "Test",
	}

	results := []storage.SearchResult{result}

	for _, mode := range []string{"summary", "preview", "full"} {
		processed := handler.applyResponseMode(results, mode)

		if processed[0].FileID != result.FileID {
			t.Errorf("%s mode should preserve FileID", mode)
		}
		if processed[0].FilePath != result.FilePath {
			t.Errorf("%s mode should preserve FilePath", mode)
		}
		if processed[0].Language != result.Language {
			t.Errorf("%s mode should preserve Language", mode)
		}
		if processed[0].StartLine != result.StartLine {
			t.Errorf("%s mode should preserve StartLine", mode)
		}
		if processed[0].EndLine != result.EndLine {
			t.Errorf("%s mode should preserve EndLine", mode)
		}
		if processed[0].Score != result.Score {
			t.Errorf("%s mode should preserve Score", mode)
		}
		if processed[0].NodeType != result.NodeType {
			t.Errorf("%s mode should preserve NodeType", mode)
		}
		if processed[0].NodeName != result.NodeName {
			t.Errorf("%s mode should preserve NodeName", mode)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
