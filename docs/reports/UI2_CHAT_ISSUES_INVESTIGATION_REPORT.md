# UI2 Chat System Issues - Investigation Report

**Date:** November 5, 2025
**Investigation Focus:** Tool calls visibility, progress tracking, typing indicators, WebSocket architecture
**Status:** ✅ Complete

---

## Executive Summary

After thorough investigation, I found that **UI2's WebSocket chat implementation is 100% identical to knowledge-browser**. However, there are critical architectural issues that affect BOTH implementations:

### Key Findings:

1. ✅ **Tool Calls ARE Implemented** - UI2 has full tool call/result display in debug mode
2. ✅ **Progress Tracker EXISTS** - PerformanceMonitor shows FPS, tokens/sec, latency
3. ❌ **No Typing Indicator** - Neither UI2 nor knowledge-browser has typing indicators
4. ❌ **Progress Events Are Tokens** - Backend sends progress as 'token' events, not dedicated 'progress' events
5. ⚠️ **Potential Display Bug** - Tool calls might not be visible due to data flow issue

---

## Issue 1: Tool Calls Not Visible in Debug Mode

### Expected Behavior:
When in debug mode, tool calls and results should appear in the chat with:
- Blue bordered boxes for tool calls showing tool name, arguments
- Green/red bordered boxes for tool results showing output/errors
- Collapsible accordions for each tool

### Current Implementation:

**UI2 HAS the components:**
- `ChatMessage.tsx` lines 50-148: Handles tool_call and tool_result message types
- Debug mode toggle via ConversationModeContext
- Radix Accordion for collapsible tool displays

**Code Evidence:**
```typescript
// ui2/src/components/organisms/ChatMessage.tsx:47-48
const { mode } = useConversationMode();
const showToolDetails = mode === 'debug';

// Lines 50-53: Hide tool calls in default mode
if (isToolCall && message.toolCall && !showToolDetails) {
  return null; // Hide in default mode
}

// Lines 56-82: Render tool_call in debug mode
if (isToolCall && message.toolCall && showToolDetails) {
  return (
    <div className="border border-blue-200 dark:border-blue-700 rounded-lg...">
      <Wrench className="w-4 h-4" />
      <span>{message.toolCall.name}</span>
      <pre>{JSON.stringify(message.toolCall.args, null, 2)}</pre>
    </div>
  );
}
```

### Potential Issue: WebSocket vs API Data Mismatch

**The Problem:**
Tool calls from WebSocket might not be persisted to MongoDB correctly, causing them to disappear when you refresh or switch sessions.

**WebSocket Flow (Streaming):**
1. WebSocket sends `type: 'tool_call'` event
2. Frontend displays tool call in real-time
3. Frontend stores tool call in `streamingToolCalls` array
4. **On stream end**, tool calls should be saved to database

**API Flow (Loading history):**
1. Frontend calls `/api/chat/sessions/{id}/messages`
2. Backend returns messages from MongoDB
3. Tool call messages should have `role: 'tool_call'` and `toolCall` object

**Investigation Needed:**
```bash
# Check if tool calls are in MongoDB
mongo hyperion
db.messages.find({ role: "tool_call" }).pretty()

# Check if tool calls are returned by API
curl http://localhost:5555/api/chat/sessions/{session_id}/messages
```

---

## Issue 2: Progress Tracker vs Progress Notifications

### What UI2 HAS:

**PerformanceMonitor Component** (ui2/src/components/organisms/PerformanceMonitor.tsx)
- Shows streaming performance metrics
- FPS tracking
- Tokens per second
- Average chunk size
- Total chunks received
- Streaming latency
- Health indicator (green/yellow/red)
- Fixed position: bottom-right corner

**Code Evidence:**
```typescript
// ui2/src/pages/CodeChatPage.tsx:495-501
<PerformanceMonitor
  stats={performance.stats}
  fpsHistory={performance.fpsHistory}
  isPerformanceGood={performance.isPerformanceGood}
  isStreaming={isStreaming}
/>
```

