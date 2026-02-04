package mcp

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	"hyper/internal/mcp/embeddings"
	"hyper/internal/mcp/storage"
	"hyper/internal/mcp/summarizer"

	"go.uber.org/zap"
)

// Context-aware auto mode selection helpers (inlined to avoid import cycle with handlers)
const (
	defaultContextWindow     = 100000 // 100k tokens (Claude)
	contextSafetyMargin      = 1000   // Reserved buffer
	fullModeThreshold        = 50000  // > 50k tokens → full mode
	previewModeThreshold     = 20000  // > 20k tokens → preview mode
	defaultSystemPromptTokens = 2000
	defaultMessagesTokens     = 5000
)

// selectResponseModeByContext determines response mode based on remaining context tokens
func selectResponseModeByContext() string {
	// Calculate remaining context: contextWindow - systemPrompt - messages - safetyMargin
	remainingTokens := defaultContextWindow - defaultSystemPromptTokens - defaultMessagesTokens - contextSafetyMargin

	if remainingTokens > fullModeThreshold {
		return "full"
	} else if remainingTokens > previewModeThreshold {
		return "preview"
	}
	return "summary"
}

// Global cache for last code_index_search result (for auto-populating filesModified)
var (
	lastCodeSearchCache     []string
	lastCodeSearchTimestamp time.Time
	codeSearchCacheMutex    sync.RWMutex
	codeSearchCacheTTL      = 5 * time.Minute
)

// parseQueryForFilenameBoost extracts tokens, file extensions, and potential exact filenames from the search query
// Returns:
// - tokens: lowercase tokens from the query (split on spaces and punctuation)
// - extensions: file extensions found in the query (e.g., ".tsx", ".go")
// - exactFilename: potential exact filename if detected (e.g., "App.tsx")
func parseQueryForFilenameBoost(query string) (tokens []string, extensions []string, exactFilename string) {
	queryLower := strings.ToLower(query)

	// Extract file extensions using regex (e.g., .tsx, .go, .js, .ts)
	extPattern := regexp.MustCompile(`\.\w+`)
	extMatches := extPattern.FindAllString(queryLower, -1)
	extensions = extMatches

	// Split query into tokens (split on whitespace and common punctuation)
	tokenPattern := regexp.MustCompile(`[a-zA-Z0-9_]+(?:\.[a-zA-Z0-9]+)?`)
	tokenMatches := tokenPattern.FindAllString(queryLower, -1)

	// Filter out common stopwords
	stopwords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "from": true, "as": true, "is": true, "are": true,
		"was": true, "were": true, "be": true, "been": true, "have": true, "has": true,
		"main": true, "component": true, "function": true, "class": true, "interface": true,
		"code": true, "file": true, "script": true, "module": true,
	}

	for _, token := range tokenMatches {
		if !stopwords[token] && len(token) > 1 {
			tokens = append(tokens, token)
		}
	}

	// Detect exact filename pattern (word with extension, e.g., "App.tsx", "main.go")
	filenamePattern := regexp.MustCompile(`\b([a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)\b`)
	if match := filenamePattern.FindString(queryLower); match != "" {
		exactFilename = match
	}

	return tokens, extensions, exactFilename
}

// calculateFilenameBoost calculates a score boost multiplier based on filename matching
// Returns a multiplier where:
// - 1.0 = no boost
// - 1.3 = 30% boost for filename token match
// - 1.4 = 40% boost for extension match
// - 1.5 = 50% boost for exact filename match
func calculateFilenameBoost(filePath string, queryTokens []string, queryExtensions []string, exactFilename string) float64 {
	if filePath == "" {
		return 1.0
	}

	// Extract filename from path and lowercase it
	filename := strings.ToLower(path.Base(filePath))
	filenameWithoutExt := filename
	fileExt := path.Ext(filename)
	if fileExt != "" {
		filenameWithoutExt = filename[:len(filename)-len(fileExt)]
	}

	boost := 1.0

	// HIGHEST PRIORITY: Exact filename match (50% boost)
	if exactFilename != "" && filename == exactFilename {
		return 1.5
	}

	// HIGH PRIORITY: Extension match (40% boost)
	extensionMatched := false
	for _, ext := range queryExtensions {
		if fileExt == ext {
			boost = 1.4
			extensionMatched = true
			break
		}
	}

	// MEDIUM PRIORITY: Filename token match (30% boost)
	// Only apply if we don't already have an extension match
	if !extensionMatched {
		for _, token := range queryTokens {
			// Check if token appears in filename (without extension)
			if strings.Contains(filenameWithoutExt, token) || strings.Contains(filename, token) {
				boost = 1.3
				break
			}
		}
	}

	return boost
}

// parseRetrieveMode parses the retrieve mode parameter and returns the retrieve type, chunk size in lines, and t-shirt size
// Maps: chunk-s→50/s, chunk-m→100/m, chunk-l→200/l, chunk-xl→400/xl, chunk→200/l (backward compat), full→0/empty
func parseRetrieveMode(mode string) (retrieveType string, chunkLines int, tshirtSize string) {
	switch mode {
	case "chunk-s":
		return "chunk", 50, "s"
	case "chunk-m":
		return "chunk", 100, "m"
	case "chunk-l":
		return "chunk", 200, "l"
	case "chunk-xl":
		return "chunk", 400, "xl"
	case "chunk":
		// Backward compatibility: chunk defaults to chunk-l (200 lines)
		return "chunk", 200, "l"
	case "full":
		return "full", 0, ""
	default:
		// Default to chunk-l for unknown values
		return "chunk", 200, "l"
	}
}

