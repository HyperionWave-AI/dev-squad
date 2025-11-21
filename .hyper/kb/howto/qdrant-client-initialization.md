# How to Initialize and Use Qdrant Client

**Collection:** howto
**Tags:** qdrant, vector-database, embeddings, semantic-search, go
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide explains how to initialize and use Qdrant vector database client for semantic search capabilities. You'll learn how to configure the client, manage collections, generate embeddings, and perform vector similarity searches.

## Prerequisites

- Qdrant server running (local or cloud)
- Go 1.25 with HTTP client
- Understanding of vector embeddings
- Knowledge of [Qdrant Integration](../qdrant-integration.md)

## When to Use This Guide

- Implementing semantic code search
- Building knowledge base with similarity search
- Creating recommendation systems
- Storing and querying vector embeddings

---

## Steps

### Step 1: Define Qdrant Client Structure

Create `internal/storage/qdrant_client.go`:

```go
package storage

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

type QdrantClient struct {
    baseURL      string
    apiKey       string
    httpClient   *http.Client
    dimensions   int
    embeddingFn  EmbeddingFunc
}

// EmbeddingFunc generates vector embeddings from text
type EmbeddingFunc func(text string) ([]float32, error)

type QdrantConfig struct {
    BaseURL    string
    APIKey     string
    Dimensions int
    Timeout    time.Duration
}
```

### Step 2: Create Client Constructor

Initialize the Qdrant client with configuration:

```go
func NewQdrantClient(config *QdrantConfig, embeddingFn EmbeddingFunc) *QdrantClient {
    if config.Timeout == 0 {
        config.Timeout = 30 * time.Second
    }

    return &QdrantClient{
        baseURL:     config.BaseURL,
        apiKey:      config.APIKey,
        dimensions:  config.Dimensions,
        embeddingFn: embeddingFn,
        httpClient: &http.Client{
            Timeout: config.Timeout,
        },
    }
}

// Ping verifies Qdrant connection
func (c *QdrantClient) Ping() error {
    url := fmt.Sprintf("%s/collections", c.baseURL)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return fmt.Errorf("failed to create request: %w", err)
    }
    
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to ping Qdrant: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("Qdrant ping failed with status: %d", resp.StatusCode)
    }
    
    return nil
}
```

### Step 3: Implement Collection Management

Add methods to create and manage collections:

```go
// EnsureCollection creates collection if it doesn't exist
func (c *QdrantClient) EnsureCollection(collectionName string) error {
    // Check if collection exists
    exists, err := c.collectionExists(collectionName)
    if err != nil {
        return err
    }
    
    if exists {
        return nil // Collection already exists
    }
    
    // Create collection
    return c.createCollection(collectionName)
}

func (c *QdrantClient) collectionExists(name string) (bool, error) {
    url := fmt.Sprintf("%s/collections/%s", c.baseURL, name)
    
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return false, err
    }
    
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()
    
    return resp.StatusCode == http.StatusOK, nil
}

func (c *QdrantClient) createCollection(name string) error {
    url := fmt.Sprintf("%s/collections/%s", c.baseURL, name)
    
    payload := map[string]interface{}{
        "vectors": map[string]interface{}{
            "size":     c.dimensions,
            "distance": "Cosine", // Cosine similarity
        },
    }
    
    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %w", err)
    }
    
    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to create collection: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("create collection failed: %s", string(bodyBytes))
    }
    
    return nil
}

// DeleteCollection removes a collection and all its points
func (c *QdrantClient) DeleteCollection(name string) error {
    url := fmt.Sprintf("%s/collections/%s", c.baseURL, name)
    
    req, err := http.NewRequest("DELETE", url, nil)
    if err != nil {
        return err
    }
    
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to delete collection: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("delete collection failed with status: %d", resp.StatusCode)
    }
    
    return nil
}
```

### Step 4: Implement Point Operations

Add methods to upsert and delete vectors:

```go
type Point struct {
    ID      string                 `json:"id"`
    Vector  []float32              `json:"vector"`
    Payload map[string]interface{} `json:"payload"`
}

// UpsertPoint stores or updates a vector point
func (c *QdrantClient) UpsertPoint(
    collectionName string,
    id string,
    text string,
    payload map[string]interface{},
) error {
    // Generate embedding from text
    vector, err := c.embeddingFn(text)
    if err != nil {
        return fmt.Errorf("failed to generate embedding: %w", err)
    }
    
    // Add text to payload
    if payload == nil {
        payload = make(map[string]interface{})
    }
    payload["text"] = text
    payload["createdAt"] = time.Now().UTC().Format(time.RFC3339)
    
    // Build request
    url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, collectionName)
    
    requestBody := map[string]interface{}{
        "points": []Point{
            {
                ID:      id,
                Vector:  vector,
                Payload: payload,
            },
        },
    }
    
    body, err := json.Marshal(requestBody)
    if err != nil {
        return fmt.Errorf("failed to marshal request: %w", err)
    }
    
    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to upsert point: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("upsert failed: %s", string(bodyBytes))
    }
    
    return nil
}

// DeletePoint removes a point from collection
func (c *QdrantClient) DeletePoint(collectionName string, id string) error {
    url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, collectionName)
    
    requestBody := map[string]interface{}{
        "points": []string{id},
    }
    
    body, err := json.Marshal(requestBody)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return fmt.Errorf("failed to delete point: %w", err)
    }
    defer resp.Body.Close()
    
    return nil
}
```

### Step 5: Implement Semantic Search

Add vector similarity search:

```go
type SearchResult struct {
    ID      string                 `json:"id"`
    Score   float64                `json:"score"`
    Payload map[string]interface{} `json:"payload"`
}

// Search performs semantic similarity search
func (c *QdrantClient) Search(
    collectionName string,
    query string,
    limit int,
    filter map[string]interface{},
) ([]SearchResult, error) {
    // Generate query embedding
    queryVector, err := c.embeddingFn(query)
    if err != nil {
        return nil, fmt.Errorf("failed to generate query embedding: %w", err)
    }
    
    // Build search request
    url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collectionName)
    
    requestBody := map[string]interface{}{
        "vector": queryVector,
        "limit":  limit,
        "with_payload": true,
    }
    
    // Add filter if provided
    if filter != nil {
        requestBody["filter"] = buildQdrantFilter(filter)
    }
    
    body, err := json.Marshal(requestBody)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    if c.apiKey != "" {
        req.Header.Set("api-key", c.apiKey)
    }
    
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("search request failed: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        bodyBytes, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("search failed: %s", string(bodyBytes))
    }
    
    // Parse response
    var response struct {
        Result []struct {
            ID      interface{}            `json:"id"`
            Score   float64                `json:"score"`
            Payload map[string]interface{} `json:"payload"`
        } `json:"result"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return nil, fmt.Errorf("failed to decode response: %w", err)
    }
    
    // Convert to SearchResult
    results := make([]SearchResult, len(response.Result))
    for i, r := range response.Result {
        results[i] = SearchResult{
            ID:      fmt.Sprintf("%v", r.ID),
            Score:   r.Score,
            Payload: r.Payload,
        }
    }
    
    return results, nil
}

func buildQdrantFilter(filter map[string]interface{}) map[string]interface{} {
    must := []map[string]interface{}{}
    
    for key, value := range filter {
        must = append(must, map[string]interface{}{
            "key": key,
            "match": map[string]interface{}{
                "value": value,
            },
        })
    }
    
    return map[string]interface{}{
        "must": must,
    }
}
```

### Step 6: Create Embedding Function

Implement embedding generation (using TEI as example):

```go
package storage

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
)

// NewTEIEmbeddingFunc creates an embedding function using TEI service
func NewTEIEmbeddingFunc(teiURL string) EmbeddingFunc {
    client := &http.Client{Timeout: 10 * time.Second}
    
    return func(text string) ([]float32, error) {
        requestBody := map[string]string{
            "inputs": text,
        }
        
        body, err := json.Marshal(requestBody)
        if err != nil {
            return nil, err
        }
        
        req, err := http.NewRequest("POST", teiURL, bytes.NewBuffer(body))
        if err != nil {
            return nil, err
        }
        
        req.Header.Set("Content-Type", "application/json")
        
        resp, err := client.Do(req)
        if err != nil {
            return nil, fmt.Errorf("TEI request failed: %w", err)
        }
        defer resp.Body.Close()
        
        if resp.StatusCode != http.StatusOK {
            return nil, fmt.Errorf("TEI returned status: %d", resp.StatusCode)
        }
        
        var embeddings [][]float32
        if err := json.NewDecoder(resp.Body).Decode(&embeddings); err != nil {
            return nil, fmt.Errorf("failed to decode embeddings: %w", err)
        }
        
        if len(embeddings) == 0 || len(embeddings[0]) == 0 {
            return nil, fmt.Errorf("empty embedding response")
        }
        
        return embeddings[0], nil
    }
}
```

### Step 7: Initialize in Application

Set up Qdrant client in `main.go`:

```go
package main

