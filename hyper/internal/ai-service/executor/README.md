# AI Stream Executor Package

## Overview

The `executor` package provides a **universal AI streaming execution framework** that works for both parent chat (WebSocket) and subchat (background) contexts. It extracts and unifies the core streaming logic from `chat_websocket.go:1364-2061` and `coordinator_tools.go:2868-3400`.

## Key Design Principles

### 1. Universal Stream Processing
- **Single implementation** for both parent chat and subchat
- **Context-agnostic** - works with any output destination
- **The ONLY difference** between contexts is:
  - Tool filtering (coordinator vs. implementation tools)
  - Output destination (WebSocket vs. ProgressNotifier)

### 2. Resilient Execution
- Continues processing even if client disconnects
- Panic recovery prevents crashes
- Saves data to database regardless of client state
- Buffer overflow protection

### 3. Observable & Debuggable
- Comprehensive logging at all stages
- Prometheus metrics integration
- Clear state tracking (tokens, tool calls, disconnection)

### 4. Interruptible
- Non-blocking interrupt detection
- Priority interrupt handling (checked before each event)
- Graceful interrupt acknowledgment

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     StreamExecutor                          │
│  (Universal stream processing logic)                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ├──────────────────────────────────┐
                              │                                  │
                    ┌─────────▼─────────┐          ┌───────────▼──────────┐
                    │  WebSocketSink    │          │ ProgressNotification │
                    │  (Parent Chat)    │          │  Sink (Subchat)      │
                    └───────────────────┘          └──────────────────────┘
                              │                                  │
                    ┌─────────▼─────────┐          ┌───────────▼──────────┐
                    │  gorilla/websocket│          │  ProgressNotifier    │
                    │  (Real-time UI)   │          │  (Parent Chat UI)    │
                    └───────────────────┘          └──────────────────────┘
```

## Core Components

### 1. `stream_executor.go` - Core Streaming Logic

**`StreamConfig`**: Configuration for stream execution
- `SessionID`: Chat session for message storage
- `CompanyID`: Multi-tenancy identifier
- `SystemPrompt`: AI system prompt
- `AllowedTools`: Tool filter (nil = all, [] = none, ["tool1"] = specific)
- `OutputSink`: Destination for stream output
- `InterruptCh`: Channel for user interrupts
- `ToolResultProcessor`: Custom result processing
- `CompletionValidator`: Custom completion logic

**`StreamExecutor`**: Main executor
- `NewStreamExecutor()`: Creates executor instance
- `Execute()`: Runs the universal streaming loop

**Key Methods**:
- `handleToken()`: Process text tokens
- `handleToolCall()`: Process tool invocations
- `handleToolResult()`: Process tool results

### 2. `sinks.go` - Output Destinations

**`StreamOutputSink`**: Interface for output destinations
- `SendToken()`: Send text chunk
- `SendToolCall()`: Send tool invocation
- `SendToolResult()`: Send tool result
- `SendDone()`: Signal completion
- `SendError()`: Send error message
- `IsDisconnected()`: Check client connection

**Implementations**:

**`WebSocketSink`**: For parent chat
- Direct streaming to WebSocket client
- Detects client disconnection (CloseGoingAway, etc.)
- Thread-safe with mutex protection

**`ProgressNotificationSink`**: For subchat
- Streams via ProgressNotifier to parent chat
- No individual token streaming (avoids flooding)
- Never "disconnects" (runs to completion)

### 3. `interfaces.go` - Service Abstractions

**`ChatServiceInterface`**: Chat operations
- `SaveMessage()`: Save message to database
- `SaveToolCall()`: Save tool call to database
- `SaveToolResult()`: Save tool result to database

**`AIServiceInterface`**: AI streaming
- `StreamChatWithTools()`: Stream with all tools
- `StreamChatWithToolsFiltered()`: Stream with specific tools
- `GetConfig()`: Get AI configuration

**`CompletionValidatorFunc`**: Custom completion logic
- Check if streaming should stop based on custom criteria

**`ToolResultProcessorFunc`**: Custom result processing
- Process tool results for size limits, formatting, etc.

## Usage Examples

### Parent Chat (WebSocket)

```go
import (
    "hyper/internal/ai-service/executor"
    "hyper/internal/handlers"
)

// Create WebSocket sink
sink := executor.NewWebSocketSink(conn, logger)

