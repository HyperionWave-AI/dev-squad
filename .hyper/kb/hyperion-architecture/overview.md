# ## Overview

**Collection:** hyperion-architecture
**Created:** 2025-11-20

---

## Overview

Hyperion follows a microservices-inspired architecture with clear separation between backend Go services, frontend React application, and external integrations (MongoDB, Qdrant, AI providers).

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         Frontend                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  React 19 + Vite + TypeScript                        │   │
│  │  - Atomic Design (atoms/molecules/organisms)         │   │
│  │  - Radix UI + Tailwind CSS                           │   │
│  │  - Service Layer (REST + WebSocket)                  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────┬───────────────────────────────────────┘
                      │ HTTP/WebSocket
                      ↓
┌─────────────────────────────────────────────────────────────┐
│                    Go Backend (Coordinator)                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  HTTP Server (Gin)                                   │   │
│  │  - REST API Handlers                                 │   │
│  │  - WebSocket Chat                                    │   │
│  │  - JWT Middleware                                    │   │
│  │  - Rate Limiting                                     │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Internal Services                                   │   │
│  │  ├── MCP (Model Context Protocol)                   │   │
│  │  │   ├── Storage (MongoDB + Qdrant)                 │   │
│  │  │   ├── Handlers (Tools)                           │   │
│  │  │   ├── Embeddings (TEI/Ollama/Voyage/OpenAI)      │   │
│  │  │   ├── Indexer (Code parsing + chunking)          │   │
│  │  │   └── Parser (AST: Go/JS/TS/Python)              │   │
│  │  ├── AI Service                                      │   │
│  │  │   ├── Provider (Claude/OpenAI)                   │   │
│  │  │   └── Tools (Task/Knowledge/Code/Reflection)     │   │
│  │  └── Business Logic                                  │   │
│  │      ├── Task Management                            │   │
│  │      ├── Knowledge Management                        │   │
│  │      └── Code Indexing                              │   │
│  └──────────────────────────────────────────────────────┘   │
└─────┬───────────────────┬───────────────────┬───────────────┘
      │                   │                   │
      ↓                   ↓                   ↓
┌───────────┐      ┌────────────┐      ┌──────────────┐
│  MongoDB  │      │  Qdrant    │      │  AI Provider │
│  - Tasks  │      │  - Code    │      │  - Claude    │
│  - KB     │      │  - KB      │      │  - OpenAI    │
│  - Reflect│      │  - Lessons │      └──────────────┘
└───────────┘      └────────────┘
```

---

## Backend Service Structure

### Directory Organization

```
/hyper
├── cmd/
│   └── coordinator/          # Main entry point
│       └── main.go           # Server initialization
├── internal/
│   ├── ai-service/           # AI/LLM integration
│   │   ├── provider.go       # Multi-provider support
│   │   └── tools/            # MCP tool implementations
│   ├── api/                  # REST API layer
│   │   └── rest_handler.go   # Route registration
│   ├── handlers/             # HTTP request handlers
│   │   ├── chat_handler.go
│   │   ├── knowledge_handler.go
│   │   ├── task_handler.go
│   │   └── code_handler.go
│   ├── middleware/           # HTTP middleware
│   │   ├── jwt_auth.go
│   │   ├── rate_limit.go
│   │   └── cors.go
│   ├── mcp/                  # Model Context Protocol
│   │   ├── storage/          # MongoDB + Qdrant clients
│   │   │   ├── knowledge.go
│   │   │   ├── tasks.go
│   │   │   ├── qdrant_client.go
│   │   │   └── code_index.go
│   │   ├── handlers/         # MCP request handlers
│   │   │   ├── knowledge_handlers.go
│   │   │   ├── task_handlers.go
│   │   │   └── code_index_handlers.go
│   │   ├── embeddings/       # Embedding providers
│   │   │   ├── interface.go
│   │   │   ├── tei.go
│   │   │   ├── ollama.go
│   │   │   ├── voyage.go
│   │   │   └── openai.go
│   │   ├── indexer/          # Code indexing service
│   │   │   ├── indexer.go
│   │   │   ├── chunker.go
│   │   │   └── watcher.go
│   │   └── parser/           # AST parsing
│   │       ├── go_parser.go
│   │       ├── js_parser.go
│   │       ├── ts_parser.go
│   │       └── python_parser.go
│   └── services/             # Business logic
│       ├── task_service.go
│       ├── knowledge_service.go
│       └── code_service.go
└── pkg/                      # Shared utilities
    ├── logger/
    ├── utils/
    └── errors/
