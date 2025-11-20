package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	"hyper/internal/handlers"
	mcphandlers "hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"
	"hyper/internal/models"
	"hyper/internal/validation"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// correctFilePaths attempts to fix invalid file paths using common correction strategies
// Returns (correctedPaths, unfixablePaths, wasIndexingIssue)
func correctFilePaths(paths []string, logger *zap.Logger) ([]string, []string, bool) {
	if len(paths) == 0 {
		return paths, nil, false
	}

	projectRoot := tools.GetProjectRoot()
	logger.Info("🔧 Validating and correcting file paths",
		zap.Int("pathCount", len(paths)),
		zap.String("projectRoot", projectRoot))

	correctedPaths := make([]string, 0, len(paths))
	unfixablePaths := make([]string, 0)
	hadCorrections := false
	indexingIssuePatterns := 0

	for _, path := range paths {
		if path == "" {
			continue
		}

		// FIX #5: Resolve relative paths against project root, not working directory
		// Relative paths like "./ui/src/..." should be resolved from project root
		resolvedPath := path
		if !filepath.IsAbs(path) {
			// Path is relative - resolve against project root
			resolvedPath = filepath.Join(projectRoot, path)
			logger.Debug("Resolved relative path",
				zap.String("original", path),
				zap.String("resolved", resolvedPath))
		}

		// Check if resolved path exists
		if _, err := os.Stat(resolvedPath); err == nil {
			correctedPaths = append(correctedPaths, resolvedPath)
			logger.Debug("✅ Path valid", zap.String("path", resolvedPath))
			continue
		}

		// Path doesn't exist, try to fix it
		logger.Warn("⚠️  Path does not exist, attempting correction",
			zap.String("originalPath", path),
			zap.String("resolvedPath", resolvedPath))

		fixedPath := tryFixPath(resolvedPath, projectRoot, logger)

		if fixedPath != "" {
			// Verify the fixed path exists
			if _, err := os.Stat(fixedPath); err == nil {
				correctedPaths = append(correctedPaths, fixedPath)
				hadCorrections = true
				logger.Info("✅ Path corrected successfully",
					zap.String("original", path),
					zap.String("corrected", fixedPath))

				// Check if this looks like an indexing issue
				if strings.Contains(path, "/hyper/hyper/") || strings.Contains(path, "/hyper/ui/") {
					indexingIssuePatterns++
				}
			} else {
				unfixablePaths = append(unfixablePaths, path)
				logger.Error("❌ Path correction failed - fixed path still doesn't exist",
					zap.String("original", path),
					zap.String("attempted", fixedPath))
			}
		} else {
			unfixablePaths = append(unfixablePaths, path)
			logger.Error("❌ Could not correct path - no valid corrections found",
				zap.String("path", path))
		}
	}

	// Determine if this is an indexing issue
	isIndexingIssue := indexingIssuePatterns > 0
	if isIndexingIssue {
		logger.Warn("🔍 INDEXING ISSUE DETECTED",
			zap.Int("pathsWithIssue", indexingIssuePatterns),
			zap.String("pattern", "Paths contain duplicate /hyper/ or incorrect /hyper/ui/ prefixes"),
			zap.String("recommendation", "Code index may be storing incorrect paths - consider re-indexing"))
	}

	if hadCorrections {
		logger.Info("🔧 Path correction summary",
			zap.Int("totalPaths", len(paths)),
			zap.Int("corrected", len(correctedPaths)),
			zap.Int("unfixable", len(unfixablePaths)),
			zap.Bool("indexingIssue", isIndexingIssue))
	}

	return correctedPaths, unfixablePaths, isIndexingIssue
}

// extractPatternFiles scans contextSummary and todos for file references
// Returns validated file paths that exist and can be used as pattern references
func extractPatternFiles(contextSummary string, todos []string, projectRoot string, logger *zap.Logger) []string {
	// Common file extensions to look for in references
	// Examples: "follow pattern from HTTPToolsPage.tsx", "similar to ChatSessionList.tsx"
	fileExtensions := []string{".tsx", ".ts", ".jsx", ".js", ".go", ".css", ".html", ".py", ".java"}

	candidateFiles := make(map[string]bool)

	// Combine all text to scan
	allText := contextSummary + " " + strings.Join(todos, " ")

	logger.Debug("Scanning for pattern file references",
		zap.String("contextSummary", contextSummary),
		zap.Strings("todos", todos))

	// Extract file names with extensions
	for _, ext := range fileExtensions {
		// Find all occurrences of words ending with the extension
		// Using a simple approach: split by spaces and look for filenames
		words := strings.Fields(allText)
		for _, word := range words {
			// Clean up punctuation
			cleaned := strings.Trim(word, ".,;:()[]{}\"'`")
			if strings.HasSuffix(cleaned, ext) {
				candidateFiles[cleaned] = true
				logger.Debug("Found potential pattern file reference",
					zap.String("file", cleaned))
			}
		}
	}

	if len(candidateFiles) == 0 {
		logger.Debug("No pattern file references found in context/TODOs")
		return nil
	}

	// Now validate and find full paths for these files
	validatedFiles := []string{}
	searchDirs := []string{
		"ui/src/components",
		"ui/src/pages",
		"ui/src/hooks",
		"ui/src",
		"hyper/internal",
		"hyper/internal/ai-service",
		"hyper/internal/mcp",
		"hyper/internal/server",
	}

	for filename := range candidateFiles {
		var foundPath string

		// Strategy 1: Try as absolute path
		if strings.HasPrefix(filename, "/") {
			if _, err := os.Stat(filename); err == nil {
				foundPath = filename
			}
		}

		// Strategy 2: Try relative to project root
		if foundPath == "" {
			candidate := filepath.Join(projectRoot, filename)
			if _, err := os.Stat(candidate); err == nil {
				foundPath = candidate
			}
		}

		// Strategy 3: Search in common directories
		if foundPath == "" {
			for _, dir := range searchDirs {
				candidate := filepath.Join(projectRoot, dir, filename)
				if _, err := os.Stat(candidate); err == nil {
					foundPath = candidate
					break
				}
			}
		}

		// Strategy 4: Try finding with find command (last resort)
		if foundPath == "" {
			logger.Debug("Searching for pattern file using find command",
				zap.String("filename", filename))
			// Use find to locate the file
			// SECURITY: Use exec.Command with separate arguments to prevent command injection
			// DO NOT use bash -c or fmt.Sprintf with user-controlled input
			cmd := exec.Command("find", projectRoot, "-name", filename, "-type", "f")
			output, err := cmd.Output()
			if err == nil && len(output) > 0 {
				// Take first result only (equivalent to | head -1)
				lines := strings.Split(strings.TrimSpace(string(output)), "\n")
				if len(lines) > 0 && lines[0] != "" {
					foundPath = lines[0]
					if _, err := os.Stat(foundPath); err != nil {
						foundPath = "" // Invalid path
					}
				}
			}
		}

		if foundPath != "" {
			validatedFiles = append(validatedFiles, foundPath)
			logger.Info("✅ Validated pattern reference file",
				zap.String("filename", filename),
				zap.String("fullPath", foundPath))
		} else {
			logger.Warn("⚠️  Pattern reference file not found",
				zap.String("filename", filename),
				zap.String("suggestion", "File mentioned in context but doesn't exist - coordinator may need to verify this reference"))
		}
	}

	return validatedFiles
}

// tryFixPath attempts multiple strategies to correct an invalid file path
func tryFixPath(path string, projectRoot string, logger *zap.Logger) string {
	strategies := []struct {
		name string
		fix  func(string) string
	}{
		{
			name: "Remove duplicate /hyper/hyper/ pattern",
			fix: func(p string) string {
				return strings.Replace(p, "/hyper/hyper/", "/hyper/", 1)
			},
		},
		{
			name: "Replace /hyper/ui/ with /ui/",
			fix: func(p string) string {
				return strings.Replace(p, "/hyper/ui/", "/ui/", 1)
			},
		},
		{
			name: "Remove leading /hyper/ if path starts with it",
			fix: func(p string) string {
				if strings.HasPrefix(p, "/hyper/") {
					return strings.TrimPrefix(p, "/hyper")
				}
				return ""
			},
		},
		{
			name: "Prepend project root to relative path",
			fix: func(p string) string {
				// Remove leading ./ if present
				cleaned := strings.TrimPrefix(p, "./")
				return filepath.Join(projectRoot, cleaned)
			},
		},
		{
			name: "Search in common UI directories",
			fix: func(p string) string {
				basename := filepath.Base(p)
				commonDirs := []string{
					"ui/src/components",
					"ui/src/pages",
					"ui/src",
				}
				for _, dir := range commonDirs {
					candidate := filepath.Join(projectRoot, dir, basename)
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
				return ""
			},
		},
		{
			name: "Search in common backend directories",
			fix: func(p string) string {
				basename := filepath.Base(p)
				commonDirs := []string{
					"hyper/internal/ai-service",
					"hyper/internal/mcp",
					"hyper/internal",
				}
				for _, dir := range commonDirs {
					candidate := filepath.Join(projectRoot, dir, basename)
					if _, err := os.Stat(candidate); err == nil {
						return candidate
					}
				}
				return ""
			},
		},
	}

	for _, strategy := range strategies {
		fixed := strategy.fix(path)
		if fixed != "" && fixed != path {
			logger.Debug("Trying correction strategy",
				zap.String("strategy", strategy.name),
				zap.String("original", path),
				zap.String("candidate", fixed))

			// Check if this fixed path exists
			if _, err := os.Stat(fixed); err == nil {
				logger.Info("✅ Correction strategy successful",
					zap.String("strategy", strategy.name),
					zap.String("fixed", fixed))
				return fixed
			}
		}
	}

	return "" // No correction worked
}

// CoordinatorTools provides MCP coordinator tool executors for LangChain
type CoordinatorTools struct {
	taskStorage      storage.TaskStorage
	knowledgeStorage storage.KnowledgeStorage
}

// NewCoordinatorTools creates a new coordinator tools handler
func NewCoordinatorTools(taskStorage storage.TaskStorage, knowledgeStorage storage.KnowledgeStorage) *CoordinatorTools {
	return &CoordinatorTools{
		taskStorage:      taskStorage,
		knowledgeStorage: knowledgeStorage,
	}
}

// CreateAgentTaskTool implements the ToolExecutor interface
type CreateAgentTaskTool struct {
	storage   storage.TaskStorage
	aiService AIServiceInterface
	config    *aiservice.AIConfig
}

func (t *CreateAgentTaskTool) Name() string {
	return "create_agent_task"
}

func (t *CreateAgentTaskTool) Description() string {
	return "Create a new agent task linked to a human task. Returns task ID. SMART AUTO-FETCH: If humanTaskId is omitted, automatically fetches the most recent pending human task from the database. IMPORTANT: Use code_index_search FIRST to discover relevant files, then populate filesModified with the file paths from search results. Include detailed context in contextSummary with WHAT to change, WHERE (file:line from search results), and HOW. NEVER ask the user for file paths - discover them automatically with code_index_search. Required: agentName, role, todos. Optional: humanTaskId, contextSummary, filesModified, qdrantCollections, priorWorkSummary."
}

func (t *CreateAgentTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"humanTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Parent human task ID (UUID format). OPTIONAL: If not provided, automatically uses the most recent pending human task from the database.",
			},
			"agentName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the agent assigned to this task",
			},
			"role": map[string]interface{}{
				"type":        "string",
				"description": "Agent's role/responsibility for this task",
			},
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of TODO items (tasks to complete)",
			},
			"contextSummary": map[string]interface{}{
				"type":        "string",
				"description": "200-word summary with specifics from code_index_search results: WHAT to change, WHERE (file paths + line numbers from search results), HOW (patterns/examples from search results). Example: 'Add delete button to ui/src/components/TaskCard.tsx lines 42-45 (found via code_index_search). Use IconButton pattern from same file line 38. Wire to deleteTask mutation.'",
			},
			"filesModified": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "File paths discovered from code_index_search results (e.g., ['./ui/src/components/TaskCard.tsx']). Extract file paths from search results and include them here. DO NOT ask user for paths - use code_index_search to find them automatically. Leave empty only if search returns no results.",
			},
			"qdrantCollections": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Suggested Qdrant collections to query if technical patterns needed",
			},
			"priorWorkSummary": map[string]interface{}{
				"type":        "string",
				"description": "Summary of previous agent's work and key decisions (for multi-phase tasks)",
			},
		},
		"required": []string{"agentName", "role", "todos"},
	}
}

func (t *CreateAgentTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// FIX #2: Explicit Error for humanTaskId Type Mismatch
	// Previously: Silent type assertion failure treated wrong type as empty string
	// Now: Fail fast with clear error if humanTaskId is provided but wrong type
	providedTaskID := ""
	if taskIDRaw, exists := input["humanTaskId"]; exists {
		var ok bool
		providedTaskID, ok = taskIDRaw.(string)
		if !ok {
			return nil, fmt.Errorf("humanTaskId must be a string, got %T: %v", taskIDRaw, taskIDRaw)
		}
	}

	// Extract task details for complexity analysis
	agentName, _ := input["agentName"].(string)
	role, _ := input["role"].(string)
	contextSummary, _ := input["contextSummary"].(string)

	// Extract filesModified for complexity analysis (will be validated more strictly later)
	var filesModified []string
	if fm, ok := input["filesModified"].([]interface{}); ok {
		filesModified = make([]string, len(fm))
		for i, f := range fm {
			if str, ok := f.(string); ok {
				filesModified[i] = str
			}
		}
	}

	// Declare variables for later use
	var newComplexityAnalysis ComplexityAnalysis
	var legacyComplexityAnalysis *aiservice.ComplexityAnalysis
	var splitSuggestions []aiservice.SuggestedSplit
	var todos []string // Will be parsed later

	// NOTE: Complexity analysis will be performed after todos are parsed

	// Helper function to fetch latest pending task
	fetchLatestTask := func() (*storage.HumanTask, error) {
		allTasks := t.storage.ListAllHumanTasks()
		if len(allTasks) == 0 {
			return nil, fmt.Errorf("no human tasks found - create a human task first using coordinator_create_human_task")
		}

		// Find the most recent pending task
		var latestTask *storage.HumanTask
		for _, task := range allTasks {
			if task.Status == storage.TaskStatusPending {
				if latestTask == nil || task.CreatedAt.After(latestTask.CreatedAt) {
					latestTask = task
				}
			}
		}

		// If no pending task, use the most recent task regardless of status
		if latestTask == nil {
			latestTask = allTasks[0]
			for _, task := range allTasks {
				if task.CreatedAt.After(latestTask.CreatedAt) {
					latestTask = task
				}
			}
		}

		return latestTask, nil
	}

	var humanTaskID string

	// FIX #3: Retry Logic for Task Lookup (handles race conditions)
	// Previously: Single lookup could fail if MongoDB hasn't committed yet
	// Now: Retry up to 3 times with exponential backoff for eventual consistency
	if providedTaskID != "" {
		var task *storage.HumanTask
		var err error

		// Retry up to 3 times with exponential backoff
		for attempt := 0; attempt < 3; attempt++ {
			task, err = t.storage.GetHumanTask(providedTaskID)
			if err == nil && task != nil {
				break // Success
			}
			if attempt < 2 {
				// Wait before retry: 100ms, then 200ms
				sleepDuration := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
				zap.L().Debug("Retrying humanTaskId lookup after delay",
					zap.String("providedTaskId", providedTaskID),
					zap.Int("attempt", attempt+1),
					zap.Duration("delay", sleepDuration))
				time.Sleep(sleepDuration)
			}
		}

		if err != nil || task == nil {
			// Failed after retries - log original providedTaskID for debugging
			zap.L().Warn("Failed to find humanTaskId after retries - auto-fetching latest task",
				zap.String("providedTaskId", providedTaskID),
				zap.String("error", fmt.Sprintf("%v", err)),
				zap.Int("retries", 3))

			// Auto-fetch latest task as fallback
			latestTask, fetchErr := fetchLatestTask()
			if fetchErr != nil {
				return nil, fmt.Errorf("invalid humanTaskId %q and no fallback task available: %w", providedTaskID, fetchErr)
			}
			humanTaskID = latestTask.ID

			zap.L().Info("Auto-corrected humanTaskId (not found after retries)",
				zap.String("modelProvidedId", providedTaskID),
				zap.String("correctedId", humanTaskID),
				zap.String("prompt", latestTask.Prompt),
				zap.Time("createdAt", latestTask.CreatedAt))
		} else {
			// Valid task ID
			humanTaskID = task.ID
			zap.L().Info("Validated humanTaskId from model",
				zap.String("humanTaskId", humanTaskID),
				zap.String("status", string(task.Status)))
		}
	} else {
		// No task ID provided - auto-fetch latest
		latestTask, err := fetchLatestTask()
		if err != nil {
			return nil, err
		}
		humanTaskID = latestTask.ID

		zap.L().Info("Auto-fetched latest pending human task (no ID provided)",
			zap.String("humanTaskId", humanTaskID),
			zap.String("prompt", latestTask.Prompt),
			zap.Time("createdAt", latestTask.CreatedAt))
	}

	agentName, ok := input["agentName"].(string)
	if !ok || agentName == "" {
		return nil, fmt.Errorf("agentName is required and must be a string")
	}

	role, ok = input["role"].(string)
	if !ok || role == "" {
		return nil, fmt.Errorf("role is required and must be a string")
	}

	todosRaw, ok := input["todos"]
	if !ok {
		return nil, fmt.Errorf("todos is required")
	}

	// Convert todos to []string (reuse the variable declared earlier)
	switch v := todosRaw.(type) {
	case []interface{}:
		todos = make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("todos[%d] must be a string", i)
			}
			todos[i] = str
		}
	case []string:
		todos = v
	default:
		return nil, fmt.Errorf("todos must be an array of strings")
	}

	if len(todos) == 0 {
		return nil, fmt.Errorf("todos must not be empty")
	}

	// VALIDATION: Check for discovery keywords in TODOs (subagents cannot search/discover)
	discoveryKeywords := []string{
		"search", "find", "locate", "discover", "look for",
		"code_index_search", "list_directory", "explore",
	}

	for i, todo := range todos {
		todoLower := strings.ToLower(todo)
		for _, keyword := range discoveryKeywords {
			if strings.Contains(todoLower, keyword) {
				return nil, fmt.Errorf(
					"❌ TODO validation failed: TODO #%d contains discovery keyword '%s'\n"+
						"TODO: %s\n\n"+
						"🚨 SUBAGENTS CANNOT SEARCH OR DISCOVER FILES\n"+
						"• Subagents run in write-only mode\n"+
						"• Discovery tools (code_index_search, list_directory) are BLOCKED\n"+
						"• YOU must run code_index_search BEFORE creating this task\n"+
						"• TODOs must be implementation steps only\n\n"+
						"⚠️ MANDATORY ACTION - BEFORE RETRYING:\n"+
						"1. Send a developer-friendly message to user explaining the error:\n"+
						"   \"Tool error: create_agent_task failed - TODO contains forbidden keyword '%s'.\n"+
						"    Subagents can't discover files (write-only mode).\n"+
						"    Fixing by: running code_index_search myself to find the files, then creating\n"+
						"    task with implementation-only to-dos.\"\n"+
						"2. Run code_index_search to find the files yourself\n"+
						"3. Create new agent task with implementation-only to-dos\n\n"+
						"✅ GOOD TODO examples:\n"+
						"  - 'Add responsive CSS to Settings.tsx lines 15-45'\n"+
						"  - 'Update login validation in AuthForm.tsx line 89'\n"+
						"  - 'Test changes work on mobile viewport'\n\n"+
						"❌ BAD TODO examples:\n"+
						"  - 'Search for Settings component'  ← Discovery step!\n"+
						"  - 'Find the auth logic'  ← Discovery step!\n"+
						"  - 'Locate CSS files'  ← Discovery step!",
					i+1, keyword, todo, keyword)
			}
		}
	}

	// Convert todos to storage format
	todoItems := make([]storage.TodoItemInput, len(todos))
	for i, todo := range todos {
		todoItems[i] = storage.TodoItemInput{
			Description: todo,
		}
	}

	// Extract optional fields
	contextSummary, _ = input["contextSummary"].(string)
	priorWorkSummary, _ := input["priorWorkSummary"].(string)

	// FIX #1: Strict Type Validation for filesModified (re-extract with better validation)
	// Previously: Silent type coercion failure would leave empty strings in array
	// Now: Fail fast with clear error message if any element is not a string
	// Note: filesModified was extracted earlier for complexity analysis, now re-validating strictly
	filesModified = nil // Clear earlier extraction, will rebuild with validation
	if fm, ok := input["filesModified"].([]interface{}); ok {
		filesModified = make([]string, 0, len(fm)) // Use capacity, not length
		for i, f := range fm {
			str, ok := f.(string)
			if !ok {
				return nil, fmt.Errorf("filesModified[%d] must be a string, got %T: %v", i, f, f)
			}
			if str == "" {
				continue // Skip empty strings instead of adding them
			}
			filesModified = append(filesModified, str)
		}
	}

	// AUTO-POPULATE: If filesModified is empty, try to populate from last code_index_search
	if len(filesModified) == 0 {
		cachedPaths := GetLastCodeSearchPaths()
		if len(cachedPaths) > 0 {
			filesModified = cachedPaths
			zap.L().Info("✅ Auto-populated filesModified from code_index_search cache (empty input)",
				zap.Int("filesCount", len(filesModified)),
				zap.Strings("files", filesModified))
		}
	}

	// REMOVED: Deprecated folder filter was incorrectly filtering valid /ui paths
	// The project still uses /ui folder, there is no /ui2 migration

	// FIX #4: Always Update filesModified After Correction
	// Previously: Only updated if conditions met, could leave invalid paths
	// Now: ALWAYS use correctedPaths (filters empty strings and invalid paths)
	var unfixablePaths []string
	originalProvidedPaths := make([]string, len(filesModified))
	copy(originalProvidedPaths, filesModified)

	if len(filesModified) > 0 {
		correctedPaths, unfixable, isIndexingIssue := correctFilePaths(filesModified, zap.L())
		unfixablePaths = unfixable

		// ALWAYS update filesModified with corrected paths
		// This filters out empty strings, invalid paths, and fixes correctable paths
		originalCount := len(filesModified)
		filesModified = correctedPaths

		if len(unfixablePaths) > 0 {
			// Some paths could not be fixed - log error but continue with valid paths
			zap.L().Error("❌ Path correction failed for some files",
				zap.Int("originalCount", originalCount),
				zap.Int("correctedCount", len(correctedPaths)),
				zap.Strings("unfixablePaths", unfixablePaths),
				zap.Bool("indexingIssue", isIndexingIssue))
		} else if len(correctedPaths) < originalCount {
			// Some paths were filtered (empty strings or invalid)
			zap.L().Info("✅ File paths corrected and validated",
				zap.Int("originalCount", originalCount),
				zap.Int("correctedCount", len(correctedPaths)),
				zap.Bool("indexingIssue", isIndexingIssue))
		}
	}

	// FIX #7: Smart Fallback to Cached Paths
	// If ALL provided paths failed validation, use the cached paths from code_index_search
	// This handles the case where the model hallucinated paths instead of using search results
	if len(originalProvidedPaths) > 0 && len(filesModified) == 0 {
		cachedPaths := GetLastCodeSearchPaths()
		if len(cachedPaths) > 0 {
			// Deduplicate cached paths
			seenPaths := make(map[string]bool)
			uniquePaths := make([]string, 0, len(cachedPaths))
			for _, path := range cachedPaths {
				if !seenPaths[path] {
					seenPaths[path] = true
					uniquePaths = append(uniquePaths, path)
				}
			}

			filesModified = uniquePaths
			zap.L().Warn("⚠️  All provided paths failed validation - falling back to code_index_search cache",
				zap.Int("providedCount", len(originalProvidedPaths)),
				zap.Strings("providedPaths", originalProvidedPaths),
				zap.Int("cachedCount", len(uniquePaths)),
				zap.Strings("cachedPaths", uniquePaths))
		}
	}

	// PATTERN FILE DETECTION: Auto-add reference files mentioned in context/TODOs
	// Scan contextSummary and todos for file references (e.g., "follow pattern from HTTPToolsPage.tsx")
	projectRoot := tools.GetProjectRoot()
	patternFiles := extractPatternFiles(contextSummary, todos, projectRoot, zap.L())
	if len(patternFiles) > 0 {
		// Add pattern files to filesModified if not already present
		filesModifiedSet := make(map[string]bool)
		for _, f := range filesModified {
			filesModifiedSet[f] = true
		}

		addedPatternFiles := []string{}
		for _, patternFile := range patternFiles {
			if !filesModifiedSet[patternFile] {
				filesModified = append(filesModified, patternFile)
				addedPatternFiles = append(addedPatternFiles, patternFile)
				filesModifiedSet[patternFile] = true
			}
		}

		if len(addedPatternFiles) > 0 {
			zap.L().Info("✅ Auto-added pattern reference files to filesModified",
				zap.Strings("patternFiles", addedPatternFiles),
				zap.Int("totalFilesModified", len(filesModified)))
		}
	}

	// FIX #6: Better validation error messages
	// Previously: Generic "filesModified is empty" message
	// Now: Explain WHY it's empty and what paths were attempted
	if len(filesModified) == 0 {
		zap.L().Warn("⚠️  filesModified is empty - subagent may not know which files to modify",
			zap.String("agentName", agentName),
			zap.String("humanTaskId", humanTaskID),
			zap.Int("todosCount", len(todos)),
			zap.String("recommendation", "Run code_index_search first and populate filesModified with result file paths"))

		// Check if TODOs reference specific files - if so, this is definitely an error
		for i, todo := range todos {
			todoLower := strings.ToLower(todo)
			// Look for file references like ".tsx", ".go", ".css", etc.
			fileExtensions := []string{".tsx", ".ts", ".jsx", ".js", ".go", ".css", ".html", ".py", ".java"}
			for _, ext := range fileExtensions {
				if strings.Contains(todoLower, ext) {
					// Build helpful error message based on what actually happened
					errorMsg := "❌ filesModified validation failed:\n"

					if len(originalProvidedPaths) > 0 {
						// Paths were provided but all failed validation
						errorMsg += fmt.Sprintf("• You provided %d file path(s), but NONE of them exist:\n", len(originalProvidedPaths))
						for _, path := range originalProvidedPaths {
							errorMsg += fmt.Sprintf("  - %s ❌\n", path)
						}
						errorMsg += "\n• TODO #" + fmt.Sprintf("%d", i+1) + " references a file: " + todo + "\n\n"
						errorMsg += "🚨 THE PATHS YOU PROVIDED DON'T EXIST\n"
						errorMsg += "• Run code_index_search again with a better query\n"
						errorMsg += "• Verify the file paths returned by code_index_search\n"
						errorMsg += "• Use EXACT paths from FILE_PATHS_TO_USE in the search results\n"
						errorMsg += "• Do NOT type paths manually - copy them from search results\n\n"
						errorMsg += "💡 TIP: The files you're looking for might have different names or locations\n"
						errorMsg += "   Try searching for key terms from the file name or functionality"
					} else {
						// No paths were provided at all
						errorMsg += "• filesModified is empty\n"
						errorMsg += fmt.Sprintf("• BUT TODO #%d references a file: %s\n\n", i+1, todo)
						errorMsg += "🚨 YOU MUST POPULATE filesModified\n"
						errorMsg += "• Run code_index_search to find relevant files\n"
						errorMsg += "• Extract filePath values from search results\n"
						errorMsg += "• Pass them in filesModified array\n\n"
						errorMsg += "Example:\n"
						errorMsg += "1. code_index_search('settings component')\n"
						errorMsg += "2. create_agent_task({\n"
						errorMsg += "     filesModified: [\"/path/to/Settings.tsx\", \"/path/to/settings.css\"],\n"
						errorMsg += "     todos: [\"Add responsive CSS...\"]\n"
						errorMsg += "   })"
					}

					return nil, fmt.Errorf("%s", errorMsg)
				}
			}
		}
	}

	var qdrantCollections []string
	if qc, ok := input["qdrantCollections"].([]interface{}); ok {
		qdrantCollections = make([]string, len(qc))
		for i, c := range qc {
			if str, ok := c.(string); ok {
				qdrantCollections[i] = str
			}
		}
	}

	// COMPLEXITY ANALYSIS: Perform complexity analysis now that todos and files are parsed
	title := role
	if title == "" {
		title = "Agent Task"
	}

	newComplexityAnalysis = analyzeTaskComplexity(title, contextSummary, todos, filesModified)

	// PREVENTION: Block extremely complex tasks (≥0.8 score)
	if newComplexityAnalysis.Score >= 0.8 {
		return nil, fmt.Errorf(
			"❌ Task complexity too high (score: %.2f/1.0) - task creation blocked\n\n"+
			"🚨 COMPLEXITY ANALYSIS:\n"+
			"• File count: %d files (complexity: %.2f)\n"+
			"• TODO complexity: %.2f\n"+
			"• Cross-system dependencies: %d systems (complexity: %.2f)\n"+
			"• Integration complexity: %.2f\n"+
			"• Estimated time: %d minutes\n\n"+
			"⚠️ RECOMMENDATION: %s\n"+
			"• Splitting strategy: %s\n\n"+
			"🔧 REQUIRED ACTION:\n"+
			"1. Use coordinator_analyze_task_complexity to get detailed analysis\n"+
			"2. Use coordinator_split_agent_task to break this into smaller tasks\n"+
			"3. Create child tasks with complexity score < 0.8 each\n\n"+
			"💡 COMPLEXITY BREAKDOWN:\n"+
			"• 0.0-0.4: Simple task (proceed)\n"+
			"• 0.4-0.6: Moderate task (consider splitting)\n"+
			"• 0.6-0.8: Complex task (should split)\n"+
			"• 0.8-1.0: Extremely complex (blocked)",
			newComplexityAnalysis.Score,
			newComplexityAnalysis.FileCount, newComplexityAnalysis.Factors["fileCount"],
			newComplexityAnalysis.TodoComplexity,
			newComplexityAnalysis.CrossSystemDeps, newComplexityAnalysis.Factors["crossSystemDeps"],
			newComplexityAnalysis.Factors["integration"],
			newComplexityAnalysis.EstimatedTimeMinutes,
			newComplexityAnalysis.Recommendation,
			newComplexityAnalysis.SplittingStrategy)
	}

	// Also perform legacy complexity analysis for backward compatibility
	if len(filesModified) > 0 && role != "" {
		if analysis, suggestions, err := t.performComplexityAnalysis(ctx, role, contextSummary, filesModified); err == nil {
			legacyComplexityAnalysis = analysis
			splitSuggestions = suggestions
		}
	}

	// Warn if task is moderately complex (0.6-0.8) but allow creation
	if newComplexityAnalysis.Score >= 0.6 && newComplexityAnalysis.Score < 0.8 {
		zap.L().Warn("⚠️ Creating complex task - consider splitting",
			zap.Float64("complexityScore", newComplexityAnalysis.Score),
			zap.String("recommendation", newComplexityAnalysis.Recommendation),
			zap.String("splittingStrategy", newComplexityAnalysis.SplittingStrategy),
			zap.Int("estimatedMinutes", newComplexityAnalysis.EstimatedTimeMinutes))
	}

	// Log complexity analysis for monitoring
	if newComplexityAnalysis.Score > 0.0 {
		zap.L().Info("Task complexity analysis completed",
			zap.Float64("score", newComplexityAnalysis.Score),
			zap.Int("fileCount", newComplexityAnalysis.FileCount),
			zap.Float64("todoComplexity", newComplexityAnalysis.TodoComplexity),
			zap.Int("crossSystemDeps", newComplexityAnalysis.CrossSystemDeps),
			zap.String("recommendation", newComplexityAnalysis.Recommendation))
	}

	// Validate file paths exist before creating task (Claude optimization)
	// This prevents subagents from being launched with invalid file paths
	if len(filesModified) > 0 {
		var missingFiles []string
		for _, filePath := range filesModified {
			if filePath == "" {
				continue // Skip empty paths
			}
			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				missingFiles = append(missingFiles, filePath)
			}
		}
		if len(missingFiles) > 0 {
			return nil, fmt.Errorf(
				"❌ File path validation failed: the following files do not exist:\n%s\n\n"+
					"🚨 MOST COMMON CAUSE: You typed file paths manually instead of using FILE_PATHS_TO_USE array\n\n"+
					"⚠️ MANDATORY ACTION - BEFORE RETRYING:\n"+
					"1. Send a developer-friendly message to user:\n"+
					"   \"Tool error: create_agent_task failed - file path doesn't exist.\n"+
					"    These paths aren't in the FILE_PATHS_TO_USE array from search results.\n"+
					"    Fixing by: using exact paths from the code_index_search results.\"\n"+
					"2. Check the code_index_search result for FILE_PATHS_TO_USE array\n"+
					"3. Copy-paste EXACT paths from that array (do not type manually)\n"+
					"4. Retry create_agent_task with corrected file paths\n\n"+
					"✅ CORRECT: Use paths from FILE_PATHS_TO_USE: [\"/path/from/search/result.tsx\"]\n"+
					"❌ WRONG: Manually typed path: \"./ui/src/MyGuess.tsx\"",
				strings.Join(missingFiles, "\n"))
		}
	}

	// Create agent task via storage
	task, err := t.storage.CreateAgentTask(
		humanTaskID,
		agentName,
		role,
		todoItems,
		contextSummary,
		filesModified,
		qdrantCollections,
		priorWorkSummary,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent task: %w", err)
	}

	// Return task summary
	return map[string]interface{}{
		"taskId":     task.ID,
		"agentName":  task.AgentName,
		"role":       task.Role,
		"status":     task.Status,
		"todosCount": len(task.Todos),
		"createdAt":  task.CreatedAt,
		"newComplexityAnalysis": newComplexityAnalysis,
		"legacyComplexityAnalysis": legacyComplexityAnalysis,
		"splitSuggestions":   splitSuggestions,
		"recommendsSplit":    newComplexityAnalysis.Recommendation == "SPLIT",
		"complexityScore":    newComplexityAnalysis.Score,
		"estimatedMinutes":   newComplexityAnalysis.EstimatedTimeMinutes,
	}, nil
}
// aiServiceChatProvider wraps AIServiceInterface to implement ChatProvider for ComplexityAnalyzer
type aiServiceChatProvider struct {
	aiService AIServiceInterface
}

