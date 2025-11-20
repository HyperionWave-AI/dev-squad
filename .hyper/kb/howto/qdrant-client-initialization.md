# Qdrant Client Initialization Pattern

## Overview

Standard pattern for initializing Qdrant vector database client in Hyperion services with proper configuration for embedding generation and collection management.

## Technology

- Qdrant Vector Database
- TEI (Text Embeddings Inference) Service
- nomic-embed-text-v1.5 model (768 dimensions)

## Use Case

Use this pattern when you need to initialize a Qdrant client for vector storage and semantic search operations. The client handles embedding generation through TEI service and manages collections for knowledge storage.

## Implementation

### Basic Client Setup

**File Reference**: `hyper/internal/mcp/storage/qdrant_client.go:37-66`

```go
func NewQdrantClient(baseURL string, knowledgeCollectionName string) *QdrantClient {
    qdrantKey := os.Getenv("QDRANT_API_KEY")
    teiURL := os.Getenv("TEI_URL")
    if teiURL == "" {
        teiURL = "http://embedding-service:8080"
    }

    if knowledgeCollectionName == "" {
        knowledgeCollectionName = "dev_squad_knowledge"
    }

    teiClient := embeddings.NewTEIClient(teiURL)

    client := &QdrantClient{
        baseURL:                 baseURL,
        qdrantAPIKey:            qdrantKey,
        teiClient:               teiClient,
        vectorDimension:         768, // TEI nomic-embed-text-v1.5
        knowledgeCollectionName: knowledgeCollectionName,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }

    client.embeddingFunc = client.generateTEIEmbedding
    return client
}
```

## Key Points

### Environment Variables

- **QDRANT_URL**: Vector DB endpoint (default: `http://qdrant:6333`)
- **QDRANT_API_KEY**: Optional authentication token
- **TEI_URL**: Embedding service endpoint (default: `http://embedding-service:8080`)
- **QDRANT_KNOWLEDGE_COLLECTION**: Collection name (default: `dev_squad_knowledge`)

### Configuration Defaults

- **Vector dimension**: 768 (nomic-embed-text-v1.5 model)
- **Distance metric**: Cosine similarity
- **HTTP timeout**: 30 seconds
- **Default collection**: "dev_squad_knowledge"

### Best Practices

1. **Fallback defaults**: Always provide sensible defaults for optional configuration
2. **TEI integration**: Use dedicated TEI client for embedding generation
3. **Timeout configuration**: Set reasonable HTTP client timeout for vector operations
4. **Embedding function**: Inject embedding function for testability
5. **Client pooling**: HTTP client is reusable and thread-safe

### Architecture Notes

- Client handles both embedding generation (via TEI) and vector storage (Qdrant)
- Embedding function is configurable, allowing mock implementations for testing
- Vector dimension is hardcoded to match TEI model output
- Collection name can be overridden for multi-tenant or test scenarios

## Metadata

- **Domain**: vector-database
- **Language**: go
- **Pattern**: initialization
- **Technology**: qdrant