```

### Key Go Packages

| Package | Purpose |
|---------|---------|
| `github.com/gin-gonic/gin` | HTTP framework |
| `github.com/gorilla/websocket` | WebSocket support |
| `go.mongodb.org/mongo-driver` | MongoDB client |
| `github.com/golang-jwt/jwt/v5` | JWT authentication |
| `go.uber.org/zap` | Structured logging |
| `github.com/modelcontextprotocol/go-sdk` | MCP protocol |

### Main Entry Point

**File:** `/hyper/cmd/coordinator/main.go`

```go
func main() {
    // 1. Load configuration
    loadEnvConfig()

    // 2. Initialize logging
    logger := initLogger()

    // 3. Connect to MongoDB
    mongoClient, err := connectMongoDB()
    if err != nil {
        logger.Fatal("MongoDB connection failed", zap.Error(err))
    }

    // 4. Connect to Qdrant
    qdrantClient := initQdrantClient()

    // 5. Initialize storage layer
    storage := storage.NewStorage(mongoClient, qdrantClient)

    // 6. Initialize services
    taskService := services.NewTaskService(storage)
    knowledgeService := services.NewKnowledgeService(storage)
    codeService := services.NewCodeService(storage)

    // 7. Initialize handlers
    handlers := handlers.NewHandlers(taskService, knowledgeService, codeService)

    // 8. Setup HTTP server
    router := gin.Default()

    // 9. Register middleware
    router.Use(middleware.CORS())
    router.Use(middleware.JWTAuth())
    router.Use(middleware.RateLimit())

    // 10. Register routes
    api := router.Group("/api/v1")
    handlers.RegisterRoutes(api)

    // 11. Initialize file watcher (optional)
    if os.Getenv("ENABLE_FILE_WATCHER") == "true" {
        watcher := indexer.NewFileWatcher(codeService)
        go watcher.Start()
    }

    // 12. Start server
    port := os.Getenv("HTTP_PORT")
    logger.Info("Server starting", zap.String("port", port))
    router.Run(":" + port)
}
```

---

## Frontend Component Organization

### Directory Structure (Atomic Design)

```
/ui/src
├── components/
│   ├── atoms/                # Basic building blocks
│   │   ├── Button.tsx
│   │   ├── Input.tsx
│   │   ├── Badge.tsx
│   │   ├── Spinner.tsx
│   │   └── Icon.tsx
│   ├── molecules/            # Combinations of atoms
│   │   ├── SearchBar.tsx
│   │   ├── TaskCard.tsx
│   │   ├── ChatMessage.tsx
│   │   └── FileItem.tsx
│   ├── organisms/            # Complex components
│   │   ├── TaskList.tsx
│   │   ├── ChatWindow.tsx
│   │   ├── CodeSearch.tsx
│   │   ├── KnowledgeBase.tsx
│   │   └── FileInspector.tsx
│   └── templates/            # Page layouts
│       ├── MainLayout.tsx
│       └── ChatLayout.tsx
├── pages/                    # Page components (routes)
│   ├── TasksPage.tsx
│   ├── KnowledgeBasePage.tsx
│   ├── CodeSearchPage.tsx
│   └── ChatPage.tsx
├── hooks/                    # Custom React hooks
│   ├── useStreamingPerformance.ts
│   ├── useKeyboardShortcuts.ts
│   ├── useLocalStorage.ts
│   └── useDebounce.ts
├── services/                 # API clients
│   ├── restClient.ts
│   ├── codeClient.ts
│   ├── knowledgeService.ts
│   ├── mcpService.ts
│   └── chatService.ts
├── types/                    # TypeScript definitions
│   ├── tasks.ts
│   ├── knowledge.ts
│   ├── code.ts
│   └── chat.ts
├── utils/                    # Utility functions
│   ├── cn.ts                 # className utility
│   ├── formatters.ts
│   └── validators.ts
├── contexts/                 # React contexts
│   └── ConversationModeContext.tsx
├── App.tsx                   # Main app component
├── main.tsx                  # Entry point
└── index.css                 # Global styles
```

### Atomic Design Principles

**1. Atoms (Basic Elements)**
- Single-purpose components
- No dependencies on other components
- Highly reusable

```typescript
// Button.tsx
export function Button({ children, variant, size, ...props }: ButtonProps) {
  return (
    <button className={cn(buttonVariants({ variant, size }))} {...props}>
      {children}
    </button>
  )
}
```

**2. Molecules (Composite Components)**
- Combine multiple atoms
- Single responsibility
- Still reusable across contexts

```typescript
// SearchBar.tsx
export function SearchBar({ onSearch }: SearchBarProps) {
  return (
    <div className="flex gap-2">
      <Input placeholder="Search..." onChange={handleChange} />
      <Button onClick={onSearch}>
        <Search size={20} />
      </Button>
    </div>
  )
}
```

**3. Organisms (Complex Features)**
- Business logic and state management
- Compose molecules and atoms
- Feature-specific

```typescript
// TaskList.tsx
export function TaskList() {
  const [tasks, setTasks] = useState<Task[]>([])

  useEffect(() => {
    loadTasks()
  }, [])

  return (
    <div className="space-y-4">
      {tasks.map(task => (
        <TaskCard key={task.id} task={task} onUpdate={handleUpdate} />
      ))}
    </div>
  )
}
```

**4. Templates (Page Layouts)**
- Page structure and layout
- Compose organisms
- Route-specific

```typescript
// MainLayout.tsx
export function MainLayout({ children }: LayoutProps) {
  return (
    <div className="min-h-screen flex">
      <Sidebar />
      <main className="flex-1 p-6">
        {children}
      </main>
    </div>
  )
}
```

---

## Integration Points

### 1. Frontend → Backend (REST API)

**Protocol:** HTTP/HTTPS
**Format:** JSON
**Base Path:** `/api/v1`

```typescript
// Frontend service
const response = await fetch('/api/v1/tasks', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ prompt: 'Create feature' }),
})
```

```go
// Backend handler
func (h *TaskHandler) CreateTask(c *gin.Context) {
    var req CreateTaskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    task, err := h.taskService.Create(req.Prompt)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, task)
}
```

### 2. Frontend → Backend (WebSocket)

**Protocol:** WebSocket
**Path:** `/ws/chat`
**Format:** JSON messages

```typescript
// Frontend WebSocket
const ws = new WebSocket('ws://localhost:7095/ws/chat')