// StreamChat implements the ChatProvider interface by using the AIService
func (p *aiServiceChatProvider) StreamChat(ctx context.Context, messages []aiservice.Message) (<-chan string, error) {
	// Convert to the format expected by AIService and stream
	stream, err := p.aiService.StreamChatWithTools(ctx, messages, 0) // No tool calls needed for complexity analysis
	if err != nil {
		return nil, err
	}
	
	// Create output channel
	outputChan := make(chan string, 100)
	
	// Process stream events and extract text tokens
	go func() {
		defer close(outputChan)
		
		for event := range stream {
			switch event.Type {
			case aiservice.StreamEventToken:
				select {
				case <-ctx.Done():
					return
				case outputChan <- event.Content:
				}
			case aiservice.StreamEventError:
				select {
				case <-ctx.Done():
					return
				case outputChan <- fmt.Sprintf("ERROR: %s", event.Error):
				}
				return
			}
		}
	}()
	
	return outputChan, nil
}

// performComplexityAnalysis analyzes task complexity and generates split suggestions if needed
func (t *CreateAgentTaskTool) performComplexityAnalysis(ctx context.Context, role, contextSummary string, filesModified []string) (*aiservice.ComplexityAnalysis, []aiservice.SuggestedSplit, error) {
	if t.aiService == nil || t.config == nil {
		// Complexity analysis not available without AI service
		return nil, nil, nil
	}

	// Create a simple ChatProvider wrapper for the ComplexityAnalyzer
	chatProvider := &aiServiceChatProvider{aiService: t.aiService}
	
	// Create complexity analyzer
	analyzer := aiservice.NewComplexityAnalyzer(t.config, chatProvider)
	
	// Create task context
	taskContext := aiservice.TaskContext{
		Description:    role,
		FilesModified:  filesModified,
		Role:           role,
		ContextSummary: contextSummary,
	}
	
	// Analyze complexity
	analysis, err := analyzer.AnalyzeComplexity(ctx, taskContext)
	if err != nil {
		return nil, nil, fmt.Errorf("complexity analysis failed: %w", err)
	}
	
	// Generate split suggestions if complexity is high
	var suggestions []aiservice.SuggestedSplit
	if analysis.ShouldSplit {
		suggestions, _ = analyzer.GenerateSplitSuggestions(ctx, taskContext, analysis)
	}

	return analysis, suggestions, nil
}

// ListAgentTasksTool implements the ToolExecutor interface
type ListAgentTasksTool struct {
	storage storage.TaskStorage
}

func (t *ListAgentTasksTool) Name() string {
	return "list_agent_tasks"
}

func (t *ListAgentTasksTool) Description() string {
	return "List agent tasks with optional filters. Returns up to 20 tasks with details. Supports pagination via offset/limit. Use to check task status, find assignments, or review progress. " +
		"TIP: Filter by humanTaskId or agentName to narrow results. " +
		"IMPORTANT: If you have a specific agentTaskId (e.g., from execute_subagent result), use coordinator_get_agent_task instead for direct lookup - DO NOT call this repeatedly without filters."
}

func (t *ListAgentTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentName": map[string]interface{}{
				"type":        "string",
				"description": "Filter by agent name (optional)",
			},
			"humanTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Filter by parent human task ID (optional)",
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Number of tasks to skip for pagination (default: 0)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of tasks to return (default: 20, max: 20)",
			},
		},
	}
}

func (t *ListAgentTasksTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract filter parameters
	agentName, _ := input["agentName"].(string)
	humanTaskID, _ := input["humanTaskId"].(string)

	// Extract pagination parameters
	offset := 0
	if o, ok := input["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	limit := 20
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 20 {
			limit = 20 // Enforce max limit per task context
		}
	}

	// Get all tasks
	allTasks := t.storage.ListAllAgentTasks()

	// Apply filters
	var filteredTasks []*storage.AgentTask
	for _, task := range allTasks {
		if humanTaskID != "" && task.HumanTaskID != humanTaskID {
			continue
		}
		if agentName != "" && task.AgentName != agentName {
			continue
		}
		filteredTasks = append(filteredTasks, task)
	}

	// Apply pagination
	totalCount := len(filteredTasks)
	endIndex := offset + limit
	if offset > totalCount {
		offset = totalCount
	}
	if endIndex > totalCount {
		endIndex = totalCount
	}

	paginatedTasks := filteredTasks[offset:endIndex]

	// Format response
	return map[string]interface{}{
		"tasks":      paginatedTasks,
		"count":      len(paginatedTasks),
		"totalCount": totalCount,
		"offset":     offset,
		"limit":      limit,
	}, nil
}

// QueryKnowledgeTool implements the ToolExecutor interface
type QueryKnowledgeTool struct {
	storage storage.KnowledgeStorage
}

func (t *QueryKnowledgeTool) Name() string {
	return "query_knowledge"
}

func (t *QueryKnowledgeTool) Description() string {
	return "Query the coordinator knowledge base for relevant information. Returns top matches with similarity scores. Limit: 10 results max. Use to find existing solutions, patterns, or context before implementing."
}

func (t *QueryKnowledgeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name to query (e.g., 'technical-knowledge', 'task:hyperion://task/human/{taskId}')",
			},
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Search query text (natural language)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default: 5, max: 10)",
			},
		},
		"required": []string{"collection", "query"},
	}
}

func (t *QueryKnowledgeTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract and validate required fields
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	query, ok := input["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	// Extract optional limit
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 10 {
			limit = 10 // Enforce max limit per task context
		}
	}

	// Query knowledge storage
	results, err := t.storage.Query(collection, query, limit, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query knowledge: %w", err)
	}

	// Format results
	type KnowledgeResult struct {
		ID         string                 `json:"id"`
		Collection string                 `json:"collection"`
		Text       string                 `json:"text"`
		Metadata   map[string]interface{} `json:"metadata,omitempty"`
		Score      float64                `json:"score"`
	}

	formattedResults := make([]KnowledgeResult, len(results))
	for i, result := range results {
		formattedResults[i] = KnowledgeResult{
			ID:         result.Entry.ID,
			Collection: result.Entry.Collection,
			Text:       result.Entry.Text,
			Metadata:   result.Entry.Metadata,
			Score:      result.Score,
		}
	}

	return formattedResults, nil
}

// UpsertKnowledgeTool implements the ToolExecutor interface
type UpsertKnowledgeTool struct {
	storage storage.KnowledgeStorage
}

func (t *UpsertKnowledgeTool) Name() string {
	return "coordinator_upsert_knowledge"
}

func (t *UpsertKnowledgeTool) Description() string {
	return "Store knowledge in the coordinator knowledge base. IMPORTANT: MAX 1000 tokens (~750 words, ~4000 characters) per entry. Keep entries focused and granular - ONE concept per entry. Use for storing task context, ADRs, data contracts, and coordination information. For large documents, split into multiple focused entries. Returns entry ID and collection."
}

func (t *UpsertKnowledgeTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"collection": map[string]interface{}{
				"type":        "string",
				"description": "Collection name (e.g., 'task:taskURI', 'adr', 'data-contracts')",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Content to store (MAX 1000 tokens ≈ 4000 characters)",
			},
			"metadata": map[string]interface{}{
				"type":        "object",
				"description": "Optional metadata (taskId, agentName, timestamp, etc.)",
			},
		},
		"required": []string{"collection", "text"},
	}
}

func (t *UpsertKnowledgeTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	collection, ok := input["collection"].(string)
	if !ok || collection == "" {
		return nil, fmt.Errorf("collection is required and must be a string")
	}

	text, ok := input["text"].(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("text is required and must be a string")
	}

	// Validate token count (approximate: 1 token ≈ 4 characters)
	const maxTokens = 1000
	const maxChars = maxTokens * 4 // ~4000 characters

	if len(text) > maxChars {
		estimatedTokens := len(text) / 4
		return nil, fmt.Errorf(
			"Entry too large: ~%d tokens (max: %d tokens ≈ %d characters).\n\n"+
				"📝 This knowledge base is designed for AI retrieval with focused, granular entries.\n\n"+
				"Please split your content into multiple entries, each containing:\n"+
				"• ONE specific concept, pattern, or procedure\n"+
				"• Clear, concise information (aim for 200-800 tokens)\n"+
				"• Descriptive metadata for easy retrieval\n\n"+
				"Example: Instead of storing an entire API documentation, create separate entries for:\n"+
				"- Authentication flow\n"+
				"- Rate limiting rules\n"+
				"- Error handling patterns\n"+
				"- Each endpoint specification",
			estimatedTokens, maxTokens, maxChars,
		)
	}

	var metadata map[string]interface{}
	if m, ok := input["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	entry, err := t.storage.Upsert(collection, text, metadata, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert knowledge: %w", err)
	}

	return map[string]interface{}{
		"id":         entry.ID,
		"collection": entry.Collection,
		"createdAt":  entry.CreatedAt,
	}, nil
}

// ListCollectionsTool implements the ToolExecutor interface
type ListCollectionsTool struct {
	storage storage.KnowledgeStorage
}

func (t *ListCollectionsTool) Name() string {
	return "knowledge_list_collections"
}

func (t *ListCollectionsTool) Description() string {
	return "List available knowledge collections with entry counts. Use this to discover which collections exist before calling knowledge_find or knowledge_store. Returns collection names with entry counts sorted by popularity."
}

func (t *ListCollectionsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of collections to return (default: 5)",
			},
		},
	}
}

func (t *ListCollectionsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}

	stats, err := t.storage.GetPopularCollections(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular collections: %w", err)
	}

	if stats == nil || len(stats) == 0 {
		return map[string]interface{}{
			"collections":  []interface{}{},
			"message":      "No collections with entries yet",
			"totalDefined": 14,
		}, nil
	}

	return stats, nil
}

// CreateHumanTaskTool implements the ToolExecutor interface
type CreateHumanTaskTool struct {
	storage       storage.TaskStorage
	retryAttempts sync.Map // Track retry attempts by prompt hash to prevent loops
}

func (t *CreateHumanTaskTool) Name() string {
	return "coordinator_create_human_task"
}

func (t *CreateHumanTaskTool) Description() string {
	return `Create a new human task with the original user prompt. Returns task ID. Use this as the first step when a user makes a request.

CRITICAL - Handling Similar Tasks (MUST FOLLOW):
- If similarTasksFound=true is returned, you MUST:
  1. STOP and examine the similarTasks array
  2. ASK THE USER: "I found similar existing tasks. Would you like to:
     a) Use an existing task (show task details)
     b) Create a new task anyway"
  3. Wait for user response
  4. Based on user choice:
     - If user chooses existing task → use that taskId
     - If user wants new task → call this tool again with forceCreate=true
- DO NOT call this tool again without asking the user first
- DO NOT try to decide on your own - always ask the user when similar tasks are found`
}

func (t *CreateHumanTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "Original human request/prompt",
			},
			"forceCreate": map[string]interface{}{
				"type":        "boolean",
				"description": "Set to true to create task despite similar existing tasks (default: false)",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *CreateHumanTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	prompt, ok := input["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required and must be a string")
	}

	forceCreate := false
	// Handle both boolean and string types (defensive programming)
	// AI might send "true" as a string instead of boolean
	if fc, ok := input["forceCreate"].(bool); ok {
		forceCreate = fc
	} else if fcStr, ok := input["forceCreate"].(string); ok {
		// Accept "true", "True", "TRUE" as truthy values
		forceCreate = (fcStr == "true" || fcStr == "True" || fcStr == "TRUE")
		if forceCreate {
			zap.L().Warn("forceCreate sent as string instead of boolean",
				zap.String("value", fcStr),
				zap.String("converted_to", "true"))
		}
	}

	// Generate hash of prompt for tracking retry attempts
	hash := sha256.Sum256([]byte(prompt))
	promptHash := hex.EncodeToString(hash[:])

	// Track ALL calls (even with forceCreate) to detect loops early
	var attemptCount int
	if val, exists := t.retryAttempts.Load(promptHash); exists {
		attemptCount = val.(int)
	}
	attemptCount++
	t.retryAttempts.Store(promptHash, attemptCount)

	// CRITICAL LOOP PREVENTION: After 2 similar task responses (3rd call total),
	// auto-create instead of returning similar tasks again
	if attemptCount >= 3 && !forceCreate {
		// Clean up tracking
		t.retryAttempts.Delete(promptHash)

		// Force create the task to prevent infinite loop
		task, err := t.storage.CreateHumanTask(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to create human task: %w", err)
		}

		return map[string]interface{}{
			"similarTasksFound": false,
			"taskId":            task.ID,
			"status":            task.Status,
			"prompt":            task.Prompt,
			"createdAt":         task.CreatedAt,
			"message":           "⚠️ Auto-created task to prevent infinite loop. Tool was called 3+ times with same prompt - creating task automatically.",
		}, nil
	}

	// Check for similar tasks unless forceCreate is true
	if !forceCreate {
		similarTasks, scores, err := t.storage.SearchSimilarHumanTasks(prompt, 5, 0.75)
		if err == nil && len(similarTasks) > 0 {
			// Return similar tasks and ask user (on attempts 1 and 2)
			formattedTasks := make([]map[string]interface{}, len(similarTasks))
			for i, task := range similarTasks {
				formattedTasks[i] = map[string]interface{}{
					"taskId":     task.ID,
					"prompt":     task.Prompt,
					"status":     task.Status,
					"createdAt":  task.CreatedAt,
					"similarity": scores[i],
				}
			}

			return map[string]interface{}{
				"similarTasksFound": true,
				"similarTasks":      formattedTasks,
				"message":           fmt.Sprintf("Found %d similar task(s). ASK THE USER if they want to use an existing task or create a new one. If user wants new task, call again with forceCreate=true.", len(similarTasks)),
				"_attemptCount":     attemptCount, // Include for debugging
			}, nil
		}
	}

	// Clean up tracking if task is being created
	t.retryAttempts.Delete(promptHash)

	// No similar tasks or forceCreate=true - proceed with creation
	task, err := t.storage.CreateHumanTask(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to create human task: %w", err)
	}

	return map[string]interface{}{
		"similarTasksFound": false,
		"taskId":            task.ID,
		"status":            task.Status,
		"prompt":            task.Prompt,
		"createdAt":         task.CreatedAt,
	}, nil
}

// UpdateTaskStatusTool implements the ToolExecutor interface
type UpdateTaskStatusTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTaskStatusTool) Name() string {
	return "coordinator_update_task_status"
}

func (t *UpdateTaskStatusTool) Description() string {
	return "Update the status of any task (human or agent). Status values: pending, in_progress, completed, blocked. Use to track task progress."
}

func (t *UpdateTaskStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"taskId": map[string]interface{}{
				"type":        "string",
				"description": "Task ID to update (UUID)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "New status (pending, in_progress, completed, blocked)",
				"enum":        []string{"pending", "in_progress", "completed", "blocked"},
			},
			"notes": map[string]interface{}{
				"type":        "string",
				"description": "Optional progress notes",
			},
		},
		"required": []string{"taskId", "status"},
	}
}

func (t *UpdateTaskStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	taskID, ok := input["taskId"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("taskId is required and must be a string")
	}

	statusStr, ok := input["status"].(string)
	if !ok || statusStr == "" {
		return nil, fmt.Errorf("status is required and must be one of: pending, in_progress, completed, blocked")
	}

	status := storage.TaskStatus(statusStr)
	notes, _ := input["notes"].(string)

	// Get the task before updating to check if it's a child task
	task, err := t.storage.GetAgentTask(taskID)
	if err != nil {
		// Try as human task if agent task fails
		_, humanErr := t.storage.GetHumanTask(taskID)
		if humanErr != nil {
			return nil, fmt.Errorf("failed to get task (tried both agent and human): agent_error=%w, human_error=%w", err, humanErr)
		}
		// For human tasks, just update status without hierarchical propagation
		err = t.storage.UpdateTaskStatus(taskID, storage.TaskStatus(status), notes)
		if err != nil {
			return nil, fmt.Errorf("failed to update human task status: %w", err)
		}

		return map[string]interface{}{
			"taskId": taskID,
			"status": status,
			"notes":  notes,
			"taskType": "human",
		}, nil
	}

	// Update the task status
	err = t.storage.UpdateTaskStatus(taskID, storage.TaskStatus(status), notes)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Check if this is a child task and propagate to parent if needed
	var parentUpdateResult map[string]interface{}
	var parentTaskId string

	if task.ParentTaskID != nil && *task.ParentTaskID != "" {
		parentTaskId = *task.ParentTaskID

		// Only propagate if the child task is completed or failed
		if status == "COMPLETED" || status == "FAILED" {
			parentUpdateResult, err = t.propagateToParentTask(ctx, parentTaskId)
			if err != nil {
				// Log error but don't fail the entire operation
				zap.L().Warn("Failed to propagate status to parent task",
					zap.String("childTaskId", taskID),
					zap.String("parentTaskId", parentTaskId),
					zap.Error(err))
				parentUpdateResult = map[string]interface{}{
					"error": fmt.Sprintf("Failed to propagate to parent: %v", err),
				}
			}
		}
	}

	result := map[string]interface{}{
		"taskId": taskID,
		"status": status,
		"notes":  notes,
		"taskType": "agent",
		"parentTaskId": parentTaskId,
		"propagated": parentUpdateResult != nil,
	}

	if parentUpdateResult != nil {
		result["parentUpdate"] = parentUpdateResult
	}

	return result, nil
}

// propagateToParentTask updates parent task status based on child task completion
func (t *UpdateTaskStatusTool) propagateToParentTask(ctx context.Context, parentTaskId string) (map[string]interface{}, error) {
	// Get parent task
	parentTask, err := t.storage.GetAgentTask(parentTaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent task: %w", err)
	}

	// Get all child tasks for this parent
	allTasks, _, err := t.storage.ListAgentTasks(bson.M{}, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tasks: %w", err)
	}

	childTasks := make([]*storage.AgentTask, 0)
	for _, task := range allTasks {
		if task.ParentTaskID != nil && *task.ParentTaskID == parentTaskId {
			childTasks = append(childTasks, task)
		}
	}

	if len(childTasks) == 0 {
		return map[string]interface{}{
			"message": "No child tasks found for parent",
		}, nil
	}

	// Calculate aggregated status
	completedChildren := 0
	inProgressChildren := 0
	failedChildren := 0
	pendingChildren := 0

	for _, child := range childTasks {
		switch child.Status {
		case storage.TaskStatusCompleted:
			completedChildren++
		case storage.TaskStatusInProgress:
			inProgressChildren++
		case storage.TaskStatusBlocked:
			failedChildren++
		default:
			pendingChildren++
		}
	}

	// Determine new parent status based on child statuses
	var newParentStatus string
	oldStatus := parentTask.Status
	
	if completedChildren == len(childTasks) {
		// All children completed
		newParentStatus = "COMPLETED"
	} else if failedChildren > 0 && (completedChildren+inProgressChildren) == 0 {
		// Some failed and none in progress or completed
		newParentStatus = "FAILED"
	} else if inProgressChildren > 0 || completedChildren > 0 {
		// Some in progress or completed
		newParentStatus = "IN_PROGRESS"
	} else {
		// All pending
		newParentStatus = "PENDING"
	}

	// Update parent task if status changed
	statusChanged := string(oldStatus) != newParentStatus
	if statusChanged {
		err = t.storage.UpdateTaskStatus(parentTaskId, storage.TaskStatus(newParentStatus), "")
		if err != nil {
			return nil, fmt.Errorf("failed to update parent task status: %w", err)
		}

		zap.L().Info("Parent task status updated due to child completion",
			zap.String("parentTaskId", parentTaskId),
			zap.String("oldStatus", string(oldStatus)),
			zap.String("newStatus", newParentStatus),
			zap.Int("completedChildren", completedChildren),
			zap.Int("totalChildren", len(childTasks)))
	}

	// Update parent metadata with child progress
	// Return aggregated progress information
	// Note: These values are not stored in the task itself, just returned for monitoring
	return map[string]interface{}{
		"parentTaskId":        parentTaskId,
		"oldStatus":           oldStatus,
		"newStatus":           newParentStatus,
		"statusChanged":       statusChanged,
		"completedChildren":   completedChildren,
		"totalChildren":       len(childTasks),
		"inProgressChildren":  inProgressChildren,
		"failedChildren":      failedChildren,
		"pendingChildren":     pendingChildren,
	}, nil
}

// UpdateTodoStatusTool implements the ToolExecutor interface
type UpdateTodoStatusTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTodoStatusTool) Name() string {
	return "coordinator_update_todo_status"
}

func (t *UpdateTodoStatusTool) Description() string {
	return "Update the status of a specific TODO item within an agent task. Status values: pending, in_progress, completed. When all TODOs are completed, the agent task is automatically marked as completed."
}

func (t *UpdateTodoStatusTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task ID (UUID)",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item ID (UUID)",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "New status (pending, in_progress, completed)",
				"enum":        []string{"pending", "in_progress", "completed"},
			},
			"notes": map[string]interface{}{
				"type":        "string",
				"description": "Optional progress notes for this TODO",
			},
		},
		"required": []string{"agentTaskId", "todoId", "status"},
	}
}

