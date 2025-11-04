package handlers

import (
	"context"
	"fmt"
	"time"

	"hyper/internal/mcp/storage"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReflectionToolHandler manages MCP reflection tool operations
type ReflectionToolHandler struct {
	reflectionStorage *storage.ReflectionStorage
}

// NewReflectionToolHandler creates a new reflection tool handler
func NewReflectionToolHandler(storage *storage.ReflectionStorage) *ReflectionToolHandler {
	return &ReflectionToolHandler{
		reflectionStorage: storage,
	}
}

// RegisterReflectionTools registers reflection tools with the MCP server
func (h *ReflectionToolHandler) RegisterReflectionTools(server *mcp.Server) error {
	// Register reflection_record_decision tool
	if err := h.registerRecordDecision(server); err != nil {
		return fmt.Errorf("failed to register reflection_record_decision tool: %w", err)
	}

	// Register reflection_record_outcome tool
	if err := h.registerRecordOutcome(server); err != nil {
		return fmt.Errorf("failed to register reflection_record_outcome tool: %w", err)
	}

	// Register reflection_extract_lesson tool
	if err := h.registerExtractLesson(server); err != nil {
		return fmt.Errorf("failed to register reflection_extract_lesson tool: %w", err)
	}

	// Register reflection_suggest_lesson_from_error tool
	if err := h.registerSuggestLessonFromError(server); err != nil {
		return fmt.Errorf("failed to register reflection_suggest_lesson_from_error tool: %w", err)
	}

	// Register reflection_query_relevant_lessons tool
	if err := h.registerQueryRelevantLessons(server); err != nil {
		return fmt.Errorf("failed to register reflection_query_relevant_lessons tool: %w", err)
	}

	return nil
}

// registerRecordDecision registers the reflection_record_decision tool
func (h *ReflectionToolHandler) registerRecordDecision(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "reflection_record_decision",
		Description: "Record a decision with context, reasoning, and predictions. This enables metacognitive tracking - the system remembers WHY it decided something, with what confidence, and what it predicted would happen.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"chatId": {
					Type:        "string",
					Description: "Chat session ID for this decision",
				},
				"taskId": {
					Type:        "string",
					Description: "Optional task ID if this decision is related to a specific task",
				},
				"context": {
					Type:        "object",
					Description: "Context in which the decision was made",
					Properties: map[string]*jsonschema.Schema{
						"userRequest": {
							Type:        "string",
							Description: "What the user asked for",
						},
						"availableInfo": {
							Type:        "string",
							Description: "What information was available when making the decision",
						},
						"uncertainty": {
							Type:        "string",
							Description: "What was uncertain or unknown",
						},
					},
				},
				"decision": {
					Type:        "object",
					Description: "The decision that was made",
					Properties: map[string]*jsonschema.Schema{
						"action": {
							Type:        "string",
							Description: "What action was decided upon",
						},
						"reasoning": {
							Type:        "string",
							Description: "Why this decision was made",
						},
						"alternatives": {
							Type:        "array",
							Description: "What other options were considered",
							Items: &jsonschema.Schema{
								Type: "string",
							},
						},
						"confidence": {
							Type:        "number",
							Description: "Confidence in this decision (0.0 to 1.0)",
						},
					},
				},
				"predictions": {
					Type:        "object",
					Description: "What outcomes were predicted",
					Properties: map[string]*jsonschema.Schema{
						"successProbability": {
							Type:        "number",
							Description: "Expected probability of success (0.0 to 1.0)",
						},
						"timeEstimate": {
							Type:        "string",
							Description: "Estimated time to complete",
						},
						"risks": {
							Type:        "array",
							Description: "Anticipated risks",
							Items: &jsonschema.Schema{
								Type: "string",
							},
						},
					},
				},
			},
			Required: []string{"chatId", "context", "decision"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleRecordDecision(args)
		return result, err
	})

	return nil
}

