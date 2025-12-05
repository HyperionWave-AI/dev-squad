package aiservice

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// tool_executor_validation.go - Task and Path Validation for Tool Execution
//
// This file contains validation logic for task IDs and file paths before tool execution.
// It ensures that:
// - Human task IDs exist before creating agent tasks
// - File paths from code search actually exist on the filesystem
// - Agent task IDs are valid before executing subagents

// TaskValidator validates task IDs before tool execution.
// It tracks validation attempts and provides auto-correction for common mistakes.
type TaskValidator struct {
	taskIdValidationAttempts int             // Counter for validation attempts
	pathValidationRetries    map[string]bool // Track path validation retries
	toolRegistry             *ToolRegistry   // Reference to execute validation calls
}

// NewTaskValidator creates a new TaskValidator.
func NewTaskValidator(registry *ToolRegistry) *TaskValidator {
	return &TaskValidator{
		taskIdValidationAttempts: 0,
		pathValidationRetries:    make(map[string]bool),
		toolRegistry:             registry,
	}
}

// ValidateHumanTaskId validates that a human task ID exists in the database.
// It uses the cached ID for instant validation if available, otherwise queries the database.
// Returns (exists, error).
func (tv *TaskValidator) ValidateHumanTaskId(ctx context.Context, humanTaskId string, cachedId string) (bool, error) {
	// FIX #9: Check cached taskId FIRST (instant validation, no database needed)
	if humanTaskId == cachedId && cachedId != "" {
		log.Printf("[TaskId Validator] ✅ Instant validation: humanTaskId '%s' matches cached task (no DB lookup needed)", humanTaskId)
		return true, nil
	}

	// Not the cached ID - do database validation with retry
	// FIX #8: Retry validation to handle MongoDB eventual consistency
	for attempt := 0; attempt < 3; attempt++ {
		// Call coordinator_list_human_tasks to get all tasks
		listTasksCall := ToolCall{
			ID:   "taskid_validation",
			Name: "coordinator_list_human_tasks",
			Args: map[string]interface{}{},
		}
		listResult := tv.toolRegistry.ExecuteToolCall(ctx, listTasksCall)

		if listResult.Error == "" {
			if outputMap, ok := listResult.Output.(map[string]interface{}); ok {
				if tasks, ok := outputMap["tasks"].([]interface{}); ok {
					for _, task := range tasks {
						if taskMap, ok := task.(map[string]interface{}); ok {
							// Check taskId field (matching HumanTask JSON schema with json:"taskId" tag)
							if taskId, ok := taskMap["taskId"].(string); ok && taskId == humanTaskId {
								return true, nil
							}
						}
					}
				}
			}
		}

		// Not found yet - retry with exponential backoff
		if attempt < 2 {
			sleepDuration := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
			log.Printf("[TaskId Validator] Task '%s' not found yet - retrying after %v (attempt %d/3)",
				humanTaskId, sleepDuration, attempt+1)
			time.Sleep(sleepDuration)
		}
	}

	return false, nil
}

// RecordInvalidTaskIdAttempt increments the validation attempt counter.
// Returns the current attempt count.
func (tv *TaskValidator) RecordInvalidTaskIdAttempt() int {
	tv.taskIdValidationAttempts++
	return tv.taskIdValidationAttempts
}

// GetTaskIdValidationAttempts returns the current attempt count.
func (tv *TaskValidator) GetTaskIdValidationAttempts() int {
	return tv.taskIdValidationAttempts
}

// ShouldStopAfterInvalidTaskId returns true if we've exceeded the max validation attempts.
func (tv *TaskValidator) ShouldStopAfterInvalidTaskId() bool {
	return tv.taskIdValidationAttempts >= 3
}

// GetInvalidTaskIdError returns the error message after max attempts.
func (tv *TaskValidator) GetInvalidTaskIdError(humanTaskId string) string {
	return fmt.Sprintf("❌ CRITICAL ERROR: create_agent_task called with INVALID humanTaskId 3 times.\n\n"+
		"INVALID humanTaskId PROVIDED: '%s'\n\n"+
		"🛑 The humanTaskId you are providing DOES NOT EXIST in the database.\n"+
		"✅ This means:\n"+
		"   1. You are hallucinating or generating the task ID instead of copying it from tool responses\n"+
		"   2. The task ID may have been typed incorrectly\n"+
		"   3. The task may have been deleted\n\n"+
		"🔍 CORRECT APPROACH:\n"+
		"   1. Call coordinator_list_human_tasks to see ALL existing tasks\n"+
		"   2. Find the task that matches the user's request\n"+
		"   3. COPY the EXACT 'taskId' field from the task object\n"+
		"   4. Use that EXACT UUID as 'humanTaskId' when calling create_agent_task\n\n"+
		"❌ DO NOT:\n"+
		"   - Generate UUIDs yourself\n"+
		"   - Use descriptive names like 'add-feature' or 'fix-bug'\n"+
		"   - Try to guess or construct the task ID\n\n"+
		"⚠️ Execution stopped after 3 invalid attempts. Please review the instructions above and try again.",
		humanTaskId)
}

