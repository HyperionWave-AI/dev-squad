# Set auto-recreate flag (development only)

**Collection:** infrastructure
**Created:** 2025-11-20

---

## Overview

Hyperion uses Qdrant as a vector database for semantic search capabilities. This document describes how Qdrant is integrated, configured, and used throughout the project.

## Client Initialization

### Location
**File:** `/hyper/internal/mcp/storage/qdrant_client.go`

### Basic Client Creation

```go
func NewQdrantClient(baseURL string, knowledgeCollectionName string) *QdrantClient {
    qdrantKey := os.Getenv("QDRANT_API_KEY")
    teiURL := os.Getenv("TEI_URL") // Default: "http://embedding-service:8080"

    return &QdrantClient{
        baseURL:                 baseURL,
        qdrantAPIKey:            qdrantKey,
        teiClient:               embeddings.NewTEIClient(teiURL),
        vectorDimension:         768, // TEI nomic-embed-text-v1.5 dimension
        knowledgeCollectionName: knowledgeCollectionName,
        httpClient:              &http.Client{Timeout: 30 * time.Second},
    }
}
```

### Client with Custom Embedding Provider

```go
func NewQdrantClientWithEmbeddingClient(
    baseURL string,
    knowledgeCollectionName string,
    embeddingClient embeddings.EmbeddingClient
) *QdrantClient {
    // Allows injection of custom embedding provider
    // Useful for testing or alternative models
}
```

## Configuration

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `QDRANT_URL` | Yes | `http://localhost:7333` | Qdrant HTTP API endpoint |
| `QDRANT_API_KEY` | No | - | API key for authentication |
| `TEI_URL` | No | `http://embedding-service:8080` | Text Embeddings Inference endpoint |
| `QDRANT_CODE_COLLECTION` | No | `code_index` | Collection name for code indexing |
| `QDRANT_KNOWLEDGE_COLLECTION` | No | `hyper-knowledge-base` | Collection name for knowledge base |

### Connection Settings

- **HTTP Client Timeout:** 30 seconds
- **Authentication:** Via `api-key` header (if QDRANT_API_KEY is set)
- **Health Check:** Ping endpoint available at `/collections`

## Collections

### 1. Code Index Collection

**Name:** `code_index` (configurable)

**Purpose:** Semantic code search across indexed repositories

**Configuration:**
```go
// Vector dimension: 768 (default, provider-dependent)
// Distance metric: Cosine
// Indexing threshold: 20000 points
```

**Payload Schema:**
```json
{
  "fileId": "string",
  "filePath": "string",
  "content": "string",
  "startLine": "number",
  "endLine": "number",
  "language": "string",
  "chunkType": "ast|line-based",
  "nodeType": "function|class|method|interface",
  "nodeName": "string",
  "signature": "string"
}
```

### 2. Unified Knowledge Collection

**Name:** `hyper-knowledge-base` (configurable)

**Purpose:** All knowledge entries with MongoDB collection filtering

**Key Features:**
- Unified collection replaces per-collection Qdrant collections
- Filtering via `collectionId` payload field
- Optional task scoping via `taskId` field

**Payload Schema:**
```json
{
  "text": "string",
  "id": "string",
  "collectionId": "string",
  "taskId": "string (optional)",
  "createdAt": "ISO8601",
  "metadata": {
    "sourceFile": "string",
    "tags": ["array"],
    "version": "string"
  }
}
```

### 3. Reflection Lessons Collection

**Name:** `reflection_lessons_{dimensions}` (dimension-specific)

**Purpose:** Metacognitive lesson storage for AI learning

**Features:**
- Dimension-specific collection names
- Pattern-based learning storage
- Context and confidence tracking

## Embedding Models

### Embedding Interface

```go
type EmbeddingClient interface {
    CreateEmbedding(text string) ([]float32, error)
    CreateEmbeddings(texts []string) ([][]float32, error)
    GetDimensions() int
}
```

### Supported Providers

#### 1. TEI (Text Embeddings Inference) - Default

**Configuration:**
- Model: `nomic-embed-text-v1.5`
- Dimensions: 768
- URL: `http://embedding-service:8080`

**Use Case:** Local development and production (self-hosted)

#### 2. Ollama - Local GPU

**Configuration:**
- Model: `nomic-embed-text` (configurable via `OLLAMA_MODEL`)
- URL: `http://localhost:7335`
- Env: `OLLAMA_URL`, `OLLAMA_MODEL`

**Use Case:** Local development with GPU

#### 3. Voyage AI - Cloud Production

**Configuration:**
- Model: `voyage-3` (configurable via `VOYAGE_MODEL`)
- Requires API key: `VOYAGE_API_KEY`

**Use Case:** Production deployments

#### 4. OpenAI - Cloud

**Configuration:**
- Model: `text-embedding-3-small` (configurable via `OPENAI_MODEL`)
- Requires API key: `OPENAI_API_KEY`

**Use Case:** Alternative cloud provider

## Usage Patterns

### Storage Pattern

**File:** `/hyper/internal/mcp/storage/qdrant_client.go` (Lines 332-390)

```go
func (c *QdrantClient) StorePoint(
    collectionName string,
    id string,
    text string,
    metadata map[string]interface{}
) error {
    // 1. Generate embedding from text
    vector, err := c.embeddingFunc(text)
    if err != nil {
        return fmt.Errorf("failed to generate embedding: %w", err)
    }

    // 2. Build payload with metadata
    payload := map[string]interface{}{
        "text":      text,
        "id":        id,
        "createdAt": time.Now().UTC().Format(time.RFC3339),
    }

    // 3. Merge custom metadata
    for k, v := range metadata {
        payload[k] = v
    }

    // 4. Upsert point to Qdrant
    return c.upsertPoint(collectionName, id, vector, payload)
}
```

