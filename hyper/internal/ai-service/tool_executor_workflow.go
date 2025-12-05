package aiservice

import (
	"fmt"
	"log"
	"strings"
)

// tool_executor_workflow.go - Workflow State Management for Tool Execution
//
// This file contains the workflow state tracking logic for the 5-step coordinator workflow.
// It tracks progress through the workflow and provides validation and guidance.
//
// The 5-step coordinator workflow:
// 1. List existing tasks (coordinator_list_human_tasks) -> Step 0 -> 1
// 2. Create human task (coordinator_create_human_task) -> Step 1 -> 2
// 3. Search for relevant code (code_index_search) -> Step 2 -> 3
// 4. Create agent task (create_agent_task) -> Step 3 -> 4
// 5. Execute the specialist agent (execute_subagent) -> Step 4 -> 5

// WorkflowState tracks coordinator workflow progress across tool executions.
// This helps enforce the correct sequence of tool calls and provides guidance.
type WorkflowState struct {
	Step                   int    // 0=initial, 1=listed, 2=created, 3=searched, 4=agent_task, 5=done
	HumanTaskId            string // Task ID from step 2
	SearchCompleted        bool   // Prevents multiple searches
	AgentTaskId            string // Agent task ID from step 4
	LastCreatedHumanTaskId string // FIX #9: Cache for instant validation
	LastCreatedAgentTaskId string // FIX #10: Cache for instant validation
}

// NewWorkflowState creates a new workflow state initialized at step 0.
func NewWorkflowState() *WorkflowState {
	return &WorkflowState{
		Step:            0,
		HumanTaskId:     "",
		SearchCompleted: false,
		AgentTaskId:     "",
	}
}

// ValidateTool checks if a tool call is allowed in the current workflow state.
// This is only enforced for fallback models; primary models have no restrictions.
// Returns (allowed, blockMessage).
func (ws *WorkflowState) ValidateTool(toolName string, usingFallback bool) (bool, string) {
	if !usingFallback {
		return true, "" // No enforcement for primary model
	}

	switch toolName {
	case "coordinator_list_human_tasks":
		if ws.Step == 0 || ws.Step == 1 || ws.Step == 2 {
			// Allow listing tasks at any early step to retrieve exact taskId
			return true, ""
		}
		return false, "❌ BLOCKED: You already have taskId. NEXT: Call appropriate tool for current step."

	case "coordinator_create_human_task":
		if ws.Step == 1 && ws.HumanTaskId == "" {
			return true, ""
		}
		if ws.HumanTaskId != "" {
			// State-aware guidance based on workflow progress
			if ws.SearchCompleted {
				return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'create_agent_task'.", ws.HumanTaskId)
			}
			return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'code_index_search'.", ws.HumanTaskId)
		}
		return false, "❌ BLOCKED: Call 'coordinator_list_human_tasks' first."

	case "code_index_search":
		if ws.Step == 2 && ws.HumanTaskId != "" && !ws.SearchCompleted {
			return true, ""
		}
		if ws.SearchCompleted {
			return false, "❌ BLOCKED: Search done. NEXT: Call 'create_agent_task'."
		}
		return false, "❌ BLOCKED: Create human task first."

	case "create_agent_task":
		if ws.Step == 3 && ws.HumanTaskId != "" && ws.SearchCompleted {
			return true, ""
		}
		return false, "❌ BLOCKED: Run 'code_index_search' first to get file paths."

	case "execute_subagent":
		if ws.Step == 4 && ws.AgentTaskId != "" {
			return true, ""
		}
		return false, "❌ BLOCKED: Call 'create_agent_task' first."
	}

	return true, ""
}

// UpdateAfterToolExecution updates the workflow state after a successful tool execution.
// This should be called after each tool completes successfully.
func (ws *WorkflowState) UpdateAfterToolExecution(toolName string, result ToolResult, logPrefix string) {
	if result.Error != "" {
		return // Don't update state on error
	}

	switch toolName {
	case "coordinator_list_human_tasks":
		if ws.Step == 0 {
			ws.Step = 1
			log.Printf("[%s Workflow State] Step 1 complete: listed tasks", logPrefix)
		}

	case "coordinator_create_human_task":
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			if taskID, hasTaskID := outputMap["taskId"].(string); hasTaskID && taskID != "" {
				ws.Step = 2
				ws.HumanTaskId = taskID
				ws.LastCreatedHumanTaskId = taskID // FIX #9: Cache for instant validation
				log.Printf("[%s Workflow State] Step 2 complete: created human task %s", logPrefix, taskID)
			} else if similarTasksFound, _ := outputMap["similarTasksFound"].(bool); similarTasksFound {
				// Case 2: Similar task found - use existing task instead of creating new one
				if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
					if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
						if existingTaskID, ok := firstTask["taskId"].(string); ok && existingTaskID != "" {
							ws.Step = 2
							ws.HumanTaskId = existingTaskID
							ws.LastCreatedHumanTaskId = existingTaskID // FIX #9: Cache for instant validation
							log.Printf("[%s Workflow State] Step 2 complete: using existing similar task %s", logPrefix, existingTaskID)
						}
					}
				}
			}
		}

	case "code_index_search":
		ws.Step = 3
		ws.SearchCompleted = true
		log.Printf("[%s Workflow State] Step 3 complete: code search done", logPrefix)

	case "create_agent_task":
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			if agentTaskID, hasAgentTaskID := outputMap["taskId"].(string); hasAgentTaskID && agentTaskID != "" {
				ws.Step = 4
				ws.AgentTaskId = agentTaskID
				ws.LastCreatedAgentTaskId = agentTaskID // FIX #10: Cache for instant validation
				log.Printf("[%s Workflow State] Step 4 complete: created agent task %s", logPrefix, agentTaskID)
			}
		}

	case "execute_subagent":
		ws.Step = 5
		log.Printf("[%s Workflow State] Step 5 complete: subagent launched", logPrefix)
	}
}