// GetLastCodeSearchPaths returns the cached file paths from the most recent code_index_search
// Returns nil if cache is expired (older than 5 minutes) or empty
func GetLastCodeSearchPaths() []string {
	codeSearchCacheMutex.RLock()
	defer codeSearchCacheMutex.RUnlock()

	// Check if cache is expired
	if time.Since(lastCodeSearchTimestamp) > codeSearchCacheTTL {
		return nil
	}

	// Return a copy to avoid external modification
	if len(lastCodeSearchCache) == 0 {
		return nil
	}
	result := make([]string, len(lastCodeSearchCache))
	copy(result, lastCodeSearchCache)
	return result
}

// CodeIndexSearchTool implements the ToolExecutor interface for code search
// NOW FULLY FUNCTIONAL with embedding and Qdrant clients
type CodeIndexSearchTool struct {
	codeIndexStorage *storage.CodeIndexStorage
	embeddingClient  embeddings.EmbeddingClient
	qdrantClient     *storage.QdrantClient
	summarizer       summarizer.CodeSummarizer
	logger           *zap.Logger
}

// filterResultByMode filters a SearchResult based on the response mode
// summary: returns only Summary, FilePath, StartLine, EndLine, Score, NodeName (~100-200 tokens)
// preview: returns Summary + first 20 lines of Content + metadata (~300-500 tokens)
// full: returns all fields (current behavior, ~500-2000 tokens)
func filterResultByMode(result *storage.SearchResult, mode string, logger *zap.Logger) {
	originalContentLen := len(result.Content)
	originalSymbolsCount := len(result.Symbols)
	originalImportsCount := len(result.Imports)

	result.ResponseMode = mode

	switch mode {
	case "summary":
		// Keep only essential fields for summary mode
		result.Content = ""
		result.ChunkNum = 0
		result.ChunkType = ""
		result.Symbols = nil
		result.Imports = nil
		result.DocContent = ""
		result.FullFileRetrieved = false
		result.ChunkSize = ""

		logger.Debug("🔬 PHASE1: filterResultByMode - SUMMARY mode applied",
			zap.String("filePath", result.FilePath),
			zap.Int("originalContentLen", originalContentLen),
			zap.Int("newContentLen", 0),
			zap.Int("symbolsRemoved", originalSymbolsCount),
			zap.Int("importsRemoved", originalImportsCount),
			zap.String("summaryKept", truncateForLog(result.Summary, 50)))

	case "preview":
		// Keep summary + first 20 lines of content
		originalLines := 0
		if result.Content != "" {
			lines := strings.Split(result.Content, "\n")
			originalLines = len(lines)
			if len(lines) > 20 {
				lines = lines[:20]
				result.Content = strings.Join(lines, "\n") + "\n... (truncated)"
			}
		}
		// Keep metadata but remove some verbose fields
		result.Symbols = nil
		result.Imports = nil

		logger.Debug("🔬 PHASE1: filterResultByMode - PREVIEW mode applied",
			zap.String("filePath", result.FilePath),
			zap.Int("originalLines", originalLines),
			zap.Int("previewLines", 20),
			zap.Int("newContentLen", len(result.Content)),
			zap.Int("symbolsRemoved", originalSymbolsCount),
			zap.Int("importsRemoved", originalImportsCount))

	case "full":
		// Keep everything as-is (default behavior)
		// No filtering needed
		logger.Debug("🔬 PHASE1: filterResultByMode - FULL mode (no filtering)",
			zap.String("filePath", result.FilePath),
			zap.Int("contentLen", originalContentLen),
			zap.Int("symbolsCount", originalSymbolsCount),
			zap.Int("importsCount", originalImportsCount))
	}
}

func (t *CodeIndexSearchTool) Name() string {
	return "code_index_search"
}

func (t *CodeIndexSearchTool) Description() string {
	return "Search for code using natural language queries with tiered response modes to control token usage. Default limit: 10 results (max: 50). Response modes: 'summary' (~100-200 tokens - summary + metadata only), 'preview' (~300-500 tokens - summary + first 20 lines), 'full' (~500-2000 tokens - everything, default). IMPORTANT: retrieve='full' returns only the single best match to conserve tokens - use chunk modes (chunk-s/m/l/xl) for exploring multiple results. NOTE: Requires code index to be populated via MCP endpoint first."
}

