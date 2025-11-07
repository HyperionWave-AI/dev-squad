package aiservice

import (
	"os"
	"strings"
)

// workflow_guide.go
//
// This file contains the 5-step coordinator workflow guidance logic that helps
// AI agents (especially smaller models like Haiku) navigate the Hyperion coordinator
// workflow without getting lost or repeating steps.
//
// The 5-step coordinator workflow:
// 1. List existing tasks (coordinator_list_human_tasks)
// 2. Create human task (coordinator_create_human_task)
// 3. Search for relevant code (code_index_search)
// 4. Create agent task (create_agent_task)
// 5. Execute the specialist agent (execute_subagent)
//
// Key functions:
// - filterToolsByWorkflowState: Reduces tool count from 38 to 8-12 based on current phase
// - generateWorkflowStateGuidance: Provides explicit step-by-step instructions after each tool call
// - extractFilePathsFromCodeIndexResult: Extracts file paths from code search results
// - validateFilePaths: Verifies file paths exist before passing to agents
// - extractSessionIDFromSystemPrompt: Gets session ID for agent task handoff

var (
	// Phase 1 - Initial Assessment: List/search for existing work
	workflowPhase1Tools = []string{
		"coordinator_list_human_tasks",
		"coordinator_list_agent_tasks",
		"coordinator_get_agent_task",
		"coordinator_query_knowledge",
	}

	// Phase 2 - Task Creation: Create and manage tasks
	workflowPhase2Tools = []string{
		"coordinator_create_human_task",
		"create_agent_task",   // FIXED: No coordinator_ prefix
		"list_agent_tasks",    // FIXED: No coordinator_ prefix
		"execute_subagent",    // FIXED: No coordinator_ prefix
		"list_subagents",      // FIXED: No coordinator_ prefix
		"coordinator_update_task_status",
		"coordinator_update_todo_status",
	}

	// Phase 3 - Code Discovery: Search and index code
	workflowPhase3Tools = []string{
		"code_index_search",
		"code_index_status",
		"code_index_add_folder",
		"code_index_scan",
		"code_index_remove_folder",
	}

	// Phase 4 - Knowledge Management: Store and retrieve knowledge
	workflowPhase4Tools = []string{
		"knowledge_find",
		"coordinator_upsert_knowledge",
		"knowledge_store",
		"coordinator_get_popular_collections",
	}

	// Core tools - always included in every request
	coreTools = []string{
		"bash",
		"file_read",
		"file_write",
		"apply_patch",
		"coordinator_add_task_prompt_notes",
		"coordinator_update_task_prompt_notes",
		"coordinator_add_todo_prompt_notes",
		"coordinator_update_todo_prompt_notes",
	}
)

// filterToolsByWorkflowState analyzes tool call history and returns relevant tools
// This reduces token usage by ~70% (from 38 tools to 8-12 tools per request)
func filterToolsByWorkflowState(toolCallHistory []ToolResult) []string {
	// Start with core tools (always included)
	relevantTools := make(map[string]bool)
	for _, tool := range coreTools {
		relevantTools[tool] = true
	}

	// Analyze recent tool calls to determine workflow phase
	recentCalls := make(map[string]bool)
	lookbackLimit := 3 // Look at last 3 tool calls

	// Collect recent tool names
	for i := len(toolCallHistory) - 1; i >= 0 && len(recentCalls) < lookbackLimit; i-- {
		recentCalls[toolCallHistory[i].Name] = true
	}

	// Determine which phases to include based on recent activity
	includePhase1 := len(toolCallHistory) == 0 // First request - include listing tools
	includePhase2 := false
	includePhase3 := false
	includePhase4 := false

	// Check recent calls to determine active phases
	for toolName := range recentCalls {
		// If we just listed tasks, include task creation tools
		if toolName == "coordinator_list_human_tasks" || toolName == "list_agent_tasks" {
			includePhase2 = true
		}
		// If we just created a task, include code search tools
		if toolName == "coordinator_create_human_task" || toolName == "create_agent_task" {
			includePhase3 = true
		}
		// If we searched code, include knowledge and task creation tools
		if toolName == "code_index_search" {
			includePhase2 = true // Can create agent tasks
			includePhase4 = true // Can store knowledge
		}
		// If we're managing knowledge, keep those tools
		if toolName == "coordinator_upsert_knowledge" || toolName == "knowledge_store" || toolName == "knowledge_find" {
			includePhase4 = true
		}
		// If we're updating task status, keep task management tools
		if toolName == "coordinator_update_task_status" || toolName == "coordinator_update_todo_status" {
			includePhase2 = true
		}
	}

	// Add tools for active phases
	if includePhase1 {
		for _, tool := range workflowPhase1Tools {
			relevantTools[tool] = true
		}
	}
	if includePhase2 {
		for _, tool := range workflowPhase2Tools {
			relevantTools[tool] = true
		}
	}
	if includePhase3 {
		for _, tool := range workflowPhase3Tools {
			relevantTools[tool] = true
		}
	}
	if includePhase4 {
		for _, tool := range workflowPhase4Tools {
			relevantTools[tool] = true
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(relevantTools))
	for tool := range relevantTools {
		result = append(result, tool)
	}

	return result
}

// extractFilePathsFromCodeIndexResult extracts file paths from code_index_search result
func extractFilePathsFromCodeIndexResult(result ToolResult) []string {
	if result.Error != "" {
		return nil
	}

	outputMap, ok := result.Output.(map[string]interface{})
	if !ok {
		return nil
	}

	resultsArray, ok := outputMap["results"].([]interface{})
	if !ok {
		return nil
	}

	var paths []string
	for _, item := range resultsArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		if path, ok := itemMap["path"].(string); ok && path != "" {
			paths = append(paths, path)
		}
	}

	return paths
}

// validateFilePaths checks if all provided file paths exist on the filesystem
// Returns: (validPaths, invalidPaths)
func validateFilePaths(paths []string) ([]string, []string) {
	var validPaths []string
	var invalidPaths []string

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			validPaths = append(validPaths, path)
		} else {
			invalidPaths = append(invalidPaths, path)
		}
	}

	return validPaths, invalidPaths
}

// extractSessionIDFromSystemPrompt extracts the session ID from the system prompt message.
// The system prompt contains: "SESSION CONTEXT:\n- **CURRENT CHAT SESSION ID**: {sessionId}"
func extractSessionIDFromSystemPrompt(systemPrompt string) string {
	// Look for the session ID pattern in the system prompt
	// Pattern: "CURRENT CHAT SESSION ID**: {sessionId}"
	const marker = "CURRENT CHAT SESSION ID**: "
	startIdx := strings.Index(systemPrompt, marker)
	if startIdx == -1 {
		return ""
	}

	// Move past the marker
	startIdx += len(marker)

	// Find the end of the session ID (newline or end of string)
	endIdx := strings.IndexAny(systemPrompt[startIdx:], "\n\r")
	if endIdx == -1 {
		// No newline found, use rest of string
		return strings.TrimSpace(systemPrompt[startIdx:])
	}

	return strings.TrimSpace(systemPrompt[startIdx : startIdx+endIdx])
}

