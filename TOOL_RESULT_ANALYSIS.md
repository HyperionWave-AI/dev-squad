# Tool Result Flow Analysis - Critical Bug Identified

**Date:** 2025-01-25
**Status:** 🚨 CRITICAL BUG CONFIRMED
**Impact:** High - Tool results not properly delivered to AI providers

---

## Executive Summary

**BUG CONFIRMED:** The system has **TWO DIFFERENT CODE PATHS** for processing tool results, and they implement **INCOMPATIBLE message formats**:

1. **Path 1 (Lines ~1440):** Uses `Role: "system"` with `Content: toolResultMsg` (string)
2. **Path 2 (Lines ~1990):** Uses `Role: "tool_result"` with `ToolResult: {...}` (structured data)

The Anthropic provider expects `Role: "tool_result"` messages (Path 2), but **Path 1 is being executed** in most cases, resulting in tool results being added as system messages that the provider cannot properly format.

---

## 🔍 Complete Flow Analysis

### 1. Tool Execution Flow

```
User Request
    ↓
StreamChatWithTools() [langchain_service.go:600+]
    ↓
LLM requests tool call
    ↓
Execute tool via toolRegistry.ExecuteToolCall()
    ↓
Get ToolResult{ID, Name, Args, Output, Error, DurationMs}
    ↓
Add result to message history ← 🚨 BUG HAPPENS HERE
    ↓
Send messages to AI provider
    ↓
Provider converts messages to API format
    ↓
API call to Anthropic/OpenAI
```

### 2. Two Code Paths Identified

#### **PATH 1: Main Tool Processing (Lines 1360-1443) - 🚨 BUGGY**

**Location:** `langchain_service.go:1360-1443`

```go
// Line 1360: Create string representation of tool result
var toolResultMsg string
if result.Error != "" {
    toolResultMsg = fmt.Sprintf("Tool '%s' error: %s", result.Name, result.Error)
} else {
    outputJSON, err := json.Marshal(result.Output)
    toolResultMsg = fmt.Sprintf("Tool '%s' result: %s", result.Name, string(outputJSON))
}

// Line 1440: ❌ BUG - Add as SYSTEM message with string content
currentMessages = append(currentMessages, Message{
    Role:    "system",    // ❌ WRONG ROLE
    Content: toolResultMsg, // ❌ String content, not ToolResult struct
})
```

**Issues with this approach:**
- ✗ Uses `Role: "system"` instead of `Role: "tool_result"`
- ✗ Puts structured data in `Content` field as a string
- ✗ Does not populate `ToolResult` struct field
- ✗ Anthropic provider filters out system messages (provider.go:413-416)
- ✗ OpenAI provider converts all roles to basic types, losing tool result semantics

#### **PATH 2: Filtered Processing (Lines 1987-2002) - ✅ CORRECT**

**Location:** `langchain_service.go:1987-2002`

```go
// Line 1987: Add tool_call message first
currentMessages = append(currentMessages, Message{
    Role:    "tool_call",
    Content: responseText,
    ToolCall: &ToolCall{
        ID:   toolCall.ID,
        Name: toolCall.Name,
        Args: toolCall.Args,
    },
})

// Line 1992: ✅ CORRECT - Add tool_result message with proper structure
currentMessages = append(currentMessages, Message{
    Role:    "tool_result",    // ✅ Correct role
    Content: "",                // ✅ Empty content
    ToolResult: &ToolResult{   // ✅ Structured data in ToolResult field
        ID:         toolCall.ID,
        Name:       toolCall.Name,
        Output:     result.Output,
        Error:      result.Error,
        DurationMs: result.DurationMs,
    },
})
```

**Why this is correct:**
- ✓ Uses `Role: "tool_result"` (matches provider expectations)
- ✓ Populates `ToolResult` struct with structured data
- ✓ Anthropic provider recognizes and formats correctly (provider.go:457-492)
- ✓ Maintains tool call ID for request/response correlation

---

## 🎯 Provider Expectations Analysis

### Anthropic Provider (provider.go:457-492)

**Expected Message Format:**
```go
Message{
    Role: "tool_result",
    ToolResult: &ToolResult{
        ID:     "call_xyz123",
        Name:   "tool_name",
        Output: interface{}, // Actual result data
        Error:  "",
    },
}
```

**What it does with this:**
```go
if msg.Role == "tool_result" {
    if msg.ToolResult != nil {
        resultContent := msg.ToolResult.Output
        if msg.ToolResult.Error != "" {
            resultContent = map[string]interface{}{
                "error": msg.ToolResult.Error,
            }
        }

        // Formats as Anthropic API tool_result block
        apiMessages = append(apiMessages, map[string]interface{}{
            "role": "user",
            "content": []map[string]interface{}{
                {
                    "type":       "tool_result",
                    "tool_use_id": sanitizeToolID(msg.ToolResult.ID),
                    "content":    resultStr,
                },
            },
        })
    }
}
```

