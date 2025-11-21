package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"
)

type ComplexityLevel string

const (
	ComplexityLow     ComplexityLevel = "low"     // 0.0 - 0.3
	ComplexityMedium  ComplexityLevel = "medium"  // 0.3 - 0.6
	ComplexityHigh    ComplexityLevel = "high"    // 0.6 - 0.8
	ComplexityExtreme ComplexityLevel = "extreme" // 0.8 - 1.0
)

// ComplexityAnalyzer provides complexity analysis for development tasks
type ComplexityAnalyzer struct {
	llmClient       ChatProvider
	codeIndexClient interface{} // Will be defined based on existing patterns
	config          *AIConfig
}

// ComplexityAnalysis represents the result of complexity analysis
type ComplexityAnalysis struct {
	OverallScore    float64         `json:"overallScore"`    // 0.0 - 1.0
	Level           ComplexityLevel `json:"level"`           // low/medium/high/extreme
	HeuristicScores HeuristicScores `json:"heuristicScores"` // Individual heuristic scores
	Reasoning       string          `json:"reasoning"`       // Human-readable explanation
	ShouldSplit     bool            `json:"shouldSplit"`     // True if score >= 0.6
}

// HeuristicScores contains individual heuristic analysis scores
type HeuristicScores struct {
	FileCount            float64 `json:"fileCount"`            // Weight: 0.20
	FileSize             float64 `json:"fileSize"`             // Weight: 0.25
	CrossSquadImpact     float64 `json:"crossSquadImpact"`     // Weight: 0.15
	ArchitecturalScope   float64 `json:"architecturalScope"`   // Weight: 0.20
	EstimatedLineChanges float64 `json:"estimatedLineChanges"` // Weight: 0.20
}

// SuggestedSplit represents an LLM-generated task split recommendation
type SuggestedSplit struct {
	SubtaskTitle        string   `json:"subtaskTitle"`        // Brief title for the subtask
	SubtaskDescription  string   `json:"subtaskDescription"`  // Detailed description
	FilesInvolved       []string `json:"filesInvolved"`       // Files this subtask would modify
	EstimatedComplexity string   `json:"estimatedComplexity"` // "low", "medium", "high"
	Dependencies        []string `json:"dependencies"`        // Other subtasks this depends on
	Rationale           string   `json:"rationale"`           // Why this split makes sense
}

// TaskContext represents the input context for complexity analysis
type TaskContext struct {
	Description    string   `json:"description"`    // Task description
	FilesModified  []string `json:"filesModified"`  // Files that will be modified
	Role           string   `json:"role"`           // Agent role/objective
	ContextSummary string   `json:"contextSummary"` // Additional context
}

// Heuristic weights for complexity calculation
const (
	WeightFileCount            = 0.20
	WeightFileSize             = 0.25
	WeightCrossSquadImpact     = 0.15
	WeightArchitecturalScope   = 0.20
	WeightEstimatedLineChanges = 0.20
)

// Complexity thresholds
const (
	ThresholdMedium  = 0.3
	ThresholdHigh    = 0.6
	ThresholdExtreme = 0.8
	SplitThreshold   = 0.6 // Tasks with score >= 0.6 should be split
)

// NewComplexityAnalyzer creates a new complexity analyzer instance
func NewComplexityAnalyzer(config *AIConfig, llmClient ChatProvider) *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		llmClient: llmClient,
		config:    config,
	}
}

// AnalyzeComplexity performs comprehensive complexity analysis on a task
func (ca *ComplexityAnalyzer) AnalyzeComplexity(ctx context.Context, task TaskContext) (*ComplexityAnalysis, error) {
	// Validate input
	if err := ca.ValidateTaskContext(task); err != nil {
		return nil, fmt.Errorf("invalid task context: %w", err)
	}

	log.Printf("Starting complexity analysis for task: %s", task.Description)

	// Calculate individual heuristic scores
	heuristicScores := HeuristicScores{
		FileCount:            ca.analyzeFileCount(task.FilesModified),
		FileSize:             ca.analyzeFileSize(ctx, task.FilesModified),
		CrossSquadImpact:     ca.analyzeCrossSquadImpact(task.FilesModified),
		ArchitecturalScope:   ca.analyzeArchitecturalScope(task.FilesModified, task.Description),
		EstimatedLineChanges: ca.analyzeEstimatedLineChanges(task.Description, task.FilesModified),
	}

	// Calculate weighted overall score
	overallScore := ca.calculateWeightedScore(heuristicScores)

	// Determine complexity level
	level := ca.determineComplexityLevel(overallScore)

	// Generate reasoning
	reasoning := ca.generateReasoning(heuristicScores, overallScore, level)

	analysis := &ComplexityAnalysis{
		OverallScore:    overallScore,
		Level:           level,
		HeuristicScores: heuristicScores,
		Reasoning:       reasoning,
		ShouldSplit:     overallScore >= SplitThreshold,
	}

	log.Printf("Complexity analysis complete: score=%.2f, level=%s, shouldSplit=%t",
		overallScore, level, analysis.ShouldSplit)

	return analysis, nil
}

