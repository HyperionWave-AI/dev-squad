// +build llama

package embeddings

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	llama "github.com/go-skynet/go-llama.cpp"
)

// LlamaClient provides embedded embeddings using llama.cpp
// Supports GPU acceleration: Metal (macOS), CUDA (NVIDIA), Vulkan (AMD/Intel)
// No external service required - runs in-process
type LlamaClient struct {
	model      *llama.LLama
	dimensions int
	mu         sync.Mutex
	modelPath  string
}

// NewLlamaClient creates a new embedded llama.cpp client
// modelPath: path to GGUF model file (e.g., "models/nomic-embed-text-v1.5.Q4_K_M.gguf")
// If modelPath is empty, uses default location: ./models/nomic-embed-text-v1.5.Q4_K_M.gguf
func NewLlamaClient(modelPath string) (*LlamaClient, error) {
	if modelPath == "" {
		// Default model location
		modelPath = filepath.Join("models", "nomic-embed-text-v1.5.Q4_K_M.gguf")
	}

	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s\nDownload with: curl -L https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf -o %s", modelPath, modelPath)
	}

	// Load model with GPU acceleration
	// Metal/CUDA/Vulkan support is enabled at build time via CGO flags
	model, err := llama.New(modelPath,
		llama.SetContext(2048),       // Context window for embeddings
		llama.EnableEmbeddings,       // Enable embedding mode
		llama.SetGPULayers(99),       // Offload all layers to GPU (auto-detected)
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load llama.cpp model: %w", err)
	}

	return &LlamaClient{
		model:      model,
		dimensions: 768, // nomic-embed-text-v1.5 dimensions
		modelPath:  modelPath,
	}, nil
}

// CreateEmbedding generates a single embedding vector for the given text
func (c *LlamaClient) CreateEmbedding(text string) ([]float32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Generate embeddings using llama.cpp
	embeddings, err := c.model.Embeddings(text)
	if err != nil {
		return nil, fmt.Errorf("llama.cpp embedding generation failed: %w", err)
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("llama.cpp returned empty embedding")
	}

	return embeddings, nil
}

// CreateEmbeddings generates embedding vectors for multiple texts
func (c *LlamaClient) CreateEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	results := make([][]float32, len(texts))

	// Process each text sequentially
	// llama.cpp handles batching internally for optimal GPU utilization
	for i, text := range texts {
		embedding, err := c.CreateEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding for text %d: %w", i, err)
		}
		results[i] = embedding
	}

	return results, nil
}

// GetDimensions returns the number of dimensions for the embedding model
// nomic-embed-text-v1.5: 768 dimensions
func (c *LlamaClient) GetDimensions() int {
	return c.dimensions
}

// Close frees the model resources
func (c *LlamaClient) Close() {
	if c.model != nil {
		c.model.Free()
	}
}
