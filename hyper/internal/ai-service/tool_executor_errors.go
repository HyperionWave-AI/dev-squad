package aiservice

import (
	"strings"
)

// tool_executor_errors.go - Error Recovery Guidance for Tool Execution
//
// This file contains error recovery guidance that helps the AI model
// understand how to recover from common tool execution errors.
// Each error type has specific, actionable instructions.

// getErrorRecoveryGuidance provides specific recovery instructions based on error type.
// This guidance is injected into tool error responses to help the AI self-correct.
func getErrorRecoveryGuidance(toolName string, errorMsg string, args map[string]interface{}) string {
	errorLower := strings.ToLower(errorMsg)

	// Validation errors - humanTaskId invalid
	if strings.Contains(errorLower, "invalid") && strings.Contains(errorLower, "humantaskid") {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  Call coordinator_list_human_tasks RIGHT NOW to see all available tasks\n" +
			"➡️  Find the correct task in the response\n" +
			"➡️  Copy the EXACT 'taskId' field from that task\n" +
			"➡️  Then retry create_agent_task with the correct taskId\n\n" +
			"⚠️  DO NOT explain this to the user - MAKE THE TOOL CALL IMMEDIATELY!\n" +
			"⚠️  DO NOT generate or guess task IDs - copy them exactly from the list!"
	}

	// File path errors
	if strings.Contains(errorLower, "path does not exist") || strings.Contains(errorLower, "file not found") {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  Check the FILE_PATHS_TO_USE array from your code_index_search results\n" +
			"➡️  Use ONLY paths from that array - do not modify or type paths manually\n" +
			"➡️  Retry the tool call RIGHT NOW with the correct path from FILE_PATHS_TO_USE\n\n" +
			"⚠️  DO NOT explain this to the user - MAKE THE TOOL CALL IMMEDIATELY!"
	}

	// Missing required parameters
	if strings.Contains(errorLower, "required") && (strings.Contains(errorLower, "must be") || strings.Contains(errorLower, "is required")) {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  A required parameter is missing or has the wrong type\n" +
			"➡️  Check the error message to see which parameter is missing\n" +
			"➡️  Add the missing parameter with the correct value\n" +
			"➡️  Retry the tool call RIGHT NOW with all required parameters\n\n" +
			"⚠️  DO NOT explain this to the user - MAKE THE TOOL CALL IMMEDIATELY!"
	}

	// TODO validation errors (discovery keywords)
	if strings.Contains(errorLower, "todo validation failed") || strings.Contains(errorLower, "discovery keyword") {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  Your TODO contains forbidden words like 'search', 'find', 'locate', 'explore'\n" +
			"➡️  Call code_index_search RIGHT NOW to find the files yourself\n" +
			"➡️  Then retry create_agent_task with implementation-only TODOs like:\n" +
			"   - 'Add validation to AuthForm.tsx line 45'\n" +
			"   - 'Update API call in dashboard.go line 120'\n\n" +
			"⚠️  DO NOT explain this to the user - MAKE THE TOOL CALL IMMEDIATELY!\n" +
			"⚠️  Remove ALL discovery words from your TODOs!"
	}

	// Similar tasks found
	if strings.Contains(errorLower, "similar") && strings.Contains(errorLower, "task") {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  Similar tasks were found in the response\n" +
			"➡️  Call coordinator_create_human_task RIGHT NOW with forceCreate=true to create a new task\n" +
			"➡️  OR use the taskId from the similarTasks array if appropriate\n\n" +
			"⚠️  DO NOT ask the user - MAKE THE TOOL CALL IMMEDIATELY with forceCreate=true!"
	}

	// Code search errors
	if toolName == "code_index_search" {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  Code search failed or returned no results\n" +
			"➡️  PROCEED ANYWAY - call create_agent_task RIGHT NOW without file paths\n" +
			"➡️  The agent will find the files during implementation\n\n" +
			"⚠️  DO NOT ask the user - MAKE THE TOOL CALL IMMEDIATELY!\n" +
			"⚠️  DO NOT retry search - proceed to create_agent_task NOW!"
	}

	// Generic validation errors
	if strings.Contains(errorLower, "validation") || strings.Contains(errorLower, "invalid") {
		return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
			"➡️  One of your parameters failed validation\n" +
			"➡️  Read the error message carefully to see what's wrong\n" +
			"➡️  Correct the parameter value\n" +
			"➡️  Retry the tool call RIGHT NOW with the corrected parameters\n\n" +
			"⚠️  DO NOT explain this to the user - MAKE THE TOOL CALL IMMEDIATELY!"
	}

	// Generic error guidance
	return "🔧 IMMEDIATE ACTION REQUIRED - MAKE THIS TOOL CALL NOW:\n" +
		"➡️  Read the error message above carefully\n" +
		"➡️  Identify what went wrong\n" +
		"➡️  Correct the issue based on the error details\n" +
		"➡️  Retry the tool call RIGHT NOW with corrected parameters\n\n" +
		"⚠️  DO NOT explain this to the user - MAKE THE TOOL CALL IMMEDIATELY!\n" +
		"⚠️  If error persists after 1 retry, try a different approach but KEEP MAKING TOOL CALLS!"
}
