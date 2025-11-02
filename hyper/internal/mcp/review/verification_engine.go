package review

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/storage"

	"go.uber.org/zap"
)

// VerificationEngine validates references in knowledge entries
type VerificationEngine struct {
	codeIndexStorage *storage.CodeIndexStorage
	qdrantClient     *storage.QdrantClient
	embeddingClient  embeddings.EmbeddingClient
	logger           *zap.Logger
	cache            *ValidationCache
	projectRoot      string // Git repository root for commit validation
}

// NewVerificationEngine creates a new verification engine instance
func NewVerificationEngine(
	codeIndexStorage *storage.CodeIndexStorage,
	qdrantClient *storage.QdrantClient,
	embeddingClient embeddings.EmbeddingClient,
	projectRoot string,
	logger *zap.Logger,
) *VerificationEngine {
	return &VerificationEngine{
		codeIndexStorage: codeIndexStorage,
		qdrantClient:     qdrantClient,
		embeddingClient:  embeddingClient,
		logger:           logger,
		cache:            NewValidationCache(5 * time.Minute), // 5-minute TTL
		projectRoot:      projectRoot,
	}
}

// VerifyEntry validates all references in a knowledge entry
// Returns VerificationResult with validation details
func (ve *VerificationEngine) VerifyEntry(entryID, entryText string) (*VerificationResult, error) {
	startTime := time.Now()

	// Step 1: Parse all references from entry text
	references := parseReferences(entryText)

	result := &VerificationResult{
		TotalReferences: len(references),
	}

	if len(references) == 0 {
		result.ValidationTime = time.Since(startTime)
		return result, nil
	}

	// Step 2: Validate each reference in parallel (with concurrency limit)
	semaphore := make(chan struct{}, 10) // Max 10 concurrent validations
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range references {
		wg.Add(1)
		go func(ref *Reference) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			valid, err := ve.validateReference(ref)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				ref.ErrorMessage = err.Error()
				result.BrokenReferences = append(result.BrokenReferences, *ref)
			} else if valid {
				ref.Validated = true
				result.ValidReferences++
			} else {
				result.InvalidReferences = append(result.InvalidReferences, *ref)
			}
		}(&references[i])
	}

	wg.Wait()
	result.ValidationTime = time.Since(startTime)

	ve.logger.Info("Entry verification completed",
		zap.String("entryID", entryID),
		zap.Int("total", result.TotalReferences),
		zap.Int("valid", result.ValidReferences),
		zap.Int("invalid", len(result.InvalidReferences)),
		zap.Int("broken", len(result.BrokenReferences)),
		zap.Duration("duration", result.ValidationTime))

	return result, nil
}

// validateReference validates a single reference based on its type
func (ve *VerificationEngine) validateReference(ref *Reference) (bool, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s:%s", ref.Type, ref.Value)
	if cached, found := ve.cache.Get(cacheKey); found {
		return cached, nil
	}

	// Validate based on type
	var valid bool
	var err error

	switch ref.Type {
	case ReferenceTypeFileLine:
		valid, err = ve.validateFileLine(ref.Value)
	case ReferenceTypeFunction:
		valid, err = ve.validateFunction(ref.Value)
	case ReferenceTypeCommit:
		valid, err = ve.validateCommit(ref.Value)
	case ReferenceTypeAPI:
		valid, err = ve.validateAPIEndpoint(ref.Value)
	case ReferenceTypeFile:
		valid, err = ve.validateFile(ref.Value)
	default:
		return false, fmt.Errorf("unknown reference type: %s", ref.Type)
	}

	// Cache the result
	if err == nil {
		ve.cache.Set(cacheKey, valid)
	}

	return valid, err
}