func (t *CodeIndexSearchTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query (e.g., 'authentication logic', 'error handling for API calls')",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 10, max: 50)",
			},
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Optional: filter results to a specific folder path",
			},
			"retrieve": map[string]interface{}{
				"type":        "string",
				"description": "Content retrieval mode: 'chunk' (default - 200 lines), 'chunk-s' (50 lines), 'chunk-m' (100 lines), 'chunk-l' (200 lines), 'chunk-xl' (400 lines), or 'full' (entire file)",
				"enum":        []string{"chunk", "chunk-s", "chunk-m", "chunk-l", "chunk-xl", "full"},
			},
			"responseMode": map[string]interface{}{
				"type":        "string",
				"description": "Response mode to control token usage: 'summary' (~100-200 tokens - summary + metadata only), 'preview' (~300-500 tokens - summary + first 20 lines), 'full' (~500-2000 tokens - everything, default)",
				"enum":        []string{"summary", "preview", "full", "auto"},
			},
			"functionName": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Filter by function name pattern (supports regex, e.g., 'handleAuth.*' or 'createUser')",
			},
			"className": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Filter by exact class name (e.g., 'UserService', 'TaskHandler')",
			},
			"nodeType": map[string]interface{}{
				"type":        "string",
				"description": "Optional: Filter by code element type",
				"enum":        []string{"function", "class", "method", "interface", "import"},
			},
			"hasDocstring": map[string]interface{}{
				"type":        "boolean",
				"description": "Optional: Filter to only well-documented code (true) or undocumented code (false)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *CodeIndexSearchTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Validate required fields
	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	// Parse limit parameter (default: 10, max: 50)
	limit := 10
	if l, ok := input["limit"].(float64); ok {
		limit = int(l)
		if limit > 50 {
			limit = 50
		}
	}

	// Parse retrieve mode (default: "chunk")
	retrieveModeParam := "chunk"
	if mode, ok := input["retrieve"].(string); ok {
		retrieveModeParam = mode
	}
	retrieveType, chunkLines, tshirtSize := parseRetrieveMode(retrieveModeParam)

	// Parse response mode (default: "full")
	responseModeParam := "full"
	if mode, ok := input["responseMode"].(string); ok {
		responseModeParam = mode
	}

	// 🔬 PHASE3: Context-Aware Auto Mode Selection
	// If responseMode is "auto", determine the appropriate mode based on remaining context
	if responseModeParam == "auto" {
		// Use inlined helper to select mode based on estimated remaining context
		selectedMode := selectResponseModeByContext()
		responseModeParam = selectedMode

		// Calculate remaining for logging (same logic as selectResponseModeByContext)
		remainingTokens := defaultContextWindow - defaultSystemPromptTokens - defaultMessagesTokens - contextSafetyMargin

		t.logger.Info("🔬 PHASE3: Auto mode selection applied",
			zap.String("selectedMode", selectedMode),
			zap.Int("remainingTokens", remainingTokens),
			zap.String("query", query))
	}

	t.logger.Info("🔬 PHASE1: Response mode parameter parsed",
		zap.String("responseMode", responseModeParam),
		zap.String("query", query),
		zap.Int("limit", limit),
		zap.String("retrieveMode", retrieveModeParam))

	// MCP Best Practice: Limit full file retrieval to top 1 result to conserve context
	// Full mode returns entire file content which can be very large
	// For exploring multiple results, users should use chunk modes (chunk-s/m/l/xl)
	if retrieveType == "full" {
		limit = 1
	}

	// Get current project root
	projectRoot := tools.GetProjectRoot()

	// Lookup collection name from code_index_map
	mapping, err := t.codeIndexStorage.GetPathMapping(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup collection mapping: %w", err)
	}
	if mapping == nil {
		return nil, fmt.Errorf("no code index found for project root '%s' - please restart coordinator to auto-index", projectRoot)
	}

	collectionName := mapping.QdrantCollection
	t.logger.Info("Code search: using collection",
		zap.String("collection", collectionName),
		zap.String("projectRoot", projectRoot),
		zap.String("query", query))

	// Parse optional folderPath filter
	var folderPathFilter string
	if fp, ok := input["folderPath"].(string); ok && fp != "" {
		folderPathFilter = fp
	}

	// Parse structural filter parameters
	var structuralFilter storage.StructuralFilter
	var hasStructuralFilters bool

	if functionName, ok := input["functionName"].(string); ok && functionName != "" {
		structuralFilter.FunctionName = functionName
		hasStructuralFilters = true
	}
	if className, ok := input["className"].(string); ok && className != "" {
		structuralFilter.ClassName = className
		hasStructuralFilters = true
	}
	if nodeType, ok := input["nodeType"].(string); ok && nodeType != "" {
		structuralFilter.NodeType = nodeType
		hasStructuralFilters = true
	}
	if hasDocstring, ok := input["hasDocstring"].(bool); ok {
		structuralFilter.HasDocstring = &hasDocstring
		hasStructuralFilters = true
	}

	// Apply MongoDB pre-filtering if structural filters provided
	var chunkIDs []string
	if hasStructuralFilters {
		chunkIDs, err = t.codeIndexStorage.FindChunksWithFilters(structuralFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to apply structural filters: %w", err)
		}
		t.logger.Info("Code search: structural pre-filter applied",
			zap.Int("matchingChunks", len(chunkIDs)),
			zap.String("functionName", structuralFilter.FunctionName),
			zap.String("className", structuralFilter.ClassName),
			zap.String("nodeType", structuralFilter.NodeType))

		// If no chunks match the structural filter, return empty results
		if len(chunkIDs) == 0 {
			return map[string]interface{}{
				"success":           true,
				"query":             query,
				"FILE_PATHS_TO_USE": []string{},
				"results":           []storage.SearchResult{},
				"resultCount":       0,
				"message":           "No code chunks match the specified structural filters",
			}, nil
		}
	}

	// Generate embedding for query
	queryEmbedding, err := t.embeddingClient.CreateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}
	t.logger.Info("Code search: embedding generated",
		zap.Int("dimensions", len(queryEmbedding)),
		zap.String("collection", collectionName))

	// Search in Qdrant using the correct collection with optional chunk ID filter
	var searchResp *storage.CodeIndexSearchResponse
	if len(chunkIDs) > 0 {
		// Use structural filter with chunk IDs
		searchResp, err = t.qdrantClient.SearchCodeIndexWithFilter(collectionName, queryEmbedding, limit, nil, chunkIDs)
	} else {
		// Standard semantic search without structural filters
		searchResp, err = t.qdrantClient.SearchCodeIndex(collectionName, queryEmbedding, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to search in collection '%s': %w", collectionName, err)
	}
	t.logger.Info("Code search: Qdrant search response",
		zap.Int("resultCount", len(searchResp.Result)),
		zap.String("collection", collectionName))

	// FILENAME-AWARE SCORE BOOSTING + DOCSTRING BOOSTING
	// Parse query once to extract filename matching criteria
	queryTokens, queryExtensions, exactFilename := parseQueryForFilenameBoost(query)
	t.logger.Debug("Query parsed for filename boosting",
		zap.Strings("tokens", queryTokens),
		zap.Strings("extensions", queryExtensions),
		zap.String("exactFilename", exactFilename))

	// Apply score boosting to each hit based on filename matching AND docstring presence
	boostedCount := 0
	docstringBoostedCount := 0
	for i := range searchResp.Result {
		hit := &searchResp.Result[i]

		// Extract file path from payload
		var filePath string
		if fp, ok := hit.Payload["filePath"].(string); ok {
			filePath = fp
		}

		// Calculate filename boost multiplier
		filenameBoostMultiplier := calculateFilenameBoost(filePath, queryTokens, queryExtensions, exactFilename)

		// Calculate docstring boost multiplier (1.2x for documented code)
		docstringBoostMultiplier := 1.0
		if hasDoc, ok := hit.Payload["hasDocstring"].(bool); ok && hasDoc {
			docstringBoostMultiplier = 1.2
			docstringBoostedCount++
		}

		// Combine boosts (multiplicative)
		totalBoostMultiplier := filenameBoostMultiplier * docstringBoostMultiplier

		// Apply boost to score (multiplicative)
		originalScore := hit.Score
		hit.Score = float32(float64(originalScore) * totalBoostMultiplier)

		// Clamp score to maximum of 1.0
		if hit.Score > 1.0 {
			hit.Score = 1.0
		}

		// Log boosted scores for debugging
		if totalBoostMultiplier > 1.0 {
			boostedCount++
			t.logger.Debug("Score boosted",
				zap.String("filePath", filePath),
				zap.Float64("originalScore", float64(originalScore)),
				zap.Float64("filenameBoost", filenameBoostMultiplier),
				zap.Float64("docstringBoost", docstringBoostMultiplier),
				zap.Float64("totalBoost", totalBoostMultiplier),
				zap.Float64("boostedScore", float64(hit.Score)))
		}
	}

	// Re-sort results by boosted scores (descending order)
	// This ensures files with filename matches and docstrings rank higher
	if boostedCount > 0 {
		// Simple bubble sort for small result sets (max 50 items)
		for i := 0; i < len(searchResp.Result)-1; i++ {
			for j := i + 1; j < len(searchResp.Result); j++ {
				if searchResp.Result[j].Score > searchResp.Result[i].Score {
					searchResp.Result[i], searchResp.Result[j] = searchResp.Result[j], searchResp.Result[i]
				}
			}
		}
		t.logger.Info("Results re-sorted after boosting",
			zap.Int("boostedCount", boostedCount),
			zap.Int("docstringBoostedCount", docstringBoostedCount),
			zap.Int("totalResults", len(searchResp.Result)))
	}

	// Build results (filter out archived paths)
	var results []storage.SearchResult
	for _, hit := range searchResp.Result {
		// Extract file path first to check for archived directories
		var filePath string
		if fp, ok := hit.Payload["filePath"].(string); ok {
			filePath = fp
		}

		// FILTER: Skip archived/deprecated files and tmp directories
		if strings.Contains(filePath, "/.archived/") || strings.Contains(filePath, "/.archive/") || strings.Contains(filePath, "/tmp/") {
			t.logger.Debug("Skipping archived file from code search results",
				zap.String("filePath", filePath))
			continue
		}

		// FILTER: Skip files not matching folderPath filter
		if folderPathFilter != "" && !strings.HasPrefix(filePath, folderPathFilter) {
			continue
		}

		result := storage.SearchResult{
			Score:    hit.Score,
			FilePath: filePath,
		}

		// Extract payload fields
		if fileID, ok := hit.Payload["fileId"].(string); ok {
			result.FileID = fileID
		}
		if folderID, ok := hit.Payload["folderId"].(string); ok {
			result.FolderID = folderID
		}
		if folderPath, ok := hit.Payload["folderPath"].(string); ok {
			result.FolderPath = folderPath
		}
		if relativePath, ok := hit.Payload["relativePath"].(string); ok {
			result.RelativePath = relativePath
		}
		if language, ok := hit.Payload["language"].(string); ok {
			result.Language = language
		}
		if chunkNum, ok := hit.Payload["chunkNum"].(float64); ok {
			result.ChunkNum = int(chunkNum)
		}
		if startLine, ok := hit.Payload["startLine"].(float64); ok {
			result.StartLine = int(startLine)
		}
		if endLine, ok := hit.Payload["endLine"].(float64); ok {
			result.EndLine = int(endLine)
		}

		// Extract AST metadata if present
		if chunkType, ok := hit.Payload["chunkType"].(string); ok {
			result.ChunkType = chunkType
		}
		if nodeType, ok := hit.Payload["nodeType"].(string); ok {
			result.NodeType = nodeType
		}
		if nodeName, ok := hit.Payload["nodeName"].(string); ok {
			result.NodeName = nodeName
		}
		if signature, ok := hit.Payload["signature"].(string); ok {
			result.Signature = signature
		}

		// Extract enhanced AST metadata from smart chunking
		if symbols, ok := hit.Payload["symbols"].([]interface{}); ok {
			for _, sym := range symbols {
				if str, ok := sym.(string); ok {
					result.Symbols = append(result.Symbols, str)
				}
			}
		}
		if imports, ok := hit.Payload["imports"].([]interface{}); ok {
			for _, imp := range imports {
				if str, ok := imp.(string); ok {
					result.Imports = append(result.Imports, str)
				}
			}
		}
		if hasDocstring, ok := hit.Payload["hasDocstring"].(bool); ok {
			result.HasDocstring = hasDocstring
		}
		if docContent, ok := hit.Payload["docContent"].(string); ok {
			result.DocContent = docContent
		}

		// Handle content based on retrieve type
		if retrieveType == "chunk" {
			// Sized chunk retrieval: fetch N lines around the match
			if result.FileID != "" && chunkLines > 0 {
				// Calculate target line range centered around the match
				matchMidpoint := (result.StartLine + result.EndLine) / 2
				targetStart := matchMidpoint - (chunkLines / 2)
				targetEnd := targetStart + chunkLines - 1
				if targetStart < 1 {
					targetStart = 1
					targetEnd = targetStart + chunkLines - 1
				}

				// Fetch overlapping chunks from storage
				overlappingChunks, err := t.codeIndexStorage.GetChunksByFileIDAndLineRange(result.FileID, targetStart, targetEnd)
				if err != nil {
					t.logger.Warn("Failed to fetch sized chunk",
						zap.String("fileID", result.FileID),
						zap.Int("targetStart", targetStart),
						zap.Int("targetEnd", targetEnd),
						zap.Error(err))
					// Fallback to Qdrant chunk content
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				} else if len(overlappingChunks) > 0 {
					// Concatenate overlapping chunks and extract exact line range
					var fullContent strings.Builder
					minLine := overlappingChunks[0].StartLine
					for _, chunk := range overlappingChunks {
						fullContent.WriteString(chunk.Content)
					}

					// Extract exactly chunkLines from the concatenated content
					allLines := strings.Split(fullContent.String(), "\n")
					startIdx := targetStart - minLine
					endIdx := targetEnd - minLine + 1
					if startIdx < 0 {
						startIdx = 0
					}
					if endIdx > len(allLines) {
						endIdx = len(allLines)
					}

					extractedLines := allLines[startIdx:endIdx]
					result.Content = strings.Join(extractedLines, "\n")
					result.StartLine = targetStart
					result.EndLine = targetStart + len(extractedLines) - 1
					result.ChunkSize = tshirtSize
				} else {
					// Fallback to Qdrant chunk if no chunks found
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				}
			} else {
				// No sizing requested or invalid chunkLines, use Qdrant chunk
				if content, ok := hit.Payload["content"].(string); ok {
					result.Content = content
				}
			}
		} else if retrieveType == "full" {
			// Fetch entire file content from MongoDB
			if result.FileID != "" {
				allChunks, err := t.codeIndexStorage.GetChunksByFileID(result.FileID)
				if err != nil {
					t.logger.Warn("Failed to fetch full file content",
						zap.String("fileID", result.FileID),
						zap.Error(err))
					// Fallback to chunk content
					if content, ok := hit.Payload["content"].(string); ok {
						result.Content = content
					}
				} else {
					// Concatenate all chunks to build full file content
					var fullContent strings.Builder
					for _, chunk := range allChunks {
						fullContent.WriteString(chunk.Content)
					}
					result.Content = fullContent.String()
					result.FullFileRetrieved = true
				}
			}
		}

		results = append(results, result)
	}

	t.logger.Info("Code search completed",
		zap.String("query", query),
		zap.String("retrieveMode", retrieveModeParam),
		zap.String("retrieveType", retrieveType),
		zap.Int("chunkLines", chunkLines),
		zap.Int("results", len(results)))

	// Apply AI summarization to results (graceful degradation if unavailable)
	results = t.summarizeResults(ctx, results)

	// Apply response mode filtering to reduce token usage
	t.logger.Info("🔬 PHASE1: Starting response mode filtering",
		zap.String("responseMode", responseModeParam),
		zap.Int("resultCount", len(results)))

	for i := range results {
		filterResultByMode(&results[i], responseModeParam, t.logger)
	}

	t.logger.Info("🔬 PHASE1: Response mode filtering COMPLETE",
		zap.String("responseMode", responseModeParam),
		zap.Int("resultCount", len(results)),
		zap.String("tokenEstimate", estimateTokenReduction(responseModeParam)))

	// CRITICAL: Extract file paths into a prominent list so AI can't miss them
	filePaths := make([]string, 0, len(results))
	for _, result := range results {
		filePaths = append(filePaths, result.FilePath)
	}

	// CACHE: Store file paths for auto-populating filesModified in create_agent_task
	codeSearchCacheMutex.Lock()
	lastCodeSearchCache = make([]string, len(filePaths))
	copy(lastCodeSearchCache, filePaths)
	lastCodeSearchTimestamp = time.Now()
	codeSearchCacheMutex.Unlock()

	t.logger.Info("Code search: cached file paths for auto-population",
		zap.Int("cachedPaths", len(filePaths)),
		zap.Time("cachedAt", lastCodeSearchTimestamp))

	response := map[string]interface{}{
		"success":           true,
		"query":             query,
		"FILE_PATHS_TO_USE": filePaths, // PROMINENT - AI must use these exact paths
		"results":           results,
		"resultCount":       len(results),
		"INSTRUCTIONS":      "USE THE EXACT FILE PATHS FROM 'FILE_PATHS_TO_USE' ARRAY - DO NOT GUESS OR MODIFY THEM",
	}

	// Add guidance message when full mode is used
	if retrieveType == "full" {
		response["guidance"] = "Returning top 1 result with full file content to conserve tokens. For multi-result exploration, use chunk-based retrieval modes: chunk-s (50 lines), chunk-m (100 lines), chunk-l (200 lines), or chunk-xl (400 lines)."
	}

	return response, nil
}

