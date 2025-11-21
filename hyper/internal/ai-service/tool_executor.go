package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// Debug logger that writes to a specific file for investigation
var debugLogFile *os.File

func init() {
	var err error
	debugLogFile, err = os.OpenFile("/tmp/tool_executor_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open debug log file: %v", err)
	}
}

func debugLog(format string, args ...interface{}) {
	if debugLogFile != nil {
		timestamp := time.Now().Format("15:04:05.000")
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(debugLogFile, "[%s] %s\n", timestamp, msg)
		debugLogFile.Sync() // Flush immediately
	}
}

// tool_executor.go - Tool Execution Orchestration
//
// This file contains the core tool execution logic for the AI service, handling:
//   - Tool-enabled streaming chat with circuit breakers and loop detection
//   - Filtered tool execution for subagents (security-constrained context)
//   - Rate limit detection and automatic fallback to backup models
//   - Result caching to prevent duplicate tool executions
//   - Context window management with sliding windows
//   - Workflow state tracking and validation
//   - Progressive warning system for loop prevention
//
// The orchestration supports two execution modes:
//   1. StreamChatWithTools: Full tool access for coordinator agents
//   2. StreamChatWithToolsFiltered: Restricted tool access for subagents
//
// Both methods implement sophisticated circuit breakers, caching, and validation
// to ensure reliable tool execution and prevent infinite loops.

// getErrorRecoveryGuidance provides specific recovery instructions based on error type
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