// Create interrupt channel
interruptCh := handlers.GetMessageNotifier(logger).RegisterSession(sessionID)
defer handlers.GetMessageNotifier(logger).UnregisterSession(sessionID)

// Configure executor
config := executor.StreamConfig{
    SessionID:    sessionID,
    CompanyID:    companyID,
    SystemPrompt: systemPromptText,
    AllowedTools: nil, // All tools (coordinator mode)
    OutputSink:   sink,
    InterruptCh:  interruptCh,
    Logger:       logger,
}

// Create executor
exec := executor.NewStreamExecutor(config, chatService, aiService)

// Execute streaming
fullResponse, err := exec.Execute(ctx, messages)
if err != nil {
    logger.Error("Streaming failed", zap.Error(err))
}
```

### Subchat (Background)

```go
import (
    "hyper/internal/ai-service/executor"
    "hyper/internal/handlers"
)

// Create progress notification sink
notifier := handlers.GetProgressNotifier(logger)
sink := executor.NewProgressNotificationSink(parentSessionID, notifier, logger)

// Configure executor with filtered tools
config := executor.StreamConfig{
    SessionID:    subchatSessionID,
    CompanyID:    companyID,
    SystemPrompt: subagentSystemPrompt,
    AllowedTools: []string{
        "read_file",
        "write_file",
        "apply_patch",
        "bash",
        "coordinator_update_todo_status",
        "coordinator_upsert_knowledge",
    },
    OutputSink:  sink,
    InterruptCh: interruptCh,
    Logger:      logger,
}

// Create executor
exec := executor.NewStreamExecutor(config, chatService, aiService)

// Execute streaming (runs in background)
fullResponse, err := exec.Execute(ctx, messages)
if err != nil {
    logger.Error("Subchat execution failed", zap.Error(err))
}
```

### With Custom Tool Result Processing

```go
// Define custom processor for size limits
toolProcessor := func(toolName string, output interface{}) (string, bool, bool) {
    outputStr := fmt.Sprintf("%v", output)
    size := len(outputStr)

    if size > 500*1024 { // 500KB
        // Suppress large results
        return "⚠️ Result suppressed (too large)", true, false
    } else if size > 120*1024 { // 120KB
        // Truncate medium results
        preview := outputStr[:10*1024]
        return preview + "\n\n_[Truncated...]_", true, true
    }

    // Normal results
    return outputStr, true, true
}

config := executor.StreamConfig{
    // ... other config ...
    ToolResultProcessor: toolProcessor,
}
```

### With Custom Completion Logic

```go
// Stop after 5 tool calls
completionValidator := func(fullResponse string, toolCallCount int) bool {
    return toolCallCount >= 5
}

config := executor.StreamConfig{
    // ... other config ...
    CompletionValidator: completionValidator,
}
```

## Stream Event Flow

```
┌─────────────────────────────────────────────────────────────┐
│  AI Service (LangChain/Provider)                            │
│  Emits: StreamEvent channel                                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
        ┌─────────────────────────────────────────────┐
        │  StreamExecutor.Execute()                   │
        │                                             │
        │  For each event:                            │
        │    1. Check interrupts (non-blocking)       │
        │    2. Check context cancellation            │
        │    3. Process event:                        │
        │       - Token: accumulate + stream          │
        │       - ToolCall: save + stream             │
        │       - ToolResult: save + stream           │
        │    4. Check custom completion               │
        │    5. Buffer overflow protection            │
        └─────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    │                   │
                    ▼                   ▼
        ┌───────────────────┐  ┌────────────────────┐
        │  Database Save    │  │  OutputSink        │
        │  (ChatService)    │  │  (Stream to UI)    │
        └───────────────────┘  └────────────────────┘