// CodeIndexAddFolderTool implements the ToolExecutor interface for adding folders
type CodeIndexAddFolderTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexAddFolderTool) Name() string {
	return "code_index_add_folder"
}

func (t *CodeIndexAddFolderTool) Description() string {
	return "DO NOT USE - Project root is already indexed automatically on server startup. This tool is for admin use only via MCP endpoint. Use code_index_search directly to search the already-indexed codebase."
}

func (t *CodeIndexAddFolderTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the folder to index",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Optional description of the folder/project",
			},
		},
		"required": []string{"folderPath"},
	}
}

func (t *CodeIndexAddFolderTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Validate required fields
	folderPath, ok := input["folderPath"].(string)
	if !ok || folderPath == "" {
		return nil, fmt.Errorf("folderPath is required and must be a string")
	}

	description, _ := input["description"].(string)

	// Security: Validate folder path exists and user has access
	// TODO: Implement access control (check ALLOWED_DIRS or similar whitelist)

	// Check if folder already exists
	existing, err := t.codeIndexStorage.GetFolderByPath(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing folder: %w", err)
	}
	if existing != nil {
		return map[string]interface{}{
			"status":  "already_indexed",
			"message": "Folder already indexed. File watcher is monitoring changes.",
			"folder":  existing,
		}, nil
	}

	// Add folder to storage
	folder, err := t.codeIndexStorage.AddFolder(folderPath, description)
	if err != nil {
		return nil, fmt.Errorf("failed to add folder: %w", err)
	}

	// NOTE: File watcher and scanning require additional dependencies
	// Users should use code_index_scan MCP tool via /mcp endpoint to index files

	return map[string]interface{}{
		"status":    "added",
		"message":   "Folder added successfully. Use code_index_scan MCP tool via /mcp endpoint to index existing files.",
		"folder":    folder,
		"next_step": "Call code_index_scan MCP tool with folderPath to start indexing files",
	}, nil
}