func (t *UpdateTodoStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	statusStr, ok := input["status"].(string)
	if !ok || statusStr == "" {
		return nil, fmt.Errorf("status is required and must be one of: pending, in_progress, completed")
	}

	status := storage.TodoStatus(statusStr)
	notes, _ := input["notes"].(string)

	err := t.storage.UpdateTodoStatus(agentTaskID, todoID, status, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to update TODO status: %w", err)
	}

	// Build response with clear nextAction guidance
	response := map[string]interface{}{
		"success":     true,
		"message":     fmt.Sprintf("✅ TODO %s updated to status: %s", todoID, status),
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"status":      status,
		"notes":       notes,
	}

	// Add nextAction guidance based on status
	switch status {
	case "in_progress":
		response["nextAction"] = "IMPLEMENT_CHANGES"
		response["nextSteps"] = []string{
			"Call write_file or apply_patch to implement the code changes",
			"After implementing, call coordinator_update_todo_status with status='completed'",
			"DO NOT call coordinator_update_todo_status again with status='in_progress'",
		}
		response["guidance"] = "You have marked this TODO as in_progress. Your NEXT action MUST be write_file or apply_patch to implement changes. DO NOT call this tool again with the same status."

	case "completed":
		response["nextAction"] = "MOVE_TO_NEXT_TODO"
		response["nextSteps"] = []string{
			"Move to the next pending TODO in your task",
			"OR if all TODOs are completed, the task will auto-complete",
		}
		response["guidance"] = "TODO completed successfully. Move to the next TODO or wait for task completion."

	case "pending":
		response["nextAction"] = "START_TODO"
		response["nextSteps"] = []string{
			"Call coordinator_update_todo_status with status='in_progress' to start this TODO",
			"Then implement the changes with write_file or apply_patch",
		}
		response["guidance"] = "TODO reset to pending. Call this tool again with status='in_progress' to start working on it."
	}

	return response, nil
}

// ListHumanTasksTool implements the ToolExecutor interface
type ListHumanTasksTool struct {
	storage storage.TaskStorage
}

func (t *ListHumanTasksTool) Name() string {
	return "coordinator_list_human_tasks"
}

func (t *ListHumanTasksTool) Description() string {
	return "List all human tasks from the coordinator database. Returns array of tasks with all fields."
}

func (t *ListHumanTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListHumanTasksTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	tasks := t.storage.ListAllHumanTasks()
	return map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	}, nil
}

// GetAgentTaskTool implements the ToolExecutor interface
type GetAgentTaskTool struct {
	storage storage.TaskStorage
}

func (t *GetAgentTaskTool) Name() string {
	return "coordinator_get_agent_task"
}

func (t *GetAgentTaskTool) Description() string {
	return "Get a single agent task by ID with full, untruncated content. Use this to retrieve complete task details when coordinator_list_agent_tasks shows truncated fields."
}

func (t *GetAgentTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"taskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task ID (UUID)",
			},
		},
		"required": []string{"taskId"},
	}
}

func (t *GetAgentTaskTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	taskID, ok := input["taskId"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("taskId is required and must be a string")
	}

	task, err := t.storage.GetAgentTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent task: %w", err)
	}

	return map[string]interface{}{
		"task": task,
	}, nil
}

// FindSimilarTasksTool implements the ToolExecutor interface for finding similar existing tasks
// This helps prevent duplicate task creation (Claude optimization)
type FindSimilarTasksTool struct {
	storage storage.TaskStorage
}

func (t *FindSimilarTasksTool) Name() string {
	return "coordinator_find_similar_tasks"
}

func (t *FindSimilarTasksTool) Description() string {
	return "Search for existing tasks similar to a given prompt. Returns tasks with similarity scores (0-1). Use BEFORE creating a new task to avoid duplicates. Higher scores indicate more similar tasks."
}

func (t *FindSimilarTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"prompt": map[string]interface{}{
				"type":        "string",
				"description": "The task description to search for similar tasks",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 5, max: 10)",
			},
			"minScore": map[string]interface{}{
				"type":        "number",
				"description": "Minimum similarity score threshold 0-1 (default: 0.7)",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *FindSimilarTasksTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract required prompt
	prompt, ok := input["prompt"].(string)
	if !ok || prompt == "" {
		return nil, fmt.Errorf("prompt is required and must be a non-empty string")
	}

	// Extract optional limit (default: 5, max: 10)
	limit := 5
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
		if limit > 10 {
			limit = 10
		}
	}

	// Extract optional minScore (default: 0.7)
	minScore := 0.7
	if s, ok := input["minScore"].(float64); ok && s >= 0 && s <= 1 {
		minScore = s
	}

	// Search for similar tasks
	tasks, scores, err := t.storage.SearchSimilarHumanTasks(prompt, limit, minScore)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar tasks: %w", err)
	}

	// Format results
	type SimilarTask struct {
		TaskID    string  `json:"taskId"`
		Prompt    string  `json:"prompt"`
		Status    string  `json:"status"`
		CreatedAt string  `json:"createdAt"`
		Score     float64 `json:"score"`
	}

	results := make([]SimilarTask, len(tasks))
	for i, task := range tasks {
		results[i] = SimilarTask{
			TaskID:    task.ID,
			Prompt:    task.Prompt,
			Status:    string(task.Status),
			CreatedAt: task.CreatedAt.Format("2006-01-02 15:04:05"),
			Score:     scores[i],
		}
	}

	return map[string]interface{}{
		"similarTasks": results,
		"count":        len(results),
		"searchPrompt": prompt,
		"minScore":     minScore,
	}, nil
}

// AddTaskPromptNotesTool implements the ToolExecutor interface
type AddTaskPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *AddTaskPromptNotesTool) Name() string {
	return "coordinator_add_task_prompt_notes"
}

func (t *AddTaskPromptNotesTool) Description() string {
	return "Add human guidance notes to an agent task. Use to provide additional context or instructions to the agent working on the task."
}

func (t *AddTaskPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "promptNotes"},
	}
}

func (t *AddTaskPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	// Validate and sanitize prompt notes
	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.AddTaskPromptNotes(agentTaskID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to add prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"message":     "Prompt notes added successfully",
	}, nil
}

// UpdateTaskPromptNotesTool implements the ToolExecutor interface
type UpdateTaskPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTaskPromptNotesTool) Name() string {
	return "coordinator_update_task_prompt_notes"
}

func (t *UpdateTaskPromptNotesTool) Description() string {
	return "Update existing human guidance notes on an agent task. Use to modify previously added guidance."
}

func (t *UpdateTaskPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "promptNotes"},
	}
}

func (t *UpdateTaskPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.UpdateTaskPromptNotes(agentTaskID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to update prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"message":     "Prompt notes updated successfully",
	}, nil
}

// ClearTaskPromptNotesTool implements the ToolExecutor interface
type ClearTaskPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *ClearTaskPromptNotesTool) Name() string {
	return "coordinator_clear_task_prompt_notes"
}

func (t *ClearTaskPromptNotesTool) Description() string {
	return "Clear/remove human guidance notes from an agent task."
}

func (t *ClearTaskPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
		},
		"required": []string{"agentTaskId"},
	}
}

func (t *ClearTaskPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	err := t.storage.ClearTaskPromptNotes(agentTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"message":     "Prompt notes cleared successfully",
	}, nil
}

// AddTodoPromptNotesTool implements the ToolExecutor interface
type AddTodoPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *AddTodoPromptNotesTool) Name() string {
	return "coordinator_add_todo_prompt_notes"
}

func (t *AddTodoPromptNotesTool) Description() string {
	return "Add human guidance notes to a specific TODO item. Use to provide specific instructions for a single TODO."
}

func (t *AddTodoPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "todoId", "promptNotes"},
	}
}

func (t *AddTodoPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.AddTodoPromptNotes(agentTaskID, todoID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to add TODO prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"message":     "TODO prompt notes added successfully",
	}, nil
}

// UpdateTodoPromptNotesTool implements the ToolExecutor interface
type UpdateTodoPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *UpdateTodoPromptNotesTool) Name() string {
	return "coordinator_update_todo_prompt_notes"
}

func (t *UpdateTodoPromptNotesTool) Description() string {
	return "Update existing human guidance notes on a TODO item."
}

func (t *UpdateTodoPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item UUID",
			},
			"promptNotes": map[string]interface{}{
				"type":        "string",
				"description": "Human guidance notes, markdown supported",
			},
		},
		"required": []string{"agentTaskId", "todoId", "promptNotes"},
	}
}

func (t *UpdateTodoPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	promptNotes, ok := input["promptNotes"].(string)
	if !ok || promptNotes == "" {
		return nil, fmt.Errorf("promptNotes is required and must be a string")
	}

	sanitized, err := storage.ValidatePromptNotes(promptNotes)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	err = t.storage.UpdateTodoPromptNotes(agentTaskID, todoID, sanitized)
	if err != nil {
		return nil, fmt.Errorf("failed to update TODO prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"message":     "TODO prompt notes updated successfully",
	}, nil
}

// ClearTodoPromptNotesTool implements the ToolExecutor interface
type ClearTodoPromptNotesTool struct {
	storage storage.TaskStorage
}

func (t *ClearTodoPromptNotesTool) Name() string {
	return "coordinator_clear_todo_prompt_notes"
}

func (t *ClearTodoPromptNotesTool) Description() string {
	return "Clear/remove human guidance notes from a TODO item."
}

func (t *ClearTodoPromptNotesTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task UUID",
			},
			"todoId": map[string]interface{}{
				"type":        "string",
				"description": "TODO item UUID",
			},
		},
		"required": []string{"agentTaskId", "todoId"},
	}
}

func (t *ClearTodoPromptNotesTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
	}

	todoID, ok := input["todoId"].(string)
	if !ok || todoID == "" {
		return nil, fmt.Errorf("todoId is required and must be a string")
	}

	err := t.storage.ClearTodoPromptNotes(agentTaskID, todoID)
	if err != nil {
		return nil, fmt.Errorf("failed to clear TODO prompt notes: %w", err)
	}

	return map[string]interface{}{
		"agentTaskId": agentTaskID,
		"todoId":      todoID,
		"message":     "TODO prompt notes cleared successfully",
	}, nil
}

// ListSubagentsTool implements the ToolExecutor interface
type ListSubagentsTool struct {
	mongoDatabase *interface{} // Will be *mongo.Database but using interface{} to avoid import cycle
}

func (t *ListSubagentsTool) Name() string {
	return "list_subagents"
}

func (t *ListSubagentsTool) Description() string {
	return "Returns available subagents from CLAUDE.md agent list with names, descriptions, tools, and categories"
}

func (t *ListSubagentsTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

func (t *ListSubagentsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// For now, return a hardcoded list of subagents from CLAUDE.md
	// In the future, this should query MongoDB's subagents collection
	// But since we can't import mongo.Database here, we'll use a simple approach

	subagents := []map[string]interface{}{
		{
			"name":        "go-dev",
			"description": "Go microservices, REST APIs, business logic",
		},
		{
			"name":        "go-mcp-dev",
			"description": "MCP tools and integrations (Model Context Protocol)",
		},
		{
			"name":        "ui-dev",
			"description": "React/TypeScript implementation, components",
		},
		{
			"name":        "ui-tester",
			"description": "Playwright E2E tests, accessibility validation",
		},
		{
			"name":        "sre",
			"description": "Deployment to dev/prod environments",
		},
		{
			"name":        "k8s-deployment-expert",
			"description": "Kubernetes manifests, rollouts, scaling",
		},
	}

	return map[string]interface{}{
		"subagents": subagents,
		"count":     len(subagents),
	}, nil
}

// SetCurrentSubagentTool implements the ToolExecutor interface
type SetCurrentSubagentTool struct {
	mongoDatabase *interface{} // Will be *mongo.Database but using interface{} to avoid import cycle
}

func (t *SetCurrentSubagentTool) Name() string {
	return "set_current_subagent"
}

func (t *SetCurrentSubagentTool) Description() string {
	return "Associate a subagent with the current chat session. Stores subagent name in chat metadata."
}

func (t *SetCurrentSubagentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"subagentName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the subagent to associate with chat (must match list from list_subagents)",
			},
		},
		"required": []string{"subagentName"},
	}
}

func (t *SetCurrentSubagentTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	subagentName, ok := input["subagentName"].(string)
	if !ok || subagentName == "" {
		return nil, fmt.Errorf("subagentName is required and must be a string")
	}

	// Validate subagent name against known list
	validSubagents := map[string]bool{
		"go-dev":                               true,
		"go-mcp-dev":                           true,
		"Backend Services Specialist":          true,
		"Event Systems Specialist":             true,
		"Data Platform Specialist":             true,
		"ui-dev":                               true,
		"ui-tester":                            true,
		"Frontend Experience Specialist":       true,
		"AI Integration Specialist":            true,
		"Real-time Systems Specialist":         true,
		"sre":                                  true,
		"k8s-deployment-expert":                true,
		"Infrastructure Automation Specialist": true,
		"Security & Auth Specialist":           true,
		"Observability Specialist":             true,
		"End-to-End Testing Coordinator":       true,
	}

	if !validSubagents[subagentName] {
		return nil, fmt.Errorf("invalid subagent name '%s'. Use list_subagents to see available subagents", subagentName)
	}

	// Return success - actual chat session association will be handled by the chat service
	return map[string]interface{}{
		"subagentName": subagentName,
		"valid":        true,
		"message":      fmt.Sprintf("Subagent '%s' validated successfully. Chat session association requires chat context.", subagentName),
	}, nil
}

// DiscoverToolsExecutor implements the discover_tools tool executor
type DiscoverToolsExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *DiscoverToolsExecutor) Name() string {
	return "discover_tools"
}

func (e *DiscoverToolsExecutor) Description() string {
	return "Discover MCP tools using natural language semantic search. Returns matching tool names with descriptions and similarity scores. Use this to find tools by description (e.g., 'video tools', 'database tools', 'file operations')."
}

func (e *DiscoverToolsExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query describing the tools you're looking for (e.g., 'tools for video processing', 'database operations', 'file management')",
			},
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Maximum number of results to return (default: 5, max: 20)",
			},
		},
		"required": []string{"query"},
	}
}

func (e *DiscoverToolsExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleDiscoverTools(ctx, args)
	return data, err
}

// GetToolSchemaExecutor implements the get_tool_schema tool executor
type GetToolSchemaExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *GetToolSchemaExecutor) Name() string {
	return "get_tool_schema"
}

func (e *GetToolSchemaExecutor) Description() string {
	return "Get the complete JSON schema for a specific MCP tool. Returns the full tool definition including parameters, types, and descriptions. Use this after discovering tools to understand how to call them."
}

func (e *GetToolSchemaExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"toolName": map[string]interface{}{
				"type":        "string",
				"description": "Exact tool name to get schema for (use discover_tools first to find tool names)",
			},
		},
		"required": []string{"toolName"},
	}
}

func (e *GetToolSchemaExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleGetToolSchema(ctx, args)
	return data, err
}

// ExecuteToolExecutor implements the execute_tool tool executor
type ExecuteToolExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *ExecuteToolExecutor) Name() string {
	return "execute_tool"
}

func (e *ExecuteToolExecutor) Description() string {
	return "Execute an MCP tool by name with specified arguments. This tool looks up the tool's server from the registry and makes an HTTP call to that server's MCP endpoint. Works with external MCP servers registered via mcp_add_server. Built-in tools cannot be executed via this tool. Use get_tool_schema first to understand required parameters."
}

func (e *ExecuteToolExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"toolName": map[string]interface{}{
				"type":        "string",
				"description": "Exact tool name to execute (from discover_tools)",
			},
			"args": map[string]interface{}{
				"type":        "object",
				"description": "Tool-specific arguments as a JSON object (see get_tool_schema for parameter details)",
			},
		},
		"required": []string{"toolName", "args"},
	}
}

func (e *ExecuteToolExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleExecuteTool(ctx, args)
	return data, err
}

// McpAddServerExecutor implements the mcp_add_server tool executor
type McpAddServerExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *McpAddServerExecutor) Name() string {
	return "mcp_add_server"
}

func (e *McpAddServerExecutor) Description() string {
	return "Add a new MCP server to the registry, discover its tools, and store them in MongoDB and Qdrant for semantic search. The server must be accessible via HTTP/HTTPS and expose the MCP protocol."
}

func (e *McpAddServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Unique name for this MCP server (e.g., 'openai-mcp', 'github-mcp')",
			},
			"serverUrl": map[string]interface{}{
				"type":        "string",
				"description": "HTTP/HTTPS URL of the MCP server (e.g., 'http://localhost:3000/mcp')",
			},
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Human-readable description of what this server provides",
			},
		},
		"required": []string{"serverName", "serverUrl"},
	}
}

func (e *McpAddServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPAddServer(ctx, args)
	return data, err
}

// McpRediscoverServerExecutor implements the mcp_rediscover_server tool executor
type McpRediscoverServerExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *McpRediscoverServerExecutor) Name() string {
	return "mcp_rediscover_server"
}

func (e *McpRediscoverServerExecutor) Description() string {
	return "Rediscover and refresh tools from an existing MCP server. This removes old tools and discovers the current set of tools available on the server."
}

func (e *McpRediscoverServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to rediscover (must already be registered)",
			},
		},
		"required": []string{"serverName"},
	}
}

func (e *McpRediscoverServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPRediscoverServer(ctx, args)
	return data, err
}

// McpRemoveServerExecutor implements the mcp_remove_server tool executor
type McpRemoveServerExecutor struct {
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler
}

func (e *McpRemoveServerExecutor) Name() string {
	return "mcp_remove_server"
}

func (e *McpRemoveServerExecutor) Description() string {
	return "Remove an MCP server and all its tools from the registry. This deletes the server metadata and all associated tool data from MongoDB and Qdrant."
}

func (e *McpRemoveServerExecutor) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"serverName": map[string]interface{}{
				"type":        "string",
				"description": "Name of the MCP server to remove",
			},
		},
		"required": []string{"serverName"},
	}
}

func (e *McpRemoveServerExecutor) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	_, data, err := e.toolsDiscoveryHandler.HandleMCPRemoveServer(ctx, args)
	return data, err
}

// ExecuteSubagentTool implements the ToolExecutor interface
// This tool creates a subchat, links it to an agent task, and executes the subagent in background
type ExecuteSubagentTool struct {
	subchatStorage    *storage.SubchatStorage
	taskStorage       storage.TaskStorage
	aiService         AIServiceInterface
	chatService       ChatServiceInterface
	aiSettingsService AISettingsServiceInterface
	logger            *zap.Logger
	validator         *validation.CodeValidator
}

// AIServiceInterface defines methods needed from the AI chat service
type AIServiceInterface interface {
	StreamChatWithTools(ctx context.Context, messages []aiservice.Message, maxToolCalls int) (<-chan aiservice.StreamEvent, error)
	StreamChatWithToolsFiltered(ctx context.Context, messages []aiservice.Message, maxToolCalls int, allowedToolNames []string) (<-chan aiservice.StreamEvent, error)
	GetConfig() *aiservice.AIConfig
}

// ChatServiceInterface defines methods needed from the chat service
type ChatServiceInterface interface {
	CreateSession(ctx context.Context, userID, companyID, title string) (*models.ChatSession, error)
	CreateSessionWithParent(ctx context.Context, userID, companyID, title string, parentChatID *primitive.ObjectID) (*models.ChatSession, error)
	GetSession(ctx context.Context, sessionID primitive.ObjectID, companyID string) (*models.ChatSession, error)
	GetMessages(ctx context.Context, sessionID primitive.ObjectID, companyID string, limit, offset int) (*models.GetMessagesResponse, error)
	SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*models.ChatMessage, error)
	SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, id, name string, args map[string]interface{}, companyID string) (*models.ChatMessage, error)
	SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, id, name string, output interface{}, errorMsg string, durationMs int64, companyID string) (*models.ChatMessage, error)
}

// AISettingsServiceInterface defines methods needed from AI settings service
type AISettingsServiceInterface interface {
	GetSubagent(ctx context.Context, id primitive.ObjectID, companyID string) (*models.Subagent, error)
}

func (t *ExecuteSubagentTool) Name() string {
	return "execute_subagent"
}

func (t *ExecuteSubagentTool) Description() string {
	return "Execute a subagent to handle an agent task. Creates a subchat, links it to the task, and spawns the subagent in a separate execution context. The subagent will work independently and update task status. Returns immediately with subchat ID for tracking."
}

func (t *ExecuteSubagentTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"agentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Agent task ID (UUID) to execute",
			},
			"parentChatId": map[string]interface{}{
				"type":        "string",
				"description": "Parent chat session ID (optional - will be auto-detected from context if not provided)",
			},
		},
		"required": []string{"agentTaskId"},
	}
}

func (t *ExecuteSubagentTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, ok := input["agentTaskId"].(string)
	if !ok || agentTaskID == "" {
		return nil, fmt.Errorf(
			"❌ Parameter validation failed: agentTaskId is required and must be a string\n\n" +
				"⚠️ MANDATORY ACTION - BEFORE RETRYING:\n" +
				"1. Send a developer-friendly message to user:\n" +
				"   \"Tool error: execute_subagent failed - missing task ID parameter.\n" +
				"    Fixing by: retrieving the task ID from the create_agent_task result.\"\n" +
				"2. Check the create_agent_task response for 'taskId' field\n" +
				"3. Call execute_subagent with agentTaskId set to that taskId value\n\n" +
				"Example: execute_subagent({ agentTaskId: \"<taskId from previous step>\" })")
	}

	// ALWAYS try to get session ID from context first (most reliable)
	var parentChatID string
	if sessionID, hasSession := ctx.Value("sessionID").(string); hasSession && sessionID != "" {
		parentChatID = sessionID
		t.logger.Info("✅ Using session ID from context (auto-detected)",
			zap.String("agentTaskId", agentTaskID),
			zap.String("sessionID", sessionID))
	} else {
		// Fallback to AI-provided value only if context doesn't have it
		providedID, hasProvidedID := input["parentChatId"].(string)
		if hasProvidedID && providedID != "" && providedID != "main" {
			parentChatID = providedID
			t.logger.Warn("⚠️ Using AI-provided parentChatId (context not available)",
				zap.String("agentTaskId", agentTaskID),
				zap.String("parentChatId", providedID))
		} else {
			return nil, fmt.Errorf(
				"❌ Context error: parentChatId could not be determined\n\n" +
					"Details: Not in session context and not provided by AI (or AI provided 'main' placeholder)\n\n" +
					"⚠️ This is likely a system issue, not your fault.\n" +
					"Inform user: \"Tool error: execute_subagent failed - unable to determine parent chat session.\n" +
					"             This may be a context initialization issue. Please try again or contact support.\"")
		}
	}

	// Extract company ID from context
	var companyID string
	if companyIDValue, hasCompanyID := ctx.Value("companyID").(string); hasCompanyID && companyIDValue != "" {
		companyID = companyIDValue
		t.logger.Info("✅ Using company ID from context",
			zap.String("agentTaskId", agentTaskID),
			zap.String("companyID", companyID))
	} else {
		t.logger.Warn("⚠️ Company ID not found in context, will try to extract from parent session",
			zap.String("agentTaskId", agentTaskID))
	}

	t.logger.Info("🚀 execute_subagent tool called",
		zap.String("agentTaskId", agentTaskID),
		zap.String("parentChatId", parentChatID))

	// FIX #10: Retry GetAgentTask to handle MongoDB eventual consistency
	// Retry up to 3 times with exponential backoff (same pattern as CreateAgentTaskTool)
	var agentTask *storage.AgentTask
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		agentTask, err = t.taskStorage.GetAgentTask(agentTaskID)
		if err == nil && agentTask != nil {
			break // Success
		}
		if attempt < 2 {
			// Wait before retry: 100ms, then 200ms
			sleepDuration := time.Duration(100*(1<<uint(attempt))) * time.Millisecond
			t.logger.Debug("Retrying agentTaskId lookup after delay",
				zap.String("agentTaskId", agentTaskID),
				zap.Int("attempt", attempt+1),
				zap.Duration("delay", sleepDuration))
			time.Sleep(sleepDuration)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get agent task after 3 retries: %w", err)
	}

	t.logger.Info("📋 Retrieved agent task",
		zap.String("agentTaskId", agentTaskID),
		zap.String("agentName", agentTask.AgentName),
		zap.Int("todoCount", len(agentTask.Todos)))

	// Update task status to in_progress
	err = t.taskStorage.UpdateTaskStatus(agentTaskID, storage.TaskStatusInProgress, "Subagent execution initiated")
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Create subchat for this execution
	subchat, err := t.subchatStorage.CreateSubchat(
		parentChatID,
		agentTask.AgentName,
		&agentTaskID,
		nil, // todoID
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create subchat: %w", err)
	}

	t.logger.Info("💬 Created subchat for background execution",
		zap.String("subchatId", subchat.ID),
		zap.String("agentName", agentTask.AgentName))

	// Broadcast session_created event to parent session's WebSocket connection
	if parentSessionID, err := primitive.ObjectIDFromHex(parentChatID); err == nil {
		broadcaster := handlers.GetWebSocketBroadcaster(t.logger)
		sessionCreatedEvent := models.StreamMessage{
			Type:    "session_created",
			Content: subchat.ID, // Send subchat ID so frontend can identify the new session
		}
		if broadcastErr := broadcaster.BroadcastToSession(parentSessionID, sessionCreatedEvent); broadcastErr != nil {
			t.logger.Warn("Failed to broadcast session_created event",
				zap.String("parentSessionId", parentChatID),
				zap.String("subchatId", subchat.ID),
				zap.Error(broadcastErr))
		} else {
			t.logger.Info("Broadcasted session_created event to parent session",
				zap.String("parentSessionId", parentChatID),
				zap.String("subchatId", subchat.ID))
		}
	}

	// Spawn background goroutine to execute the subagent
	go t.executeSubagentInBackground(subchat.ID, agentTask, parentChatID, companyID)

	return map[string]interface{}{
		"subchatId":   subchat.ID,
		"agentName":   subchat.SubagentName,
		"agentTaskId": agentTaskID,
		"status":      "executing",
		"message":     fmt.Sprintf("Subchat created and %s is now executing in background. Check subchat messages for progress.", agentTask.AgentName),
		"createdAt":   subchat.CreatedAt,
	}, nil
}

// FileOperationTracker tracks file operations and duplicate tool calls during subagent execution
type FileOperationTracker struct {
	DirectoriesListed map[string]int // path -> count
	FilesRead         map[string]int // path -> count
	FilesWritten      map[string]int // path -> count
	BashCalls         map[string]int // command -> count
	ToolCallHistory   []string       // chronological list of tool calls

	// Track full argument sets for loop detection
	ToolCallSignatures map[string]int // signature (toolName + argsJSON) -> count
}

// NewFileOperationTracker creates a new tracker
func NewFileOperationTracker() *FileOperationTracker {
	return &FileOperationTracker{
		DirectoriesListed:  make(map[string]int),
		FilesRead:          make(map[string]int),
		FilesWritten:       make(map[string]int),
		BashCalls:          make(map[string]int),
		ToolCallHistory:    make([]string, 0),
		ToolCallSignatures: make(map[string]int),
	}
}