### Search Pattern

**File:** `/hyper/internal/mcp/storage/qdrant_client.go` (Lines 392-495)

```go
func (c *QdrantClient) SearchSimilar(
    collectionName string,
    query string,
    limit int,
    voteBoost ...float64
) ([]*QdrantQueryResult, error) {
    // 1. Generate query embedding
    queryVector, err := c.embeddingFunc(query)

    // 2. Perform vector search
    results, err := c.searchPoints(collectionName, queryVector, limit)

    // 3. Apply optional vote boosting
    if len(voteBoost) > 0 && voteBoost[0] > 0 {
        results = c.applyVoteBoosting(results, voteBoost[0])
    }

    return results, nil
}
```

### Vote Boosting Formula

**Purpose:** Boost search results based on community voting

**Formula:**
```
finalScore = semanticScore * (1 + voteBoost * normalizedVoteScore)

where:
  normalizedVoteScore = voteScore / (1 + abs(voteScore))
  voteBoost = configurable multiplier (e.g., 0.3)
```

**Example:**
```
Semantic score: 0.85
Vote score: +5 upvotes
Vote boost: 0.3

normalizedVoteScore = 5 / (1 + 5) = 0.833
finalScore = 0.85 * (1 + 0.3 * 0.833) = 0.85 * 1.25 = 1.0625
```

## Collection Management

### Create Collection

```go
func (c *QdrantClient) EnsureCollection(name string, dimension int) error {
    // Checks if collection exists
    // Creates if missing with proper vector configuration
    // Returns error if dimension mismatch detected
}
```

### Dimension Mismatch Handling

**Problem:** Vector dimensions change when switching embedding providers

**Solution:** Auto-migration with reindexing

```go
if dimErr, ok := err.(*storage.DimensionMismatchError); ok {
    // Fetch all entries from MongoDB
    entries, _ := fetchEntriesFromMongoDB()

    // Recreate collection with new dimension
    err = qdrantClient.RecreateCollectionWithReindex(
        collectionName,
        entries,
        newDimension
    )
}
```

### Delete Collection

```go
func (c *QdrantClient) DeleteCollection(name string) error {
    // Permanently removes collection and all points
    // Use with caution - data cannot be recovered
}
```

## Connection Handling

### Health Check

```go
func (c *QdrantClient) Ping() error {
    // Verifies Qdrant service is accessible
    // Returns error if connection fails
}
```

**Endpoint:** `GET {QDRANT_URL}/collections`

### Error Handling

**Common Errors:**
1. **Connection Timeout:** Check QDRANT_URL and network
2. **Dimension Mismatch:** Use RecreateCollectionWithReindex
3. **Authentication Failed:** Verify QDRANT_API_KEY
4. **Collection Not Found:** Use EnsureCollection before operations

## Best Practices

### 1. Always Use Unified Knowledge Collection

```go
// ✅ GOOD - Use unified collection with collectionId filter
payload["collectionId"] = mongoCollectionID

// ❌ BAD - Creating separate Qdrant collections per knowledge collection
// (Deprecated pattern)
```

### 2. Include Task Scoping When Relevant

```go
if taskId != nil {
    payload["taskId"] = *taskId
}
```

### 3. Handle Dimension Mismatches Gracefully

```go
err := qdrantClient.EnsureCollection(collName, expectedDimension)
if dimErr, ok := err.(*storage.DimensionMismatchError); ok {
    // Trigger reindexing workflow
    log.Warn("Dimension mismatch detected, reindexing required")
}
```

### 4. Use Vote Boosting for Knowledge Quality

```go
// Boost factor 0.3 = 30% additional weight for highly-voted content
results, err := qdrantClient.SearchSimilar(collection, query, 10, 0.3)
```

## Performance Considerations

### Indexing Thresholds

- **Small collections (< 20K points):** No indexing, brute-force search
- **Large collections (> 20K points):** Automatic HNSW indexing

### Timeout Configuration

- **HTTP Client:** 30 seconds (configurable)
- **Embedding Generation:** Depends on provider (TEI ~100ms, OpenAI ~500ms)

### Batch Operations

```go
// For bulk indexing, use batch upsert
func (c *QdrantClient) UpsertBatch(
    collectionName string,
    points []QdrantPoint
) error {
    // Processes points in batches of 100
    // More efficient than individual upserts
}
```

## Related Documents

- [MongoDB Integration](./mongodb-integration.md) - Data persistence layer
- [Configuration Reference](./configuration-reference.md) - All environment variables
- [Component Architecture](./component-architecture.md) - System integration overview

## Troubleshooting

### Issue: "Collection dimension mismatch"

**Cause:** Embedding provider changed or model updated

**Solution:**
```bash
# Set auto-recreate flag (development only)
export CODE_INDEX_AUTO_RECREATE=true

# Or manually trigger reindex via API
curl -X POST http://localhost:7095/api/v1/code/reindex
```

### Issue: "Connection refused to Qdrant"

**Cause:** Qdrant service not running or wrong URL

**Solution:**
```bash
# Check Qdrant status
docker ps | grep qdrant

# Verify URL
echo $QDRANT_URL

# Test connection
curl http://localhost:7333/collections
```

### Issue: "Embedding generation timeout"

**Cause:** Embedding service overloaded or unreachable

**Solution:**
```bash
# Check embedding service
curl http://embedding-service:8080/health

# Switch to alternative provider
export EMBEDDING=voyage
export VOYAGE_API_KEY=your-key
```
