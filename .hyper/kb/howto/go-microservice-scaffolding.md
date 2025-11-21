# How to Scaffold a New Go Microservice

**Collection:** howto
**Tags:** go, microservices, scaffolding, architecture, backend
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide walks you through creating a new Go 1.25 microservice following Hyperion's architectural patterns. You'll learn how to structure directories, initialize dependencies, and set up the main entry point with proper error handling and configuration.

## Prerequisites

- Go 1.25 or later installed
- Basic understanding of Go modules
- Familiarity with environment variables
- Knowledge of [Component Architecture](../component-architecture.md)

## When to Use This Guide

- Creating a new standalone microservice
- Adding a service to an existing microservices ecosystem
- Setting up a Go backend with HTTP server capabilities
- Building services that integrate with MongoDB, Qdrant, or external APIs

---

## Steps

### Step 1: Initialize Go Module

Create your service directory and initialize the Go module:

```bash
# Create service directory
mkdir -p my-service
cd my-service

# Initialize Go module with your module path
go mod init github.com/your-org/my-service

# Create basic directory structure
mkdir -p cmd/server
mkdir -p internal/{handlers,middleware,services,storage}
mkdir -p pkg/{logger,utils,errors}
```

**Directory Structure Explanation:**
- `cmd/server/` - Main application entry point
- `internal/` - Private application code (not importable by other projects)
  - `handlers/` - HTTP request handlers
  - `middleware/` - HTTP middleware (auth, logging, CORS)
  - `services/` - Business logic layer
  - `storage/` - Data access layer (database clients)
- `pkg/` - Public library code (can be imported by other projects)

### Step 2: Create Main Entry Point

Create `cmd/server/main.go` with proper initialization flow:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

func main() {
    // Step 1: Initialize logger
    logger, err := initLogger()
    if err != nil {
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    defer logger.Sync()

    // Step 2: Load configuration from environment
    config := loadConfig()

    // Step 3: Initialize external dependencies
    storage, err := initStorage(config, logger)
    if err != nil {
        logger.Fatal("Failed to initialize storage", zap.Error(err))
    }
    defer storage.Close()

    // Step 4: Initialize services (business logic)
    services := initServices(storage, logger)

    // Step 5: Set up HTTP router
    router := setupRouter(services, logger)

    // Step 6: Start HTTP server with graceful shutdown
    startServer(router, config.Port, logger)
}

func initLogger() (*zap.Logger, error) {
    env := os.Getenv("ENVIRONMENT")
    if env == "production" {
        return zap.NewProduction()
    }
    return zap.NewDevelopment()
}

func loadConfig() *Config {
    return &Config{
        Port:         getEnv("HTTP_PORT", "8080"),
        MongoURI:     mustGetEnv("MONGODB_URI"),
        DatabaseName: getEnv("MONGODB_DATABASE", "my_service_db"),
    }
}

func setupRouter(services *Services, logger *zap.Logger) *gin.Engine {
    // Set Gin mode
    if os.Getenv("ENVIRONMENT") == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    router := gin.New()
    
    // Global middleware
    router.Use(gin.Recovery())
    router.Use(requestLogger(logger))
    
    // Health check endpoint
    router.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{"status": "healthy"})
    })

    // API routes
    v1 := router.Group("/api/v1")
    {
        // Register your handlers here
        // v1.GET("/users", services.UserHandler.List)
        // v1.POST("/users", services.UserHandler.Create)
    }

    return router
}

func startServer(router *gin.Engine, port string, logger *zap.Logger) {
    srv := &http.Server{
        Addr:           ":" + port,
        Handler:        router,
        ReadTimeout:    10 * time.Second,
        WriteTimeout:   10 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1 MB
    }

    // Start server in goroutine
    go func() {
        logger.Info("Starting HTTP server", zap.String("port", port))
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatal("Failed to start server", zap.Error(err))
        }
    }()

    // Wait for interrupt signal for graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    // Graceful shutdown with 5 second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        logger.Fatal("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
}

// Helper functions
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func mustGetEnv(key string) string {
    value := os.Getenv(key)
    if value == "" {
        log.Fatalf("Environment variable %s is required", key)
    }
    return value
}
```

### Step 3: Define Configuration Structure

Create `internal/config.go` to centralize configuration:

```go
package internal

type Config struct {
    // Server configuration
    Port        string
    Environment string

    // Database configuration
    MongoURI     string
    DatabaseName string

    // External services
    QdrantURL string
    RedisURL  string

    // Authentication
    JWTSecret  string
    EnableJWT  bool
}

// Validate ensures all required configuration is present
func (c *Config) Validate() error {
    if c.MongoURI == "" {
        return fmt.Errorf("MONGODB_URI is required")
    }
    // Add more validation as needed
    return nil
}
```

### Step 4: Create Storage Layer

Create `internal/storage/storage.go` for data access:

```go
package storage

import (
    "context"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.uber.org/zap"
)

type Storage struct {
    client *mongo.Client
    db     *mongo.Database
    logger *zap.Logger
}

func NewStorage(mongoURI, dbName string, logger *zap.Logger) (*Storage, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Configure MongoDB client
    clientOpts := options.Client().
        ApplyURI(mongoURI).
        SetMaxPoolSize(100).
        SetMinPoolSize(10).
        SetMaxConnIdleTime(30 * time.Minute)

    // Connect to MongoDB
    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    // Verify connection
    if err := client.Ping(ctx, nil); err != nil {
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

    logger.Info("Connected to MongoDB", zap.String("database", dbName))

    return &Storage{
        client: client,
        db:     client.Database(dbName),
        logger: logger,
    }, nil
}

func (s *Storage) Close() error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return s.client.Disconnect(ctx)
}