// registerRecordOutcome registers the reflection_record_outcome tool
func (h *ReflectionToolHandler) registerRecordOutcome(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "reflection_record_outcome",
		Description: "Record the actual outcome of a decision and compare it to predictions. Links to the original decision to enable confidence calibration - learning whether the system tends to be overconfident or underconfident.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"decisionId": {
					Type:        "string",
					Description: "ID of the original decision reflection",
				},
				"outcome": {
					Type:        "object",
					Description: "What actually happened",
					Properties: map[string]*jsonschema.Schema{
						"success": {
							Type:        "boolean",
							Description: "Whether the decision led to success",
						},
						"actualResult": {
							Type:        "string",
							Description: "What actually happened",
						},
						"userFeedback": {
							Type:        "string",
							Description: "Feedback from the user",
						},
						"rootCause": {
							Type:        "string",
							Description: "Root cause if there were issues",
						},
					},
				},
				"analysis": {
					Type:        "object",
					Description: "Analysis of prediction accuracy",
					Properties: map[string]*jsonschema.Schema{
						"predictionAccuracy": {
							Type:        "number",
							Description: "How accurate were the predictions (0.0 to 1.0)",
						},
						"missedSignals": {
							Type:        "array",
							Description: "What signals were missed",
							Items: &jsonschema.Schema{
								Type: "string",
							},
						},
						"confidenceCalibration": {
							Type:        "string",
							Description: "Was confidence appropriate? (overconfident/underconfident/well-calibrated)",
						},
					},
				},
			},
			Required: []string{"decisionId", "outcome", "analysis"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleRecordOutcome(args)
		return result, err
	})

	return nil
}

// registerExtractLesson registers the reflection_extract_lesson tool
func (h *ReflectionToolHandler) registerExtractLesson(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "reflection_extract_lesson",
		Description: "Extract a transferable lesson from experience. Stores patterns that can be applied to future similar situations - this is how the system learns from mistakes and successes.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"patternName": {
					Type:        "string",
					Description: "Name of the pattern (e.g., 'hardcoding-dynamic-values')",
				},
				"context": {
					Type:        "string",
					Description: "In what context does this lesson apply",
				},
				"problem": {
					Type:        "string",
					Description: "What problem occurred",
				},
				"solution": {
					Type:        "string",
					Description: "What was the solution",
				},
				"antipattern": {
					Type:        "string",
					Description: "What NOT to do (the mistake)",
				},
				"applicableTo": {
					Type:        "array",
					Description: "What types of situations is this applicable to",
					Items: &jsonschema.Schema{
						Type: "string",
					},
				},
				"confidence": {
					Type:        "number",
					Description: "Confidence in this lesson (0.0 to 1.0)",
				},
			},
			Required: []string{"patternName", "problem", "solution"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleExtractLesson(args)
		return result, err
	})

	return nil
}

// handleRecordDecision handles the reflection_record_decision tool call
func (h *ReflectionToolHandler) handleRecordDecision(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract chatId (required)
	chatId, ok := args["chatId"].(string)
	if !ok || chatId == "" {
		return createErrorResult("chatId parameter is required"), nil, nil
	}

	// Extract taskId (optional)
	taskId, _ := args["taskId"].(string)

	// Extract confidence from decision.confidence
	confidence := 0.5 // default
	if decision, ok := args["decision"].(map[string]interface{}); ok {
		if conf, ok := decision["confidence"].(float64); ok {
			confidence = conf
		}
	}

	// Create reflection
	reflection := &storage.Reflection{
		Type:       "decision",
		ChatID:     chatId,
		TaskID:     taskId,
		Timestamp:  time.Now().UTC(),
		Data:       args,
		Confidence: confidence,
		Tags:       []string{"decision", "metacognition"},
	}

	// Store in MongoDB
	decisionId, err := h.reflectionStorage.StoreReflection(reflection)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to store decision: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Decision recorded\n\nDecision ID: %s\nConfidence: %.2f\nChat: %s\n\nThis decision will be tracked for outcome comparison and confidence calibration.",
		decisionId, confidence, chatId)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"decisionId": decisionId,
		"stored":     true,
	}, nil
}

// handleRecordOutcome handles the reflection_record_outcome tool call
func (h *ReflectionToolHandler) handleRecordOutcome(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract decisionId (required)
	decisionId, ok := args["decisionId"].(string)
	if !ok || decisionId == "" {
		return createErrorResult("decisionId parameter is required"), nil, nil
	}

	// Get the original decision
	decision, err := h.reflectionStorage.GetReflectionByID(decisionId)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to get original decision: %s", err.Error())), nil, nil
	}

	// Create outcome reflection
	outcome := &storage.Reflection{
		Type:      "outcome",
		ChatID:    decision.ChatID,
		TaskID:    decision.TaskID,
		Timestamp: time.Now().UTC(),
		Data:      args,
		Tags:      []string{"outcome", "metacognition"},
	}

	// Store outcome
	outcomeId, err := h.reflectionStorage.StoreReflection(outcome)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to store outcome: %s", err.Error())), nil, nil
	}

	// Link decision and outcome
	err = h.reflectionStorage.LinkReflections(decisionId, outcomeId)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to link reflections: %s", err.Error())), nil, nil
	}

	// Extract calibration
	calibration := "unknown"
	if analysis, ok := args["analysis"].(map[string]interface{}); ok {
		if cal, ok := analysis["confidenceCalibration"].(string); ok {
			calibration = cal
		}
	}

	resultText := fmt.Sprintf("✓ Outcome recorded and linked to decision\n\nOutcome ID: %s\nDecision ID: %s\nCalibration: %s\n\nThe system now has decision → outcome data for learning.",
		outcomeId, decisionId, calibration)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"outcomeId":   outcomeId,
		"linked":      true,
		"calibration": calibration,
	}, nil
}

