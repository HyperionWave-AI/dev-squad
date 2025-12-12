# Code Summarizer Module

A comprehensive code summarization system with intelligent caching, token budget management, and detailed metrics collection.

## Overview

The summarizer module provides LLM-based code summarization with the following capabilities:

- **Intelligent Caching**: LRU cache with TTL support for efficient summary reuse
- **Token Budget Management**: Per-user token limits with graceful degradation
- **Comprehensive Metrics**: Performance tracking, cache statistics, and token usage monitoring
- **Graceful Fallback**: Heuristic summarization when LLM is unavailable or budget exhausted
- **Thread-Safe Operations**: All components use RWMutex for concurrent access

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    LLMSummarizer                             │
│  (Main orchestrator - coordinates all components)            │
└──────────────────────┬──────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
    ┌────────┐  ┌──────────────┐  ┌──────────────┐
    │ Cache  │  │ TokenManager │  │   Metrics    │
    │ (LRU)  │  │  (Budget)    │  │ (Collection) │
    └────────┘  └──────────────┘  └──────────────┘
        │              │              │
        └──────────────┼──────────────┘
                       │
                       ▼
                  ┌──────────────┐
                  │  LLMClient   │
                  │ (OpenAI API) │
                  └──────────────┘
```

## Components

### 1. Cache Module (`cache.go`)

Implements an LRU (Least Recently Used) cache with TTL support.

**Features:**
- O(1) average case performance for Get/Set operations
- Automatic eviction when capacity is reached
- TTL-based expiration of entries
- Detailed statistics (hits, misses, evictions, hit rate)

**Configuration:**
```go
cache := NewLRUCache(
    1000,              // maxSize: maximum entries
    24*time.Hour,      // ttl: time-to-live for entries
    logger,            // logger: zap logger
)
```

**Usage:**
```go
// Store a summary
cache.Set(key, &CodeSummary{...})

// Retrieve a summary
if summary, ok := cache.Get(key); ok {
    // Use cached summary
}

// Get statistics
stats := cache.Stats()
fmt.Printf("Hit rate: %.2f%%\n", stats.HitRate*100)
```

### 2. Token Manager Module (`token_manager.go`)

Manages token budget allocation and enforcement.

**Features:**
- Per-user token budget limits
- Token estimation for code snippets
- Graceful degradation when budget exhausted
- Metrics tracking (total usage, blocked requests, block rate)

**Configuration:**
```go
estimator := &SimpleTokenEstimator{}
tm := NewTokenManager(
    5000,              // budget: total token budget
    100,               // perResult: tokens per summarization
    estimator,         // estimator: token estimation strategy
    logger,            // logger: zap logger
)
```

**Usage:**
```go
// Check if summarization is possible
if tm.CanSummarize(code) {
    // Perform summarization
    tm.RecordUsage(estimatedTokens)
} else {
    // Skip summarization, budget exhausted
    // Fall back to heuristic summarization
}

// Get metrics
metrics := tm.GetMetrics()
fmt.Printf("Block rate: %.2f%%\n", metrics.BlockRate)
```

**Token Estimation:**
- Default: 1 token ≈ 4 characters
- Customizable via `TokenEstimator` interface
- Supports different estimation strategies

### 3. Metrics Module (`metrics.go`)

Collects and tracks comprehensive metrics for summarization operations.

**Features:**
- Summarization type tracking (AI, heuristic, cached)
- Latency tracking with percentile calculations (P95, P99)
- Token usage monitoring
- Cache hit rate tracking
- Error rate monitoring
- Prometheus-compatible metrics export

**Configuration:**
```go
collector := NewMetricsCollector(logger)
```

**Usage:**
```go
// Record a summarization event
collector.RecordSummarization("llm", 150, 250, false)

// Update cache statistics
collector.UpdateCacheStats(0.85, 1024)

// Get metrics
metrics := collector.GetMetrics()
breakdown := collector.GetSummaryBreakdown()
stats := collector.GetPerformanceStats()

// Export Prometheus metrics
promMetrics := collector.PrometheusMetrics()

