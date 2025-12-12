# HTTP API Architecture

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Clients                              │
│  (cURL, Python, JavaScript, External Services, etc.)            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    HTTP Server (Gin)                             │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Middleware Stack                                         │   │
│  │ ├─ Error Handling (panic recovery)                       │   │
│  │ ├─ Request ID Generation                                │   │
│  │ ├─ Structured Logging                                   │   │
│  │ └─ CORS Support                                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                         │                                        │
│  ┌──────────────────────┴──────────────────────────────────┐   │
│  │ Route Handlers                                           │   │
│  │ ├─ POST /api/summarize                                  │   │
│  │ ├─ POST /api/summarize/batch                            │   │
│  │ ├─ GET /api/metrics                                     │   │
│  │ ├─ GET /api/health                                      │   │
│  │ ├─ GET /api/ready                                       │   │
│  │ ├─ GET /api/live                                        │   │
│  │ └─ GET /                                                │   │
│  └──────────────────────────────────────────────────────────┘   │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                  SummarizerHandlers                              │
│  ├─ Request Validation                                          │
│  ├─ Error Handling                                              │
│  ├─ Response Formatting                                         │
│  └─ Logging                                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                  CodeSummarizer Interface                        │
│  (Implemented by LLMSummarizer)                                 │
│  ├─ Summarize(ctx, code, metadata) → CodeSummary              │
│  └─ Close() → error                                            │
└────────────────────────┬────────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ LLM Client   │  │ Cache System │  │ Token Manager│
│ (OpenAI,    │  │ (LRU Cache)  │  │ (Budget)     │
│  Claude)    │  │              │  │              │
└──────────────┘  └──────────────┘  └──────────────┘
        │                │                │
        ▼                ▼                ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐
│ LLM APIs     │  │ In-Memory    │  │ Per-User     │
│              │  │ Storage      │  │ Token Budget │
└──────────────┘  └──────────────┘  └──────────────┘
```

## Request Flow

### Single Summarization Flow

```
1. Client Request
   POST /api/summarize
   {
     "code": "...",
     "metadata": {...}
   }
   │
   ▼
2. HTTP Handler (HandleSummarize)
   ├─ Parse JSON
   ├─ Validate request
   ├─ Generate/track request ID
   │
   ▼
3. SummarizerHandlers
   ├─ Call summarizer.Summarize()
   │
   ▼
4. LLMSummarizer
   ├─ Check cache (cache hit?)
   │  └─ Return cached result
   ├─ Check token budget
   ├─ Call LLM client
   ├─ Record token usage
   ├─ Cache result
   │
   ▼
5. Response
   {
     "success": true,
     "summary": {...},
     "requestId": "..."
   }
```

### Batch Summarization Flow

```
1. Client Request
   POST /api/summarize/batch
   {
     "items": [
       {"id": "item1", "code": "..."},
       {"id": "item2", "code": "..."}
     ]
   }
   │
   ▼
2. HTTP Handler (HandleBatchSummarize)
   ├─ Parse JSON
   ├─ Validate batch request
   ├─ Validate each item
   │
   ▼
3. Process Each Item
   ├─ Item 1
   │  ├─ Call summarizer.Summarize()
   │  ├─ Record result/error
   │  └─ Track duration
   ├─ Item 2
   │  ├─ Call summarizer.Summarize()
   │  ├─ Record result/error
   │  └─ Track duration
   └─ ...
   │
   ▼
4. Aggregate Results
   ├─ Collect successful results
   ├─ Collect errors
   ├─ Calculate statistics
   │
   ▼
5. Response
   {
     "success": true/false,
     "results": [...],
     "errors": [...],
     "statistics": {...}
   }
```

## Data Flow

### Request Types

```
SummarizeRequest
├─ code: string (required, max 100KB)
├─ metadata: CodeMetadata (optional)
│  ├─ filePath: string
│  ├─ language: string
│  ├─ nodeType: string
│  ├─ nodeName: string
│  ├─ signature: string
│  ├─ docContent: string
│  ├─ lineStart: int
│  └─ lineEnd: int
└─ userId: string (optional)

BatchSummarizeRequest
├─ items: []BatchItem (required, 1-100 items)
│  ├─ id: string (required, unique)
│  ├─ code: string (required, max 100KB)
│  └─ metadata: CodeMetadata (optional)
└─ userId: string (optional)
```

### Response Types

```
SummarizeResponse
├─ success: bool
├─ summary: CodeSummary (if success)
│  ├─ text: string
│  ├─ type: string ("llm", "heuristic", "cached")
│  ├─ tokenCount: int
│  ├─ generatedAt: time.Time
│  └─ cacheHit: bool
├─ error: string (if error)
├─ timestamp: time.Time
└─ requestId: string