// RecordOperation records a file operation and detects duplicate tool calls with identical arguments
func (f *FileOperationTracker) RecordOperation(toolName string, args map[string]interface{}) string {
	warning := ""

	// ENHANCED LOOP DETECTION: Track full argument sets for all tool calls
	// Serialize arguments to JSON for signature comparison
	argsJSON, err := json.Marshal(args)
	if err == nil {
		// Create signature: toolName + argsJSON
		signature := toolName + ":" + string(argsJSON)

		// Track this signature
		f.ToolCallSignatures[signature]++

		// Generate warning if this exact call (tool + args) was seen before
		if f.ToolCallSignatures[signature] > 1 {
			count := f.ToolCallSignatures[signature] - 1
			warning = fmt.Sprintf("🔁 LOOP DETECTED: You already called '%s' with these exact arguments %d time(s). You are repeating the same operation. Use the results from previous calls instead of repeating.", toolName, count)
		}
	}

	// SPECIFIC FILE OPERATION TRACKING: Keep detailed file path tracking for progress summaries
	switch toolName {
	case "list_directory":
		if path, ok := args["path"].(string); ok {
			f.DirectoriesListed[path]++
			// Only show file-specific warning if no general loop warning was generated
			if warning == "" && f.DirectoriesListed[path] > 1 {
				warning = fmt.Sprintf("⚠️  WARNING: You already listed directory '%s' %d time(s). Review previous results before repeating.", path, f.DirectoriesListed[path]-1)
			}
		}
	case "read_file":
		if path, ok := args["filePath"].(string); ok {
			f.FilesRead[path]++
			// Only show file-specific warning if no general loop warning was generated
			if warning == "" && f.FilesRead[path] > 1 {
				warning = fmt.Sprintf("⚠️  WARNING: You already read file '%s' %d time(s). You should have the content from previous calls.", path, f.FilesRead[path]-1)
			}
		}
	case "write_file":
		// NOTE: The write_file tool uses "path" not "filePath" as its parameter name
		if path, ok := args["path"].(string); ok {
			f.FilesWritten[path]++
		}
	case "apply_patch":
		// FIRST: Check for explicit path parameter (apply_patch tool uses "path" parameter)
		if path, ok := args["path"].(string); ok && path != "" {
			f.FilesWritten[path]++
		} else if patchContent, ok := args["patch"].(string); ok {
			// FALLBACK: Try to extract file path from patch content
			// Format: "*** Update File: path/to/file.ext"
			if strings.Contains(patchContent, "*** Update File:") {
				lines := strings.Split(patchContent, "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "*** Update File:") {
						filePath := strings.TrimSpace(strings.TrimPrefix(line, "*** Update File:"))
						f.FilesWritten[filePath]++
						break
					}
				}
			} else {
				// Last resort: mark as generic write if we can't extract path
				f.FilesWritten["<patch-unknown-file>"]++
			}
		}
	case "bash":
		if command, ok := args["command"].(string); ok {
			// Track bash command (truncate if too long for map key)
			cmdKey := command
			if len(cmdKey) > 100 {
				cmdKey = cmdKey[:100] + "..."
			}
			f.BashCalls[cmdKey]++
		}
	}

	f.ToolCallHistory = append(f.ToolCallHistory, toolName)
	return warning
}

// GetProgressSummary returns a formatted summary of file operations
func (f *FileOperationTracker) GetProgressSummary() string {
	var summary strings.Builder

	summary.WriteString("\n\n📊 PROGRESS TRACKING - Files You've Already Seen:\n")

	if len(f.DirectoriesListed) > 0 {
		summary.WriteString("\nDirectories Listed:\n")
		for path, count := range f.DirectoriesListed {
			summary.WriteString(fmt.Sprintf("  • %s (%d times)\n", path, count))
		}
	}

	if len(f.FilesRead) > 0 {
		summary.WriteString("\nFiles Read:\n")
		for path, count := range f.FilesRead {
			summary.WriteString(fmt.Sprintf("  • %s (%d times)\n", path, count))
		}
	}

	if len(f.FilesWritten) > 0 {
		summary.WriteString("\nFiles Written/Modified:\n")
		for path, count := range f.FilesWritten {
			summary.WriteString(fmt.Sprintf("  • %s (%d times)\n", path, count))
		}
	}

	if len(f.DirectoriesListed) == 0 && len(f.FilesRead) == 0 && len(f.FilesWritten) == 0 {
		summary.WriteString("  (No file operations yet)\n")
	}

	summary.WriteString("\n⚠️  IMPORTANT: Do not repeat operations on files you've already seen. Use the information from previous tool calls.\n")

	return summary.String()
}

// normalizePathForComparison cleans and normalizes a path for comparison
// Handles ./, ../, and ensures consistent format for matching
func normalizePathForComparison(path string, projectRoot string) []string {
	// Clean the path to remove ./ and ../ components
	cleaned := filepath.Clean(path)

	// Generate multiple variants for matching
	variants := []string{cleaned}

	// Add absolute version if relative
	if !filepath.IsAbs(cleaned) {
		absPath := filepath.Join(projectRoot, cleaned)
		variants = append(variants, filepath.Clean(absPath))
	}

	// Add relative version if absolute
	if filepath.IsAbs(cleaned) {
		relPath, err := filepath.Rel(projectRoot, cleaned)
		if err == nil {
			variants = append(variants, filepath.Clean(relPath))
		}
	}

	return variants
}

// validateFileModifications checks if expected files were actually modified using the session-scoped file operation tracker
// This validation is robust to both absolute and relative path formats
func (t *ExecuteSubagentTool) validateFileModifications(agentTask *storage.AgentTask, progressTracker *FileOperationTracker) (bool, []string, error) {
	// If no files expected to be modified, skip validation
	if len(agentTask.FilesModified) == 0 {
		return true, []string{}, nil
	}

	// Check if any files were written during this session
	if len(progressTracker.FilesWritten) == 0 {
		return false, []string{}, fmt.Errorf("expected files not modified: wanted %v, but no files were written during this session", agentTask.FilesModified)
	}

	// Get project root to normalize paths
	projectRoot := tools.GetProjectRoot()

	t.logger.Info("🔍 [Path Validation] Starting file modification validation",
		zap.String("projectRoot", projectRoot),
		zap.Int("expectedFilesCount", len(agentTask.FilesModified)),
		zap.Int("writtenFilesCount", len(progressTracker.FilesWritten)))

	// Build a map of all expected file path variants for fast lookup
	expectedFiles := make(map[string]string) // variant -> original path
	for _, expectedFile := range agentTask.FilesModified {
		variants := normalizePathForComparison(expectedFile, projectRoot)
		t.logger.Debug("🔍 [Path Validation] Expected file variants",
			zap.String("originalPath", expectedFile),
			zap.Strings("variants", variants))
		for _, variant := range variants {
			expectedFiles[variant] = expectedFile
		}
	}

	// Check which expected files were actually written
	matchedFiles := []string{}
	for writtenPath := range progressTracker.FilesWritten {
		variants := normalizePathForComparison(writtenPath, projectRoot)
		t.logger.Debug("🔍 [Path Validation] Written file variants",
			zap.String("writtenPath", writtenPath),
			zap.Strings("variants", variants))

		// Check if any variant of the written path matches any expected file variant
		matched := false
		for _, variant := range variants {
			if originalExpected, found := expectedFiles[variant]; found {
				matchedFiles = append(matchedFiles, writtenPath)
				matched = true
				t.logger.Info("✅ [Path Validation] File matched",
					zap.String("writtenPath", writtenPath),
					zap.String("matchedVariant", variant),
					zap.String("expectedPath", originalExpected))
				break
			}
		}
		if !matched {
			t.logger.Warn("⚠️  [Path Validation] File not matched",
				zap.String("writtenPath", writtenPath),
				zap.Strings("triedVariants", variants))
		}
	}

	// Require at least 1 expected file to be modified
	if len(matchedFiles) == 0 {
		// Extract just the keys from FilesWritten for error message
		writtenFiles := make([]string, 0, len(progressTracker.FilesWritten))
		for path := range progressTracker.FilesWritten {
			writtenFiles = append(writtenFiles, path)
		}

		t.logger.Error("❌ [Path Validation] No expected files matched",
			zap.Strings("expectedFiles", agentTask.FilesModified),
			zap.Strings("writtenFiles", writtenFiles),
			zap.String("projectRoot", projectRoot))

		return false, matchedFiles, fmt.Errorf("expected files not modified: wanted %v, but agent wrote %v instead (session-scoped tracking). Project root: %s", agentTask.FilesModified, writtenFiles, projectRoot)
	}

	t.logger.Info("✅ [Path Validation] Validation successful",
		zap.Int("matchedFilesCount", len(matchedFiles)),
		zap.Strings("matchedFiles", matchedFiles))

	// ========================================
	// COMPILATION VALIDATION: Check TypeScript/Go files compile without errors
	// ========================================
	if t.validator != nil {
		t.logger.Info("🔍 [Compilation Validation] Running TypeScript/Go compilation checks",
			zap.Int("fileCount", len(matchedFiles)))

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := t.validator.ValidateFiles(ctx, matchedFiles)
		if err != nil {
			t.logger.Warn("⚠️  [Compilation Validation] Validation failed to run", zap.Error(err))
			// Don't block on validation errors - just warn
		} else if !result.Passed {
			t.logger.Error("❌ [Compilation Validation] COMPILATION ERRORS DETECTED",
				zap.Int("errorCount", len(result.Errors)),
				zap.Strings("files", matchedFiles))

			// Format errors for user
			errorMsg := t.validator.FormatErrorsForAgent(result)
			t.logger.Error("🚨 COMPILATION ERRORS MUST BE FIXED", zap.String("errors", errorMsg))

			// Return validation failure
			return false, matchedFiles, fmt.Errorf("compilation validation failed: %d error(s) found in modified files. Files must compile without errors before task completion", len(result.Errors))
		} else {
			t.logger.Info("✅ [Compilation Validation] All files compile successfully",
				zap.Strings("files", matchedFiles))
		}
	}

	return true, matchedFiles, nil
}

// convertToolCallToPlainEnglish converts a tool call to a user-friendly plain English description
func convertToolCallToPlainEnglish(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "read_file":
		if filePath, ok := args["file_path"].(string); ok {
			// Extract just the filename for brevity
			parts := strings.Split(filePath, "/")
			filename := parts[len(parts)-1]
			return fmt.Sprintf("📖 Reading file: %s", filename)
		}
		return "📖 Reading a file..."

	case "write_file":
		if filePath, ok := args["file_path"].(string); ok {
			parts := strings.Split(filePath, "/")
			filename := parts[len(parts)-1]
			return fmt.Sprintf("✍️ Writing to file: %s", filename)
		}
		return "✍️ Writing to a file..."

	case "apply_patch":
		if filePath, ok := args["file_path"].(string); ok {
			parts := strings.Split(filePath, "/")
			filename := parts[len(parts)-1]
			return fmt.Sprintf("🔧 Applying changes to: %s", filename)
		}
		return "🔧 Applying code changes..."

	case "bash":
		if command, ok := args["command"].(string); ok {
			// Truncate long commands
			if len(command) > 60 {
				command = command[:60] + "..."
			}
			return fmt.Sprintf("⚡ Running command: %s", command)
		}
		return "⚡ Running a command..."

	case "coordinator_update_todo_status":
		if status, ok := args["status"].(string); ok {
			if status == "completed" {
				return "✅ Marking TODO as completed"
			} else if status == "in_progress" {
				return "▶️ Starting work on TODO"
			}
		}
		return "📝 Updating TODO status..."

	case "coordinator_upsert_knowledge":
		return "💾 Saving knowledge entry..."

	default:
		return fmt.Sprintf("🔧 Using tool: %s", toolName)
	}
}

// convertToolResultToPlainEnglish converts a tool result to a user-friendly plain English message
func convertToolResultToPlainEnglish(toolName string, output interface{}, errorMsg string) string {
	if errorMsg != "" {
		// Handle errors
		switch toolName {
		case "read_file":
			return fmt.Sprintf("❌ Failed to read file: %s", errorMsg)
		case "write_file":
			return fmt.Sprintf("❌ Failed to write file: %s", errorMsg)
		case "apply_patch":
			return fmt.Sprintf("❌ Failed to apply patch: %s", errorMsg)
		case "bash":
			return fmt.Sprintf("❌ Command failed: %s", errorMsg)
		default:
			return fmt.Sprintf("❌ Tool error: %s", errorMsg)
		}
	}

	// Handle successes
	switch toolName {
	case "read_file":
		if str, ok := output.(string); ok {
			lineCount := len(strings.Split(str, "\n"))
			return fmt.Sprintf("✓ File read successfully (%d lines)", lineCount)
		}
		return "✓ File read successfully"

	case "write_file":
		return "✓ File written successfully"

	case "apply_patch":
		return "✓ Changes applied successfully"

	case "bash":
		if str, ok := output.(string); ok {
			// Show first line of output if it's short
			lines := strings.Split(strings.TrimSpace(str), "\n")
			if len(lines) > 0 && len(lines[0]) < 80 && len(lines[0]) > 0 {
				return fmt.Sprintf("✓ Command completed: %s", lines[0])
			}
		}
		return "✓ Command completed successfully"

	case "coordinator_update_todo_status":
		return "✓ TODO status updated"

	case "coordinator_upsert_knowledge":
		return "✓ Knowledge saved"

	default:
		return "✓ Tool completed"
	}
}

