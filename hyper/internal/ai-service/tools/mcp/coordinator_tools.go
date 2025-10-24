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
	"hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"
	"hyper/internal/models"

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

		// Check if path exists as-is
		if _, err := os.Stat(path); err == nil {
			correctedPaths = append(correctedPaths, path)
			logger.Debug("✅ Path valid", zap.String("path", path))
			continue
		}

		// Path doesn't exist, try to fix it
		logger.Warn("⚠️  Path does not exist, attempting correction",
			zap.String("originalPath", path))

		fixedPath := tryFixPath(path, projectRoot, logger)

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

		// Strategy 4: Try finding with bash find command (last resort)
		if foundPath == "" {
			logger.Debug("Searching for pattern file using find command",
				zap.String("filename", filename))
			// Use find to locate the file
			findCmd := fmt.Sprintf("find %s -name %s -type f 2>/dev/null | head -1", projectRoot, filename)
			output, err := exec.Command("bash", "-c", findCmd).Output()
			if err == nil && len(output) > 0 {
				foundPath = strings.TrimSpace(string(output))
				if _, err := os.Stat(foundPath); err != nil {
					foundPath = "" // Invalid path
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
	storage storage.TaskStorage
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
	// SMART AUTO-FETCH: Extract humanTaskId from input but ALWAYS validate it exists in database
	// If invalid/missing, auto-fetch the latest pending task (don't trust the model)
	providedTaskID, _ := input["humanTaskId"].(string)

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

	// If task ID was provided, validate it exists in database
	if providedTaskID != "" {
		task, err := t.storage.GetHumanTask(providedTaskID)
		if err != nil || task == nil {
			// INVALID TASK ID - Model hallucinated or sent wrong ID
			zap.L().Warn("Model provided invalid humanTaskId - auto-fetching latest task",
				zap.String("providedTaskId", providedTaskID),
				zap.String("error", fmt.Sprintf("%v", err)),
				zap.String("reason", "task_not_found_in_database"))

			// Auto-fetch latest task
			latestTask, fetchErr := fetchLatestTask()
			if fetchErr != nil {
				return nil, fetchErr
			}
			humanTaskID = latestTask.ID

			zap.L().Info("Auto-corrected humanTaskId (model sent invalid ID)",
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

	role, ok := input["role"].(string)
	if !ok || role == "" {
		return nil, fmt.Errorf("role is required and must be a string")
	}

	todosRaw, ok := input["todos"]
	if !ok {
		return nil, fmt.Errorf("todos is required")
	}

	// Convert todos to []string
	var todos []string
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
						"✅ GOOD TODO examples:\n"+
						"  - 'Add responsive CSS to Settings.tsx lines 15-45'\n"+
						"  - 'Update login validation in AuthForm.tsx'\n"+
						"  - 'Test changes work on mobile viewport'\n\n"+
						"❌ BAD TODO examples:\n"+
						"  - 'Search for Settings component'  ← Discovery step!\n"+
						"  - 'Find the auth logic'  ← Discovery step!\n"+
						"  - 'Locate CSS files'  ← Discovery step!",
					i+1, keyword, todo)
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
	contextSummary, _ := input["contextSummary"].(string)
	priorWorkSummary, _ := input["priorWorkSummary"].(string)

	var filesModified []string
	if fm, ok := input["filesModified"].([]interface{}); ok {
		filesModified = make([]string, len(fm))
		for i, f := range fm {
			if str, ok := f.(string); ok {
				filesModified[i] = str
			}
		}
	}

	// AUTO-POPULATE: If filesModified is empty, try to populate from last code_index_search
	if len(filesModified) == 0 {
		cachedPaths := GetLastCodeSearchPaths()
		if len(cachedPaths) > 0 {
			filesModified = cachedPaths
			zap.L().Info("✅ Auto-populated filesModified from code_index_search cache",
				zap.Int("filesCount", len(filesModified)),
				zap.Strings("files", filesModified))
		}
	}

	// PATH CORRECTION: Fix invalid paths before validation (defensive programming)
	// Only runs if paths exist but some are invalid
	if len(filesModified) > 0 {
		correctedPaths, unfixablePaths, isIndexingIssue := correctFilePaths(filesModified, zap.L())
		if len(unfixablePaths) > 0 {
			// Some paths could not be fixed - this will fail validation
			zap.L().Error("❌ Path correction failed for some files",
				zap.Strings("unfixablePaths", unfixablePaths),
				zap.Bool("indexingIssue", isIndexingIssue))
			// Don't return error here - let validation handle it with proper error message
		} else if len(correctedPaths) != len(filesModified) {
			// Paths were corrected successfully
			filesModified = correctedPaths
			zap.L().Info("✅ File paths corrected and validated",
				zap.Int("correctedCount", len(correctedPaths)),
				zap.Bool("indexingIssue", isIndexingIssue))
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

	// VALIDATION: Warn if filesModified is empty
	// This is a strong indicator the coordinator didn't use code_index_search results
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
					return nil, fmt.Errorf(
						"❌ filesModified validation failed:\n"+
							"• filesModified is empty\n"+
							"• BUT TODO #%d references a file: %s\n\n"+
							"🚨 YOU MUST POPULATE filesModified\n"+
							"• Run code_index_search to find relevant files\n"+
							"• Extract filePath values from search results\n"+
							"• Pass them in filesModified array\n\n"+
							"Example:\n"+
							"1. code_index_search('settings component')\n"+
							"2. create_agent_task({\n"+
							"     filesModified: [\"/path/to/Settings.tsx\", \"/path/to/settings.css\"],\n"+
							"     todos: [\"Add responsive CSS...\"]\n"+
							"   })",
						i+1, todo)
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
			return nil, fmt.Errorf("file validation failed: the following files do not exist:\n%s\n\nPlease verify the file paths from code_index_search results and ensure they are copied exactly", strings.Join(missingFiles, "\n"))
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
	}, nil
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
	results, err := t.storage.Query(collection, query, limit)
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
	return "Store knowledge in the coordinator knowledge base. Use for storing task context, ADRs, data contracts, and coordination information. Returns entry ID and collection."
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
				"description": "Content to store",
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

	var metadata map[string]interface{}
	if m, ok := input["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	entry, err := t.storage.Upsert(collection, text, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert knowledge: %w", err)
	}

	return map[string]interface{}{
		"id":         entry.ID,
		"collection": entry.Collection,
		"createdAt":  entry.CreatedAt,
	}, nil
}

// GetPopularCollectionsTool implements the ToolExecutor interface
type GetPopularCollectionsTool struct {
	storage storage.KnowledgeStorage
}

func (t *GetPopularCollectionsTool) Name() string {
	return "coordinator_get_popular_collections"
}

func (t *GetPopularCollectionsTool) Description() string {
	return "Get top N knowledge collections by entry count. Use for discovering which collections contain the most knowledge. Returns collection names with entry counts."
}

func (t *GetPopularCollectionsTool) InputSchema() map[string]interface{} {
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

func (t *GetPopularCollectionsTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
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
	if fc, ok := input["forceCreate"].(bool); ok {
		forceCreate = fc
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

	err := t.storage.UpdateTaskStatus(taskID, status, notes)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	return map[string]interface{}{
		"taskId": taskID,
		"status": status,
		"notes":  notes,
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
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
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
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
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
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
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
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
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
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
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
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler
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
		return nil, fmt.Errorf("agentTaskId is required and must be a string")
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
			return nil, fmt.Errorf("parentChatId could not be determined: not in context and not provided by AI (or AI provided 'main' placeholder)")
		}
	}

	t.logger.Info("🚀 execute_subagent tool called",
		zap.String("agentTaskId", agentTaskID),
		zap.String("parentChatId", parentChatID))

	// Get the agent task to extract subagent name and details
	agentTask, err := t.taskStorage.GetAgentTask(agentTaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent task: %w", err)
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

	// Spawn background goroutine to execute the subagent
	go t.executeSubagentInBackground(subchat.ID, agentTask, parentChatID)

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
		// For apply_patch, the file path is embedded in the patch content
		// Format: "*** Update File: path/to/file.ext"
		if patchContent, ok := args["patch"].(string); ok {
			// Extract file path from patch
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
				// Fallback: mark as generic write if we can't extract path
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

	return true, matchedFiles, nil
}

// executeSubagentInBackground runs the subagent AI streaming in a background goroutine
func (t *ExecuteSubagentTool) executeSubagentInBackground(subchatID string, agentTask *storage.AgentTask, parentChatID string) {
	// Create a new background context with generous timeout for long-running tasks
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

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

	// Get parent session to inherit userID and companyID
	parentSession, err := t.chatService.GetSession(ctx, parentSessionID, "dev-company")
	if err != nil {
		t.logger.Error("Failed to get parent chat session",
			zap.String("parentChatId", parentChatID),
			zap.Error(err))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Failed to get parent session: %v", err))
		return
	}

	// Use parent session's userID and companyID for subchat
	userID := parentSession.UserID
	companyID := parentSession.CompanyID
	sessionTitle := fmt.Sprintf("Subchat: %s - %s", agentTask.AgentName, agentTask.Role)

	t.logger.Info("Creating subchat session with parent's credentials",
		zap.String("subchatId", subchatID),
		zap.String("parentChatId", parentChatID),
		zap.String("userId", userID),
		zap.String("companyId", companyID))

	chatSession, err := t.chatService.CreateSessionWithParent(ctx, userID, companyID, sessionTitle, &parentSessionID)
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
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "user", taskPrompt, companyID)
	if err != nil {
		t.logger.Warn("Failed to save initial user message",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		// Continue execution even if message save fails
	}

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

	for event := range aiStream {
		select {
		case <-ctx.Done():
			t.logger.Warn("⏱️ Subagent execution cancelled by timeout",
				zap.String("subchatId", subchatID))
			t.handleExecutionFailure(agentTask.ID, "Execution timeout")
			return
		default:
			switch event.Type {
			case aiservice.StreamEventToken:
				fullResponse += event.Content

			case aiservice.StreamEventToolCall:
				toolCallCount++

				// 📊 COMPREHENSIVE LOGGING: Log every tool call with timestamp and details
				t.logger.Info("🔧 TOOL CALL",
					zap.String("subchatId", subchatID),
					zap.Int("callNumber", toolCallCount),
					zap.String("toolName", event.ToolCall.Name),
					zap.Any("args", event.ToolCall.Args),
					zap.Time("timestamp", time.Now()))

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
				_, err := t.chatService.SaveToolCall(ctx, chatSession.ID, event.ToolCall.ID, event.ToolCall.Name, event.ToolCall.Args, companyID)
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

					// Save scaffold as visible message
					_, err := t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", forceScaffold, companyID)
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

				_, err := t.chatService.SaveToolResult(ctx, chatSession.ID, event.ToolResult.ID, event.ToolResult.Name, summarizedOutput, event.ToolResult.Error, event.ToolResult.DurationMs, companyID)
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

	t.logger.Info("📝 Saving final AI response to subchat",
		zap.String("subchatId", subchatID),
		zap.Int("responseLength", len(fullResponse)))

	// Save final AI response to subchat
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", fullResponse, companyID)
	if err != nil {
		t.logger.Error("Failed to save subagent final response",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

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
}

// buildExecutionPhaseSystemPrompt creates a strict system prompt using OPERATIONAL enforcement language
// Uses concrete "WRITE-ONLY MODE" instead of abstract "PHASE: EXECUTE" for better model compliance
func (t *ExecuteSubagentTool) buildExecutionPhaseSystemPrompt() string {
	return `╔══════════════════════════════════════════════════════════════╗
║                  WRITE-ONLY MODE ACTIVATED                    ║
╚══════════════════════════════════════════════════════════════╝

🚨 YOU ARE NOW IN WRITE-ONLY MODE 🚨

ALLOWED TOOLS (you may ONLY use these):
✅ read_file       - Read source file ONCE per file
✅ write_file      - Write/create files
✅ apply_patch     - Apply code changes
✅ bash            - Run commands/tests
✅ coordinator_update_todo_status - Mark TODO complete
✅ coordinator_upsert_knowledge   - Save decisions

BLOCKED TOOLS (these will FAIL):
❌ code_index_search - Discovery disabled in WRITE-ONLY MODE
❌ list_directory    - Discovery disabled in WRITE-ONLY MODE
❌ All coordinator tools (for task creation, listing, etc.)

═══════════════════════════════════════════════════════════════

⏱️ MANDATORY WORKFLOW - YOU MUST COMPLETE WITHIN 3 STEPS PER TODO:

Step 1: read_file on target file (ONCE ONLY)
Step 2: write_file or apply_patch to implement changes
Step 3: coordinator_update_todo_status to mark TODO completed

REPEAT for next TODO.

═══════════════════════════════════════════════════════════════

🎯 EXECUTION SCORING (visible after each tool call):

+20 points: write_file or apply_patch executed
+10 points: coordinator_update_todo_status (completed)
 +5 points: read_file (first read of a file)
 -5 points: read_file (duplicate read of same file)
-10 points: calling same tool twice with identical args
-20 points: exceeding 1 read without a write
-50 points: exceeding 3 reads without a write (HARD LIMIT)

Target score: +30 per TODO (read once, write once, update status)

═══════════════════════════════════════════════════════════════

⚠️ ENFORCEMENT RULES (RUNTIME - NOT SUGGESTIONS):

0. READ ONLY FILES SPECIFIED IN TASK
   • Task specifies EXACT file paths to modify
   • Do NOT explore, search, or read other files
   • Do NOT call list_directory - IT IS BLOCKED
   • Use the exact paths provided in filesModified

1. File content is CACHED after first read_file
   • Subsequent read_file on same file returns cached summary
   • You will NOT receive full content again - implement now

2. After 1 read_file call without write:
   • You receive a FORCED WRITE SCAFFOLD
   • You MUST complete the scaffold and submit immediately

3. You MAY NOT read any file more than ONCE
   • Second read returns: "CACHED - use previous content"
   • This forces you to implement, not re-read

4. Maximum 3 tool calls per TODO:
   • Call 1: read_file
   • Call 2: write_file/apply_patch
   • Call 3: coordinator_update_todo_status
   • Extra calls trigger immediate scoring penalty

═══════════════════════════════════════════════════════════════

📋 TASK CONTRACT (arriving in next message):

You will receive:
• Exact file paths to modify (use these EXACT paths)
• Specific TODO items with context hints
• Role and objective

You must produce:
• Modified files (via write_file or apply_patch)
• Updated TODO status for each item
• Knowledge entries for key decisions

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

// RegisterCoordinatorTools registers all coordinator tools with the tool registry
func RegisterCoordinatorTools(
	registry *aiservice.ToolRegistry,
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	toolsDiscoveryHandler *handlers.ToolsDiscoveryHandler,
	subchatStorage *storage.SubchatStorage,
	aiService AIServiceInterface,
	chatService ChatServiceInterface,
	aiSettingsService AISettingsServiceInterface,
	logger *zap.Logger,
) error {
	tools := []aiservice.ToolExecutor{
		// Existing tools
		&CreateAgentTaskTool{storage: taskStorage},
		&ListAgentTasksTool{storage: taskStorage},
		&QueryKnowledgeTool{storage: knowledgeStorage},

		// New tools
		&UpsertKnowledgeTool{storage: knowledgeStorage},
		&GetPopularCollectionsTool{storage: knowledgeStorage},
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

		// Subagent tools
		&ListSubagentsTool{mongoDatabase: nil},
		&SetCurrentSubagentTool{mongoDatabase: nil},
		&ExecuteSubagentTool{
			subchatStorage:    subchatStorage,
			taskStorage:       taskStorage,
			aiService:         aiService,
			chatService:       chatService,
			aiSettingsService: aiSettingsService,
			logger:            logger,
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
