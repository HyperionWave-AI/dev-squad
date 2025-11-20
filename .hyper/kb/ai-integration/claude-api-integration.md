# Hyperion Claude API Integration - AI Service Layer

**Collection:** ai-integration
**Tags:** Claude, Anthropic, LangChain, AI-service
**File Reference:** internal/ai-service/provider.go
**Version:** 1.0

---

HYPERION CLAUDE API INTEGRATION - AI SERVICE LAYER

Claude/Anthropic Integration (internal/ai-service/):

AI PROVIDERS SUPPORTED:
- Anthropic Claude API (primary, via LangChain)
- OpenAI GPT (fallback)
- Custom providers (pluggable architecture)

ANTHROPIC PROVIDER (ai-service/provider.go):
Implementation:
- Uses langchaingo for Claude API client
- Supports Claude models (claude-sonnet, claude-opus)
- Tool calling enabled for function execution
- Token counting via Anthropic SDK

CLAUDE API FEATURES:
- Model selection: Environment-driven configuration
- Tool execution: Automatic tool calling for MCP tools
- Streaming: Server-sent events for real-time responses
- Temperature/sampling: Configurable inference parameters
- Max tokens: Controllable output length

TASK SUMMARIZATION (ai-service/summarizer.go):
- AI-powered summaries for tasks (≤100 tokens)
- Fallback to text truncation if AI unavailable
- Async operation: Doesn't block task creation
- Optional: Gracefully degrades if not configured

AI SERVICE INITIALIZATION (main.go):
Optional service - logs if failed:
```go
var taskSummarizer storage.TaskSummarizer
// Summarizer can be nil (graceful degradation)
```

CONFIGURATION:
Environment variables:
- AI_PROVIDER: anthropic|openai (default)
- ANTHROPIC_API_KEY: Claude API key
- CLAUDE_MODEL: Model name (e.g., claude-3-sonnet)
- OPENAI_API_KEY: OpenAI key (fallback)

USAGE PATTERNS:
- Chat streaming: Real-time response chunks via WebSocket
- Tool calling: Automatic function invocation
- Context generation: Prompt engineering for task understanding
- Summarization: Extract key points from long content

LIMITATIONS:
- Rate limiting: API quotas respected
- Token budgets: Max tokens enforced per request
- Streaming timeouts: Connection management
- Error recovery: Graceful fallbacks