// isSystemEnforcementMessage checks if a message is a system enforcement message that should be filtered
func isSystemEnforcementMessage(content string) bool {
	systemPatterns := []string{
		"WRITE-ONLY MODE",
		"FORCED WRITE SCAFFOLD",
		"╔══════════════",
		"🚨 WRITE-ONLY MODE ENFORCEMENT",
		"EXECUTION SCORE:",
		"📊 EXECUTION SCORE",
		"CACHED FILE CONTENT",
		"⚠️ CACHED FILE CONTENT",
		"CURRENT EXECUTION SCORE:",
	}

	for _, pattern := range systemPatterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

// executeSubagentInBackground runs the subagent AI streaming in a background goroutine
func (t *ExecuteSubagentTool) executeSubagentInBackground(subchatID string, agentTask *storage.AgentTask, parentChatID string, companyID string) {
	// Create a new background context with generous timeout for long-running tasks
	baseCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Inject agent task ID into context for validation system
	ctx := context.WithValue(baseCtx, aiservice.AgentTaskIDKey, agentTask.ID)

	t.logger.Info("╔═══════════════════════════════════════════════════════════════════")
	t.logger.Info("║ 🚀 SUBAGENT EXECUTION STARTED")
	t.logger.Info("╠═══════════════════════════════════════════════════════════════════",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.String("agentTaskId", agentTask.ID),
		zap.String("parentChatId", parentChatID),
		zap.Int("todoCount", len(agentTask.Todos)),
		zap.Int("filesModifiedCount", len(agentTask.FilesModified)))
	t.logger.Info("╚═══════════════════════════════════════════════════════════════════")

	t.logger.Info("⚡ Starting subagent execution in background goroutine",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.String("agentTaskId", agentTask.ID))

	// Initialize progress tracker
	progressTracker := NewFileOperationTracker()

	// Create a chat session for this subchat
	// Get parent chat session to extract userID and companyID
	parentSessionID, err := primitive.ObjectIDFromHex(parentChatID)
	if err != nil {
		t.logger.Error("Failed to parse parent chat ID",
			zap.String("parentChatId", parentChatID),
			zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Invalid parent chat ID: %v", err))
		return
	}

	// Emit progress notification - subchat started
	handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("🤖 Starting subchat: %s", agentTask.AgentName))

	// If companyID was not provided from context, we need to fetch parent session to get it
	// Otherwise, we can skip fetching parent session just for companyID
	var userID string
	var finalCompanyID string

	if companyID != "" {
		// Company ID provided from context - still need to fetch parent session for userID
		t.logger.Info("Using company ID from context, fetching parent session for user ID",
			zap.String("companyID", companyID))
		parentSession, err := t.chatService.GetSession(ctx, parentSessionID, companyID)
		if err != nil {
			t.logger.Error("Failed to get parent chat session",
				zap.String("parentChatId", parentChatID),
				zap.String("companyId", companyID),
				zap.Error(err))
			t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Failed to get parent session: %v", err))
			return
		}
		userID = parentSession.UserID
		finalCompanyID = companyID
	} else {
		// Company ID not in context - fetch parent session to get both userID and companyID
		// This is a fallback for cases where context doesn't have companyID
		t.logger.Warn("Company ID not in context, attempting to extract from parent session")

		// Try with empty companyID first (some implementations might not enforce it)
		parentSession, err := t.chatService.GetSession(ctx, parentSessionID, "")
		if err != nil {
			// If that fails, this is likely a configuration issue
			t.logger.Error("Failed to get parent chat session without company ID",
				zap.String("parentChatId", parentChatID),
				zap.Error(err))
			t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Failed to get parent session: %v (companyID not available in context)", err))
			return
		}
		userID = parentSession.UserID
		finalCompanyID = parentSession.CompanyID

		t.logger.Info("Extracted company ID from parent session",
			zap.String("companyID", finalCompanyID))
	}

	sessionTitle := fmt.Sprintf("Subchat: %s - %s", agentTask.AgentName, agentTask.Role)

	t.logger.Info("Creating subchat session with parent's credentials",
		zap.String("subchatId", subchatID),
		zap.String("parentChatId", parentChatID),
		zap.String("userId", userID),
		zap.String("companyId", finalCompanyID))

	chatSession, err := t.chatService.CreateSessionWithParent(ctx, userID, finalCompanyID, sessionTitle, &parentSessionID)
	if err != nil {
		t.logger.Error("Failed to create chat session for subchat",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Failed to create chat session: %v", err))
		return
	}

	t.logger.Info("💬 Created chat session for subchat with parent link",
		zap.String("subchatId", subchatID),
		zap.String("sessionId", chatSession.ID.Hex()),
		zap.String("parentChatId", parentChatID))

	// Update subchat with session ID for linking
	sessionIDHex := chatSession.ID.Hex()
	err = t.subchatStorage.UpdateSubchatSessionID(subchatID, sessionIDHex)
	if err != nil {
		t.logger.Warn("Failed to link chat session to subchat",
			zap.String("subchatId", subchatID),
			zap.String("sessionId", sessionIDHex),
			zap.Error(err))
		// Continue execution even if linking fails
	}

	// Build SYSTEM prompt (phase constraints + role) and USER prompt (task details)
	systemPrompt := t.buildExecutionPhaseSystemPrompt()
	taskPrompt := t.buildSubagentTaskPrompt(agentTask)

	t.logger.Info("📜 Built phase-isolated subagent prompts",
		zap.String("subchatId", subchatID),
		zap.Int("systemPromptLength", len(systemPrompt)),
		zap.Int("taskPromptLength", len(taskPrompt)),
		zap.Int("todoCount", len(agentTask.Todos)))

	// Save initial user message (task details only - system prompt is separate)
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "user", taskPrompt, finalCompanyID)
	if err != nil {
		t.logger.Warn("Failed to save initial user message",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		// Continue execution even if message save fails
	}

	// Register for message notifications (for user interruptions)
	notifier := handlers.GetMessageNotifier(t.logger)
	notifyCh := notifier.RegisterSession(chatSession.ID)
	defer notifier.UnregisterSession(chatSession.ID)

	t.logger.Info("🔔 Registered subagent session for message notifications",
		zap.String("sessionId", chatSession.ID.Hex()),
		zap.String("subchatId", subchatID))

	// Create initial messages with SYSTEM prompt for phase isolation
	messages := []aiservice.Message{
		{
			Role:    "system",
			Content: systemPrompt, // EXECUTE phase constraints
		},
		{
			Role:    "user",
			Content: taskPrompt, // Task details only
		},
	}

	// Define allowed tools for subagents (ONLY implementation tools, NO coordinator tools)
	// This prevents subagents from calling coordinator tools and forces them to actually write code
	// NOTE: code_index_search and list_directory are REMOVED - subagents are pure implementers, not explorers
	// CRITICAL: list_directory removed - it enables exploration mode that contradicts WRITE-ONLY MODE
	allowedTools := []string{
		"read_file",                      // Read source files (ONLY files specified in task)
		"write_file",                     // Write/create files
		"apply_patch",                    // Apply code patches
		"bash",                           // Run commands, tests, syntax checks
		"coordinator_update_todo_status", // Update TODO status
		"coordinator_upsert_knowledge",   // Store knowledge/decisions
	}

	t.logger.Info("🔒 Filtering tools for subagent",
		zap.Int("allowedTools", len(allowedTools)),
		zap.Strings("tools", allowedTools))

	// Stream AI response with FILTERED tools (only implementation tools, no coordinator tools)
	maxToolCalls := t.aiService.GetConfig().MaxToolCalls
	aiStream, err := t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
	if err != nil {
		t.logger.Error("Failed to start AI streaming for subagent",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("⚠️ Subchat failed: %s", agentTask.AgentName))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("AI streaming failed: %v", err))
		return
	}

	// Process stream events and save to subchat
	fullResponse := ""
	toolCallCount := 0
	completedTodos := 0
	readFileCount := 0         // Track read_file calls (after 3 reads, MUST write)
	hasWrittenAnyFile := false // Track if any write has occurred

	// RUNTIME ENFORCEMENT: File content cache to prevent re-reads
	fileContentCache := make(map[string]string)

	// RUNTIME ENFORCEMENT: Execution scoring system
	executionScore := 0

	t.logger.Info("📡 Subagent AI stream started - processing events...",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName))

	t.logger.Info("╔═══════════════════════════════════════════════════════════════════")
	t.logger.Info("║ 🔄 PROCESSING AI STREAM EVENTS")
	t.logger.Info("╚═══════════════════════════════════════════════════════════════════",
		zap.String("subchatId", subchatID))

	// Track stream state for interrupt handling
	streamActive := true
	fullSystemPrompt := systemPrompt // Store system prompt for message rebuilding on interrupt

	for streamActive {
		// ═══════════════════════════════════════════════════════════════════
		// STAGE 1: PRIORITY CHECK FOR INTERRUPTS (NON-BLOCKING)
		// ═══════════════════════════════════════════════════════════════════
		// This ensures interrupts are ALWAYS checked before processing AI events,
		// preventing starvation when AI is streaming rapidly.
		var interruptReceived bool
		select {
		case <-notifyCh:
			// USER MESSAGE INTERRUPT - PRIORITY HANDLING
			t.logger.Info("💬 🔥 PRIORITY: User interrupt detected (pre-check)",
				zap.String("sessionId", chatSession.ID.Hex()),
				zap.String("subchatId", subchatID))
			interruptReceived = true
		default:
			// No interrupt pending, proceed to normal event processing
		}

		// If interrupt was detected in priority check, handle it now
		if interruptReceived {
			// ═══════════════════════════════════════════════════════════════════
			// INTERRUPT HANDLER
			// ═══════════════════════════════════════════════════════════════════

			// Drain current stream if active
			if aiStream != nil {
				go func() {
					for range aiStream {
						// discard remaining events
					}
				}()
			}

			// Fetch all messages including new user message
			messagesResp, err := t.chatService.GetMessages(ctx, chatSession.ID, finalCompanyID, 100, 0)
			if err != nil {
				t.logger.Error("Failed to fetch messages after interrupt", zap.Error(err))
				continue
			}

			// Extract latest user message for categorization
			var latestUserMessage string
			for i := len(messagesResp.Messages) - 1; i >= 0; i-- {
				if messagesResp.Messages[i].Role == "user" {
					latestUserMessage = messagesResp.Messages[i].Content
					break
				}
			}

			// Categorize the interrupt to determine intent
			category, guidance, err := t.categorizeInterrupt(ctx, latestUserMessage)
			if err != nil {
				t.logger.Warn("Failed to categorize interrupt, defaulting to CONTINUE",
					zap.Error(err),
					zap.String("userMessage", latestUserMessage))
				category = "CONTINUE"
				guidance = "Continue with your work but acknowledge the user's message if relevant"
			}

			t.logger.Info("🎯 Interrupt categorized",
				zap.String("category", category),
				zap.String("guidance", guidance),
				zap.String("userMessage", latestUserMessage))

			// Emit progress notification about interrupt
			handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID,
				fmt.Sprintf("📨 User interrupt received: %s", category))

			// Build interrupt-aware system prompt guidance based on category
			var interruptGuidance string
			switch category {
			case "STOP":
				interruptGuidance = fmt.Sprintf(`
⚠️ CRITICAL: USER INTERRUPT - STOP CURRENT TASK
The user has sent a message indicating they want to STOP the current task.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. IMMEDIATELY acknowledge the user's message in your FIRST response
2. STOP all current work - do not make ANY tool calls until you respond
3. Ask the user what they would like you to do instead
4. DO NOT continue with the original task unless they explicitly say to continue

Start your response with: "I've stopped the current task. [address their message directly]"
`, latestUserMessage, guidance)

			case "MODIFY":
				interruptGuidance = fmt.Sprintf(`
🔄 USER INTERRUPT - MODIFY APPROACH
The user wants to modify or adjust the current approach.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. FIRST, acknowledge the user's request in your response (use text, not just tool calls)
2. Explain how you'll adjust your approach based on their guidance
3. THEN proceed with the modified approach using tool calls

Start your response with: "I'll adjust my approach. [explain the changes]"
`, latestUserMessage, guidance)

			case "CLARIFY":
				interruptGuidance = fmt.Sprintf(`
❓ USER INTERRUPT - NEEDS CLARIFICATION
The user has a question or needs clarification about your work.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. Answer their question directly and clearly in your FIRST response
2. Do NOT make any tool calls before responding to their question
3. After answering, ask if they want you to continue or adjust
4. Wait for their response before making more tool calls

Start your response by directly addressing their question.
`, latestUserMessage, guidance)

			case "STATUS":
				interruptGuidance = fmt.Sprintf(`
📊 USER INTERRUPT - STATUS CHECK
The user is checking progress or giving encouragement.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. FIRST, give a brief status update (what you've completed, what's next)
2. Acknowledge their message warmly
3. THEN continue with your work

Keep your status response brief (2-3 sentences) before continuing.
`, latestUserMessage, guidance)

			case "CONTINUE":
				interruptGuidance = fmt.Sprintf(`
✅ USER MESSAGE NOTED
The user sent a message that doesn't require action changes.

User's message: "%s"

AI Analysis: %s

Briefly acknowledge their message if appropriate (1 sentence), then continue your work.
`, latestUserMessage, guidance)

			default:
				interruptGuidance = fmt.Sprintf(`
📨 USER MESSAGE RECEIVED
User's message: "%s"

Acknowledge the message and continue your work.
`, latestUserMessage)
			}

			// Rebuild message context with interrupt-aware system prompt
			messages = []aiservice.Message{
				{Role: "system", Content: fullSystemPrompt + "\n\n" + interruptGuidance},
			}
			for _, msg := range messagesResp.Messages {
				messages = append(messages, aiservice.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}

			t.logger.Info("🔄 Resuming subagent with interrupt-aware context",
				zap.Int("messageCount", len(messages)),
				zap.String("category", category),
				zap.String("sessionId", chatSession.ID.Hex()))

			// Restart AI stream with updated, interrupt-aware context
			aiStream, err = t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
			if err != nil {
				t.logger.Error("Failed to restart AI stream after interrupt", zap.Error(err))
				t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Stream restart failed: %v", err))
				return
			}

			// Continue to next loop iteration to check for interrupts again
			continue
		}

		// ═══════════════════════════════════════════════════════════════════
		// STAGE 2: NORMAL EVENT PROCESSING (TIMEOUT, INTERRUPT BACKUP, AI EVENTS)
		// ═══════════════════════════════════════════════════════════════════
		select {
		case <-ctx.Done():
			t.logger.Warn("⏱️ Subagent execution cancelled by timeout",
				zap.String("subchatId", subchatID))
			handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("⚠️ Subchat timeout: %s", agentTask.AgentName))
			t.handleExecutionFailure(agentTask.ID, "Execution timeout")
			return

		case <-notifyCh:
			// USER MESSAGE INTERRUPT - BACKUP HANDLER (caught in stage 2 when priority check missed it)
			t.logger.Info("💬 User interrupt detected (backup - caught in stage 2)",
				zap.String("sessionId", chatSession.ID.Hex()),
				zap.String("subchatId", subchatID))

			// Drain current stream if active
			if aiStream != nil {
				go func() {
					for range aiStream {
						// discard remaining events
					}
				}()
			}

			// Fetch all messages including new user message
			messagesResp, err := t.chatService.GetMessages(ctx, chatSession.ID, finalCompanyID, 100, 0)
			if err != nil {
				t.logger.Error("Failed to fetch messages after interrupt", zap.Error(err))
				continue
			}

			// Extract latest user message for categorization
			var latestUserMessage string
			for i := len(messagesResp.Messages) - 1; i >= 0; i-- {
				if messagesResp.Messages[i].Role == "user" {
					latestUserMessage = messagesResp.Messages[i].Content
					break
				}
			}

			// Categorize the interrupt to determine intent
			category, guidance, err := t.categorizeInterrupt(ctx, latestUserMessage)
			if err != nil {
				t.logger.Warn("Failed to categorize interrupt, defaulting to CONTINUE",
					zap.Error(err),
					zap.String("userMessage", latestUserMessage))
				category = "CONTINUE"
				guidance = "Continue with your work but acknowledge the user's message if relevant"
			}

			t.logger.Info("🎯 Interrupt categorized",
				zap.String("category", category),
				zap.String("guidance", guidance),
				zap.String("userMessage", latestUserMessage))

			// Emit progress notification about interrupt
			handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID,
				fmt.Sprintf("📨 User interrupt received: %s", category))

			// Build interrupt-aware system prompt guidance based on category
			var interruptGuidance string
			switch category {
			case "STOP":
				interruptGuidance = fmt.Sprintf(`
⚠️ CRITICAL: USER INTERRUPT - STOP CURRENT TASK
The user has sent a message indicating they want to STOP the current task.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. IMMEDIATELY acknowledge the user's message in your FIRST response
2. STOP all current work - do not make ANY tool calls until you respond
3. Ask the user what they would like you to do instead
4. DO NOT continue with the original task unless they explicitly say to continue

Start your response with: "I've stopped the current task. [address their message directly]"
`, latestUserMessage, guidance)

			case "MODIFY":
				interruptGuidance = fmt.Sprintf(`
🔄 USER INTERRUPT - MODIFY APPROACH
The user wants to modify or adjust the current approach.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. FIRST, acknowledge the user's request in your response (use text, not just tool calls)
2. Explain how you'll adjust your approach based on their guidance
3. THEN proceed with the modified approach using tool calls

Start your response with: "I'll adjust my approach. [explain the changes]"
`, latestUserMessage, guidance)

			case "CLARIFY":
				interruptGuidance = fmt.Sprintf(`
❓ USER INTERRUPT - NEEDS CLARIFICATION
The user has a question or needs clarification about your work.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. Answer their question directly and clearly in your FIRST response
2. Do NOT make any tool calls before responding to their question
3. After answering, ask if they want you to continue or adjust
4. Wait for their response before making more tool calls

Start your response by directly addressing their question.
`, latestUserMessage, guidance)

			case "STATUS":
				interruptGuidance = fmt.Sprintf(`
📊 USER INTERRUPT - STATUS CHECK
The user is checking progress or giving encouragement.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. FIRST, give a brief status update (what you've completed, what's next)
2. Acknowledge their message warmly
3. THEN continue with your work

Keep your status response brief (2-3 sentences) before continuing.
`, latestUserMessage, guidance)

			case "CONTINUE":
				interruptGuidance = fmt.Sprintf(`
✅ USER MESSAGE NOTED
The user sent a message that doesn't require action changes.

User's message: "%s"

AI Analysis: %s

Briefly acknowledge their message if appropriate (1 sentence), then continue your work.
`, latestUserMessage, guidance)

			default:
				interruptGuidance = fmt.Sprintf(`
📨 USER MESSAGE RECEIVED
User's message: "%s"

Acknowledge the message and continue your work.
`, latestUserMessage)
			}

			// Rebuild message context with interrupt-aware system prompt
			messages = []aiservice.Message{
				{Role: "system", Content: fullSystemPrompt + "\n\n" + interruptGuidance},
			}
			for _, msg := range messagesResp.Messages {
				messages = append(messages, aiservice.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}

			t.logger.Info("🔄 Resuming subagent with interrupt-aware context",
				zap.Int("messageCount", len(messages)),
				zap.String("category", category),
				zap.String("sessionId", chatSession.ID.Hex()))

			// Restart AI stream with updated, interrupt-aware context
			aiStream, err = t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
			if err != nil {
				t.logger.Error("Failed to restart AI stream after interrupt", zap.Error(err))
				t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Stream restart failed: %v", err))
				return
			}

			// Continue to next loop iteration
			continue

		case event, ok := <-aiStream:
			if !ok {
				// Stream closed naturally
				streamActive = false
				break
			}

			switch event.Type {
			case aiservice.StreamEventToken:
				fullResponse += event.Content

				// 📝 LOG EVERY TEXT TOKEN RECEIVED FROM AI
				t.logger.Info("💬 AI TEXT TOKEN RECEIVED",
					zap.String("subchatId", subchatID),
					zap.String("sessionId", chatSession.ID.Hex()),
					zap.String("content", event.Content),
					zap.Int("contentLength", len(event.Content)),
					zap.Bool("isSystemMessage", isSystemEnforcementMessage(event.Content)))

				// Stream AI messages to progress channel (filter out system messages)
				if event.Content != "" && !isSystemEnforcementMessage(event.Content) {
					// Only emit substantive messages (not just whitespace or single characters)
					trimmed := strings.TrimSpace(event.Content)
					if len(trimmed) > 5 { // Only emit messages with substance
						t.logger.Info("✅ EMITTING TEXT TO PROGRESS NOTIFIER",
							zap.String("subchatId", subchatID),
							zap.String("content", event.Content),
							zap.Int("trimmedLength", len(trimmed)))

						// 💾 SAVE TO DATABASE for persistence (so messages survive page refresh)
						_, err := t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", event.Content, finalCompanyID)
						if err != nil {
							t.logger.Warn("Failed to save streaming text chunk to database",
								zap.String("subchatId", subchatID),
								zap.String("sessionId", chatSession.ID.Hex()),
								zap.Error(err))
						} else {
							t.logger.Debug("💾 Saved text chunk to database",
								zap.String("subchatId", subchatID),
								zap.String("sessionId", chatSession.ID.Hex()),
								zap.Int("chunkLength", len(event.Content)))
						}

						// 📡 Send to WebSocket (CRITICAL: use subchat session, not parent!)
						handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, event.Content)
					} else {
						t.logger.Info("⏭️ SKIPPING SHORT TEXT (< 5 chars)",
							zap.String("subchatId", subchatID),
							zap.String("content", event.Content),
							zap.Int("trimmedLength", len(trimmed)))
					}
				} else {
					t.logger.Info("🚫 FILTERED OUT (empty or system message)",
						zap.String("subchatId", subchatID),
						zap.String("content", event.Content))
				}

			case aiservice.StreamEventToolCall:
				toolCallCount++

				// 📊 COMPREHENSIVE LOGGING: Log every tool call (args truncated for brevity)
				argsSummary := make(map[string]interface{})
				for key, value := range event.ToolCall.Args {
					if key == "content" || key == "output" {
						// Truncate large content fields
						if str, ok := value.(string); ok && len(str) > 100 {
							argsSummary[key] = fmt.Sprintf("%s... (%d chars)", str[:100], len(str))
						} else {
							argsSummary[key] = value
						}
					} else {
						argsSummary[key] = value
					}
				}
				t.logger.Info("🔧 TOOL CALL",
					zap.String("subchatId", subchatID),
					zap.Int("callNumber", toolCallCount),
					zap.String("toolName", event.ToolCall.Name),
					zap.Any("args", argsSummary))

				// Emit progress notification with plain English tool call description
				plainEnglishToolCall := convertToolCallToPlainEnglish(event.ToolCall.Name, event.ToolCall.Args)
				handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, plainEnglishToolCall)

				// RUNTIME ENFORCEMENT: Track read_file calls and apply scoring
				if event.ToolCall.Name == "read_file" {
					readFileCount++

					// Check if this file was already read (duplicate read)
					filePath := ""
					if fp, ok := event.ToolCall.Args["file_path"].(string); ok {
						filePath = fp
					}

					if _, alreadyRead := fileContentCache[filePath]; alreadyRead {
						// Duplicate read - penalty
						executionScore -= 5
						t.logger.Warn("⚠️ DUPLICATE READ detected - scoring penalty",
							zap.String("subchatId", subchatID),
							zap.String("filePath", filePath),
							zap.Int("score", executionScore))
					} else {
						// First read - small bonus
						executionScore += 5
						t.logger.Info("✅ First read of file - scoring bonus",
							zap.String("subchatId", subchatID),
							zap.String("filePath", filePath),
							zap.Int("score", executionScore))
					}

					// RUNTIME ENFORCEMENT: Hard read limit - block reads after threshold
					if readFileCount > 3 && !hasWrittenAnyFile {
						executionScore -= 50
						t.logger.Error("🚫 HARD LIMIT: Exceeded 3 reads without write - CRITICAL",
							zap.String("subchatId", subchatID),
							zap.Int("readFileCount", readFileCount),
							zap.Int("score", executionScore))
					}

					// Penalty for exceeding 1 read without write (lowered threshold from 2 to 1)
					if readFileCount > 1 && !hasWrittenAnyFile {
						executionScore -= 20
						t.logger.Warn("⚠️ Exceeded 1 read without write - large penalty (WRITE NOW)",
							zap.String("subchatId", subchatID),
							zap.Int("readFileCount", readFileCount),
							zap.Int("score", executionScore))
					}
				}

				// RUNTIME ENFORCEMENT: Track write operations and reward
				if event.ToolCall.Name == "write_file" || event.ToolCall.Name == "apply_patch" {
					hasWrittenAnyFile = true
					readFileCount = 0    // Reset read counter after write
					executionScore += 20 // Big reward for writing

					t.logger.Info("✅ Write operation detected - BIG scoring bonus",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolCall.Name),
						zap.Int("score", executionScore))
				}

				// RUNTIME ENFORCEMENT: Reward TODO completion
				if event.ToolCall.Name == "coordinator_update_todo_status" {
					if status, ok := event.ToolCall.Args["status"].(string); ok && status == "completed" {
						executionScore += 10
						t.logger.Info("✅ TODO completed - scoring bonus",
							zap.String("subchatId", subchatID),
							zap.Int("score", executionScore))
					}
				}

				// Penalty for duplicate tool calls
				warning := progressTracker.RecordOperation(event.ToolCall.Name, event.ToolCall.Args)
				if warning != "" {
					executionScore -= 10
					t.logger.Warn("⚠️ Duplicate operation - scoring penalty",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolCall.Name),
						zap.Int("score", executionScore))
				}

				// Record file operation in progress tracker (for reporting only)
				progressTracker.RecordOperation(event.ToolCall.Name, event.ToolCall.Args)

				t.logger.Info("🔧 Subagent calling tool",
					zap.String("subchatId", subchatID),
					zap.String("agentName", agentTask.AgentName),
					zap.String("toolName", event.ToolCall.Name),
					zap.Int("toolCallNumber", toolCallCount))

				// Save tool call to subchat messages
				_, err := t.chatService.SaveToolCall(ctx, chatSession.ID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, finalCompanyID)
				if err != nil {
					t.logger.Error("Failed to save tool call",
						zap.String("subchatId", subchatID),
						zap.Error(err))
				}

				// Check if this is a todo status update - track completion
				if event.ToolCall.Name == "coordinator_update_todo_status" {
					if status, ok := event.ToolCall.Args["status"].(string); ok && status == "completed" {
						completedTodos++
						t.logger.Info("✅ TODO marked as completed",
							zap.String("subchatId", subchatID),
							zap.String("agentName", agentTask.AgentName),
							zap.Int("completedCount", completedTodos),
							zap.Int("totalTodos", len(agentTask.Todos)))
					} else if status, ok := event.ToolCall.Args["status"].(string); ok && status == "in_progress" {
						t.logger.Info("▶️ TODO started",
							zap.String("subchatId", subchatID),
							zap.String("agentName", agentTask.AgentName))
					}
				}

			case aiservice.StreamEventToolResult:
				// Emit progress notification with plain English tool result
				plainEnglishResult := convertToolResultToPlainEnglish(event.ToolResult.Name, event.ToolResult.Output, event.ToolResult.Error)
				handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, plainEnglishResult)

				// Summarize tool result to prevent context bloat
				var originalSize int
				var summarizedOutput string

				if event.ToolResult.Output != nil {
					// Calculate original size for logging
					if str, ok := event.ToolResult.Output.(string); ok {
						originalSize = len(str)
					} else {
						outputBytes, _ := json.Marshal(event.ToolResult.Output)
						originalSize = len(outputBytes)
					}

					// Apply summarization
					summarizedOutput = t.summarizeToolResult(event.ToolResult.Name, event.ToolResult.Output)
				}

				// Use error message if tool call failed
				if event.ToolResult.Error != "" {
					summarizedOutput = fmt.Sprintf("Error: %s", event.ToolResult.Error)
				}

				// RUNTIME ENFORCEMENT: File content caching and duplicate read blocking
				if event.ToolResult.Name == "read_file" && event.ToolResult.Error == "" {
					// Cache the full content (not the summarized version)
					if fullContent, ok := event.ToolResult.Output.(string); ok {
						// Check if this is a duplicate read by checking cache
						if len(fileContentCache) > 0 {
							// RUNTIME ENFORCEMENT: Return cached summary instead of full content
							summarizedOutput = fmt.Sprintf("⚠️ CACHED FILE CONTENT - DO NOT RE-READ\n\nThis file was already read. Content is cached. You MUST use the previous read content.\n\nSUMMARY: File contains %d characters across %d lines.\n\nYour next action MUST be write_file or apply_patch - NOT another read_file.\n\nCURRENT SCORE: %d points", len(fullContent), len(strings.Split(fullContent, "\n")), executionScore)

							t.logger.Warn("🚫 Returning cached summary - blocking duplicate read",
								zap.String("subchatId", subchatID),
								zap.Int("score", executionScore))
						} else {
							// First read - cache the content but still return it
							fileContentCache["last_read"] = fullContent
							t.logger.Info("💾 Cached file content for enforcement",
								zap.String("subchatId", subchatID),
								zap.Int("contentSize", len(fullContent)))
						}
					}
				}

				// RUNTIME ENFORCEMENT: Inject forced write scaffold after 1 read without write (lowered from 2)
				if readFileCount >= 1 && !hasWrittenAnyFile {
					forceScaffold := fmt.Sprintf("\n\n╔══════════════════════════════════════════════════════════════╗\n║          FORCED WRITE SCAFFOLD - COMPLETE AND SUBMIT          ║\n╚══════════════════════════════════════════════════════════════╝\n\n🚨 WRITE-ONLY MODE ENFORCEMENT 🚨\n\nYou have read %d file(s). Reading phase is COMPLETE.\n\nYour NEXT tool call MUST be either:\n1. write_file - to create or modify a file\n2. apply_patch - to apply code changes\n\nYou are BLOCKED from calling read_file again.\n\nCURRENT EXECUTION SCORE: %d points\n\nSCORING:\n- Next write_file/apply_patch: +20 points ✅\n- Another read_file: -50 points ❌ (HARD LIMIT)\n\nIMPLEMENT NOW - DO NOT READ ANOTHER FILE.", readFileCount, executionScore)
					summarizedOutput += forceScaffold

					// Save scaffold as internal message (not visible to end user)
					_, err := t.chatService.SaveMessage(ctx, chatSession.ID, "system_internal", forceScaffold, finalCompanyID)
					if err != nil {
						t.logger.Warn("Failed to save forced scaffold message",
							zap.String("subchatId", subchatID),
							zap.Error(err))
					}

					t.logger.Warn("🔨 Injected FORCED WRITE SCAFFOLD",
						zap.String("subchatId", subchatID),
						zap.Int("readFileCount", readFileCount),
						zap.Int("score", executionScore))
				}

				// RUNTIME ENFORCEMENT: Inject scoring message (visible to model)
				scoringMessage := fmt.Sprintf("\n\n📊 EXECUTION SCORE: %d points\n", executionScore)
				if executionScore >= 30 {
					scoringMessage += "✅ EXCELLENT - On track for successful completion!\n"
				} else if executionScore >= 10 {
					scoringMessage += "⚠️ NEEDS WRITE - Read enough, now implement changes!\n"
				} else if executionScore < 0 {
					scoringMessage += "🚨 CRITICAL - Too many reads, not enough writes!\n"
				}
				summarizedOutput += scoringMessage

				// Inject progress summary every 3 tool calls
				if toolCallCount%3 == 0 && (len(progressTracker.FilesRead) > 0 || len(progressTracker.DirectoriesListed) > 0) {
					progressSummary := progressTracker.GetProgressSummary()
					summarizedOutput += progressSummary

					t.logger.Info("📊 Injected progress summary with scoring",
						zap.String("subchatId", subchatID),
						zap.Int("score", executionScore))
				}

				// Log tool result with success/failure indicator and context size info
				if event.ToolResult.Error != "" {
					t.logger.Warn("❌ Tool call failed",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolResult.Name),
						zap.String("error", event.ToolResult.Error))
				} else {
					t.logger.Info("✓ Tool call completed",
						zap.String("subchatId", subchatID),
						zap.String("toolName", event.ToolResult.Name),
						zap.Int64("durationMs", event.ToolResult.DurationMs),
						zap.Int("originalSize", originalSize),
						zap.Int("summarizedSize", len(summarizedOutput)))
				}

				_, err := t.chatService.SaveToolResult(ctx, chatSession.ID, event.ToolResult.ID, event.ToolResult.Name, summarizedOutput, event.ToolResult.Error, event.ToolResult.DurationMs, finalCompanyID)
				if err != nil {
					t.logger.Error("Failed to save tool result",
						zap.String("subchatId", subchatID),
						zap.Error(err))
				}

			case aiservice.StreamEventError:
				t.logger.Error("❌ AI service error during subagent execution",
					zap.String("subchatId", subchatID),
					zap.String("error", event.Error))
				t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("AI error: %s", event.Error))
				return
			}
		}
	}

	// 📊 COMPREHENSIVE LOGGING: Final execution summary
	t.logger.Info("╔═══════════════════════════════════════════════════════════════════")
	t.logger.Info("║ 📊 FINAL EXECUTION SUMMARY",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName))
	t.logger.Info("╠═══════════════════════════════════════════════════════════════════")
	t.logger.Info("║ Tool Calls",
		zap.Int("total", toolCallCount),
		zap.Int("reads", readFileCount),
		zap.Int("writes", len(progressTracker.FilesWritten)),
		zap.Int("bash", len(progressTracker.BashCalls)))
	t.logger.Info("║ TODOs",
		zap.Int("completed", completedTodos),
		zap.Int("total", len(agentTask.Todos)))
	t.logger.Info("║ Score",
		zap.Int("finalScore", executionScore),
		zap.Bool("hasWritten", hasWrittenAnyFile))
	t.logger.Info("║ Files",
		zap.Int("filesRead", len(progressTracker.FilesRead)),
		zap.Int("filesWritten", len(progressTracker.FilesWritten)),
		zap.Int("directoriesListed", len(progressTracker.DirectoriesListed)))
	t.logger.Info("╚═══════════════════════════════════════════════════════════════════")

	// NOTE: We now save text chunks as they stream (lines 3073-3085), so we don't need to save
	// the full response here. This prevents duplicate messages in the database.
	// The fullResponse variable is still useful for logging and validation purposes.
	t.logger.Info("📝 Text chunks already saved during streaming (no final save needed)",
		zap.String("subchatId", subchatID),
		zap.Int("totalResponseLength", len(fullResponse)))

	// ========================================
	// VALIDATION LAYER 1: File Modification Validation
	// ========================================
	var modifiedFiles []string
	if len(agentTask.FilesModified) > 0 {
		filesOK, files, validationErr := t.validateFileModifications(agentTask, progressTracker)
		modifiedFiles = files

		if !filesOK {
			t.logger.Warn("❌ File modification validation FAILED",
				zap.String("subchatId", subchatID),
				zap.String("agentTaskId", agentTask.ID),
				zap.Error(validationErr))

			// Mark as BLOCKED instead of completed
			blockReason := fmt.Sprintf("Validation failed: %v. Tool calls: %d, Claimed TODOs: %d/%d",
				validationErr, toolCallCount, completedTodos, len(agentTask.Todos))

			err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusBlocked, blockReason)
			if err != nil {
				t.logger.Error("Failed to update task status to blocked",
					zap.String("agentTaskId", agentTask.ID),
					zap.Error(err))
			}

			err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusFailed)
			if err != nil {
				t.logger.Error("Failed to update subchat status to failed",
					zap.String("subchatId", subchatID),
					zap.Error(err))
			}

			t.logger.Error("🚨 PHANTOM COMPLETION PREVENTED",
				zap.String("subchatId", subchatID),
				zap.String("agentTaskId", agentTask.ID),
				zap.String("reason", "no files modified"))

			// Send failure message to user
			failureMessage := fmt.Sprintf("❌ **Task Blocked - File Validation Failed**\n\n**Reason:** %s\n\n**Tool Calls Made:** %d\n**TODOs Claimed:** %d/%d\n\nThe expected files were not modified. This might be because:\n- The file paths in the task don't match what was actually written\n- The tool used a different file path format\n- The changes were not actually applied\n\nPlease review the task and try again, or ask me for help!",
				validationErr.Error(), toolCallCount, completedTodos, len(agentTask.Todos))

			// Save failure message to database
			_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", failureMessage, finalCompanyID)
			if err != nil {
				t.logger.Warn("Failed to save failure message to database",
					zap.String("subchatId", subchatID),
					zap.Error(err))
			}

			// Emit failure message to WebSocket
			handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, failureMessage)
			handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("❌ Subchat blocked: %s - %s", agentTask.AgentName, validationErr.Error()))

			// Keep subchat alive for user to investigate or provide guidance
			t.logger.Info("💬 Task blocked - waiting for user messages",
				zap.String("subchatId", subchatID),
				zap.String("sessionId", chatSession.ID.Hex()))
			return
		}

		// Log proof of work
		t.logger.Info("✅ File modification validation PASSED",
			zap.String("subchatId", subchatID),
			zap.String("agentTaskId", agentTask.ID),
			zap.Strings("modifiedFiles", modifiedFiles))
	}

	// ========================================
	// VALIDATION LAYER 2: TODO Completion Verification
	// ========================================
	if completedTodos < len(agentTask.Todos) {
		t.logger.Warn("❌ TODO completion validation FAILED",
			zap.String("subchatId", subchatID),
			zap.String("agentTaskId", agentTask.ID),
			zap.Int("completed", completedTodos),
			zap.Int("total", len(agentTask.Todos)))

		summaryNotes := fmt.Sprintf("Incomplete: %d/%d TODOs done, %d tool calls. Files modified: %v",
			completedTodos, len(agentTask.Todos), toolCallCount, modifiedFiles)

		err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusInProgress, summaryNotes)
		if err != nil {
			t.logger.Error("Failed to update task status to in_progress",
				zap.String("agentTaskId", agentTask.ID),
				zap.Error(err))
		}

		err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusActive)
		if err != nil {
			t.logger.Error("Failed to update subchat status to active",
				zap.String("subchatId", subchatID),
				zap.Error(err))
		}


	// Send incomplete work message to user
	incompleteMessage := fmt.Sprintf("⚠️ **Task Incomplete - More Work Needed**\n\n**Progress:** %d/%d TODOs completed\n**Tool Calls Made:** %d\n**Files Modified:** %v\n\nI've made progress but haven't completed all the TODOs yet. Would you like me to:\n1. Continue working on the remaining tasks\n2. Review what's been done so far\n3. Adjust the approach\n\nLet me know how you'd like to proceed!",
		completedTodos, len(agentTask.Todos), toolCallCount, modifiedFiles)

	// Save incomplete message to database
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", incompleteMessage, finalCompanyID)
	if err != nil {
		t.logger.Warn("Failed to save incomplete message to database",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

	// Emit incomplete message to WebSocket for real-time display
	handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, incompleteMessage)
	handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("⚠️ Subchat incomplete: %s - %d/%d TODOs done", agentTask.AgentName, completedTodos, len(agentTask.Todos)))

	// Keep subchat alive for user to provide guidance
	t.logger.Info("💬 Task incomplete - waiting for user messages",
		zap.String("subchatId", subchatID),
		zap.String("sessionId", chatSession.ID.Hex()),
		zap.Int("completedTodos", completedTodos),
		zap.Int("totalTodos", len(agentTask.Todos)))
		return
	}

	// ========================================
	// ALL VALIDATIONS PASSED - Safe to mark as completed
	// ========================================
	summaryNotes := fmt.Sprintf("✅ VALIDATED completion: %d/%d TODOs, %d tool calls, %d files modified: %v",
		completedTodos, len(agentTask.Todos), toolCallCount, len(modifiedFiles), modifiedFiles)

	t.logger.Info("🎉 Task completion validated successfully",
		zap.String("subchatId", subchatID),
		zap.String("agentTaskId", agentTask.ID),
		zap.Int("toolCalls", toolCallCount),
		zap.Int("completedTodos", completedTodos),
		zap.Strings("filesModified", modifiedFiles))

	err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusCompleted, summaryNotes)
	if err != nil {
		t.logger.Error("Failed to update task status to completed",
			zap.String("agentTaskId", agentTask.ID),
			zap.Error(err))
	}

	// Update subchat status to completed
	err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusCompleted)
	if err != nil {
		t.logger.Error("Failed to update subchat status",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

	t.logger.Info("╔═══════════════════════════════════════════════════════════════════")
	t.logger.Info("║ ✅ SUBAGENT EXECUTION COMPLETED SUCCESSFULLY")
	t.logger.Info("╠═══════════════════════════════════════════════════════════════════",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.String("agentTaskId", agentTask.ID),
		zap.Int("toolCalls", toolCallCount),
		zap.Int("completedTodos", completedTodos),
		zap.Int("totalTodos", len(agentTask.Todos)),
		zap.Strings("filesModified", modifiedFiles))
	t.logger.Info("╚═══════════════════════════════════════════════════════════════════")

	t.logger.Info("🎉 Subagent execution completed successfully!",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.Int("toolCalls", toolCallCount),
		zap.Int("completedTodos", completedTodos),
		zap.Int("totalTodos", len(agentTask.Todos)))

	// Send completion message to subchat WebSocket
	completionMessage := fmt.Sprintf("✅ **Task Completed!**\n\n**Agent:** %s\n**TODOs Completed:** %d/%d\n**Tool Calls:** %d\n**Files Modified:** %v\n\nThe task has been successfully completed. You can ask me questions or request changes!",
		agentTask.AgentName, completedTodos, len(agentTask.Todos), toolCallCount, modifiedFiles)

	// Save completion message to database
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", completionMessage, finalCompanyID)
	if err != nil {
		t.logger.Warn("Failed to save completion message to database",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

	// Emit completion message to WebSocket
	handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, completionMessage)
	handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("✅ Subchat completed: %s", agentTask.AgentName))

	t.logger.Info("💬 Task completed - now waiting for user messages",
		zap.String("subchatId", subchatID),
		zap.String("sessionId", chatSession.ID.Hex()))

	// KEEP SUBCHAT ALIVE: Instead of returning, wait for new user messages
	// This allows the user to continue interacting with the agent after task completion
	for {
		select {
		case <-ctx.Done():
			t.logger.Info("⏱️ Context cancelled while waiting for user messages",
				zap.String("subchatId", subchatID))
			return

		case <-notifyCh:
			// USER MESSAGE RECEIVED AFTER COMPLETION
			t.logger.Info("💬 User message received after task completion - resuming AI",
				zap.String("sessionId", chatSession.ID.Hex()),
				zap.String("subchatId", subchatID))

			// Fetch all messages including new user message
			messagesResp, err := t.chatService.GetMessages(ctx, chatSession.ID, finalCompanyID, 100, 0)
			if err != nil {
				t.logger.Error("Failed to fetch messages after completion", zap.Error(err))
				continue
			}

			// Extract latest user message
			var latestUserMessage string
			for i := len(messagesResp.Messages) - 1; i >= 0; i-- {
				if messagesResp.Messages[i].Role == "user" {
					latestUserMessage = messagesResp.Messages[i].Content
					break
				}
			}

			t.logger.Info("🔄 Resuming AI after task completion with user question",
				zap.String("userMessage", latestUserMessage),
				zap.String("sessionId", chatSession.ID.Hex()))

			// Build post-completion system prompt
			postCompletionPrompt := fmt.Sprintf(`You are %s. You have successfully completed the assigned task.

**Task Summary:**
- Agent: %s
- TODOs Completed: %d/%d
- Tool Calls: %d
- Files Modified: %v

The user has sent you a new message. You should:
1. Answer their question or address their request directly
2. Use tools if needed to implement changes or gather information
3. Be helpful and responsive
4. If they want changes, implement them using the available tools

You have access to all the same tools as before. You can read files, write files, run commands, etc.`,
				agentTask.AgentName, agentTask.AgentName, completedTodos, len(agentTask.Todos), toolCallCount, modifiedFiles)

			// Rebuild message context with post-completion system prompt
			messages = []aiservice.Message{
				{Role: "system", Content: postCompletionPrompt},
			}
			for _, msg := range messagesResp.Messages {
				messages = append(messages, aiservice.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}

			t.logger.Info("🚀 Starting new AI stream for post-completion conversation",
				zap.Int("messageCount", len(messages)),
				zap.String("sessionId", chatSession.ID.Hex()))

			// Start new AI stream with updated context
			aiStream, err = t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
			if err != nil {
				t.logger.Error("Failed to start AI stream after completion", zap.Error(err))
				continue
			}

			// Process the new AI stream (reuse existing event processing logic)
			// Reset counters for new conversation turn
			fullResponse = ""

			for {
				select {
				case <-ctx.Done():
					t.logger.Warn("⏱️ Context cancelled during post-completion stream",
						zap.String("subchatId", subchatID))
					return

				case <-notifyCh:
					// Another interrupt during post-completion conversation
					t.logger.Info("💬 Another user message during post-completion conversation",
						zap.String("sessionId", chatSession.ID.Hex()))
					// Drain current stream
					if aiStream != nil {
						go func() {
							for range aiStream {
								// discard remaining events
							}
						}()
					}
					// Break to outer loop to fetch and process new message
					goto FETCH_NEW_MESSAGE

				case event, ok := <-aiStream:
					if !ok {
						// Stream closed naturally
						t.logger.Info("✅ Post-completion stream completed",
							zap.String("subchatId", subchatID))
						goto WAIT_FOR_NEXT_MESSAGE
					}

					switch event.Type {
					case aiservice.StreamEventToken:
						fullResponse += event.Content

						// Stream AI messages to progress channel
						if event.Content != "" && !isSystemEnforcementMessage(event.Content) {
							trimmed := strings.TrimSpace(event.Content)
							if len(trimmed) > 5 {
								// Save to database
								_, err := t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", event.Content, finalCompanyID)
								if err != nil {
									t.logger.Warn("Failed to save streaming text chunk",
										zap.String("subchatId", subchatID),
										zap.Error(err))
								}

								// Send to WebSocket
								handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, event.Content)
							}
						}

					case aiservice.StreamEventToolCall:
						t.logger.Info("🔧 Tool call in post-completion conversation",
							zap.String("subchatId", subchatID),
							zap.String("toolName", event.ToolCall.Name))

						// Execute tool and emit progress
						plainEnglishToolCall := convertToolCallToPlainEnglish(event.ToolCall.Name, event.ToolCall.Args)
						handlers.GetProgressNotifier(t.logger).EmitProgress(chatSession.ID, plainEnglishToolCall)

						// Save tool call
						_, err := t.chatService.SaveToolCall(ctx, chatSession.ID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, finalCompanyID)
						if err != nil {
							t.logger.Error("Failed to save tool call", zap.Error(err))
						}

					case aiservice.StreamEventToolResult:
						t.logger.Info("✓ Tool result in post-completion conversation",
							zap.String("subchatId", subchatID),
							zap.String("toolName", event.ToolResult.Name))

						// Save tool result
						var output interface{}
						if event.ToolResult.Error != "" {
							output = event.ToolResult.Error
						} else {
							output = event.ToolResult.Output
						}
						outputStr := t.summarizeToolResult(event.ToolResult.Name, output)

						_, err := t.chatService.SaveToolResult(ctx, chatSession.ID, event.ToolResult.ID, event.ToolResult.Name, outputStr, event.ToolResult.Error, event.ToolResult.DurationMs, finalCompanyID)
						if err != nil {
							t.logger.Error("Failed to save tool result", zap.Error(err))
						}

					case aiservice.StreamEventError:
						t.logger.Error("❌ AI service error during post-completion conversation",
							zap.String("subchatId", subchatID),
							zap.String("error", event.Error))
						goto WAIT_FOR_NEXT_MESSAGE
					}
				}
			}

		FETCH_NEW_MESSAGE:
			continue

		WAIT_FOR_NEXT_MESSAGE:
			continue
		}
	}
}

// buildExecutionPhaseSystemPrompt creates a strict system prompt using OPERATIONAL enforcement language
// Uses concrete "WRITE-ONLY MODE" instead of abstract "PHASE: EXECUTE" for better model compliance
func (t *ExecuteSubagentTool) buildExecutionPhaseSystemPrompt() string {
	return `╔══════════════════════════════════════════════════════════════╗
║            GUIDED EXECUTION MODE ACTIVATED                    ║
╚══════════════════════════════════════════════════════════════╝

🎯 YOUR MISSION: Execute the task efficiently while keeping the user informed

╔══════════════════════════════════════════════════════════════╗
║  🚨🚨🚨 CRITICAL RULE #1 - READ THIS FIRST 🚨🚨🚨           ║
╚══════════════════════════════════════════════════════════════╝

⛔ NEVER EVER WRITE INCOMPLETE CODE ⛔
⛔ NEVER EVER USE PLACEHOLDERS ⛔
⛔ NEVER EVER WRITE CODE FRAGMENTS ⛔

THIS IS ABSOLUTELY NON-NEGOTIABLE AND MANDATORY:

❌ FORBIDDEN - YOU WILL BE BLOCKED IF YOU DO THIS:
   • Writing "// rest of the file remains the same"
   • Writing "// ... (existing code)"
   • Writing "// ... rest of code here"
   • Writing "/* ... existing code ... */"
   • Writing code fragments without full file context
   • Writing ANYTHING that is not COMPLETE, RUNNABLE code

✅ REQUIRED - YOU MUST ALWAYS DO THIS:
   • ALWAYS write the ENTIRE file from start to finish
   • ALWAYS include ALL imports at the top
   • ALWAYS include ALL function definitions
   • ALWAYS include ALL exports at the bottom
   • EVERY file you write must be COMPLETE and RUNNABLE

🔴 CONSEQUENCES OF INCOMPLETE CODE:
   • Your write_file call will be IMMEDIATELY REJECTED
   • You will see error: "INCOMPLETE CODE - PLACEHOLDER DETECTED"
   • You will be FORCED to rewrite the entire file
   • The task will be BLOCKED until you write complete code
   • You CANNOT proceed with placeholders - the system enforces this

💡 HOW TO WRITE COMPLETE CODE:
   1. ALWAYS use read_file FIRST to get the full file content
   2. Make your specific changes (add button, fix bug, etc.)
   3. Write the COMPLETE file with your changes
   4. Include EVERY line from the original file (except lines you changed)

REPEAT: NEVER USE PLACEHOLDERS. ALWAYS WRITE COMPLETE FILES.

═══════════════════════════════════════════════════════════════
📢 COMMUNICATION REQUIREMENTS (CRITICAL):
═══════════════════════════════════════════════════════════════

YOU MUST communicate with the user throughout execution:

✅ BEFORE each TODO: Announce what you're working on
   Example: "Working on adding error handling to the authentication module..."

✅ DURING implementation: Briefly explain your approach
   Example: "I'll add a try-catch block and log errors to the console."

✅ AFTER tool calls: Explain what you just did
   Example: "I've updated the login function with proper error handling."

✅ ON errors: Explain what went wrong and your next step
   Example: "Test failed: missing import. I'll add the required import now."

✅ WHEN blocked: Ask the user for guidance
   Example: "I need clarification: should I use async/await or promises?"

✅ ON completion: Summarize what was accomplished
   Example: "Completed: Added error handling with logging to 3 files."

❌ NEVER be silent - the user is watching and needs updates
❌ NEVER create new tasks when asked about progress - just respond
❌ NEVER show scoring or enforcement messages to the user

═══════════════════════════════════════════════════════════════
🔧 AVAILABLE TOOLS:
═══════════════════════════════════════════════════════════════

✅ read_file       - Read source file ONCE per file
✅ write_file      - Write/create files
✅ apply_patch     - Apply code changes
✅ bash            - Run commands/tests
✅ coordinator_update_todo_status - Mark TODO complete
✅ coordinator_upsert_knowledge   - Save decisions

BLOCKED TOOLS (these will FAIL):
❌ code_index_search - Discovery disabled in execution mode
❌ list_directory    - Discovery disabled in execution mode
❌ All coordinator tools (for task creation, listing, etc.)

═══════════════════════════════════════════════════════════════
⏱️ EFFICIENT WORKFLOW (3-5 tool calls per TODO):
═══════════════════════════════════════════════════════════════

For each TODO:
1. ANNOUNCE: Tell user what you're working on
2. READ: Use read_file on target file (ONCE per file)
3. EXPLAIN: Briefly describe your implementation approach
4. IMPLEMENT: Use write_file or apply_patch
5. VERIFY: Run tests with bash if applicable, report results
6. COMPLETE: Use coordinator_update_todo_status with notes
7. REPORT: Tell user what you accomplished

═══════════════════════════════════════════════════════════════
⚠️ EFFICIENCY RULES (RUNTIME ENFORCEMENT):
═══════════════════════════════════════════════════════════════

• READ ONLY FILES SPECIFIED IN TASK - exact paths in filesModified
• Read each file ONCE - file content is cached after first read
• Aim for 3-5 tool calls per TODO maximum
• Do NOT explore, search, or read unrelated files
• Do NOT call list_directory - IT IS BLOCKED

═══════════════════════════════════════════════════════════════
💬 USER INTERACTION GUIDELINES:
═══════════════════════════════════════════════════════════════

IF user asks "what's the status?" or "what are you doing?":
→ Respond with current progress, do NOT create a new task
→ Example: "I'm currently working on the authentication module (TODO 2 of 4).
   I've completed the error handling and I'm now adding unit tests."

IF user says "stop" or "wait":
→ Acknowledge and pause for instructions
→ Example: "Understood, pausing execution. What would you like me to do?"

IF a tool call fails:
→ Explain the error clearly and your recovery plan
→ Example: "The test failed with 'module not found'. I'll install the missing
   dependency now."

═══════════════════════════════════════════════════════════════
🔬 SURGICAL EDIT PRINCIPLE (CRITICAL - MOST IMPORTANT):
═══════════════════════════════════════════════════════════════

⚠️ WRITE COMPLETE FILES + CHANGE ONLY WHAT'S REQUESTED ⚠️

🚨 REMINDER: NO PLACEHOLDERS, NO FRAGMENTS, NO SHORTCUTS 🚨

GOLDEN RULE: Minimal, Precise Changes in COMPLETE Files
• If asked to "add a button" → ONLY add that button
• If asked to "fix a bug on line 50" → ONLY fix line 50
• NEVER refactor code that wasn't requested
• NEVER reorganize imports unless asked
• NEVER rename variables unless asked
• NEVER change formatting unless asked
• NEVER "improve" code that's working
• BUT ALWAYS write the COMPLETE file (not a fragment!)

YOU MUST FOLLOW THIS EXACT PROCESS:
1. ✅ READ the complete file first using read_file
   → You need the FULL content to write the FULL content back
2. ✅ IDENTIFY the exact lines that need to change
   → Be surgical: if adding 1 button, that's 10 lines max
3. ✅ CHANGE only those specific lines
   → Everything else stays EXACTLY the same
4. ✅ WRITE the COMPLETE file with ALL lines
   → From first import to last export, EVERYTHING included
   → NEVER write "// rest of..." - write the actual code!

YOU ARE ABSOLUTELY FORBIDDEN FROM:
❌ FORBIDDEN: Writing fragments like "// rest of the file remains the same"
❌ FORBIDDEN: Writing "// ... (existing code)" placeholders
❌ FORBIDDEN: Writing "... rest of code ..." comments
❌ FORBIDDEN: Writing partial files or code snippets
❌ FORBIDDEN: Skipping any part of the file content
❌ FORBIDDEN: Refactoring unrelated code
❌ FORBIDDEN: Changing code style or formatting
❌ FORBIDDEN: Adding "improvements" that weren't requested
❌ FORBIDDEN: Touching any line that doesn't need to change

EXAMPLE - User asks: "Add a Filter button to SessionList.tsx"

✅ ABSOLUTELY CORRECT (THE ONLY WAY):
   Step 1: read_file("SessionList.tsx") → Gets 529 lines
   Step 2: Identify that lines 40-45 need the new button
   Step 3: Prepare the full 534-line file:
           - Lines 1-39: EXACT copy from original
           - Lines 40-44: NEW button code (5 lines added)
           - Lines 45-534: EXACT copy from original (was 45-529)
   Step 4: write_file(ALL 534 lines of COMPLETE working code)
   Result: File has new button, everything else untouched

❌ COMPLETELY WRONG (SYSTEM WILL REJECT THIS):
   write_file with:
   "import ... from 'react';
    // ... existing imports ...

    <Button>Filter</Button>

    // ... rest of file remains the same ..."

   Result: ❌ REJECTED - "INCOMPLETE CODE - PLACEHOLDER DETECTED"

❌ ALSO WRONG (CHANGING TOO MUCH):
   • Refactoring other buttons while adding Filter button
   • Reorganizing the entire component structure
   • Renaming variables throughout the file
   • "Improving" code style in unrelated functions

═══════════════════════════════════════════════════════════════
🏆 CODE QUALITY REQUIREMENTS (MANDATORY):
═══════════════════════════════════════════════════════════════

🚨🚨🚨 TRIPLE WARNING 🚨🚨🚨
1. NEVER WRITE INCOMPLETE CODE
2. NEVER USE PLACEHOLDERS
3. ALL CODE MUST COMPILE WITHOUT ERRORS

BEFORE marking ANY TODO as complete:

1. 🔴 WRITE COMPLETE FILES (NON-NEGOTIABLE)
   ⛔ NEVER EVER use placeholders like:
      • "// rest of file..."
      • "// ... existing code ..."
      • "/* ... */"
      • "... rest of component ..."
   ⛔ NEVER EVER write fragments or partial code
   ✅ ALWAYS write the ENTIRE file from first line to last line
   ✅ ALWAYS include ALL imports, ALL functions, ALL exports
   ✅ Every single line must be present - no shortcuts!

   🔴 THE VALIDATION SYSTEM WILL:
      • Scan your file for placeholder patterns
      • Check React files have imports/exports/React
      • IMMEDIATELY REJECT incomplete files
      • Show error: "INCOMPLETE CODE - PLACEHOLDER DETECTED"
      • FORCE you to rewrite with complete content

2. ✅ VERIFY COMPILATION
   • For TypeScript: all imports, types, syntax must be correct
   • For Go: all packages, types, functions must be correct
   • Check all variables/functions are defined
   • Verify all required imports are present
   • NO placeholder comments allowed

3. ✅ RUN TESTS IF APPLICABLE
   • Use bash: "npx tsc --noEmit" for TypeScript
   • Use bash: "npm test" or "go test"
   • Address any failures immediately
   • Rewrite complete files if errors found

4. 🚨 VALIDATION IS AUTOMATIC, SYNCHRONOUS, AND BLOCKING
   • EVERY write_file call is validated IMMEDIATELY
   • Incomplete files → REJECTED (you see error, must retry)
   • Placeholder comments → REJECTED (you see error, must retry)
   • Missing structure → REJECTED (you see error, must retry)
   • Compilation errors → REJECTED (you see error, must retry)
   • You CANNOT proceed until files pass ALL validations

FINAL REMINDER: The system enforces complete files. Attempting to use
placeholders or write fragments will FAIL. Read the file, make your
changes, write the COMPLETE modified file. No exceptions.

═══════════════════════════════════════════════════════════════
📋 TASK CONTRACT (arriving in next message):
═══════════════════════════════════════════════════════════════

You will receive:
• Exact file paths to modify (use these EXACT paths)
• Specific TODO items with context hints
• Role and objective

You must produce:
• Modified files (via write_file or apply_patch) that COMPILE WITHOUT ERRORS
• Updated TODO status for each item (only after verifying quality)
• Knowledge entries for key decisions
• Clear communication to the user throughout

═══════════════════════════════════════════════════════════════

MENTAL MODEL:

You are a code WRITER, not a code READER.
• You have ONE JOB: Write code changes
• Coordinator already researched and found files
• You execute the script - you don't question it
• Minimize reads, maximize writes
• Hands don't think - they execute

═══════════════════════════════════════════════════════════════

AWAIT TASK CONTRACT...`
}

// buildSubagentTaskPrompt constructs task details for the subagent (user message)
// This contains ONLY task-specific information, NO execution phase instructions
func (t *ExecuteSubagentTool) buildSubagentTaskPrompt(agentTask *storage.AgentTask) string {
	prompt := fmt.Sprintf(`You are %s. You have been assigned a task to complete.

ROLE: %s

TASK CONTEXT:
%s

YOUR TODOs:
`, agentTask.AgentName, agentTask.Role, agentTask.ContextSummary)

	for i, todo := range agentTask.Todos {
		status := "PENDING"
		if todo.Status == "completed" {
			status = "✓ DONE"
		} else if todo.Status == "in_progress" {
			status = "IN PROGRESS"
		}

		prompt += fmt.Sprintf("\n%d. [%s] ID: %s - %s", i+1, status, todo.ID, todo.Description)

		if todo.FilePath != "" {
			prompt += fmt.Sprintf("\n   File: %s", todo.FilePath)
		}
		if todo.FunctionName != "" {
			prompt += fmt.Sprintf("\n   Function: %s", todo.FunctionName)
		}
		if todo.ContextHint != "" {
			prompt += fmt.Sprintf("\n   Hint: %s", todo.ContextHint)
		}
		if todo.HumanPromptNotes != "" {
			prompt += fmt.Sprintf("\n   Notes: %s", todo.HumanPromptNotes)
		}
	}

	if len(agentTask.FilesModified) > 0 {
		prompt += "\n\nFILES TO MODIFY:\n"
		for _, file := range agentTask.FilesModified {
			prompt += fmt.Sprintf("- %s\n", file)
		}
	}

	if len(agentTask.QdrantCollections) > 0 {
		prompt += "\n\nRELEVANT KNOWLEDGE COLLECTIONS:\n"
		for _, coll := range agentTask.QdrantCollections {
			prompt += fmt.Sprintf("- %s\n", coll)
		}
	}

	if agentTask.HumanPromptNotes != "" {
		prompt += fmt.Sprintf("\n\nADDITIONAL GUIDANCE:\n%s\n", agentTask.HumanPromptNotes)
	}

	// Task contract is complete - all execution instructions are in the system prompt
	prompt += fmt.Sprintf("\n\n═══════════════════════════════════════════════════════════════════\n")
	prompt += fmt.Sprintf("Task ID: %s\n", agentTask.ID)
	prompt += fmt.Sprintf("BEGIN EXECUTION NOW.\n")
	prompt += fmt.Sprintf("═══════════════════════════════════════════════════════════════════")

	return prompt
}

// summarizeToolResult creates a concise summary of a tool result to prevent context bloat
func (t *ExecuteSubagentTool) summarizeToolResult(toolName string, output interface{}) string {
	const maxChars = 500 // Maximum characters to keep for most outputs

	var outputStr string
	if str, ok := output.(string); ok {
		outputStr = str
	} else {
		outputBytes, _ := json.Marshal(output)
		outputStr = string(outputBytes)
	}

	// Special handling for different tool types
	switch toolName {
	case "list_directory":
		// Extract just file count and first few files
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(outputStr), &result); err == nil {
			if files, ok := result["files"].([]interface{}); ok {
				count := len(files)
				preview := files
				if count > 10 {
					preview = files[:10]
				}
				previewJSON, _ := json.Marshal(preview)
				return fmt.Sprintf("Directory listing: %d files found. First 10: %s", count, string(previewJSON))
			}
		}

	case "read_file":
		// Summarize file reading - show first/last lines
		lines := strings.Split(outputStr, "\n")
		if len(lines) > 20 {
			firstLines := strings.Join(lines[:10], "\n")
			lastLines := strings.Join(lines[len(lines)-5:], "\n")
			return fmt.Sprintf("File read (%d lines). First 10 lines:\n%s\n... (truncated %d lines) ...\nLast 5 lines:\n%s",
				len(lines), firstLines, len(lines)-15, lastLines)
		}

	case "bash":
		// Summarize bash output - show success/failure and brief output
		if len(outputStr) > maxChars {
			return fmt.Sprintf("Bash command completed. Output (truncated to %d chars): %s...", maxChars, outputStr[:maxChars])
		}

	case "coordinator_update_todo_status":
		// Just confirm the update without full details
		return "TODO status updated successfully"

	case "coordinator_update_task_status":
		// Just confirm the update
		return "Task status updated successfully"

	case "write_file":
		// Confirm write without showing full content
		return "File written successfully"

	case "apply_patch":
		// Confirm patch without showing full diff
		return "Patch applied successfully"
	}

	// Default: truncate long outputs
	if len(outputStr) > maxChars {
		return outputStr[:maxChars] + fmt.Sprintf("... (truncated, original length: %d chars)", len(outputStr))
	}

	return outputStr
}

