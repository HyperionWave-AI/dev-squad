# Tool Call ID Debugging - Added Diagnostic Logging

**Date:** 2025-11-21
**Status:** 🔍 INVESTIGATION IN PROGRESS
**Issue:** Tool results saved with empty tool_call_id causing API errors

## Problem Statement

When AI processes tool calls across multiple iterations, tool result messages are being saved to MongoDB with **empty `ID` fields** in their `ToolResult` data. When these messages are loaded for subsequent AI requests, the empty IDs become empty `tool_call_id` values in the LLM API request, causing 400 errors:

```
'messages.23' : for 'role:tool' the following must be satisfied
[('messages.23.tool_call_id' : property 'tool_call_id' is missing)]
```

## Evidence

1. **Request Log Analysis** (`logs/19.req.json`):
   - Message 18 shows tool result with `"tool_call_id": ""` (empty)
   - Should have been `"functions.list_directory:5"` to match preceding tool call

2. **Error Pattern**:
   - Errors at messages 18 and 23 in different requests
   - User reported: "tool calls from last iteration are not saved"
   - Happens when loading messages for new AI request (not during initial save)

3. **Session Context**:
   - SessionID: `69202286a612c7f0a7a7c9b8`
   - 50 messages loaded from database
   - Using OpenAI provider with `moonshotai/kimi-k2-instruct-0905` model

## Diagnostic Logging Added

### 1. Tool Call Save Logging
**File:** `hyper/internal/ai-service/executor/stream_executor.go:382-395`

```go
// Before saving tool call
e.logger.Info("💾 Saving tool call to database",
    zap.String("sessionId", e.config.SessionID.Hex()),
    zap.String("toolCallID", event.ToolCall.ID),
    zap.String("toolName", event.ToolCall.Name))

// After successful save
e.logger.Info("✅ Tool call saved successfully",
    zap.String("sessionId", e.config.SessionID.Hex()),
    zap.String("toolCallID", event.ToolCall.ID),
    zap.String("toolName", event.ToolCall.Name))
```

**Purpose**: Confirm tool calls have valid IDs when saved

### 2. Tool Result Save Logging
**File:** `hyper/internal/ai-service/executor/stream_executor.go:434-467`

```go
// Before saving tool result
e.logger.Info("💾 Saving tool result to database",
    zap.String("sessionId", e.config.SessionID.Hex()),
    zap.String("toolResultID", event.ToolResult.ID),
    zap.String("toolName", event.ToolResult.Name),
    zap.Int("outputLength", len(outputStr)),
    zap.String("error", event.ToolResult.Error))

// Check for empty ID
if event.ToolResult.ID == "" {
    e.logger.Error("🚨 BUG DETECTED: ToolResult.ID is EMPTY before SaveToolResult!",
        zap.String("sessionId", e.config.SessionID.Hex()),
        zap.String("toolName", event.ToolResult.Name),
        zap.Int64("durationMs", event.ToolResult.DurationMs))
}

// After successful save
e.logger.Info("✅ Tool result saved successfully",
    zap.String("sessionId", e.config.SessionID.Hex()),
    zap.String("toolResultID", event.ToolResult.ID),
    zap.String("toolName", event.ToolResult.Name))
```

**Purpose**: Detect if ToolResult.ID is empty BEFORE database save

### 3. SaveToolResult Parameter Logging
**File:** `hyper/internal/services/chat_service.go:440-451`

```go
// Check toolCallID parameter
if toolCallID == "" {
    s.logger.Error("🚨 BUG DETECTED: SaveToolResult called with EMPTY toolCallID!",
        zap.String("sessionId", sessionID.Hex()),
        zap.String("toolName", toolName),
        zap.Int64("durationMs", durationMs))
} else {
    s.logger.Info("💾 SaveToolResult preparing to save",
        zap.String("sessionId", sessionID.Hex()),
        zap.String("toolCallID", toolCallID),
        zap.String("toolName", toolName),
        zap.Int64("durationMs", durationMs))
}
```

**Purpose**: Verify the toolCallID parameter passed to SaveToolResult

### 4. Message Retrieval Logging
**File:** `hyper/internal/ai-service/message_converter.go:42-48`

