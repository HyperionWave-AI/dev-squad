package tools

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"hyper/internal/models"
)

// ResultFormatter handles structured presentation of tool results for AI decision-making
type ResultFormatter struct {
	maxKeyMatches int // Maximum key matches to extract per result
	contextChars  int // Characters of context around each match
}

// NewResultFormatter creates a new result formatter
func NewResultFormatter() *ResultFormatter {
	return &ResultFormatter{
		maxKeyMatches: 3,
		contextChars:  50,
	}
}

// FormatSearchResults converts raw search results into structured presentation
func (rf *ResultFormatter) FormatSearchResults(
	rawResults []map[string]interface{},
	searchQuery string,
	executionTimeMs int64,
) *models.StructuredToolResultResponse {
	if len(rawResults) == 0 {
		return &models.StructuredToolResultResponse{
			TotalResults:      0,
			GroupCount:        0,
			HighPriorityCount: 0,
			Groups:            []models.ResultGroup{},
			Recommendations:   []models.PriorityRecommendation{},
			SearchQuery:       searchQuery,
			ExecutionTime:     executionTimeMs,
			Timestamp:         time.Now(),
		}
	}

	// Step 1: Convert raw results to structured results with ranking
	structuredResults := rf.convertToStructuredResults(rawResults)

	// Step 2: Group results by file/module
	groups := rf.groupResultsByFile(structuredResults)

	// Step 3: Generate priority recommendations
	recommendations := rf.generateRecommendations(groups)

	// Step 4: Count high-priority results
	highPriorityCount := 0
	for _, group := range groups {
		for _, result := range group.Results {
			if result.Priority == "high" {
				highPriorityCount++
			}
		}
	}

	return &models.StructuredToolResultResponse{
		TotalResults:      len(rawResults),
		GroupCount:        len(groups),
		HighPriorityCount: highPriorityCount,
		Groups:            groups,
		Recommendations:   recommendations,
		SearchQuery:       searchQuery,
		ExecutionTime:     executionTimeMs,
		Timestamp:         time.Now(),
	}
}

