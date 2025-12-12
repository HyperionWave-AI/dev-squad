# Database Persistence - Code Structure Reference

## File Organization

```
hyper/internal/mcp/summarizer/
├── summarizer.go              (MODIFY)
├── cache.go                   (MODIFY)
├── config.go                  (MODIFY)
├── db_cache.go                (CREATE)
├── db_cache_test.go           (CREATE)
├── llm_client.go
├── token_manager.go
├── metrics.go
├── heuristic.go
├── prompts.go
├── http_server.go
└── handlers.go
```

## Modification Summary

### 1. cache.go - Add Interface Method

**Current**:
```go
type SummaryCache interface {
    Get(key string) (*CodeSummary, bool)
    Set(key string, summary *CodeSummary)
    Delete(key string)
    Clear()
    Stats() CacheStats
}
```

**Modified**:
```go
type SummaryCache interface {
    Get(key string) (*CodeSummary, bool)
    Set(key string, summary *CodeSummary)
    Delete(key string)
    Clear()
    Stats() CacheStats
    ClearExpired() error  // NEW: For database caches
}
```

**LRUCache Implementation**:
```go
// Add to LRUCache struct
func (c *LRUCache) ClearExpired() error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    now := time.Now()
    var keysToDelete []string
    
    for key, entry := range c.cache {
        if c.ttl > 0 && now.After(entry.expiresAt) {
            keysToDelete = append(keysToDelete, key)
        }
    }
    
    for _, key := range keysToDelete {
        entry := c.cache[key]
        c.evictEntry(key, entry)
    }
    
    return nil
}
```

### 2. summarizer.go - Add Database Cache Support

**Add to LLMSummarizer struct**:
```go
type LLMSummarizer struct {
    config         SummarizerConfig
    cache          SummaryCache      // Memory cache (LRU)
    dbCache        SummaryCache      // Database cache (MongoDB) - NEW
    tokenManager   *TokenManager
    metrics        *MetricsCollector
    llmClient      LLMClient
    logger         *zap.Logger
    mu             sync.RWMutex
}
```

**Add new constructor**:
```go
func NewLLMSummarizerWithDB(config SummarizerConfig, db *mongo.Database, logger *zap.Logger) (*LLMSummarizer, error) {
    // Validation
    if err := validateConfig(config); err != nil {
        return nil, fmt.Errorf("invalid summarizer config: %w", err)
    }
    
    if logger == nil {
        logger = zap.NewNop()
    }
    
    // Create LLM client
    llmClient, err := NewLLMClientFromConfig(config, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create LLM client: %w", err)
    }
    
    // Create memory cache
    cache := NewLRUCache(config.CacheSize, config.CacheTTL, logger)
    
    // Create database cache if enabled
    var dbCache SummaryCache
    if config.DatabasePersistenceEnabled && db != nil {
        var err error
        dbCache, err = NewDatabaseCache(db, logger)
        if err != nil {
            logger.Warn("Failed to initialize database cache, continuing without persistence", zap.Error(err))
            // Don't fail - continue without database cache
        }
    }
    
    // Create token manager
    tokenManager := NewTokenManager(config.TokenBudget, config.TokenPerResult, &SimpleTokenEstimator{}, logger)
    
    // Create metrics collector
    metricsCollector := NewMetricsCollector(logger)
    
    return &LLMSummarizer{
        config:         config,
        cache:          cache,
        dbCache:        dbCache,
        tokenManager:   tokenManager,
        metrics:        metricsCollector,
        llmClient:      llmClient,
        logger:         logger,
    }, nil
}
```

**Update Summarize() method**:
```go
func (s *LLMSummarizer) Summarize(ctx context.Context, code string, metadata CodeMetadata) (*CodeSummary, error) {
    if !s.config.Enabled {
        return nil, fmt.Errorf("summarizer is disabled")
    }
    
    startTime := time.Now()
    cacheKey := GenerateCacheKey(code, metadata)
    
    // TIER 1: Check memory cache first (fastest)
    if s.config.CacheEnabled {
        if cached, ok := s.cache.Get(cacheKey); ok {
            s.logger.Debug("Cache hit - memory",
                zap.String("file", metadata.FilePath),
                zap.String("node_type", metadata.NodeType),
            )
            cached.CacheHit = true
            latencyMs := time.Since(startTime).Milliseconds()
            s.metrics.RecordSummarization("cached", latencyMs, cached.TokenCount, true)
            return cached, nil
        }
    }
    
    // TIER 2: Check database cache (if enabled)
    if s.dbCache != nil {
        if cached, ok := s.dbCache.Get(cacheKey); ok {
            s.logger.Debug("Cache hit - database",
                zap.String("file", metadata.FilePath),
                zap.String("node_type", metadata.NodeType),
            )
            // Load into memory cache for faster future access
            if s.config.CacheEnabled {
                s.cache.Set(cacheKey, cached)
            }
            cached.CacheHit = true
            latencyMs := time.Since(startTime).Milliseconds()
            s.metrics.RecordSummarization("db_cached", latencyMs, cached.TokenCount, true)
            return cached, nil
        }
    }
    
    // TIER 3: Check token budget
    if !s.tokenManager.CanSummarize(code) {
        s.logger.Warn("Token budget exhausted, cannot summarize",
            zap.String("file", metadata.FilePath),
        )
        s.metrics.RecordError("token_budget_exhausted")
        return nil, fmt.Errorf("token budget exhausted")
    }
    
    // TIER 4: Generate new summary via LLM
    summaryText, err := s.llmClient.Summarize(ctx, code, metadata)
    if err != nil {
        s.logger.Error("Failed to generate summary with LLM",
            zap.String("file", metadata.FilePath),
            zap.Error(err),
        )
        s.metrics.RecordError("llm_error")
        return nil, fmt.Errorf("failed to generate summary: %w", err)
    }
    
    summary := &CodeSummary{
        Text:        summaryText,
        Type:        "llm",
        TokenCount:  estimateTokenCount(summaryText),
        GeneratedAt: time.Now(),
        CacheHit:    false,
    }
    
    // Record token usage
    s.tokenManager.RecordUsage(summary.TokenCount)
    
    // Store in both caches
    if s.config.CacheEnabled {
        s.cache.Set(cacheKey, summary)
    }
    if s.dbCache != nil {
        s.dbCache.Set(cacheKey, summary)
    }
    
    latencyMs := time.Since(startTime).Milliseconds()
    s.metrics.RecordSummarization("llm", latencyMs, summary.TokenCount, false)
    
    return summary, nil
}
```