// CodeIndexScanTool implements the ToolExecutor interface for scanning folders
type CodeIndexScanTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexScanTool) Name() string {
	return "code_index_scan"
}

func (t *CodeIndexScanTool) Description() string {
	return "DO NOT USE - Code is already indexed automatically on server startup. This tool requires direct MCP access and will fail if called. Use code_index_search directly to search the already-indexed codebase."
}

func (t *CodeIndexScanTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the folder to scan",
			},
		},
		"required": []string{"folderPath"},
	}
}

func (t *CodeIndexScanTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	folderPath, ok := input["folderPath"].(string)
	if !ok || folderPath == "" {
		return nil, fmt.Errorf("folderPath is required and must be a string")
	}

	// NOTE: Full scanning requires:
	// - File system scanner to detect code files
	// - Embedding client to generate embeddings
	// - Qdrant client to store vectors
	//
	// Return an actual error to prevent AI from retrying
	return nil, fmt.Errorf("code scanning requires direct MCP tool access for file system scanning and embedding generation - this tool cannot perform scanning operations in this context")
}

// CodeIndexStatusTool implements the ToolExecutor interface for index status
type CodeIndexStatusTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexStatusTool) Name() string {
	return "code_index_status"
}

func (t *CodeIndexStatusTool) Description() string {
	return "Get the status of the code index to verify what folders and files are already indexed. The project root is auto-indexed on startup - use this to confirm indexing is complete before using code_index_search."
}