// analyzeFileCount calculates complexity based on number of files (Weight: 0.20)
func (ca *ComplexityAnalyzer) analyzeFileCount(files []string) float64 {
	fileCount := len(files)

	// Scoring logic:
	// 1 file = 0.0, 2-3 files = 0.2, 4-6 files = 0.5, 7-10 files = 0.8, 11+ files = 1.0
	switch {
	case fileCount <= 1:
		return 0.0
	case fileCount <= 3:
		return 0.2
	case fileCount <= 6:
		return 0.5
	case fileCount <= 10:
		return 0.8
	default:
		return 1.0
	}
}

// analyzeFileSize calculates complexity based on estimated file sizes (Weight: 0.25)
func (ca *ComplexityAnalyzer) analyzeFileSize(ctx context.Context, files []string) float64 {
	if len(files) == 0 {
		return 0.0
	}

	// Check for context cancellation
	if ctx.Err() != nil {
		log.Printf("Context cancelled during file size analysis: %v", ctx.Err())
		return 0.0
	}

	// For now, estimate based on file extensions and count
	// In a full implementation, this would read actual file sizes
	totalEstimatedLines := 0
	for _, file := range files {
		estimatedLines := ca.estimateFileSizeFromPath(file)
		totalEstimatedLines += estimatedLines
	}

	// Scoring logic based on total estimated lines:
	// <100 lines = 0.0, 100-500 = 0.3, 500-1500 = 0.6, 1500-3000 = 0.8, 3000+ = 1.0
	switch {
	case totalEstimatedLines < 100:
		return 0.0
	case totalEstimatedLines < 500:
		return 0.3
	case totalEstimatedLines < 1500:
		return 0.6
	case totalEstimatedLines < 3000:
		return 0.8
	default:
		return 1.0
	}
}

// analyzeCrossSquadImpact calculates complexity based on cross-team impact (Weight: 0.15)
func (ca *ComplexityAnalyzer) analyzeCrossSquadImpact(files []string) float64 {
	if len(files) == 0 {
		return 0.0
	}

	// Analyze file paths to determine different domains/squads
	domains := make(map[string]bool)

	for _, file := range files {
		domain := ca.extractDomainFromPath(file)
		domains[domain] = true
	}

	domainCount := len(domains)

	// Scoring logic:
	// 1 domain = 0.0, 2 domains = 0.4, 3 domains = 0.7, 4+ domains = 1.0
	switch {
	case domainCount <= 1:
		return 0.0
	case domainCount == 2:
		return 0.4
	case domainCount == 3:
		return 0.7
	default:
		return 1.0
	}
}

// analyzeArchitecturalScope calculates complexity based on architectural impact (Weight: 0.20)
func (ca *ComplexityAnalyzer) analyzeArchitecturalScope(files []string, description string) float64 {
	if len(files) == 0 {
		return 0.0
	}

	score := 0.0

	// Check for architectural keywords in description
	architecturalKeywords := []string{
		"database", "schema", "migration", "api", "endpoint", "service", "interface",
		"architecture", "refactor", "restructure", "framework", "library", "dependency",
		"auth", "authentication", "authorization", "security", "middleware", "config",
	}

	descLower := strings.ToLower(description)
	keywordMatches := 0

	for _, keyword := range architecturalKeywords {
		if strings.Contains(descLower, keyword) {
			keywordMatches++
		}
	}

	// Keyword-based scoring (0.0 - 0.5)
	keywordScore := math.Min(float64(keywordMatches)*0.1, 0.5)

	// File path-based scoring (0.0 - 0.5)
	pathScore := ca.analyzeArchitecturalPathsScore(files)

	score = keywordScore + pathScore
	return math.Min(score, 1.0)
}