// Collection helper
func (s *Storage) Collection(name string) *mongo.Collection {
    return s.db.Collection(name)
}
```

### Step 5: Add Essential Dependencies

Update `go.mod` with common dependencies:

```bash
# HTTP framework
go get github.com/gin-gonic/gin@latest

# MongoDB driver
go get go.mongodb.org/mongo-driver/mongo@latest

# Structured logging
go get go.uber.org/zap@latest

# JWT authentication
go get github.com/golang-jwt/jwt/v5@latest

# Environment variable loading (optional)
go get github.com/joho/godotenv@latest

# Tidy up dependencies
go mod tidy
```

### Step 6: Create Environment Configuration

Create `.env.example` file:

```bash
# Server Configuration
HTTP_PORT=8080
ENVIRONMENT=development

# Database
MONGODB_URI=mongodb://admin:admin123@localhost:27017/?authSource=admin
MONGODB_DATABASE=my_service_db

# Authentication
JWT_SECRET=your-secret-key-change-in-production
ENABLE_JWT=false

# External Services (optional)
QDRANT_URL=http://localhost:6333
REDIS_URL=redis://localhost:6379
```

Copy to `.env` and customize:
```bash
cp .env.example .env
```

### Step 7: Add Request Logging Middleware

Create `internal/middleware/logger.go`:

```go
package middleware

import (
    "time"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        query := c.Request.URL.RawQuery

        // Process request
        c.Next()

        // Log after request
        duration := time.Since(start)
        
        logger.Info("HTTP request",
            zap.String("method", c.Request.Method),
            zap.String("path", path),
            zap.String("query", query),
            zap.Int("status", c.Writer.Status()),
            zap.Duration("duration", duration),
            zap.String("clientIP", c.ClientIP()),
            zap.String("userAgent", c.Request.UserAgent()),
        )
    }
}
```

### Step 8: Create Makefile for Build Automation

Create `Makefile` in project root:

```makefile
.PHONY: help build run dev test lint clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

build: ## Build the service binary
	@echo "Building service..."
	go build -o bin/server ./cmd/server
	@echo "✓ Build complete: bin/server"

run: build ## Build and run the service
	@echo "Starting service..."
	./bin/server

dev: ## Run with hot reload (requires air)
	air

test: ## Run tests
	go test -v -race -cover ./...

lint: ## Run linter
	golangci-lint run ./...

clean: ## Remove build artifacts
	rm -rf bin/
	go clean -cache

install: ## Install dependencies
	go mod download
	go mod tidy
```

### Step 9: Add Air Configuration for Hot Reload

Create `.air.toml` for development hot reload:

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_error = true

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false
```

### Step 10: Run and Verify

Start your service:

```bash
# Development mode with hot reload
make dev

# Or build and run
make run

# Test health endpoint
curl http://localhost:8080/health
```

Expected response:
```json
{"status": "healthy"}
```

---

## Best Practices

### 1. Fail-Fast Error Handling
```go
// ✅ GOOD - Fail immediately on critical errors
if err := initDatabase(); err != nil {
    logger.Fatal("Database initialization failed", zap.Error(err))
}

// ❌ BAD - Silently ignoring errors
if err := initDatabase(); err != nil {
    logger.Warn("Database failed, continuing anyway")
}
```

### 2. Graceful Shutdown
Always implement graceful shutdown to:
- Finish processing in-flight requests
- Close database connections cleanly
- Release system resources properly

### 3. Configuration Validation
Validate configuration at startup before initializing dependencies:
```go
if err := config.Validate(); err != nil {
    logger.Fatal("Invalid configuration", zap.Error(err))
}
```

### 4. Structured Logging
Use structured logging with zap for:
- Better debugging in production
- Easy log aggregation and querying
- Performance (zap is extremely fast)

### 5. Environment-Based Configuration
- Use environment variables for all configuration
- Never hardcode secrets or URLs
- Provide sensible defaults where appropriate

---

## Common Pitfalls

### 1. Not Handling Context Timeouts
```go
// ✅ GOOD - Use timeouts
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// ❌ BAD - Using background context without timeout
ctx := context.Background()
```

### 2. Ignoring Graceful Shutdown
Without graceful shutdown, in-flight requests will be terminated abruptly when the service stops.

### 3. Not Closing Resources
Always defer Close() for:
- Database connections
- File handles
- HTTP clients
- Loggers

### 4. Mixing Concerns
Keep separation between:
- Handlers (HTTP layer)
- Services (business logic)
- Storage (data access)

---

## Related Documentation

- [Component Architecture](../component-architecture.md) - Overall system structure
- [MongoDB Integration](../mongodb-integration.md) - Database setup
- [JWT Authentication](./jwt-authentication-middleware.md) - Auth middleware
- [REST API Patterns](./rest-api-endpoint-patterns.md) - Endpoint design

---

## Troubleshooting

### Issue: "Cannot connect to MongoDB"

**Cause:** MongoDB not running or wrong connection string

**Solution:**
```bash
# Check MongoDB is running
docker ps | grep mongo

# Test connection with mongosh
mongosh "mongodb://admin:admin123@localhost:27017"

# Verify MONGODB_URI in .env
echo $MONGODB_URI
```

### Issue: "Port already in use"

**Cause:** Another process is using the configured port

**Solution:**
```bash
# Find process using the port
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use a different port
export HTTP_PORT=8081
```

### Issue: "Module not found"

**Cause:** Dependencies not installed

**Solution:**
```bash
# Download all dependencies
go mod download

# Clean and re-download
go clean -modcache
go mod tidy
```