// StreamChatWithTools sends messages to AI provider with tool execution support.
// This is the main entry point for coordinator agents that need full tool access.
//
// Features:
//   - Automatic rate limit detection with fallback to backup models
//   - Tool result caching to prevent duplicate executions
//   - Circuit breakers to detect and stop infinite loops
//   - Sliding context window to manage token limits
//   - Workflow state tracking and validation
//   - Smart tool filtering to reduce token usage by 70%
//   - Progressive warnings to guide AI away from loops
//
// Parameters:
//   - ctx: Context with identity and request ID
//   - messages: Conversation history
//   - maxToolCalls: Maximum tool calls before stopping (default 5)
//
// Returns:
//   - Channel of StreamEvents (tokens, tool calls, tool results, errors)
//   - Error if initial validation fails
func (s *ChatService) StreamChatWithTools(ctx context.Context, messages []Message, maxToolCalls int) (<-chan StreamEvent, error) {
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Tool-enabled request from %s (%s) - RequestID: %s - Provider: %s Model: %s",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model)
	} else {
		log.Printf("[ChatService] Tool-enabled request (no identity) - RequestID: %s - Provider: %s Model: %s",
			requestID, s.config.Provider, s.config.Model)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Try to switch back to primary model if we're using fallback
	if s.usingFallback {
		log.Printf("[Rate Limit Recovery] Attempting to switch back to primary model: %s", s.originalModel)

		// Restore original configuration
		s.config.Model = s.originalModel
		s.config.Provider = s.originalProvider
		s.config.APIKey = s.originalAPIKey

		// Recreate provider with original config
		primaryProvider, err := NewChatProvider(s.config)
		if err != nil {
			log.Printf("[Rate Limit Recovery] Failed to recreate primary provider, staying on fallback: %v", err)
		} else {
			// Update provider and mark as no longer using fallback
			s.provider = primaryProvider
			s.usingFallback = false
			log.Printf("[Rate Limit Recovery] Successfully switched back to primary model: %s", s.originalModel)
		}
	}

	// Default max tool calls to prevent loops
	if maxToolCalls <= 0 {
		maxToolCalls = 5
	}

	// Create output channel for events
	eventChan := make(chan StreamEvent, 100)

	// Get tools for LangChain
	tools := s.toolRegistry.GetToolsForLangChain()

	// Check if provider supports tools
	supportsTools := false
	if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
		supportsTools = toolProvider.SupportsTools()
	}

	if !supportsTools || len(tools) == 0 {
		// Fallback to text-only streaming
		log.Printf("[ChatService] Provider doesn't support tools or no tools registered - RequestID: %s", requestID)
		go func() {
			defer close(eventChan)
			textChan, err := s.provider.StreamChat(ctx, messages)
			if err != nil {
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}
			for chunk := range textChan {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
			}
		}()
		return eventChan, nil
	}

	// Start tool-enabled streaming
	go func() {
		defer close(eventChan)
		debugLog("========== GOROUTINE STARTED ==========")
		debugLog("Request ID: %s, MaxToolCalls: %d", requestID, maxToolCalls)

		toolCallCount := 0
		iterationCount := 0
		currentMessages := append([]Message{}, messages...) // Copy messages
		debugLog("Initial message count: %d", len(currentMessages))

		// Tool result cache: prevent duplicate tool executions
		resultCache := NewToolResultCache()

		// Circuit breaker: track recent tool calls to detect infinite loops
		recentToolCalls := make([]string, 0, 10)
		consecutiveFailures := 0     // Track CONSECUTIVE failures of the same tool+args
		lastFailedSignature := ""    // Signature of the last failed tool call
		pathValidationRetries := make(map[string]bool) // Track file path validation retries for code_index_search
		taskIdValidationAttempts := 0                  // Track taskId validation attempts for create_agent_task (max 3)
		lastCreatedHumanTaskId := ""                   // FIX #9: Cache taskId from coordinator_create_human_task for instant validation
		lastCreatedAgentTaskId := ""                   // FIX #10: Cache agentTaskId from create_agent_task for instant validation

		// Tool call history: track all executed tools for smart filtering (reduces token usage by ~70%)
		toolCallHistory := make([]ToolResult, 0, 20)
		toolCallSignature := func(name string, args map[string]interface{}) string {
			argsJSON, _ := json.Marshal(args)
			return fmt.Sprintf("%s(%s)", name, string(argsJSON))
		}

		// WORKFLOW STATE ENFORCEMENT: Track coordinator workflow progress (fallback model only)
		workflowState := map[string]interface{}{
			"step":            0,     // 0=initial, 1=listed, 2=created, 3=searched, 4=agent_task, 5=done
			"humanTaskId":     "",    // Store taskId from step 2
			"searchCompleted": false, // Prevent multiple searches
			"agentTaskId":     "",    // Store agentTaskId from step 4
		}

		// Function to validate workflow tool calls (only enforced for fallback model)
		validateWorkflowTool := func(toolName string) (bool, string) {
			if !s.usingFallback {
				return true, "" // No enforcement for primary model
			}

			step := workflowState["step"].(int)
			humanTaskId := workflowState["humanTaskId"].(string)
			searchCompleted := workflowState["searchCompleted"].(bool)

			switch toolName {
			case "coordinator_list_human_tasks":
				if step == 0 || step == 1 || step == 2 {
					// Allow listing tasks at any early step to retrieve exact taskId
					return true, ""
				}
				return false, "❌ BLOCKED: You already have taskId. NEXT: Call appropriate tool for current step."

			case "coordinator_create_human_task":
				if step == 1 && humanTaskId == "" {
					return true, ""
				}
				if humanTaskId != "" {
					// State-aware guidance based on workflow progress
					if searchCompleted {
						return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'create_agent_task'.", humanTaskId)
					}
					return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'code_index_search'.", humanTaskId)
				}
				return false, "❌ BLOCKED: Call 'coordinator_list_human_tasks' first."

			case "code_index_search":
				if step == 2 && humanTaskId != "" && !searchCompleted {
					return true, ""
				}
				if searchCompleted {
					return false, "❌ BLOCKED: Search done. NEXT: Call 'create_agent_task'."
				}
				return false, "❌ BLOCKED: Create human task first."

			case "create_agent_task":
				if step == 3 && humanTaskId != "" && searchCompleted {
					return true, ""
				}
				return false, "❌ BLOCKED: Run 'code_index_search' first to get file paths."

			case "execute_subagent":
				agentTaskId := workflowState["agentTaskId"].(string)
				if step == 4 && agentTaskId != "" {
					return true, ""
				}
				return false, "❌ BLOCKED: Call 'create_agent_task' first."
			}

			return true, ""
		}

		// Per-tool circuit breaker thresholds (max duplicate attempts before stopping)
		// Claude models get higher thresholds as they're better at adapting
		isClaudeModel := strings.Contains(strings.ToLower(s.config.Model), "claude") ||
			strings.Contains(strings.ToLower(s.config.Provider), "anthropic")

		var circuitBreakerThresholds map[string]int
		if isClaudeModel {
			// Claude-optimized thresholds: More lenient to allow legitimate multi-file operations
			circuitBreakerThresholds = map[string]int{
				"read_file":         5, // Allow reading multiple files
				"write_file":        2, // Allow one retry for writes
				"list_directory":    4, // Allow exploring directories
				"bash":              5, // Allow command variations
				"code_index_search": 2, // Strict: one search + one retry max
				"create_agent_task": 4, // Allow retries for parameter refinement
				// Default for other tools: 6 attempts
			}
			log.Printf("[Circuit Breaker] Using Claude-optimized thresholds (more lenient)")
		} else {
			// GPT thresholds: More conservative
			circuitBreakerThresholds = map[string]int{
				"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
				"write_file":        1, // Never allow duplicate writes
				"list_directory":    2, // Stop after 2 attempts
				"bash":              3, // Allow more for command variations
				"code_index_search": 3, // Allow query refinement
				// Default for other tools: 4 attempts
			}
			log.Printf("[Circuit Breaker] Using GPT thresholds (conservative)")
		}

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			iterationCount++
			debugLog("---------- ITERATION %d START ----------", iterationCount)
			debugLog("Current tool call count: %d/%d", toolCallCount, maxToolCalls)

			// CRITICAL: Reload full tools array at start of each iteration
			// This prevents the filtered tools from previous iteration being reused
			tools = s.toolRegistry.GetToolsForLangChain()
			debugLog("Reloaded tools: %d", len(tools))

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// Apply sliding window BEFORE context exceeds model's token limit
			// Claude: 200K tokens (≈800KB text) - use 150KB threshold
			// GPT: 32K tokens (≈128KB text) - use 40KB threshold to be safe
			var maxContextSize int
			var maxMessages int
			if isClaudeModel {
				maxContextSize = 150000 // 150KB for Claude (≈37K tokens, leaves room for output)
				maxMessages = 20        // Keep more messages for Claude
				log.Printf("[Context Window] Using Claude limits: %d chars, max %d messages", maxContextSize, maxMessages)
			} else {
				maxContextSize = 40000 // 40KB for GPT (≈10K tokens)
				maxMessages = 6        // Conservative for GPT
				log.Printf("[Context Window] Using GPT limits: %d chars, max %d messages", maxContextSize, maxMessages)
			}

			if contextSize > maxContextSize {
				log.Printf("[Sliding Window] Context size %d chars exceeds threshold %d chars, applying window",
					contextSize, maxContextSize)
				currentMessages = applySlidingWindow(currentMessages, maxMessages)

				// Recalculate after trimming
				contextSize = 0
				for _, msg := range currentMessages {
					contextSize += len(msg.Content)
				}
			}

			// Log iteration details
			log.Printf("[AI Processing] Iteration: %d, Request: %d chars, Context: %d chars, Tool calls so far: %d",
				iterationCount, contextSize, contextSize, toolCallCount)

			// DEBUG: Log context details before LLM API call to identify accumulation
			contextSize = calculateContextSize(currentMessages)
			toolResultPreview := getToolResultPreview(currentMessages, 200)
			log.Printf("[DEBUG Context] Before LLM call - Messages: %d, Total size: %d chars, Tool result preview: %s",
				len(currentMessages), contextSize, toolResultPreview)

			// SMART TOOL FILTERING: Reduce token usage by 70% by sending only relevant tools
			// This applies to ALL models to reduce rate limit issues
			originalToolCount := len(tools)
			relevantToolNames := filterToolsByWorkflowState(toolCallHistory)
			filteredTools := s.toolRegistry.GetFilteredToolsForLangChain(relevantToolNames)

			// Only apply smart filtering if it actually reduces the tool count
			// Keep all tools if filtering would include most of them anyway (>30 tools)
			if len(filteredTools) < originalToolCount && len(filteredTools) <= 30 {
				tools = filteredTools
				log.Printf("[Smart Tool Filter] Reduced from %d to %d tools (%.0f%% reduction) - Tool history: %d calls",
					originalToolCount, len(tools), 100.0*(1.0-float64(len(tools))/float64(originalToolCount)), len(toolCallHistory))

				// Log which tools are being sent for debugging
				toolNames := make([]string, 0, len(tools))
				for _, tool := range tools {
					if tool.Function != nil {
						toolNames = append(toolNames, tool.Function.Name)
					}
				}
				log.Printf("[Smart Tool Filter] Sending tools: %v", toolNames)
			} else {
				log.Printf("[Smart Tool Filter] Keeping all %d tools (filtering not beneficial)", originalToolCount)
			}

			// PHASE 3: PRESCRIPTIVE STATE MACHINE - Only allow ONE tool per workflow step
			// This forces ALL models into a linear workflow with zero ambiguity
			// Each step unlocks exactly ONE required tool - model has no choice but to follow the sequence
			// Applied to ALL models (not just Claude) to ensure consistent coordinator workflow
			if false { // DISABLED: Workflow enforcement (was blocking direct tool execution)
				step := workflowState["step"].(int)
				originalCount := len(tools)
				debugLog("PRESCRIPTIVE FILTER: Current workflow step = %d, tools before filter = %d", step, originalCount)

				// Define allowed tools per step (WHITELIST approach)
				var allowedTools []string
				switch step {
				case 0: // Step 0: ONLY allow coordinator_list_human_tasks
					allowedTools = []string{"coordinator_list_human_tasks"}
				case 1: // Step 1: ONLY allow coordinator_create_human_task
					allowedTools = []string{"coordinator_create_human_task"}
				case 2: // Step 2: ONLY allow code_index_search
					allowedTools = []string{"code_index_search"}
				case 3: // Step 3: ONLY allow create_agent_task
					allowedTools = []string{"create_agent_task"}
				case 4: // Step 4: ONLY allow execute_subagent
					allowedTools = []string{"execute_subagent"}
				case 5: // Step 5: Workflow complete - NO TOOLS NEEDED (subagent is executing)
					// DO NOT provide list_agent_tasks - it causes hallucinated humanTaskIds
					// The subagent is executing in background, coordinator should inform user and STOP
					allowedTools = []string{}
				default:
					allowedTools = nil // Unknown step, allow all
				}

				// Filter tools using whitelist
				if allowedTools != nil {
					// Create set for O(1) lookup
					allowedSet := make(map[string]bool)
					for _, name := range allowedTools {
						allowedSet[name] = true
					}

					filteredTools := make([]llms.Tool, 0, len(allowedTools))
					for _, tool := range tools {
						if tool.Function != nil && allowedSet[tool.Function.Name] {
							filteredTools = append(filteredTools, tool)
						}
					}
					tools = filteredTools
				}

				debugLog("PRESCRIPTIVE FILTER: Allowed tools for step %d = %v", step, allowedTools)
				debugLog("PRESCRIPTIVE FILTER: After filter = %d tools", len(tools))
				if originalCount != len(tools) {
					log.Printf("[Phase 3 Prescriptive Filter] Step %d: Filtered %d → %d tools (allowed: %v)",
						step, originalCount, len(tools), allowedTools)
				}
			}

			// DEBUG: Log tools being passed to LLM
			log.Printf("[DEBUG Tools] Passing %d tools to LLM provider %s", len(tools), s.config.Provider)
			if len(tools) > 0 {
				toolNames := make([]string, 0, 3)
				for i := 0; i < len(tools) && i < 3; i++ {
					if tools[i].Function != nil {
						toolNames = append(toolNames, tools[i].Function.Name)
					}
				}
				log.Printf("[DEBUG Tools] Sample tools: %v", toolNames)
			}

			// WORKFLOW STATE GUIDANCE: Inject iteration and tool call progress into system prompt
			// This helps the LLM understand where it is in the workflow and avoid loops
			if len(currentMessages) > 0 && currentMessages[0].Role == "system" {
				// Build workflow state summary
				workflowGuidance := fmt.Sprintf(`

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WORKFLOW PROGRESS (Iteration %d)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Current Iteration: %d / %d
Total Tool Calls Made: %d / %d

⚠️  IMPORTANT: If you see "BLOCKED" or "NEXT:" in recent tool results,
    those messages contain CRITICAL guidance about what to do next.
    You MUST follow the "NEXT:" instructions - they tell you exactly which tool to call.

⚠️  AVOID LOOPS: Do NOT retry the same tool that was just BLOCKED.
    Follow the "NEXT:" guidance instead.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, iterationCount, iterationCount, s.config.MaxIterations, toolCallCount, maxToolCalls)

				// INJECT CAPTURED HUMAN TASK ID: If we captured a humanTaskId from coordinator_create_human_task,
				// inject it directly into the system prompt so Claude can use it without extraction
				if humanTaskID, ok := workflowState["humanTaskId"].(string); ok && humanTaskID != "" {
					taskIDGuidance := fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎯 CAPTURED TASK ID
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

The human task has been created/found. Use this EXACT taskId value:

**humanTaskId**: "%s"

When calling create_agent_task, use EXACTLY:
{
  "humanTaskId": "%s",
  ...
}

DO NOT generate or make up a different task ID. Use the value shown above.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, humanTaskID, humanTaskID)
					workflowGuidance += taskIDGuidance
					log.Printf("[Workflow State] Injected humanTaskId into system prompt: %s", humanTaskID)
				}

				// Append to existing system prompt (don't replace it)
				currentMessages[0].Content = currentMessages[0].Content + workflowGuidance
				log.Printf("[Workflow Guidance] Injected iteration %d progress into system prompt", iterationCount)
			}

			// Call provider with tools
			toolProvider := s.provider.(ToolCapableProvider)
			response, err := toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
			if err != nil {
				// Check if this is a rate limit error and we have a fallback model configured
				if isRateLimitError(err) && s.config.FallbackModel != "" {
					log.Printf("[Rate Limit] Detected rate limit error, switching to fallback model: %s → %s",
						s.config.Model, s.config.FallbackModel)

					// Notify user about fallback
					fallbackMsg := fmt.Sprintf("\n\n⚠️  RATE LIMIT DETECTED: Primary model '%s' has hit its rate limit.\n"+
						"🔄 Automatically switching to fallback model '%s' (local, no rate limits)...\n\n",
						s.config.Model, s.config.FallbackModel)
					eventChan <- StreamEvent{Type: StreamEventToken, Content: fallbackMsg}

					// Save original model, provider, and API key
					originalModel := s.config.Model
					originalProvider := s.config.Provider
					originalAPIKey := s.config.APIKey

					// Switch to fallback model
					s.config.Model = s.config.FallbackModel

					// Switch to Anthropic provider for Claude models
					if strings.Contains(strings.ToLower(s.config.FallbackModel), "claude") {
						s.config.Provider = "anthropic"
						// Load Anthropic API key from environment
						anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
						if anthropicKey == "" {
							log.Printf("[ChatService] ERROR - RequestID: %s - ANTHROPIC_API_KEY not found in environment", requestID)
							eventChan <- StreamEvent{Type: StreamEventError,
								Error: "Rate limit error and ANTHROPIC_API_KEY not configured for fallback"}
							return
						}
						s.config.APIKey = anthropicKey
					}

					// Recreate provider with fallback model
					fallbackProvider, err := NewChatProvider(s.config)
					if err != nil {
						log.Printf("[ChatService] ERROR - RequestID: %s - Failed to create fallback provider: %v", requestID, err)
						eventChan <- StreamEvent{Type: StreamEventError,
							Error: fmt.Sprintf("Rate limit error and failed to switch to fallback model: %v", err)}
						return
					}

					// Update provider and mark as using fallback
					s.provider = fallbackProvider
					s.usingFallback = true

					// CRITICAL FIX: If we fell back to Claude, swap to Claude-optimized system prompt and circuit breakers
					if strings.Contains(strings.ToLower(s.config.FallbackModel), "claude") {
						log.Printf("[Rate Limit] Detected Claude fallback - swapping to Claude-optimized system prompt and thresholds")

						// Replace the system prompt in currentMessages with Claude-optimized version
						for i := range currentMessages {
							if currentMessages[i].Role == "system" {
								// Swap out FILE_PATHS_TO_USE fiction for real JSON examples
								claudePrompt := getClaudeSystemPrompt()

								// Extract session context from old prompt (the critical guidance section)
								oldPrompt := currentMessages[i].Content
								sessionContextStart := strings.Index(oldPrompt, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\nCRITICAL SYSTEM BEHAVIOR")
								if sessionContextStart > 0 {
									sessionContext := oldPrompt[sessionContextStart:]
									currentMessages[i].Content = claudePrompt + "\n\n" + sessionContext
								} else {
									currentMessages[i].Content = claudePrompt
								}

								log.Printf("[Rate Limit] Swapped system prompt: %d chars → %d chars", len(oldPrompt), len(currentMessages[i].Content))
								break
							}
						}

						// Re-calculate circuit breaker thresholds for Claude (more lenient)
						circuitBreakerThresholds = map[string]int{
							"read_file":         5, // Allow reading multiple files
							"write_file":        2, // Allow one retry for writes
							"list_directory":    4, // Allow exploring directories
							"bash":              5, // Allow command variations
							"code_index_search": 2, // Strict: one search + one retry max
							"create_agent_task": 4, // Allow retries for parameter refinement
							// Default for other tools: 6 attempts
						}
						log.Printf("[Circuit Breaker] Re-applied Claude thresholds after fallback")
					}

					// Retry with fallback provider
					if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
						response, err = toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
						if err != nil {
							log.Printf("[ChatService] ERROR - RequestID: %s - Fallback model also failed: %v", requestID, err)
							eventChan <- StreamEvent{Type: StreamEventError,
								Error: fmt.Sprintf("Both primary and fallback models failed: %v", err)}
							return
						}
						log.Printf("[Rate Limit] Successfully switched to fallback model '%s'", s.config.FallbackModel)

						// Send success notification
						successMsg := fmt.Sprintf("✅ Successfully switched to '%s'. Will automatically retry primary model on next request...\n\n", s.config.FallbackModel)
						eventChan <- StreamEvent{Type: StreamEventToken, Content: successMsg}

						// Note: We will automatically try to switch back to the primary model on the next request
						// The original configuration is saved in s.originalModel, s.originalProvider, s.originalAPIKey
						_, _, _ = originalModel, originalProvider, originalAPIKey // Mark as used to avoid compiler warning
					} else {
						log.Printf("[ChatService] ERROR - Fallback provider doesn't support tools")
						eventChan <- StreamEvent{Type: StreamEventError, Error: "Fallback provider doesn't support tools"}
						return
					}
				} else {
					// Not a rate limit error or no fallback configured - just fail
					log.Printf("[ChatService] ERROR - RequestID: %s - Tool call failed: %v", requestID, err)
					eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
					return
				}
			}

			// Collect response tokens (but don't stream yet)
			var responseText string
			responseTokens := 0
			var collectedChunks []string
			for chunk := range response.TextChannel {
				collectedChunks = append(collectedChunks, chunk)
				responseText += chunk
				responseTokens++
			}

			debugLog("AI RESPONSE: iteration=%d, tokens=%d, toolCalls=%d", iterationCount, responseTokens, len(response.ToolCalls))

			// Log iteration response details
			log.Printf("[AI Processing] Iteration: %d complete, Response: %d tokens, Tool calls requested: %d",
				iterationCount, responseTokens, len(response.ToolCalls))

			// DEBUG: Check if responseText contains tool JSON
			if strings.Contains(responseText, `[{"id":"`) {
				log.Printf("[DEBUG TOOL_EXECUTOR] ⚠️  WARNING: responseText contains tool JSON!")
				preview := responseText
				if len(preview) > 300 {
					preview = preview[:300] + "..."
				}
				log.Printf("[DEBUG TOOL_EXECUTOR] responseText preview: %s", preview)
			} else {
				log.Printf("[DEBUG TOOL_EXECUTOR] ✓ responseText is clean (no tool JSON)")
				preview := responseText
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				log.Printf("[DEBUG TOOL_EXECUTOR] responseText preview: %s", preview)
			}

			// Check stop reason to determine if turn has ended
			// According to Claude API docs:
			// - "end_turn": model finished naturally, no more tool calls
			// - "tool_use": model is requesting tool execution
			// - "max_tokens", "stop_sequence", etc.: other stop conditions
			if response.StopReason == "end_turn" || len(response.ToolCalls) == 0 {
				debugLog("EXIT: StopReason=%s, ToolCalls=%d - streaming final response (%d chunks)",
					response.StopReason, len(response.ToolCalls), len(collectedChunks))
				// Turn ended - stream the collected text (final response)
				for _, chunk := range collectedChunks {
					eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				}
				log.Printf("[ChatService] Stream complete - RequestID: %s - StopReason: %s - Total iterations: %d, Tool calls: %d",
					requestID, response.StopReason, iterationCount, toolCallCount)
				debugLog("========== GOROUTINE ENDED (stop_reason=%s) ==========", response.StopReason)
				return
			}

			// Tool calls pending - suppress text output
			log.Printf("[ChatService] Suppressing %d tokens of text - tool calls pending: %d",
				len(collectedChunks), len(response.ToolCalls))

			// Process each tool call
			for _, toolCall := range response.ToolCalls {
				toolCallCount++
				debugLog("TOOL CALL #%d: name=%s", toolCallCount, toolCall.Name)
				if toolCallCount > maxToolCalls {
					log.Printf("[ChatService] Max tool calls reached (%d) - RequestID: %s", maxToolCalls, requestID)
					eventChan <- StreamEvent{Type: StreamEventError, Error: fmt.Sprintf("maximum tool calls (%d) exceeded", maxToolCalls)}
					debugLog("========== GOROUTINE ENDED (max tools reached) ==========")
					return
				}

				// Log tool request with arguments
				argsJSON, _ := json.Marshal(toolCall.Args)
				log.Printf("[Tool Request] AI requested tool '%s' with args: %s",
					toolCall.Name, string(argsJSON))
				debugLog("TOOL CALL: name=%s, args=%s", toolCall.Name, string(argsJSON))

				// WORKFLOW VALIDATION: Check if this tool call is allowed in current workflow state
				var result ToolResult
				if s.usingFallback {
					allowed, blockMessage := validateWorkflowTool(toolCall.Name)
					if !allowed {
						log.Printf("[Workflow Enforcer] BLOCKED tool '%s' - %s", toolCall.Name, blockMessage)

						// Create a blocking error result so model understands the tool failed
						result = ToolResult{
							ID:         toolCall.ID,
							Name:       toolCall.Name,
							Output:     nil,
							Error:      blockMessage,
							DurationMs: 0,
						}

						// Send the error result to the model
						eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

						// Add tool result to message history as an error
						currentMessages = append(currentMessages, Message{
							Role:    "assistant",
							Content: fmt.Sprintf("I attempted to call '%s' but it was blocked.", toolCall.Name),
						})
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: fmt.Sprintf("TOOL ERROR from '%s': %s", toolCall.Name, blockMessage),
						})

						// Continue to next iteration - don't execute the blocked tool
						continue
					}
				}

				// Send tool call event
				eventChan <- StreamEvent{Type: StreamEventToolCall, ToolCall: &toolCall}

				// Generate signature for cache and circuit breaker
				signature := toolCallSignature(toolCall.Name, toolCall.Args)

				// Check tool result cache BEFORE execution
				cachedResult, found := resultCache.Get(signature)

				// SPECIAL: Skip cache for coordinator_create_human_task when similar tasks found
				// This allows loop prevention logic to run and auto-create on 3rd attempt
				skipCache := false
				if found && toolCall.Name == "coordinator_create_human_task" {
					if outputMap, ok := cachedResult.Output.(map[string]interface{}); ok {
						if similarTasksFound, exists := outputMap["similarTasksFound"].(bool); exists && similarTasksFound {
							skipCache = true
							log.Printf("[Tool Cache SKIP] Skipping cache for '%s' - similar tasks detected, allowing loop prevention to run", toolCall.Name)
						}
					}
				}

				if found && !skipCache {
					// Use cached result - avoid redundant execution
					result = *cachedResult
					log.Printf("[Tool Cache HIT] Using cached result for '%s' - skipping execution", toolCall.Name)

					// Add cache hit notice to the result so AI knows it's cached
					cacheNotice := fmt.Sprintf("🔁 CACHED RESULT: You already called '%s' with these exact arguments. Using previous result instead of re-executing.", toolCall.Name)

					// Prepend cache notice to the output
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						// Clone the map to avoid mutating the cached version
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_cacheNotice"] = cacheNotice
						result.Output = newOutput
					}
				} else {
					// Execute tool (no cached result available)
					// Inject humanTaskId from workflowState into context for auto-population
					toolCtx := ctx
					if humanTaskID, ok := workflowState["humanTaskId"].(string); ok && humanTaskID != "" {
						toolCtx = context.WithValue(ctx, "lastHumanTaskId", humanTaskID)
					}
					result = s.toolRegistry.ExecuteToolCall(toolCtx, toolCall)
					log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// GPT FILE PATH VALIDATION: For code_index_search, validate file paths before proceeding
				// This ONLY applies to GPT models, NOT Claude (Claude has its own file validation in create_agent_task)
				if toolCall.Name == "code_index_search" && result.Error == "" && !s.usingFallback {
					isGPTModel := !strings.Contains(strings.ToLower(s.config.Model), "claude")

					if isGPTModel {
						// Extract file paths from search results
						filePaths := extractFilePathsFromCodeIndexResult(result)

						if len(filePaths) > 0 {
							validPaths, invalidPaths := validateFilePaths(filePaths)

							log.Printf("[GPT Path Validator] code_index_search returned %d paths: %d valid, %d invalid",
								len(filePaths), len(validPaths), len(invalidPaths))

							if len(invalidPaths) > 0 {
								// Some paths are invalid - check if this is already a retry
								retryAttemptKey := fmt.Sprintf("code_index_search_retry_%v", toolCall.Args["query"])
								hasRetried := pathValidationRetries[retryAttemptKey]

								if hasRetried {
									// Already retried once - stop execution with clear error
									log.Printf("[GPT Path Validator] code_index_search retry also returned invalid paths - stopping execution")

									errorMsg := fmt.Sprintf("❌ CRITICAL ERROR: code_index_search returned invalid file paths (files don't exist on filesystem):\n\n"+
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

									eventChan <- StreamEvent{Type: StreamEventError, Error: errorMsg}
									return
								} else {
									// First time encountering invalid paths - try automatic retry with refined query
									log.Printf("[GPT Path Validator] First invalid path detection - attempting automatic retry with refined query")

									// Mark this as a retry attempt
									pathValidationRetries[retryAttemptKey] = true

									// Send warning to user about automatic retry
									warningMsg := fmt.Sprintf("\n\n⚠️  FILE PATH VALIDATION WARNING:\n"+
										"code_index_search returned %d invalid file paths (files don't exist).\n"+
										"🔄 Automatically retrying search with refined query...\n\n"+
										"Invalid paths found:\n%s\n\n",
										len(invalidPaths), strings.Join(invalidPaths, "\n"))
									eventChan <- StreamEvent{Type: StreamEventToken, Content: warningMsg}

									// Modify the tool result to indicate validation failure
									// Inject validation warning into result so GPT knows to refine the search
									if outputMap, ok := result.Output.(map[string]interface{}); ok {
										outputMap["_validationWarning"] = fmt.Sprintf(
											"⚠️ VALIDATION FAILED: %d out of %d file paths are INVALID (files don't exist on filesystem). "+
												"You MUST retry code_index_search with a different query to find the CORRECT file paths. "+
												"DO NOT proceed with create_agent_task using these invalid paths. "+
												"Invalid paths: %s",
											len(invalidPaths), len(filePaths), strings.Join(invalidPaths, "\n"))
										outputMap["invalidPaths"] = invalidPaths
										outputMap["validPaths"] = validPaths
										result.Output = outputMap
									}

									log.Printf("[GPT Path Validator] Injected validation warning into result - GPT should retry search")
								}
							} else {
								log.Printf("[GPT Path Validator] All paths valid - proceeding normally")
							}
						}
					}
				}

				// CACHE INVALIDATION: Clear related caches after successful write operations
				// CRITICAL: This must run BEFORE TaskId Validator to prevent stale cache reads
				// Bug fix: Previously ran after validation, causing validator to see stale cached data
				if result.Error == "" {
					switch toolCall.Name {
					case "coordinator_create_human_task":
						// Clear all list_human_tasks cache entries (any filter parameters)
						count := resultCache.DeletePrefix("coordinator_list_human_tasks:")
						log.Printf("[Cache Invalidation] coordinator_create_human_task: cleared %d coordinator_list_human_tasks cache entries", count)

					case "coordinator_create_agent_task":
						// Clear all list_agent_tasks cache entries (any filter parameters)
						count := resultCache.DeletePrefix("coordinator_list_agent_tasks:")
						log.Printf("[Cache Invalidation] coordinator_create_agent_task: cleared %d coordinator_list_agent_tasks cache entries", count)

					case "coordinator_update_task_status":
						// Clear human tasks, agent tasks, and specific task get caches
						count1 := resultCache.DeletePrefix("coordinator_list_human_tasks:")
						count2 := resultCache.DeletePrefix("coordinator_list_agent_tasks:")
						count3 := resultCache.DeletePrefix("coordinator_get_agent_task:")
						log.Printf("[Cache Invalidation] coordinator_update_task_status: cleared %d human_tasks + %d agent_tasks + %d get_task cache entries", count1, count2, count3)

					case "coordinator_update_todo_status":
						// Clear agent tasks and specific task get caches
						count1 := resultCache.DeletePrefix("coordinator_list_agent_tasks:")
						count2 := resultCache.DeletePrefix("coordinator_get_agent_task:")
						log.Printf("[Cache Invalidation] coordinator_update_todo_status: cleared %d agent_tasks + %d get_task cache entries", count1, count2)

					case "coordinator_add_task_prompt_notes", "coordinator_update_task_prompt_notes", "coordinator_clear_task_prompt_notes":
						// Clear specific task get cache
						count := resultCache.DeletePrefix("coordinator_get_agent_task:")
						log.Printf("[Cache Invalidation] %s: cleared %d get_task cache entries", toolCall.Name, count)

					case "coordinator_add_todo_prompt_notes", "coordinator_update_todo_prompt_notes", "coordinator_clear_todo_prompt_notes":
						// Clear specific task get cache
						count := resultCache.DeletePrefix("coordinator_get_agent_task:")
						log.Printf("[Cache Invalidation] %s: cleared %d get_task cache entries", toolCall.Name, count)

					case "coordinator_upsert_knowledge":
						// Clear knowledge query cache
						count := resultCache.DeletePrefix("coordinator_query_knowledge:")
						log.Printf("[Cache Invalidation] coordinator_upsert_knowledge: cleared %d knowledge query cache entries", count)

					case "coordinator_clear_task_board":
						// Clear ALL coordinator caches (nuclear option)
						count1 := resultCache.DeletePrefix("coordinator_list_human_tasks:")
						count2 := resultCache.DeletePrefix("coordinator_list_agent_tasks:")
						count3 := resultCache.DeletePrefix("coordinator_get_agent_task:")
						count4 := resultCache.DeletePrefix("coordinator_query_knowledge:")
						log.Printf("[Cache Invalidation] coordinator_clear_task_board: cleared %d human_tasks + %d agent_tasks + %d get_task + %d knowledge cache entries",
							count1, count2, count3, count4)
					}
				}

				// TASK ID VALIDATION: For create_agent_task, validate humanTaskId exists before proceeding
				if toolCall.Name == "create_agent_task" && result.Error == "" {
					// Extract humanTaskId from arguments
					var humanTaskId string
					if id, ok := toolCall.Args["humanTaskId"].(string); ok {
						humanTaskId = id
					}

					// Validate humanTaskId if provided
					if humanTaskId != "" {
						// FIX #9: Check cached taskId FIRST (instant validation, no database needed)
						// This eliminates race conditions when model uses taskId immediately after creation
						taskExists := false
						if humanTaskId == lastCreatedHumanTaskId && lastCreatedHumanTaskId != "" {
							taskExists = true
							log.Printf("[TaskId Validator] ✅ Instant validation: humanTaskId '%s' matches last created task (no DB lookup needed)", humanTaskId)
						} else {
							// Not the cached ID - do database validation with retry
							// FIX #8: Retry validation to handle MongoDB eventual consistency
							for attempt := 0; attempt < 3; attempt++ {
								// Call coordinator_list_human_tasks to get all tasks
								listTasksCall := ToolCall{
									ID:   "taskid_validation",
									Name: "coordinator_list_human_tasks",
									Args: map[string]interface{}{},
								}
								listResult := s.toolRegistry.ExecuteToolCall(ctx, listTasksCall)

								if listResult.Error == "" {
									if outputMap, ok := listResult.Output.(map[string]interface{}); ok {
										if tasks, ok := outputMap["tasks"].([]interface{}); ok {
											for _, task := range tasks {
												if taskMap, ok := task.(map[string]interface{}); ok {
													// Check taskId field (matching HumanTask JSON schema with json:"taskId" tag)
													if taskId, ok := taskMap["taskId"].(string); ok && taskId == humanTaskId {
														taskExists = true
														break
													}
												}
											}
										}
									}
								}

								if taskExists {
									break // Found it, stop retrying
								}

								// Not found yet - retry with exponential backoff
								if attempt < 2 {
									sleepDuration := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
									log.Printf("[TaskId Validator] Task '%s' not found yet - retrying after %v (attempt %d/3)",
										humanTaskId, sleepDuration, attempt+1)
									time.Sleep(sleepDuration)
								}
							}
						}

						if !taskExists {
							// TaskId is invalid even after retries - increment attempt counter
							taskIdValidationAttempts++
							log.Printf("[TaskId Validator] Invalid humanTaskId '%s' after 3 retries with backoff - Attempt %d/3", humanTaskId, taskIdValidationAttempts)

							if taskIdValidationAttempts >= 3 {
								// After 3 attempts, stop execution with clear error
								log.Printf("[TaskId Validator] Failed 3 times - stopping execution")
								errorMsg := fmt.Sprintf("❌ CRITICAL ERROR: create_agent_task called with INVALID humanTaskId 3 times.\n\n"+
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

								eventChan <- StreamEvent{Type: StreamEventError, Error: errorMsg}
								return
							}

							// First or second attempt - inject warning and ask model to list tasks
							warningMsg := fmt.Sprintf("\n\n⚠️  TASK ID VALIDATION ERROR (Attempt %d/3):\n"+
								"The humanTaskId '%s' DOES NOT EXIST in the database.\n"+
								"🔄 You MUST call coordinator_list_human_tasks to get the correct taskId.\n\n",
								taskIdValidationAttempts, humanTaskId)
							eventChan <- StreamEvent{Type: StreamEventToken, Content: warningMsg}

							// Replace the result with an error result
							result = ToolResult{
								ID:   result.ID,
								Name: "create_agent_task",
								Args: result.Args,
								Output: map[string]interface{}{
									"_validationError": "BLOCKED",
									"_reason":          fmt.Sprintf("Invalid humanTaskId: '%s' does not exist", humanTaskId),
									"NEXT":             "Call coordinator_list_human_tasks to see all tasks, find the correct task, and COPY its exact 'taskId' field",
								},
								Error: fmt.Sprintf("❌ BLOCKED: humanTaskId '%s' is INVALID (does not exist). "+
									"You MUST call coordinator_list_human_tasks to get all tasks and find the EXACT taskId. "+
									"DO NOT hallucinate or generate task IDs. COPY the exact UUID from the tool response. "+
									"This is attempt %d/3. After 3 failures, execution will stop.",
									humanTaskId, taskIdValidationAttempts),
							}

							log.Printf("[TaskId Validator] Blocked create_agent_task - injected error asking model to list tasks")
						} else {
							log.Printf("[TaskId Validator] ✅ humanTaskId '%s' validated successfully (may have required retries)", humanTaskId)
						}
					}
				}

				// FIX #10: AGENT TASK ID AUTO-CORRECTION for execute_subagent
				// Automatically replace hallucinated/wrong agentTaskId with the cached correct one
				if toolCall.Name == "execute_subagent" {
					var providedAgentTaskId string
					if id, ok := toolCall.Args["agentTaskId"].(string); ok {
						providedAgentTaskId = id
					}

					// If we have a cached agentTaskId from create_agent_task, use it
					if lastCreatedAgentTaskId != "" {
						// Check if model provided wrong ID (hallucinated)
						if providedAgentTaskId != lastCreatedAgentTaskId {
							log.Printf("[AgentTaskId Auto-Correct] Model provided wrong agentTaskId: '%s', replacing with correct cached ID: '%s'",
								providedAgentTaskId, lastCreatedAgentTaskId)

							// REPLACE the wrong ID with the correct cached one
							toolCall.Args["agentTaskId"] = lastCreatedAgentTaskId

							// Re-execute the tool with corrected arguments
							result = s.toolRegistry.ExecuteToolCall(ctx, toolCall)

							log.Printf("[AgentTaskId Auto-Correct] ✅ Re-executed execute_subagent with correct agentTaskId: '%s'", lastCreatedAgentTaskId)
						} else {
							log.Printf("[AgentTaskId Validator] ✅ Instant validation: agentTaskId '%s' matches last created task", providedAgentTaskId)
						}
					} else if providedAgentTaskId != "" {
						// No cached ID - let the tool validate via GetAgentTask (with retry)
						log.Printf("[AgentTaskId Validator] No cached agentTaskId, will validate '%s' via GetAgentTask", providedAgentTaskId)
					}
				}

				// Send tool result event (full result to client for display)
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// Track tool execution in history for smart filtering (keep last 20)
				toolCallHistory = append(toolCallHistory, result)
				if len(toolCallHistory) > 20 {
					toolCallHistory = toolCallHistory[1:] // Remove oldest
				}

				// CRITICAL: Track CONSECUTIVE failed tool calls - stop on 3+ consecutive identical failures
				// This allows: compile → error → fix → compile (normal flow)
				// But prevents: compile → error → compile → error → compile (infinite loop)
				if result.Error != "" {
					// Check if this is the same as the last failed call
					if lastFailedSignature == signature {
						consecutiveFailures++
						if consecutiveFailures >= 3 {
							// Third CONSECUTIVE failure with same args - stop!
							log.Printf("[Circuit Breaker] Tool '%s' failed 3 times CONSECUTIVELY with identical arguments - stopping", toolCall.Name)
							// Return error to AI, don't stop execution
							// The AI should see this error and try a different approach
							loopWarning := fmt.Sprintf("❌ CRITICAL: Tool '%s' has FAILED 3 TIMES IN A ROW with identical arguments.\n\n"+
								"Error: %s\n\n"+
								"🛑 This approach is NOT working. You MUST try something different:\n"+
								"   - If file not found: List the directory first to see what files actually exist\n"+
								"   - If path wrong: Try a different path or check your working directory\n"+
								"   - If tool incompatible: Use a completely different tool or approach\n\n"+
								"DO NOT call this tool with these arguments again!", toolCall.Name, result.Error)

							// Add warning to current messages so AI sees it
							currentMessages = append(currentMessages, Message{
								Role:    "system",
								Content: loopWarning,
							})

							// Reset counters
							consecutiveFailures = 0
							lastFailedSignature = ""
						} else {
							lastFailedSignature = signature
						}
					} else {
						// Different failure, reset counter
						consecutiveFailures = 1
						lastFailedSignature = signature
					}
				} else {
					// Success - reset failure tracking
					consecutiveFailures = 0
					lastFailedSignature = ""
				}

				// Circuit breaker: check for repeated tool calls AND warn the AI
				recentToolCalls = append(recentToolCalls, signature)
				if len(recentToolCalls) > 10 {
					recentToolCalls = recentToolCalls[1:]
				}

				// Count how many times this exact tool+args was called in ALL history
				totalCount := 0
				for _, sig := range recentToolCalls {
					if sig == signature {
						totalCount++
					}
				}

				// Get tool-specific threshold
				threshold := circuitBreakerThresholds[toolCall.Name]
				if threshold == 0 {
					threshold = 4 // Default threshold
				}

				// Progressive warnings to AI (inject into context so AI sees them)
				var loopWarning string
				if totalCount == 2 {
					// First duplicate - gentle warning
					loopWarning = fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. You should use the result from the previous call instead of repeating the same operation.", toolCall.Name)
				} else if totalCount == 3 && threshold > 3 {
					// Second duplicate - stronger warning (only if threshold allows)
					loopWarning = fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach - do NOT call this tool again with the same arguments.", toolCall.Name)
				} else if totalCount >= threshold {
					// Threshold reached - trigger circuit breaker
					log.Printf("[Circuit Breaker] Tool '%s' called %d times (threshold: %d) - stopping infinite loop", toolCall.Name, totalCount, threshold)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop and cannot complete this task.", toolCall.Name, totalCount),
					}
					return
				}

				// Log tool execution with comprehensive response data (Claude optimization)
				if result.Error != "" {
					log.Printf("[ChatService] Tool '%s' failed - RequestID: %s - Error: %s - Duration: %dms",
						result.Name, requestID, result.Error, result.DurationMs)
					// Log complete error response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					log.Printf("[Tool Response - ERROR] Tool: %s | Args: %s | Error: %s",
						result.Name, string(argsJSON), result.Error)
				} else {
					log.Printf("[ChatService] Tool '%s' succeeded - RequestID: %s - Duration: %dms",
						result.Name, requestID, result.DurationMs)
					// Log complete success response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					outputJSON, _ := json.Marshal(result.Output)
					log.Printf("[Tool Response - SUCCESS] Tool: %s | Args: %s | Output: %s",
						result.Name, string(argsJSON), string(outputJSON))
				}

				// Add assistant response to history (brief)
				currentMessages = append(currentMessages, Message{
					Role:    "assistant",
					Content: responseText,
				})

				// Add tool result to message history
				var toolResultMsg string
				if result.Error != "" {
					// Check if this is a permanent failure that shouldn't be retried
					errorLower := strings.ToLower(result.Error)
					isPermanentError := strings.Contains(errorLower, "requires mcp endpoint") ||
						strings.Contains(errorLower, "not supported") ||
						strings.Contains(errorLower, "cannot be used") ||
						strings.Contains(errorLower, "requires direct mcp")

					if isPermanentError {
						toolResultMsg = fmt.Sprintf("PERMANENT ERROR - Tool '%s' cannot be used in this context: %s. DO NOT retry this tool - it will not work.", result.Name, result.Error)
					} else {
						// ENHANCED ERROR GUIDANCE: Provide specific recovery instructions
						recoveryGuidance := getErrorRecoveryGuidance(result.Name, result.Error, toolCall.Args)
						toolResultMsg = fmt.Sprintf("❌ ERROR in tool '%s': %s\n\n%s", result.Name, result.Error, recoveryGuidance)

						// Also send error as visible message to user
						eventChan <- StreamEvent{
							Type:    StreamEventToken,
							Content: fmt.Sprintf("\n\n⚠️  Tool Error: %s\n💡 %s\n\n", result.Error, recoveryGuidance),
						}
					}
				} else {
					// Marshal output to JSON for context
					outputJSON, err := json.Marshal(result.Output)
					if err != nil {
						toolResultMsg = fmt.Sprintf("Tool '%s' result: <serialization error: %v>", result.Name, err)
					} else {
						// Check if output contains an error field (common pattern for tools returning error in response)
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if errorField, hasError := outputMap["error"]; hasError && errorField != nil {
								errorStr := fmt.Sprintf("%v", errorField)
								toolResultMsg = fmt.Sprintf("PERMANENT ERROR - Tool '%s' returned error: %s. DO NOT retry this tool.", result.Name, errorStr)
							} else {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
							}
						} else {
							toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
						}
					}
				}

				// CRITICAL FIX: If we generated a loop warning, EMBED it in JSON result
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

					// Send warning as a visible message to the user
					eventChan <- StreamEvent{Type: StreamEventToken, Content: "\n\n" + loopWarning + "\n\n"}

					// Try to embed warning into the JSON result instead of prepending as text
					if strings.HasPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name)) {
						jsonPart := strings.TrimPrefix(toolResultMsg, fmt.Sprintf("Tool '%s' result: ", result.Name))
						var resultData map[string]interface{}
						if err := json.Unmarshal([]byte(jsonPart), &resultData); err == nil {
							resultData["_loopWarning"] = loopWarning
							if newJSON, err := json.Marshal(resultData); err == nil {
								toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(newJSON))
							} else {
								// fallback: text prepend if reserialization fails
								toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
							}
						} else {
							// fallback: text prepend if JSON parse fails
							toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
						}
					} else {
						// fallback: if message doesn't match JSON result format
						toolResultMsg = fmt.Sprintf("%s\n\n%s", loopWarning, toolResultMsg)
					}
				}

				// DISABLED: Tool result truncation removed per user request
				// Full tool results will be sent to AI without size limitations
				// Previous truncation: 10KB (default) or 30KB (fallback model)
				// Note: This may cause token limit errors with very large tool outputs
				// maxToolResultSize := 10000 // 10KB per tool result (default)
				// if s.usingFallback {
				// 	maxToolResultSize = 30000 // 30KB for fallback model - preserve more context
				// }
				// if len(toolResultMsg) > maxToolResultSize {
				// 	originalSize := len(toolResultMsg)
				// 	// Keep first portion and add truncation notice
				// 	truncatedSize := maxToolResultSize - 500
				// 	toolResultMsg = toolResultMsg[:truncatedSize] + fmt.Sprintf("\n\n... [TRUNCATED: Result was %d chars, showing first %d chars to prevent token limit. If you need more, use a more specific query or process the data in smaller chunks.] ...", originalSize, truncatedSize)
				// 	log.Printf("[Tool Result Truncation] Truncated tool '%s' result from %d to %d chars to prevent token limit",
				// 		result.Name, originalSize, len(toolResultMsg))
				// }

				// CRITICAL FIX: Add tool_call message BEFORE tool_result (required by Anthropic API)
				// This ensures proper conversation history tracking
				// Strip tool call JSON from content (some models like Groq include it in text)
				cleanContent := responseText
				if strings.Contains(cleanContent, `[{"id":"`) {
					// Find and remove the JSON array part
					if idx := strings.Index(cleanContent, `[{"id":"`); idx >= 0 {
						cleanContent = strings.TrimSpace(cleanContent[:idx])
					}
				}

				currentMessages = append(currentMessages, Message{
					Role:    "tool_call",
					Content: cleanContent,
					ToolCall: &ToolCall{
						ID:   toolCall.ID,
						Name: toolCall.Name,
						Args: toolCall.Args,
					},
				})


			// GUARDRAIL: Check if adding this tool result would exceed 120KB context limit
			// Calculate current context size
			currentContextSize := calculateContextSize(currentMessages)

			// Estimate size of the tool result we're about to add
			toolResultSize := len(fmt.Sprintf("%v", toolResultMsg))
			if result.Error != "" {
				toolResultSize += len(result.Error)
			}

			const maxContextBeforeToolResult = 120000 // 120KB limit

			// If adding this result would exceed 120KB, replace it with a warning
			if currentContextSize+toolResultSize > maxContextBeforeToolResult {
				log.Printf("[Context Guardrail] Tool result would exceed 120KB limit (current: %d, result: %d, total: %d). Blocking with guidance message.",
					currentContextSize, toolResultSize, currentContextSize+toolResultSize)

				// Replace the result with a helpful error message
				toolResultMsg = "⚠️ Context is too big, try different approach to reduce command output.\n\n" +
					fmt.Sprintf("The tool '%s' output (%d bytes) would push context over 120KB limit (current: %d bytes).\n\n",
						toolCall.Name, toolResultSize, currentContextSize) +
					"Suggestions:\n" +
					"• Use filters, grep, or head/tail to limit output size\n" +
					"• Process data in smaller chunks\n" +
					"• Write large outputs to files instead of returning them\n" +
					"• Use pagination or summarization for list operations"

				result.Error = "Context size limit exceeded"
			}
				// CRITICAL FIX: Add tool_result message with proper structure (not system role)
				// This ensures the AI provider actually receives and processes the tool results
				// The toolResultMsg is used for Output to preserve truncation and loop warning logic
				currentMessages = append(currentMessages, Message{
					Role:    "tool_result",
					Content: "",
					ToolResult: &ToolResult{
						ID:         toolCall.ID,
						Name:       toolCall.Name,
						Output:     toolResultMsg, // Preserve processed result with truncation/warnings
						Error:      result.Error,
						DurationMs: result.DurationMs,
					},
				})

				// WORKFLOW STATE UPDATE: Update workflow state after successful tool execution
				// Apply to ALL models to match prescriptive filter behavior (line 714)
				if result.Error == "" {
					debugLog("WORKFLOW: Tool %s succeeded, checking for state update", toolCall.Name)
					switch toolCall.Name {
					case "coordinator_list_human_tasks":
						if workflowState["step"].(int) == 0 {
							workflowState["step"] = 1
							log.Printf("[Workflow State] Step 1 complete: listed tasks")
							debugLog("WORKFLOW: Updated state to step 1")
						}

					case "coordinator_create_human_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if taskID, hasTaskID := outputMap["taskId"].(string); hasTaskID && taskID != "" {
								workflowState["step"] = 2
								workflowState["humanTaskId"] = taskID
								lastCreatedHumanTaskId = taskID // FIX #9: Cache for instant validation
								log.Printf("[Workflow State] Step 2 complete: created human task %s", taskID)
							} else if similarTasksFound, _ := outputMap["similarTasksFound"].(bool); similarTasksFound {
								// Case 2: Similar task found - use existing task instead of creating new one
								if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
									if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
										if existingTaskID, ok := firstTask["taskId"].(string); ok && existingTaskID != "" {
											workflowState["step"] = 2
											workflowState["humanTaskId"] = existingTaskID
											lastCreatedHumanTaskId = existingTaskID // FIX #9: Cache for instant validation
											log.Printf("[Workflow State] Step 2 complete: using existing similar task %s", existingTaskID)
										}
									}
								}
							}
						}

					case "code_index_search":
						workflowState["step"] = 3
						workflowState["searchCompleted"] = true
						log.Printf("[Workflow State] Step 3 complete: code search done")
						debugLog("WORKFLOW: Updated state to step 3 (after code_index_search)")

					case "create_agent_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if agentTaskID, hasAgentTaskID := outputMap["taskId"].(string); hasAgentTaskID && agentTaskID != "" {
								workflowState["step"] = 4
								workflowState["agentTaskId"] = agentTaskID
								lastCreatedAgentTaskId = agentTaskID // FIX #10: Cache for instant validation
								log.Printf("[Workflow State] Step 4 complete: created agent task %s", agentTaskID)
							}
						}

					case "execute_subagent":
						workflowState["step"] = 5
						log.Printf("[Workflow State] Step 5 complete: subagent launched")
					}
				}

				// FALLBACK MODEL ENHANCEMENT: Add explicit state tracking for workflow comprehension
				// Haiku (smaller model) benefits from explicit guidance on workflow state and next steps
				if s.usingFallback {
					// Extract session ID from system prompt (first message)
					sessionID := ""
					if len(currentMessages) > 0 && currentMessages[0].Role == "system" {
						sessionID = extractSessionIDFromSystemPrompt(currentMessages[0].Content)
					}

					stateGuidance := s.generateWorkflowStateGuidance(toolCall.Name, result, toolCallCount, sessionID)
					if stateGuidance != "" {
						log.Printf("[Fallback State Tracking] Injecting workflow guidance after tool '%s' with sessionID: %s", toolCall.Name, sessionID)
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: stateGuidance,
						})
					}
				}

				log.Printf("[AI Processing] Context after tool %d: %d messages, %d total chars",
					toolCallCount, len(currentMessages), func() int {
						sum := 0
						for _, m := range currentMessages {
							sum += len(m.Content)
						}
						return sum
					}())
				debugLog("END OF TOOL CALL: currentMessages=%d, continuing to next iteration", len(currentMessages))
			}
			debugLog("END OF ALL TOOL CALLS IN THIS ITERATION: toolCallCount=%d, looping back", toolCallCount)
		}

		debugLog("LOOP EXITED: toolCallCount=%d/%d, iterationCount=%d/%d", toolCallCount, maxToolCalls, iterationCount, s.config.MaxIterations)
		// Check which limit was reached
		if toolCallCount >= maxToolCalls {
			// Max tool calls reached
			log.Printf("[ChatService] Max tool calls reached - RequestID: %s - Tool calls: %d, Iterations: %d",
				requestID, toolCallCount, iterationCount)

			// Send error notification to user via WebSocket
			eventChan <- StreamEvent{
				Type: StreamEventError,
				Error: fmt.Sprintf(
					"⚠️ maximum tool calls limit reached (%d tool calls).\n\n"+
						"The AI has reached the maximum number of tool calls allowed for this request.\n\n"+
						"**What happened:**\n"+
						"- The AI made %d tool calls across %d reasoning iterations\n"+
						"- This usually indicates a complex task or a retry loop\n\n"+
						"**What you can do:**\n"+
						"1. **Break the task into smaller steps** - Ask for one thing at a time\n"+
						"2. **Provide more specific instructions** - Reduce ambiguity\n"+
						"3. **Check for errors** - Review any error messages above\n"+
						"4. **Continue the conversation** - Send a follow-up message to continue\n\n"+
						"The conversation has been saved. You can continue by sending a new message.",
					maxToolCalls, toolCallCount, iterationCount),
			}
		} else {
			// Max iterations reached
			log.Printf("[ChatService] Max iterations reached - RequestID: %s - Total iterations: %d, Tool calls: %d",
				requestID, iterationCount, toolCallCount)

			// Send error notification to user via WebSocket
			eventChan <- StreamEvent{
				Type: StreamEventError,
				Error: fmt.Sprintf(
					"⚠️ Maximum iteration limit reached (%d iterations).\n\n"+
						"The AI needs more steps to complete this task than currently allowed.\n\n"+
						"**What happened:**\n"+
						"- The AI made %d tool calls across %d reasoning iterations\n"+
						"- This usually indicates a complex task or a retry loop\n\n"+
						"**What you can do:**\n"+
						"1. **Break the task into smaller steps** - Ask for one thing at a time\n"+
						"2. **Provide more specific instructions** - Reduce ambiguity\n"+
						"3. **Check for errors** - Review any error messages above\n"+
						"4. **Increase the limit** - Set MAX_ITERATIONS higher in .env (current: %d)\n\n"+
						"The conversation has been saved. You can continue by sending a new message.",
					iterationCount, toolCallCount, iterationCount, s.config.MaxIterations),
			}
		}
	}()

	return eventChan, nil
}

