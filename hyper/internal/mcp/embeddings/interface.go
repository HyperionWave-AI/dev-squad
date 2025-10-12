package embeddings

// EmbeddingClient is the interface for embedding generation services
// Implemented by both OpenAIClient and TEIClient
type EmbeddingClient interface {
	// CreateEmbedding generates a single embedding vector for the given text
	CreateEmbedding(text string) ([]float32, error)

	// CreateEmbeddings generates embedding vectors for multiple texts
	CreateEmbeddings(texts []string) ([][]float32, error)

	// GetDimensions returns the number of dimensions for the embedding model
	GetDimensions() int
}
