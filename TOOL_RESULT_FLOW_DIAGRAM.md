# Tool Result Flow - Visual Diagrams

## Current (Buggy) Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER REQUEST                                 │
│              "Create a human task for feature X"                     │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│              LANGCHAIN SERVICE: StreamChatWithTools()                │
│                                                                       │
│  1. Prepare messages + tools                                         │
│  2. Call AI Provider (Anthropic/OpenAI)                             │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│                    AI PROVIDER RESPONSE                              │
│                                                                       │
│  TextChannel: "I'll create that task for you..."                    │
│  ToolCalls: [{                                                       │
│    ID: "call_123",                                                   │
│    Name: "coordinator_create_human_task",                           │
│    Args: {prompt: "Feature X"}                                      │
│  }]                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│           TOOL EXECUTION (Line 993-1270)                             │
│                                                                       │
│  result = toolRegistry.ExecuteToolCall(toolCall)                     │
│                                                                       │
│  ToolResult{                                                         │
│    ID: "call_123",                                                   │
│    Name: "coordinator_create_human_task",                           │
│    Output: {taskId: "task_abc123", ...},                            │
│    Error: "",                                                        │
│    DurationMs: 45                                                    │
│  }                                                                   │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│      🚨 BUG LOCATION: ADD RESULT TO MESSAGE HISTORY (Line 1360-1443) │
│                                                                       │
│  // Convert ToolResult to string                                     │
│  outputJSON = marshal(result.Output)                                 │
│  toolResultMsg = "Tool 'coordinator_create_human_task' result: {...}"│
│                                                                       │
│  // ❌ Add as SYSTEM message with string content                    │
│  currentMessages.append(Message{                                     │
│    Role: "system",          ← ❌ WRONG ROLE                          │
│    Content: toolResultMsg,  ← ❌ String, not ToolResult struct       │
│  })                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│       NEXT AI ITERATION: Convert Messages to Provider Format         │
│                     (provider.go:400-550)                            │
│                                                                       │
│  For Anthropic Provider:                                             │
│  ┌─────────────────────────────────────────────────────┐            │
│  │ System Messages (Line 413-416)                      │            │
│  │ ─────────────────────────                           │            │
│  │ if msg.Role == "system" {                           │            │
│  │   systemPrompt = msg.Content                        │            │
│  │   continue  ← ❌ SKIPS MESSAGE, doesn't process it  │            │
│  │ }                                                    │            │
│  └─────────────────────────────────────────────────────┘            │
│                                                                       │
│  ┌─────────────────────────────────────────────────────┐            │
│  │ Tool Result Messages (Line 457-492)                 │            │
│  │ ───────────────────────────                         │            │
│  │ if msg.Role == "tool_result" {                      │            │
│  │   // Format for Anthropic API                       │            │
│  │   // ❌ NEVER REACHED - no messages have this role! │            │
│  │ }                                                    │            │
│  └─────────────────────────────────────────────────────┘            │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│              ANTHROPIC API REQUEST (Missing Tool Results!)           │
│                                                                       │
│  {                                                                   │
│    "model": "claude-sonnet-4",                                       │
│    "messages": [                                                     │
│      {role: "user", content: "Create a human task for feature X"},  │
│      {role: "assistant", content: [{                                 │
│        type: "tool_use",                                             │
│        id: "call_123",                                               │
│        name: "coordinator_create_human_task",                       │
│        input: {prompt: "Feature X"}                                 │
│      }]},                                                            │
│      ❌ MISSING: Tool result for call_123!                          │
│    ],                                                                │
│    "system": "You are AI... Tool 'coordinator...' result: {...}"    │
│  }                                                                   │
│                                                                       │
│  ❌ API ERROR: "Missing tool result for tool_use_id: call_123"      │
│  OR AI doesn't see the result and retries the same tool!            │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
                    ♾️  INFINITE LOOP