func (t *CodeIndexStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *CodeIndexStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	status, err := t.codeIndexStorage.GetIndexStatus()
	if err != nil {
		return nil, fmt.Errorf("failed to get index status: %w", err)
	}

	return status, nil
}

// CodeIndexRemoveFolderTool implements the ToolExecutor interface for removing folders
type CodeIndexRemoveFolderTool struct {
	codeIndexStorage *storage.CodeIndexStorage
}

func (t *CodeIndexRemoveFolderTool) Name() string {
	return "code_index_remove_folder"
}

func (t *CodeIndexRemoveFolderTool) Description() string {
	return "DO NOT USE - Destructive operation for admin use only. This will delete indexed data. The project root is already indexed - use code_index_search to search it."
}

func (t *CodeIndexRemoveFolderTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"folderPath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the folder to remove (must match the path used when adding)",
			},
		},
		"required": []string{"folderPath"},
	}
}

func (t *CodeIndexRemoveFolderTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	folderPath, ok := input["folderPath"].(string)
	if !ok || folderPath == "" {
		return nil, fmt.Errorf("folderPath is required and must be a string")
	}

	// Check if folder exists
	folder, err := t.codeIndexStorage.GetFolderByPath(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check folder: %w", err)
	}
	if folder == nil {
		return nil, fmt.Errorf("folder not found in index: %s", folderPath)
	}

	// Remove folder
	err = t.codeIndexStorage.RemoveFolder(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to remove folder: %w", err)
	}

	return map[string]interface{}{
		"status":     "removed",
		"message":    "Folder removed successfully",
		"folderPath": folderPath,
	}, nil
}

