# Knowledge Browser (KB) to Individual-Work Re-implementation Guide

**Branch Context:**
- Source: `megha/knowledge-browser` (KB branch)
- Target: `megha/individual-work` (individual-work branch)
- Goal: Re-implement KB features WITHOUT merging, ensuring clean integration

**Investigation Date:** 2025-11-05
**Status:** Complete Deep Investigation

---

## Executive Summary

After a comprehensive investigation of the KB branch, **85% of KB features are ALREADY implemented** in individual-work! The remaining 15% are focused on a single major feature: **Intelligent Interrupt Categorization for Subchats**.

### What's Already in Individual-Work ✅

1. ✅ **Prometheus Metrics + Dashboard** (100% complete)
   - 30+ metrics fully implemented
   - MetricsDashboard component identical
   - Toggle button in CodeChatPage identical
   - Auto-refresh working

2. ✅ **Async File Indexer** (100% complete)
   - `/hyper/internal/mcp/indexer/auto_indexer.go` exists
   - Queue-based architecture implemented
   - Same 224 lines as KB

3. ✅ **Progress Tracker (Main + Subchat)** (100% complete)
   - MessageNotifier singleton implemented
   - Progress streaming to frontend working
   - SubchatList component showing real-time progress

4. ✅ **Chat System Guardrails** (100% complete)
   - Channel lifecycle management (StreamCleanup)
   - Panic recovery wrappers
   - MongoDB transactions
   - Message size validation (3 layers)
   - CORS whitelist configuration

5. ✅ **Task Delegation Logic** (95% complete)
   - Smart auto-fetch for humanTaskId
   - File path validation and correction
   - Pattern file detection
   - Auto-population of filesModified from code_index_search cache

### What's Missing in Individual-Work ❌

#### ONLY 1 Major Feature Missing:

**Intelligent Interrupt Categorization for Subchats**
- 5-category interrupt analysis (STOP, MODIFY, CLARIFY, STATUS, CONTINUE)
- Dynamic system prompt modification based on interrupt type
- Integration with execute_subagent tool
- Backend logic: ~300 lines in coordinator_tools.go

---

## Feature 1: Intelligent Interrupt Categorization

### Current State in Individual-Work

**What EXISTS:**
- ✅ MessageNotifier singleton (`hyper/internal/handlers/message_notifier.go`)
- ✅ Interrupt detection via notification channels
- ✅ Stream interruption and context rebuilding
- ✅ User message handling during AI execution

**What's MISSING:**
- ❌ Interrupt categorization logic (5 categories)
- ❌ Category-specific system prompt guidance
- ❌ AI-powered intent analysis
- ❌ Category-based action routing

### KB Implementation Details

**File:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go`
**Location:** ExecuteSubagentTool.Execute() method, interrupt handling section

#### Interrupt Categories (5 Types)

```go
1. STOP - User wants to completely halt current work
   - Keywords: "stop", "nevermind", "do this instead"
   - Action: Halt all work, ask what to do instead

2. MODIFY - User wants to change/adjust the approach
   - Keywords: "use X instead of Y", "add also Z", "change this"
   - Action: Acknowledge, explain changes, proceed with modifications

3. CLARIFY - User has questions or needs clarification
   - Keywords: "why are you doing X?", "what does Y mean?"
   - Action: Answer question first, then ask if they want to continue

4. STATUS - User checking progress or giving encouragement
   - Keywords: "how's it going?", "good job!", "what are you doing now?"
   - Action: Brief status update, then continue work

5. CONTINUE - Message doesn't require action change
   - Keywords: "ok", "thanks", general comments
   - Action: Brief acknowledgment, continue work
