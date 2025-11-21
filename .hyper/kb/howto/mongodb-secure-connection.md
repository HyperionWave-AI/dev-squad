# How to Connect to MongoDB Securely

**Collection:** howto
**Tags:** mongodb, database, security, connection, go
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide demonstrates how to establish secure MongoDB connections in Go applications using user JWT identity for all database operations. This pattern is essential for multi-tenant applications and ensures data isolation between users and organizations.

## Prerequisites

- MongoDB 4.4 or later installed
- Go 1.25 with `go.mongodb.org/mongo-driver`
- Understanding of JWT authentication
- Knowledge of [MongoDB Integration](../mongodb-integration.md)

## When to Use This Guide

- Building multi-tenant SaaS applications
- Implementing user-scoped database access
- Following zero-trust security principles
- Ensuring data isolation between organizations

**Security Requirement:** Hyperion mandates ALL MongoDB operations use user JWT identity (`database.NewSecureMongoClient`). System-level service identities are prohibited.

---

## Steps

### Step 1: Install MongoDB Driver

Add the official MongoDB Go driver:

```bash
go get go.mongodb.org/mongo-driver/mongo
go get go.mongodb.org/mongo-driver/mongo/options
go get go.mongodb.org/mongo-driver/bson
```

### Step 2: Define Configuration Structure

Create `internal/database/config.go`:

```go
package database

import (
    "fmt"
    "time"
)

type MongoConfig struct {
    // Connection
    URI          string
    Database     string
    
    // Pool settings
    MaxPoolSize  uint64
    MinPoolSize  uint64
    MaxIdleTime  time.Duration
    
    // Timeouts
    ConnectTimeout time.Duration
    PingTimeout    time.Duration
    
    // Security
    TLSEnabled bool
    TLSCert    string
}

// DefaultConfig returns production-ready defaults
func DefaultConfig() *MongoConfig {
    return &MongoConfig{
        MaxPoolSize:    100,
        MinPoolSize:    10,
        MaxIdleTime:    30 * time.Minute,
        ConnectTimeout: 10 * time.Second,
        PingTimeout:    5 * time.Second,
        TLSEnabled:     true,
    }
}

func (c *MongoConfig) Validate() error {
    if c.URI == "" {
        return fmt.Errorf("MongoDB URI is required")
    }
    if c.Database == "" {
        return fmt.Errorf("Database name is required")
    }
    return nil
}
```

### Step 3: Create Secure MongoDB Client

Implement user-scoped client creation:

```go
package database

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.uber.org/zap"
)

type SecureMongoClient struct {
    client    *mongo.Client
    database  *mongo.Database
    userID    string
    companyID string
    logger    *zap.Logger
}

// NewSecureMongoClient creates a MongoDB client with user identity context
// REQUIRED: All operations must use this pattern with JWT-derived userId/companyId
func NewSecureMongoClient(
    config *MongoConfig,
    userID string,
    companyID string,
    logger *zap.Logger,
) (*SecureMongoClient, error) {
    // Validate user identity
    if userID == "" {
        return nil, fmt.Errorf("userID is required for secure MongoDB access")
    }
    if companyID == "" {
        return nil, fmt.Errorf("companyID is required for secure MongoDB access")
    }

    // Validate configuration
    if err := config.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    // Build client options
    clientOpts := buildClientOptions(config)

    // Connect to MongoDB
    ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
    defer cancel()

    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    // Verify connection
    pingCtx, pingCancel := context.WithTimeout(context.Background(), config.PingTimeout)
    defer pingCancel()

    if err := client.Ping(pingCtx, nil); err != nil {
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

    logger.Info("MongoDB connection established",
        zap.String("database", config.Database),
        zap.String("companyId", companyID),
    )

    return &SecureMongoClient{
        client:    client,
        database:  client.Database(config.Database),
        userID:    userID,
        companyID: companyID,
        logger:    logger,
    }, nil
}

func buildClientOptions(config *MongoConfig) *options.ClientOptions {
    opts := options.Client().
        ApplyURI(config.URI).
        SetMaxPoolSize(config.MaxPoolSize).
        SetMinPoolSize(config.MinPoolSize).
        SetMaxConnIdleTime(config.MaxIdleTime).
        SetConnectTimeout(config.ConnectTimeout)

    // Enable TLS if configured
    if config.TLSEnabled {
        tlsConfig := &tls.Config{
            InsecureSkipVerify: false, // Always verify in production
        }
        opts.SetTLSConfig(tlsConfig)
    }

    return opts
}

// Close gracefully disconnects from MongoDB
func (c *SecureMongoClient) Close(ctx context.Context) error {
    c.logger.Info("Closing MongoDB connection", zap.String("companyId", c.companyID))
    return c.client.Disconnect(ctx)
}
```

