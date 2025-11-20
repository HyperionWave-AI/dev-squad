# Go Microservice Scaffolding Pattern

## Overview

Standard scaffolding pattern for initializing Go microservices in Hyperion using dependency injection, service layer architecture, and the Gin web framework.

## Technology

- Go 1.25
- Gin Framework
- Zap Logger
- MongoDB for persistence

## Use Case

Use this pattern when creating new HTTP-based microservices in the Hyperion ecosystem. This pattern establishes consistent service initialization, dependency management, and routing setup across all services.

## Implementation

### HTTP Server Initialization

**File Reference**: `hyper/internal/server/http_server.go:121-220`

```go
func StartHTTPServer(
    ctx context.Context,
    port string,
    taskStorage storage.TaskStorage,
    knowledgeStorage storage.KnowledgeStorage,
    // ... other dependencies
    mongoDatabase *mongo.Database,
    logger *zap.Logger,
) error {
    // 1. Initialize services
    chatService, err := services.NewChatService(mongoDatabase, logger)
    if err != nil {
        return err
    }

    aiSettingsService, err := services.NewAISettingsService(mongoDatabase, logger)
    if err != nil {
        return err
    }

    // 2. Load configuration
    aiConfig, err := aiservice.LoadAIConfig(".env.hyper")
    if err != nil {
        return err
    }

    // 3. Create business logic services
    aiChatService, err := aiservice.NewChatService(aiConfig)
    if err != nil {
        return err
    }

    // 4. Setup router and middleware
    router := gin.Default()
    router.Use(cors.Default())
    router.Use(middleware.OptionalJWTMiddleware())

    // 5. Register routes
    api := router.Group("/api/v1")
    chatHandler.RegisterRoutes(api.Group("/chat"))

    // 6. Start server
    return router.Run(":" + port)
}
```

## Key Points

### Architecture Patterns

1. **Dependency Injection**: All storage clients and services passed as parameters
2. **Service Layer**: Business logic encapsulated in service structs (ChatService, AISettingsService)
3. **Config Loading**: Environment-driven configuration via .env files
4. **Error Propagation**: Early return on initialization errors (fail-fast)
5. **Logger Integration**: Zap logger passed to all components for structured logging
6. **Graceful Composition**: Build complex services from simple, testable components

### Service Initialization

```go
// Services receive dependencies through constructor
func NewChatService(db *mongo.Database, logger *zap.Logger) (*ChatService, error) {
    service := &ChatService{
        mongoClient:        db.Client(),
        sessionsCollection: db.Collection("chat_sessions"),
        messagesCollection: db.Collection("chat_messages"),
        logger:             logger,
    }

    // Initialize indexes, connections, etc.
    if err := service.setupIndexes(); err != nil {
        return nil, err
    }

    return service, nil
}
```

### Router and Middleware Setup

```go
router := gin.Default()

// Global middleware
router.Use(cors.Default())
router.Use(middleware.OptionalJWTMiddleware())

// Route grouping
api := router.Group("/api/v1")
{
    chatHandler := NewChatHandler(chatService)
    chatHandler.RegisterRoutes(api.Group("/chat"))

    taskHandler := NewTaskHandler(taskStorage)
    taskHandler.RegisterRoutes(api.Group("/tasks"))
}
```

### Configuration Loading

```go
// Load from .env file
aiConfig, err := aiservice.LoadAIConfig(".env.hyper")
if err != nil {
    return fmt.Errorf("failed to load AI config: %w", err)
}

// Access config values
service := aiservice.NewChatService(aiConfig)
```

### Best Practices

1. **Fail Fast**: Return errors immediately during initialization
2. **Centralized Dependencies**: Pass all dependencies through function parameters
3. **Structured Logging**: Use zap.Logger with fields for context
4. **Middleware Order**: Apply CORS before authentication
5. **Route Grouping**: Use Gin route groups for API versioning and organization
6. **Context Propagation**: Pass context.Context through service calls

### Error Handling

```go
// Constructor pattern with error return
func StartHTTPServer(...) error {
    service, err := services.NewChatService(db, logger)
    if err != nil {
        return fmt.Errorf("failed to initialize chat service: %w", err)
    }

    // Continue initialization...
}
```

### Handler Registration

```go
// Handler encapsulates service dependencies
type ChatHandler struct {
    service *ChatService
    logger  *zap.Logger
}

// RegisterRoutes groups related endpoints
func (h *ChatHandler) RegisterRoutes(group *gin.RouterGroup) {
    group.POST("/sessions", h.CreateSession)
    group.GET("/sessions/:id", h.GetSession)
    group.POST("/messages", h.SendMessage)
}
```

## Metadata

- **Domain**: microservices
- **Language**: go
- **Pattern**: scaffolding
- **Technology**: go, gin