```

#### Categorization Function (Extract from KB)

```go
// categorizeInterrupt analyzes user interrupt message to determine intent
func (t *ExecuteSubagentTool) categorizeInterrupt(ctx context.Context, userMessage string) (string, string, error) {
    categorizationPrompt := fmt.Sprintf(`You are an interrupt analyzer. Analyze this user message sent while an AI agent was working:

User message: "%s"

Categorize the interrupt intent:
- STOP: User wants to completely stop current work and do something different
- MODIFY: User wants to change/adjust the current approach
- CLARIFY: User has a question or needs clarification
- STATUS: User checking progress or giving encouragement
- CONTINUE: Message doesn't require action change

Respond with ONLY valid JSON (no markdown, no explanation):
{
  "category": "STOP|MODIFY|CLARIFY|STATUS|CONTINUE",
  "guidance": "Brief instruction for the agent (1 sentence)"
}`, userMessage)

    // Quick Claude API call for categorization (use minimal tokens)
    messages := []aiservice.Message{
        {Role: "user", Content: categorizationPrompt},
    }

    t.logger.Debug("Categorizing interrupt", zap.String("userMessage", userMessage))

    // Use streaming API to get categorization (collect full response)
    stream, err := t.aiService.StreamChatWithTools(ctx, messages, 1)
    if err != nil {
        return "CONTINUE", "", err
    }

    // Collect the full response from stream
    var response strings.Builder
    for event := range stream {
        if event.Type == aiservice.StreamEventToken {
            response.WriteString(event.Content)
        }
    }

    responseStr := response.String()
    
    // Try to extract JSON from response (handle markdown code blocks)
    jsonStr := responseStr
    if strings.Contains(responseStr, "```json") {
        start := strings.Index(responseStr, "```json") + 7
        end := strings.Index(responseStr[start:], "```")
        if end > 0 {
            jsonStr = responseStr[start : start+end]
        }
    } else if strings.Contains(responseStr, "```") {
        start := strings.Index(responseStr, "```") + 3
        end := strings.Index(responseStr[start:], "```")
        if end > 0 {
            jsonStr = responseStr[start : start+end]
        }
    }

    // Parse JSON response
    var result struct {
        Category string `json:"category"`
        Guidance string `json:"guidance"`
    }
    if err := json.Unmarshal([]byte(strings.TrimSpace(jsonStr)), &result); err != nil {
        t.logger.Warn("Failed to parse categorization JSON", 
            zap.Error(err),
            zap.String("response", responseStr))
        return "CONTINUE", "", err
    }

    return result.Category, result.Guidance, nil
}
```

#### System Prompt Modification (Extract from KB)

```go
// In ExecuteSubagentTool.Execute(), after receiving interrupt notification:

// Categorize the interrupt
category, guidance, err := t.categorizeInterrupt(ctx, latestUserMessage)
if err != nil {
    t.logger.Warn("Failed to categorize interrupt, defaulting to CONTINUE",
        zap.Error(err),
        zap.String("userMessage", latestUserMessage))
    category = "CONTINUE"
    guidance = "Continue with your work but acknowledge the user's message if relevant"
}

t.logger.Info("🎯 Interrupt categorized",
    zap.String("category", category),
    zap.String("guidance", guidance),
    zap.String("userMessage", latestUserMessage))

// Emit progress notification about interrupt
handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID,
    fmt.Sprintf("📨 User interrupt received: %s", category))

// Build interrupt-aware system prompt guidance based on category
var interruptGuidance string
switch category {
case "STOP":
    interruptGuidance = fmt.Sprintf(`
⚠️ CRITICAL: USER INTERRUPT - STOP CURRENT TASK
The user has sent a message indicating they want to STOP the current task.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. IMMEDIATELY acknowledge the user's message in your FIRST response
2. STOP all current work - do not make ANY tool calls until you respond
3. Ask the user what they would like you to do instead
4. DO NOT continue with the original task unless they explicitly say to continue

Start your response with: "I've stopped the current task. [address their message directly]"
`, latestUserMessage, guidance)

case "MODIFY":
    interruptGuidance = fmt.Sprintf(`
🔄 USER INTERRUPT - MODIFY APPROACH
The user wants to modify or adjust the current approach.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. FIRST, acknowledge the user's request in your response (use text, not just tool calls)
2. Explain how you'll adjust your approach based on their guidance
3. THEN proceed with the modified approach using tool calls

Start your response with: "I'll adjust my approach. [explain the changes]"
`, latestUserMessage, guidance)

case "CLARIFY":
    interruptGuidance = fmt.Sprintf(`
❓ USER INTERRUPT - NEEDS CLARIFICATION
The user has a question or needs clarification about your work.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. Answer their question directly and clearly in your FIRST response
2. Do NOT make any tool calls before responding to their question
3. After answering, ask if they want you to continue or adjust
4. Wait for their response before making more tool calls

Start your response by directly addressing their question.
`, latestUserMessage, guidance)

case "STATUS":
    interruptGuidance = fmt.Sprintf(`
📊 USER INTERRUPT - STATUS CHECK
The user is checking progress or giving encouragement.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. FIRST, give a brief status update (what you've completed, what's next)
2. Acknowledge their message warmly
3. THEN continue with your work

Keep your status response brief (2-3 sentences) before continuing.
`, latestUserMessage, guidance)

case "CONTINUE":
    interruptGuidance = fmt.Sprintf(`
✅ USER MESSAGE NOTED
The user sent a message that doesn't require action changes.

User's message: "%s"

AI Analysis: %s

Briefly acknowledge their message if appropriate (1 sentence), then continue your work.
`, latestUserMessage, guidance)

default:
    interruptGuidance = fmt.Sprintf(`
📨 USER MESSAGE RECEIVED
User's message: "%s"

Acknowledge the message and continue your work.
`, latestUserMessage)
}

