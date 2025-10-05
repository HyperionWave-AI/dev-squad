package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// OpenAI Embeddings API structures
type embeddingRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format,omitempty"`
}

type embeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// generateOpenAIEmbedding generates embeddings using OpenAI API
func (c *QdrantClient) generateOpenAIEmbedding(text string) ([]float64, error) {
	if c.openAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY not configured")
	}

	// Retry logic with exponential backoff
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		embedding, err := c.callOpenAIEmbeddingAPI(text)
		if err == nil {
			return embedding, nil
		}

		lastErr = err

		// Exponential backoff: 1s, 2s, 4s
		if attempt < maxRetries-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// callOpenAIEmbeddingAPI makes a single API call to OpenAI embeddings endpoint
func (c *QdrantClient) callOpenAIEmbeddingAPI(text string) ([]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqBody := embeddingRequest{
		Input: text,
		Model: c.embeddingModel,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.openAIBaseURL+"/embeddings", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.openAIAPIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(embResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned from OpenAI")
	}

	embedding := embResp.Data[0].Embedding
	if len(embedding) != c.vectorDimension {
		return nil, fmt.Errorf("unexpected embedding dimension: got %d, expected %d", len(embedding), c.vectorDimension)
	}

	return embedding, nil
}