// analyzeEstimatedLineChanges calculates complexity based on estimated code changes (Weight: 0.20)
func (ca *ComplexityAnalyzer) analyzeEstimatedLineChanges(description string, files []string) float64 {
	if len(files) == 0 {
		return 0.0
	}

	// Estimate line changes based on task description keywords
	descLower := strings.ToLower(description)

	// High-impact keywords suggest more line changes
	highImpactKeywords := []string{
		"refactor", "rewrite", "restructure", "redesign", "overhaul",
		"implement", "create", "build", "develop", "add new",
	}

	mediumImpactKeywords := []string{
		"update", "modify", "change", "improve", "enhance", "extend",
		"fix", "bug", "issue", "problem",
	}

	lowImpactKeywords := []string{
		"tweak", "adjust", "minor", "small", "quick", "simple",
	}

	score := 0.3 // Base score

	for _, keyword := range highImpactKeywords {
		if strings.Contains(descLower, keyword) {
			score += 0.3
			break
		}
	}

	for _, keyword := range mediumImpactKeywords {
		if strings.Contains(descLower, keyword) {
			score += 0.2
			break
		}
	}

	for _, keyword := range lowImpactKeywords {
		if strings.Contains(descLower, keyword) {
			score -= 0.2
			break
		}
	}

	// Adjust based on file count
	fileCountMultiplier := math.Min(float64(len(files))*0.1, 0.4)
	score += fileCountMultiplier

	return math.Min(math.Max(score, 0.0), 1.0)
}

// Helper methods

// calculateWeightedScore computes the overall weighted complexity score
func (ca *ComplexityAnalyzer) calculateWeightedScore(scores HeuristicScores) float64 {
	weightedScore := scores.FileCount*WeightFileCount +
		scores.FileSize*WeightFileSize +
		scores.CrossSquadImpact*WeightCrossSquadImpact +
		scores.ArchitecturalScope*WeightArchitecturalScope +
		scores.EstimatedLineChanges*WeightEstimatedLineChanges

	return math.Min(math.Max(weightedScore, 0.0), 1.0)
}

// determineComplexityLevel maps a score to a complexity level
func (ca *ComplexityAnalyzer) determineComplexityLevel(score float64) ComplexityLevel {
	switch {
	case score < ThresholdMedium:
		return ComplexityLow
	case score < ThresholdHigh:
		return ComplexityMedium
	case score < ThresholdExtreme:
		return ComplexityHigh
	default:
		return ComplexityExtreme
	}
}

// generateReasoning creates human-readable explanation of complexity analysis
func (ca *ComplexityAnalyzer) generateReasoning(scores HeuristicScores, overallScore float64, level ComplexityLevel) string {
	var reasoning strings.Builder

	reasoning.WriteString(fmt.Sprintf("Overall complexity score: %.2f (%s)\n\n", overallScore, level))
	reasoning.WriteString("Heuristic breakdown:\n")
	reasoning.WriteString(fmt.Sprintf("• File count impact: %.2f (weight: %.0f%%)\n", scores.FileCount, WeightFileCount*100))
	reasoning.WriteString(fmt.Sprintf("• File size impact: %.2f (weight: %.0f%%)\n", scores.FileSize, WeightFileSize*100))
	reasoning.WriteString(fmt.Sprintf("• Cross-squad impact: %.2f (weight: %.0f%%)\n", scores.CrossSquadImpact, WeightCrossSquadImpact*100))
	reasoning.WriteString(fmt.Sprintf("• Architectural scope: %.2f (weight: %.0f%%)\n", scores.ArchitecturalScope, WeightArchitecturalScope*100))
	reasoning.WriteString(fmt.Sprintf("• Estimated line changes: %.2f (weight: %.0f%%)\n", scores.EstimatedLineChanges, WeightEstimatedLineChanges*100))

	if overallScore >= SplitThreshold {
		reasoning.WriteString("\nRecommendation: This task should be split into smaller subtasks due to high complexity.")
	}

	return reasoning.String()
}

