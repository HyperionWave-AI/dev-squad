package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	mcphandlers "hyper/internal/mcp/handlers"
	"hyper/internal/mcp/storage"

	"go.uber.org/zap"
)

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

// AnalyzeComplexityTool implements complexity analysis for tasks
type AnalyzeComplexityTool struct {
	aiService AIServiceInterface
}

func (t *AnalyzeComplexityTool) Name() string {
	return "analyze_complexity"
}

func (t *AnalyzeComplexityTool) Description() string {
	return "Analyze task complexity using 5 heuristics (file count, file size, cross-squad impact, architectural scope, estimated line changes). Returns complexity score (0.0-1.0), level (low/medium/high/extreme), and split suggestions if score >= 0.6. IMPORTANT: Call this BEFORE creating agent tasks to determine if work should be split. Only available when complexity analysis mode is enabled."
}

func (t *AnalyzeComplexityTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"description": map[string]interface{}{
				"type":        "string",
				"description": "Task description - what needs to be done and why",
			},
			"filesModified": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "string",
				},
				"description": "Array of file paths that will be modified (discovered from code_index_search)",
			},
			"role": map[string]interface{}{
				"type":        "string",
				"description": "Agent role or objective for this work",
			},
			"contextSummary": map[string]interface{}{
				"type":        "string",
				"description": "Additional context about the changes needed",
			},
		},
		"required": []string{"description", "filesModified"},
	}
}

func (t *AnalyzeComplexityTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Check if complexity analysis mode is enabled
	complexityMode := ctx.Value("complexityAnalysisMode")
	if complexityMode == nil || !complexityMode.(bool) {
		return map[string]interface{}{
			"error":   "Complexity analysis mode is disabled",
			"message": "Enable complexity analysis mode (purple toggle button) to use this tool",
		}, nil
	}

	// Extract inputs
	description, _ := input["description"].(string)
	role, _ := input["role"].(string)
	contextSummary, _ := input["contextSummary"].(string)

	// Extract filesModified array
	var filesModified []string
	if fm, ok := input["filesModified"].([]interface{}); ok {
		for _, f := range fm {
			if str, ok := f.(string); ok {
				filesModified = append(filesModified, str)
			}
		}
	}

	// Validate required fields
	if description == "" {
		return nil, fmt.Errorf("description is required")
	}
	if len(filesModified) == 0 {
		return nil, fmt.Errorf("filesModified is required and must not be empty")
	}

	zap.L().Info("🔍 Analyzing task complexity",
		zap.String("description", description),
		zap.Int("filesCount", len(filesModified)))

	// Get AI config and create chat provider
	config := t.aiService.GetConfig()
	chatProvider, err := aiservice.NewChatProvider(config, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat provider: %w", err)
	}

	// Create complexity analyzer
	analyzer := aiservice.NewComplexityAnalyzer(config, chatProvider)

	// Build task context
	taskContext := aiservice.TaskContext{
		Description:    description,
		FilesModified:  filesModified,
		Role:           role,
		ContextSummary: contextSummary,
	}

	// Perform complexity analysis
	analysis, err := analyzer.AnalyzeComplexity(ctx, taskContext)
	if err != nil {
		return nil, fmt.Errorf("complexity analysis failed: %w", err)
	}

	zap.L().Info("📊 Complexity Analysis Complete",
		zap.Float64("score", analysis.OverallScore),
		zap.String("level", string(analysis.Level)),
		zap.Bool("shouldSplit", analysis.ShouldSplit))

	// Build result
	result := map[string]interface{}{
		"overallScore": analysis.OverallScore,
		"level":        string(analysis.Level),
		"shouldSplit":  analysis.ShouldSplit,
		"reasoning":    analysis.Reasoning,
		"heuristicScores": map[string]interface{}{
			"fileCount":            analysis.HeuristicScores.FileCount,
			"fileSize":             analysis.HeuristicScores.FileSize,
			"crossSquadImpact":     analysis.HeuristicScores.CrossSquadImpact,
			"architecturalScope":   analysis.HeuristicScores.ArchitecturalScope,
			"estimatedLineChanges": analysis.HeuristicScores.EstimatedLineChanges,
		},
		"recommendation": fmt.Sprintf("Complexity: %.2f (%s) - %s",
			analysis.OverallScore,
			analysis.Level,
			analyzer.GetComplexityDescription(analysis.Level)),
	}

	// Generate split suggestions if task is complex
	if analysis.ShouldSplit {
		zap.L().Info("⚠️  Task is COMPLEX - generating split suggestions",
			zap.Float64("score", analysis.OverallScore))

		suggestions, err := analyzer.GenerateSplitSuggestions(ctx, taskContext, analysis)
		if err != nil {
			zap.L().Warn("Failed to generate split suggestions", zap.Error(err))
			result["splitSuggestionsError"] = err.Error()
		} else {
			zap.L().Info("✅ Generated split suggestions",
				zap.Int("count", len(suggestions)))

			suggestionsArray := make([]map[string]interface{}, len(suggestions))
			for i, s := range suggestions {
				suggestionsArray[i] = map[string]interface{}{
					"subtaskTitle":        s.SubtaskTitle,
					"subtaskDescription":  s.SubtaskDescription,
					"filesInvolved":       s.FilesInvolved,
					"estimatedComplexity": s.EstimatedComplexity,
					"dependencies":        s.Dependencies,
					"rationale":           s.Rationale,
				}
			}
			result["splitSuggestions"] = suggestionsArray
			result["recommendedSplitCount"] = analyzer.GetRecommendedSplitCount(analysis)
		}
	}

	return result, nil
}

