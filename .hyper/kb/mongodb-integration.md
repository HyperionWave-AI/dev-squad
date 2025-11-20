# MongoDB Database Integration

**Collection:** infrastructure
**Tags:** mongodb, database, security, connection-pooling
**Version:** 1.0
**Last Updated:** 2025-11-20

---

## Overview

Hyperion uses MongoDB as the primary document database for storing structured data including tasks, knowledge entries, code index metadata, and reflection data. This document describes MongoDB configuration, connection management, and security model.

## Database Initialization

### Location
**File:** `/hyper/cmd/coordinator/main.go` (Lines 294-300)

### Connection Setup

```go
func initializeDatabase() (*mongo.Client, error) {
    mongoURI := os.Getenv("MONGODB_URI")
    if mongoURI == "" {
        return nil, fmt.Errorf("MONGODB_URI environment variable is required")
    }

    mongoDatabase := os.Getenv("MONGODB_DATABASE")
    if mongoDatabase == "" {
        mongoDatabase = "hyper_db" // Default database name
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }

    // Verify connection
    err = client.Ping(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
    }

    return client, nil
}
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MONGODB_URI` | **Yes** | - | MongoDB connection string |
| `MONGODB_DATABASE` | No | `hyper_db` | Database name |

### Connection String Format

#### Local Development
```bash
MONGODB_URI=mongodb://admin:admin123@localhost:27017/?authSource=admin
```

#### Production with Replica Set
```bash
MONGODB_URI=mongodb://user:password@mongo1:27017,mongo2:27017,mongo3:27017/hyper_db?replicaSet=rs0&authSource=admin
```

#### MongoDB Atlas (Cloud)
```bash
MONGODB_URI=mongodb+srv://username:password@cluster0.mongodb.net/hyper_db?retryWrites=true&w=majority
```

### Connection Options

#### Recommended Settings

```go
clientOptions := options.Client().ApplyURI(mongoURI).
    SetMaxPoolSize(100).                    // Connection pool size
    SetMinPoolSize(10).                     // Minimum connections
    SetMaxConnIdleTime(30 * time.Minute).   // Idle connection timeout
    SetServerSelectionTimeout(5 * time.Second). // Server selection timeout
    SetConnectTimeout(10 * time.Second)     // Initial connection timeout
```

## Security Model

### JWT Identity-Based Access

**Requirement:** All MongoDB operations MUST use user JWT identity, not system service identities.

#### Implementation Pattern

```go
// ✅ GOOD - User identity from JWT
func (h *Handler) GetUserData(c *gin.Context) {
    userID := c.GetString("userId")  // From JWT middleware
    companyID := c.GetString("companyId")

    // Use secure client with user context
    client := database.NewSecureMongoClient(userID, companyID)

    // Query with user filtering
    filter := bson.M{
        "userId": userID,
        "companyId": companyID,
    }

    result, err := client.FindOne(ctx, "user_data", filter)
}

// ❌ BAD - System-level access without user context
func (h *Handler) GetUserData(c *gin.Context) {
    // Direct database access without user identity
    result, err := db.Collection("user_data").FindOne(ctx, bson.M{})
}
```

### JWT Middleware

**File:** `/hyper/internal/middleware/jwt_auth.go`

```go
func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractToken(c)

        claims, err := parseJWT(token)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "Invalid token"})
            return
        }

        // Set user context for downstream handlers
        c.Set("userId", claims.UserID)
        c.Set("companyId", claims.CompanyID)

        c.Next()
    }
}
```

### Development Mode

**Optional:** Bypass JWT validation for local development

```go
// Enable with environment variable
devMode := os.Getenv("DEV_MODE") == "true"

if devMode {
    // Use default user ID for development
    c.Set("userId", "dev-user-id")
    c.Set("companyId", "dev-company-id")
}
```

## Collections Overview

### Knowledge Base Collections

| Collection | Purpose | Document Count (Est.) |
|------------|---------|----------------------|
| `collections` | Knowledge collection metadata | 10-50 |
| `knowledge_entries` | Individual knowledge entries | 1,000-100,000 |
| `knowledge_votes` | User votes on entries | 5,000-500,000 |

### Task Management Collections

| Collection | Purpose | Document Count (Est.) |
|------------|---------|----------------------|
| `human_tasks` | User-created tasks | 100-10,000 |
| `agent_tasks` | AI agent task assignments | 500-50,000 |

### Code Index Collections

| Collection | Purpose | Document Count (Est.) |
|------------|---------|----------------------|
| `indexed_folders` | Folder index configuration | 10-100 |
| `indexed_files` | File metadata | 10,000-1,000,000 |
| `file_chunks` | Code chunks with AST metadata | 100,000-10,000,000 |

