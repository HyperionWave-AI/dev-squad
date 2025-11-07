# Tool Result Bug - Quick Reference

**Status:** 🚨 CRITICAL BUG IDENTIFIED - DO NOT MERGE CODE WITH THIS PATTERN

---

## 🚨 The Bug in 30 Seconds

**Location:** `langchain_service.go:1440-1443`

**Problem:** Tool results added with wrong role and wrong format

**Impact:** AI providers (Anthropic/OpenAI) don't receive tool results properly, causing infinite loops

---

## ❌ FORBIDDEN PATTERN (Current Bug)

```go
// Line 1360-1390: Convert ToolResult to string
var toolResultMsg string
outputJSON, _ := json.Marshal(result.Output)
toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))

// Line 1440-1443: ❌ Add as system message - WRONG!
currentMessages = append(currentMessages, Message{
    Role:    "system",        // ❌ Wrong role
    Content: toolResultMsg,   // ❌ String, not structured data
})
```

### Why This Is Wrong:
1. Uses `Role: "system"` instead of `Role: "tool_result"`
2. Converts structured data to string
3. Doesn't populate `ToolResult` struct field
4. Anthropic provider skips system messages
5. AI never sees tool execution results
6. Causes infinite retry loops

---

## ✅ CORRECT PATTERN (Required Fix)

```go
// Add tool_call message (assistant acknowledging the call)
currentMessages = append(currentMessages, Message{
    Role:    "tool_call",
    Content: responseText,  // AI's text before making the call
    ToolCall: &ToolCall{
        ID:   toolCall.ID,
        Name: toolCall.Name,
        Args: toolCall.Args,
    },
})

// Add tool_result message (the actual result)
currentMessages = append(currentMessages, Message{
    Role:    "tool_result",  // ✅ Correct role
    Content: "",              // ✅ Empty - data is in ToolResult
    ToolResult: &ToolResult{  // ✅ Structured data
        ID:         toolCall.ID,
        Name:       toolCall.Name,
        Output:     result.Output,      // Keep as interface{}
        Error:      result.Error,
        DurationMs: result.DurationMs,
    },
})
```

### Why This Is Correct:
1. Uses `Role: "tool_result"` (expected by providers)
2. Keeps data structured (no JSON stringification)
3. Populates `ToolResult` struct properly
4. Anthropic provider recognizes and formats correctly
5. AI sees tool results immediately
6. Workflow continues smoothly

---

## 📍 File Locations

### Bug Location
- **File:** `/Users/maxmednikov/MaxSpace/hyper/hyper/internal/ai-service/langchain_service.go`
- **Lines:** 1360-1443 (main processing path)
- **Function:** `StreamChatWithTools()`

### Correct Implementation Reference
- **File:** `/Users/maxmednikov/MaxSpace/hyper/hyper/internal/ai-service/langchain_service.go`
- **Lines:** 1987-2002 (filtered processing path)
- **Function:** `StreamChatWithTools()`

### Provider Handling
- **Anthropic:** `provider.go:457-492` (expects `tool_result` role)
- **OpenAI:** `provider.go:160-176` (needs update to handle `tool_result`)

---

## 🔍 How to Detect the Bug

### In Code Review
```bash
# Search for the buggy pattern
grep -n 'Role.*"system"' langchain_service.go | grep -i tool

# Look for string conversion of tool results
grep -n 'toolResultMsg.*fmt.Sprintf' langchain_service.go
```

### In Logs
```bash
# Look for system messages with tool results
grep "Role.*system.*Tool.*result" logs.txt

# Should see tool_result messages instead
grep "Role.*tool_result" logs.txt
```

### In Runtime Behavior
**Symptoms:**
- AI requests a tool
- Tool executes successfully
- AI immediately requests the same tool again (infinite loop)
- Circuit breaker triggers after N attempts
- AI claims "tool doesn't exist" or "failed" when it succeeded

---

## 🧪 Testing the Fix

### Before Fix
```go
// Test will fail - tool result not properly delivered
func TestToolResultDelivery(t *testing.T) {
    // ... setup ...

    // Execute tool
    result := registry.ExecuteToolCall(ctx, toolCall)

    // Check message history
    lastMsg := messages[len(messages)-1]

    // ❌ Currently fails
    assert.Equal(t, "tool_result", lastMsg.Role)
    assert.NotNil(t, lastMsg.ToolResult)
}
```

### After Fix
```go
// Test will pass - tool result properly delivered
func TestToolResultDelivery(t *testing.T) {
    // ... setup ...

    // Execute tool
    result := registry.ExecuteToolCall(ctx, toolCall)

    // Check message history
    lastMsg := messages[len(messages)-1]

    // ✅ Now passes
    assert.Equal(t, "tool_result", lastMsg.Role)
    assert.NotNil(t, lastMsg.ToolResult)
    assert.Equal(t, "call_123", lastMsg.ToolResult.ID)
}
```

---

## 📋 Checklist for Developers

### Before Committing Any Tool Result Code:

- [ ] Tool results use `Role: "tool_result"`
- [ ] ToolResult struct is populated, not just Content string
- [ ] Tool call ID is preserved in ToolResult.ID
- [ ] No JSON stringification of Output field (keep as interface{})
- [ ] Provider tests verify correct message format
- [ ] Integration tests confirm AI sees tool results
- [ ] No infinite retry loops in manual testing

---

## 🎯 Impact Summary

| Component | Bug Impact | After Fix |
|-----------|-----------|-----------|
| **Anthropic API** | Tool results missing from messages | Properly formatted tool_result blocks |
| **OpenAI API** | Tool results as system prompts (confusing) | Proper tool_result handling (needs implementation) |
| **AI Behavior** | Infinite retry loops, hallucinations | Smooth workflow, sees all results |
| **User Experience** | Tasks fail, frustration | Tasks complete successfully |
| **Circuit Breaker** | Triggers frequently (hiding bug) | Rarely needed (workflow is smooth) |

---

## 🔗 Related Documents

- **Full Analysis:** `TOOL_RESULT_ANALYSIS.md`
- **Flow Diagrams:** `TOOL_RESULT_FLOW_DIAGRAM.md`
- **MCP Standards:** `/Users/maxmednikov/MaxSpace/Hyperion/.claude/schema-standards.md`

---

## ⚡ Emergency Fix (If You Need It Now)

**Minimal change to fix the immediate bug:**

**File:** `langchain_service.go`
**Line:** 1440

**Replace:**
```go
currentMessages = append(currentMessages, Message{
    Role:    "system",
    Content: toolResultMsg,
})
```

**With:**
```go
currentMessages = append(currentMessages, Message{
    Role:    "tool_call",
    Content: responseText,
    ToolCall: &ToolCall{ID: toolCall.ID, Name: toolCall.Name, Args: toolCall.Args},
})
currentMessages = append(currentMessages, Message{
    Role:    "tool_result",
    Content: "",
    ToolResult: &ToolResult{
        ID:     toolCall.ID,
        Name:   toolCall.Name,
        Output: result.Output,
        Error:  result.Error,
    },
})
```

**Then:**
1. Remove toolResultMsg construction (lines 1360-1438)
2. Run tests: `go test ./internal/ai-service -v`
3. Manual test with Anthropic API
4. Verify no infinite loops

---

**Version:** 1.0
**Date:** 2025-01-25
**Status:** Ready for Implementation