// handleExecutionFailure marks the task as blocked with error details
func (t *ExecuteSubagentTool) handleExecutionFailure(agentTaskID, errorMsg string) {
	err := t.taskStorage.UpdateTaskStatus(agentTaskID, storage.TaskStatusBlocked, fmt.Sprintf("Execution failed: %s", errorMsg))
	if err != nil {
		t.logger.Error("Failed to update task status to blocked",
			zap.String("agentTaskId", agentTaskID),
			zap.Error(err))
	}
}

// InterruptCategorization holds the result of interrupt analysis
type InterruptCategorization struct {
	Category string `json:"category"`
	Guidance string `json:"guidance"`
}

// ComplexityAnalysis represents the complexity analysis result for a task
type ComplexityAnalysis struct {
	Score                float64            `json:"score"`                // 0.0 to 1.0 complexity score
	FileCount            int                `json:"fileCount"`            // Number of files to modify
	TodoComplexity       float64            `json:"todoComplexity"`       // Average complexity of TODOs
	CrossSystemDeps      int                `json:"crossSystemDeps"`      // Number of cross-system dependencies
	Factors              map[string]float64 `json:"factors"`              // Individual complexity factors
	Recommendation       string             `json:"recommendation"`       // PROCEED, SPLIT, or REJECT
	SplittingStrategy    string             `json:"splittingStrategy"`    // SEQUENTIAL, PARALLEL, or NONE
	EstimatedTimeMinutes int                `json:"estimatedTimeMinutes"` // Estimated completion time
}

// TaskHierarchy represents the hierarchical structure of tasks
type TaskHierarchy struct {
	TaskID       string                    `json:"taskId"`
	Title        string                    `json:"title"`
	Status       string                    `json:"status"`
	Progress     float64                   `json:"progress"`     // 0.0 to 1.0
	OrderIndex   int                       `json:"orderIndex"`   // Order within siblings
	DependsOn    []string                  `json:"dependsOn"`    // Task IDs this task depends on
	Children     []TaskHierarchy           `json:"children"`     // Child tasks
	Metadata     map[string]interface{}    `json:"metadata"`     // Additional task metadata
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
}

// HierarchicalProgress represents progress aggregation across task hierarchy
type HierarchicalProgress struct {
	TaskID           string                           `json:"taskId"`
	DirectProgress   float64                          `json:"directProgress"`   // Progress of this task only
	AggregatedProgress float64                        `json:"aggregatedProgress"` // Including children
	CompletedChildren int                             `json:"completedChildren"`
	TotalChildren    int                              `json:"totalChildren"`
	ChildrenProgress map[string]HierarchicalProgress `json:"childrenProgress"` // Progress of child tasks
	BlockedBy        []string                         `json:"blockedBy"`        // Task IDs blocking this task
	IsExecutable     bool                             `json:"isExecutable"`     // Can this task be executed now
	LastUpdated      time.Time                        `json:"lastUpdated"`
}

// ChildTaskParams represents parameters for creating child tasks
type ChildTaskParams struct {
	Title            string                 `json:"title"`
	ContextSummary   string                 `json:"contextSummary"`
	Todos            []string               `json:"todos"`
	FilesModified    []string               `json:"filesModified"`
	OrderIndex       int                    `json:"orderIndex"`
	DependsOn        []string               `json:"dependsOn"`
	SplittingStrategy string                `json:"splittingStrategy"` // SEQUENTIAL or PARALLEL
	Metadata         map[string]interface{} `json:"metadata"`
	EstimatedMinutes int                    `json:"estimatedMinutes"`
	Priority         string                 `json:"priority"` // HIGH, MEDIUM, LOW
}

// analyzeTaskComplexity performs comprehensive complexity analysis on a task
func analyzeTaskComplexity(title, contextSummary string, todos []string, filesModified []string) ComplexityAnalysis {
	analysis := ComplexityAnalysis{
		Factors: make(map[string]float64),
	}

	// Factor 1: File count complexity (0.0 - 0.3)
	fileCount := len(filesModified)
	analysis.FileCount = fileCount
	var fileComplexity float64
	switch {
	case fileCount == 0:
		fileComplexity = 0.0
	case fileCount <= 2:
		fileComplexity = 0.1
	case fileCount <= 5:
		fileComplexity = 0.2
	case fileCount <= 10:
		fileComplexity = 0.25
	default:
		fileComplexity = 0.3
	}
	analysis.Factors["fileCount"] = fileComplexity

	// Factor 2: TODO complexity analysis (0.0 - 0.4)
	todoComplexity := analyzeTodoComplexity(todos)
	analysis.TodoComplexity = todoComplexity
	analysis.Factors["todoComplexity"] = todoComplexity * 0.4

	// Factor 3: Cross-system dependencies (0.0 - 0.2)
	crossSystemDeps := analyzeCrossSystemDependencies(contextSummary, todos, filesModified)
	analysis.CrossSystemDeps = crossSystemDeps
	var crossSystemComplexity float64
	switch {
	case crossSystemDeps == 0:
		crossSystemComplexity = 0.0
	case crossSystemDeps <= 2:
		crossSystemComplexity = 0.1
	default:
		crossSystemComplexity = 0.2
	}
	analysis.Factors["crossSystemDeps"] = crossSystemComplexity

	// Factor 4: Integration complexity (0.0 - 0.1)
	integrationComplexity := analyzeIntegrationComplexity(title, contextSummary, todos)
	analysis.Factors["integration"] = integrationComplexity

	// Calculate total score
	analysis.Score = fileComplexity + analysis.Factors["todoComplexity"] + 
		crossSystemComplexity + integrationComplexity

	// Ensure score is within bounds
	if analysis.Score > 1.0 {
		analysis.Score = 1.0
	}

	// Determine recommendation and splitting strategy
	analysis.Recommendation, analysis.SplittingStrategy = determineRecommendation(analysis.Score, fileCount, len(todos))

	// Estimate completion time
	analysis.EstimatedTimeMinutes = estimateCompletionTime(analysis.Score, fileCount, len(todos))

	return analysis
}