// CreateAgentTaskTool implements the ToolExecutor interface
type CreateAgentTaskTool struct {
	storage   storage.TaskStorage
	aiService AIServiceInterface // For complexity analysis
}

func (t *CreateAgentTaskTool) Name() string {
	return "create_agent_task"
}

func (t *CreateAgentTaskTool) Description() string {
	return "Create a new agent task linked to a human task. Returns task ID. SMART AUTO-FETCH: If humanTaskId is omitted, automatically fetches the most recent pending human task from the database. IMPORTANT: Use code_index_search FIRST to discover relevant files, then populate filesModified with the file paths from search results. Include detailed context in contextSummary with WHAT to change, WHERE (file:line from search results), and HOW. NEVER ask the user for file paths - discover them automatically with code_index_search. COMPLEXITY ANALYSIS REQUIREMENT: When complexity analysis mode is ON (purple toggle), you MUST call coordinator_analyze_complexity FIRST and pass the result in the complexityAnalysis field. This is MANDATORY and task creation will fail without it. Required: agentName, role, todos. Optional: humanTaskId, contextSummary, filesModified, qdrantCollections, priorWorkSummary, complexityAnalysis."
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
			"complexityAnalysis": map[string]interface{}{
				"type":        "object",
				"description": "MANDATORY when complexity analysis mode is ON. Result from coordinator_analyze_complexity tool call. Must include: overallScore, level, shouldSplit, reasoning. Pass the entire result object from the analyze_complexity tool.",
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
	contextSummary, _ := input["contextSummary"].(string)
	priorWorkSummary, _ := input["priorWorkSummary"].(string)

	// FIX #1: Strict Type Validation for filesModified
	// Previously: Silent type coercion failure would leave empty strings in array
	// Now: Fail fast with clear error message if any element is not a string
	var filesModified []string
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

	// MANDATORY COMPLEXITY ANALYSIS CHECK: When toggle is ON, require pre-analysis
	var complexityAnalysisResult *map[string]interface{}
	complexityModeEnabled := ctx.Value("complexityAnalysisMode") != nil && ctx.Value("complexityAnalysisMode").(bool)

	if complexityModeEnabled {
		zap.L().Info("🔍 Complexity Analysis Mode: ENABLED - checking for mandatory pre-analysis",
			zap.String("agentName", agentName),
			zap.Int("filesCount", len(filesModified)),
			zap.Int("todosCount", len(todos)))

		// Check if complexityAnalysis was provided
		complexityAnalysisInput, hasComplexityAnalysis := input["complexityAnalysis"].(map[string]interface{})

		if !hasComplexityAnalysis || complexityAnalysisInput == nil {
			// FAIL: Complexity analysis mode is ON but no analysis was provided
			return nil, fmt.Errorf(
				"❌ COMPLEXITY ANALYSIS REQUIRED\n\n"+
					"Complexity Analysis Mode is ON (purple toggle enabled).\n"+
					"You MUST call coordinator_analyze_complexity BEFORE creating agent tasks.\n\n"+
					"📋 MANDATORY WORKFLOW:\n"+
					"1. Call coordinator_analyze_complexity with:\n"+
					"   - description: %q\n"+
					"   - role: %q\n"+
					"   - contextSummary: (your context)\n"+
					"   - filesModified: %v\n\n"+
					"2. Review the complexity score and split suggestions\n\n"+
					"3. If shouldSplit is true (score >= 0.6):\n"+
					"   - Split the task into multiple smaller agent tasks\n"+
					"   - Use the splitSuggestions from the analysis result\n"+
					"   - Create separate agent tasks for each subtask\n\n"+
					"4. If shouldSplit is false (score < 0.6):\n"+
					"   - Call create_agent_task again with complexityAnalysis field:\n"+
					"   - Pass the ENTIRE result object from analyze_complexity\n\n"+
					"Example:\n"+
					"  analysis = coordinator_analyze_complexity({...})\n"+
					"  if analysis.shouldSplit:\n"+
					"    # Create multiple smaller tasks\n"+
					"  else:\n"+
					"    coordinator_create_agent_task({..., complexityAnalysis: analysis})\n\n"+
					"⚠️  This check is MANDATORY when the purple toggle is ON.\n"+
					"💡 To disable this check, turn OFF the complexity analysis toggle in the UI.",
				role,
				role,
				filesModified)
		}

		// Validate that complexityAnalysis has required fields
		overallScore, hasScore := complexityAnalysisInput["overallScore"].(float64)
		level, hasLevel := complexityAnalysisInput["level"].(string)
		shouldSplit, hasShouldSplit := complexityAnalysisInput["shouldSplit"].(bool)
		reasoning, hasReasoning := complexityAnalysisInput["reasoning"].(string)

		if !hasScore || !hasLevel || !hasShouldSplit || !hasReasoning {
			return nil, fmt.Errorf(
				"❌ INVALID COMPLEXITY ANALYSIS\n\n"+
					"The complexityAnalysis object is incomplete. Required fields:\n"+
					"- overallScore (float64): %v (present: %v)\n"+
					"- level (string): %v (present: %v)\n"+
					"- shouldSplit (bool): %v (present: %v)\n"+
					"- reasoning (string): %v (present: %v)\n\n"+
					"💡 Pass the ENTIRE result object from coordinator_analyze_complexity.\n"+
					"   Do NOT manually construct the complexityAnalysis field.",
				hasScore, hasScore, hasLevel, hasLevel, hasShouldSplit, hasShouldSplit, hasReasoning, hasReasoning)
		}

		// Log successful validation
		zap.L().Info("✅ Complexity analysis validated",
			zap.Float64("score", overallScore),
			zap.String("level", level),
			zap.Bool("shouldSplit", shouldSplit))

		// WARN if task should be split but coordinator is creating it anyway
		if shouldSplit {
			zap.L().Warn("⚠️  HIGH COMPLEXITY TASK - Should be split but proceeding anyway",
				zap.Float64("score", overallScore),
				zap.String("level", level),
				zap.String("reasoning", reasoning))

			// Include warning in logs but allow task creation (coordinator made an informed decision)
		}

		// Store the validated complexity analysis for return
		complexityAnalysisResult = &complexityAnalysisInput
	} else {
		zap.L().Debug("Complexity analysis mode: DISABLED - no pre-analysis required")
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

	// Return task summary with optional complexity analysis
	result := map[string]interface{}{
		"taskId":     task.ID,
		"agentName":  task.AgentName,
		"role":       task.Role,
		"status":     task.Status,
		"todosCount": len(task.Todos),
		"createdAt":  task.CreatedAt,
	}

	// Include complexity analysis if performed
	if complexityAnalysisResult != nil {
		result["complexityAnalysis"] = *complexityAnalysisResult
	}

	return result, nil
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
	return "List human tasks from the coordinator database with pagination. Use limit/offset for large task lists."
}

func (t *ListHumanTasksTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"status": map[string]interface{}{
				"type":        "string",
				"description": "Filter by status (pending, in_progress, completed)",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of tasks to return (default: 20, max: 50)",
				"default":     20,
			},
			"offset": map[string]interface{}{
				"type":        "integer",
				"description": "Number of tasks to skip for pagination",
				"default":     0,
			},
		},
	}
}

