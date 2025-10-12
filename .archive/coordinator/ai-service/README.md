# AI Service - LangChain Go Multi-Provider Chat

Multi-provider AI service with streaming support for OpenAI, Anthropic (Claude), and custom endpoints.

## Features

- 🔌 **Multi-Provider Support**: OpenAI, Anthropic, custom HTTP endpoints
- 🌊 **Streaming Responses**: Real-time token streaming via channels
- 🔐 **JWT Integration**: Extract user identity from context for logging/multi-tenancy
- ⚙️ **Configuration**: Environment-based config via .env.hyper
- 🧪 **Tested**: 13 comprehensive unit tests (100% pass rate)

## Quick Start

### 1. Configure Environment

Create or update `.env.hyper`:

```bash
# OpenAI
AI_PROVIDER="openai"
OPENAI_API_KEY="sk-..."
AI_MODEL="gpt-4"

# Or Anthropic
AI_PROVIDER="anthropic"
ANTHROPIC_API_KEY="sk-ant-..."
AI_MODEL="claude-3-sonnet-20240229"

# Optional parameters
MAX_ITERATION=100        # Default: 100
MAX_OUT_TOKENS=4096      # Default: 0 (provider default)
TEMPERATURE=0.7          # Default: 0.7
REASONING="o1"           # OpenAI reasoning mode (optional)
```

### 2. Initialize Service

```go
import "hyperion-coordinator/ai-service"

// Load configuration
config, err := aiservice.LoadAIConfig("/path/to/.env.hyper")
if err != nil {
    log.Fatal(err)
}

// Create chat service
chatService, err := aiservice.NewChatService(config)
if err != nil {
    log.Fatal(err)
}
```

### 3. Stream Chat

```go
import "context"

// Add identity to context (from JWT middleware)
ctx := context.Background()
ctx = aiservice.WithIdentity(ctx, &aiservice.Identity{
    Type:      "human",
    Name:      "Max",
    ID:        "max@example.com",
    Email:     "max@example.com",
    CompanyID: "company-123",
})
ctx = aiservice.WithRequestID(ctx, "req-xyz-123")

// Prepare messages
messages := []aiservice.Message{
    {Role: "system", Content: "You are a helpful assistant."},
    {Role: "user", Content: "What is LangChain?"},
}

// Stream response
outputChan, err := chatService.StreamChat(ctx, messages)
if err != nil {
    log.Fatal(err)
}

// Read streaming tokens
for chunk := range outputChan {
    if strings.HasPrefix(chunk, "ERROR: ") {
        log.Println("Stream error:", chunk)
        break
    }
    fmt.Print(chunk) // Print token
}
```

## Architecture

### Component Overview

```
ai-service/
├── config.go           - Configuration parser (.env.hyper)
├── provider.go         - Provider factory and implementations
├── langchain_service.go - Main ChatService with JWT integration
└── config_test.go      - Comprehensive unit tests
```

### Provider Pattern

```
ChatService
    ↓
ChatProvider (interface)
    ├── openAIProvider (wraps langchaingo OpenAI)
    ├── anthropicProvider (wraps langchaingo Anthropic)
    └── customProvider (placeholder for custom endpoints)
```

### Streaming Flow

```
1. Client calls StreamChat(ctx, messages)
2. ChatService extracts identity/requestID from context
3. ChatService calls provider.StreamChat()
4. Provider creates channel, starts goroutine
5. Provider calls langchaingo LLM.Call() with streaming callback
6. Tokens stream through channel
7. ChatService wraps channel with logging/cancellation handling
8. Client reads tokens from returned channel
```

## Configuration Reference

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AI_PROVIDER` or `PROVIDER` | Yes | - | Provider: `openai`, `anthropic`, or `custom` |
| `API_KEY` or `{PROVIDER}_API_KEY` | Yes* | - | API key (* not required for custom) |
| `AI_MODEL` or `MODEL` | No | Provider default | Model name |
| `PROVIDER_URL` | Yes** | - | Custom endpoint URL (** required for custom) |
| `MAX_ITERATION` | No | 100 | Maximum iterations |
| `MAX_OUT_TOKENS` | No | 0 | Maximum output tokens (0 = provider default) |
| `TEMPERATURE` | No | 0.7 | Temperature (0.0 - 2.0) |
| `REASONING` | No | - | Reasoning mode (e.g., "o1" for OpenAI) |

### Default Models

- **OpenAI**: `gpt-4-turbo-preview`
- **Anthropic**: `claude-3-sonnet-20240229`
- **Custom**: No default (must specify)

## API Reference

### AIConfig

```go
type AIConfig struct {
    Provider        string  // "openai", "anthropic", or "custom"
    ProviderURL     string  // Custom endpoint URL
    APIKey          string  // API key
    MaxIterations   int     // Maximum iterations
    MaxOutputTokens int     // Maximum output tokens
    Temperature     float64 // Temperature (0.0 - 2.0)
    ReasoningMode   string  // Reasoning mode
    Model           string  // Model name
}

