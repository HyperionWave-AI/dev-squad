# Code Summarizer HTTP API

## Overview

The Code Summarizer HTTP API provides REST endpoints for direct code summarization without needing to use the coordinator tools. This is a critical component that enables external services to integrate with the summarizer.

## Features

✅ **Single Code Summarization** - POST /api/summarize
✅ **Batch Processing** - POST /api/summarize/batch (up to 100 items)
✅ **Performance Metrics** - GET /api/metrics
✅ **Health Checks** - GET /api/health, GET /api/ready, GET /api/live
✅ **Request Tracking** - Automatic request ID generation and tracking
✅ **Error Handling** - Comprehensive error responses with proper HTTP status codes
✅ **CORS Support** - Cross-origin requests enabled
✅ **Comprehensive Testing** - Full unit test coverage

## API Endpoints

### 1. Single Code Summarization

**Endpoint:** `POST /api/summarize`

**Request:**
```json
{
  "code": "func hello() { println(\"Hello\") }",
  "metadata": {
    "filePath": "main.go",
    "language": "go",
    "nodeType": "function",
    "nodeName": "hello",
    "signature": "func hello()",
    "lineStart": 1,
    "lineEnd": 3
  },
  "userId": "user123"
}
```

**Response (Success):**
```json
{
  "success": true,
  "summary": {
    "text": "This function prints 'Hello' to the console.",
    "type": "llm",
    "tokenCount": 12,
    "generatedAt": "2025-11-25T16:36:41Z",
    "cacheHit": false
  },
  "timestamp": "2025-11-25T16:36:41Z",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Response (Error):**
```json
{
  "success": false,
  "error": "code exceeds maximum size of 100KB",
  "timestamp": "2025-11-25T16:36:41Z",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Status Codes:**
- `200 OK` - Summarization successful
- `400 Bad Request` - Invalid request or validation error
- `500 Internal Server Error` - Server error during summarization

### 2. Batch Code Summarization

**Endpoint:** `POST /api/summarize/batch`

**Request:**
```json
{
  "items": [
    {
      "id": "item1",
      "code": "func hello() {}",
      "metadata": {
        "filePath": "main.go",
        "language": "go"
      }
    },
    {
      "id": "item2",
      "code": "func world() {}",
      "metadata": {
        "filePath": "main.go",
        "language": "go"
      }
    }
  ],
  "userId": "user123"
}
```

**Response:**
```json
{
  "success": true,
  "results": [
    {
      "id": "item1",
      "summary": {
        "text": "Empty hello function.",
        "type": "llm",
        "tokenCount": 5,
        "generatedAt": "2025-11-25T16:36:41Z",
        "cacheHit": false
      },
      "success": true,
      "durationMs": 245
    },
    {
      "id": "item2",
      "summary": {
        "text": "Empty world function.",
        "type": "llm",
        "tokenCount": 5,
        "generatedAt": "2025-11-25T16:36:41Z",
        "cacheHit": false
      },
      "success": true,
      "durationMs": 198
    }
  ],
  "errors": [],
  "statistics": {
    "totalItems": 2,
    "successfulItems": 2,
    "failedItems": 0,
    "totalTokens": 10,
    "totalDurationMs": 443,
    "averageDurationMs": 221.5
  },
  "timestamp": "2025-11-25T16:36:41Z",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Constraints:**
- Maximum 100 items per batch
- Each item code limited to 100KB
- Duplicate item IDs not allowed

**Status Codes:**
- `200 OK` - Batch processing completed (even if some items failed)
- `400 Bad Request` - Invalid request or validation error

### 3. Performance Metrics

**Endpoint:** `GET /api/metrics`

**Response:**
```json
{
  "success": true,
  "metrics": {
    "totalSummarizations": 42,
    "successfulSummarizations": 40,
    "failedSummarizations": 2,
    "averageLatencyMs": 234.5,
    "p95LatencyMs": 450,
    "p99LatencyMs": 890,
    "totalTokensUsed": 4200,
    "averageTokensPerSummary": 105
  },
  "cache": {
    "hits": 15,
    "misses": 27,
    "hitRate": 0.357,
    "size": 12,
    "maxSize": 1000,
    "evictions": 0
  },
  "tokens": {
    "budgetPerUser": 5000,
    "usedByUser": 2100,
    "remainingBudget": 2900,
    "resetTime": "2025-11-26T00:00:00Z"
  },
  "timestamp": "2025-11-25T16:36:41Z"
}
```

**Status Codes:**
- `200 OK` - Metrics retrieved successfully

### 4. Health Check

**Endpoint:** `GET /api/health`

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-25T16:36:41Z",
  "uptimeSeconds": 3600,
  "version": "1.0.0",
  "checks": {
    "summarizer": "ok"
  }
}
```

**Status Codes:**
- `200 OK` - Service is healthy

### 5. Readiness Probe

**Endpoint:** `GET /api/ready`

**Response (Ready):**
```json
{
  "ready": true,
  "timestamp": "2025-11-25T16:36:41Z"
}
```

**Response (Not Ready):**
```json
{
  "ready": false,
  "message": "Summarizer not initialized",
  "timestamp": "2025-11-25T16:36:41Z"
}
```

**Status Codes:**
- `200 OK` - Service is ready
- `503 Service Unavailable` - Service is not ready

### 6. Liveness Probe

**Endpoint:** `GET /api/live`

**Response:**
```json
{
  "alive": true,
  "timestamp": "2025-11-25T16:36:41Z"
}
```

**Status Codes:**
- `200 OK` - Service is alive

### 7. Service Information

**Endpoint:** `GET /`

**Response:**
```json
{
  "service": "code-summarizer",
  "version": "1.0.0",
  "endpoints": {
    "health": "GET /api/health",
    "ready": "GET /api/ready",
    "live": "GET /api/live",
    "summarize": "POST /api/summarize",
    "batch": "POST /api/summarize/batch",
    "metrics": "GET /api/metrics"
  }
}
```

## Request Headers

### Optional Headers

- **X-Request-ID** - Custom request ID for tracking. If not provided, a UUID will be generated.
  ```
  X-Request-ID: my-custom-request-id-123
  ```

### Response Headers

- **X-Request-ID** - Echo of the request ID for tracking
  ```
  X-Request-ID: my-custom-request-id-123
  ```

## Error Handling

All error responses follow a consistent format:

```json
{
  "error": "Error message",
  "message": "Detailed error description",
  "code": "ERROR_CODE",
  "timestamp": "2025-11-25T16:36:41Z",
  "requestId": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Common Error Codes

- `INVALID_REQUEST` - Malformed JSON or missing required fields
- `VALIDATION_ERROR` - Request validation failed
- `INTERNAL_ERROR` - Server error during processing
- `NOT_FOUND` - Endpoint not found
- `TOKEN_BUDGET_EXHAUSTED` - User token budget exceeded

## Usage Examples

### cURL

**Single Summarization:**
```bash
curl -X POST http://localhost:8080/api/summarize \
  -H "Content-Type: application/json" \
  -d '{
    "code": "func hello() { println(\"Hello\") }",
    "metadata": {
      "filePath": "main.go",
      "language": "go"
    }
  }'