// GetGuidance generates workflow state guidance for the fallback model.
// This helps smaller models like Haiku understand the workflow and what to do next.
func (ws *WorkflowState) GetGuidance(toolName string, result ToolResult, toolCallCount int, sessionID string) string {
	// Skip guidance if tool failed
	if result.Error != "" {
		return ""
	}

	var guidance string

	switch toolName {
	case "coordinator_list_human_tasks":
		// Step 1 complete - guide to Step 2
		guidance = "✅ STEP 1 COMPLETE: You checked existing tasks.\n" +
			"➡️ NEXT ACTION: Call 'coordinator_create_human_task' with the user's exact request.\n" +
			"   Example: {\"prompt\": \"<user's exact words>\"}\n" +
			"🔒 DO NOT call coordinator_list_human_tasks again - you already have the results."

	case "coordinator_create_human_task":
		// Step 2 complete - extract taskId and guide to Step 3
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			taskID, hasTaskID := outputMap["taskId"].(string)
			similarTasksFound, _ := outputMap["similarTasksFound"].(bool)

			if similarTasksFound {
				// Duplicate detected - extract first similar task's ID and proceed
				var firstTaskID string
				if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
					if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
						if tid, ok := firstTask["taskId"].(string); ok {
							firstTaskID = tid
						}
					}
				}

				if firstTaskID != "" {
					// Found a similar task - use it and proceed to Step 2
					guidance = fmt.Sprintf("⚠️ SIMILAR TASK FOUND: A task with similar intent already exists.\n"+
						"📝 SAVE THIS: humanTaskId = \"%s\" (using existing similar task)\n"+
						"➡️ NEXT ACTION: Call 'code_index_search' ONCE to find relevant files.\n"+
						"   Example: {\"query\": \"<what user wants to change>\", \"limit\": 15}\n"+
						"🔒 DO NOT call coordinator_create_human_task again - you already have a taskId.\n"+
						"💡 To create a NEW task instead, call coordinator_create_human_task with forceCreate=true", firstTaskID)
				} else {
					// No task ID found in similar tasks - tell model to force create
					guidance = "⚠️ DUPLICATE TASK DETECTED: Similar tasks exist but no ID was found.\n" +
						"➡️ NEXT ACTION: Call coordinator_create_human_task with forceCreate=true to create anyway.\n" +
						"   Example: {\"prompt\": \"<user's exact words>\", \"forceCreate\": true}"
				}
			} else if hasTaskID {
				// Task created successfully - guide to Step 3
				guidance = fmt.Sprintf("✅ STEP 2 COMPLETE: Human task created successfully.\n"+
					"📝 SAVE THIS: humanTaskId = \"%s\"\n"+
					"➡️ NEXT ACTION: Call 'code_index_search' ONCE to find relevant files.\n"+
					"   Example: {\"query\": \"<what user wants to change>\", \"limit\": 15}\n"+
					"🔒 DO NOT call coordinator_create_human_task again - you already have the taskId.\n"+
					"🔒 You will need this taskId for Step 4 (create_agent_task).", taskID)
			}
		}

	case "code_index_search":
		// Step 3 complete - extract file paths and guide to Step 4
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			filePathsRaw, hasFilePaths := outputMap["FILE_PATHS_TO_USE"]
			_, hasResults := outputMap["results"]

			if hasFilePaths || hasResults {
				filePathsCount := 0
				if filePaths, ok := filePathsRaw.([]interface{}); ok {
					filePathsCount = len(filePaths)
				}

				guidance = fmt.Sprintf("✅ STEP 3 COMPLETE: Code search returned %d file(s).\n"+
					"📝 EXTRACT: Copy file paths from FILE_PATHS_TO_USE array above.\n"+
					"➡️ NEXT ACTION: Call 'create_agent_task' with:\n"+
					"   - humanTaskId: \"<taskId from Step 2>\"\n"+
					"   - agentName: \"ui-dev\" (for UI changes) or \"go-dev\" (for backend)\n"+
					"   - role: \"Brief mission description\"\n"+
					"   - contextSummary: \"WHAT to change, WHERE (file:line from search), HOW\"\n"+
					"   - filesModified: [\"<COPY exact paths from FILE_PATHS_TO_USE>\"]\n"+
					"   - todos: [{description: \"Implement X in file Y\", filePath, contextHint}]\n\n"+
					"🚨 CRITICAL:\n"+
					"   • filesModified MUST NOT be empty - populate with paths from FILE_PATHS_TO_USE\n"+
					"   • TODOs must be implementation steps, NOT discovery steps\n"+
					"   • DO NOT create TODOs like 'Search for...' or 'Find...'\n"+
					"   • Subagent CANNOT run code_index_search - it's blocked in write-only mode\n\n"+
					"🔒 DO NOT call code_index_search again - you already have the file paths.\n"+
					"🔒 Use EXACT paths from FILE_PATHS_TO_USE array - do NOT type paths manually!", filePathsCount)
			}
		}

	case "create_agent_task":
		// Step 4 complete - extract agent task ID and guide to Step 5
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			agentTaskID, hasAgentTaskID := outputMap["taskId"].(string)
			agentName, _ := outputMap["agentName"].(string)

			if hasAgentTaskID {
				// Use actual session ID instead of placeholder
				parentChatIDValue := sessionID
				if parentChatIDValue == "" {
					parentChatIDValue = "<session-id-not-found>"
				}
				guidance = fmt.Sprintf("✅ STEP 4 COMPLETE: Agent task created successfully.\n"+
					"📝 Agent Task ID: \"%s\"\n"+
					"➡️ NEXT ACTION (FINAL): Call 'execute_subagent' to launch the agent:\n"+
					"   {\"agentTaskId\": \"%s\", \"parentChatId\": \"%s\"}\n"+
					"🔒 DO NOT call create_agent_task again - the task is created.\n"+
					"✅ After execute_subagent, the %s agent will implement the changes - YOU ARE DONE!", agentTaskID, agentTaskID, parentChatIDValue, agentName)
			}
		}

	case "execute_subagent":
		// Step 5 complete - extract agentTaskId and tell coordinator to STOP
		if outputMap, ok := result.Output.(map[string]interface{}); ok {
			agentTaskID, hasAgentTaskID := outputMap["agentTaskId"].(string)
			agentName, _ := outputMap["agentName"].(string)
			subchatID, _ := outputMap["subchatId"].(string)

			if hasAgentTaskID {
				guidance = fmt.Sprintf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
					"✅ WORKFLOW COMPLETE - YOUR JOB IS DONE!\n"+
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
					"The %s agent is executing your request in background.\n"+
					"   • Agent Task ID: %s\n"+
					"   • Subchat ID: %s\n\n"+
					"🛑 STOP HERE - DO NOT CALL ANY MORE TOOLS\n"+
					"🛑 DO NOT call list_agent_tasks, coordinator_get_agent_task, or any monitoring tools\n"+
					"🛑 DO NOT try to check status - the agent is working independently\n\n"+
					"✅ YOUR ONLY ACTION: Inform the user that work has begun.\n"+
					"   Example: \"I've delegated this to the %s agent. They're working on it now.\"\n\n"+
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━",
					agentName, agentTaskID, subchatID, agentName)
			} else {
				// Fallback if agentTaskId not found
				guidance = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
					"✅ WORKFLOW COMPLETE - YOUR JOB IS DONE!\n" +
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
					"The specialist agent is executing the request in background.\n\n" +
					"🛑 STOP HERE - DO NOT CALL ANY MORE TOOLS\n" +
					"✅ YOUR ONLY ACTION: Inform the user that work has begun.\n\n" +
					"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
			}
		}
	}

	// If we generated guidance, add a clear separator
	if guidance != "" {
		guidance = "\n" + strings.Repeat("━", 70) + "\n" +
			"🤖 WORKFLOW STATE TRACKER (for your guidance)\n" +
			strings.Repeat("━", 70) + "\n" +
			guidance + "\n" +
			strings.Repeat("━", 70)
	}

	return guidance
}

// GetHumanTaskIdForInjection returns the human task ID if available, for injection into prompts.
func (ws *WorkflowState) GetHumanTaskIdForInjection() string {
	return ws.HumanTaskId
}

// Reset clears all workflow state. Useful for testing or starting a new workflow.
func (ws *WorkflowState) Reset() {
	ws.Step = 0
	ws.HumanTaskId = ""
	ws.SearchCompleted = false
	ws.AgentTaskId = ""
	ws.LastCreatedHumanTaskId = ""
	ws.LastCreatedAgentTaskId = ""
}