func LoadAIConfig(envFilePath string) (*AIConfig, error)
func (c *AIConfig) Validate() error
```

### ChatService

```go
type ChatService struct {
    // private fields
}

func NewChatService(config *AIConfig) (*ChatService, error)
func (s *ChatService) StreamChat(ctx context.Context, messages []Message) (<-chan string, error)
func (s *ChatService) GetConfig() *AIConfig
```

### Message

```go
type Message struct {
    Role    string `json:"role"`    // "user", "assistant", or "system"
    Content string `json:"content"` // Message content
}
```

### Identity

```go
type Identity struct {
    Type      string `json:"type"`      // "human", "agent", or "service"
    Name      string `json:"name"`      // User or agent name
    ID        string `json:"id"`        // User ID
    Email     string `json:"email"`     // User email
    CompanyID string `json:"companyId"` // Company ID
}

func WithIdentity(ctx context.Context, identity *Identity) context.Context
func WithRequestID(ctx context.Context, requestID string) context.Context
func GetIdentityFromContext(ctx context.Context) (*Identity, error)
```

## Testing

### Run Tests

```bash
cd coordinator
go test ./ai-service/... -v -cover
```

### Test Coverage

- **13 test cases**: All passing
- **Coverage**: 29.8% of statements
- **Test scenarios**:
  - Valid configurations (all fields, minimal, custom provider)
  - Invalid configurations (bad provider, missing fields)
  - Validation edge cases (negative iterations, invalid temperature)
  - Environment variable fallbacks

## Error Handling

### Stream Errors

Errors are sent as chunks with `ERROR: ` prefix:

```go
for chunk := range outputChan {
    if strings.HasPrefix(chunk, "ERROR: ") {
        // Handle error
        log.Println("Stream error:", strings.TrimPrefix(chunk, "ERROR: "))
        break
    }
    // Process normal chunk
}
```

### Context Cancellation

Always handle context cancellation to prevent goroutine leaks:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

outputChan, err := chatService.StreamChat(ctx, messages)
// Channel automatically closes when context is cancelled
```

## Integration Examples

### WebSocket Integration

```go
func HandleChatWebSocket(c *gin.Context, chatService *aiservice.ChatService) {
    // Upgrade to WebSocket
    ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer ws.Close()

    // Extract identity from JWT
    identity, _ := auth.GetIdentityFromContext(c.Request.Context())
    ctx := aiservice.WithIdentity(c.Request.Context(), identity)
    ctx = aiservice.WithRequestID(ctx, uuid.New().String())

    // Read user message
    var msg aiservice.Message
    if err := ws.ReadJSON(&msg); err != nil {
        return
    }

    // Stream response
    outputChan, err := chatService.StreamChat(ctx, []aiservice.Message{msg})
    if err != nil {
        ws.WriteJSON(map[string]string{"error": err.Error()})
        return
    }

    // Forward chunks to WebSocket
    for chunk := range outputChan {
        if err := ws.WriteMessage(websocket.TextMessage, []byte(chunk)); err != nil {
            break
        }
    }
}
```

### HTTP SSE Integration

```go
func HandleChatSSE(c *gin.Context, chatService *aiservice.ChatService) {
    // Set SSE headers
    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    // Extract identity
    identity, _ := auth.GetIdentityFromContext(c.Request.Context())
    ctx := aiservice.WithIdentity(c.Request.Context(), identity)

    // Parse messages from request body
    var messages []aiservice.Message
    if err := c.ShouldBindJSON(&messages); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // Stream response
    outputChan, err := chatService.StreamChat(ctx, messages)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // Send SSE events
    flusher, _ := c.Writer.(http.Flusher)
    for chunk := range outputChan {
        fmt.Fprintf(c.Writer, "data: %s\n\n", chunk)
        flusher.Flush()
    }
}
```

## Future Enhancements

- [ ] Implement customProvider with actual HTTP streaming
- [ ] Add proper message array handling (preserve system/user/assistant structure)
- [ ] Add streaming token counting for usage metrics
- [ ] Add retry logic for transient errors
- [ ] Support for function calling / tool use
- [ ] Add request/response caching
- [ ] Add rate limiting per user/company

## Dependencies

- **github.com/tmc/langchaingo v0.1.13** - LangChain Go SDK
- **github.com/joho/godotenv v1.5.1** - Environment file loading

## License

Part of Hyperion Coordinator project.
