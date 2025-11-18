package aiservice

import (
	"context"
	"strings"
	"testing"
	"time"
)

// MockChatProvider implements ChatProvider for testing
type MockChatProvider struct {
	response string
	err      error
}

func (m *MockChatProvider) StreamChat(ctx context.Context, messages []Message) (<-chan string, error) {
	if m.err != nil {
		return nil, m.err
	}
	
	ch := make(chan string, 1)
	ch <- m.response
	close(ch)
	return ch, nil
}

func TestComplexityAnalyzer_AnalyzeFileCount(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		name     string
		files    []string
		expected float64
	}{
		{"No files", []string{}, 0.0},
		{"One file", []string{"file1.go"}, 0.0},
		{"Two files", []string{"file1.go", "file2.go"}, 0.2},
		{"Three files", []string{"file1.go", "file2.go", "file3.go"}, 0.2},
		{"Six files", []string{"f1.go", "f2.go", "f3.go", "f4.go", "f5.go", "f6.go"}, 0.5},
		{"Ten files", make([]string, 10), 0.8},
		{"Fifteen files", make([]string, 15), 1.0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill slice with dummy filenames for large tests
			for i := range tt.files {
				if tt.files[i] == "" {
					tt.files[i] = "file" + string(rune('0'+i)) + ".go"
				}
			}
			
			result := analyzer.analyzeFileCount(tt.files)
			if result != tt.expected {
				t.Errorf("analyzeFileCount() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_AnalyzeFileSize(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	ctx := context.Background()
	
	tests := []struct {
		name     string
		files    []string
		expected float64
	}{
		{"No files", []string{}, 0.0},
		{"Small Go file", []string{"small.go"}, 0.3}, // 200 lines estimated
		{"Multiple small files", []string{"a.go", "b.go"}, 0.6}, // 400 lines total
		{"Large project", []string{"a.go", "b.go", "c.go", "d.go", "e.go", "f.go", "g.go", "h.go"}, 1.0}, // 1600 lines
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.analyzeFileSize(ctx, tt.files)
			if result != tt.expected {
				t.Errorf("analyzeFileSize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_AnalyzeCrossSquadImpact(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		name     string
		files    []string
		expected float64
	}{
		{"No files", []string{}, 0.0},
		{"Single domain", []string{"/project/ui/component.tsx", "/project/ui/styles.css"}, 0.0},
		{"Two domains", []string{"/project/ui/component.tsx", "/project/api/handler.go"}, 0.4},
		{"Three domains", []string{"/project/ui/comp.tsx", "/project/api/handler.go", "/project/db/migration.sql"}, 0.7},
		{"Four domains", []string{"/project/ui/a.tsx", "/project/api/b.go", "/project/db/c.sql", "/project/config/d.yaml"}, 1.0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.analyzeCrossSquadImpact(tt.files)
			if result != tt.expected {
				t.Errorf("analyzeCrossSquadImpact() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_AnalyzeArchitecturalScope(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		name        string
		files       []string
		description string
		minExpected float64 // Minimum expected score
		maxExpected float64 // Maximum expected score
	}{
		{"No files", []string{}, "simple task", 0.0, 0.0},
		{"Simple task", []string{"component.tsx"}, "update button color", 0.0, 0.3},
		{"Database task", []string{"migration.sql"}, "add database schema migration", 0.3, 0.8},
		{"Auth task", []string{"auth.go", "middleware.go"}, "implement authentication system", 0.4, 1.0},
		{"Architecture refactor", []string{"service.go", "config.yaml"}, "refactor service architecture", 0.5, 1.0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.analyzeArchitecturalScope(tt.files, tt.description)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("analyzeArchitecturalScope() = %v, want between %v and %v", result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestComplexityAnalyzer_AnalyzeEstimatedLineChanges(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		name        string
		description string
		files       []string
		minExpected float64
		maxExpected float64
	}{
		{"Small tweak", "minor tweak to button", []string{"button.tsx"}, 0.0, 0.4},
		{"Bug fix", "fix validation bug", []string{"form.tsx"}, 0.3, 0.7},
		{"New feature", "implement new dashboard", []string{"dash.tsx", "api.go"}, 0.5, 1.0},
		{"Major refactor", "refactor entire auth system", []string{"auth.go", "middleware.go", "config.yaml"}, 0.7, 1.0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.analyzeEstimatedLineChanges(tt.description, tt.files)
			if result < tt.minExpected || result > tt.maxExpected {
				t.Errorf("analyzeEstimatedLineChanges() = %v, want between %v and %v", result, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestComplexityAnalyzer_CalculateWeightedScore(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	scores := HeuristicScores{
		FileCount:             0.5,  // Weight: 0.20
		FileSize:              0.6,  // Weight: 0.25
		CrossSquadImpact:      0.4,  // Weight: 0.15
		ArchitecturalScope:    0.8,  // Weight: 0.20
		EstimatedLineChanges:  0.7,  // Weight: 0.20
	}
	
	expected := 0.5*0.20 + 0.6*0.25 + 0.4*0.15 + 0.8*0.20 + 0.7*0.20
	result := analyzer.calculateWeightedScore(scores)
	
	if result != expected {
		t.Errorf("calculateWeightedScore() = %v, want %v", result, expected)
	}
}

func TestComplexityAnalyzer_DetermineComplexityLevel(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		score    float64
		expected ComplexityLevel
	}{
		{0.0, ComplexityLow},
		{0.2, ComplexityLow},
		{0.3, ComplexityMedium},
		{0.5, ComplexityMedium},
		{0.6, ComplexityHigh},
		{0.7, ComplexityHigh},
		{0.8, ComplexityExtreme},
		{1.0, ComplexityExtreme},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := analyzer.determineComplexityLevel(tt.score)
			if result != tt.expected {
				t.Errorf("determineComplexityLevel(%v) = %v, want %v", tt.score, result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_AnalyzeComplexity(t *testing.T) {
	config := &AIConfig{
		Provider: "test",
		Model:    "test-model",
	}
	mockClient := &MockChatProvider{}
	analyzer := NewComplexityAnalyzer(config, mockClient)
	
	ctx := context.Background()
	task := TaskContext{
		Description:    "Implement user authentication system",
		FilesModified:  []string{"auth.go", "middleware.go", "config.yaml", "user.go"},
		Role:           "Backend developer",
		ContextSummary: "Add JWT-based authentication with middleware",
	}
	
	analysis, err := analyzer.AnalyzeComplexity(ctx, task)
	if err != nil {
		t.Fatalf("AnalyzeComplexity() error = %v", err)
	}
	
	if analysis == nil {
		t.Fatal("AnalyzeComplexity() returned nil analysis")
	}
	
	// Verify analysis structure
	if analysis.OverallScore < 0.0 || analysis.OverallScore > 1.0 {
		t.Errorf("OverallScore = %v, want between 0.0 and 1.0", analysis.OverallScore)
	}
	
	if analysis.Level == "" {
		t.Error("Level should not be empty")
	}
	
	if analysis.Reasoning == "" {
		t.Error("Reasoning should not be empty")
	}
	
	// Verify heuristic scores are within bounds
	scores := analysis.HeuristicScores
	if scores.FileCount < 0.0 || scores.FileCount > 1.0 {
		t.Errorf("FileCount score = %v, want between 0.0 and 1.0", scores.FileCount)
	}
	if scores.FileSize < 0.0 || scores.FileSize > 1.0 {
		t.Errorf("FileSize score = %v, want between 0.0 and 1.0", scores.FileSize)
	}
	if scores.CrossSquadImpact < 0.0 || scores.CrossSquadImpact > 1.0 {
		t.Errorf("CrossSquadImpact score = %v, want between 0.0 and 1.0", scores.CrossSquadImpact)
	}
	if scores.ArchitecturalScope < 0.0 || scores.ArchitecturalScope > 1.0 {
		t.Errorf("ArchitecturalScope score = %v, want between 0.0 and 1.0", scores.ArchitecturalScope)
	}
	if scores.EstimatedLineChanges < 0.0 || scores.EstimatedLineChanges > 1.0 {
		t.Errorf("EstimatedLineChanges score = %v, want between 0.0 and 1.0", scores.EstimatedLineChanges)
	}
}

func TestComplexityAnalyzer_GenerateSplitSuggestions(t *testing.T) {
	config := &AIConfig{
		Provider: "test",
		Model:    "test-model",
	}
	
	// Mock LLM response with valid JSON
	mockResponse := `[
		{
			"subtaskTitle": "Authentication Core",
			"subtaskDescription": "Implement JWT token generation and validation",
			"filesInvolved": ["auth.go"],
			"estimatedComplexity": "medium",
			"dependencies": [],
			"rationale": "Core auth logic should be implemented first"
		},
		{
			"subtaskTitle": "Middleware Integration",
			"subtaskDescription": "Add authentication middleware to routes",
			"filesInvolved": ["middleware.go"],
			"estimatedComplexity": "low",
			"dependencies": ["Authentication Core"],
			"rationale": "Middleware depends on core auth functionality"
		}
	]`
	
	mockClient := &MockChatProvider{response: mockResponse}
	analyzer := NewComplexityAnalyzer(config, mockClient)
	
	ctx := context.Background()
	task := TaskContext{
		Description:    "Implement user authentication system",
		FilesModified:  []string{"auth.go", "middleware.go"},
		Role:           "Backend developer",
		ContextSummary: "Add JWT-based authentication",
	}
	
	analysis := &ComplexityAnalysis{
		OverallScore: 0.7, // Above split threshold
		Level:        ComplexityHigh,
		ShouldSplit:  true,
	}
	
	suggestions, err := analyzer.GenerateSplitSuggestions(ctx, task, analysis)
	if err != nil {
		t.Fatalf("GenerateSplitSuggestions() error = %v", err)
	}
	
	if len(suggestions) != 2 {
		t.Errorf("Expected 2 suggestions, got %d", len(suggestions))
	}
	
	// Verify first suggestion
	if suggestions[0].SubtaskTitle != "Authentication Core" {
		t.Errorf("First suggestion title = %v, want 'Authentication Core'", suggestions[0].SubtaskTitle)
	}
	
	if suggestions[0].EstimatedComplexity != "medium" {
		t.Errorf("First suggestion complexity = %v, want 'medium'", suggestions[0].EstimatedComplexity)
	}
}

func TestComplexityAnalyzer_ValidateTaskContext(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		name    string
		task    TaskContext
		wantErr bool
	}{
		{
			name: "Valid task",
			task: TaskContext{
				Description:   "Test task",
				FilesModified: []string{"file.go"},
			},
			wantErr: false,
		},
		{
			name: "Valid task with role",
			task: TaskContext{
				Role:          "Developer",
				FilesModified: []string{"file.go"},
			},
			wantErr: false,
		},
		{
			name: "Missing description and role",
			task: TaskContext{
				FilesModified: []string{"file.go"},
			},
			wantErr: true,
		},
		{
			name: "Missing files",
			task: TaskContext{
				Description: "Test task",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.ValidateTaskContext(tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTaskContext() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestComplexityAnalyzer_IsComplexTask(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		score    float64
		expected bool
	}{
		{0.0, false},
		{0.3, false},
		{0.59, false},
		{0.6, true},
		{0.8, true},
		{1.0, true},
	}
	
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := analyzer.IsComplexTask(tt.score)
			if result != tt.expected {
				t.Errorf("IsComplexTask(%v) = %v, want %v", tt.score, result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_GetComplexityDescription(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		level    ComplexityLevel
		contains string
	}{
		{ComplexityLow, "Low complexity"},
		{ComplexityMedium, "Medium complexity"},
		{ComplexityHigh, "High complexity"},
		{ComplexityExtreme, "Extreme complexity"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			result := analyzer.GetComplexityDescription(tt.level)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("GetComplexityDescription(%v) = %v, should contain %v", tt.level, result, tt.contains)
			}
		})
	}
}

func TestComplexityAnalyzer_GetRecommendedSplitCount(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		level    ComplexityLevel
		expected int
	}{
		{ComplexityLow, 2},
		{ComplexityMedium, 2},
		{ComplexityHigh, 3},
		{ComplexityExtreme, 5},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			analysis := &ComplexityAnalysis{Level: tt.level}
			result := analyzer.GetRecommendedSplitCount(analysis)
			if result != tt.expected {
				t.Errorf("GetRecommendedSplitCount(%v) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_EstimateFileSizeFromPath(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		path     string
		expected int
	}{
		{"", 0},
		{"file.go", 200},
		{"component.tsx", 150},
		{"script.js", 120},
		{"module.py", 180},
		{"Service.java", 250},
		{"main.cpp", 300},
		{"unknown.xyz", 100},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := analyzer.estimateFileSizeFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("estimateFileSizeFromPath(%v) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestComplexityAnalyzer_ExtractDomainFromPath(t *testing.T) {
	analyzer := &ComplexityAnalyzer{}
	
	tests := []struct {
		path     string
		expected string
	}{
		{"", "unknown"},
		{"/project/internal/auth/service.go", "auth"},
		{"/project/src/ui/components/Button.tsx", "ui"},
		{"/project/lib/database/migration.sql", "database"},
		{"/simple/file.go", "simple"},
		{"file.go", "unknown"},
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := analyzer.extractDomainFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("extractDomainFromPath(%v) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// Benchmark tests for performance
func BenchmarkComplexityAnalyzer_AnalyzeComplexity(b *testing.B) {
	config := &AIConfig{Provider: "test", Model: "test-model"}
	mockClient := &MockChatProvider{}
	analyzer := NewComplexityAnalyzer(config, mockClient)
	
	ctx := context.Background()
	task := TaskContext{
		Description:    "Implement complex feature",
		FilesModified:  []string{"a.go", "b.go", "c.go", "d.go"},
		Role:           "Developer",
		ContextSummary: "Complex implementation task",
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := analyzer.AnalyzeComplexity(ctx, task)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkComplexityAnalyzer_AnalyzeFileCount(b *testing.B) {
	analyzer := &ComplexityAnalyzer{}
	files := make([]string, 10)
	for i := range files {
		files[i] = "file" + string(rune('0'+i)) + ".go"
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.analyzeFileCount(files)
	}
}