ws.onmessage = (event) => {
  const message = JSON.parse(event.data)
  handleIncomingMessage(message)
}

ws.send(JSON.stringify({
  type: 'user_message',
  content: 'Hello',
  sessionId: 'session-uuid',
}))
```

```go
// Backend WebSocket handler
func (h *ChatHandler) HandleWebSocket(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        return
    }
    defer conn.Close()

    for {
        var msg Message
        if err := conn.ReadJSON(&msg); err != nil {
            break
        }

        response := h.processMessage(msg)
        conn.WriteJSON(response)
    }
}
```

### 3. Backend → MongoDB

**Driver:** `go.mongodb.org/mongo-driver`
**Connection:** Connection pool
**Operations:** CRUD + Aggregation

```go
// Storage layer
func (s *TaskStorage) Create(task *Task) error {
    collection := s.db.Collection("human_tasks")

    doc := bson.M{
        "taskId":    task.ID,
        "prompt":    task.Prompt,
        "status":    task.Status,
        "createdAt": time.Now(),
    }

    _, err := collection.InsertOne(context.Background(), doc)
    return err
}
```

### 4. Backend → Qdrant

**Client:** Custom HTTP client
**Protocol:** HTTP REST API
**Operations:** Upsert, Search, Delete

```go
// Qdrant storage
func (q *QdrantClient) SearchSimilar(
    collection string,
    query string,
    limit int,
) ([]*QueryResult, error) {
    // 1. Generate embedding
    vector, err := q.embeddingFunc(query)

    // 2. Search Qdrant
    results, err := q.searchPoints(collection, vector, limit)

    return results, err
}
```

### 5. Backend → AI Provider

**Providers:** Claude (Anthropic), OpenAI
**Protocol:** HTTPS REST API
**Format:** JSON (streaming optional)

```go
// AI provider interface
type AIProvider interface {
    Chat(messages []Message, tools []Tool) (*Response, error)
    StreamChat(messages []Message, tools []Tool) (<-chan Token, error)
}

// Claude provider
func (p *ClaudeProvider) Chat(
    messages []Message,
    tools []Tool,
) (*Response, error) {
    req := anthropic.MessageRequest{
        Model:    "claude-sonnet-4-5",
        Messages: messages,
        Tools:    tools,
        MaxTokens: 4096,
    }

    return p.client.CreateMessage(req)
}
```

---

## Data Flow Examples

### Example 1: Create Human Task

```
1. User clicks "Create Task" in UI
   ↓
2. Frontend: restClient.createHumanTask("Add dark mode")
   ↓ HTTP POST /api/v1/tasks
3. Backend: TaskHandler.CreateTask()
   ↓
4. Backend: TaskService.Create()
   ↓
5. Backend: MongoDB.InsertOne(human_tasks)
   ↓