// estimateFileSizeFromPath estimates file size based on file path and extension
func (ca *ComplexityAnalyzer) estimateFileSizeFromPath(filePath string) int {
	if filePath == "" {
		return 0
	}

	// Simple heuristic based on file extension
	if strings.HasSuffix(filePath, ".go") {
		return 200 // Average Go file
	}
	if strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx") {
		return 150 // Average TypeScript file
	}
	if strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx") {
		return 120 // Average JavaScript file
	}
	if strings.HasSuffix(filePath, ".py") {
		return 180 // Average Python file
	}
	if strings.HasSuffix(filePath, ".java") {
		return 250 // Average Java file
	}
	if strings.HasSuffix(filePath, ".cpp") || strings.HasSuffix(filePath, ".c") {
		return 300 // Average C/C++ file
	}
	return 100 // Default estimate
}

// extractDomainFromPath extracts domain/squad from file path
func (ca *ComplexityAnalyzer) extractDomainFromPath(filePath string) string {
	if filePath == "" {
		return "unknown"
	}

	// Extract domain from path structure
	parts := strings.Split(filePath, "/")

	// Look for common domain indicators
	for i, part := range parts {
		if part == "internal" || part == "src" || part == "lib" {
			if i+1 < len(parts) {
				return parts[i+1] // Return the next part as domain
			}
		}
	}

	// Fallback: use the directory containing the file
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}

	return "unknown"
}

// analyzeArchitecturalPathsScore analyzes file paths for architectural complexity
func (ca *ComplexityAnalyzer) analyzeArchitecturalPathsScore(files []string) float64 {
	architecturalPaths := []string{
		"config", "auth", "middleware", "service", "api", "database", "db",
		"migration", "schema", "model", "controller", "handler", "router",
	}

	score := 0.0
	for _, file := range files {
		fileLower := strings.ToLower(file)
		for _, archPath := range architecturalPaths {
			if strings.Contains(fileLower, archPath) {
				score += 0.1
				break // Only count once per file
			}
		}
	}

	return math.Min(score, 0.5) // Cap at 0.5 for path-based scoring
}

// IsComplexTask returns true if the task should be considered complex based on score
func (ca *ComplexityAnalyzer) IsComplexTask(score float64) bool {
	return score >= SplitThreshold
}

// GetComplexityDescription returns a human-readable description of the complexity level
func (ca *ComplexityAnalyzer) GetComplexityDescription(level ComplexityLevel) string {
	switch level {
	case ComplexityLow:
		return "Low complexity - straightforward implementation with minimal risk"
	case ComplexityMedium:
		return "Medium complexity - moderate implementation effort with some coordination needed"
	case ComplexityHigh:
		return "High complexity - significant implementation effort requiring careful planning"
	case ComplexityExtreme:
		return "Extreme complexity - very large scope requiring task decomposition"
	default:
		return "Unknown complexity level"
	}
}

// ValidateTaskContext ensures the task context has sufficient information for analysis
func (ca *ComplexityAnalyzer) ValidateTaskContext(task TaskContext) error {
	if task.Description == "" && task.Role == "" {
		return fmt.Errorf("task context must have either description or role")
	}

	if len(task.FilesModified) == 0 {
		return fmt.Errorf("task context must specify files to be modified")
	}

	return nil
}

// GetRecommendedSplitCount suggests how many subtasks a complex task should be split into
func (ca *ComplexityAnalyzer) GetRecommendedSplitCount(analysis *ComplexityAnalysis) int {
	switch analysis.Level {
	case ComplexityHigh:
		return 3 // Split into 3 subtasks
	case ComplexityExtreme:
		return 5 // Split into 5 subtasks
	default:
		return 2 // Default split into 2 subtasks
	}
}

