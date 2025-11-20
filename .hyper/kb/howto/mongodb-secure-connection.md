# MongoDB Secure Connection Pattern

## Overview

Standard MongoDB connection pattern for Hyperion microservices using the official Go driver with proper initialization, authentication, and security controls.

## Technology

- MongoDB 7.0 with Go Driver
- User JWT-based authentication
- Multi-tenant data isolation

## Use Case

Use this pattern when initializing MongoDB connections in any Hyperion service. This pattern enforces security standards requiring user JWT identity for all operations and proper multi-tenant isolation.

## Implementation

### Main Application Initialization

**File Reference**: `hyper/cmd/coordinator/main.go:294-331`

```go
// 1. Get connection configuration from environment
mongoURI := os.Getenv("MONGODB_URI")
if mongoURI == "" {
    logger.Fatal("MONGODB_URI environment variable is required")
}

mongoDatabase := os.Getenv("MONGODB_DATABASE")
if mongoDatabase == "" {
    mongoDatabase = "coordinator_db1" // default fallback
}

// 2. Create connection with timeout context
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 3. Connect using options
clientOptions := options.Client().ApplyURI(mongoURI)
mongoClient, err := mongo.Connect(ctx, clientOptions)
if err != nil {
    logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
}

// 4. Always defer disconnect
defer func() {
    if err := mongoClient.Disconnect(context.Background()); err != nil {
        logger.Error("Error disconnecting from MongoDB", zap.Error(err))
    }
}()

// 5. Verify connection with ping
if err := mongoClient.Ping(ctx, nil); err != nil {
    logger.Fatal("Failed to ping MongoDB", zap.Error(err))
}

// 6. Get database instance
db := mongoClient.Database(mongoDatabase)
```

### Service Initialization Pattern

**File Reference**: `hyper/internal/services/chat_service.go:7-47`

Services receive a `*mongo.Database` and create collections with indexes:

```go
func NewChatService(db *mongo.Database, logger *zap.Logger) (*ChatService, error) {
    service := &ChatService{
        mongoClient:        db.Client(),
        sessionsCollection: db.Collection("chat_sessions"),
        messagesCollection: db.Collection("chat_messages"),
        logger:             logger,
    }

    // Create indexes for efficient queries
    ctx := context.Background()

    // Composite index for user isolation
    _, err := service.sessionsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{
            {Key: "userId", Value: 1},
            {Key: "companyId", Value: 1},
        },
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create sessions user index: %w", err)
    }

    return service, nil
}
```

## Key Points

### Security Requirements

- **JWT Identity Required**: Per CLAUDE.md security standards, all MongoDB operations MUST use user JWT identity extracted from request context
- **Company Isolation**: Always filter by `companyId` for multi-tenant data isolation
- **No System Accounts**: Never use system service identities for MongoDB connections
- **User Context Propagation**: Extract `userId` and `companyId` from JWT claims in middleware (`middleware/auth.go`)

### Best Practices

1. **Environment-driven config**: URI and database name from env vars
2. **Connection timeout**: Always use context with timeout for Connect()
3. **Ping verification**: Verify connection after establishing
4. **Graceful cleanup**: Defer disconnect in separate goroutine to handle errors
5. **Index creation**: Create indexes during service initialization, not runtime
6. **Error wrapping**: Use `fmt.Errorf("context: %w", err)` for error chains

### Configuration

Environment variables required:
- `MONGODB_URI`: Connection string (required)
- `MONGODB_DATABASE`: Database name (default: coordinator_db1)

## Metadata

- **Domain**: database
- **Language**: go
- **Pattern**: connection-initialization
- **Technology**: mongodb