6. Backend: Return {taskId, status, createdAt}
   ↓ HTTP 200 Response
7. Frontend: Display success, update task list
```

### Example 2: Semantic Code Search

```
1. User enters query: "authentication logic"
   ↓
2. Frontend: codeClient.search({query, retrieve: "chunk-m"})
   ↓ HTTP POST /api/v1/code/search
3. Backend: CodeHandler.Search()
   ↓
4. Backend: Generate embedding from query
   ↓ HTTP POST to TEI/Embedding service
5. Embedding Service: Return vector [768 dimensions]
   ↓
6. Backend: QdrantClient.SearchSimilar(vector, limit: 10)
   ↓ HTTP POST to Qdrant
7. Qdrant: Vector similarity search
   ↓
8. Qdrant: Return top 10 results with scores
   ↓
9. Backend: Fetch chunk metadata from MongoDB
   ↓
10. Backend: Return formatted results
   ↓ HTTP 200 Response
11. Frontend: Display code results with highlighting
```

### Example 3: AI Chat with Tool Calling

```
1. User: "What tasks are pending?"
   ↓
2. Frontend: Send message via WebSocket
   ↓
3. Backend: Receive message, prepare AI request
   ↓
4. Backend: Claude API with tools=[list_human_tasks]
   ↓ HTTPS POST to Anthropic API
5. Claude: Analyze message, decide to use tool
   ↓
6. Claude: Return tool_use block
   ↓
7. Backend: Execute tool: TaskStorage.List(status="pending")
   ↓
8. Backend: MongoDB query: {status: "pending"}
   ↓
9. MongoDB: Return matching tasks
   ↓
10. Backend: Send tool result to Claude
    ↓ HTTPS POST to Anthropic API
11. Claude: Generate natural language response
    ↓
12. Backend: Stream response via WebSocket
    ↓
13. Frontend: Display streaming response to user
```

---

## Deployment Architecture

### Development

```
┌──────────────┐
│   Developer  │
│   Machine    │
├──────────────┤
│  Docker      │
│  Compose:    │
│  - MongoDB   │
│  - Qdrant    │
│  - TEI       │
└──────────────┘
       ↑
       │ localhost
       ↓
┌──────────────┐
│ Go Backend   │
│ :7095        │
└──────────────┘
       ↑
       │ localhost
       ↓
┌──────────────┐
│ Vite Dev     │
│ Server :5173 │
└──────────────┘
```

### Production (GKE)

```
┌────────────────────────────────────┐
│          Load Balancer             │
└────────────────┬───────────────────┘
                 │
     ┌───────────┴───────────┐
     │                       │
┌────▼─────┐         ┌──────▼──────┐
│ Backend  │         │   Frontend  │
│ Pods     │◄────────┤   (Static)  │
│ (Go)     │         │   (Nginx)   │
└────┬─────┘         └─────────────┘
     │
     ├──────► MongoDB Atlas (Cloud)
     ├──────► Qdrant (Pod or Cloud)
     └──────► Claude API (Anthropic)
```

**Kubernetes Resources:**
- Deployments: backend, frontend
- Services: ClusterIP, LoadBalancer
- ConfigMaps: environment variables
- Secrets: API keys, credentials
- PersistentVolumes: Qdrant data (if self-hosted)

---

## Related Documents

- [API Service Layer](./api-service-layer.md) - REST endpoints
- [MongoDB Integration](./mongodb-integration.md) - Database layer
- [Qdrant Integration](./qdrant-integration.md) - Vector database
- [UI Client Stack](./ui-client-stack.md) - Frontend technologies
- [Data Contracts](./data-contracts.md) - Type definitions
- [Configuration Reference](./configuration-reference.md) - Environment variables

## Best Practices

### Backend

1. **Separation of Concerns:** Handlers → Services → Storage
2. **Error Handling:** Return structured errors with context
3. **Logging:** Use structured logging (zap) with fields
4. **Testing:** Unit tests for services, integration tests for handlers
5. **Security:** JWT validation, rate limiting, input validation

### Frontend

1. **Atomic Design:** Build from atoms → molecules → organisms
2. **Type Safety:** Use TypeScript for all components
3. **State Management:** Context for global state, hooks for local
4. **Error Boundaries:** Catch rendering errors gracefully
5. **Accessibility:** ARIA labels, keyboard navigation, semantic HTML

### Integration

1. **API Versioning:** Use `/api/v1` prefix
2. **Error Responses:** Consistent error format
3. **Pagination:** Limit + offset for large datasets
4. **Caching:** Cache static data (collections, file metadata)
5. **Monitoring:** Log all integration points for debugging