import (
    "os"
    "your-project/internal/storage"
)

func main() {
    // Load configuration
    qdrantURL := os.Getenv("QDRANT_URL")
    qdrantAPIKey := os.Getenv("QDRANT_API_KEY")
    teiURL := os.Getenv("TEI_URL")
    
    // Create embedding function
    embeddingFn := storage.NewTEIEmbeddingFunc(teiURL)
    
    // Initialize Qdrant client
    qdrantConfig := &storage.QdrantConfig{
        BaseURL:    qdrantURL,
        APIKey:     qdrantAPIKey,
        Dimensions: 768, // nomic-embed-text-v1.5
        Timeout:    30 * time.Second,
    }
    
    qdrantClient := storage.NewQdrantClient(qdrantConfig, embeddingFn)
    
    // Verify connection
    if err := qdrantClient.Ping(); err != nil {
        log.Fatalf("Failed to connect to Qdrant: %v", err)
    }
    
    // Ensure collections exist
    if err := qdrantClient.EnsureCollection("knowledge_base"); err != nil {
        log.Fatalf("Failed to ensure collection: %v", err)
    }
    
    log.Println("Qdrant client initialized successfully")
}
```

### Step 8: Use in Application Logic

Example: Store and search knowledge entries:

```go
// Store knowledge entry
func storeKnowledge(client *QdrantClient, entry KnowledgeEntry) error {
    payload := map[string]interface{}{
        "collectionId": entry.CollectionID,
        "taskId":       entry.TaskID,
        "metadata":     entry.Metadata,
    }
    
    return client.UpsertPoint(
        "knowledge_base",
        entry.ID,
        entry.Text,
        payload,
    )
}

// Search knowledge
func searchKnowledge(client *QdrantClient, query string, collectionID string) ([]SearchResult, error) {
    filter := map[string]interface{}{
        "collectionId": collectionID,
    }
    
    return client.Search("knowledge_base", query, 10, filter)
}
```

---

## Best Practices

### 1. Connection Pooling
Reuse Qdrant client across requests (HTTP client has built-in pooling).

### 2. Dimension Consistency
Ensure vector dimensions match between embedding model and collection:
```go
if len(vector) != c.dimensions {
    return fmt.Errorf("dimension mismatch: got %d, expected %d", len(vector), c.dimensions)
}
```

### 3. Error Handling
Always handle embedding generation errors separately from Qdrant errors.

### 4. Batch Operations
For bulk indexing, batch multiple points in single request:
```go
requestBody := map[string]interface{}{
    "points": []Point{point1, point2, point3, ...},
}
```

### 5. Collection Naming
Use descriptive, dimension-specific collection names:
- `code_index_768` (for 768-dim embeddings)
- `knowledge_base_1024` (for 1024-dim embeddings)

---

## Related Documentation

- [Qdrant Integration](../qdrant-integration.md) - Architecture details
- [MongoDB Integration](../mongodb-integration.md) - Metadata storage
- [Configuration Reference](../configuration-reference.md) - Environment variables

---

## Troubleshooting

### Issue: "Connection refused"

**Solution:**
```bash
# Check Qdrant is running
docker ps | grep qdrant

# Test connection
curl http://localhost:6333/collections
```

### Issue: "Dimension mismatch"

**Solution:**
```go
// Recreate collection with correct dimensions
client.DeleteCollection("knowledge_base")
client.EnsureCollection("knowledge_base")
```

### Issue: "Embedding generation timeout"

**Solution:**
```go
// Increase HTTP client timeout
httpClient: &http.Client{Timeout: 60 * time.Second}
```