// summarizeResults applies AI-powered summarization to search results
// Implements graceful degradation: if summarization fails, returns results without summaries
func (t *CodeIndexSearchTool) summarizeResults(ctx context.Context, results []storage.SearchResult) []storage.SearchResult {
	if t.summarizer == nil {
		t.logger.Info("Summarizer not initialized, skipping summarization")
		return results
	}

	const tokenBudget = 5000
	var tokensUsed int

	t.logger.Info("Starting result summarization with LLM",
		zap.Int("resultCount", len(results)),
		zap.Int("tokenBudget", tokenBudget))

	for i := range results {
		// Check if we've exceeded token budget
		if tokensUsed >= tokenBudget {
			t.logger.Info("Summary token budget exhausted",
				zap.Int("tokensUsed", tokensUsed),
				zap.Int("resultsProcessed", i),
				zap.Int("totalResults", len(results)))
			break
		}

		// Build metadata for this result
		metadata := summarizer.CodeMetadata{
			FilePath:   results[i].FilePath,
			Language:   results[i].Language,
			NodeType:   results[i].NodeType,
			NodeName:   results[i].NodeName,
			Signature:  results[i].Signature,
			DocContent: results[i].DocContent,
			LineStart:  results[i].StartLine,
			LineEnd:    results[i].EndLine,
		}

		// Attempt to summarize this result
		summary, err := t.summarizer.Summarize(ctx, results[i].Content, metadata)
		if err != nil {
			t.logger.Warn("Failed to summarize result",
				zap.String("filePath", results[i].FilePath),
				zap.Int("startLine", results[i].StartLine),
				zap.Error(err))
			// Continue without summary for this result (graceful degradation)
			continue
		}

		// Update result with summary information
		results[i].Summary = summary.Text
		results[i].SummaryType = summary.Type
		results[i].SummaryTokens = summary.TokenCount

		tokensUsed += summary.TokenCount

		t.logger.Info("Summarized result successfully",
			zap.String("filePath", results[i].FilePath),
			zap.String("summaryPreview", truncateForLog(summary.Text, 100)),
			zap.Int("summaryTokens", summary.TokenCount),
			zap.Int("cumulativeTokens", tokensUsed))
	}

	t.logger.Info("Summarization complete",
		zap.Int("totalResults", len(results)),
		zap.Int("tokensUsed", tokensUsed))

	return results
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// estimateTokenReduction returns estimated token savings for logging
func estimateTokenReduction(mode string) string {
	switch mode {
	case "summary":
		return "~100-200 tokens (80% reduction from full)"
	case "preview":
		return "~300-500 tokens (50% reduction from full)"
	case "full":
		return "~500-2000 tokens (no reduction)"
	default:
		return "unknown mode"
	}
}

// RegisterCodeIndexTools registers code index tools with the tool registry
// NOW REQUIRES: embedding client and Qdrant client for full search functionality

// CodeIndexGetFullContentTool implements the ToolExecutor interface for fetching full code content
// Enables on-demand code fetching for token-efficient workflows
type CodeIndexGetFullContentTool struct {
	codeIndexStorage *storage.CodeIndexStorage
	logger           *zap.Logger
}

func (t *CodeIndexGetFullContentTool) Name() string {
	return "code_index_get_full_content"
}

func (t *CodeIndexGetFullContentTool) Description() string {
	return "Fetch full code content for a specific file and optional line range. Enables on-demand code retrieval for token-efficient workflows: search returns summaries (~1000 tokens) → AI picks 2-3 results → fetch full code (~1500 tokens) = 75% token reduction. Input: filePath (required), startLine (optional), endLine (optional). If line range not specified, returns entire file. Returns file metadata, content, and actual line range retrieved."
}

func (t *CodeIndexGetFullContentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"filePath": map[string]interface{}{
				"type":        "string",
				"description": "Absolute or relative file path (e.g., './ui/src/components/TaskCard.tsx' or '/Users/user/project/main.go')",
			},
			"startLine": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Starting line number (1-based). If not provided with endLine, returns entire file.",
			},
			"endLine": map[string]interface{}{
				"type":        "integer",
				"description": "Optional: Ending line number (1-based, inclusive). If not provided with startLine, returns entire file.",
			},
		},
		"required": []string{"filePath"},
	}
}

func (t *CodeIndexGetFullContentTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	t.logger.Info("🚀 PHASE2: code_index_get_full_content INVOKED",
		zap.Any("input", input))

	// Validate required fields
	filePath, ok := input["filePath"].(string)
	if !ok || filePath == "" {
		t.logger.Error("🚀 PHASE2: VALIDATION FAILED - filePath missing")
		return nil, fmt.Errorf("filePath is required and must be a string")
	}

	// Parse optional line range parameters
	startLine := 0
	endLine := 0

	if sl, ok := input["startLine"].(float64); ok {
		startLine = int(sl)
	}
	if el, ok := input["endLine"].(float64); ok {
		endLine = int(el)
	}

	isFullFile := startLine == 0 && endLine == 0
	t.logger.Info("🚀 PHASE2: Fetching code content",
		zap.String("filePath", filePath),
		zap.Int("startLine", startLine),
		zap.Int("endLine", endLine),
		zap.Bool("isFullFile", isFullFile))

	// Fetch content from storage
	result, err := t.codeIndexStorage.GetContentByFilePathAndLineRange(filePath, startLine, endLine)
	if err != nil {
		t.logger.Error("🚀 PHASE2: STORAGE FETCH FAILED",
			zap.String("filePath", filePath),
			zap.Int("startLine", startLine),
			zap.Int("endLine", endLine),
			zap.Error(err))
		return nil, fmt.Errorf("failed to fetch code content: %w", err)
	}

	contentLen := len(result.Content)
	linesFetched := result.EndLine - result.StartLine + 1
	t.logger.Info("🚀 PHASE2: Code content fetched SUCCESSFULLY",
		zap.String("filePath", result.FilePath),
		zap.String("relativePath", result.RelativePath),
		zap.String("language", result.Language),
		zap.Int("startLine", result.StartLine),
		zap.Int("endLine", result.EndLine),
		zap.Int("linesFetched", linesFetched),
		zap.Int("totalFileLines", result.LineCount),
		zap.Int("contentBytes", contentLen),
		zap.Int64("fileSize", result.Size))

	// Build response
	response := map[string]interface{}{
		"success":        true,
		"filePath":       result.FilePath,
		"relativePath":   result.RelativePath,
		"language":       result.Language,
		"lineCount":      result.LineCount,
		"content":        result.Content,
		"startLine":      result.StartLine,
		"endLine":        result.EndLine,
		"requestedStart": result.RequestedStart,
		"requestedEnd":   result.RequestedEnd,
		"size":           result.Size,
		"indexedAt":      result.IndexedAt,
	}

	// Add note if requested range was adjusted
	rangeAdjusted := false
	if (startLine > 0 || endLine > 0) && (result.StartLine != startLine || result.EndLine != endLine) {
		response["note"] = fmt.Sprintf("Requested lines %d-%d, returned %d-%d (adjusted to valid range)", startLine, endLine, result.StartLine, result.EndLine)
		rangeAdjusted = true
	}

	t.logger.Info("🚀 PHASE2: code_index_get_full_content COMPLETE",
		zap.String("filePath", result.FilePath),
		zap.Int("linesFetched", linesFetched),
		zap.Int("contentBytes", contentLen),
		zap.Bool("rangeAdjusted", rangeAdjusted),
		zap.Bool("isFullFile", isFullFile),
		zap.String("tokenEstimate", fmt.Sprintf("~%d tokens", contentLen/4))) // rough estimate: 4 chars per token

	return response, nil
}

func RegisterCodeIndexTools(
	registry *aiservice.ToolRegistry,
	codeIndexStorage *storage.CodeIndexStorage,
	embeddingClient embeddings.EmbeddingClient,
	qdrantClient *storage.QdrantClient,
	codeSummarizer summarizer.CodeSummarizer,
	logger *zap.Logger,
) error {
	tools := []aiservice.ToolExecutor{
		&CodeIndexSearchTool{
			codeIndexStorage: codeIndexStorage,
			embeddingClient:  embeddingClient,
			qdrantClient:     qdrantClient,
			summarizer:       codeSummarizer,
			logger:           logger,
		},
		&CodeIndexGetFullContentTool{
			codeIndexStorage: codeIndexStorage,
			logger:           logger,
		},
		&CodeIndexAddFolderTool{codeIndexStorage: codeIndexStorage},
		&CodeIndexScanTool{codeIndexStorage: codeIndexStorage},
		&CodeIndexStatusTool{codeIndexStorage: codeIndexStorage},
		&CodeIndexRemoveFolderTool{codeIndexStorage: codeIndexStorage},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