```go
// Check if ID is empty when retrieved from MongoDB
if dbMsg.ToolResult.ID == "" {
    fmt.Printf("🚨 BUG DETECTED: ToolResult.ID is EMPTY when retrieved from MongoDB!\n")
    fmt.Printf("   Tool Name: %s\n", dbMsg.ToolResult.Name)
    fmt.Printf("   Message ID: %s\n", dbMsg.ID.Hex())
    fmt.Printf("   Message Role: %s\n", dbMsg.Role)
    fmt.Printf("   This will cause 'tool_call_id is missing' error in next AI request!\n")
}
```

**Purpose**: Detect if ID is empty when loading messages from database

## Expected Log Output

### Scenario 1: Bug is in ToolResult Event Creation
If the bug is during tool execution, we'll see:
```
🚨 BUG DETECTED: ToolResult.ID is EMPTY before SaveToolResult!
   sessionId: xxxxx
   toolName: list_directory
```

### Scenario 2: Bug is in SaveToolResult Call
If the parameter is empty:
```
🚨 BUG DETECTED: SaveToolResult called with EMPTY toolCallID!
   sessionId: xxxxx
   toolName: list_directory
```

### Scenario 3: Bug is in MongoDB Storage/Retrieval
If ID is saved but lost in database:
```
🚨 BUG DETECTED: ToolResult.ID is EMPTY when retrieved from MongoDB!
   Tool Name: list_directory
   Message ID: xxxxx
```

## Data Flow Trace

1. **Tool Execution**
   - `ToolRegistry.ExecuteToolCall()` creates `ToolResult{ID: toolCall.ID}`
   - `toolCall.ID` comes from LLM response via LangChain

2. **Event Stream**
   - `StreamEvent{Type: StreamEventToolResult, ToolResult: &result}`
   - `stream_executor.go:434` logs `event.ToolResult.ID`

3. **Database Save**
   - `stream_executor.go:452` calls `SaveToolResult(event.ToolResult.ID, ...)`
   - `chat_service.go:446` logs the `toolCallID` parameter
   - MongoDB insert with `ToolResult.ID = toolCallID`

4. **Database Retrieve**
   - `GetSessionMessages()` loads all fields (no projection filter)
   - `message_converter.go:42` checks if `dbMsg.ToolResult.ID == ""`
   - Copies to `langchainMsg.ToolResult.ID`

5. **API Request**
   - `provider.go:330` uses `msg.ToolResult.ID` as `ToolCallID`
   - Sent to LLM API - fails if empty

## Next Steps

1. **Reproduce the Error**:
   - Run a chat session that triggers multiple tool calls
   - Watch logs for 🚨 BUG DETECTED messages

2. **Identify the Source**:
   - Check which logging statement fires first
   - This will pinpoint where the ID becomes empty

3. **Fix Based on Findings**:
   - If during event creation → Fix tool result event generation
   - If during save → Fix parameter passing
   - If during retrieval → Fix MongoDB query/BSON mapping

4. **Verify Fix**:
   - Ensure all tool results have valid IDs
   - Test multi-iteration tool call scenarios
   - Confirm no more "tool_call_id is missing" errors

## Files Modified

1. `hyper/internal/ai-service/executor/stream_executor.go`
   - Added logging before/after tool call save (lines 382-395)
   - Added logging before/after tool result save (lines 434-467)
   - Added empty ID detection before save (lines 442-447)

2. `hyper/internal/services/chat_service.go`
   - Added logging in SaveToolResult (lines 440-451)
   - Detects empty toolCallID parameter

3. `hyper/internal/ai-service/message_converter.go`
   - Added logging when converting DB messages (lines 42-48)
   - Detects empty ID after MongoDB retrieval

## Related Files

- **Tool Execution**: `hyper/internal/ai-service/tool_registry.go`
- **Tool Executor**: `hyper/internal/ai-service/tool_executor.go`
- **Provider**: `hyper/internal/ai-service/provider.go`
- **Models**: `hyper/internal/models/chat.go`

## User Report

User stated: "tool calls from last iteration are not saved and then follow up call results in error"

This suggests the issue happens when:
1. First iteration: Tool calls execute successfully
2. Tool results saved to database (with empty ID?)
3. Second iteration: Load messages from database
4. Empty tool_call_id causes API error

The diagnostic logging will confirm this hypothesis.