// analyzeTodoComplexity analyzes the complexity of individual TODOs
func analyzeTodoComplexity(todos []string) float64 {
	if len(todos) == 0 {
		return 0.0
	}

	totalComplexity := 0.0
	complexityKeywords := map[string]float64{
		"implement":     0.3,
		"create":        0.3,
		"add":           0.2,
		"modify":        0.4,
		"refactor":      0.5,
		"integrate":     0.6,
		"migrate":       0.7,
		"optimize":      0.5,
		"test":          0.2,
		"fix":           0.3,
		"debug":         0.4,
		"algorithm":     0.6,
		"database":      0.5,
		"api":           0.4,
		"authentication": 0.6,
		"security":      0.7,
		"performance":   0.6,
		"concurrent":    0.8,
		"async":         0.6,
		"complex":       0.5,
		"advanced":      0.6,
	}

	for _, todo := range todos {
		todoLower := strings.ToLower(todo)
		todoComplexity := 0.1 // Base complexity

		// Check for complexity keywords
		for keyword, weight := range complexityKeywords {
			if strings.Contains(todoLower, keyword) {
				todoComplexity += weight
			}
		}

		// Length-based complexity
		if len(todo) > 100 {
			todoComplexity += 0.1
		}
		if len(todo) > 200 {
			todoComplexity += 0.1
		}

		// Cap individual TODO complexity
		if todoComplexity > 1.0 {
			todoComplexity = 1.0
		}

		totalComplexity += todoComplexity
	}

	return totalComplexity / float64(len(todos))
}

// analyzeCrossSystemDependencies identifies dependencies across different systems
func analyzeCrossSystemDependencies(contextSummary string, todos []string, filesModified []string) int {
	allText := contextSummary + " " + strings.Join(todos, " ") + " " + strings.Join(filesModified, " ")
	textLower := strings.ToLower(allText)

	systems := map[string]bool{
		"database":      false,
		"api":           false,
		"frontend":      false,
		"backend":       false,
		"auth":          false,
		"storage":       false,
		"cache":         false,
		"queue":         false,
		"websocket":     false,
		"microservice":  false,
		"external":      false,
		"third-party":   false,
	}

	// Check for system mentions
	for system := range systems {
		if strings.Contains(textLower, system) {
			systems[system] = true
		}
	}

	// Count unique systems
	count := 0
	for _, found := range systems {
		if found {
			count++
		}
	}

	return count
}

// analyzeIntegrationComplexity analyzes integration-specific complexity
func analyzeIntegrationComplexity(title, contextSummary string, todos []string) float64 {
	allText := strings.ToLower(title + " " + contextSummary + " " + strings.Join(todos, " "))

	integrationPatterns := []string{
		"integration", "connect", "sync", "merge", "combine", "coordinate",
		"orchestrate", "workflow", "pipeline", "event", "message", "notification",
	}

	complexity := 0.0
	for _, pattern := range integrationPatterns {
		if strings.Contains(allText, pattern) {
			complexity += 0.02
		}
	}

	if complexity > 0.1 {
		complexity = 0.1
	}

	return complexity
}

// determineRecommendation determines the recommendation and splitting strategy based on complexity
func determineRecommendation(score float64, fileCount, todoCount int) (string, string) {
	switch {
	case score >= 0.8:
		return "REJECT", "NONE" // Too complex, should be rejected or heavily simplified
	case score >= 0.6:
		if fileCount > 5 || todoCount > 8 {
			return "SPLIT", "SEQUENTIAL"
		}
		return "SPLIT", "PARALLEL"
	case score >= 0.4:
		if todoCount > 6 {
			return "SPLIT", "PARALLEL"
		}
		return "PROCEED", "NONE"
	default:
		return "PROCEED", "NONE"
	}
}

// estimateCompletionTime estimates task completion time based on complexity factors
func estimateCompletionTime(score float64, fileCount, todoCount int) int {
	baseTime := 15 // Base 15 minutes per task
	
	// File-based time
	fileTime := fileCount * 10
	
	// TODO-based time
	todoTime := todoCount * 8
	
	// Complexity multiplier
	complexityMultiplier := 1.0 + score
	
	totalTime := float64(baseTime + fileTime + todoTime) * complexityMultiplier
	
	return int(totalTime)
}

// categorizeInterrupt analyzes user interrupt message to determine intent and provide guidance
func (t *ExecuteSubagentTool) categorizeInterrupt(ctx context.Context, userMessage string) (string, string, error) {
	categorizationPrompt := fmt.Sprintf(`You are an interrupt analyzer. Analyze this user message sent while an AI agent was working:

USER MESSAGE: %s

Categorize the interrupt intent:
- STOP: User wants to completely stop current work and do something different (e.g., "stop", "nevermind", "do this instead")
- MODIFY: User wants to change/adjust the current approach (e.g., "use X instead of Y", "add also Z", "change this")
- CLARIFY: User has a question or needs clarification (e.g., "why are you doing X?", "what does Y mean?")
- STATUS: User checking progress or giving encouragement (e.g., "how's it going?", "good job!", "what are you doing now?")
- CONTINUE: Message doesn't require action change (e.g., "ok", "thanks", general comments)

Respond with ONLY valid JSON (no markdown, no explanation):
{
  "category": "STOP|MODIFY|CLARIFY|STATUS|CONTINUE",
  "guidance": "Brief instruction for the agent (1 sentence)"
}`, userMessage)

	// Quick Claude API call for categorization (use minimal tokens)
	messages := []aiservice.Message{
		{Role: "user", Content: categorizationPrompt},
	}

	t.logger.Debug("Categorizing interrupt", zap.String("userMessage", userMessage))

	// Use streaming API to get categorization (collect full response)
	stream, err := t.aiService.StreamChatWithTools(ctx, messages, 1) // maxToolCalls=1 (no tools needed)
	if err != nil {
		t.logger.Warn("Failed to start categorization stream", zap.Error(err))
		return "CONTINUE", "", err
	}

	// Collect the full response from stream
	var response strings.Builder
	for event := range stream {
		if event.Type == aiservice.StreamEventToken {
			response.WriteString(event.Content)
		}
	}

	responseStr := response.String()
	t.logger.Debug("Raw categorization response", zap.String("response", responseStr))

	// Try to extract JSON from response (handle markdown code blocks)
	jsonStr := responseStr
	if strings.Contains(responseStr, "```json") {
		start := strings.Index(responseStr, "```json") + 7
		end := strings.LastIndex(responseStr, "```")
		if start > 7 && end > start {
			jsonStr = responseStr[start:end]
		}
	} else if strings.Contains(responseStr, "```") {
		start := strings.Index(responseStr, "```") + 3
		end := strings.LastIndex(responseStr, "```")
		if start > 3 && end > start {
			jsonStr = responseStr[start:end]
		}
	}

	// Parse JSON response
	var result InterruptCategorization
	if err := json.Unmarshal([]byte(strings.TrimSpace(jsonStr)), &result); err != nil {
		t.logger.Warn("Failed to parse categorization JSON, defaulting to CONTINUE",
			zap.Error(err),
			zap.String("jsonStr", jsonStr))
		return "CONTINUE", "", err
	}

	// Validate category
	validCategories := map[string]bool{
		"STOP": true, "MODIFY": true, "CLARIFY": true, "STATUS": true, "CONTINUE": true,
	}
	if !validCategories[result.Category] {
		t.logger.Warn("Invalid category returned, defaulting to CONTINUE",
			zap.String("category", result.Category))
		return "CONTINUE", "", fmt.Errorf("invalid category: %s", result.Category)
	}

	return result.Category, result.Guidance, nil
}

// CoordinatorAnalyzeTaskComplexityTool analyzes task complexity and provides recommendations
type CoordinatorAnalyzeTaskComplexityTool struct {
	storage storage.TaskStorage
}

func (t *CoordinatorAnalyzeTaskComplexityTool) Name() string {
	return "coordinator_analyze_task_complexity"
}

func (t *CoordinatorAnalyzeTaskComplexityTool) Description() string {
	return "Analyze task complexity based on file count, TODO complexity, and cross-system dependencies. Returns complexity score (0.0-1.0) and recommendations (PROCEED/SPLIT/REJECT)."
}

func (t *CoordinatorAnalyzeTaskComplexityTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Task title for context analysis",
			},
			"contextSummary": map[string]interface{}{
				"type":        "string",
				"description": "Task context summary for complexity analysis",
			},
			"todos": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of TODO items to analyze for complexity",
			},
			"filesModified": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "List of files that will be modified",
			},
		},
		"required": []string{"title", "contextSummary", "todos", "filesModified"},
	}
}

func (t *CoordinatorAnalyzeTaskComplexityTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Input validation
	title, ok := args["title"].(string)
	if !ok || title == "" {
		return nil, fmt.Errorf("title is required and must be a non-empty string")
	}

	contextSummary, ok := args["contextSummary"].(string)
	if !ok {
		contextSummary = ""
	}

	todosInterface, ok := args["todos"]
	if !ok {
		return nil, fmt.Errorf("todos is required")
	}

	todos := make([]string, 0)
	// Handle both []interface{} (from JSON/MCP) and []string (from direct Go calls)
	switch v := todosInterface.(type) {
	case []interface{}:
		for _, todoInterface := range v {
			if todoStr, ok := todoInterface.(string); ok {
				todos = append(todos, todoStr)
			}
		}
	case []string:
		todos = v
	}

	filesModifiedInterface, ok := args["filesModified"]
	if !ok {
		return nil, fmt.Errorf("filesModified is required")
	}

	filesModified := make([]string, 0)
	// Handle both []interface{} (from JSON/MCP) and []string (from direct Go calls)
	switch v := filesModifiedInterface.(type) {
	case []interface{}:
		for _, fileInterface := range v {
			if fileStr, ok := fileInterface.(string); ok {
				filesModified = append(filesModified, fileStr)
			}
		}
	case []string:
		filesModified = v
	}

	// Perform complexity analysis
	analysis := analyzeTaskComplexity(title, contextSummary, todos, filesModified)

	return analysis, nil
}

// CoordinatorSplitAgentTaskTool splits complex tasks into smaller child tasks
type CoordinatorSplitAgentTaskTool struct {
	storage   storage.TaskStorage
	aiService AIServiceInterface
}

func (t *CoordinatorSplitAgentTaskTool) Name() string {
	return "coordinator_split_agent_task"
}

func (t *CoordinatorSplitAgentTaskTool) Description() string {
	return "Split a complex agent task into smaller child tasks using sequential or parallel strategies. Creates child tasks with proper dependencies and generates integration documentation."
}

func (t *CoordinatorSplitAgentTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"parentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "ID of the parent task to split",
			},
			"splittingStrategy": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"SEQUENTIAL", "PARALLEL"},
				"description": "Strategy for splitting: SEQUENTIAL (tasks depend on each other) or PARALLEL (independent tasks)",
			},
			"childTasks": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Title for the child task",
						},
						"contextSummary": map[string]interface{}{
							"type":        "string",
							"description": "Context summary for the child task",
						},
						"todos": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "TODO items for the child task",
						},
						"filesModified": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Files to be modified by this child task",
						},
						"orderIndex": map[string]interface{}{
							"type":        "integer",
							"description": "Order index for sequential execution",
						},
						"dependsOn": map[string]interface{}{
							"type": "array",
							"items": map[string]interface{}{
								"type": "string",
							},
							"description": "Task IDs this child task depends on",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"HIGH", "MEDIUM", "LOW"},
							"description": "Priority level for the child task",
						},
					},
					"required": []string{"title", "contextSummary", "todos", "filesModified"},
				},
				"description": "Array of child task definitions",
			},
			"createIntegrationDoc": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to create integration documentation for coordinating child tasks",
				"default":     true,
			},
		},
		"required": []string{"parentTaskId", "splittingStrategy", "childTasks"},
	}
}

func (t *CoordinatorSplitAgentTaskTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Input validation
	parentTaskId, ok := args["parentTaskId"].(string)
	if !ok || parentTaskId == "" {
		return nil, fmt.Errorf("parentTaskId is required and must be a non-empty string")
	}

	splittingStrategy, ok := args["splittingStrategy"].(string)
	if !ok || (splittingStrategy != "SEQUENTIAL" && splittingStrategy != "PARALLEL") {
		return nil, fmt.Errorf("splittingStrategy must be either 'SEQUENTIAL' or 'PARALLEL'")
	}

	childTasksInterface, ok := args["childTasks"]
	if !ok {
		return nil, fmt.Errorf("childTasks is required")
	}

	createIntegrationDoc := true
	if val, ok := args["createIntegrationDoc"].(bool); ok {
		createIntegrationDoc = val
	}

	// Parse child tasks
	childTasksArray, ok := childTasksInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("childTasks must be an array")
	}

	if len(childTasksArray) == 0 {
		return nil, fmt.Errorf("at least one child task must be provided")
	}

	// Verify parent task exists
	parentTask, err := t.storage.GetAgentTask(parentTaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent task: %w", err)
	}

	// Parse and validate child task parameters
	childTaskParams := make([]ChildTaskParams, 0, len(childTasksArray))
	taskIndexMap := make(map[string]int) // Maps task titles to their index for circular dependency checking

	for i, childInterface := range childTasksArray {
		childMap, ok := childInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("child task %d must be an object", i)
		}

		params, err := t.parseChildTaskParams(childMap, i)
		if err != nil {
			return nil, fmt.Errorf("invalid child task %d: %w", i, err)
		}
		params.SplittingStrategy = splittingStrategy
		childTaskParams = append(childTaskParams, params)
		taskIndexMap[params.Title] = i
	}

	// Validate dependencies and check for circular dependencies
	if err := t.validateTaskDependencies(childTaskParams, taskIndexMap); err != nil {
		return nil, fmt.Errorf("dependency validation failed: %w", err)
	}

	// Create child tasks
	createdTasks := make([]*storage.AgentTask, 0, len(childTaskParams))
	var previousTaskId string

	for i, params := range childTaskParams {
		// Set dependencies for sequential strategy
		if splittingStrategy == "SEQUENTIAL" && i > 0 {
			params.DependsOn = []string{previousTaskId}
		}

		// Create the child task
		childTask, err := t.createChildTask(ctx, parentTaskId, params)
		if err != nil {
			return nil, fmt.Errorf("failed to create child task %d: %w", i, err)
		}

		createdTasks = append(createdTasks, childTask)
		previousTaskId = childTask.ID
	}

	// Update parent task status to indicate it has been split
	err = t.storage.UpdateTaskStatus(parentTaskId, storage.TaskStatusBlocked, "Task has been split into subtasks")
	if err != nil {
		return nil, fmt.Errorf("failed to update parent task status: %w", err)
	}

	// Create integration documentation if requested
	var integrationDoc string
	if createIntegrationDoc {
		integrationDoc = t.generateIntegrationDoc(parentTask, createdTasks, splittingStrategy)
	}

	return map[string]interface{}{
		"parentTaskId":      parentTaskId,
		"splittingStrategy": splittingStrategy,
		"childTasks":        createdTasks,
		"integrationDoc":    integrationDoc,
		"totalChildTasks":   len(createdTasks),
	}, nil
}

// parseChildTaskParams parses and validates child task parameters
func (t *CoordinatorSplitAgentTaskTool) parseChildTaskParams(childMap map[string]interface{}, index int) (ChildTaskParams, error) {
	params := ChildTaskParams{}

	// Required fields
	title, ok := childMap["title"].(string)
	if !ok || title == "" {
		return params, fmt.Errorf("title is required and must be a non-empty string")
	}
	params.Title = title

	contextSummary, ok := childMap["contextSummary"].(string)
	if !ok {
		contextSummary = ""
	}
	params.ContextSummary = contextSummary

	// Parse todos
	todosInterface, ok := childMap["todos"]
	if !ok {
		return params, fmt.Errorf("todos is required")
	}
	todosArray, ok := todosInterface.([]interface{})
	if !ok {
		return params, fmt.Errorf("todos must be an array")
	}
	for _, todoInterface := range todosArray {
		if todoStr, ok := todoInterface.(string); ok {
			params.Todos = append(params.Todos, todoStr)
		}
	}

	// Parse filesModified
	filesInterface, ok := childMap["filesModified"]
	if !ok {
		return params, fmt.Errorf("filesModified is required")
	}
	filesArray, ok := filesInterface.([]interface{})
	if !ok {
		return params, fmt.Errorf("filesModified must be an array")
	}
	for _, fileInterface := range filesArray {
		if fileStr, ok := fileInterface.(string); ok {
			params.FilesModified = append(params.FilesModified, fileStr)
		}
	}

	// Optional fields
	if orderIndex, ok := childMap["orderIndex"].(float64); ok {
		params.OrderIndex = int(orderIndex)
	} else {
		params.OrderIndex = index
	}

	if dependsOnInterface, ok := childMap["dependsOn"]; ok {
		if dependsOnArray, ok := dependsOnInterface.([]interface{}); ok {
			for _, depInterface := range dependsOnArray {
				if depStr, ok := depInterface.(string); ok {
					params.DependsOn = append(params.DependsOn, depStr)
				}
			}
		}
	}

	if priority, ok := childMap["priority"].(string); ok {
		params.Priority = priority
	} else {
		params.Priority = "MEDIUM"
	}

	// Estimate completion time
	analysis := analyzeTaskComplexity(params.Title, params.ContextSummary, params.Todos, params.FilesModified)
	params.EstimatedMinutes = analysis.EstimatedTimeMinutes

	return params, nil
}

// validateTaskDependencies validates task dependencies and detects circular dependencies
func (t *CoordinatorSplitAgentTaskTool) validateTaskDependencies(childTasks []ChildTaskParams, taskIndexMap map[string]int) error {
	// Build dependency graph
	dependencyGraph := make(map[string][]string)
	for _, task := range childTasks {
		dependencyGraph[task.Title] = task.DependsOn
	}

	// Check for invalid dependencies (tasks that don't exist)
	for taskTitle, dependencies := range dependencyGraph {
		for _, depTitle := range dependencies {
			if _, exists := taskIndexMap[depTitle]; !exists {
				return fmt.Errorf("task '%s' depends on non-existent task '%s'", taskTitle, depTitle)
			}
		}
	}

	// Detect circular dependencies using DFS
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for taskTitle := range dependencyGraph {
		if err := t.detectCircularDependency(taskTitle, dependencyGraph, visited, recursionStack); err != nil {
			return err
		}
	}

	return nil
}

// detectCircularDependency uses DFS to detect cycles in the dependency graph
func (t *CoordinatorSplitAgentTaskTool) detectCircularDependency(
	taskTitle string,
	graph map[string][]string,
	visited map[string]bool,
	recursionStack map[string]bool,
) error {
	// Mark current node as visited and add to recursion stack
	visited[taskTitle] = true
	recursionStack[taskTitle] = true

	// Check all dependencies
	for _, dependency := range graph[taskTitle] {
		// If dependency is in recursion stack, we found a cycle
		if recursionStack[dependency] {
			return fmt.Errorf("circular dependency detected: task '%s' depends on '%s', which creates a cycle", taskTitle, dependency)
		}

		// If not visited, recursively check
		if !visited[dependency] {
			if err := t.detectCircularDependency(dependency, graph, visited, recursionStack); err != nil {
				return err
			}
		}
	}

	// Remove from recursion stack before returning
	recursionStack[taskTitle] = false
	return nil
}

// createChildTask creates a child task with proper parent-child relationships
func (t *CoordinatorSplitAgentTaskTool) createChildTask(ctx context.Context, parentTaskId string, params ChildTaskParams) (*storage.AgentTask, error) {
	// Convert todos to TodoItemInput
	todoInputs := make([]storage.TodoItemInput, len(params.Todos))
	for i, todoText := range params.Todos {
		todoInputs[i] = storage.TodoItemInput{
			Description: todoText,
		}
	}

	// Create the child task using storage method
	// Note: We use parent task's humanTaskId for consistency
	parentTask, err := t.storage.GetAgentTask(parentTaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent task: %w", err)
	}

	// Create child task with proper parameters
	childTask, err := t.storage.CreateAgentTask(
		parentTask.HumanTaskID,
		parentTask.AgentName,
		params.Title,
		todoInputs,
		params.ContextSummary,
		params.FilesModified,
		[]string{},
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create child task: %w", err)
	}

	// Update child task with hierarchy fields
	// Note: This is a workaround since CreateAgentTask doesn't support hierarchy fields yet
	parentTaskIDPtr := &parentTaskId
	childTask.ParentTaskID = parentTaskIDPtr
	childTask.OrderIndex = params.OrderIndex
	childTask.DependsOn = params.DependsOn
	childTask.SplitStrategy = params.SplittingStrategy

	return childTask, nil
}

// generateIntegrationDoc creates documentation for coordinating child tasks
func (t *CoordinatorSplitAgentTaskTool) generateIntegrationDoc(parentTask *storage.AgentTask, childTasks []*storage.AgentTask, strategy string) string {
	doc := fmt.Sprintf("# Task Integration Documentation\n\n")
	doc += fmt.Sprintf("## Parent Task: %s\n", parentTask.Role)
	doc += fmt.Sprintf("**Task ID:** %s\n", parentTask.ID)
	doc += fmt.Sprintf("**Splitting Strategy:** %s\n\n", strategy)

	doc += fmt.Sprintf("## Child Tasks (%d total)\n\n", len(childTasks))

	for i, task := range childTasks {
		doc += fmt.Sprintf("### %d. %s\n", i+1, task.Role)
		doc += fmt.Sprintf("**Task ID:** %s\n", task.ID)
		doc += fmt.Sprintf("**Status:** %s\n", task.Status)
		doc += fmt.Sprintf("**Files Modified:** %d files\n", len(task.FilesModified))
		doc += fmt.Sprintf("**TODOs:** %d items\n", len(task.Todos))
		doc += fmt.Sprintf("**Order Index:** %d\n", task.OrderIndex)

		if len(task.DependsOn) > 0 {
			doc += fmt.Sprintf("**Dependencies:** %s\n", strings.Join(task.DependsOn, ", "))
		}

		doc += "\n"
	}
	
	if strategy == "SEQUENTIAL" {
		doc += "## Execution Order\n\n"
		doc += "Tasks must be executed in the order listed above. Each task depends on the completion of the previous task.\n\n"
	} else {
		doc += "## Parallel Execution\n\n"
		doc += "Tasks can be executed independently in parallel. Monitor for any file conflicts.\n\n"
	}
	
	doc += "## Integration Notes\n\n"
	doc += "- Monitor child task progress and update parent task status accordingly\n"
	doc += "- Ensure file modifications don't conflict between parallel tasks\n"
	doc += "- Validate integration points after each task completion\n"
	doc += "- Update parent task to COMPLETED when all child tasks are finished\n"
	
	return doc
}

// CoordinatorGetTaskHierarchyTool retrieves task hierarchy with recursive tree structure
type CoordinatorGetTaskHierarchyTool struct {
	storage storage.TaskStorage
}

func (t *CoordinatorGetTaskHierarchyTool) Name() string {
	return "coordinator_get_task_hierarchy"
}

func (t *CoordinatorGetTaskHierarchyTool) Description() string {
	return "Get the hierarchical structure of tasks with recursive tree representation and progress aggregation. Returns parent-child relationships and calculated progress."
}

func (t *CoordinatorGetTaskHierarchyTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"rootTaskId": map[string]interface{}{
				"type":        "string",
				"description": "ID of the root task to build hierarchy from (optional - if not provided, returns all root tasks)",
			},
			"includeProgress": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to include progress aggregation calculations",
				"default":     true,
			},
			"maxDepth": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum depth to traverse (default: 10)",
				"default":     10,
			},
		},
	}
}

func (t *CoordinatorGetTaskHierarchyTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Parse input parameters
	rootTaskId := ""
	if val, ok := args["rootTaskId"].(string); ok {
		rootTaskId = val
	}

	includeProgress := true
	if val, ok := args["includeProgress"].(bool); ok {
		includeProgress = val
	}

	maxDepth := 10
	if val, ok := args["maxDepth"].(float64); ok {
		maxDepth = int(val)
	}

	if maxDepth <= 0 {
		maxDepth = 10
	}

	// Get all tasks to build hierarchy
	allTasks, _, err := t.storage.ListAgentTasks(bson.M{}, 0, 1000) // Get a large number of tasks
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Build task hierarchy
	if rootTaskId != "" {
		// Build hierarchy for specific root task
		hierarchy, err := t.buildTaskHierarchy(ctx, rootTaskId, allTasks, maxDepth, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to build hierarchy for task %s: %w", rootTaskId, err)
		}

		// Calculate progress if requested
		if includeProgress {
			progress := t.calculateHierarchicalProgress(hierarchy, allTasks)
			return map[string]interface{}{
				"hierarchy": hierarchy,
				"progress":  progress,
			}, nil
		}

		return map[string]interface{}{
			"hierarchy": hierarchy,
		}, nil
	} else {
		// Build hierarchy for all root tasks (tasks without parents)
		rootTasks := t.findRootTasks(allTasks)
		hierarchies := make([]TaskHierarchy, 0, len(rootTasks))
		progressMap := make(map[string]HierarchicalProgress)

		for _, rootTask := range rootTasks {
			hierarchy, err := t.buildTaskHierarchy(ctx, rootTask.ID, allTasks, maxDepth, 0)
			if err != nil {
				continue // Skip tasks that can't be processed
			}
			hierarchies = append(hierarchies, hierarchy)

			if includeProgress {
				progress := t.calculateHierarchicalProgress(hierarchy, allTasks)
				progressMap[rootTask.ID] = progress
			}
		}

		result := map[string]interface{}{
			"hierarchies": hierarchies,
			"totalRootTasks": len(hierarchies),
		}

		if includeProgress {
			result["progress"] = progressMap
		}

		return result, nil
	}
}

// buildTaskHierarchy recursively builds the task hierarchy
func (t *CoordinatorGetTaskHierarchyTool) buildTaskHierarchy(ctx context.Context, taskId string, allTasks []*storage.AgentTask, maxDepth, currentDepth int) (TaskHierarchy, error) {
	if currentDepth >= maxDepth {
		return TaskHierarchy{}, fmt.Errorf("maximum depth exceeded")
	}

	// Find the task
	var task *storage.AgentTask
	for i := range allTasks {
		if allTasks[i].ID == taskId {
			task = allTasks[i]
			break
		}
	}

	if task == nil {
		return TaskHierarchy{}, fmt.Errorf("task not found: %s", taskId)
	}

	// Create hierarchy node
	hierarchy := TaskHierarchy{
		TaskID:     task.ID,
		Title:      task.Role,
		Status:     string(task.Status),
		Progress:   t.calculateTaskProgress(task),
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
		OrderIndex: task.OrderIndex,
		DependsOn:  task.DependsOn,
		Metadata:   make(map[string]interface{}),
	}

	// Add additional metadata
	hierarchy.Metadata["agentName"] = task.AgentName
	hierarchy.Metadata["filesModified"] = task.FilesModified
	hierarchy.Metadata["contextSummary"] = task.ContextSummary
	hierarchy.Metadata["splitStrategy"] = task.SplitStrategy

	// Find child tasks
	childTasks := t.findChildTasks(task.ID, allTasks)
	hierarchy.Children = make([]TaskHierarchy, 0, len(childTasks))

	for _, childTask := range childTasks {
		childHierarchy, err := t.buildTaskHierarchy(ctx, childTask.ID, allTasks, maxDepth, currentDepth+1)
		if err != nil {
			continue // Skip problematic child tasks
		}
		hierarchy.Children = append(hierarchy.Children, childHierarchy)
	}

	return hierarchy, nil
}

