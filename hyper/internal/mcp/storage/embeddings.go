package storage

import (
	"fmt"
)

// generateTEIEmbedding generates embeddings using TEI (Hugging Face Text Embeddings Inference)
func (c *QdrantClient) generateTEIEmbedding(text string) ([]float64, error) {
	if c.teiClient == nil {
		return nil, fmt.Errorf("TEI client not initialized")
	}

	// Get embedding from TEI (returns float32)
	embedding32, err := c.teiClient.CreateEmbedding(text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TEI embedding: %w", err)
	}

	// Convert float32 to float64 for compatibility with existing Qdrant code
	embedding64 := make([]float64, len(embedding32))
	for i, v := range embedding32 {
		embedding64[i] = float64(v)
	}

	return embedding64, nil
}