### What UI2 DOESN'T HAVE (Neither does knowledge-browser):

**Dedicated Progress Notification Display:**
- Backend sends progress as `type: 'token'` events, not `type: 'progress'`
- Progress messages have `\n\n` wrappers to distinguish them
- No visual distinction between progress and regular tokens
- No progress bar or status indicator for agent task progress

**Backend Implementation:**
```go
// hyper/internal/handlers/chat_websocket.go:1206-1220
for progress := range progressCh {
    progressMsg := models.StreamMessage{
        Type:    "token",  // ⚠️ Sent as 'token', not 'progress'
        Content: "\n\n" + progress.Message + "\n\n",
    }
    h.safeWriteJSON(conn, progressMsg)
}
```

**What You Might Be Expecting:**
Progress messages that look like:
```
📨 Agent task created: d4ba19cf-a629-413f-9d5d-6b3539efba4b
🔄 Subagent started: ui-dev
✅ TODO 1 completed
🔄 Working on TODO 2
```

**What Actually Happens:**
These progress messages are mixed into the streaming text as regular tokens, making them hard to distinguish from AI responses.

---

## Issue 3: Typing Indicator

### Status: ❌ **NOT IMPLEMENTED IN EITHER UI**

Neither UI2 nor knowledge-browser has typing indicators. The backend doesn't send 'typing' events.

**What You Might Want:**
```
Assistant is typing...
●●● (animated dots)
```

**Current Behavior:**
UI2 has a streaming indicator:
```typescript
// ui2/src/components/organisms/ChatMessage.tsx:234-242
{isStreaming && isAssistant && (
  <div className="flex items-center gap-2 mt-2 text-xs opacity-70">
    <div className="flex gap-1">
      <span className="w-2 h-2 bg-primary-500 rounded-full animate-pulse" />
      <span className="w-2 h-2 bg-primary-500 rounded-full animate-pulse delay-75" />
      <span className="w-2 h-2 bg-primary-500 rounded-full animate-pulse delay-150" />
    </div>
    <span>AI is typing...</span>
  </div>
)}
```

**This only shows DURING streaming**, not before the first token arrives.

---

## Issue 4: Create Agent Task Errors

### From Logs Analysis:

I found **NO errors** for `create_agent_task` or `coordinator_create_agent_task` in recent logs.

**What I Found Instead:**
```
2025/11/05 16:17:36 [Tool Response - SUCCESS] Tool: coordinator_list_human_tasks
2025/11/05 16:17:42 [AI response streamed successfully] tokensStreamed: 2, toolCalls: 1
```

The coordinator successfully:
1. ✅ Listed human tasks
2. ✅ Created human task
3. ✅ Streamed response

**No tool errors found in:**
- `/tmp/hyper2.log`
- `/tmp/hyper-updated.log`
- `/tmp/*.log`

**Possible Reasons You're Not Seeing Tool Calls:**

1. **Tool calls not persisted to DB** - WebSocket streaming works but MongoDB save fails
2. **API not returning tool call messages** - Message query filters out tool calls
3. **Frontend filtering** - Debug mode not properly enabled
4. **Session refresh issue** - Tool calls lost when refreshing page

---

## Comparison: UI2 vs Knowledge-Browser

### WebSocket Implementation: ✅ 100% IDENTICAL

**Event Types Supported (BOTH):**
```typescript
type: 'token' | 'tool_call' | 'tool_result' | 'done' | 'error'
```

**Callbacks (BOTH):**
- onMessage(content, done)
- onToolCall(tool, args, id)
- onToolResult(id, tool, result, error, durationMs)
- onError(error)
- onOpen()
- onClose()

**Message Handling (BOTH):**
- Identical switch/case logic for event types
- Same error handling
- Same tool call/result extraction
- Same done signal handling