**Current Bug Impact:**
- System messages with `Role: "system"` are extracted separately (line 413-416)
- They never reach the tool_result processing block
- Anthropic API never receives tool results in proper format
- AI doesn't see tool execution results!

### OpenAI Provider (provider.go:160-176)

**Expected Message Format:**
Currently, OpenAI provider only handles basic roles:

```go
for _, msg := range messages {
    var msgType llms.ChatMessageType
    switch msg.Role {
    case "user":
        msgType = llms.ChatMessageTypeHuman
    case "assistant":
        msgType = llms.ChatMessageTypeAI
    case "system":
        msgType = llms.ChatMessageTypeSystem
    default:
        msgType = llms.ChatMessageTypeHuman
    }
    msgContents = append(msgContents, llms.TextParts(msgType, msg.Content))
}
```

**Current Bug Impact:**
- Tool results added as "system" messages get processed as system prompts
- Tool result data is in string format, not structured
- OpenAI API receives tool results as system messages (confusing)
- No `tool_result` handling implemented for OpenAI

---

## 🚨 Bug Root Cause

### Why Two Code Paths Exist

Looking at the code structure, there appear to be **two different processing modes**:

1. **Main Processing Path** (Lines 993-1600)
   - Default tool execution path
   - Iterates through `response.ToolCalls`
   - Has extensive validation, caching, circuit breakers
   - **Uses buggy system message approach**

2. **Filtered Processing Path** (Lines 1900-2100)
   - Appears to be a separate implementation
   - Has different workflow state tracking
   - **Uses correct tool_result message format**

**The Issue:** The main processing path (used in most cases) was never updated to use the proper `tool_result` message format.

### Evidence from Code

**Path 1 is executed when:**
```go
// Line 993: Process each tool call
for _, toolCall := range response.ToolCalls {
    // ... validation, caching, execution ...

    // Line 1440: BUG - Add as system message
    currentMessages = append(currentMessages, Message{
        Role:    "system",
        Content: toolResultMsg,
    })
}
```

**Path 2 is executed when:**
```go
// Line 1900: Appears to be a separate filtered path
// This path correctly uses tool_result role
```

---

## 📊 Impact Assessment

### Severity: **CRITICAL**

| Impact Area | Severity | Details |
|------------|----------|---------|
| **Anthropic API** | 🔴 High | Tool results never delivered in proper format; AI cannot see tool execution results |
| **OpenAI API** | 🟡 Medium | Tool results delivered as system messages; confusing but may partially work |
| **Tool Execution** | 🟢 Low | Tools execute correctly; only result delivery is broken |
| **User Experience** | 🔴 High | AI appears to ignore tool results and may retry infinitely |

### Symptoms Users Would Experience

1. **Infinite Tool Call Loops**
   - AI requests a tool
   - Tool executes successfully
   - AI doesn't see the result
   - AI requests the same tool again
   - Loop continues until circuit breaker triggers

2. **Hallucinated Responses**
   - AI makes assumptions about tool results
   - Provides incorrect information based on guesses
   - User sees tool execution succeeded but AI response is wrong

3. **Confusion About Tool Availability**
   - AI may claim tools don't exist
   - Or claim tools failed when they succeeded
   - Because it never sees the actual results

---

## ✅ Recommended Fix

### Solution Overview

**Replace Path 1's buggy message format with Path 2's correct format.**

### Code Changes Required

**File:** `langchain_service.go`
**Location:** Lines 1440-1443

#### Current (Buggy) Code:
```go
currentMessages = append(currentMessages, Message{
    Role:    "system",
    Content: toolResultMsg,
})
```

#### Fixed Code:
```go
// Add assistant message with tool_call
currentMessages = append(currentMessages, Message{
    Role:    "tool_call",
    Content: responseText,
    ToolCall: &ToolCall{
        ID:   toolCall.ID,
        Name: toolCall.Name,
        Args: toolCall.Args,
    },
})

// Add tool_result message with proper structure
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
```

### Additional Considerations

1. **Remove toolResultMsg string construction** (Lines 1360-1392)
   - This entire block becomes unnecessary
   - Tool result data stays structured
   - Warnings/truncation should be applied to Output field instead

2. **Update OpenAI Provider** (provider.go:160-176)
   - Add handling for `Role: "tool_result"`
   - Convert ToolResult struct to OpenAI's expected format
   - Similar to Anthropic implementation

