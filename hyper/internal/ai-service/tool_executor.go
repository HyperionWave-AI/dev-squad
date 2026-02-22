package aiservice

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"hyper/internal/config"
)

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
//
// Related files:
//   - tool_executor_debug.go: Debug logging utilities
//   - tool_executor_errors.go: Error recovery guidance
//   - tool_executor_circuit.go: Circuit breaker logic (planned)
//   - tool_executor_workflow.go: Workflow state management (planned)
//   - tool_executor_fallback.go: Rate limit handling (planned)

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
		primaryProvider, err := NewChatProvider(s.config, nil)
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

		// Initialize modular components for tool execution
		resultCache := NewToolResultCache()
		circuitBreaker := NewCircuitBreaker(s.config.Model, s.config.Provider)
		workflowState := NewWorkflowState()
		validator := NewTaskValidator(s.toolRegistry)

		// Tool call history: track all executed tools for smart filtering (reduces token usage by ~70%)
		toolCallHistory := make([]ToolResult, 0, 20)

		// Model limits for context window management
		modelLimits := GetModelLimits(s.config.Model, s.config.Provider)

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			// CONTEXT CANCELLATION CHECK: Check at start of each iteration (stop button support)
			select {
			case <-ctx.Done():
				log.Printf("[ChatService] Context cancelled during tool execution loop - RequestID: %s - iterations: %d, tool calls: %d",
					requestID, iterationCount, toolCallCount)
				eventChan <- StreamEvent{Type: StreamEventError, Error: "execution stopped by user"}
				return
			default:
				// Context still active, continue
			}

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
			// Uses ModelLimits from tool_executor_fallback.go
			if contextSize > modelLimits.MaxContextSize {
				log.Printf("[Sliding Window] Context size %d chars exceeds threshold %d chars, applying window",
					contextSize, modelLimits.MaxContextSize)
				currentMessages = applySlidingWindow(currentMessages, modelLimits.MaxMessages)

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

			// NOTE: Smart Tool Filter REMOVED - main chat now has full tool access
			// The coordinator should use discover_tools and delegate work to subchats
			log.Printf("[Tool Access] Full tool access enabled - %d tools available", len(tools))

			// PHASE 3: PRESCRIPTIVE STATE MACHINE - Only allow ONE tool per workflow step
			// This forces ALL models into a linear workflow with zero ambiguity
			// Each step unlocks exactly ONE required tool - model has no choice but to follow the sequence
			// Applied to ALL models (not just Claude) to ensure consistent coordinator workflow
			if false { // DISABLED: Workflow enforcement (was blocking direct tool execution)
				step := workflowState.Step
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

					filteredTools := make([]Tool, 0, len(allowedTools))
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
				if humanTaskID := workflowState.HumanTaskId; humanTaskID != "" {
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

					// Keep provider/auth settings unchanged and switch models only.
					// Claude fallback uses the configured provider route (native Anthropic or OpenAI-compatible gateway).
					if strings.Contains(strings.ToLower(s.config.FallbackModel), "claude") {
						log.Printf("[Rate Limit] Claude fallback selected - using configured provider route")
					}

					// Recreate provider with fallback model
					fallbackProvider, err := NewChatProvider(s.config, nil)
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

						// Re-create circuit breaker with Claude thresholds (more lenient)
						circuitBreaker = NewCircuitBreaker(s.config.Model, s.config.Provider)
						log.Printf("[Circuit Breaker] Re-created with Claude thresholds after fallback")
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
			// According to Claude API docs, stop_reason is the ONLY authoritative flag:
			// - "end_turn": model finished naturally, turn is over
			// - "tool_use": model is requesting tool execution, turn continues
			// - "max_tokens", "stop_sequence", etc.: other completion reasons
			//
			// IMPORTANT: Tool call count is NOT a reliable indicator of end turn!
			// The model can return 0 tool calls with stop_reason="tool_use" if it's continuing.
			if response.StopReason == "end_turn" {
				debugLog("EXIT: StopReason=end_turn - streaming final response (%d chunks)", len(collectedChunks))
				// Turn ended - stream the collected text (final response)
				for _, chunk := range collectedChunks {
					eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				}
				log.Printf("[ChatService] Stream complete - RequestID: %s - StopReason: end_turn - Total iterations: %d, Tool calls: %d",
					requestID, iterationCount, toolCallCount)
				debugLog("========== GOROUTINE ENDED (stop_reason=end_turn) ==========")
				return
			}

			// If we get here with 0 tool calls and stop_reason != "end_turn", something is wrong
			if len(response.ToolCalls) == 0 {
				log.Printf("[ChatService] WARNING: StopReason=%s but no tool calls - treating as end turn", response.StopReason)
				// Stream response and end
				for _, chunk := range collectedChunks {
					eventChan <- StreamEvent{Type: StreamEventToken, Content: chunk}
				}
				log.Printf("[ChatService] Stream complete - RequestID: %s - StopReason: %s (fallback) - Total iterations: %d, Tool calls: %d",
					requestID, response.StopReason, iterationCount, toolCallCount)
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
				// Uses WorkflowState from tool_executor_workflow.go
				var result ToolResult
				if s.usingFallback {
					allowed, blockMessage := workflowState.ValidateTool(toolCall.Name, s.usingFallback)
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
				// Uses GenerateSignature from tool_executor_circuit.go
				signature := GenerateSignature(toolCall.Name, toolCall.Args)

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
					// CONTEXT CANCELLATION CHECK: Before executing tool (stop button support)
					select {
					case <-ctx.Done():
						log.Printf("[ChatService] Context cancelled before tool '%s' execution - RequestID: %s",
							toolCall.Name, requestID)
						eventChan <- StreamEvent{Type: StreamEventError, Error: "execution stopped by user"}
						return
					default:
						// Context still active, continue
					}

					// Execute tool (no cached result available)
					// Inject humanTaskId from workflowState into context for auto-population
					toolCtx := ctx
					if humanTaskID := workflowState.HumanTaskId; humanTaskID != "" {
						toolCtx = context.WithValue(ctx, LastHumanTaskIDKey, humanTaskID)
					}
					result = s.toolRegistry.ExecuteToolCall(toolCtx, toolCall)
					log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)

					// CODE RESULT SUMMARIZATION: For code_index_search, intelligently summarize results
					// to reduce token usage while preserving critical information for AI decision-making
					if toolCall.Name == "code_index_search" && result.Error == "" {
						// Extract results array from output
						if outputMap, ok := result.Output.(map[string]interface{}); ok {
							if resultsArray, hasResults := outputMap["results"].([]interface{}); hasResults && len(resultsArray) > 0 {
								// Create summarizer and generate summary
								summarizer := NewCodeResultSummarizer(2000) // Max 2000 tokens for summary
								summary := summarizer.SummarizeResults(resultsArray)

								// Log the summarization
								originalSize := len(fmt.Sprintf("%v", resultsArray))
								summarySize := len(summary)
								reductionPercent := 100.0 * (1.0 - float64(summarySize)/float64(originalSize))

								log.Printf("[Code Result Summarization] code_index_search: %d results summarized\n"+
									"  Original size: %d bytes\n"+
									"  Summary size: %d bytes\n"+
									"  Token reduction: %.1f%%",
									len(resultsArray), originalSize, summarySize, reductionPercent)

								// Add summary to output for AI to see
								outputMap["_summary"] = summary
								outputMap["_summaryStats"] = map[string]interface{}{
									"originalSize":     originalSize,
									"summarySize":      summarySize,
									"reductionPercent": reductionPercent,
								}
								result.Output = outputMap
							}
						}
					}
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
								// Uses TaskValidator from tool_executor_validation.go
								if validator.HasRetriedPathValidation(toolCall.Args["query"]) {
									// Already retried once - stop execution with clear error
									log.Printf("[GPT Path Validator] code_index_search retry also returned invalid paths - stopping execution")
									eventChan <- StreamEvent{Type: StreamEventError, Error: GetInvalidPathsError(invalidPaths)}
									return
								} else {
									// First time encountering invalid paths - try automatic retry with refined query
									log.Printf("[GPT Path Validator] First invalid path detection - attempting automatic retry with refined query")

									// Mark this as a retry attempt
									validator.RecordPathValidationRetry(toolCall.Args["query"])

									// Send warning to user about automatic retry
									eventChan <- StreamEvent{Type: StreamEventToken, Content: GetInvalidPathsWarning(invalidPaths)}

									// Modify the tool result to indicate validation failure
									// Inject validation warning into result so GPT knows to refine the search
									InjectPathValidationWarning(&result, invalidPaths, validPaths, len(filePaths))

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
					// Uses TaskValidator from tool_executor_validation.go
					if humanTaskId != "" {
						taskExists, _ := validator.ValidateHumanTaskId(ctx, humanTaskId, workflowState.LastCreatedHumanTaskId)

						if !taskExists {
							// TaskId is invalid even after retries - increment attempt counter
							attempt := validator.RecordInvalidTaskIdAttempt()
							log.Printf("[TaskId Validator] Invalid humanTaskId '%s' after 3 retries with backoff - Attempt %d/3", humanTaskId, attempt)

							if validator.ShouldStopAfterInvalidTaskId() {
								// After 3 attempts, stop execution with clear error
								log.Printf("[TaskId Validator] Failed 3 times - stopping execution")
								eventChan <- StreamEvent{Type: StreamEventError, Error: validator.GetInvalidTaskIdError(humanTaskId)}
								return
							}

							// First or second attempt - inject warning and ask model to list tasks
							eventChan <- StreamEvent{Type: StreamEventToken, Content: validator.GetInvalidTaskIdWarning(humanTaskId)}

							// Replace the result with an error result
							result = validator.CreateBlockedResult(result.ID, humanTaskId)

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
					// Uses WorkflowState.LastCreatedAgentTaskId from tool_executor_workflow.go
					if workflowState.LastCreatedAgentTaskId != "" {
						// Check if model provided wrong ID (hallucinated)
						if providedAgentTaskId != workflowState.LastCreatedAgentTaskId {
							log.Printf("[AgentTaskId Auto-Correct] Model provided wrong agentTaskId: '%s', replacing with correct cached ID: '%s'",
								providedAgentTaskId, workflowState.LastCreatedAgentTaskId)

							// REPLACE the wrong ID with the correct cached one
							toolCall.Args["agentTaskId"] = workflowState.LastCreatedAgentTaskId

							// Re-execute the tool with corrected arguments
							result = s.toolRegistry.ExecuteToolCall(ctx, toolCall)

							log.Printf("[AgentTaskId Auto-Correct] ✅ Re-executed execute_subagent with correct agentTaskId: '%s'", workflowState.LastCreatedAgentTaskId)
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

				// Send tool result event (full result to client for display)
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// Track tool execution in history for smart filtering (keep last 20)
				toolCallHistory = append(toolCallHistory, result)
				if len(toolCallHistory) > 20 {
					toolCallHistory = toolCallHistory[1:] // Remove oldest
				}

				// CIRCUIT BREAKER: Track failures and duplicate calls using CircuitBreaker struct
				// Uses CircuitBreaker from tool_executor_circuit.go
				if result.Error != "" {
					circuitBreaker.RecordFailure(signature)
					if circuitBreaker.ShouldBreakOnFailure() {
						// Third CONSECUTIVE failure with same args - warn AI
						log.Printf("[Circuit Breaker] Tool '%s' failed 3 times CONSECUTIVELY with identical arguments", toolCall.Name)
						loopWarning := circuitBreaker.GetConsecutiveFailureWarning(toolCall.Name, result.Error)
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: loopWarning,
						})
						circuitBreaker.RecordSuccess() // Reset after warning
					}
				} else {
					circuitBreaker.RecordSuccess()
				}

				// Circuit breaker: track repeated tool calls and warn AI
				totalCount := circuitBreaker.RecordToolCall(signature)

				// Progressive warnings to AI (inject into context so AI sees them)
				var loopWarning string
				if circuitBreaker.ShouldBreak(toolCall.Name, totalCount) {
					// Threshold reached - trigger circuit breaker
					log.Printf("[Circuit Breaker] Tool '%s' called %d times - stopping infinite loop", toolCall.Name, totalCount)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: circuitBreaker.GetCircuitBreakerError(toolCall.Name, totalCount),
					}
					return
				}
				loopWarning = circuitBreaker.GetWarning(toolCall.Name, totalCount)

				// Tool execution complete - no verbose logging needed

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
						// NOTE: Error guidance is included in tool result for AI to process,
						// but NOT streamed to user UI (internal guidance only)
						recoveryGuidance := getErrorRecoveryGuidance(result.Name, result.Error, toolCall.Args)
						toolResultMsg = fmt.Sprintf("❌ ERROR in tool '%s': %s\n\n%s", result.Name, result.Error, recoveryGuidance)
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
				// NOTE: Loop warning is included in tool result for AI to process,
				// but NOT streamed to user UI (internal guidance only)
				if loopWarning != "" {
					log.Printf("[Loop Detection] %s", loopWarning)

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

				// GUARDRAIL: Check if adding this tool result would exceed context limit
				// Calculate current context size
				currentContextSize := calculateContextSize(currentMessages)

				// Estimate size of the tool result we're about to add
				toolResultSize := len(fmt.Sprintf("%v", toolResultMsg))
				if result.Error != "" {
					toolResultSize += len(result.Error)
				}

				// Get max context size from config (default 500KB, configurable via MAX_CONTEXT_SIZE env var)
				maxContextBeforeToolResult := config.GetMaxContextSize()

				// If adding this result would exceed limit, replace it with a warning
				if currentContextSize+toolResultSize > maxContextBeforeToolResult {
					log.Printf("[Context Guardrail] Tool result would exceed %s limit (current: %d, result: %d, total: %d). Blocking with guidance message.",
						config.FormatSize(maxContextBeforeToolResult), currentContextSize, toolResultSize, currentContextSize+toolResultSize)

					// Replace the result with a helpful error message
					toolResultMsg = "⚠️ Context is too big, try different approach to reduce command output.\n\n" +
						fmt.Sprintf("The tool '%s' output (%d bytes) would push context over %s limit (current: %d bytes).\n\n",
							toolCall.Name, toolResultSize, config.FormatSize(maxContextBeforeToolResult), currentContextSize) +
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
				// Uses WorkflowState from tool_executor_workflow.go
				workflowState.UpdateAfterToolExecution(toolCall.Name, result, "Workflow State")
				debugLog("WORKFLOW: Tool %s completed, state updated", toolCall.Name)

				// FALLBACK MODEL ENHANCEMENT: Add explicit state tracking for workflow comprehension
				// Haiku (smaller model) benefits from explicit guidance on workflow state and next steps
				// Uses WorkflowState.GetGuidance from tool_executor_workflow.go
				if s.usingFallback {
					// Extract session ID from system prompt (first message)
					sessionID := ""
					if len(currentMessages) > 0 && currentMessages[0].Role == "system" {
						sessionID = extractSessionIDFromSystemPrompt(currentMessages[0].Content)
					}

					stateGuidance := workflowState.GetGuidance(toolCall.Name, result, toolCallCount, sessionID)
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

		// Initialize modular components for tool execution (same as StreamChatWithTools)
		resultCache := NewToolResultCache()
		circuitBreaker := NewCircuitBreaker(s.config.Model, s.config.Provider)
		workflowState := NewWorkflowState()

		for toolCallCount < maxToolCalls && iterationCount < s.config.MaxIterations {
			// CONTEXT CANCELLATION CHECK: Check at start of each iteration (stop button support)
			select {
			case <-ctx.Done():
				log.Printf("[ChatService] Context cancelled during tool execution loop (filtered) - RequestID: %s - iterations: %d, tool calls: %d",
					requestID, iterationCount, toolCallCount)
				eventChan <- StreamEvent{Type: StreamEventError, Error: "execution stopped by user"}
				return
			default:
				// Context still active, continue
			}

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

			// Check stop reason to determine if turn has ended
			// According to Claude API docs, stop_reason is the ONLY authoritative flag
			if response.StopReason == "end_turn" {
				return
			}

			// If we get here with 0 tool calls and stop_reason != "end_turn", treat as end turn
			if len(response.ToolCalls) == 0 {
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
				// Uses WorkflowState from tool_executor_workflow.go
				var result ToolResult
				if s.usingFallback {
					allowed, blockMessage := workflowState.ValidateTool(toolCall.Name, s.usingFallback)
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
				// Uses GenerateSignature from tool_executor_circuit.go
				signature := GenerateSignature(toolCall.Name, toolCall.Args)

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
						// CONTEXT CANCELLATION CHECK: Before executing tool (stop button support)
						select {
						case <-ctx.Done():
							log.Printf("[ChatService] Context cancelled before tool '%s' execution (filtered) - RequestID: %s",
								toolCall.Name, requestID)
							eventChan <- StreamEvent{Type: StreamEventError, Error: "execution stopped by user"}
							return
						default:
							// Context still active, continue
						}

						// Execute tool (no cached result available)
						// Inject humanTaskId from workflowState into context for auto-population
						toolCtx := ctx
						if humanTaskID := workflowState.HumanTaskId; humanTaskID != "" {
							toolCtx = context.WithValue(ctx, LastHumanTaskIDKey, humanTaskID)
						}
						result = s.toolRegistry.ExecuteToolCall(toolCtx, toolCall)
						log.Printf("[Tool Cache MISS] Executed '%s' - storing result in cache", toolCall.Name)
					}

					// Store in cache for future duplicate calls
					resultCache.Set(signature, &result)
				}

				// Send tool result event
				eventChan <- StreamEvent{Type: StreamEventToolResult, ToolResult: &result}

				// Tool execution complete - errors handled in circuit breaker below

				// CIRCUIT BREAKER: Track failures and duplicate calls using CircuitBreaker struct
				// Uses CircuitBreaker from tool_executor_circuit.go (same as StreamChatWithTools)
				if result.Error != "" {
					circuitBreaker.RecordFailure(signature)
					if circuitBreaker.ShouldBreakOnFailure() {
						// Third CONSECUTIVE failure with same args - warn AI
						log.Printf("[Circuit Breaker - Filtered] Tool '%s' failed 3 times CONSECUTIVELY with identical arguments", toolCall.Name)
						loopWarning := circuitBreaker.GetConsecutiveFailureWarning(toolCall.Name, result.Error)
						currentMessages = append(currentMessages, Message{
							Role:    "system",
							Content: loopWarning,
						})
						circuitBreaker.RecordSuccess() // Reset after warning
					}
				} else {
					circuitBreaker.RecordSuccess()
				}

				// Circuit breaker: track repeated tool calls and warn AI
				totalCount := circuitBreaker.RecordToolCall(signature)

				// Progressive warnings to AI (inject into context so AI sees them)
				var loopWarning string
				if circuitBreaker.ShouldBreak(toolCall.Name, totalCount) {
					// Threshold reached - trigger circuit breaker
					log.Printf("[Circuit Breaker - Filtered] Tool '%s' called %d times - stopping infinite loop", toolCall.Name, totalCount)
					eventChan <- StreamEvent{
						Type:  StreamEventError,
						Error: circuitBreaker.GetCircuitBreakerError(toolCall.Name, totalCount),
					}
					return
				}
				loopWarning = circuitBreaker.GetWarning(toolCall.Name, totalCount)

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

				// GUARDRAIL: Check if adding this tool result would exceed context limit
				// Calculate current context size
				currentContextSize := calculateContextSize(currentMessages)

				// Estimate size of the tool result we're about to add
				var toolResultSize int
				if result.Error != "" {
					toolResultSize = len(result.Error)
				} else {
					toolResultSize = len(fmt.Sprintf("%v", result.Output))
				}

				// Get max context size from config (default 500KB, configurable via MAX_CONTEXT_SIZE env var)
				maxContextBeforeToolResult := config.GetMaxContextSize()

				// If adding this result would exceed limit, replace it with a warning
				if currentContextSize+toolResultSize > maxContextBeforeToolResult {
					log.Printf("[Context Guardrail] Tool result would exceed %s limit (current: %d, result: %d, total: %d). Blocking with guidance message.",
						config.FormatSize(maxContextBeforeToolResult), currentContextSize, toolResultSize, currentContextSize+toolResultSize)

					// Replace the result with a helpful error message
					result.Output = "⚠️ Context is too big, try different approach to reduce command output.\n\n" +
						fmt.Sprintf("The tool '%s' output (%d bytes) would push context over %s limit (current: %d bytes).\n\n",
							toolCall.Name, toolResultSize, config.FormatSize(maxContextBeforeToolResult), currentContextSize) +
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
				// Uses WorkflowState from tool_executor_workflow.go (same as StreamChatWithTools)
				workflowState.UpdateAfterToolExecution(toolCall.Name, result, "Workflow State - Filtered")
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

// NOTE: generateWorkflowStateGuidance has been moved to tool_executor_workflow.go
// as WorkflowState.GetGuidance() method. This function is kept as a stub for backwards
// compatibility but should not be called directly.
// See: tool_executor_workflow.go for the implementation.
