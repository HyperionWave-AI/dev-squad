package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiservice "hyper/internal/ai-service"
	"hyper/internal/ai-service/executor"
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

// chatServiceAdapter adapts ChatServiceInterface to executor.ChatServiceInterface.
// The executor interface expects string output while our interface uses interface{}.
type chatServiceAdapter struct {
	service   ChatServiceInterface
	sessionID primitive.ObjectID
	companyID string
}

func (a *chatServiceAdapter) SaveMessage(ctx context.Context, sessionID primitive.ObjectID, role, content, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveMessage(ctx, sessionID, role, content, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

func (a *chatServiceAdapter) SaveToolCall(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName string, args map[string]interface{}, companyID string) (*interface{}, error) {
	msg, err := a.service.SaveToolCall(ctx, sessionID, toolCallID, toolName, args, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

func (a *chatServiceAdapter) SaveToolResult(ctx context.Context, sessionID primitive.ObjectID, toolCallID, toolName, output, errorMsg string, durationMs int64, companyID string) (*interface{}, error) {
	// Convert string output to interface{} for the underlying service
	msg, err := a.service.SaveToolResult(ctx, sessionID, toolCallID, toolName, output, errorMsg, durationMs, companyID)
	if err != nil {
		return nil, err
	}
	var result interface{} = msg
	return &result, nil
}

// progressNotifierAdapter adapts handlers.ProgressNotifier to executor.ProgressNotifierInterface
type progressNotifierAdapter struct {
	notifier *handlers.ProgressNotifier
}

func (p *progressNotifierAdapter) EmitProgress(sessionID primitive.ObjectID, message string) {
	p.notifier.EmitProgress(sessionID, message)
}

// broadcasterAdapter adapts handlers.WebSocketBroadcaster to executor.WebSocketBroadcasterInterface
// This allows SubchatOutputSink to broadcast directly to connected WebSocket clients
type broadcasterAdapter struct {
	broadcaster *handlers.WebSocketBroadcaster
}

func (b *broadcasterAdapter) BroadcastToSession(sessionID primitive.ObjectID, message models.StreamMessage) error {
	return b.broadcaster.BroadcastToSession(sessionID, message)
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
// REFACTORED: Now uses StreamExecutor for clean, consistent execution (same as direct subagent chat)
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

	// ============================================================================
	// STREAMEXECUTOR-BASED EXECUTION (Clean, consistent with direct subagent chat)
	// ============================================================================

	// Define allowed tools for subagents (ONLY implementation tools, NO coordinator tools)
	allowedTools := []string{
		"read_file",                      // Read source files
		"write_file",                     // Write/create files
		"apply_patch",                    // Apply code patches
		"bash",                           // Run commands, tests
		"coordinator_update_todo_status", // Update TODO status
		"coordinator_upsert_knowledge",   // Save knowledge/decisions
		"code_index_search",              // Search codebase (needed for context)
	}

	t.logger.Info("🔒 Configuring StreamExecutor with filtered tools",
		zap.Int("allowedTools", len(allowedTools)),
		zap.Strings("tools", allowedTools))

	// Create adapters for streaming
	broadcaster := &broadcasterAdapter{
		broadcaster: handlers.GetWebSocketBroadcaster(t.logger),
	}
	progressNotifier := &progressNotifierAdapter{
		notifier: handlers.GetProgressNotifier(t.logger),
	}

	// Create output sink that sends to subchat session and notifies parent
	// KEY FIX: Uses broadcaster for direct WebSocket streaming (no pre-registration needed)
	outputSink := executor.NewSubchatOutputSink(
		chatSession.ID,    // subchat session for real-time display
		parentSessionID,   // parent session for progress notifications
		broadcaster,       // WebSocket broadcaster for subchat streaming
		progressNotifier,  // progress notifier for parent updates
		agentTask.AgentName,
		t.logger,
	)

	// Create chat service adapter for StreamExecutor
	chatAdapter := &chatServiceAdapter{
		service:   t.chatService,
		sessionID: chatSession.ID,
		companyID: finalCompanyID,
	}

	// Configure StreamExecutor
	execConfig := executor.StreamConfig{
		SessionID:    chatSession.ID,
		CompanyID:    finalCompanyID,
		SystemPrompt: systemPrompt,
		AllowedTools: allowedTools,
		OutputSink:   outputSink,
		InterruptCh:  notifyCh,
		Logger:       t.logger,
	}

	// Create and execute StreamExecutor (same as direct subagent chat!)
	exec := executor.NewStreamExecutor(execConfig, chatAdapter, t.aiService)

	// Create initial messages (note: system prompt is in execConfig, not in messages)
	messages := []aiservice.Message{
		{
			Role:    "user",
			Content: taskPrompt, // Task details only - system prompt injected by StreamExecutor
		},
	}

	t.logger.Info("🚀 Executing subchat via StreamExecutor",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.Int("messageCount", len(messages)))

	// Execute via StreamExecutor (clean, consistent with direct subagent chat)
	fullResponse, err := exec.Execute(ctx, messages)
	if err != nil {
		t.logger.Error("StreamExecutor failed",
			zap.String("subchatId", subchatID),
			zap.Error(err))
		handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID, fmt.Sprintf("⚠️ Subchat failed: %s", agentTask.AgentName))
		t.handleExecutionFailure(agentTask.ID, fmt.Sprintf("Execution failed: %v", err))
		return
	}

	// ============================================================================
	// EXECUTION COMPLETED - Update task status
	// ============================================================================

	t.logger.Info("╔═══════════════════════════════════════════════════════════════════")
	t.logger.Info("║ ✅ SUBAGENT EXECUTION COMPLETED")
	t.logger.Info("╠═══════════════════════════════════════════════════════════════════",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName),
		zap.String("agentTaskId", agentTask.ID),
		zap.Int("responseLength", len(fullResponse)))
	t.logger.Info("╚═══════════════════════════════════════════════════════════════════")

	// Mark task as completed
	summaryNotes := fmt.Sprintf("✅ Completed via StreamExecutor. Response length: %d", len(fullResponse))
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

	// Emit completion message
	completionMessage := fmt.Sprintf("✅ **Task Completed!**\n\n**Agent:** %s\n\nThe task has been successfully completed. You can ask me questions or request changes!",
		agentTask.AgentName)

	// Save completion message to database
	_, err = t.chatService.SaveMessage(ctx, chatSession.ID, "assistant", completionMessage, finalCompanyID)
	if err != nil {
		t.logger.Warn("Failed to save completion message",
			zap.String("subchatId", subchatID),
			zap.Error(err))
	}

	t.logger.Info("🎉 Subagent execution completed successfully via StreamExecutor",
		zap.String("subchatId", subchatID),
		zap.String("agentName", agentTask.AgentName))

	// Note: Post-completion conversation handling is now done by the StreamExecutor's
	// interrupt handling mechanism, which is the same as direct subagent chat
	_ = progressTracker // Suppress unused variable warning (tracker used for file operation logging)
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