BatchSummarizeResponse
├─ success: bool
├─ results: []BatchSummarizeResult
│  ├─ id: string
│  ├─ summary: CodeSummary (if success)
│  ├─ success: bool
│  ├─ error: string (if error)
│  └─ durationMs: int64
├─ errors: []BatchError
│  ├─ id: string
│  └─ error: string
├─ statistics: BatchStatistics
│  ├─ totalItems: int
│  ├─ successfulItems: int
│  ├─ failedItems: int
│  ├─ totalTokens: int
│  ├─ totalDurationMs: int64
│  └─ averageDurationMs: float64
├─ timestamp: time.Time
└─ requestId: string
```

## Error Handling Strategy

```
┌─────────────────────────────────────────┐
│ HTTP Request                             │
└────────────────┬────────────────────────┘
                 │
                 ▼
         ┌───────────────┐
         │ Parse JSON    │
         └───────┬───────┘
                 │
         ┌───────▼────────┐
         │ Valid JSON?    │
         ├────────────────┤
         │ No  ──────────→ 400 Bad Request
         │ Yes            (INVALID_REQUEST)
         └───────┬────────┘
                 │
         ┌───────▼────────┐
         │ Validate Data  │
         └───────┬───────┘
                 │
         ┌───────▼────────┐
         │ Valid Data?    │
         ├────────────────┤
         │ No  ──────────→ 400 Bad Request
         │ Yes            (VALIDATION_ERROR)
         └───────┬────────┘
                 │
         ┌───────▼────────┐
         │ Process        │
         └───────┬───────┘
                 │
         ┌───────▼────────┐
         │ Success?       │
         ├────────────────┤
         │ Yes ──────────→ 200 OK
         │ No             (with result)
         └───────┬────────┘
                 │
         ┌───────▼────────┐
         │ Error Type?    │
         ├────────────────┤
         │ Budget ───────→ 400 Bad Request
         │ Timeout ──────→ 500 Internal Error
         │ Other ────────→ 500 Internal Error
         └────────────────┘
```

## Concurrency Model

```
┌──────────────────────────────────────────┐
│ HTTP Server (Gin)                        │
│ ├─ Goroutine per request                │
│ └─ Thread-safe handlers                 │
└──────────────────────────────────────────┘
         │
         ├─ Request 1 ──→ Handler 1 ──→ Summarizer
         ├─ Request 2 ──→ Handler 2 ──→ Summarizer
         ├─ Request 3 ──→ Handler 3 ──→ Summarizer
         └─ Request N ──→ Handler N ──→ Summarizer
                                │
                                ▼
                    ┌──────────────────────┐
                    │ Thread-Safe Cache    │
                    │ (RWMutex protected)  │
                    └──────────────────────┘
                                │
                                ▼
                    ┌──────────────────────┐
                    │ Thread-Safe Metrics  │
                    │ (Atomic operations)  │
                    └──────────────────────┘
```

## Performance Characteristics

### Latency

```
Single Request Latency:
├─ JSON Parsing: ~1ms
├─ Validation: ~1ms
├─ Cache Lookup: ~0.1ms (hit) or ~200ms (miss)
├─ LLM Call: ~200-500ms
├─ Response Formatting: ~1ms
└─ Total: ~200-500ms (cache hit) or ~400-700ms (cache miss)

Batch Request Latency:
├─ JSON Parsing: ~2ms
├─ Validation: ~5ms
├─ Per-Item Processing: ~200-500ms each
├─ Aggregation: ~5ms
└─ Total: ~(200-500ms × N) + 12ms
```

### Memory Usage

```
Per Request:
├─ Request struct: ~1KB
├─ Response struct: ~2KB
├─ Temporary buffers: ~10KB
└─ Total: ~13KB

Cache (1000 items):
├─ Per entry: ~5KB average
├─ Total: ~5MB
└─ With overhead: ~6MB

Metrics:
├─ Per metric: ~100 bytes
├─ Total: ~50KB
└─ Negligible
```

## Scalability

### Horizontal Scaling

```
Load Balancer
    │
    ├─ Instance 1 (Port 8080)
    ├─ Instance 2 (Port 8080)
    ├─ Instance 3 (Port 8080)
    └─ Instance N (Port 8080)

Each instance:
├─ Independent cache
├─ Independent metrics
└─ Shared LLM API quota
```

### Vertical Scaling

```
Single Instance Capacity:
├─ Concurrent requests: Limited by goroutines
├─ Memory: ~6MB cache + ~13KB per request
├─ CPU: Depends on LLM latency (mostly I/O bound)
└─ Throughput: ~10-20 requests/sec (LLM limited)
```

## Integration Points

### With Existing Components

```
HTTP API
    │
    ├─→ CodeSummarizer Interface
    │   └─→ LLMSummarizer
    │       ├─→ LLMClient (OpenAI, Claude)
    │       ├─→ SummaryCache (LRU)
    │       ├─→ TokenManager
    │       └─→ MetricsCollector
    │
    ├─→ Gin Framework
    │   └─→ HTTP routing & middleware
    │
    └─→ Logging (zap)
        └─→ Structured logging
```

## Future Enhancements

```
HTTP API v2.0
├─ Database Persistence
│  └─ Store summaries for retrieval
├─ Rate Limiting
│  └─ Per-user/IP rate limits
├─ Circuit Breaker
│  └─ Graceful degradation
├─ Distributed Caching
│  └─ Redis integration
├─ Async Batch Processing
│  └─ Job queue system
├─ Distributed Tracing
│  └─ OpenTelemetry integration
├─ WebSocket Support
│  └─ Real-time streaming
└─ GraphQL API
   └─ Alternative query interface
```
