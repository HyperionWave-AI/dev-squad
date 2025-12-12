package aiservice

import (
	"fmt"
	"testing"
)

// TestDetectFileType tests file type detection for various file paths
func TestDetectFileType(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	tests := []struct {
		filePath string
		expected string
	}{
		// UI files
		{"./ui/src/components/TaskCard.tsx", "ui"},
		{"./ui/src/pages/Dashboard.jsx", "ui"},
		{"./ui/src/styles/main.css", "ui"},
		{"./ui/src/styles/theme.scss", "ui"},

		// Backend files
		{"./hyper/internal/handlers/auth.go", "backend"},
		{"./hyper/internal/middleware/logger.go", "backend"},
		{"./cmd/server/main.go", "backend"},

		// Test files
		{"./hyper/internal/handlers/auth_test.go", "test"},
		{"./ui/src/components/Button.test.tsx", "test"},
		{"./ui/src/utils/helpers.spec.ts", "test"},

		// Config files
		{".env", "config"},
		{"./config/app.yaml", "config"},
		{"./config/database.json", "config"},
		{"./docker-compose.yml", "config"},

		// Other files
		{"./README.md", "other"},
		{"./Makefile", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := summarizer.DetectFileType(tt.filePath)
			if result != tt.expected {
				t.Errorf("DetectFileType(%s) = %s, want %s", tt.filePath, result, tt.expected)
			}
		})
	}
}

// TestCalculateRelevanceScore tests relevance scoring logic
func TestCalculateRelevanceScore(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	tests := []struct {
		matchType      string
		hasContextHint bool
		expectedMin    float64
		expectedMax    float64
	}{
		// Exact match without context
		{"exact", false, 0.75, 0.85},
		// Exact match with context
		{"exact", true, 0.90, 1.0},
		// Partial match without context
		{"partial", false, 0.45, 0.55},
		// Partial match with context
		{"partial", true, 0.60, 0.70},
		// Contextual match without context
		{"contextual", false, 0.25, 0.35},
		// Contextual match with context
		{"contextual", true, 0.40, 0.50},
	}

	for _, tt := range tests {
		t.Run(tt.matchType, func(t *testing.T) {
			score := summarizer.CalculateRelevanceScore(tt.matchType, tt.hasContextHint)
			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("CalculateRelevanceScore(%s, %v) = %f, want between %f and %f",
					tt.matchType, tt.hasContextHint, score, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

// TestExtractMetadata tests metadata extraction from result maps
func TestExtractMetadata(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	result := map[string]interface{}{
		"filePath":  "./ui/src/components/TaskCard.tsx",
		"lineNumber": 42.0,
		"matchType": "exact",
		"context":   "delete button implementation",
	}

	metadata := summarizer.ExtractMetadata(result)

	if metadata.FilePath != "./ui/src/components/TaskCard.tsx" {
		t.Errorf("Expected filePath './ui/src/components/TaskCard.tsx', got '%s'", metadata.FilePath)
	}

	if metadata.LineNumber != 42 {
		t.Errorf("Expected lineNumber 42, got %d", metadata.LineNumber)
	}

	if metadata.FileType != "ui" {
		t.Errorf("Expected fileType 'ui', got '%s'", metadata.FileType)
	}

	if metadata.MatchType != "exact" {
		t.Errorf("Expected matchType 'exact', got '%s'", metadata.MatchType)
	}

	if metadata.RelevanceScore < 0.9 || metadata.RelevanceScore > 1.0 {
		t.Errorf("Expected relevance score between 0.9 and 1.0, got %f", metadata.RelevanceScore)
	}
}

// TestSummarizeResults tests the full summarization pipeline
func TestSummarizeResults(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	results := []interface{}{
		map[string]interface{}{
			"filePath":  "./ui/src/components/TaskCard.tsx",
			"lineNumber": 42.0,
			"matchType": "exact",
			"context":   "delete button",
		},
		map[string]interface{}{
			"filePath":  "./ui/src/pages/Settings.tsx",
			"lineNumber": 15.0,
			"matchType": "partial",
			"context":   "button component",
		},
		map[string]interface{}{
			"filePath":  "./hyper/internal/handlers/auth.go",
			"lineNumber": 89.0,
			"matchType": "exact",
			"context":   "authentication logic",
		},
		map[string]interface{}{
			"filePath":  "./hyper/internal/handlers/handlers_test.go",
			"lineNumber": 12.0,
			"matchType": "contextual",
			"context":   "test case",
		},
	}

	summary := summarizer.SummarizeResults(results)

	// Verify summary contains expected content
	if len(summary) == 0 {
		t.Error("Expected non-empty summary")
	}

	// Check for category headers
	if !contains(summary, "UI Components") {
		t.Error("Expected 'UI Components' in summary")
	}

	if !contains(summary, "Backend Services") {
		t.Error("Expected 'Backend Services' in summary")
	}

	if !contains(summary, "Tests") {
		t.Error("Expected 'Tests' in summary")
	}

	// Check for file paths
	if !contains(summary, "TaskCard.tsx") {
		t.Error("Expected 'TaskCard.tsx' in summary")
	}

	if !contains(summary, "auth.go") {
		t.Error("Expected 'auth.go' in summary")
	}

	// Check for most relevant recommendation
	if !contains(summary, "Most relevant") {
		t.Error("Expected 'Most relevant' in summary")
	}
}

// TestSummarizeResultsEmpty tests summarization with empty results
func TestSummarizeResultsEmpty(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	summary := summarizer.SummarizeResults([]interface{}{})

	if summary != "No results found." {
		t.Errorf("Expected 'No results found.', got '%s'", summary)
	}
}

// TestSummarizeResultsSingleFile tests summarization with single file
func TestSummarizeResultsSingleFile(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	results := []interface{}{
		map[string]interface{}{
			"filePath":  "./ui/src/components/Button.tsx",
			"lineNumber": 10.0,
			"matchType": "exact",
			"context":   "button component",
		},
	}

	summary := summarizer.SummarizeResults(results)

	if !contains(summary, "Found 1 results") {
		t.Error("Expected 'Found 1 results' in summary")
	}

	if !contains(summary, "Button.tsx") {
		t.Error("Expected 'Button.tsx' in summary")
	}
}

// TestSummarizeResultsLargeSet tests summarization with many files
func TestSummarizeResultsLargeSet(t *testing.T) {
	summarizer := NewCodeResultSummarizer(2000)

	// Create 15 results
	results := make([]interface{}, 15)
	for i := 0; i < 15; i++ {
		results[i] = map[string]interface{}{
			"filePath":  "./ui/src/components/Component" + string(rune(i)) + ".tsx",
			"lineNumber": float64(i * 10),
			"matchType": "partial",
			"context":   "component",
		}
	}

	summary := summarizer.SummarizeResults(results)

	if !contains(summary, "Found 15 results") {
		t.Error("Expected 'Found 15 results' in summary")
	}

	// Verify summary is much smaller than raw results
	rawSize := len(fmt.Sprintf("%v", results))
	summarySize := len(summary)

	if summarySize >= rawSize {
		t.Errorf("Summary should be smaller than raw results. Raw: %d, Summary: %d", rawSize, summarySize)
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