### Step 4: Implement User-Scoped Query Methods

Add methods that enforce user/company filtering:

```go
// Collection returns a collection with automatic user scoping
func (c *SecureMongoClient) Collection(name string) *UserScopedCollection {
    return &UserScopedCollection{
        collection: c.database.Collection(name),
        userID:     c.userID,
        companyID:  c.companyID,
        logger:     c.logger,
    }
}

type UserScopedCollection struct {
    collection *mongo.Collection
    userID     string
    companyID  string
    logger     *zap.Logger
}

// FindOne with automatic user/company filtering
func (c *UserScopedCollection) FindOne(
    ctx context.Context,
    customFilter bson.M,
) *mongo.SingleResult {
    // Merge custom filter with security filter
    filter := c.buildSecureFilter(customFilter)
    
    c.logger.Debug("MongoDB FindOne",
        zap.String("collection", c.collection.Name()),
        zap.Any("filter", filter),
    )
    
    return c.collection.FindOne(ctx, filter)
}

// Find with automatic user/company filtering
func (c *UserScopedCollection) Find(
    ctx context.Context,
    customFilter bson.M,
    opts ...*options.FindOptions,
) (*mongo.Cursor, error) {
    // Merge custom filter with security filter
    filter := c.buildSecureFilter(customFilter)
    
    c.logger.Debug("MongoDB Find",
        zap.String("collection", c.collection.Name()),
        zap.Any("filter", filter),
    )
    
    return c.collection.Find(ctx, filter, opts...)
}

// InsertOne with automatic user/company injection
func (c *UserScopedCollection) InsertOne(
    ctx context.Context,
    document bson.M,
) (*mongo.InsertOneResult, error) {
    // Inject user identity into document
    document = c.injectUserIdentity(document)
    
    c.logger.Debug("MongoDB InsertOne",
        zap.String("collection", c.collection.Name()),
    )
    
    return c.collection.InsertOne(ctx, document)
}

// UpdateOne with automatic user/company filtering
func (c *UserScopedCollection) UpdateOne(
    ctx context.Context,
    customFilter bson.M,
    update bson.M,
) (*mongo.UpdateResult, error) {
    // Merge custom filter with security filter
    filter := c.buildSecureFilter(customFilter)
    
    c.logger.Debug("MongoDB UpdateOne",
        zap.String("collection", c.collection.Name()),
        zap.Any("filter", filter),
    )
    
    return c.collection.UpdateOne(ctx, filter, update)
}

// DeleteOne with automatic user/company filtering
func (c *UserScopedCollection) DeleteOne(
    ctx context.Context,
    customFilter bson.M,
) (*mongo.DeleteResult, error) {
    // Merge custom filter with security filter
    filter := c.buildSecureFilter(customFilter)
    
    c.logger.Debug("MongoDB DeleteOne",
        zap.String("collection", c.collection.Name()),
        zap.Any("filter", filter),
    )
    
    return c.collection.DeleteOne(ctx, filter)
}

// buildSecureFilter merges user/company filters with custom filter
func (c *UserScopedCollection) buildSecureFilter(customFilter bson.M) bson.M {
    if customFilter == nil {
        customFilter = bson.M{}
    }
    
    // Add companyId filter (required for all queries)
    customFilter["companyId"] = c.companyID
    
    return customFilter
}

// injectUserIdentity adds user/company fields to documents
func (c *UserScopedCollection) injectUserIdentity(doc bson.M) bson.M {
    if doc == nil {
        doc = bson.M{}
    }
    
    // Auto-inject identity fields
    doc["userId"] = c.userID
    doc["companyId"] = c.companyID
    doc["createdAt"] = time.Now().UTC()
    doc["updatedAt"] = time.Now().UTC()
    
    return doc
}
```