// validateFileLine validates file:line or file:startLine-endLine references
func (ve *VerificationEngine) validateFileLine(ref string) (bool, error) {
	// Parse file:line or file:startLine-endLine
	parts := strings.Split(ref, ":")
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid file:line format")
	}

	filePath := parts[0]
	lineRange := parts[1]

	// Use code_index_search to find the file
	query := fmt.Sprintf("file:%s", filepath.Base(filePath))
	embedding, err := ve.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return false, fmt.Errorf("embedding generation failed: %w", err)
	}

	// Get the collection name from the code index storage
	// For simplicity, use the default code index collection
	collectionName := storage.CodeIndexCollection
	if collectionName == "" {
		collectionName = "code_index_default"
	}

	searchResp, err := ve.qdrantClient.SearchCodeIndex(collectionName, embedding, 10)
	if err != nil {
		return false, fmt.Errorf("code search failed: %w", err)
	}

	// Check if file exists in results
	for _, hit := range searchResp.Result {
		if hitPath, ok := hit.Payload["filePath"].(string); ok {
			if strings.HasSuffix(hitPath, filePath) || strings.Contains(hitPath, filePath) {
				// File exists! Now check if line number is reasonable
				fileID, ok := hit.Payload["fileId"].(string)
				if !ok {
					continue
				}

				// Parse line range
				if strings.Contains(lineRange, "-") {
					// Range: startLine-endLine
					rangeParts := strings.Split(lineRange, "-")
					if len(rangeParts) != 2 {
						return false, fmt.Errorf("invalid line range format")
					}
					startLine, err := strconv.Atoi(rangeParts[0])
					if err != nil {
						return false, fmt.Errorf("invalid start line number: %w", err)
					}
					endLine, err := strconv.Atoi(rangeParts[1])
					if err != nil {
						return false, fmt.Errorf("invalid end line number: %w", err)
					}

					// Get file chunks to verify line range exists
					chunks, err := ve.codeIndexStorage.GetChunksByFileIDAndLineRange(fileID, startLine, endLine)
					if err == nil && len(chunks) > 0 {
						return true, nil
					}
				} else {
					// Single line
					lineNum, err := strconv.Atoi(lineRange)
					if err != nil {
						return false, fmt.Errorf("invalid line number: %w", err)
					}

					// Get file chunks to verify line exists
					chunks, err := ve.codeIndexStorage.GetChunksByFileID(fileID)
					if err == nil && len(chunks) > 0 {
						// Check if line number is within file bounds
						lastChunk := chunks[len(chunks)-1]
						if lineNum <= lastChunk.EndLine && lineNum >= 1 {
							return true, nil
						}
					}
				}
			}
		}
	}

	return false, nil
}

// validateFunction validates function/method name references
func (ve *VerificationEngine) validateFunction(funcName string) (bool, error) {
	// Search code index for function definition
	query := fmt.Sprintf("func %s function definition", funcName)
	embedding, err := ve.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return false, err
	}

	collectionName := storage.CodeIndexCollection
	if collectionName == "" {
		collectionName = "code_index_default"
	}

	searchResp, err := ve.qdrantClient.SearchCodeIndex(collectionName, embedding, 5)
	if err != nil {
		return false, err
	}

	// Check if any result contains the function name
	for _, hit := range searchResp.Result {
		if content, ok := hit.Payload["content"].(string); ok {
			if strings.Contains(content, funcName) {
				return true, nil
			}
		}
	}

	return false, nil
}

// validateCommit validates git commit hash references
func (ve *VerificationEngine) validateCommit(commitHash string) (bool, error) {
	// Use git to verify commit exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "log", "--oneline", "-n", "1", commitHash)
	cmd.Dir = ve.projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Commit doesn't exist or git command failed
		return false, nil
	}

	return len(output) > 0, nil
}

// validateAPIEndpoint validates API endpoint references (e.g., "POST /api/v1/endpoint")
func (ve *VerificationEngine) validateAPIEndpoint(endpoint string) (bool, error) {
	// Parse method and path
	parts := strings.SplitN(endpoint, " ", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid API endpoint format")
	}

	method := parts[0]
	path := parts[1]

	// Search for route registration in code
	query := fmt.Sprintf("%s %s route handler registration", method, path)
	embedding, err := ve.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return false, err
	}

	collectionName := storage.CodeIndexCollection
	if collectionName == "" {
		collectionName = "code_index_default"
	}

	searchResp, err := ve.qdrantClient.SearchCodeIndex(collectionName, embedding, 10)
	if err != nil {
		return false, err
	}

	// Look for route definition in results
	for _, hit := range searchResp.Result {
		if content, ok := hit.Payload["content"].(string); ok {
			// Check for route registration patterns
			if strings.Contains(content, method) && strings.Contains(content, path) {
				return true, nil
			}
		}
	}

	return false, nil
}

// validateFile validates file path references (without line numbers)
func (ve *VerificationEngine) validateFile(filePath string) (bool, error) {
	// Similar to validateFileLine but without line number check
	query := fmt.Sprintf("file:%s", filepath.Base(filePath))
	embedding, err := ve.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return false, err
	}

	collectionName := storage.CodeIndexCollection
	if collectionName == "" {
		collectionName = "code_index_default"
	}

	searchResp, err := ve.qdrantClient.SearchCodeIndex(collectionName, embedding, 5)
	if err != nil {
		return false, err
	}

	for _, hit := range searchResp.Result {
		if hitPath, ok := hit.Payload["filePath"].(string); ok {
			if strings.HasSuffix(hitPath, filePath) || strings.Contains(hitPath, filePath) {
				return true, nil
			}
		}
	}

	return false, nil
}

// ClearCache clears all cached validation results
func (ve *VerificationEngine) ClearCache() {
	ve.cache.Clear()
}

// GetCacheSize returns the current number of cached validation results
func (ve *VerificationEngine) GetCacheSize() int {
	return ve.cache.Size()
}