// Log all metrics
collector.LogMetrics()
```

**Metrics Tracked:**
- Total summarizations
- AI-generated summaries
- Heuristic summaries
- Cached summaries
- Errors and error rate
- Latency (average, P95, P99)
- Token usage
- Cache hit rate
- Cache size

### 4. Summarizer Module (`summarizer.go`)

Main orchestrator that coordinates all components.

**Features:**
- LLM-based code summarization
- Automatic caching with TTL
- Token budget enforcement
- Comprehensive metrics collection
- Graceful error handling

**Configuration:**
```go
config := SummarizerConfig{
    Enabled:             true,
    Model:               "gpt-4",
    MaxTokens:           500,
    CacheEnabled:        true,
    FallbackToHeuristic: true,
    CacheSize:           1000,
    CacheTTL:            24 * time.Hour,
    LLMAPIKey:           "sk-...",
    LLMTimeout:          30 * time.Second,
    TokenBudget:         5000,
    TokenPerResult:      100,
    MetricsEnabled:      true,
}

summarizer, err := NewLLMSummarizer(config, logger)
if err != nil {
    log.Fatal(err)
}
defer summarizer.Close()
```

**Usage:**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

metadata := CodeMetadata{
    FilePath:  "main.go",
    Language:  "go",
    NodeType:  "function",
    NodeName:  "main",
    LineStart: 10,
    LineEnd:   50,
}

summary, err := summarizer.Summarize(ctx, code, metadata)
if err != nil {
    // Handle error - may fall back to heuristic
    log.Printf("Summarization failed: %v", err)
}

// Use summary
fmt.Println(summary.Text)
fmt.Printf("Type: %s, Tokens: %d\n", summary.Type, summary.TokenCount)
```

## Caching Behavior

### Cache Key Generation

Cache keys are generated from:
1. File path
2. Node type (function, class, method, etc.)
3. Node name
4. MD5 hash of code content

This ensures that identical code in different contexts is cached separately.

### TTL and Expiration

- Entries expire after the configured TTL
- Expired entries are automatically removed on access
- Default TTL: 24 hours

### LRU Eviction

When cache reaches capacity:
1. Least recently used entry is identified
2. Entry is removed from cache
3. New entry is added

### Cache Statistics

```go
stats := cache.Stats()
fmt.Printf("Hits: %d\n", stats.Hits)
fmt.Printf("Misses: %d\n", stats.Misses)
fmt.Printf("Evictions: %d\n", stats.Evictions)
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate*100)
fmt.Printf("Size: %d/%d\n", stats.Size, stats.MaxSize)
```

## Token Budget Management

### Budget Allocation

- Total budget: configured per user/session
- Per-result allocation: estimated tokens per summarization
- Remaining budget: total - used

### Budget Enforcement

```
CanSummarize(code):
  1. Estimate tokens for code
  2. Check if estimated <= remaining budget
  3. Return true/false
  4. If false, record blocked request
```

### Graceful Degradation

When budget is exhausted:
1. `CanSummarize()` returns false
2. LLM summarization is skipped
3. Heuristic summarization is used (if enabled)
4. Blocked request is recorded in metrics

### Token Estimation

Default estimation: 1 token ≈ 4 characters

```go
code := "func main() { ... }"
tokens := EstimateTokensForCode(code)
// tokens ≈ len(code) / 4
```

## Metrics and Monitoring

### Metrics Collection

All summarization operations are tracked:

```go
collector.RecordSummarization(
    "llm",      // type: "llm", "heuristic", or "cached"
    150,        // latencyMs
    250,        // tokensUsed
    false,      // cacheHit
)
```

### Performance Statistics

```go
stats := collector.GetPerformanceStats()
fmt.Printf("Average Latency: %.2f ms\n", stats["averageLatencyMs"])
fmt.Printf("P95 Latency: %d ms\n", stats["p95LatencyMs"])
fmt.Printf("P99 Latency: %d ms\n", stats["p99LatencyMs"])
fmt.Printf("Cache Hit Rate: %.2f%%\n", stats["cacheHitRate"].(float64)*100)
```

### Summary Breakdown

```go
breakdown := collector.GetSummaryBreakdown()
fmt.Printf("Total: %d\n", breakdown["total"])
fmt.Printf("AI: %d (%.2f%%)\n", breakdown["ai"], breakdown["aiPercent"])
fmt.Printf("Heuristic: %d (%.2f%%)\n", breakdown["heuristic"], breakdown["heuristicPercent"])
fmt.Printf("Cached: %d (%.2f%%)\n", breakdown["cached"], breakdown["cachedPercent"])
```