### 3. config.go - Add Configuration Options

**Add to SummarizerConfig**:
```go
type SummarizerConfig struct {
    Enabled             bool
    Model               string
    MaxTokens           int
    CacheEnabled        bool
    FallbackToHeuristic bool
    CacheSize           int
    CacheTTL            time.Duration
    LLMAPIKey           string
    LLMTimeout          time.Duration
    TokenBudget         int
    TokenPerResult      int
    MetricsEnabled      bool
    LogLevel            string
    
    // NEW: Database persistence options
    DatabasePersistenceEnabled bool
    DatabaseURL                string
    DatabaseName               string
    CleanupInterval            time.Duration
}
```

## New Files

### db_cache.go Structure

```go
package summarizer

import (
    "context"
    "fmt"
    "time"
    
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.uber.org/zap"
)

// DatabaseCache implements SummaryCache using MongoDB
type DatabaseCache struct {
    collection *mongo.Collection
    logger     *zap.Logger
    stats      cacheStatsInternal
}

// DBCacheSummary represents a summary stored in the database
type DBCacheSummary struct {
    ID        string                 `bson:"_id,omitempty"`
    CacheKey  string                 `bson:"cacheKey"`
    FilePath  string                 `bson:"filePath"`
    NodeType  string                 `bson:"nodeType"`
    NodeName  string                 `bson:"nodeName"`
    CodeHash  string                 `bson:"codeHash"`
    Summary   *CodeSummary           `bson:"summary"`
    Metadata  CodeMetadata           `bson:"metadata"`
    ExpiresAt time.Time              `bson:"expiresAt"`
    CreatedAt time.Time              `bson:"createdAt"`
    UpdatedAt time.Time              `bson:"updatedAt"`
}

// Methods to implement:
// - NewDatabaseCache(db *mongo.Database, logger *zap.Logger) (*DatabaseCache, error)
// - createIndexes(collection *mongo.Collection, logger *zap.Logger) error
// - Get(key string) (*CodeSummary, bool)
// - Set(key string, summary *CodeSummary)
// - Delete(key string)
// - Clear()
// - Stats() CacheStats
// - ClearExpired() error
```

### db_cache_test.go Structure

```go
package summarizer

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.uber.org/zap"
)

// Test functions:
// - TestDatabaseCache_SetAndGet(t *testing.T)
// - TestDatabaseCache_Delete(t *testing.T)
// - TestDatabaseCache_Clear(t *testing.T)
// - TestDatabaseCache_Stats(t *testing.T)
// - TestDatabaseCache_Expiration(t *testing.T)
// - TestDatabaseCache_ClearExpired(t *testing.T)
// - TestDatabaseCache_Upsert(t *testing.T)
```

## Integration Example

```go
// In main.go or initialization code
func initializeSummarizer(logger *zap.Logger) (*summarizer.LLMSummarizer, error) {
    config := summarizer.SummarizerConfig{
        Enabled:                    true,
        Model:                      "gpt-4",
        MaxTokens:                  500,
        CacheEnabled:               true,
        CacheSize:                  1000,
        CacheTTL:                   24 * time.Hour,
        DatabasePersistenceEnabled: true,
        DatabaseURL:                os.Getenv("MONGODB_URI"),
        DatabaseName:               "dev_squad",
        CleanupInterval:            1 * time.Hour,
    }
    
    // Connect to MongoDB
    client, err := mongo.Connect(context.Background(), 
        options.Client().ApplyURI(config.DatabaseURL))
    if err != nil {
        return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
    }
    
    db := client.Database(config.DatabaseName)
    
    // Create summarizer with database persistence
    sum, err := summarizer.NewLLMSummarizerWithDB(config, db, logger)
    if err != nil {
        return nil, fmt.Errorf("failed to create summarizer: %w", err)
    }
    
    // Start cleanup goroutine
    go func() {
        ticker := time.NewTicker(config.CleanupInterval)
        defer ticker.Stop()
        
        for range ticker.C {
            if dbCache, ok := sum.dbCache.(*summarizer.DatabaseCache); ok {
                if err := dbCache.ClearExpired(); err != nil {
                    logger.Error("Failed to clean expired entries", zap.Error(err))
                }
            }
        }
    }()
    
    return sum, nil
}
```

## Testing Checklist

- [ ] Unit tests for DatabaseCache CRUD operations
- [ ] Integration tests with MongoDB
- [ ] Hybrid cache tests (memory + database)
- [ ] TTL and expiration tests
- [ ] Performance benchmarks
- [ ] Failure scenario tests (database unavailable)
- [ ] Concurrent access tests
- [ ] Index creation verification

## Deployment Checklist

- [ ] Create MongoDB collection and indexes
- [ ] Add configuration environment variables
- [ ] Deploy with persistence disabled (default)
- [ ] Monitor for issues
- [ ] Enable persistence in configuration
- [ ] Verify cache hit rates
- [ ] Monitor database size growth