// GetInvalidTaskIdWarning returns a warning message for retry attempts.
func (tv *TaskValidator) GetInvalidTaskIdWarning(humanTaskId string) string {
	return fmt.Sprintf("⚠️  TASK ID VALIDATION ERROR (Attempt %d/3):\n"+
		"The humanTaskId '%s' DOES NOT EXIST in the database.\n"+
		"🔄 You MUST call coordinator_list_human_tasks to get the correct taskId.\n\n",
		tv.taskIdValidationAttempts, humanTaskId)
}

// CreateBlockedResult creates a ToolResult for a blocked create_agent_task call.
func (tv *TaskValidator) CreateBlockedResult(toolCallID string, humanTaskId string) ToolResult {
	return ToolResult{
		ID:   toolCallID,
		Name: "create_agent_task",
		Args: nil,
		Output: map[string]interface{}{
			"_validationError": "BLOCKED",
			"_reason":          fmt.Sprintf("Invalid humanTaskId: '%s' does not exist", humanTaskId),
			"NEXT":             "Call coordinator_list_human_tasks to see all tasks, find the correct task, and COPY its exact 'taskId' field",
		},
		Error: fmt.Sprintf("❌ BLOCKED: humanTaskId '%s' is INVALID (does not exist). "+
			"You MUST call coordinator_list_human_tasks to get all tasks and find the EXACT taskId. "+
			"DO NOT hallucinate or generate task IDs. COPY the exact UUID from the tool response. "+
			"This is attempt %d/3. After 3 failures, execution will stop.",
			humanTaskId, tv.taskIdValidationAttempts),
	}
}

// HasRetiedPathValidation checks if we've already retried path validation for this query.
func (tv *TaskValidator) HasRetriedPathValidation(query interface{}) bool {
	retryKey := fmt.Sprintf("code_index_search_retry_%v", query)
	return tv.pathValidationRetries[retryKey]
}

// RecordPathValidationRetry marks that we've retried path validation for this query.
func (tv *TaskValidator) RecordPathValidationRetry(query interface{}) {
	retryKey := fmt.Sprintf("code_index_search_retry_%v", query)
	tv.pathValidationRetries[retryKey] = true
}

// GetInvalidPathsError returns the error message when path validation fails after retry.
func GetInvalidPathsError(invalidPaths []string) string {
	return fmt.Sprintf("❌ CRITICAL ERROR: code_index_search returned invalid file paths (files don't exist on filesystem):\n\n"+
		"INVALID PATHS:\n"+
		"%s\n\n"+
		"🛑 Even after retrying the search, the code index is returning paths to files that don't exist.\n"+
		"✅ This means:\n"+
		"   1. The code index may be stale or out of sync with the filesystem\n"+
		"   2. The files may have been moved or deleted\n"+
		"   3. The search query may be finding old/archived files\n\n"+
		"🔍 NEXT STEPS:\n"+
		"   - Ask the user to verify the file locations\n"+
		"   - Try a different search query\n"+
		"   - Check if files exist in a different directory\n"+
		"   - Consider re-indexing the codebase\n\n"+
		"DO NOT proceed with create_agent_task using these invalid paths!",
		strings.Join(invalidPaths, "\n"))
}

// GetInvalidPathsWarning returns a warning message for first path validation failure.
func GetInvalidPathsWarning(invalidPaths []string) string {
	return fmt.Sprintf("⚠️  FILE PATH VALIDATION WARNING:\n"+
		"code_index_search returned %d invalid file paths (files don't exist).\n"+
		"🔄 Automatically retrying search with refined query...\n\n"+
		"Invalid paths found:\n%s\n\n",
		len(invalidPaths), strings.Join(invalidPaths, "\n"))
}

// InjectPathValidationWarning adds validation warning to tool result.
func InjectPathValidationWarning(result *ToolResult, invalidPaths, validPaths []string, filePaths int) {
	if outputMap, ok := result.Output.(map[string]interface{}); ok {
		outputMap["_validationWarning"] = fmt.Sprintf(
			"⚠️ VALIDATION FAILED: %d out of %d file paths are INVALID (files don't exist on filesystem). "+
				"You MUST retry code_index_search with a different query to find the CORRECT file paths. "+
				"DO NOT proceed with create_agent_task using these invalid paths. "+
				"Invalid paths: %s",
			len(invalidPaths), filePaths, strings.Join(invalidPaths, "\n"))
		outputMap["invalidPaths"] = invalidPaths
		outputMap["validPaths"] = validPaths
		result.Output = outputMap
	}
}

// Reset clears all validation state. Useful for testing.
func (tv *TaskValidator) Reset() {
	tv.taskIdValidationAttempts = 0
	tv.pathValidationRetries = make(map[string]bool)
}