```

## Error Handling

### Panic Recovery
```go
defer func() {
    if r := recover() {
        logger.Error("Panic during streaming", zap.Any("panic", r))
        // Save partial response
        // Notify client
    }
}()
```

### Client Disconnection
- Detected via `IsDisconnected()` check
- Processing continues in background
- Database saves still occur
- Logged for debugging

### Context Cancellation
- Checked on every event
- Saves accumulated response
- Returns `ctx.Err()`

### Buffer Overflow
- Checked on every token
- Truncates at `MaxStreamBufferBytes` (5MB)
- Emits truncation notice
- Records metric

## Metrics

The executor records the following Prometheus metrics:

- `AIStreamTokens`: Total tokens streamed (counter)
- `AIStreamDuration`: Stream duration distribution (histogram)
- `AIResponseTruncations`: Truncation count (counter)

## Testing

### Unit Test Structure

```go
func TestStreamExecutor_ParentChat(t *testing.T) {
    // Mock ChatService
    mockChatService := &MockChatService{}

    // Mock AIService
    mockAIService := &MockAIService{}

    // Create test WebSocket
    conn := createTestWebSocket(t)

    // Create executor
    sink := NewWebSocketSink(conn, logger)
    config := StreamConfig{
        SessionID: sessionID,
        OutputSink: sink,
        // ...
    }
    exec := NewStreamExecutor(config, mockChatService, mockAIService)

    // Execute
    response, err := exec.Execute(ctx, messages)

    // Assertions
    assert.NoError(t, err)
    assert.NotEmpty(t, response)
}
```

## Migration Guide

### From chat_websocket.go

**Before** (lines 1364-2061):
```go
func (h *ChatWebSocketHandler) streamAIResponse(...) {
    // 697 lines of streaming logic
    for event := range aiStream {
        switch event.Type {
        case aiservice.StreamEventToken:
            // Handle token
        case aiservice.StreamEventToolCall:
            // Handle tool call
        // ...
        }
    }
}
```

**After**:
```go
func (h *ChatWebSocketHandler) streamAIResponse(...) {
    sink := executor.NewWebSocketSink(conn, h.logger)
    config := executor.StreamConfig{
        SessionID: sessionID,
        CompanyID: companyID,
        SystemPrompt: systemPromptText,
        AllowedTools: allowedTools,
        OutputSink: sink,
        InterruptCh: interruptCh,
        Logger: h.logger,
    }
    exec := executor.NewStreamExecutor(config, h.chatService, h.aiService)
    fullResponse, err := exec.Execute(ctx, messages)
}
```

### From coordinator_tools.go

**Before** (lines 2868-3400):
```go
func (t *ExecuteSubagentTool) executeSubagentInBackground(...) {
    // 532 lines of subchat streaming logic
    for event := range aiStream {
        switch event.Type {
        case aiservice.StreamEventToken:
            fullResponse += event.Content
        case aiservice.StreamEventToolCall:
            // Handle tool call
        // ...
        }
    }
}
```

**After**:
```go
func (t *ExecuteSubagentTool) executeSubagentInBackground(...) {
    notifier := handlers.GetProgressNotifier(t.logger)
    sink := executor.NewProgressNotificationSink(parentSessionID, notifier, t.logger)
    config := executor.StreamConfig{
        SessionID: chatSession.ID,
        CompanyID: companyID,
        SystemPrompt: systemPrompt,
        AllowedTools: allowedTools,
        OutputSink: sink,
        InterruptCh: notifyCh,
        Logger: t.logger,
    }
    exec := executor.NewStreamExecutor(config, t.chatService, t.aiService)
    fullResponse, err := exec.Execute(ctx, messages)
}
```

## Design Decisions

### Why Separate Sinks?
- **Abstraction**: StreamExecutor doesn't know about WebSocket or ProgressNotifier
- **Testability**: Easy to mock output destinations
- **Flexibility**: New output types (gRPC, HTTP/2, SSE) just need new sink

### Why Not Use Channels?
- **Simplicity**: Direct method calls are easier to understand
- **Error Handling**: Methods can return errors immediately
- **Backpressure**: Not needed - we control the output pace

### Why Keep Database Saves in Executor?
- **Consistency**: Database state must match stream state
- **Atomicity**: Save happens at correct points (before tool calls, etc.)
- **Recovery**: Enables recovery from crashes/disconnects

## Future Enhancements

1. **Streaming Compression**: Compress large outputs
2. **Incremental Saves**: Save tokens in batches to reduce DB load
3. **Stream Multiplexing**: Send to multiple sinks simultaneously
4. **Rate Limiting**: Control output rate per client
5. **Resume Support**: Resume interrupted streams

## References

- Original parent chat implementation: `hyper/internal/handlers/chat_websocket.go:1364-2061`
- Original subchat implementation: `hyper/internal/ai-service/tools/mcp/coordinator_tools.go:2868-3400`
- Config limits: `hyper/internal/config/limits.go`
- Metrics: `hyper/internal/metrics/registry.go`
- Stream events: `hyper/internal/ai-service/langchain_service.go`