### Reflection (Metacognitive) Collections

| Collection | Purpose | Document Count (Est.) |
|------------|---------|----------------------|
| `reflections` | Decision and outcome tracking | 1,000-100,000 |
| `error_patterns` | Recurring error pattern detection | 100-10,000 |

## Connection Pooling

### Pool Configuration

**Default Settings:**
```go
MaxPoolSize:         100  // Maximum connections
MinPoolSize:         10   // Minimum idle connections
MaxConnIdleTime:     30m  // Close idle connections after 30 minutes
ServerSelectionTimeout: 5s  // Timeout for server selection
```

### Pool Monitoring

```go
func monitorConnectionPool(client *mongo.Client) {
    poolMonitor := &event.PoolMonitor{
        Event: func(e *event.PoolEvent) {
            switch e.Type {
            case event.PoolCreated:
                log.Info("Connection pool created")
            case event.ConnectionCreated:
                log.Debug("New connection created")
            case event.ConnectionClosed:
                log.Debug("Connection closed")
            case event.GetFailed:
                log.Warn("Failed to get connection from pool")
            }
        },
    }

    // Attach monitor during client creation
    clientOptions.SetPoolMonitor(poolMonitor)
}
```

## Query Patterns

### Common Operations

#### Insert with Timestamps

```go
doc := bson.M{
    "taskId":    taskID,
    "status":    "pending",
    "createdAt": time.Now().UTC(),
    "updatedAt": time.Now().UTC(),
}

result, err := collection.InsertOne(ctx, doc)
```

#### Update with Timestamp

```go
update := bson.M{
    "$set": bson.M{
        "status":    "completed",
        "updatedAt": time.Now().UTC(),
    },
}

filter := bson.M{"taskId": taskID}
result, err := collection.UpdateOne(ctx, filter, update)
```

#### Upsert Pattern

```go
filter := bson.M{
    "entryId": entryID,
    "userId":  userID,
}

update := bson.M{
    "$set": bson.M{
        "vote":      "+",
        "updatedAt": time.Now().UTC(),
    },
    "$setOnInsert": bson.M{
        "createdAt": time.Now().UTC(),
    },
}

opts := options.Update().SetUpsert(true)
result, err := collection.UpdateOne(ctx, filter, update, opts)
```

#### Aggregation Pipeline

```go
pipeline := []bson.M{
    {
        "$match": bson.M{
            "status": "completed",
            "createdAt": bson.M{
                "$gte": startDate,
                "$lte": endDate,
            },
        },
    },
    {
        "$group": bson.M{
            "_id":   "$agentName",
            "count": bson.M{"$sum": 1},
        },
    },
    {
        "$sort": bson.M{"count": -1},
    },
}

cursor, err := collection.Aggregate(ctx, pipeline)
```

## Indexing Strategy

### Automatic Index Creation

```go
func ensureIndexes(db *mongo.Database) error {
    // Knowledge entries
    _, err := db.Collection("knowledge_entries").Indexes().CreateMany(ctx, []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "entryId", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{{Key: "collectionId", Value: 1}},
        },
        {
            Keys: bson.D{
                {Key: "taskId", Value: 1},
            },
            Options: options.Index().SetSparse(true),
        },
    })

    return err
}
```

### Index Best Practices

1. **Unique Indexes:** Prevent duplicate entries
2. **Compound Indexes:** Support multi-field queries
3. **Sparse Indexes:** Allow null values for optional fields
4. **Text Indexes:** Enable full-text search

## Error Handling

### Common Errors

#### Connection Failures

```go
if err := client.Ping(ctx, nil); err != nil {
    if mongoErr, ok := err.(mongo.CommandError); ok {
        switch mongoErr.Code {
        case 13: // Unauthorized
            return fmt.Errorf("authentication failed: %w", err)
        case 6: // HostUnreachable
            return fmt.Errorf("cannot reach MongoDB: %w", err)
        default:
            return fmt.Errorf("MongoDB error: %w", err)
        }
    }
}
```

#### Duplicate Key Errors

```go
if mongo.IsDuplicateKeyError(err) {
    // Handle duplicate key gracefully
    log.Warn("Duplicate entry detected", "key", entryID)
    return ErrAlreadyExists
}
```

#### Write Concern Errors

```go
if writeConcernErr, ok := err.(mongo.WriteConcernError); ok {
    log.Error("Write concern not satisfied",
        "code", writeConcernErr.Code,
        "message", writeConcernErr.Message)
}
```