// handleExtractLesson handles the reflection_extract_lesson tool call
func (h *ReflectionToolHandler) handleExtractLesson(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract patternName (required)
	patternName, ok := args["patternName"].(string)
	if !ok || patternName == "" {
		return createErrorResult("patternName parameter is required"), nil, nil
	}

	// Extract confidence (default 0.8 for lessons)
	confidence := 0.8
	if conf, ok := args["confidence"].(float64); ok {
		confidence = conf
	}

	// Extract context
	contextStr, _ := args["context"].(string)

	// Create lesson reflection
	lesson := &storage.Reflection{
		Type:       "lesson",
		Timestamp:  time.Now().UTC(),
		Data:       args,
		Confidence: confidence,
		Tags:       []string{"lesson", "pattern", patternName},
	}

	// Store lesson
	lessonId, err := h.reflectionStorage.StoreReflection(lesson)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to store lesson: %s", err.Error())), nil, nil
	}

	// Update experience index for fast pattern matching
	err = h.reflectionStorage.UpsertExperienceIndex(patternName, contextStr, lessonId, confidence)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to update experience index: %s", err.Error())), nil, nil
	}

	resultText := fmt.Sprintf("✓ Lesson extracted and indexed\n\nLesson ID: %s\nPattern: %s\nConfidence: %.2f\n\nThis lesson is now available for future pattern matching.",
		lessonId, patternName, confidence)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"lessonId": lessonId,
		"indexed":  true,
		"pattern":  patternName,
	}, nil
}

// registerSuggestLessonFromError registers the reflection_suggest_lesson_from_error tool
func (h *ReflectionToolHandler) registerSuggestLessonFromError(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "reflection_suggest_lesson_from_error",
		Description: "Get automatic lesson suggestion from a recurring error pattern. When an error occurs multiple times, use this tool to get auto-populated lesson fields (problem, context, pattern name) for easy lesson extraction.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"errorPatternId": {
					Type:        "string",
					Description: "ID of the error pattern returned from error tracking",
				},
			},
			Required: []string{"errorPatternId"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleSuggestLessonFromError(args)
		return result, err
	})

	return nil
}

// handleSuggestLessonFromError handles the reflection_suggest_lesson_from_error tool call
func (h *ReflectionToolHandler) handleSuggestLessonFromError(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract errorPatternId (required)
	errorPatternId, ok := args["errorPatternId"].(string)
	if !ok || errorPatternId == "" {
		return createErrorResult("errorPatternId parameter is required"), nil, nil
	}

	// Get error pattern
	pattern, err := h.reflectionStorage.GetErrorSuggestion(errorPatternId)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to get error pattern: %s", err.Error())), nil, nil
	}

	// Check if lesson already extracted
	if pattern.LessonExtracted {
		return createErrorResult(fmt.Sprintf("Lesson already extracted for this error (Lesson ID: %s)", pattern.RelatedLesson)), nil, nil
	}

	// Create suggested pattern name from error type
	suggestedPattern := fmt.Sprintf("error-%s", pattern.ErrorType)

	// Build context string from recent errors
	contextStr := fmt.Sprintf("Error type: %s, Occurred %d times between %s and %s",
		pattern.ErrorType, pattern.Occurrences,
		pattern.FirstSeen.Format(time.RFC3339), pattern.LastSeen.Format(time.RFC3339))

	// Get most recent error for stack trace
	var recentError storage.ErrorInstance
	if len(pattern.RecentErrors) > 0 {
		recentError = pattern.RecentErrors[len(pattern.RecentErrors)-1]
	}

	suggestion := map[string]interface{}{
		"errorPatternId":   errorPatternId,
		"suggestedPattern": suggestedPattern,
		"problem":          pattern.MessagePattern,
		"context":          contextStr,
		"occurrences":      pattern.Occurrences,
		"errorType":        pattern.ErrorType,
		"recentError":      recentError,
	}

	resultText := fmt.Sprintf(`💡 Lesson suggestion from error pattern

Error Pattern ID: %s
Error Type: %s
Occurrences: %d
First Seen: %s
Last Seen: %s

Suggested Lesson Fields:
• Pattern Name: %s
• Problem: %s
• Context: %s

Recent Error:
%s

Use reflection_extract_lesson with these suggested fields, or customize them based on the root cause analysis.`,
		errorPatternId,
		pattern.ErrorType,
		pattern.Occurrences,
		pattern.FirstSeen.Format("2006-01-02 15:04:05"),
		pattern.LastSeen.Format("2006-01-02 15:04:05"),
		suggestedPattern,
		pattern.MessagePattern,
		contextStr,
		recentError.Message)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, suggestion, nil
}