3. **Consolidate Code Paths**
   - Consider merging Path 1 and Path 2
   - Remove duplicate workflow state logic
   - Single source of truth for tool result handling

---

## 🧪 Test Plan

### 1. Unit Tests

**File:** `langchain_service_test.go`

```go
func TestToolResultMessageFormat(t *testing.T) {
    // Test that tool results are added with correct role
    // and ToolResult struct is populated
}

func TestAnthropicProviderToolResults(t *testing.T) {
    // Test that Anthropic provider correctly formats
    // tool_result messages for API
}

func TestOpenAIProviderToolResults(t *testing.T) {
    // Test that OpenAI provider handles tool_result
    // messages correctly
}
```

### 2. Integration Tests

**Test Case 1: Single Tool Call**
- AI requests a tool
- Tool executes successfully
- Verify AI sees the result in next prompt
- Verify AI uses result (doesn't retry)

**Test Case 2: Multiple Tool Calls**
- AI requests multiple tools in sequence
- Verify each result is delivered correctly
- Verify conversation flow is coherent

**Test Case 3: Tool Error Handling**
- Tool execution fails
- Verify error is delivered in ToolResult.Error
- Verify AI sees the error and adapts

**Test Case 4: Cross-Provider**
- Test with Anthropic (Claude)
- Test with OpenAI (GPT)
- Verify both receive tool results correctly

### 3. Manual Testing

**Scenario:** Ask AI to perform a multi-step task requiring tool calls

```
User: "Create a human task for implementing feature X,
       then create an agent task to execute it"

Expected Flow:
1. AI calls coordinator_create_human_task
2. AI sees the taskId in result
3. AI calls coordinator_create_agent_task with that taskId
4. AI confirms success to user

Current Bug Behavior:
1. AI calls coordinator_create_human_task
2. AI doesn't see the result
3. AI either:
   - Retries coordinator_create_human_task (infinite loop)
   - Hallucinates a taskId
   - Tells user it failed
```

### 4. Validation Checks

**Before Fix:**
```bash
# Enable debug logging
export LOG_LEVEL=debug

# Run AI chat with tool calls
# Check logs for:
grep "Role.*system.*toolResultMsg" logs.txt
# Should find matches (proving bug exists)
```

**After Fix:**
```bash
# Run same test
# Check logs for:
grep "Role.*tool_result" logs.txt
# Should find matches (proving fix applied)

grep "Role.*system.*toolResultMsg" logs.txt
# Should find NO matches (proving bug removed)
```

---

## 📝 Code Locations Reference

### Primary Bug Location
- **File:** `hyper/internal/ai-service/langchain_service.go`
- **Lines:** 1440-1443 (buggy system message)
- **Function:** `StreamChatWithTools()`

### Correct Implementation Reference
- **File:** `hyper/internal/ai-service/langchain_service.go`
- **Lines:** 1987-2002 (correct tool_result message)
- **Function:** `StreamChatWithTools()` (filtered path)

### Provider Implementations
- **Anthropic:** `hyper/internal/ai-service/provider.go` lines 457-492
- **OpenAI:** `hyper/internal/ai-service/provider.go` lines 160-176

### Message Struct Definition
- **File:** `hyper/internal/ai-service/provider.go`
- **Lines:** 19-31
- **Struct:** `Message`

### ToolResult Struct Definition
- **File:** `hyper/internal/ai-service/tool_registry.go`
- **Lines:** 35-42
- **Struct:** `ToolResult`

---

## 🎬 Next Steps

1. ✅ **Analysis Complete** - This document
2. ⏳ **Review Required** - Get approval for fix approach
3. ⏳ **Implement Fix** - Apply code changes
4. ⏳ **Write Tests** - Comprehensive test coverage
5. ⏳ **Integration Test** - Verify with real AI providers
6. ⏳ **Deploy** - Roll out fix to production

---

## 📚 Related Documentation

- **MCP Tool Standards:** `/Users/maxmednikov/MaxSpace/Hyperion/.claude/schema-standards.md`
- **Provider Configuration:** `hyper/internal/ai-service/provider.go`
- **Tool Registry:** `hyper/internal/ai-service/tool_registry.go`
- **Message Converter:** `hyper/internal/ai-service/message_converter.go`

---

## ⚠️ Warning

**DO NOT merge any code that uses the buggy pattern:**
```go
// ❌ FORBIDDEN PATTERN
currentMessages = append(currentMessages, Message{
    Role:    "system",
    Content: toolResultMsg, // String representation of tool result
})
```

**ONLY use the correct pattern:**
```go
// ✅ REQUIRED PATTERN
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

---

**Analysis Date:** 2025-01-25
**Analyst:** Claude (Coordinator)
**Status:** Ready for Review & Implementation