```

---

## Correct Flow (After Fix)

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER REQUEST                                 │
│              "Create a human task for feature X"                     │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│              LANGCHAIN SERVICE: StreamChatWithTools()                │
│                                                                       │
│  1. Prepare messages + tools                                         │
│  2. Call AI Provider (Anthropic/OpenAI)                             │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│                    AI PROVIDER RESPONSE                              │
│                                                                       │
│  TextChannel: "I'll create that task for you..."                    │
│  ToolCalls: [{                                                       │
│    ID: "call_123",                                                   │
│    Name: "coordinator_create_human_task",                           │
│    Args: {prompt: "Feature X"}                                      │
│  }]                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│           TOOL EXECUTION (Line 993-1270)                             │
│                                                                       │
│  result = toolRegistry.ExecuteToolCall(toolCall)                     │
│                                                                       │
│  ToolResult{                                                         │
│    ID: "call_123",                                                   │
│    Name: "coordinator_create_human_task",                           │
│    Output: {taskId: "task_abc123", ...},                            │
│    Error: "",                                                        │
│    DurationMs: 45                                                    │
│  }                                                                   │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│      ✅ FIX: ADD RESULT TO MESSAGE HISTORY (Lines 1440-1460)        │
│                                                                       │
│  // Add assistant message with tool_call                             │
│  currentMessages.append(Message{                                     │
│    Role: "tool_call",                                                │
│    Content: responseText,                                            │
│    ToolCall: &ToolCall{                                              │
│      ID: "call_123",                                                 │
│      Name: "coordinator_create_human_task",                         │
│      Args: {prompt: "Feature X"}                                    │
│    }                                                                 │
│  })                                                                  │
│                                                                       │
│  // ✅ Add tool_result message with structured data                 │
│  currentMessages.append(Message{                                     │
│    Role: "tool_result",     ← ✅ CORRECT ROLE                        │
│    Content: "",                                                      │
│    ToolResult: &ToolResult{ ← ✅ Structured data                     │
│      ID: "call_123",                                                 │
│      Name: "coordinator_create_human_task",                         │
│      Output: {taskId: "task_abc123", ...},                          │
│      Error: "",                                                      │
│      DurationMs: 45                                                  │
│    }                                                                 │
│  })                                                                  │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│       NEXT AI ITERATION: Convert Messages to Provider Format         │
│                     (provider.go:400-550)                            │
│                                                                       │
│  For Anthropic Provider:                                             │
│  ┌─────────────────────────────────────────────────────┐            │
│  │ Tool Result Messages (Line 457-492)                 │            │
│  │ ───────────────────────────                         │            │
│  │ if msg.Role == "tool_result" {                      │            │
│  │   if msg.ToolResult != nil {                        │            │
│  │     // Extract structured data                      │            │
│  │     resultContent = msg.ToolResult.Output           │            │
│  │     if msg.ToolResult.Error != "" {                 │            │
│  │       resultContent = {error: msg.ToolResult.Error} │            │
│  │     }                                                │            │
│  │                                                      │            │
│  │     // Format for Anthropic API                     │            │
│  │     apiMessages.append({                            │            │
│  │       role: "user",                                 │            │
│  │       content: [{                                   │            │
│  │         type: "tool_result",                        │            │
│  │         tool_use_id: "call_123",                    │            │
│  │         content: resultStr                          │            │
│  │       }]                                             │            │
│  │     })                                               │            │
│  │   }                                                  │            │
│  │ }                                                    │            │
│  └─────────────────────────────────────────────────────┘            │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│              ANTHROPIC API REQUEST (Complete with Tool Results!)     │
│                                                                       │
│  {                                                                   │
│    "model": "claude-sonnet-4",                                       │
│    "messages": [                                                     │
│      {role: "user", content: "Create a human task for feature X"},  │
│      {role: "assistant", content: [{                                 │
│        type: "tool_use",                                             │
│        id: "call_123",                                               │
│        name: "coordinator_create_human_task",                       │
│        input: {prompt: "Feature X"}                                 │
│      }]},                                                            │
│      {role: "user", content: [{                                      │
│        type: "tool_result",                                          │
│        tool_use_id: "call_123",                                      │
│        content: '{"taskId":"task_abc123",...}'                       │
│      }]}  ← ✅ Tool result properly included!                       │
│    ],                                                                │
│    "system": "You are AI..."                                         │
│  }                                                                   │
│                                                                       │
│  ✅ API SUCCESS: AI sees tool result and continues workflow          │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────────────┐
│                      AI NEXT RESPONSE                                │
│                                                                       │
│  "Great! I've created task task_abc123 for Feature X.               │
│   Now I'll create an agent task to execute it..."                   │
│                                                                       │
│  [Calls coordinator_create_agent_task with humanTaskId=task_abc123] │
└─────────────────────────────────────────────────────────────────────┘
                              ↓
                    ✅ WORKFLOW COMPLETES
```

---

## Side-by-Side Comparison