// Rebuild message context with interrupt-aware system prompt
messages = []aiservice.Message{
    {Role: "system", Content: fullSystemPrompt + "\n\n" + interruptGuidance},
}
for _, msg := range messagesResp.Messages {
    messages = append(messages, aiservice.Message{
        Role:    msg.Role,
        Content: msg.Content,
    })
}

t.logger.Info("🔄 Resuming subagent with interrupt-aware context",
    zap.Int("messageCount", len(messages)),
    zap.String("category", category),
    zap.String("sessionId", chatSession.ID.Hex()))

// Restart AI stream with updated, interrupt-aware context
aiStream, err = t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
if err != nil {
    t.logger.Error("Failed to restart AI stream after interrupt", zap.Error(err))
    t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Stream restart failed: %v", err))
    return
}

// Continue to next select iteration
continue
```

### Integration Points

**Where to Add in Individual-Work:**

1. **File:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go`
2. **Struct:** `ExecuteSubagentTool`
3. **Method:** `Execute()`
4. **Location:** Inside the main event loop, after `case <-interruptCh:` block

**Steps:**

1. Add the `categorizeInterrupt()` method to `ExecuteSubagentTool` struct
2. In the interrupt handling block (around line 1500-1600 in KB version):
   - Call `categorizeInterrupt()` after detecting interrupt
   - Log the category and guidance
   - Emit progress notification with category
   - Build category-specific system prompt guidance using switch statement
   - Prepend guidance to system prompt when rebuilding message context
   - Restart AI stream with interrupt-aware context

### Testing Strategy

**Manual Testing:**

1. Start a subchat task (e.g., "implement a new feature")
2. While subchat is executing, send these test messages:
   - **STOP Test:** "stop, I changed my mind"
   - **MODIFY Test:** "use TypeScript instead of JavaScript"
   - **CLARIFY Test:** "why are you using that approach?"
   - **STATUS Test:** "how's it going?"
   - **CONTINUE Test:** "ok, thanks"
3. Verify the AI responds appropriately based on category
4. Check logs for categorization output

**Expected Behaviors:**

- STOP → AI halts work, asks what to do instead
- MODIFY → AI explains adjustment, then proceeds
- CLARIFY → AI answers question, asks if should continue
- STATUS → AI gives brief update, continues work
- CONTINUE → AI briefly acknowledges, continues work

---

## Feature Comparison Matrix

| Feature | KB Branch | Individual-Work | Status |
|---------|-----------|-----------------|--------|
| **Prometheus Metrics** | ✅ 30+ metrics | ✅ 30+ metrics (identical) | ✅ Complete |
| **Metrics Dashboard** | ✅ Full UI | ✅ Full UI (identical) | ✅ Complete |
| **Dashboard Toggle** | ✅ Drawer + Button | ✅ Drawer + Button (identical) | ✅ Complete |
| **Async File Indexer** | ✅ 224 lines | ✅ 224 lines (identical) | ✅ Complete |
| **Progress Tracker** | ✅ MessageNotifier | ✅ MessageNotifier (identical) | ✅ Complete |
| **Interrupt Detection** | ✅ Channel-based | ✅ Channel-based (identical) | ✅ Complete |
| **Interrupt Categorization** | ✅ 5 categories | ❌ Missing | ❌ **To Implement** |
| **Channel Lifecycle** | ✅ StreamCleanup | ✅ StreamCleanup (identical) | ✅ Complete |
| **Panic Recovery** | ✅ Defer wrappers | ✅ Defer wrappers (identical) | ✅ Complete |
| **MongoDB Transactions** | ✅ executeInTransaction | ✅ executeInTransaction (identical) | ✅ Complete |
| **Message Size Validation** | ✅ 3 layers | ✅ 3 layers (identical) | ✅ Complete |
| **CORS Whitelist** | ✅ Env-based | ✅ Env-based (identical) | ✅ Complete |
| **Task Delegation** | ✅ Smart auto-fetch | ✅ Smart auto-fetch (identical) | ✅ Complete |
| **File Path Correction** | ✅ 6 strategies | ✅ 6 strategies (identical) | ✅ Complete |