// StreamChatWithToolsFiltered sends messages to AI provider with restricted tool access.
// This is used for subagents to prevent them from calling coordinator tools.
// Only the specified tools in allowedToolNames will be available to the AI.
//
// Security Features:
//   - Blocks coordinator tools (execute_subagent, coordinator_create_*, etc.)
//   - Validates allowlist at both filter time and execution time (defense-in-depth)
//   - Returns security violation error if blocked tools are in allowlist
//
// Parameters:
//   - ctx: Context with identity and request ID
//   - messages: Conversation history
//   - maxToolCalls: Maximum tool calls before stopping (default 5)
//   - allowedToolNames: Whitelist of tool names the AI can call
//
// Returns:
//   - Channel of StreamEvents (tokens, tool calls, tool results, errors)
//   - Error if security validation fails or initial setup fails
func (s *ChatService) StreamChatWithToolsFiltered(ctx context.Context, messages []Message, maxToolCalls int, allowedToolNames []string) (<-chan StreamEvent, error) {
	identity := s.getIdentityFromContext(ctx)
	requestID := s.getRequestIDFromContext(ctx)

	// Log the request
	if identity != nil {
		log.Printf("[ChatService] Tool-filtered request from %s (%s) - RequestID: %s - Provider: %s Model: %s - Allowed tools: %v",
			identity.Name, identity.Type, requestID, s.config.Provider, s.config.Model, allowedToolNames)
	} else {
		log.Printf("[ChatService] Tool-filtered request (no identity) - RequestID: %s - Provider: %s Model: %s - Allowed tools: %v",
			requestID, s.config.Provider, s.config.Model, allowedToolNames)
	}

	// Validate messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("messages cannot be empty")
	}

	// Default max tool calls to prevent loops
	if maxToolCalls <= 0 {
		maxToolCalls = 5
	}

	// Create output channel for events
	eventChan := make(chan StreamEvent, 100)

	// SECURITY: Validate that delegation tool is not in the allowed list
	// This prevents direct subagent chats from creating subchats and delegating
	// Note: Task management tools are allowed to enable tracking (human tasks, agent tasks, todos)
	blockedToolsForDirectSubagents := []string{
		"execute_subagent", // CRITICAL: Prevent direct subagents from creating subchats and delegating to other agents
	}

	for _, blocked := range blockedToolsForDirectSubagents {
		for _, allowed := range allowedToolNames {
			if allowed == blocked {
				return nil, fmt.Errorf("SECURITY VIOLATION: Tool '%s' is blocked in direct subagent context. Direct subagents cannot delegate to other agents", blocked)
			}
		}
	}

	// Get FILTERED tools for LangChain (only allowed tools)
	tools := s.toolRegistry.GetFilteredToolsForLangChain(allowedToolNames)

	log.Printf("[ChatService] Filtered %d tools for subagent (from allowlist of %d tools) - Security validated", len(tools), len(allowedToolNames))

	// Check if provider supports tools
	supportsTools := false
	if toolProvider, ok := s.provider.(ToolCapableProvider); ok {
		supportsTools = toolProvider.SupportsTools()
	}

	if !supportsTools || len(tools) == 0 {
		// Fallback to text-only streaming
		log.Printf("[ChatService] Provider doesn't support tools or no tools registered - RequestID: %s", requestID)
		go func() {
			defer close(eventChan)
			textChan, err := s.provider.StreamChat(ctx, messages)
			if err != nil {
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}
			for chunk := range textChan {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
			}
		}()
		return eventChan, nil
	}

	// Start tool-enabled streaming
	go func() {
		defer close(eventChan)
		debugLog("========== GOROUTINE STARTED ==========")
		debugLog("Request ID: %s, MaxToolCalls: %d", requestID, maxToolCalls)

		toolCallCount := 0
		iterationCount := 0
		currentMessages := append([]Message{}, messages...) // Copy messages
		debugLog("Initial message count: %d", len(currentMessages))

		// Tool result cache: prevent duplicate tool executions
		resultCache := NewToolResultCache()

		// Circuit breaker: track recent tool calls to detect infinite loops
		recentToolCalls := make([]string, 0, 10)
		consecutiveFailures := 0     // Track CONSECUTIVE failures of the same tool+args
		lastFailedSignature := ""    // Signature of the last failed tool call
		// pathValidationRetries not needed in fallback model (Claude handles its own validation)
		toolCallSignature := func(name string, args map[string]interface{}) string {
			argsJSON, _ := json.Marshal(args)
			return fmt.Sprintf("%s(%s)", name, string(argsJSON))
		}

		// WORKFLOW STATE ENFORCEMENT: Track coordinator workflow progress (fallback model only)
		workflowState := map[string]interface{}{
			"step":            0,     // 0=initial, 1=listed, 2=created, 3=searched, 4=agent_task, 5=done
			"humanTaskId":     "",    // Store taskId from step 2
			"searchCompleted": false, // Prevent multiple searches
			"agentTaskId":     "",    // Store agentTaskId from step 4
		}

		// Function to validate workflow tool calls (only enforced for fallback model)
		validateWorkflowTool := func(toolName string) (bool, string) {
			if !s.usingFallback {
				return true, "" // No enforcement for primary model
			}

			step := workflowState["step"].(int)
			humanTaskId := workflowState["humanTaskId"].(string)
			searchCompleted := workflowState["searchCompleted"].(bool)

			switch toolName {
			case "coordinator_list_human_tasks":
				if step == 0 || step == 1 || step == 2 {
					// Allow listing tasks at any early step to retrieve exact taskId
					return true, ""
				}
				return false, "❌ BLOCKED: You already have taskId. NEXT: Call appropriate tool for current step."

			case "coordinator_create_human_task":
				if step == 1 && humanTaskId == "" {
					return true, ""
				}
				if humanTaskId != "" {
					// State-aware guidance based on workflow progress
					if searchCompleted {
						return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'create_agent_task'.", humanTaskId)
					}
					return false, fmt.Sprintf("❌ BLOCKED: Task exists (ID: %s). NEXT: Call 'code_index_search'.", humanTaskId)
				}
				return false, "❌ BLOCKED: Call 'coordinator_list_human_tasks' first."

			case "code_index_search":
				if step == 2 && humanTaskId != "" && !searchCompleted {
					return true, ""
				}
				if searchCompleted {
					return false, "❌ BLOCKED: Search done. NEXT: Call 'create_agent_task'."
				}
				return false, "❌ BLOCKED: Create human task first."

			case "create_agent_task":
				if step == 3 && humanTaskId != "" && searchCompleted {
					return true, ""
				}
				return false, "❌ BLOCKED: Run 'code_index_search' first to get file paths."

			case "execute_subagent":
				agentTaskId := workflowState["agentTaskId"].(string)
				if step == 4 && agentTaskId != "" {
					return true, ""
				}
				return false, "❌ BLOCKED: Call 'create_agent_task' first."
			}

			return true, ""
		}

		// Per-tool circuit breaker thresholds (max duplicate attempts before stopping)
		// Claude models get higher thresholds as they're better at adapting
		isClaudeModel := strings.Contains(strings.ToLower(s.config.Model), "claude") ||
			strings.Contains(strings.ToLower(s.config.Provider), "anthropic")

		var circuitBreakerThresholds map[string]int
		if isClaudeModel {
			// Claude-optimized thresholds: More lenient to allow legitimate multi-file operations
			circuitBreakerThresholds = map[string]int{
				"read_file":         5, // Allow reading multiple files
				"write_file":        2, // Allow one retry for writes
				"list_directory":    4, // Allow exploring directories
				"bash":              5, // Allow command variations
				"code_index_search": 2, // Strict: one search + one retry max
				"create_agent_task": 4, // Allow retries for parameter refinement
				// Default for other tools: 6 attempts
			}
			log.Printf("[Circuit Breaker] Using Claude-optimized thresholds (more lenient)")
		} else {
			// GPT thresholds: More conservative
			circuitBreakerThresholds = map[string]int{
				"read_file":         2, // Stop after 2 attempts (only 1 duplicate allowed)
				"write_file":        1, // Never allow duplicate writes
				"list_directory":    2, // Stop after 2 attempts
				"bash":              3, // Allow more for command variations
				"code_index_search": 3, // Allow query refinement
				// Default for other tools: 4 attempts
			}
			log.Printf("[Circuit Breaker] Using GPT thresholds (conservative)")
		}

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			iterationCount++

			// Calculate context size BEFORE applying sliding window
			contextSize := 0
			for _, msg := range currentMessages {
				contextSize += len(msg.Content)
			}

			// SLIDING WINDOW DISABLED
			// Apply sliding window BEFORE context exceeds model's token limit
			// Claude: 200K tokens (≈800KB text) - use 150KB threshold
			// GPT: 32K tokens (≈128KB text) - use 40KB threshold to be safe
			// var maxContextSize int
			// var maxMessages int
			// if isClaudeModel {
			// 	maxContextSize = 150000 // 150KB for Claude (≈37K tokens, leaves room for output)
			// 	maxMessages = 20        // Keep more messages for Claude
			// } else {
			// 	maxContextSize = 40000 // 40KB for GPT (≈10K tokens)
			// 	maxMessages = 6        // Conservative for GPT
			// }

			// if contextSize > maxContextSize {
			// 	log.Printf("[Sliding Window - Filtered] Context size %d chars exceeds threshold %d chars, applying window",
			// 		contextSize, maxContextSize)
			// 	currentMessages = applySlidingWindow(currentMessages, maxMessages)

			// 	// Recalculate after trimming
			// 	contextSize = 0
			// 	for _, msg := range currentMessages {
			// 		contextSize += len(msg.Content)
			// 	}
			// }

			// Log iteration details with more info
			log.Printf("[AI Processing - Filtered Tools] === ITERATION %d START ===", iterationCount)
			log.Printf("[AI Processing - Filtered Tools] Iteration: %d, Request: %d chars, Context: %d chars, Tool calls so far: %d, Max iterations: %d",
				iterationCount, contextSize, contextSize, toolCallCount, s.config.MaxIterations)

			// DEBUG: Log context details before LLM API call
			contextSize = calculateContextSize(currentMessages)

			// LOG EXACT MESSAGES SENT TO AI
			log.Printf("[DEBUG - AI Input] Sending %d messages to AI (iteration %d):", len(currentMessages), iterationCount)
			for i, msg := range currentMessages {
				contentPreview := msg.Content
				if len(contentPreview) > 200 {
					contentPreview = contentPreview[:200] + "..."
				}

				if msg.Role == "tool_result" && msg.ToolResult != nil {
					outputPreview := fmt.Sprintf("%v", msg.ToolResult.Output)
					if len(outputPreview) > 500 {
						outputPreview = outputPreview[:500] + "..."
					}
					log.Printf("  [%d] Role: %s, ToolName: %s, ToolID: %s, Output: %s",
						i, msg.Role, msg.ToolResult.Name, msg.ToolResult.ID, outputPreview)
				} else if msg.Role == "tool_call" && msg.ToolCall != nil {
					argsJSON, _ := json.Marshal(msg.ToolCall.Args)
					log.Printf("  [%d] Role: %s, ToolName: %s, ToolID: %s, Args: %s",
						i, msg.Role, msg.ToolCall.Name, msg.ToolCall.ID, string(argsJSON))
				} else {
					log.Printf("  [%d] Role: %s, Content: %s", i, msg.Role, contentPreview)
				}
			}

			// Call provider with FILTERED tools
			toolProvider := s.provider.(ToolCapableProvider)
			response, err := toolProvider.StreamChatWithTools(ctx, currentMessages, tools)
			if err != nil {
				log.Printf("[ChatService] ERROR - RequestID: %s - Tool call failed: %v", requestID, err)
				eventChan <- StreamEvent{Type: StreamEventError, Error: err.Error()}
				return
			}

			// Stream response tokens
			var responseText string
			responseTokens := 0
			for chunk := range response.TextChannel {
				eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				responseText += chunk
				responseTokens++
			}

			// Log iteration response details
			log.Printf("[AI Processing - Filtered] === ITERATION %d COMPLETE ===", iterationCount)
			log.Printf("[AI Processing - Filtered] Iteration: %d complete, Response: %d tokens, Tool calls requested: %d",
				iterationCount, responseTokens, len(response.ToolCalls))

			// DEBUG: Check if responseText contains tool JSON (FILTERED PATH)
			if strings.Contains(responseText, `[{"id":"`) {
				log.Printf("[DEBUG TOOL_EXECUTOR FILTERED] ⚠️  WARNING: responseText contains tool JSON!")
				preview := responseText
				if len(preview) > 300 {
					preview = preview[:300] + "..."
				}
				log.Printf("[DEBUG TOOL_EXECUTOR FILTERED] responseText preview: %s", preview)
			} else {
				log.Printf("[DEBUG TOOL_EXECUTOR FILTERED] ✓ responseText is clean (no tool JSON)")
				preview := responseText
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				log.Printf("[DEBUG TOOL_EXECUTOR FILTERED] responseText preview: %s", preview)
			}

			// Log end turn based on stop_reason
			if response.StopReason == "end_turn" {
				log.Printf("[AI Processing - Filtered] 🏁 AI ENDED TURN - stop_reason=end_turn (natural completion)")
			} else if len(response.ToolCalls) == 0 {
				log.Printf("[AI Processing - Filtered] 🏁 AI ENDED TURN - stop_reason=%s, no tool calls", response.StopReason)
			}

			// Check stop reason to determine if turn has ended
			if response.StopReason == "end_turn" || len(response.ToolCalls) == 0 {
				// No more tool calls, we're done
				log.Printf("[ChatService - Filtered] Stream complete - RequestID: %s - StopReason: %s - Total iterations: %d, Tool calls: %d",
					requestID, response.StopReason, iterationCount, toolCallCount)
				return
			}

			// Process each tool call
			for _, toolCall := range response.ToolCalls {
				toolCallCount++
				if toolCallCount > maxToolCalls {
					log.Printf("[ChatService - Filtered] Max tool calls reached (%d) - RequestID: %s", maxToolCalls, requestID)
					eventChan <- StreamEvent{Type: StreamEventError, Error: fmt.Sprintf("maximum tool calls (%d) exceeded", maxToolCalls)}
					return
				}

				// Log tool request with arguments
				argsJSON, _ := json.Marshal(toolCall.Args)
				log.Printf("[Tool Request - Filtered] AI requested tool '%s' with args: %s",
					toolCall.Name, string(argsJSON))

				// WORKFLOW VALIDATION: Check if this tool call is allowed in current workflow state
				var result ToolResult
				if s.usingFallback {
					allowed, blockMessage := validateWorkflowTool(toolCall.Name)
					if !allowed {
						log.Printf("[Workflow Enforcer - Filtered] BLOCKED tool '%s' - %s", toolCall.Name, blockMessage)

						// Create a blocking error result so model understands the tool failed
						result = ToolResult{
							ID:         toolCall.ID,
							Name:       toolCall.Name,
							Output:     nil,
							Error:      blockMessage,
							DurationMs: 0,
						}

						// Send the error result to the model
						eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

						// Add tool result to message history as an error
						currentMessages = append(currentMessages, Message{
							Role:    "assistant",
							Content: fmt.Sprintf("I attempted to call '%s' but it was blocked.", toolCall.Name),
						})
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: fmt.Sprintf("TOOL ERROR from '%s': %s", toolCall.Name, blockMessage),
						})

						// Continue to next iteration - don't execute the blocked tool
						continue
					}
				}

				// Send tool call event
				eventChan <- StreamEvent{Type: StreamEventToolCall, ToolCall: &toolCall}

				// Generate signature for cache and circuit breaker
				signature := toolCallSignature(toolCall.Name, toolCall.Args)

				// Check tool result cache BEFORE execution
				cachedResult, found := resultCache.Get(signature)
				if found {
					// Use cached result - avoid redundant execution
					result = *cachedResult
					log.Printf("[Tool Cache HIT] Using cached result for '%s' - skipping execution", toolCall.Name)

					// Add cache hit notice to the result
					cacheNotice := fmt.Sprintf("🔁 CACHED RESULT: You already called '%s' with these exact arguments. Using previous result instead of re-executing.", toolCall.Name)
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_cacheNotice"] = cacheNotice
						result.Output = newOutput
					}
				} else {
					// RUNTIME SECURITY CHECK: Block delegation tool even if AI tries to call it
					// This is a defense-in-depth measure (should be caught earlier by filtering)
					isBlocked := false
					for _, blocked := range blockedToolsForDirectSubagents {
						if toolCall.Name == blocked {
							isBlocked = true
							result = ToolResult{
								ID:         toolCall.ID,
								Name:       toolCall.Name,
								Output:     nil,
								Error:      fmt.Sprintf("🚫 SECURITY BLOCK: Tool '%s' is not allowed in direct subagent context. You are working autonomously - do not delegate to other agents. Use task management tools (coordinator_create_human_task, coordinator_create_agent_task, etc.) to track your work, but execute it yourself.", toolCall.Name),
								DurationMs: 0,
							}
							log.Printf("[SECURITY] Blocked attempt to call '%s' in direct subagent context - Tool not in allowlist", toolCall.Name)
							break
						}
					}

					if !isBlocked {
						// Execute tool (no cached result available)
						// Inject humanTaskId from workflowState into context for auto-population
						toolCtx := ctx
						if humanTaskID, ok := workflowState["humanTaskId"].(string); ok && humanTaskID != "" {
							toolCtx = context.WithValue(ctx, "lastHumanTaskId", humanTaskID)
						}
						result = s.toolRegistry.ExecuteToolCall(toolCtx, toolCall)
						log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)
					}

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// Send tool result event
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// Log tool execution with comprehensive response data (Claude optimization)
				if result.Error != "" {
					log.Printf("[ChatService - Filtered] Tool '%s' failed - RequestID: %s - Error: %s - Duration: %dms",
						result.Name, requestID, result.Error, result.DurationMs)
					// Log complete error response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					log.Printf("[Tool Response - ERROR - Filtered] Tool: %s | Args: %s | Error: %s",
						result.Name, string(argsJSON), result.Error)
				} else {
					log.Printf("[ChatService - Filtered] Tool '%s' succeeded - RequestID: %s - Duration: %dms",
						result.Name, requestID, result.DurationMs)
					// Log complete success response for analysis
					argsJSON, _ := json.Marshal(toolCall.Args)
					outputJSON, _ := json.Marshal(result.Output)
					log.Printf("[Tool Response - SUCCESS - Filtered] Tool: %s | Args: %s | Output: %s",
						result.Name, string(argsJSON), string(outputJSON))
				}

				// CRITICAL: Track CONSECUTIVE failed tool calls - stop on 3+ consecutive identical failures
				// This allows: compile → error → fix → compile (normal flow)
				// But prevents: compile → error → compile → error → compile (infinite loop)
				if result.Error != "" {
					// Check if this is the same as the last failed call
					if lastFailedSignature == signature {
						consecutiveFailures++
						if consecutiveFailures >= 3 {
							// Third CONSECUTIVE failure with same args - stop!
							log.Printf("[Circuit Breaker - Filtered] Tool '%s' failed 3 times CONSECUTIVELY with identical arguments", toolCall.Name)
							// Return error to AI, don't stop execution
							// The AI should see this error and try a different approach
							loopWarning := fmt.Sprintf("❌ CRITICAL: Tool '%s' has FAILED 3 TIMES IN A ROW with identical arguments.\n\n"+
								"Error: %s\n\n"+
								"🛑 This approach is NOT working. You MUST try something different:\n"+
								"   - If file not found: List the directory first to see what files actually exist\n"+
								"   - If path wrong: Try a different path or check your working directory\n"+
								"   - If tool incompatible: Use a completely different tool or approach\n\n"+
								"DO NOT call this tool with these arguments again!", toolCall.Name, result.Error)

							// Add warning to current messages so AI sees it
							currentMessages = append(currentMessages, Message{
								Role:    "system",
								Content: loopWarning,
							})

							// Reset counters
							consecutiveFailures = 0
							lastFailedSignature = ""
						} else {
							lastFailedSignature = signature
						}
					} else {
						// Different failure, reset counter
						consecutiveFailures = 1
						lastFailedSignature = signature
					}
				} else {
					// Success - reset failure tracking
					consecutiveFailures = 0
					lastFailedSignature = ""
				}

				// Circuit breaker: check for repeated tool calls
				recentToolCalls = append(recentToolCalls, signature)
				if len(recentToolCalls) > 10 {
					recentToolCalls = recentToolCalls[1:]
				}

				// Count duplicates
				totalCount := 0
				for _, sig := range recentToolCalls {
					if sig == signature {
						totalCount++
					}
				}

				// Get tool-specific threshold
				threshold := circuitBreakerThresholds[toolCall.Name]
				if threshold == 0 {
					threshold = 4 // Default threshold
				}

				// Progressive warnings
				var loopWarning string
				if totalCount == 2 {
					loopWarning = fmt.Sprintf("⚠️  WARNING: You already called '%s' with these exact arguments 1 time before. Use the previous result instead of repeating.", toolCall.Name)
				} else if totalCount == 3 && threshold > 3 {
					loopWarning = fmt.Sprintf("🔁 LOOP DETECTED: You called '%s' with identical arguments 2 times already. You are stuck in a loop! Use previous results or try a DIFFERENT approach.", toolCall.Name)
				} else if totalCount >= threshold {
					log.Printf("[Circuit Breaker] Tool '%s' called %d times (threshold: %d) - stopping infinite loop", toolCall.Name, totalCount, threshold)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: fmt.Sprintf("Circuit breaker triggered: tool '%s' called repeatedly (%d times) with identical arguments. The AI is stuck in an infinite loop.", toolCall.Name, totalCount),
					}
					return
				}

				// CRITICAL FIX: Add tool_call message with tool_use block (not plain text)
				// This ensures the model has conversation memory of making the tool call
				// Role must be "tool_call" to match provider.go:429 check
				currentMessages = append(currentMessages, Message{
					Role:    "tool_call",
					Content: responseText,
					ToolCall: &ToolCall{
						ID:   toolCall.ID,
						Name: toolCall.Name,
						Args: toolCall.Args,
					},
				})

				// CRITICAL FIX: Embed loop warning into the result.Output (not as separate text)
				// This ensures it's part of the structured response the model sees
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

					// Send warning as a visible message to the user
					eventChan <- StreamEvent{Type: StreamEventToken, Content: "\n\n" + loopWarning + "\n\n"}

					// Embed warning directly into result.Output
					if outputMap, ok := result.Output.(map[string]interface{}); ok {
						// Create a new map with warning injected
						newOutput := make(map[string]interface{})
						for k, v := range outputMap {
							newOutput[k] = v
						}
						newOutput["_loopWarning"] = loopWarning
						result.Output = newOutput
					} else {
						// For non-map outputs, wrap in a map
						result.Output = map[string]interface{}{
							"result":       result.Output,
							"_loopWarning": loopWarning,
						}
					}
				}

				// DISABLED: Tool result truncation removed per user request
				// Full tool results will be sent to AI without size limitations
				// Previous truncation: 10KB with structured preview
				// Note: This may cause token limit errors with very large tool outputs
				// const maxToolResultSize = 10000
				// if outputJSON, err := json.Marshal(result.Output); err == nil {
				// 	if len(outputJSON) > maxToolResultSize {
				// 		originalSize := len(outputJSON)
				// 		truncated := string(outputJSON[:maxToolResultSize-500])
				// 		result.Output = map[string]interface{}{
				// 			"_truncated": true,
				// 			"_message":   fmt.Sprintf("Result was %d chars, showing first %d chars", originalSize, maxToolResultSize-500),
				// 			"_preview":   truncated,
				// 		}
				// 		log.Printf("[Tool Result Truncation] Truncated tool '%s' result from %d to %d chars",
				// 			result.Name, originalSize, len(truncated))
				// 	}
				// }


			// GUARDRAIL: Check if adding this tool result would exceed 120KB context limit
			// Calculate current context size
			currentContextSize := calculateContextSize(currentMessages)

			// Estimate size of the tool result we're about to add
			var toolResultSize int
			if result.Error != "" {
				toolResultSize = len(result.Error)
			} else {
				toolResultSize = len(fmt.Sprintf("%v", result.Output))
			}

			const maxContextBeforeToolResult = 120000 // 120KB limit

			// If adding this result would exceed 120KB, replace it with a warning
			if currentContextSize+toolResultSize > maxContextBeforeToolResult {
				log.Printf("[Context Guardrail] Tool result would exceed 120KB limit (current: %d, result: %d, total: %d). Blocking with guidance message.",
					currentContextSize, toolResultSize, currentContextSize+toolResultSize)

				// Replace the result with a helpful error message
				result.Output = "⚠️ Context is too big, try different approach to reduce command output.\n\n" +
					fmt.Sprintf("The tool '%s' output (%d bytes) would push context over 120KB limit (current: %d bytes).\n\n",
						toolCall.Name, toolResultSize, currentContextSize) +
					"Suggestions:\n" +
					"• Use filters, grep, or head/tail to limit output size\n" +
					"• Process data in smaller chunks\n" +
					"• Write large outputs to files instead of returning them\n" +
					"• Use pagination or summarization for list operations"

				result.Error = "Context size limit exceeded"
			}
				// CRITICAL FIX: Add tool_result message with proper role (user, not system)
				// This matches Anthropic's API format and ensures conversation continuity
				currentMessages = append(currentMessages, Message{
					Role:    "tool_result",
					Content: "", // Content is in ToolResult
					ToolResult: &ToolResult{
						ID:         toolCall.ID,
						Name:       toolCall.Name,
						Output:     result.Output,
						Error:      result.Error,
						DurationMs: result.DurationMs,
					},
				})

				// WORKFLOW STATE UPDATE: Update workflow state after successful tool execution (filtered function)
				// Apply to ALL models to match prescriptive filter behavior
				if result.Error == "" {
					switch toolCall.Name {
					case "coordinator_list_human_tasks":
						if workflowState["step"].(int) == 0 {
							workflowState["step"] = 1
							log.Printf("[Workflow State - Filtered] Step 1 complete: listed tasks")
						}

					case "coordinator_create_human_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if taskID, hasTaskID := outputMap["taskId"].(string); hasTaskID && taskID != "" {
								workflowState["step"] = 2
								workflowState["humanTaskId"] = taskID
								log.Printf("[Workflow State - Filtered] Step 2 complete: created human task %s", taskID)
							} else if similarTasksFound, _ := outputMap["similarTasksFound"].(bool); similarTasksFound {
								// Case 2: Similar task found - use existing task instead of creating new one
								if similarTasks, ok := outputMap["similarTasks"].([]interface{}); ok && len(similarTasks) > 0 {
									if firstTask, ok := similarTasks[0].(map[string]interface{}); ok {
										if existingTaskID, ok := firstTask["taskId"].(string); ok && existingTaskID != "" {
											workflowState["step"] = 2
											workflowState["humanTaskId"] = existingTaskID
											log.Printf("[Workflow State - Filtered] Step 2 complete: using existing similar task %s", existingTaskID)
										}
									}
								}
							}
						}

					case "code_index_search":
						workflowState["step"] = 3
						workflowState["searchCompleted"] = true
						log.Printf("[Workflow State - Filtered] Step 3 complete: code search done")

					case "create_agent_task":
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if agentTaskID, hasAgentTaskID := outputMap["taskId"].(string); hasAgentTaskID && agentTaskID != "" {
								workflowState["step"] = 4
								workflowState["agentTaskId"] = agentTaskID
								log.Printf("[Workflow State - Filtered] Step 4 complete: created agent task %s", agentTaskID)
							}
						}

					case "execute_subagent":
						workflowState["step"] = 5
						log.Printf("[Workflow State - Filtered] Step 5 complete: subagent launched")
					}
				}
			}
		}

		// Check which limit was reached
		if toolCallCount >= maxToolCalls {
			// Max tool calls reached
			log.Printf("[ChatService - Filtered] Max tool calls reached - RequestID: %s - Tool calls: %d, Iterations: %d",
				requestID, toolCallCount, iterationCount)

			// Send error notification to user via WebSocket
			eventChan <- StreamEvent{
				Type: StreamEventError,
				Error: fmt.Sprintf(
					"⚠️ maximum tool calls limit reached (%d tool calls).\n\n"+
						"The AI has reached the maximum number of tool calls allowed for this request.\n\n"+
						"**What happened:**\n"+
						"- The AI made %d tool calls across %d reasoning iterations\n"+
						"- This usually indicates a complex task or a retry loop\n\n"+
						"**What you can do:**\n"+
						"1. **Break the task into smaller steps** - Ask for one thing at a time\n"+
						"2. **Provide more specific instructions** - Reduce ambiguity\n"+
						"3. **Check for errors** - Review any error messages above\n"+
						"4. **Continue the conversation** - Send a follow-up message to continue\n\n"+
						"The conversation has been saved. You can continue by sending a new message.",
					maxToolCalls, toolCallCount, iterationCount),
			}
		} else {
			// Max iterations reached
			log.Printf("[ChatService - Filtered] Max iterations reached - RequestID: %s - Total iterations: %d, Tool calls: %d",
				requestID, iterationCount, toolCallCount)

			// Send error notification to user via WebSocket
			eventChan <- StreamEvent{
				Type: StreamEventError,
				Error: fmt.Sprintf(
					"⚠️ Maximum iteration limit reached (%d iterations).\n\n"+
						"The AI needs more steps to complete this task than currently allowed.\n\n"+
						"**What happened:**\n"+
						"- The AI made %d tool calls across %d reasoning iterations\n"+
						"- This usually indicates a complex task or a retry loop\n\n"+
						"**What you can do:**\n"+
						"1. **Break the task into smaller steps** - Ask for one thing at a time\n"+
						"2. **Provide more specific instructions** - Reduce ambiguity\n"+
						"3. **Check for errors** - Review any error messages above\n"+
						"4. **Increase the limit** - Set MAX_ITERATIONS higher in .env (current: %d)\n\n"+
						"The conversation has been saved. You can continue by sending a new message.",
					iterationCount, toolCallCount, iterationCount, s.config.MaxIterations),
			}
		}
	}()

	return eventChan, nil
}

// generateWorkflowStateGuidance creates explicit state tracking messages for the fallback model
// to help it understand the 5-step coordinator workflow and what to do next.
// This is crucial for smaller models like Haiku that need more explicit guidance.
func (s *ChatService) generateWorkflowStateGuidance(toolName string, result ToolResult, toolCallCount int, sessionID string) string {
	// Skip guidance if tool failed
	if result.Error != "" {
		return ""
	}

	// Extract data from tool result for guidance
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