// registerQueryRelevantLessons registers the reflection_query_relevant_lessons tool
func (h *ReflectionToolHandler) registerQueryRelevantLessons(server *mcp.Server) error {
	tool := &mcp.Tool{
		Name:        "reflection_query_relevant_lessons",
		Description: "Proactively query past lessons relevant to current situation. Use this BEFORE making decisions or taking risky actions to see if the system has learned from similar situations. Returns lessons ranked by relevance with confidence scores.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"situation": {
					Type:        "string",
					Description: "Describe the current situation, decision, or action you're about to take (e.g., 'about to modify database schema', 'implementing authentication', 'deploying to production')",
				},
				"limit": {
					Type:        "number",
					Description: "Maximum number of lessons to return (default: 5, max: 10)",
				},
			},
			Required: []string{"situation"},
		},
	}

	server.AddTool(tool, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := extractArguments(req)
		if err != nil {
			return createErrorResult(fmt.Sprintf("failed to extract arguments: %s", err.Error())), nil
		}
		result, _, err := h.handleQueryRelevantLessons(args)
		return result, err
	})

	return nil
}

// handleQueryRelevantLessons handles the reflection_query_relevant_lessons tool call
func (h *ReflectionToolHandler) handleQueryRelevantLessons(args map[string]interface{}) (*mcp.CallToolResult, interface{}, error) {
	// Extract situation (required)
	situation, ok := args["situation"].(string)
	if !ok || situation == "" {
		return createErrorResult("situation parameter is required"), nil, nil
	}

	// Extract limit (default 5, max 10)
	limit := 5
	if limitArg, ok := args["limit"].(float64); ok {
		limit = int(limitArg)
		if limit > 10 {
			limit = 10
		}
		if limit < 1 {
			limit = 1
		}
	}

	// Search for relevant lessons
	lessons, err := h.reflectionStorage.SearchLessonsByText(situation, limit)
	if err != nil {
		return createErrorResult(fmt.Sprintf("Failed to search lessons: %s", err.Error())), nil, nil
	}

	if len(lessons) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "No relevant past lessons found for this situation. This may be a novel scenario."},
			},
		}, map[string]interface{}{
			"lessonsFound": 0,
			"lessons":      []interface{}{},
		}, nil
	}

	// Format lessons as recommendations
	var resultText string
	resultText = fmt.Sprintf("💡 Found %d relevant lesson(s) from past experience:\n\n", len(lessons))

	lessonsData := make([]map[string]interface{}, len(lessons))
	for i, lesson := range lessons {
		// Extract lesson fields
		data := lesson.Data
		patternName, _ := data["patternName"].(string)
		problem, _ := data["problem"].(string)
		solution, _ := data["solution"].(string)
		contextStr, _ := data["context"].(string)
		antipattern, _ := data["antipattern"].(string)

		// Build recommendation text
		resultText += fmt.Sprintf("## Lesson %d: %s (Confidence: %.0f%%)\n\n", i+1, patternName, lesson.Confidence*100)
		resultText += fmt.Sprintf("**Problem:** %s\n\n", problem)
		resultText += fmt.Sprintf("**Solution:** %s\n\n", solution)

		if antipattern != "" {
			resultText += fmt.Sprintf("**⚠️ Don't:** %s\n\n", antipattern)
		}

		if contextStr != "" {
			resultText += fmt.Sprintf("**Context:** %s\n\n", contextStr)
		}

		resultText += fmt.Sprintf("**Lesson ID:** %s\n\n", lesson.ID)
		resultText += "---\n\n"

		// Add to structured data
		lessonsData[i] = map[string]interface{}{
			"id":          lesson.ID,
			"patternName": patternName,
			"problem":     problem,
			"solution":    solution,
			"antipattern": antipattern,
			"context":     contextStr,
			"confidence":  lesson.Confidence,
			"timestamp":   lesson.Timestamp,
		}
	}

	resultText += "💭 **Recommendation:** Review these lessons before proceeding. They represent knowledge gained from past experience in similar situations."

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: resultText},
		},
	}, map[string]interface{}{
		"lessonsFound": len(lessons),
		"lessons":      lessonsData,
		"situation":    situation,
	}, nil
}