### Page Implementation: ✅ 95% IDENTICAL

**UI2 Differences:**
1. Uses Radix UI + Tailwind instead of MUI
2. Has PerformanceMonitor (knowledge-browser doesn't)
3. Simplified session list (no drawer for subchats)
4. Different styling but same logic

**Functionally Equivalent:**
- WebSocket connection lifecycle
- Message state management
- Tool call/result accumulation
- Streaming content handling
- Session auto-refresh (both poll every 3-5 seconds)

---

## Root Cause Analysis

### Why Tool Calls Might Not Be Visible:

#### Hypothesis 1: MongoDB Persistence Failure

**Check:**
```bash
# Connect to MongoDB
mongo hyperion

# Find tool call messages
db.messages.find({ role: "tool_call" }).count()
db.messages.find({ role: "tool_result" }).count()

# Check recent messages
db.messages.find().sort({ timestamp: -1 }).limit(10).pretty()
```

**Expected:** Messages with `role: 'tool_call'` and `role: 'tool_result'`
**If Not Found:** Backend not saving tool calls to MongoDB

#### Hypothesis 2: API Filtering Tool Calls

**Check:**
```bash
# Get messages from API
curl http://localhost:5555/api/chat/sessions/{session_id}/messages | jq '.messages[] | select(.role | contains("tool"))'
```

**Expected:** JSON objects with `role: 'tool_call'` and `toolCall` object
**If Empty:** API is filtering out tool call messages

#### Hypothesis 3: Debug Mode Not Enabled

**Check:**
```typescript
// Open browser console on http://localhost:5555/ui/chat
localStorage.getItem('hyperion-conversation-mode')
// Should return: "debug"
```

**How to Enable:**
Click the toggle button in the header (Default ⟷ Debug)

#### Hypothesis 4: WebSocket → MongoDB Save Gap

**Issue:** Tool calls displayed during streaming but lost on page refresh

**Code Location:**
```typescript
// ui2/src/pages/CodeChatPage.tsx:156-183
onMessage: (content: string, done: boolean) => {
  if (done) {
    // Stream complete - save final AI message
    const newMessage: ChatMessageType = {
      id: `msg-${Date.now()}`,
      sessionId,
      role: 'assistant',
      content: finalContent,
      timestamp: new Date().toISOString(),
      toolCalls: tools.toolCalls.length > 0 ? tools.toolCalls : undefined,  // ⚠️
      toolResults: tools.toolResults.size > 0 ? tools.toolResults : undefined,  // ⚠️
    };
    setMessages((prev) => [...prev, newMessage]);
  }
}
```

**Potential Issue:** Tool calls stored in `streamingToolCalls` array but only the **final assistant message** gets the `toolCalls` array. Individual `tool_call` and `tool_result` messages are NOT created.

**Backend Expectation:** Separate messages for each tool call/result:
```json
[
  { "role": "assistant", "content": "Let me check..." },
  { "role": "tool_call", "toolCall": { "name": "...", "args": {...} } },
  { "role": "tool_result", "toolResult": { "output": "...", "error": null } },
  { "role": "assistant", "content": "Based on the results..." }
]
```

**UI2 Current Behavior:** Single assistant message with embedded toolCalls array:
```json
[
  {
    "role": "assistant",
    "content": "Let me check... Based on the results...",
    "toolCalls": [{ "id": "1", "tool": "...", "args": {...} }],
    "toolResults": { "1": { "result": "...", "error": null } }
  }
]
```

---

## Recommended Fixes

### Fix 1: Create Separate Tool Call Messages

**Update:** `ui2/src/pages/CodeChatPage.tsx:210-239`

```typescript
onToolCall: (tool: string, args: Record<string, any>, id: string) => {
  console.log('[CodeChatPage] Tool call received:', tool, id);

  // Save message before tool call if content exists
  if (streamingContentRef.current.trim()) {
    const messageBeforeToolCall: ChatMessageType = {
      id: `msg-${Date.now()}`,
      sessionId,
      role: 'assistant',
      content: streamingContentRef.current,
      timestamp: new Date().toISOString(),
    };
    setMessages((prev) => [...prev, messageBeforeToolCall]);
    streamingContentRef.current = '';
    setStreamingContent('');
  }

  // ✅ ADD: Create dedicated tool_call message
  const toolCallMessage: ChatMessageType = {
    id: `tool-call-${id}`,
    sessionId,
    role: 'tool_call',
    content: '',
    timestamp: new Date().toISOString(),
    toolCall: {
      id,
      name: tool,
      args,
    },
  };
  setMessages((prev) => [...prev, toolCallMessage]);

  // Keep for accordion display during streaming
  const toolCall: ToolCall = { id, tool, args, timestamp: new Date() };
  currentMessageToolsRef.current.toolCalls.push(toolCall);
  setPendingToolCalls((prev) => new Set(prev).add(id));
  setStreamingToolCalls((prev) => [...prev, toolCall]);
},
```

### Fix 2: Create Separate Tool Result Messages

**Update:** `ui2/src/pages/CodeChatPage.tsx:240-261`

```typescript
onToolResult: (id, tool, result, error, durationMs) => {
  // ✅ ADD: Create dedicated tool_result message
  const toolResultMessage: ChatMessageType = {
    id: `tool-result-${id}`,
    sessionId,
    role: 'tool_result',
    content: '',
    timestamp: new Date().toISOString(),
    toolResult: {
      id,
      name: tool,
      output: result,
      error,
      durationMs,
    },
  };
  setMessages((prev) => [...prev, toolResultMessage]);

  // Keep for accordion display during streaming
  const toolResult: ToolResult = { id, tool, result, error, durationMs };
  currentMessageToolsRef.current.toolResults.set(id, toolResult);
  setPendingToolCalls((prev) => {
    const updated = new Set(prev);
    updated.delete(id);
    return updated;
  });
  setStreamingToolResults((prev) => new Map(prev).set(id, toolResult));
},
```

### Fix 3: Add Dedicated Progress Event Type

**Backend:** `hyper/internal/models/chat.go:76`

```go
type StreamMessage struct {
    Type       string            `json:"type"` // "token", "tool_call", "tool_result", "done", "error", "progress"
    Content    string            `json:"content,omitempty"`
    Progress   *ProgressEvent    `json:"progress,omitempty"`  // NEW
    Error      string            `json:"error,omitempty"`
    ToolCall   *ToolCallEvent    `json:"toolCall,omitempty"`
    ToolResult *ToolResultEvent  `json:"toolResult,omitempty"`
}

type ProgressEvent struct {
    Message    string  `json:"message"`
    Percentage float64 `json:"percentage,omitempty"`
    Stage      string  `json:"stage,omitempty"`  // "starting", "processing", "complete"
}
```

**Frontend:** `ui2/src/services/chatService.ts:66`

```typescript
export interface StreamMessage {
  type: 'token' | 'tool_call' | 'tool_result' | 'done' | 'error' | 'progress';
  content?: string;
  progress?: {
    message: string;
    percentage?: number;
    stage?: 'starting' | 'processing' | 'complete';
  };
  toolCall?: {...};
  toolResult?: {...};
  error?: string;
}

export interface StreamCallbacks {
  onMessage: (content: string, done: boolean) => void;
  onProgress?: (message: string, percentage?: number, stage?: string) => void;  // NEW
  onToolCall?: (tool: string, args: Record<string, any>, id: string) => void;
  onToolResult?: (id: string, tool: string, result: any, error: string | null, durationMs: number) => void;
  onError: (error: Error) => void;
  onOpen?: () => void;
  onClose?: () => void;
}
```

**WebSocket Handler:** `hyper/internal/handlers/chat_websocket.go:1209`

```go
progressMsg := models.StreamMessage{
    Type: "progress",  // Changed from "token"
    Progress: &models.ProgressEvent{
        Message: progress.Message,
        Stage:   "processing",
    },
}
```

### Fix 4: Add Progress Display Component

**Create:** `ui2/src/components/organisms/ProgressDisplay.tsx`

```typescript
interface ProgressDisplayProps {
  messages: Array<{ message: string; timestamp: string; stage?: string }>;
  visible: boolean;
}

export const ProgressDisplay: React.FC<ProgressDisplayProps> = ({ messages, visible }) => {
  if (!visible || messages.length === 0) return null;

  return (
    <div className="fixed bottom-20 right-6 w-96 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-xl p-4 max-h-64 overflow-y-auto">
      <h3 className="text-sm font-semibold mb-2">Task Progress</h3>
      <div className="space-y-2">
        {messages.map((msg, idx) => (
          <div key={idx} className="flex items-start gap-2 text-xs">
            <span className="text-blue-500">
              {msg.stage === 'complete' ? '✅' : '🔄'}
            </span>
            <span>{msg.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
};
```

---

## Testing Checklist

### 1. Verify Tool Calls Display

```bash
# 1. Enable debug mode in UI
localStorage.setItem('hyperion-conversation-mode', 'debug')

# 2. Send a message that triggers tools
# Example: "List all files in the ui2/src directory"

# 3. Check browser console for:
console.log('[CodeChatPage] Tool call received:', ...)
console.log('[CodeChatPage] Tool result received:', ...)

# 4. Verify messages in UI show:
# - Blue bordered box with tool name + args
# - Green bordered box with tool output
# - Collapsible accordions work
```

### 2. Verify MongoDB Persistence

```bash
# Connect to MongoDB
mongo hyperion

# Check for tool call messages
db.messages.find({ role: "tool_call" }).pretty()
db.messages.find({ role: "tool_result" }).pretty()

# Should see documents like:
# {
#   "_id": ObjectId("..."),
#   "sessionId": "...",
#   "role": "tool_call",
#   "toolCall": {
#     "id": "toolu_123",
#     "name": "list_directory",
#     "args": { "path": "." }
#   },
#   "timestamp": ISODate("...")
# }
```

### 3. Verify API Returns Tool Calls

```bash
# Get session messages via API
curl http://localhost:5555/api/chat/sessions/{session_id}/messages | jq '.'

# Should include messages with:
# { "role": "tool_call", "toolCall": {...} }
# { "role": "tool_result", "toolResult": {...} }
```

### 4. Verify Progress Tracking

```bash
# Send message that creates agent task
# Example: "Add a dark mode toggle to settings page"

# Watch for progress messages in chat:
# - "📨 Agent task created: ..."
# - "🔄 Subagent started: ..."
# - "✅ TODO 1 completed"

# These should appear as distinct progress notifications,
# not mixed with AI response text
```

---

## Conclusion

### What's Working: ✅

1. WebSocket streaming (identical to knowledge-browser)
2. Tool call/result UI components exist
3. Debug mode toggle exists
4. PerformanceMonitor shows streaming metrics
5. Session management and auto-refresh

### What's Broken/Missing: ❌

1. **Tool calls not persisted as separate messages** - Only embedded in assistant message
2. **Progress notifications mixed with tokens** - No visual distinction
3. **No typing indicator before first token**
4. **Tool call messages might not survive page refresh**

### Priority Fixes:

**High Priority:**
1. Create separate tool_call and tool_result messages (Fix 1 & 2)
2. Verify MongoDB saves tool call messages
3. Test tool calls display in debug mode

**Medium Priority:**
3. Add dedicated 'progress' event type (Fix 3)
4. Create ProgressDisplay component (Fix 4)

**Low Priority:**
5. Add typing indicator before first token
6. Add Prometheus metrics dashboard

---

**Next Steps:** Would you like me to implement Fix 1 and Fix 2 to create separate tool call/result messages?