### Step 5: Create Client Factory

Build a factory to create clients from HTTP request context:

```go
package database

import (
    "fmt"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

type ClientFactory struct {
    config *MongoConfig
    logger *zap.Logger
}

func NewClientFactory(config *MongoConfig, logger *zap.Logger) *ClientFactory {
    return &ClientFactory{
        config: config,
        logger: logger,
    }
}

// FromGinContext creates a secure client from Gin context (with JWT claims)
func (f *ClientFactory) FromGinContext(c *gin.Context) (*SecureMongoClient, error) {
    // Extract user identity from JWT claims (set by auth middleware)
    userID, exists := c.Get("userId")
    if !exists {
        return nil, fmt.Errorf("userId not found in context")
    }
    
    companyID, exists := c.Get("companyId")
    if !exists {
        return nil, fmt.Errorf("companyId not found in context")
    }
    
    // Create secure client with user identity
    return NewSecureMongoClient(
        f.config,
        userID.(string),
        companyID.(string),
        f.logger,
    )
}
```

### Step 6: Use in HTTP Handlers

Integrate secure client in your handlers:

```go
package handlers

import (
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "your-project/internal/database"
)

type TaskHandler struct {
    clientFactory *database.ClientFactory
}

func NewTaskHandler(factory *database.ClientFactory) *TaskHandler {
    return &TaskHandler{clientFactory: factory}
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
    // Create user-scoped MongoDB client
    dbClient, err := h.clientFactory.FromGinContext(c)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create database client"})
        return
    }
    defer dbClient.Close(c.Request.Context())

    // Query with automatic company filtering
    collection := dbClient.Collection("tasks")
    
    cursor, err := collection.Find(c.Request.Context(), bson.M{
        "status": "pending", // Custom filter
        // companyId automatically added by UserScopedCollection
    })
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to query tasks"})
        return
    }
    defer cursor.Close(c.Request.Context())

    var tasks []Task
    if err := cursor.All(c.Request.Context(), &tasks); err != nil {
        c.JSON(500, gin.H{"error": "Failed to decode tasks"})
        return
    }

    c.JSON(200, gin.H{"tasks": tasks})
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
    var req CreateTaskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // Create user-scoped MongoDB client
    dbClient, err := h.clientFactory.FromGinContext(c)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create database client"})
        return
    }
    defer dbClient.Close(c.Request.Context())

    // Insert with automatic userId/companyId injection
    collection := dbClient.Collection("tasks")
    
    result, err := collection.InsertOne(c.Request.Context(), bson.M{
        "title":  req.Title,
        "status": "pending",
        // userId, companyId, createdAt, updatedAt automatically injected
    })
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create task"})
        return
    }

    c.JSON(201, gin.H{"id": result.InsertedID})
}
```

### Step 7: Configure Environment Variables

Set MongoDB configuration in `.env`:

```bash
# MongoDB Connection
MONGODB_URI=mongodb://admin:password@localhost:27017/?authSource=admin
MONGODB_DATABASE=my_app_db

# Connection Pool
MONGODB_MAX_POOL_SIZE=100
MONGODB_MIN_POOL_SIZE=10

# Security
MONGODB_TLS_ENABLED=true
MONGODB_TLS_CERT=/path/to/cert.pem
```

### Step 8: Initialize in Main Application

Set up client factory in `main.go`:

```go
package main

import (
    "os"
    "time"

    "your-project/internal/database"
    "your-project/internal/handlers"
)

func main() {
    logger, _ := zap.NewProduction()
    
    // Load MongoDB configuration
    mongoConfig := &database.MongoConfig{
        URI:            os.Getenv("MONGODB_URI"),
        Database:       os.Getenv("MONGODB_DATABASE"),
        MaxPoolSize:    100,
        MinPoolSize:    10,
        MaxIdleTime:    30 * time.Minute,
        ConnectTimeout: 10 * time.Second,
        PingTimeout:    5 * time.Second,
    }

    // Create client factory
    clientFactory := database.NewClientFactory(mongoConfig, logger)

    // Initialize handlers with factory
    taskHandler := handlers.NewTaskHandler(clientFactory)

    // Setup router
    router := setupRouter(taskHandler, logger)
    router.Run(":8080")
}
```

---

## Best Practices

### 1. Always Use User Identity
```go
// ✅ GOOD - User-scoped client
dbClient, err := NewSecureMongoClient(config, userID, companyID, logger)

// ❌ BAD - System-level client (PROHIBITED in Hyperion)
dbClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
```

### 2. Automatic Filter Injection
Never trust client-provided companyId - always inject from JWT:
```go
// ✅ GOOD - Server injects companyId from JWT
filter := bson.M{"companyId": jwtCompanyID}

// ❌ BAD - Accepting companyId from request body
filter := bson.M{"companyId": req.CompanyID}
```

### 3. Connection Pooling
Reuse client factory instead of creating new connections per request:
```go
// ✅ GOOD - Share factory across handlers
factory := database.NewClientFactory(config, logger)
handler := handlers.NewTaskHandler(factory)

// ❌ BAD - Creating new factory per request
func handler(c *gin.Context) {
    factory := database.NewClientFactory(config, logger)
}
```

### 4. Context Timeouts
Always use context timeouts for database operations:
```go
ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
defer cancel()

cursor, err := collection.Find(ctx, filter)
```

### 5. Graceful Shutdown
Close MongoDB connections on application shutdown:
```go
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    dbClient.Close(ctx)
}()
```

---

## Common Pitfalls

### 1. Missing User Context
Attempting to create secure client without JWT middleware:
```go
// Ensure JWT middleware runs before handler
api := router.Group("/api/v1")
api.Use(middleware.JWTAuthMiddleware(logger)) // REQUIRED
{
    api.GET("/tasks", taskHandler.ListTasks)
}
```

### 2. Leaking Data Across Companies
Forgetting to filter by companyId allows cross-company data access.

### 3. Not Handling Connection Errors
Always check and log connection errors for troubleshooting:
```go
if err := client.Ping(ctx, nil); err != nil {
    logger.Error("MongoDB connection failed", zap.Error(err))
    return nil, err
}
```

### 4. Hardcoding Database Names
Use environment variables for all configuration.

---

## Related Documentation

- [MongoDB Integration](../mongodb-integration.md) - Architecture and patterns
- [JWT Authentication](./jwt-authentication-middleware.md) - User identity extraction
- [MongoDB Schemas](../mongodb-schemas.md) - Collection structures
- [Data Contracts](../data-contracts.md) - Type definitions

---

## Troubleshooting

### Issue: "userId not found in context"

**Cause:** JWT middleware not applied or middleware ordering issue

**Solution:**
```go
// Ensure JWT middleware runs first
router.Use(middleware.JWTAuthMiddleware(logger))
router.GET("/tasks", taskHandler.ListTasks)
```

### Issue: "Connection timeout"

**Cause:** MongoDB not reachable or wrong connection string

**Solution:**
```bash
# Test connection manually
mongosh "mongodb://admin:password@localhost:27017/?authSource=admin"

# Check MongoDB is running
docker ps | grep mongo
```

### Issue: "Authentication failed"

**Cause:** Wrong credentials or auth source

**Solution:**
```bash
# Verify auth source in connection string
MONGODB_URI=mongodb://user:pass@host:27017/?authSource=admin
                                                      ^^^^^^^^^^^^
```

### Issue: "Too many connections"

**Cause:** Not reusing client factory, creating too many connections

**Solution:**
```go
// Create ONE factory at startup
factory := database.NewClientFactory(config, logger)

// Reuse factory in all handlers
taskHandler := handlers.NewTaskHandler(factory)
```