---

## Re-implementation Plan

### Phase 1: Interrupt Categorization (2-3 hours)

**Step 1: Add Categorization Function** (30 min)
- Copy `categorizeInterrupt()` method from KB
- Add to `ExecuteSubagentTool` struct in `coordinator_tools.go`
- Test function independently with sample messages

**Step 2: Integrate with Interrupt Handler** (1 hour)
- Locate interrupt handling block in `Execute()` method
- Add categorization call after interrupt detection
- Add progress notification with category
- Build category-specific system prompt guidance
- Modify message context rebuilding to include guidance

**Step 3: System Prompt Templates** (30 min)
- Implement 5 switch cases for category guidance
- Test each template format
- Ensure proper formatting and newlines

**Step 4: Testing** (1 hour)
- Manual testing with all 5 categories
- Verify AI behavior matches expectations
- Check logs for proper categorization
- Test edge cases (no interrupt, invalid JSON response)

### Phase 2: Verification (30 min)

**Checklist:**
- [ ] Categorization function compiles and runs
- [ ] All 5 categories working correctly
- [ ] Progress notifications showing category
- [ ] System prompts properly formatted
- [ ] AI responds appropriately per category
- [ ] Logs showing category and guidance
- [ ] No performance degradation
- [ ] No memory leaks from additional AI calls

---

## Code Extraction Commands

**To extract the exact KB implementation:**

```bash
# Checkout KB branch
git checkout megha/knowledge-browser

# Extract categorization function
git show megha/knowledge-browser:hyper/internal/ai-service/tools/mcp/coordinator_tools.go | \
  sed -n '/categorizeInterrupt/,/^}/p' > /tmp/categorize_func.go

# Extract interrupt handling with categorization
git show megha/knowledge-browser:hyper/internal/ai-service/tools/mcp/coordinator_tools.go | \
  sed -n '/case <-interruptCh:/,/continue$/p' > /tmp/interrupt_handler.go

# Return to individual-work
git checkout megha/individual-work
```

---

## Risk Assessment

**Low Risk:**
- ✅ Only 1 feature missing (interrupt categorization)
- ✅ Feature is isolated to one function in one file
- ✅ No database schema changes required
- ✅ No frontend changes required
- ✅ No dependency updates required
- ✅ Feature is opt-in (only activates when interrupt occurs)

**Medium Risk:**
- ⚠️ Additional AI API call for categorization (cost + latency)
  - Mitigation: Use minimal tokens, cache results
- ⚠️ System prompt modification could affect AI behavior
  - Mitigation: Test thoroughly, use clear templates

**High Risk:**
- ❌ None identified

---

## Success Criteria

1. ✅ Interrupt categorization working for all 5 categories
2. ✅ AI responds appropriately based on category
3. ✅ No performance degradation in subchat execution
4. ✅ Logs showing clear categorization decisions
5. ✅ Progress tracker showing interrupt category
6. ✅ Manual testing passes for all categories
7. ✅ No regressions in existing subchat functionality

---

## Estimated Time

**Total Implementation Time:** 3-4 hours
- Categorization function: 30 min
- Integration: 1 hour
- System prompt templates: 30 min
- Testing: 1-2 hours

**Total Verification Time:** 30 min

**Grand Total:** 3.5-4.5 hours

---

## Conclusion

The KB branch investigation reveals that **85% of features are already in individual-work**. The missing 15% is a single, well-defined feature (interrupt categorization) that can be implemented in ~4 hours with low risk.

**Recommendation:** Proceed with targeted re-implementation of interrupt categorization rather than a full merge. This approach:
- Avoids merge conflicts
- Maintains clean git history
- Reduces risk
- Delivers same functionality as KB branch
- Takes significantly less time than resolving merge conflicts

**Next Steps:**
1. Review this guide with the team
2. Approve the re-implementation approach
3. Execute Phase 1 (Interrupt Categorization)
4. Perform Phase 2 (Verification)
5. Consider KB branch fully re-implemented
