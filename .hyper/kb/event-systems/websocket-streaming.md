# Hyperion WebSocket Streaming Architecture

**Collection:** event-systems
**Tags:** WebSocket, streaming, real-time, concurrency
**File Reference:** handlers/chat_websocket.go:1-71
**Version:** 1.0

---

HYPERION WEBSOCKET STREAMING ARCHITECTURE

Real-time chat communication via WebSocket with graceful lifecycle management.

WEBSOCKET HANDLER (handlers/chat_websocket.go):
Connection Flow:
- Endpoint: GET /api/v1/chat/stream?sessionId=xxx
- Protocol: JSON frames with streaming chunks
- Handler: HandleChatWebSocket(c *gin.Context)

StreamCleanup Lifecycle Manager:
- streamCtx: context.Context for propagating cancellation
- cancelFunc: Cancels context on disconnect
- wg: sync.WaitGroup coordinates goroutine exit
- writeMutex: sync.Mutex protects concurrent WebSocket writes

Concurrent Goroutines:
1. Ping Ticker: Sends periodic pings to maintain keep-alive
2. Message Stream: Processes queries, streams AI responses in chunks
3. Both coordinate via done channel and writeMutex

Graceful Shutdown:
- Done channel closes on disconnect
- All goroutines receive signal via context cancellation
- WaitGroup.Wait() blocks until both goroutines exit
- Prevents race conditions between ping and message streaming

WRITE SAFETY:
writeMutex protects concurrent writes:
- Ping goroutine: Sends ping frames
- Message stream goroutine: Sends response chunks
- Lock ensures WebSocket frame ordering

MESSAGE FORMAT:
Client: { query: string, sessionId: string, options?: {...} }
Server chunks:
- { type: "chunk", data: string } (partial response)
- { type: "complete", data: json } (full response metadata)
- { type: "error", data: string } (error message)

PERFORMANCE:
- Buffered channels prevent blocking
- Keep-alive pings prevent connection timeout (browser defaults)
- Async processing doesn't block goroutine
- Context cancellation propagates instantly to all goroutines