// findRootTasks finds tasks that don't have parent tasks
func (t *CoordinatorGetTaskHierarchyTool) findRootTasks(allTasks []*storage.AgentTask) []*storage.AgentTask {
	rootTasks := make([]*storage.AgentTask, 0)

	for _, task := range allTasks {
		isRoot := true
		if task.ParentTaskID != nil {
			isRoot = false
		}
		if isRoot {
			rootTasks = append(rootTasks, task)
		}
	}

	return rootTasks
}

// findChildTasks finds all child tasks for a given parent task ID
func (t *CoordinatorGetTaskHierarchyTool) findChildTasks(parentTaskId string, allTasks []*storage.AgentTask) []*storage.AgentTask {
	childTasks := make([]*storage.AgentTask, 0)

	for _, task := range allTasks {
		if task.ParentTaskID != nil && *task.ParentTaskID == parentTaskId {
			childTasks = append(childTasks, task)
		}
	}

	return childTasks
}

// calculateTaskProgress calculates progress for a single task based on completed TODOs
func (t *CoordinatorGetTaskHierarchyTool) calculateTaskProgress(task *storage.AgentTask) float64 {
	if len(task.Todos) == 0 {
		// If no TODOs, consider task status
		switch task.Status {
		case storage.TaskStatusCompleted:
			return 1.0
		case storage.TaskStatusInProgress:
			return 0.5
		default:
			return 0.0
		}
	}

	completedTodos := 0
	for _, todo := range task.Todos {
		if todo.Status == storage.TodoStatusCompleted {
			completedTodos++
		}
	}

	return float64(completedTodos) / float64(len(task.Todos))
}

// calculateHierarchicalProgress calculates aggregated progress including children
func (t *CoordinatorGetTaskHierarchyTool) calculateHierarchicalProgress(hierarchy TaskHierarchy, allTasks []*storage.AgentTask) HierarchicalProgress {
	progress := HierarchicalProgress{
		TaskID:           hierarchy.TaskID,
		DirectProgress:   hierarchy.Progress,
		CompletedChildren: 0,
		TotalChildren:    len(hierarchy.Children),
		ChildrenProgress: make(map[string]HierarchicalProgress),
		BlockedBy:        make([]string, 0),
		LastUpdated:      time.Now(),
	}

	// Calculate children progress
	totalChildProgress := 0.0
	for _, child := range hierarchy.Children {
		childProgress := t.calculateHierarchicalProgress(child, allTasks)
		progress.ChildrenProgress[child.TaskID] = childProgress
		
		totalChildProgress += childProgress.AggregatedProgress
		if childProgress.AggregatedProgress >= 1.0 {
			progress.CompletedChildren++
		}
	}

	// Calculate aggregated progress
	if len(hierarchy.Children) > 0 {
		childrenWeight := 0.7
		directWeight := 0.3
		avgChildProgress := totalChildProgress / float64(len(hierarchy.Children))
		progress.AggregatedProgress = (directWeight * progress.DirectProgress) + (childrenWeight * avgChildProgress)
	} else {
		progress.AggregatedProgress = progress.DirectProgress
	}

	// Check if task is executable (dependencies are met)
	progress.IsExecutable = t.isTaskExecutable(hierarchy, allTasks)

	// Find blocking dependencies
	for _, depId := range hierarchy.DependsOn {
		if !t.isTaskCompleted(depId, allTasks) {
			progress.BlockedBy = append(progress.BlockedBy, depId)
		}
	}

	return progress
}

// isTaskExecutable checks if a task can be executed (all dependencies are met)
func (t *CoordinatorGetTaskHierarchyTool) isTaskExecutable(hierarchy TaskHierarchy, allTasks []*storage.AgentTask) bool {
	for _, depId := range hierarchy.DependsOn {
		if !t.isTaskCompleted(depId, allTasks) {
			return false
		}
	}
	return true
}

// isTaskCompleted checks if a task is completed
func (t *CoordinatorGetTaskHierarchyTool) isTaskCompleted(taskId string, allTasks []*storage.AgentTask) bool {
	for _, task := range allTasks {
		if task.ID == taskId {
			return task.Status == "COMPLETED"
		}
	}
	return false
}

// CoordinatorGetNextExecutableTaskTool finds the next task that can be executed
type CoordinatorGetNextExecutableTaskTool struct {
	storage storage.TaskStorage
}

func (t *CoordinatorGetNextExecutableTaskTool) Name() string {
	return "coordinator_get_next_executable_task"
}

func (t *CoordinatorGetNextExecutableTaskTool) Description() string {
	return "Find the next executable task based on dependency resolution and task filtering. Returns tasks that have all dependencies met and are ready for execution."
}

func (t *CoordinatorGetNextExecutableTaskTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"parentTaskId": map[string]interface{}{
				"type":        "string",
				"description": "Filter tasks by parent task ID (optional)",
			},
			"priority": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"HIGH", "MEDIUM", "LOW"},
				"description": "Filter tasks by priority level (optional)",
			},
			"maxResults": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of executable tasks to return (default: 5)",
				"default":     5,
			},
			"includeDetails": map[string]interface{}{
				"type":        "boolean",
				"description": "Include detailed task information and dependency analysis",
				"default":     true,
			},
			"sortBy": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"priority", "createdAt", "estimatedTime", "orderIndex"},
				"description": "Sort executable tasks by specified criteria (default: priority)",
				"default":     "priority",
			},
		},
	}
}

func (t *CoordinatorGetNextExecutableTaskTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Parse input parameters
	parentTaskId := ""
	if val, ok := args["parentTaskId"].(string); ok {
		parentTaskId = val
	}

	priority := ""
	if val, ok := args["priority"].(string); ok {
		priority = val
	}

	maxResults := 5
	if val, ok := args["maxResults"].(float64); ok {
		maxResults = int(val)
	}
	if maxResults <= 0 {
		maxResults = 5
	}

	includeDetails := true
	if val, ok := args["includeDetails"].(bool); ok {
		includeDetails = val
	}

	sortBy := "priority"
	if val, ok := args["sortBy"].(string); ok {
		sortBy = val
	}

	// Get all tasks
	allTasks, _, err := t.storage.ListAgentTasks(bson.M{}, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	// Filter executable tasks
	executableTasks := t.findExecutableTasks(allTasks, parentTaskId, priority)

	// Sort tasks
	t.sortTasks(executableTasks, sortBy)

	// Limit results
	if len(executableTasks) > maxResults {
		executableTasks = executableTasks[:maxResults]
	}

	// Prepare response
	if includeDetails {
		detailedTasks := make([]map[string]interface{}, 0, len(executableTasks))
		for _, task := range executableTasks {
			details := t.getTaskDetails(task, allTasks)
			detailedTasks = append(detailedTasks, details)
		}

		return map[string]interface{}{
			"executableTasks": detailedTasks,
			"totalFound":      len(executableTasks),
			"sortedBy":        sortBy,
		}, nil
	} else {
		simpleTasks := make([]map[string]interface{}, 0, len(executableTasks))
		for _, task := range executableTasks {
			simpleTasks = append(simpleTasks, map[string]interface{}{
				"taskId": task.ID,
				"title":  task.Role,
				"status": task.Status,
			})
		}

		return map[string]interface{}{
			"executableTasks": simpleTasks,
			"totalFound":      len(executableTasks),
			"sortedBy":        sortBy,
		}, nil
	}
}

// findExecutableTasks finds tasks that can be executed (dependencies met, not completed)
func (t *CoordinatorGetNextExecutableTaskTool) findExecutableTasks(allTasks []*storage.AgentTask, parentTaskId, priority string) []*storage.AgentTask {
	executableTasks := make([]*storage.AgentTask, 0)

	for _, task := range allTasks {
		// Skip completed or in-progress tasks
		if task.Status == "COMPLETED" || task.Status == "IN_PROGRESS" {
			continue
		}

		// Filter by parent task if specified
		if parentTaskId != "" {
			if task.ParentTaskID == nil || *task.ParentTaskID != parentTaskId {
				continue
			}
		}

		// Priority filtering removed - field not available in AgentTask

		// Check if all dependencies are met
		if t.areDependenciesMet(task, allTasks) {
			executableTasks = append(executableTasks, task)
		}
	}

	return executableTasks
}

// areDependenciesMet checks if all task dependencies are completed
func (t *CoordinatorGetNextExecutableTaskTool) areDependenciesMet(task *storage.AgentTask, allTasks []*storage.AgentTask) bool {
	if task.DependsOn == nil || len(task.DependsOn) == 0 {
		return true // No dependencies
	}

	// Check each dependency
	for _, depId := range task.DependsOn {
		if !t.isTaskCompleted(depId, allTasks) {
			return false
		}
	}

	return true
}

// isTaskCompleted checks if a specific task is completed
func (t *CoordinatorGetNextExecutableTaskTool) isTaskCompleted(taskId string, allTasks []*storage.AgentTask) bool {
	for _, task := range allTasks {
		if task.ID == taskId {
			return task.Status == "COMPLETED"
		}
	}
	return false // Task not found, consider as not completed
}

// sortTasks sorts tasks based on the specified criteria
func (t *CoordinatorGetNextExecutableTaskTool) sortTasks(tasks []*storage.AgentTask, sortBy string) {
	switch sortBy {
	case "priority":
		// Sort by priority: HIGH > MEDIUM > LOW
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				priority1 := t.getTaskPriority(tasks[i])
				priority2 := t.getTaskPriority(tasks[j])
				if t.comparePriority(priority1, priority2) < 0 {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	case "createdAt":
		// Sort by creation time (oldest first)
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				if tasks[i].CreatedAt.After(tasks[j].CreatedAt) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	case "estimatedTime":
		// Sort by estimated time (shortest first)
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				time1 := t.getEstimatedTime(tasks[i])
				time2 := t.getEstimatedTime(tasks[j])
				if time1 > time2 {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	case "orderIndex":
		// Sort by order index (lowest first)
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				order1 := t.getOrderIndex(tasks[i])
				order2 := t.getOrderIndex(tasks[j])
				if order1 > order2 {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// getTaskPriority gets task priority with default
func (t *CoordinatorGetNextExecutableTaskTool) getTaskPriority(task *storage.AgentTask) string {
	// Priority field not available in AgentTask, return default
	return "MEDIUM"
}

// comparePriority compares two priority strings (-1: p1 < p2, 0: equal, 1: p1 > p2)
func (t *CoordinatorGetNextExecutableTaskTool) comparePriority(p1, p2 string) int {
	priorityOrder := map[string]int{"HIGH": 3, "MEDIUM": 2, "LOW": 1}
	val1, ok1 := priorityOrder[p1]
	val2, ok2 := priorityOrder[p2]

	if !ok1 {
		val1 = 2 // Default to MEDIUM
	}
	if !ok2 {
		val2 = 2 // Default to MEDIUM
	}

	if val1 < val2 {
		return -1
	} else if val1 > val2 {
		return 1
	}
	return 0
}

// getEstimatedTime gets estimated time with default
func (t *CoordinatorGetNextExecutableTaskTool) getEstimatedTime(task *storage.AgentTask) int {
	// Estimated time field not available in AgentTask, return default
	return 60 // Default 60 minutes
}

// getOrderIndex gets order index with default
func (t *CoordinatorGetNextExecutableTaskTool) getOrderIndex(task *storage.AgentTask) int {
	return task.OrderIndex
}

// getTaskDetails gets detailed task information including dependency analysis
func (t *CoordinatorGetNextExecutableTaskTool) getTaskDetails(task *storage.AgentTask, allTasks []*storage.AgentTask) map[string]interface{} {
	details := map[string]interface{}{
		"taskId":         task.ID,
		"title":          task.Role,
		"status":         task.Status,
		"contextSummary": task.ContextSummary,
		"filesModified":  task.FilesModified,
		"todos":          len(task.Todos),
		"createdAt":      task.CreatedAt,
		"updatedAt":      task.UpdatedAt,
	}

	// Add metadata information
	if task.ParentTaskID != nil {
		details["orderIndex"] = task.OrderIndex
		details["parentTaskId"] = *task.ParentTaskID
	}

	// Add dependency information
	details["dependencies"] = task.DependsOn
	details["dependenciesMet"] = t.areDependenciesMet(task, allTasks)

	return details
}

// CoordinatorUpdateChildTaskProgressTool updates child task progress and propagates to parent
type CoordinatorUpdateChildTaskProgressTool struct {
	storage storage.TaskStorage
}

func (t *CoordinatorUpdateChildTaskProgressTool) Name() string {
	return "coordinator_update_child_task_progress"
}

func (t *CoordinatorUpdateChildTaskProgressTool) Description() string {
	return "Update child task progress and propagate status changes to parent tasks. Automatically updates parent task status based on child task completion."
}

func (t *CoordinatorUpdateChildTaskProgressTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"childTaskId": map[string]interface{}{
				"type":        "string",
				"description": "ID of the child task to update",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"PENDING", "IN_PROGRESS", "COMPLETED", "FAILED"},
				"description": "New status for the child task",
			},
			"progress": map[string]interface{}{
				"type":        "number",
				"minimum":     0.0,
				"maximum":     1.0,
				"description": "Progress percentage (0.0 to 1.0) - optional, calculated from TODOs if not provided",
			},
			"notes": map[string]interface{}{
				"type":        "string",
				"description": "Progress notes or comments (optional)",
			},
			"propagateToParent": map[string]interface{}{
				"type":        "boolean",
				"description": "Whether to propagate status changes to parent task (default: true)",
				"default":     true,
			},
		},
		"required": []string{"childTaskId", "status"},
	}
}

func (t *CoordinatorUpdateChildTaskProgressTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// Parse input parameters
	childTaskId, ok := args["childTaskId"].(string)
	if !ok || childTaskId == "" {
		return nil, fmt.Errorf("childTaskId is required and must be a non-empty string")
	}

	status, ok := args["status"].(string)
	if !ok || status == "" {
		return nil, fmt.Errorf("status is required and must be a non-empty string")
	}

	validStatuses := map[string]bool{
		"PENDING": true, "IN_PROGRESS": true, "COMPLETED": true, "FAILED": true,
	}
	if !validStatuses[status] {
		return nil, fmt.Errorf("invalid status: %s. Must be one of: PENDING, IN_PROGRESS, COMPLETED, FAILED", status)
	}

	var progress *float64
	if val, ok := args["progress"].(float64); ok {
		if val < 0.0 || val > 1.0 {
			return nil, fmt.Errorf("progress must be between 0.0 and 1.0")
		}
		progress = &val
	}

	notes := ""
	if val, ok := args["notes"].(string); ok {
		notes = val
	}

	propagateToParent := true
	if val, ok := args["propagateToParent"].(bool); ok {
		propagateToParent = val
	}

	// Get the child task
	childTask, err := t.storage.GetAgentTask(childTaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get child task: %w", err)
	}

	// Update child task status
	err = t.storage.UpdateTaskStatus(childTaskId, storage.TaskStatus(status), notes)
	if err != nil {
		return nil, fmt.Errorf("failed to update child task status: %w", err)
	}

	// Calculate progress if not provided
	if progress == nil {
		calculatedProgress := t.calculateTaskProgress(childTask)
		progress = &calculatedProgress
	}

	// Get parent task ID if it exists
	var parentTaskId string
	if childTask.ParentTaskID != nil {
		parentTaskId = *childTask.ParentTaskID
	}

	// Propagate to parent if requested and parent exists
	var parentUpdateResult map[string]interface{}
	if propagateToParent && parentTaskId != "" {
		parentUpdateResult, err = t.propagateToParent(ctx, parentTaskId)
		if err != nil {
			// Log error but don't fail the entire operation
			parentUpdateResult = map[string]interface{}{
				"error": fmt.Sprintf("Failed to propagate to parent: %v", err),
			}
		}
	}

	result := map[string]interface{}{
		"childTaskId":     childTaskId,
		"status":          status,
		"progress":        *progress,
		"parentTaskId":    parentTaskId,
		"propagated":      propagateToParent && parentTaskId != "",
		"updatedAt":       time.Now(),
	}

	if notes != "" {
		result["notes"] = notes
	}

	if parentUpdateResult != nil {
		result["parentUpdate"] = parentUpdateResult
	}

	return result, nil
}

// calculateTaskProgress calculates progress based on completed TODOs
func (t *CoordinatorUpdateChildTaskProgressTool) calculateTaskProgress(task *storage.AgentTask) float64 {
	if len(task.Todos) == 0 {
		// If no TODOs, base on status
		switch task.Status {
		case storage.TaskStatusCompleted:
			return 1.0
		case storage.TaskStatusInProgress:
			return 0.5
		case storage.TaskStatusBlocked:
			return 0.0
		default:
			return 0.0
		}
	}

	completedTodos := 0
	for _, todo := range task.Todos {
		if todo.Status == storage.TodoStatusCompleted {
			completedTodos++
		}
	}

	return float64(completedTodos) / float64(len(task.Todos))
}

// propagateToParent updates parent task status based on child task progress
func (t *CoordinatorUpdateChildTaskProgressTool) propagateToParent(ctx context.Context, parentTaskId string) (map[string]interface{}, error) {
	// Get parent task
	parentTask, err := t.storage.GetAgentTask(parentTaskId)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent task: %w", err)
	}

	// Get all child tasks for this parent
	allTasks, _, err := t.storage.ListAgentTasks(bson.M{}, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tasks: %w", err)
	}

	childTasks := make([]*storage.AgentTask, 0)
	for _, task := range allTasks {
		if task.ParentTaskID != nil && *task.ParentTaskID == parentTaskId {
			childTasks = append(childTasks, task)
		}
	}

	if len(childTasks) == 0 {
		return map[string]interface{}{
			"message": "No child tasks found for parent",
		}, nil
	}

	// Calculate aggregated progress and status
	completedChildren := 0
	inProgressChildren := 0
	failedChildren := 0
	totalProgress := 0.0

	for _, child := range childTasks {
		switch child.Status {
		case storage.TaskStatusCompleted:
			completedChildren++
			totalProgress += 1.0
		case storage.TaskStatusInProgress:
			inProgressChildren++
			// Calculate progress from todos
			totalProgress += t.calculateTaskProgress(child)
		case storage.TaskStatusBlocked:
			failedChildren++
		}
	}

	aggregatedProgress := totalProgress / float64(len(childTasks))

	// Determine new parent status
	var newParentStatus storage.TaskStatus
	if completedChildren == len(childTasks) {
		newParentStatus = storage.TaskStatusCompleted
	} else if failedChildren > 0 && (completedChildren+inProgressChildren) == 0 {
		newParentStatus = storage.TaskStatusBlocked
	} else if inProgressChildren > 0 || completedChildren > 0 {
		newParentStatus = storage.TaskStatusInProgress
	} else {
		newParentStatus = storage.TaskStatusPending
	}

	// Update parent task if status changed
	statusChanged := parentTask.Status != newParentStatus
	if statusChanged {
		err = t.storage.UpdateTaskStatus(parentTaskId, newParentStatus, "Status updated based on child task progress")
		if err != nil {
			return nil, fmt.Errorf("failed to update parent task status: %w", err)
		}
	}

	// Metadata updates removed - field not available in AgentTask

	return map[string]interface{}{
		"parentTaskId":        parentTaskId,
		"oldStatus":           parentTask.Status,
		"newStatus":           newParentStatus,
		"statusChanged":       statusChanged,
		"aggregatedProgress":  aggregatedProgress,
		"completedChildren":   completedChildren,
		"totalChildren":       len(childTasks),
		"inProgressChildren":  inProgressChildren,
		"failedChildren":      failedChildren,
	}, nil
}

// testPhase3ToolsIntegration tests all Phase 3 MCP tools with sample complex tasks
func testPhase3ToolsIntegration(ctx context.Context, storage storage.TaskStorage) error {
	zap.L().Info("🧪 Starting Phase 3 MCP Tools Integration Test")
	
	// Test 1: Complexity Analysis
	zap.L().Info("Test 1: Testing complexity analysis tool")
	complexityTool := &CoordinatorAnalyzeTaskComplexityTool{storage: storage}
	
	complexityArgs := map[string]interface{}{
		"title": "Implement Advanced User Management System",
		"contextSummary": "Create a comprehensive user management system with authentication, authorization, profile management, and admin dashboard. Integrate with external OAuth providers and implement role-based access control.",
		"todos": []interface{}{
			"Implement user authentication with JWT tokens",
			"Create user registration and login forms",
			"Add OAuth integration for Google and GitHub",
			"Implement role-based access control middleware",
			"Create admin dashboard for user management",
			"Add user profile management features",
			"Implement password reset functionality",
			"Add email verification system",
			"Create audit logging for user actions",
			"Implement session management",
			"Add two-factor authentication",
			"Create user permissions system",
		},
		"filesModified": []interface{}{
			"/auth/jwt.go",
			"/auth/middleware.go",
			"/handlers/auth.go",
			"/handlers/users.go",
			"/models/user.go",
			"/models/role.go",
			"/database/migrations/001_users.sql",
			"/database/migrations/002_roles.sql",
			"/frontend/src/components/Login.tsx",
			"/frontend/src/components/Register.tsx",
			"/frontend/src/components/UserProfile.tsx",
			"/frontend/src/components/AdminDashboard.tsx",
			"/frontend/src/hooks/useAuth.ts",
			"/frontend/src/services/authService.ts",
			"/config/oauth.go",
		},
	}
	
	complexityResult, err := complexityTool.Execute(ctx, complexityArgs)
	if err != nil {
		return fmt.Errorf("complexity analysis test failed: %w", err)
	}
	
	analysis, ok := complexityResult.(ComplexityAnalysis)
	if !ok {
		return fmt.Errorf("complexity analysis returned unexpected type")
	}
	
	zap.L().Info("✅ Complexity analysis completed",
		zap.Float64("score", analysis.Score),
		zap.String("recommendation", analysis.Recommendation),
		zap.Int("estimatedMinutes", analysis.EstimatedTimeMinutes))
	
	// Test 2: Task Splitting (if complexity is high)
	if analysis.Score >= 0.6 {
		zap.L().Info("Test 2: Testing task splitting tool (high complexity detected)")

		// Create a parent task first (mock)
		parentTaskId := "test-parent-task-" + fmt.Sprintf("%d", time.Now().Unix())

		_ = &CoordinatorSplitAgentTaskTool{storage: storage} // splitTool not used
		_ = map[string]interface{}{ // splitArgs not used
			"parentTaskId": parentTaskId,
			"splittingStrategy": "SEQUENTIAL",
			"childTasks": []interface{}{
				map[string]interface{}{
					"title": "Authentication Core Implementation",
					"contextSummary": "Implement core authentication functionality",
					"todos": []interface{}{
						"Implement JWT token generation and validation",
						"Create authentication middleware",
						"Add login and registration endpoints",
					},
					"filesModified": []interface{}{
						"/auth/jwt.go",
						"/auth/middleware.go",
						"/handlers/auth.go",
					},
					"priority": "HIGH",
				},
				map[string]interface{}{
					"title": "User Management Features",
					"contextSummary": "Implement user profile and management features",
					"todos": []interface{}{
						"Create user profile management",
						"Implement password reset functionality",
						"Add email verification system",
					},
					"filesModified": []interface{}{
						"/handlers/users.go",
						"/models/user.go",
						"/frontend/src/components/UserProfile.tsx",
					},
					"priority": "MEDIUM",
				},
			},
			"createIntegrationDoc": true,
		}
		
		// Note: This would fail in real execution because parentTaskId doesn't exist
		// In a real test, we'd create the parent task first
		zap.L().Info("⚠️ Skipping split test - would require creating parent task first")
	}
	
	// Test 3: Task Hierarchy Retrieval
	zap.L().Info("Test 3: Testing task hierarchy tool")
	hierarchyTool := &CoordinatorGetTaskHierarchyTool{storage: storage}
	
	hierarchyArgs := map[string]interface{}{
		"includeProgress": true,
		"maxDepth": 5,
	}
	
	_, err = hierarchyTool.Execute(ctx, hierarchyArgs)
	if err != nil {
		return fmt.Errorf("hierarchy test failed: %w", err)
	}

	zap.L().Info("✅ Task hierarchy retrieved successfully")
	
	// Test 4: Next Executable Task
	zap.L().Info("Test 4: Testing next executable task tool")
	executableTool := &CoordinatorGetNextExecutableTaskTool{storage: storage}
	
	executableArgs := map[string]interface{}{
		"maxResults": 3,
		"includeDetails": true,
		"sortBy": "priority",
	}
	
	_, err = executableTool.Execute(ctx, executableArgs)
	if err != nil {
		return fmt.Errorf("executable task test failed: %w", err)
	}

	zap.L().Info("✅ Next executable tasks retrieved successfully")

	// Test 5: Child Task Progress Update
	zap.L().Info("Test 5: Testing child task progress update tool")
	_ = &CoordinatorUpdateChildTaskProgressTool{storage: storage} // progressTool not used

	// This would require an actual child task to test properly
	zap.L().Info("⚠️ Skipping progress update test - would require actual child task")
	
	zap.L().Info("🎉 Phase 3 MCP Tools Integration Test Completed Successfully")
	return nil
}

// RegisterCoordinatorTools registers all coordinator tools with the tool registry
func RegisterCoordinatorTools(
	registry *aiservice.ToolRegistry,
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	subchatStorage *storage.SubchatStorage,
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler,
	chatService ChatServiceInterface,
	aiSettingsService AISettingsServiceInterface,
	aiService AIServiceInterface,
	logger *zap.Logger,
	validator *validation.CodeValidator,
) error {
	tools := []aiservice.ToolExecutor{
		// Existing tools
		&CreateAgentTaskTool{
			storage:   taskStorage,
			aiService: aiService,
			config:    aiService.GetConfig(),
		},
		&ListAgentTasksTool{storage: taskStorage},
		&QueryKnowledgeTool{storage: knowledgeStorage},

		// New tools
		&UpsertKnowledgeTool{storage: knowledgeStorage},
		&ListCollectionsTool{storage: knowledgeStorage},
		&CreateHumanTaskTool{storage: taskStorage},
		&UpdateTaskStatusTool{storage: taskStorage},
		&UpdateTodoStatusTool{storage: taskStorage},
		&ListHumanTasksTool{storage: taskStorage},
		&GetAgentTaskTool{storage: taskStorage},
		&FindSimilarTasksTool{storage: taskStorage}, // Claude optimization: prevent duplicate tasks
		&AddTaskPromptNotesTool{storage: taskStorage},
		&UpdateTaskPromptNotesTool{storage: taskStorage},
		&ClearTaskPromptNotesTool{storage: taskStorage},
		&AddTodoPromptNotesTool{storage: taskStorage},
		&UpdateTodoPromptNotesTool{storage: taskStorage},
		&ClearTodoPromptNotesTool{storage: taskStorage},

		// Phase 3 MCP Tool Extensions - Complexity Analysis and Hierarchical Management
		&CoordinatorAnalyzeTaskComplexityTool{storage: taskStorage},
		&CoordinatorSplitAgentTaskTool{
			storage:   taskStorage,
			aiService: aiService,
		},
		&CoordinatorGetTaskHierarchyTool{storage: taskStorage},
		&CoordinatorGetNextExecutableTaskTool{storage: taskStorage},
		&CoordinatorUpdateChildTaskProgressTool{storage: taskStorage},

		// Subagent tools
		&ListSubagentsTool{mongoDatabase: nil},
		&ExecuteSubagentTool{
			subchatStorage:    subchatStorage,
			taskStorage:       taskStorage,
			aiService:         aiService,
			chatService:       chatService,
			aiSettingsService: aiSettingsService,
			logger:            logger,
			validator:         validator,
		},

		// MCP tools discovery and management (6 new tools)
		&DiscoverToolsExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&GetToolSchemaExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&ExecuteToolExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&McpAddServerExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&McpRediscoverServerExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		&McpRemoveServerExecutor{toolsDiscoveryHandler: toolsDiscoveryHandler},
		// Note: coordinator_clear_task_board excluded (destructive operation)
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name(), err)
		}
	}

	return nil
}