## Transaction Support

### Multi-Document Transactions

```go
func transferData(ctx context.Context, client *mongo.Client) error {
    session, err := client.StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
        // Operation 1
        if _, err := collection1.UpdateOne(sessCtx, filter1, update1); err != nil {
            return nil, err
        }

        // Operation 2
        if _, err := collection2.InsertOne(sessCtx, document2); err != nil {
            return nil, err
        }

        return nil, nil
    }

    // Execute transaction with retry
    _, err = session.WithTransaction(ctx, callback)
    return err
}
```

**Note:** Transactions require MongoDB replica set or sharded cluster.

## Health Checks

### Database Health Endpoint

```go
func (h *HealthHandler) CheckDatabase(c *gin.Context) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err := h.db.Client().Ping(ctx, nil)
    if err != nil {
        c.JSON(503, gin.H{
            "status": "unhealthy",
            "error":  err.Error(),
        })
        return
    }

    c.JSON(200, gin.H{
        "status": "healthy",
        "database": "mongodb",
    })
}
```

## Performance Optimization

### 1. Use Projection to Limit Fields

```go
// Only fetch required fields
opts := options.FindOne().SetProjection(bson.M{
    "taskId": 1,
    "status": 1,
    "_id":    0, // Exclude _id
})

result := collection.FindOne(ctx, filter, opts)
```

### 2. Batch Operations

```go
// Batch insert for efficiency
documents := make([]interface{}, len(entries))
for i, entry := range entries {
    documents[i] = entry
}

opts := options.InsertMany().SetOrdered(false)
result, err := collection.InsertMany(ctx, documents, opts)
```

### 3. Cursor Iteration for Large Datasets

```go
cursor, err := collection.Find(ctx, filter)
if err != nil {
    return err
}
defer cursor.Close(ctx)

for cursor.Next(ctx) {
    var doc Document
    if err := cursor.Decode(&doc); err != nil {
        log.Error("Failed to decode document", "error", err)
        continue
    }

    // Process document
    processDo(doc)
}
```

### 4. Index Usage Analysis

```go
// Use explain to analyze query performance
explainResult := bson.M{}
err := collection.Database().RunCommand(ctx, bson.D{
    {Key: "explain", Value: bson.D{
        {Key: "find", Value: "collection_name"},
        {Key: "filter", Value: filter},
    }},
}).Decode(&explainResult)

// Check if index is used
executionStats := explainResult["executionStats"]
```

## Backup and Restore

### Backup Strategy

```bash
# Full database backup
mongodump --uri="mongodb://admin:password@localhost:27017" \
    --db=hyper_db \
    --out=/backup/$(date +%Y%m%d)

# Backup specific collection
mongodump --uri="mongodb://admin:password@localhost:27017" \
    --db=hyper_db \
    --collection=knowledge_entries \
    --out=/backup/$(date +%Y%m%d)
```

### Restore Strategy

```bash
# Restore full database
mongorestore --uri="mongodb://admin:password@localhost:27017" \
    --db=hyper_db \
    /backup/20251120/hyper_db

# Restore specific collection
mongorestore --uri="mongodb://admin:password@localhost:27017" \
    --db=hyper_db \
    --collection=knowledge_entries \
    /backup/20251120/hyper_db/knowledge_entries.bson
```

## Related Documents

- [MongoDB Schemas](./mongodb-schemas.md) - Detailed collection schemas
- [Configuration Reference](./configuration-reference.md) - Environment variables
- [Component Architecture](./component-architecture.md) - System integration

## Troubleshooting

### Issue: "Connection timeout"

**Solution:**
```bash
# Check MongoDB is running
docker ps | grep mongo

# Test connection
mongosh "mongodb://admin:admin123@localhost:27017"

# Verify network connectivity
nc -zv localhost 27017
```

### Issue: "Authentication failed"

**Solution:**
```bash
# Verify credentials
mongosh "mongodb://admin:admin123@localhost:27017/?authSource=admin"

# Check auth source in connection string
MONGODB_URI=mongodb://admin:password@localhost:27017/?authSource=admin
```

### Issue: "Too many connections"

**Solution:**
```go
// Reduce max pool size
clientOptions.SetMaxPoolSize(50)

// Check current connections
db.adminCommand({serverStatus: 1}).connections
```

### Issue: "Slow queries"

**Solution:**
```bash
# Enable profiling
db.setProfilingLevel(1, { slowms: 100 })

# Check slow queries
db.system.profile.find().sort({ts: -1}).limit(10)

# Analyze with explain
db.collection.find(query).explain("executionStats")
```
