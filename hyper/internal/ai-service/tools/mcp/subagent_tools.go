package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/tools"
	"hyper/internal/handlers"
	"hyper/internal/mcp/storage"
	"hyper/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

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

	case "code_index_query":
		// Summarize code search results
		if resultMap, ok := output.(map[string]interface{}); ok {
			// Extract result count
			if results, ok := resultMap["results"].([]interface{}); ok {
				count := len(results)
				if count == 0 {
					return "📄 Code search completed: No results found"
				}
				
				// Build summary of first few results
				summary := fmt.Sprintf("📄 Code search completed: Found %d result(s)", count)
				if count > 0 && count <= 3 {
					// For small result sets, show file names
					for i, r := range results {
						if i >= 3 { break }
						if resultItem, ok := r.(map[string]interface{}); ok {
							if filePath, ok := resultItem["filePath"].(string); ok {
								summary += fmt.Sprintf("\n  • %s", filePath)
							}
						}
					}
				}
				return summary
			}
		}
		return "📄 Code search completed"

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
📋 TASK CONTRACT (arriving in next message):
═══════════════════════════════════════════════════════════════

You will receive:
• Exact file paths to modify (use these EXACT paths)
• Specific TODO items with context hints
• Role and objective

You must produce:
• Modified files (via write_file or apply_patch)
• Updated TODO status for each item
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

	case "code_index_query":
		// Summarize code search results with file metadata and key logic
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(outputStr), &result); err == nil {
			if results, ok := result["results"].([]interface{}); ok {
				if len(results) == 0 {
					return "Code search: No results found"
				}

				// Build concise summary of results
				var summary strings.Builder
				summary.WriteString(fmt.Sprintf("📄 Code search: Found %d result(s)\n", len(results)))

				// Process first 3 results for detailed summary
				for i, r := range results {
					if i >= 3 { break }
					if resultItem, ok := r.(map[string]interface{}); ok {
						// Extract metadata
						filePath := ""
						if fp, ok := resultItem["filePath"].(string); ok {
							filePath = fp
						}
						startLine := 0
						if sl, ok := resultItem["startLine"].(float64); ok {
							startLine = int(sl)
						}
						endLine := 0
						if el, ok := resultItem["endLine"].(float64); ok {
							endLine = int(el)
						}
						content := ""
						if c, ok := resultItem["content"].(string); ok {
							content = c
						}

						// Build result summary
						summary.WriteString(fmt.Sprintf("\n%d. 📄 %s (lines %d-%d)\n", i+1, filePath, startLine, endLine))
						
						// Extract first meaningful line of code for context
						if len(content) > 0 {
							lines := strings.Split(content, "\n")
							summary.WriteString(fmt.Sprintf("   Preview: %s\n", strings.TrimSpace(lines[0])))
						}
					}
				}
				return summary.String()
			}
		}
		return "Code search completed"
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

// categorizeInterrupt analyzes user interrupt message to determine intent and provide guidance
func (t *ExecuteSubagentTool) categorizeInterrupt(ctx context.Context, userMessage string) (string, string, error) {
	categorizationPrompt := fmt.Sprintf(`You are an interrupt analyzer. Analyze this user message sent while an AI agent was working:

User message: "%s"

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