### Message History Before Provider Conversion

#### Buggy Version (Path 1)
```
currentMessages = [
  {Role: "user", Content: "Create a human task for feature X"},
  {Role: "assistant", Content: "I'll create that..."},
  {Role: "system", Content: "Tool 'coordinator_create_human_task' result: {\"taskId\":\"task_abc123\"}"}
  ❌ ToolResult field is nil/empty
]
```

#### Correct Version (Path 2)
```
currentMessages = [
  {Role: "user", Content: "Create a human task for feature X"},
  {Role: "tool_call", Content: "I'll create that...", ToolCall: {...}},
  {Role: "tool_result", Content: "", ToolResult: {
    ID: "call_123",
    Output: {taskId: "task_abc123"},
    Error: ""
  }}
  ✅ ToolResult field properly populated
]
```

---

## Anthropic API Format Requirements

### Tool Use Request
```json
{
  "role": "assistant",
  "content": [
    {
      "type": "tool_use",
      "id": "toolu_01A09q90qw90lq917835lq9",
      "name": "coordinator_create_human_task",
      "input": {"prompt": "Feature X"}
    }
  ]
}
```

### Tool Result Response (MUST follow immediately)
```json
{
  "role": "user",
  "content": [
    {
      "type": "tool_result",
      "tool_use_id": "toolu_01A09q90qw90lq917835lq9",
      "content": "{\"taskId\": \"task_abc123\", \"status\": \"created\"}"
    }
  ]
}
```

### ❌ What happens with current bug
```json
{
  "messages": [
    {
      "role": "assistant",
      "content": [{"type": "tool_use", "id": "call_123", ...}]
    }
    // ❌ MISSING: Tool result message
  ],
  "system": "You are... Tool 'coordinator_create_human_task' result: {...}"
  // ❌ Tool result buried in system prompt as text!
}
```

**Result:** Anthropic API either:
1. Returns error: "Missing tool result for tool_use_id"
2. Ignores the tool call and generates text response
3. AI doesn't see result, retries same tool (infinite loop)

---

## Data Structure Comparison

### ToolResult Struct (tool_registry.go:35-42)
```go
type ToolResult struct {
    ID         string      `json:"id"`
    Name       string      `json:"name"`
    Args       map[string]interface{} `json:"args"`
    Output     interface{} `json:"output,omitempty"`     // ← Structured data
    Error      string      `json:"error,omitempty"`
    DurationMs int64       `json:"durationMs"`
}
```

### Message Struct (provider.go:19-31)
```go
type Message struct {
    Role    string `json:"role"`    // "tool_result" expected
    Content string `json:"content"` // Empty for tool_result

    // ✅ This field should be populated for tool results
    ToolResult *ToolResult `json:"toolResult,omitempty"`

    // ✅ This field should be populated for tool calls
    ToolCall   *ToolCall   `json:"toolCall,omitempty"`
}
```

### Buggy Message (Current)
```go
Message{
    Role:       "system",
    Content:    "Tool 'coordinator_create_human_task' result: {\"taskId\":\"task_abc123\"}",
    ToolResult: nil,  // ❌ Not populated!
    ToolCall:   nil,
}
```

### Correct Message (After Fix)
```go
Message{
    Role:       "tool_result",
    Content:    "",
    ToolResult: &ToolResult{  // ✅ Populated with structured data
        ID:     "call_123",
        Name:   "coordinator_create_human_task",
        Output: map[string]interface{}{"taskId": "task_abc123"},
        Error:  "",
    },
    ToolCall:   nil,
}
```

---

## Code Evolution Timeline

### How This Bug Likely Happened

1. **Initial Implementation** (Path 1)
   - Developer added basic tool calling support
   - Used simple string-based approach for tool results
   - Added as system messages (quick implementation)

2. **Anthropic Integration Discovery**
   - Realized Anthropic needs structured tool_result format
   - Created correct implementation (Path 2)
   - But only applied to "filtered processing path"

3. **Bug Persists**
   - Main processing path (Path 1) never updated
   - Both paths exist in parallel
   - Path 1 executes in most cases
   - Path 2 only executes in specific filtered scenarios

4. **Current State**
   - Two incompatible implementations coexist
   - Main path is buggy but heavily used
   - Correct path exists but rarely executed

---

**Document Version:** 1.0
**Last Updated:** 2025-01-25
**Related:** TOOL_RESULT_ANALYSIS.md
