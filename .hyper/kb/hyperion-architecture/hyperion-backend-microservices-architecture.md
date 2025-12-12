# HYPERION BACKEND MICROSERVICES ARCHITECTURE

**Collection:** hyperion-architecture
**Created:** 2025-11-20

---

HYPERION BACKEND MICROSERVICES ARCHITECTURE

Service Organization (internal/services/):

UNIFIED COORDINATOR SERVICE (cmd/coordinator/main.go):
Central orchestrator combining all services:
- HTTP REST API (port 8080): Client-facing endpoints
- MCP Server (stdio): Claude Code integration
- WebSocket streaming: Real-time chat responses
- File watcher: Automatic code indexing
- Embedded UI: Web dashboard included in binary

CORE INTERNAL PACKAGES:

1. AI SERVICE (internal/ai-service/):
   - provider.go: Pluggable AI providers (Claude, OpenAI)
   - cache.go: Prompt/response caching
   - message_converter.go: Protocol translation
   - config.go: AI configuration management
   - summarizer.go: Task summarization service
   - tool_registry.go: Tool metadata management

2. MCP HANDLERS (internal/mcp/handlers/):
   - tools.go: Coordinator tool management (19+ tools)
   - tools_discovery.go: MCP hub integration
   - code_tools.go: Code indexing operations
   - filesystem_tools.go: File I/O operations
   - reflection_tools.go: Metacognitive system
   - coordination_prompts.go: Workflow orchestration

3. STORAGE LAYER (internal/mcp/storage/):
   - knowledge.go: MongoDB + Qdrant knowledge persistence
   - tasks.go: Task and TODO management
   - qdrant_client.go: Vector database operations
   - code_index_storage.go: Code metadata and embeddings
   - reflection_storage.go: Decision/outcome tracking
   - tools_storage.go: MCP tool registry

4. CODE INDEXING (internal/mcp/):
   - parser/: Language parsers (Go, JavaScript, Python)
   - indexer/: Auto-indexing engine
   - embeddings/: Multiple embedding providers
   - watcher/: Filesystem monitoring
   - review/: Code review/verification engine

5. HTTP HANDLERS (internal/handlers/):
   - chat_handler.go: REST chat endpoints
   - chat_websocket.go: WebSocket streaming
   - subchat_handler.go: Sub-conversation management
   - ai_settings_handler.go: User preference endpoints

6. MIDDLEWARE (internal/middleware/):
   - auth.go: JWT authentication (optional dev mode)
   - rate_limiter.go: Token bucket rate limiting
   - panic_recovery.go: Error handling

DATA FLOW:
User Request → HTTP Handler → AI Service → MCP Tools → Storage Layer → Response

INTEGRATION POINTS:
- MongoDB: Persistent task/knowledge storage
- Qdrant: Vector search for semantics
- Embedding Services: Ollama, OpenAI, Voyage
- Claude API: AI reasoning and tool calling
- File System: Code indexing and watching