// GenerateSplitSuggestions uses LLM to generate task split recommendations for complex tasks
func (ca *ComplexityAnalyzer) GenerateSplitSuggestions(ctx context.Context, task TaskContext, analysis *ComplexityAnalysis) ([]SuggestedSplit, error) {
	if analysis.OverallScore < SplitThreshold {
		return nil, fmt.Errorf("task complexity score %.2f is below split threshold %.2f", analysis.OverallScore, SplitThreshold)
	}

	if ca.llmClient == nil {
		return nil, fmt.Errorf("LLM client not available for split suggestions")
	}

	log.Printf("Generating LLM-powered split suggestions for complex task (score: %.2f)", analysis.OverallScore)

	// Create the prompt for LLM
	prompt := ca.buildSplitSuggestionsPrompt(task, analysis)

	// Prepare messages for LLM
	messages := []Message{
		{
			Role:    "system",
			Content: "You are an expert software architect and project manager. Your task is to analyze complex development tasks and suggest how to split them into smaller, manageable subtasks. Always respond with valid JSON.",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call LLM with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	responseChannel, err := ca.llmClient.StreamChat(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM for split suggestions: %w", err)
	}

	// Collect the full response
	var responseBuilder strings.Builder
	for token := range responseChannel {
		if strings.HasPrefix(token, "ERROR:") {
			return nil, fmt.Errorf("LLM error: %s", strings.TrimPrefix(token, "ERROR:"))
		}
		responseBuilder.WriteString(token)
	}

	response := responseBuilder.String()
	log.Printf("LLM response for split suggestions: %s", response)

	// Parse the JSON response
	var suggestions []SuggestedSplit
	if err := json.Unmarshal([]byte(response), &suggestions); err != nil {
		// Try to extract JSON from the response if it is wrapped in markdown or other text
		jsonStart := strings.Index(response, "[")
		jsonEnd := strings.LastIndex(response, "]")
		if jsonStart >= 0 && jsonEnd > jsonStart {
			jsonStr := response[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonStr), &suggestions); err != nil {
				return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
		}
	}

	// Validate suggestions
	validatedSuggestions := ca.validateSplitSuggestions(suggestions, task)

	log.Printf("Generated %d validated split suggestions", len(validatedSuggestions))
	return validatedSuggestions, nil
}

// buildSplitSuggestionsPrompt creates the prompt template for LLM split suggestions
func (ca *ComplexityAnalyzer) buildSplitSuggestionsPrompt(task TaskContext, analysis *ComplexityAnalysis) string {
	prompt := fmt.Sprintf(`Analyze this complex development task and suggest how to split it into smaller, manageable subtasks.

TASK DETAILS:
- Description: %s
- Role/Objective: %s
- Context: %s
- Files to modify: %s
- Complexity Score: %.2f (%s)
- Complexity Analysis: %s

Please suggest 2-5 subtasks that would make this work more manageable. For each subtask, provide:

RESPOND WITH VALID JSON ARRAY in this exact format:
[
  {
    "subtaskTitle": "Brief title for the subtask",
    "subtaskDescription": "Detailed description of what this subtask involves",
    "filesInvolved": ["file1.go", "file2.go"],
    "estimatedComplexity": "low|medium|high",
    "dependencies": ["other subtask titles this depends on"],
    "rationale": "Why this split makes sense and reduces complexity"
  }
]

Guidelines:
- Each subtask should be independently testable when possible
- Minimize dependencies between subtasks
- Consider logical boundaries (UI vs backend, different features, etc.)
- Ensure each subtask has clear deliverables
- Keep file modifications focused per subtask
- Aim for subtasks with "low" to "medium" complexity

Respond ONLY with the JSON array, no additional text.`,
		task.Description,
		task.Role,
		task.ContextSummary,
		strings.Join(task.FilesModified, ", "),
		analysis.OverallScore,
		analysis.Level,
		analysis.Reasoning)

	return prompt
}

// validateSplitSuggestions validates and filters the LLM-generated suggestions
func (ca *ComplexityAnalyzer) validateSplitSuggestions(suggestions []SuggestedSplit, task TaskContext) []SuggestedSplit {
	var validated []SuggestedSplit

	for _, suggestion := range suggestions {
		// Basic validation
		if suggestion.SubtaskTitle == "" || suggestion.SubtaskDescription == "" {
			log.Printf("Skipping invalid suggestion: missing title or description")
			continue
		}

		// Validate complexity level
		if suggestion.EstimatedComplexity != "low" &&
			suggestion.EstimatedComplexity != "medium" &&
			suggestion.EstimatedComplexity != "high" {
			suggestion.EstimatedComplexity = "medium" // Default fallback
		}

		// Ensure files involved are from the original task
		var validFiles []string
		for _, file := range suggestion.FilesInvolved {
			for _, originalFile := range task.FilesModified {
				if file == originalFile {
					validFiles = append(validFiles, file)
					break
				}
			}
		}
		suggestion.FilesInvolved = validFiles

		validated = append(validated, suggestion)
	}

	return validated
}