// convertToStructuredResults converts raw results to structured format with ranking
func (rf *ResultFormatter) convertToStructuredResults(
	rawResults []map[string]interface{},
) []models.StructuredResult {
	results := make([]models.StructuredResult, 0, len(rawResults))

	for i, raw := range rawResults {
		// Extract fields from raw result
		id := rf.getString(raw, "id")
		filePath := rf.getString(raw, "filePath")
		text := rf.getString(raw, "text")
		score := rf.getFloat(raw, "score")

		// Extract key matches from text
		keyMatches := rf.extractKeyMatches(text, score)

		// Generate file summary
		summary := rf.generateFileSummary(filePath, text)

		// Determine priority based on score
		priority := rf.determinePriority(score)

		result := models.StructuredResult{
			ID:         id,
			FilePath:   filePath,
			Score:      score,
			Text:       text,
			KeyMatches: keyMatches,
			Summary:    summary,
			Rank:       i + 1,
			Priority:   priority,
		}

		results = append(results, result)
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Update ranks after sorting
	for i := range results {
		results[i].Rank = i + 1
	}

	return results
}

// groupResultsByFile groups results by file path
func (rf *ResultFormatter) groupResultsByFile(
	results []models.StructuredResult,
) []models.ResultGroup {
	// Map to track groups
	groupMap := make(map[string]*models.ResultGroup)
	groupOrder := []string{} // Track insertion order

	for _, result := range results {
		filePath := result.FilePath
		if _, exists := groupMap[filePath]; !exists {
			group := &models.ResultGroup{
				GroupName:   filePath,
				GroupType:   "file",
				ResultCount: 0,
				AvgScore:    0,
				Results:     []models.StructuredResult{},
				Priority:    "low",
			}
			groupMap[filePath] = group
			groupOrder = append(groupOrder, filePath)
		}

		groupMap[filePath].Results = append(groupMap[filePath].Results, result)
		groupMap[filePath].ResultCount++
	}

	// Calculate average scores and determine group priority
	for _, filePath := range groupOrder {
		group := groupMap[filePath]
		totalScore := 0.0
		for _, result := range group.Results {
			totalScore += result.Score
		}
		group.AvgScore = totalScore / float64(group.ResultCount)
		group.Priority = rf.determinePriority(group.AvgScore)
	}

	// Sort groups by average score (highest first)
	sort.Slice(groupOrder, func(i, j int) bool {
		return groupMap[groupOrder[i]].AvgScore > groupMap[groupOrder[j]].AvgScore
	})

	// Build final groups array
	groups := make([]models.ResultGroup, 0, len(groupOrder))
	for _, filePath := range groupOrder {
		groups = append(groups, *groupMap[filePath])
	}

	return groups
}

// extractKeyMatches extracts highlighted matches from result text
func (rf *ResultFormatter) extractKeyMatches(text string, score float64) []models.KeyMatch {
	matches := []models.KeyMatch{}

	// Split text into lines
	lines := strings.Split(text, "\n")

	// Extract up to maxKeyMatches from the text
	matchCount := 0
	for lineNum, line := range lines {
		if matchCount >= rf.maxKeyMatches {
			break
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Use the line as a key match
		match := models.KeyMatch{
			Text:       strings.TrimSpace(line),
			LineNum:    lineNum + 1,
			Context:    rf.getContext(text, lineNum),
			Relevance:  score * (1.0 - float64(matchCount)*0.1), // Decrease relevance for subsequent matches
		}

		matches = append(matches, match)
		matchCount++
	}

	return matches
}

// getContext extracts surrounding context for a line
func (rf *ResultFormatter) getContext(text string, lineNum int) string {
	lines := strings.Split(text, "\n")

	if lineNum < 0 || lineNum >= len(lines) {
		return ""
	}

	// Get context from surrounding lines
	start := lineNum - 1
	if start < 0 {
		start = 0
	}

	end := lineNum + 2
	if end > len(lines) {
		end = len(lines)
	}

	contextLines := lines[start:end]
	return strings.Join(contextLines, " ")
}

// generateFileSummary creates a summary of file contents
func (rf *ResultFormatter) generateFileSummary(filePath string, content string) models.FileSummary {
	// Extract file type from extension
	ext := filepath.Ext(filePath)
	if ext != "" {
		ext = ext[1:] // Remove leading dot
	}

	// Count lines
	lineCount := len(strings.Split(content, "\n"))

	// Extract module/package name from path
	module := rf.extractModule(filePath)

	// Generate description based on file type and content
	description := rf.generateDescription(filePath, content, ext)

	return models.FileSummary{
		FilePath:    filePath,
		FileType:    ext,
		LineCount:   lineCount,
		Description: description,
		Module:      module,
	}
}

// extractModule extracts module/package name from file path
func (rf *ResultFormatter) extractModule(filePath string) string {
	// For Go files, extract package name
	if strings.HasSuffix(filePath, ".go") {
		parts := strings.Split(filePath, "/")
		if len(parts) > 1 {
			return parts[len(parts)-2] // Return directory name
		}
	}

	// For TypeScript/JavaScript files, extract component name
	if strings.HasSuffix(filePath, ".tsx") || strings.HasSuffix(filePath, ".ts") {
		parts := strings.Split(filePath, "/")
		if len(parts) > 0 {
			name := parts[len(parts)-1]
			name = strings.TrimSuffix(name, ".tsx")
			name = strings.TrimSuffix(name, ".ts")
			return name
		}
	}

	return ""
}

// generateDescription creates a brief description of file contents
func (rf *ResultFormatter) generateDescription(filePath string, content string, fileType string) string {
	// Look for common patterns in content
	contentLower := strings.ToLower(content)

	switch fileType {
	case "go":
		if strings.Contains(contentLower, "func") {
			if strings.Contains(contentLower, "handler") {
				return "HTTP handler functions"
			}
			if strings.Contains(contentLower, "service") {
				return "Service implementation with business logic"
			}
			if strings.Contains(contentLower, "repository") || strings.Contains(contentLower, "storage") {
				return "Data access/storage layer"
			}
			return "Go functions and methods"
		}
		if strings.Contains(contentLower, "type") && strings.Contains(contentLower, "struct") {
			return "Go data structures and types"
		}
		return "Go source code"

	case "tsx", "ts":
		if strings.Contains(contentLower, "component") || strings.Contains(contentLower, "export default") {
			return "React component"
		}
		if strings.Contains(contentLower, "interface") || strings.Contains(contentLower, "type") {
			return "TypeScript types and interfaces"
		}
		if strings.Contains(contentLower, "hook") || strings.Contains(contentLower, "usestate") {
			return "React hooks"
		}
		return "TypeScript/React code"

	case "json":
		return "JSON configuration or data"

	case "css", "scss":
		return "Styling rules"

	default:
		return fmt.Sprintf("%s file", fileType)
	}
}

// determinePriority determines priority level based on score
func (rf *ResultFormatter) determinePriority(score float64) string {
	if score >= 0.8 {
		return "high"
	}
	if score >= 0.5 {
		return "medium"
	}
	return "low"
}

// generateRecommendations generates priority recommendations for AI
func (rf *ResultFormatter) generateRecommendations(
	groups []models.ResultGroup,
) []models.PriorityRecommendation {
	recommendations := []models.PriorityRecommendation{}

	// Recommend top 5 files by average score
	maxRecommendations := 5
	if len(groups) < maxRecommendations {
		maxRecommendations = len(groups)
	}

	for i := 0; i < maxRecommendations; i++ {
		group := groups[i]

		// Calculate confidence based on score and match count
		confidence := group.AvgScore
		if group.ResultCount > 1 {
			confidence = confidence * (1.0 - 0.1*float64(group.ResultCount-1)) // Slight penalty for many matches
		}
		if confidence > 1.0 {
			confidence = 1.0
		}

		reason := rf.generateRecommendationReason(group)

		rec := models.PriorityRecommendation{
			Rank:       i + 1,
			FilePath:   group.GroupName,
			Reason:     reason,
			Score:      group.AvgScore,
			MatchCount: group.ResultCount,
			Confidence: confidence,
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// generateRecommendationReason creates a human-readable reason for recommendation
func (rf *ResultFormatter) generateRecommendationReason(group models.ResultGroup) string {
	if group.ResultCount == 1 {
		return fmt.Sprintf("Single high-relevance match (score: %.2f)", group.AvgScore)
	}

	return fmt.Sprintf("%d relevant matches with average score %.2f", group.ResultCount, group.AvgScore)
}

// Helper functions to safely extract values from maps

func (rf *ResultFormatter) getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (rf *ResultFormatter) getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case float32:
			return float64(v)
		case int:
			return float64(v)
		}
	}
	return 0.0
}