### Prometheus Export

```go
promMetrics := collector.PrometheusMetrics()
// Output:
// # HELP summarization_total_count Total number of summaries generated
// # TYPE summarization_total_count counter
// summarization_total_count 1234
// ...
```

## Performance Characteristics

### Cache Performance

- **Get/Set**: O(1) average case
- **Eviction**: O(1) amortized
- **Memory**: O(n) where n = cache size

### Token Manager Performance

- **CanSummarize**: O(1)
- **RecordUsage**: O(1)
- **GetMetrics**: O(1)

### Metrics Collection

- **RecordSummarization**: O(1)
- **GetMetrics**: O(1)
- **PrometheusMetrics**: O(n) where n = number of latency samples

## Configuration Best Practices

### Cache Configuration

```go
// For high-traffic scenarios
CacheSize: 5000,
CacheTTL:  24 * time.Hour,

// For low-traffic scenarios
CacheSize: 100,
CacheTTL:  1 * time.Hour,
```

### Token Budget Configuration

```go
// For generous budgets
TokenBudget:    50000,
TokenPerResult: 500,

// For strict budgets
TokenBudget:    5000,
TokenPerResult: 100,
```

### LLM Configuration

```go
// For high-quality summaries
Model:      "gpt-4",
MaxTokens:  500,
LLMTimeout: 30 * time.Second,

// For fast summaries
Model:      "gpt-3.5-turbo",
MaxTokens:  200,
LLMTimeout: 10 * time.Second,
```

## Error Handling

### Common Errors

1. **Token Budget Exhausted**
   - Cause: Too many summarizations
   - Solution: Increase budget or reduce summarization frequency

2. **LLM Timeout**
   - Cause: LLM API is slow
   - Solution: Increase timeout or use faster model

3. **LLM Error**
   - Cause: API error or invalid request
   - Solution: Check API key, model name, and request format

### Graceful Degradation

When LLM summarization fails:
1. Error is logged
2. Error is recorded in metrics
3. If `FallbackToHeuristic` is enabled, heuristic summarization is used
4. If fallback is disabled, error is returned to caller

## Testing

### Unit Tests

```bash
go test ./hyper/internal/mcp/summarizer -v
```

### Integration Tests

```bash
go test ./hyper/internal/mcp/summarizer -v -run Integration
```

### Benchmarks

```bash
go test ./hyper/internal/mcp/summarizer -bench=. -benchmem
```

## Troubleshooting

### High Cache Miss Rate

**Symptoms:** Cache hit rate < 50%

**Causes:**
- Cache size too small
- TTL too short
- Code changes frequently

**Solutions:**
- Increase `CacheSize`
- Increase `CacheTTL`
- Verify cache key generation

### Token Budget Exhausted

**Symptoms:** Many summarizations fail with "token budget exhausted"

**Causes:**
- Budget too low
- Summaries too long
- Too many summarization requests

**Solutions:**
- Increase `TokenBudget`
- Reduce `MaxTokens`
- Implement request throttling

### High Latency

**Symptoms:** P95/P99 latency > 1000ms

**Causes:**
- LLM API is slow
- Network latency
- LLM timeout too high

**Solutions:**
- Use faster LLM model
- Reduce `LLMTimeout`
- Check network connectivity

## Future Enhancements

1. **Distributed Caching**: Redis support for multi-instance deployments
2. **Advanced Token Estimation**: ML-based token estimation
3. **Adaptive Budgeting**: Dynamic budget allocation based on usage patterns
4. **Custom Summarization Strategies**: Pluggable summarization algorithms
5. **Metrics Persistence**: Store metrics in time-series database
6. **A/B Testing**: Compare different summarization strategies

## References

- [OpenAI API Documentation](https://platform.openai.com/docs)
- [Token Counting](https://platform.openai.com/docs/guides/tokens)
- [LRU Cache](https://en.wikipedia.org/wiki/Cache_replacement_policies#LRU)
- [Prometheus Metrics](https://prometheus.io/docs/concepts/data_model/)