```

**Batch Summarization:**
```bash
curl -X POST http://localhost:8080/api/summarize/batch \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {
        "id": "item1",
        "code": "func hello() {}"
      },
      {
        "id": "item2",
        "code": "func world() {}"
      }
    ]
  }'
```

**Get Metrics:**
```bash
curl http://localhost:8080/api/metrics
```

**Health Check:**
```bash
curl http://localhost:8080/api/health
```

### Python

```python
import requests
import json

# Single summarization
response = requests.post(
    'http://localhost:8080/api/summarize',
    json={
        'code': 'func hello() { println("Hello") }',
        'metadata': {
            'filePath': 'main.go',
            'language': 'go'
        }
    }
)
print(response.json())

# Batch summarization
response = requests.post(
    'http://localhost:8080/api/summarize/batch',
    json={
        'items': [
            {'id': 'item1', 'code': 'func hello() {}'},
            {'id': 'item2', 'code': 'func world() {}'}
        ]
    }
)
print(response.json())

# Get metrics
response = requests.get('http://localhost:8080/api/metrics')
print(response.json())
```

### JavaScript/TypeScript

```typescript
// Single summarization
const response = await fetch('http://localhost:8080/api/summarize', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    code: 'func hello() { println("Hello") }',
    metadata: {
      filePath: 'main.go',
      language: 'go'
    }
  })
});
const data = await response.json();
console.log(data);

// Batch summarization
const batchResponse = await fetch('http://localhost:8080/api/summarize/batch', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    items: [
      { id: 'item1', code: 'func hello() {}' },
      { id: 'item2', code: 'func world() {}' }
    ]
  })
});
const batchData = await batchResponse.json();
console.log(batchData);
```

## Configuration

The HTTP server is configured via environment variables:

```bash
# HTTP Server Configuration
SUMMARIZER_HTTP_PORT=8080           # HTTP server port (default: 8080)
SUMMARIZER_HTTP_HOST=0.0.0.0        # HTTP server host (default: 0.0.0.0)
SUMMARIZER_HTTP_READ_TIMEOUT=15s    # Read timeout (default: 15s)
SUMMARIZER_HTTP_WRITE_TIMEOUT=15s   # Write timeout (default: 15s)
SUMMARIZER_HTTP_SHUTDOWN_TIMEOUT=30s # Shutdown timeout (default: 30s)
SUMMARIZER_HTTP_CORS_ENABLED=true   # Enable CORS (default: true)
SUMMARIZER_HTTP_METRICS_ENABLED=true # Enable metrics endpoint (default: true)
```

## Performance Considerations

### Request Size Limits

- **Single Code**: Maximum 100KB per request
- **Batch Items**: Maximum 100 items per batch, 100KB each
- **Total Batch Size**: Recommended maximum 10MB

### Timeouts

- **Request Timeout**: 30 seconds per request
- **Batch Item Timeout**: 30 seconds per item
- **Server Shutdown**: 30 seconds graceful shutdown

### Caching

- Responses are cached based on code content and metadata
- Cache hit rate is tracked in metrics
- Cache TTL is configurable (default: 24 hours)

## Testing

All HTTP endpoints are fully tested with comprehensive unit tests:

```bash
cd hyper
go test ./internal/mcp/summarizer/... -v -run "TestHandle|TestRequest|TestNot|TestRoot"
```

Test coverage includes:
- ✅ Successful summarization
- ✅ Request validation
- ✅ Error handling
- ✅ Batch processing
- ✅ Partial failures
- ✅ Request ID tracking
- ✅ Health checks
- ✅ Metrics retrieval

## Integration with Kubernetes

The API provides standard Kubernetes probes:

```yaml
livenessProbe:
  httpGet:
    path: /api/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /api/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Next Steps

The HTTP API is now ready for:
1. **Database Persistence** - Store summaries for retrieval
2. **Rate Limiting** - Protect against abuse
3. **Circuit Breaker** - Error recovery pattern
4. **Distributed Caching** - Redis integration
5. **Batch Summarization** - Async job processing
6. **Distributed Tracing** - OpenTelemetry integration
