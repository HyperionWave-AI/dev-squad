# Qdrant Collection Management Pattern

## Overview

Idempotent collection creation and validation pattern for Qdrant vector database that ensures collections exist with correct vector dimensions before use.

## Technology

- Qdrant Vector Database REST API
- Go HTTP client
- Cosine similarity distance metric

## Use Case

Use this pattern when you need to ensure a Qdrant collection exists before performing vector operations. The pattern handles both creation of new collections and validation of existing collections to prevent dimension mismatches.

## Implementation

### EnsureCollection Pattern

**File Reference**: `hyper/internal/mcp/storage/qdrant_client.go:125-208`

```go
func (c *QdrantClient) EnsureCollection(collectionName string, vectorSize int) error {
    // 1. Check if collection exists
    checkURL := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
    req, _ := http.NewRequest("GET", checkURL, nil)
    c.addAuthHeader(req)

    resp, _ := c.httpClient.Do(req)
    defer resp.Body.Close()

    // 2. If exists, validate dimensions
    if resp.StatusCode == http.StatusOK {
        var collectionInfo struct {
            Result struct {
                Config struct {
                    Params struct {
                        Vectors struct {
                            Size int `json:"size"`
                        } `json:"vectors"`
                    } `json:"params"`
                } `json:"config"`
            } `json:"result"`
        }

        body, _ := io.ReadAll(resp.Body)
        json.Unmarshal(body, &collectionInfo)
        actualDim := collectionInfo.Result.Config.Params.Vectors.Size

        if actualDim != vectorSize {
            return &DimensionMismatchError{
                ExpectedDim: actualDim,
                GotDim:      vectorSize,
                Collection:  collectionName,
            }
        }
        return nil
    }

    // 3. Create if not exists
    createPayload := map[string]interface{}{
        "vectors": map[string]interface{}{
            "size":     vectorSize,
            "distance": "Cosine",
        },
    }

    payloadBytes, _ := json.Marshal(createPayload)
    createURL := fmt.Sprintf("%s/collections/%s", c.baseURL, collectionName)
    req, _ = http.NewRequest("PUT", createURL, bytes.NewReader(payloadBytes))
    req.Header.Set("Content-Type", "application/json")
    c.addAuthHeader(req)

    c.httpClient.Do(req)
    return nil
}
```

## Key Points

### Safety Features

- **Idempotent**: Safe to call multiple times - will not recreate existing collections
- **Dimension Check**: Validates existing collection has correct vector dimensions
- **Typed Error**: Returns `DimensionMismatchError` for explicit error handling
- **Auto-create**: Automatically creates collection with correct configuration if missing

### Best Practices

1. **Check before create**: Always GET collection info before attempting PUT
2. **Validate dimensions**: Compare actual vs expected dimensions to prevent data corruption
3. **Typed errors**: Use custom error types for better error handling upstream
4. **HTTP status codes**: Use 200 OK as indicator of existing collection
5. **Cosine distance**: Default to Cosine similarity for semantic search use cases

### Error Handling

```go
type DimensionMismatchError struct {
    ExpectedDim int
    GotDim      int
    Collection  string
}

func (e *DimensionMismatchError) Error() string {
    return fmt.Sprintf(
        "dimension mismatch in collection %s: expected %d, got %d",
        e.Collection, e.ExpectedDim, e.GotDim,
    )
}
```

### Common Patterns

- Call `EnsureCollection()` during service initialization
- Always specify vector size matching your embedding model
- Handle `DimensionMismatchError` separately from other errors
- Use consistent distance metric across all collections

### Configuration

- **vectorSize**: Must match embedding model output (e.g., 768 for nomic-embed-text-v1.5)
- **distance**: Cosine similarity for normalized vectors
- **collectionName**: Use namespaced names for multi-tenant scenarios

## Metadata

- **Domain**: vector-database
- **Language**: go
- **Pattern**: collection-management
- **Technology**: qdrant