// maxListHumanTasksLimit is the hard limit to prevent massive results
const maxListHumanTasksLimit = 50

func (t *ListHumanTasksTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Extract pagination params with defaults
	limit := 20
	if l, ok := input["limit"].(float64); ok && l > 0 {
		limit = int(l)
	}
	// ENFORCE max limit to prevent massive results
	if limit > maxListHumanTasksLimit {
		limit = maxListHumanTasksLimit
	}

	offset := 0
	if o, ok := input["offset"].(float64); ok && o >= 0 {
		offset = int(o)
	}

	// Get status filter if provided
	statusFilter, _ := input["status"].(string)

	// Get all tasks then apply filters and pagination
	allTasks := t.storage.ListAllHumanTasks()

	// Apply status filter if provided
	var filteredTasks []*storage.HumanTask
	for _, task := range allTasks {
		if statusFilter == "" {
			filteredTasks = append(filteredTasks, task)
		} else if string(task.Status) == statusFilter {
			filteredTasks = append(filteredTasks, task)
		}
	}

	totalCount := len(filteredTasks)

	// Apply pagination
	start := offset
	if start > totalCount {
		start = totalCount
	}
	end := start + limit
	if end > totalCount {
		end = totalCount
	}

	var paginatedTasks []*storage.HumanTask
	if start < totalCount {
		paginatedTasks = filteredTasks[start:end]
	} else {
		paginatedTasks = []*storage.HumanTask{}
	}

	return map[string]interface{}{
		"tasks":   paginatedTasks,
		"count":   len(paginatedTasks),
		"total":   totalCount,
		"offset":  offset,
		"limit":   limit,
		"hasMore": end < totalCount,
		"hint":    "Use offset parameter to fetch more tasks. Example: {\"offset\": 20, \"limit\": 20}",
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


// RegisterCoordinatorTools registers all coordinator tools with the tool registry
func RegisterCoordinatorTools(
	registry *aiservice.ToolRegistry,
	taskStorage storage.TaskStorage,
	knowledgeStorage storage.KnowledgeStorage,
	toolsDiscoveryHandler *mcphandlers.ToolsDiscoveryHandler,
	subchatStorage *storage.SubchatStorage,
	aiService AIServiceInterface,
	chatService ChatServiceInterface,
	aiSettingsService AISettingsServiceInterface,
	logger *zap.Logger,
) error {
	tools := []aiservice.ToolExecutor{
		// Complexity analysis (call BEFORE creating tasks)
		&AnalyzeComplexityTool{aiService: aiService},

		// Existing tools
		&CreateAgentTaskTool{storage: taskStorage, aiService: aiService},
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
