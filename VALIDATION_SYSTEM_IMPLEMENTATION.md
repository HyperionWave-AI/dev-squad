# Mandatory Code Validation System - Implementation Guide

## Problem Statement
AI agents make code changes that introduce TypeScript/ESLint errors (unused imports, undefined variables, type errors) and mark tasks as complete without fixing them.

## Solution Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Agent makes changes (apply_patch/file_write)            │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. POST-WRITE HOOK: Auto-run validation                    │
│    - Detect modified files                                  │
│    - Run TypeScript/ESLint/Go checks                        │
│    - Parse errors                                           │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. DECISION: Errors found?                                  │
└────────────────┬────────────────────────────────────────────┘
         Yes │   │ No
             │   └─────────────────────────┐
             ▼                             ▼
┌─────────────────────────────────┐   ┌────────────────────┐
│ 4. INJECT MANDATORY TODO:       │   │ 5. Task continues  │
│    "Fix validation errors"      │   │    normally        │
│    - Status: pending             │   └────────────────────┘
│    - IsMandatory: true           │
│    - Contains error list         │
└─────────────┬───────────────────┘
              │
              ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. AGENT LOOP: Fix errors                                  │
│    - Agent reads errors                                     │
│    - Makes corrections                                      │
│    - Re-validates automatically                             │
│    - Loop until all errors fixed                            │
└────────────────┬────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. COMPLETION BLOCKER: Check before marking complete       │
│    - Verify all mandatory TODOs are done                    │
│    - Run final validation                                   │
│    - Only allow completion if passed                        │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Steps

### Step 1: Validation Service ✅ DONE
File: `hyper/internal/validation/code_validator.go`

**Features:**
- TypeScript validation via `npx tsc --noEmit`
- Go validation via `go vet`
- Error parsing and formatting
- Filtered errors (only for modified files)

### Step 2: Post-Write Validation Hook

**Location:** `hyper/internal/mcp/handlers/filesystem_tools.go`

**Modifications needed:**

#### A. Add validation after apply_patch success

```go
// In handleApplyPatch(), after line "err = os.WriteFile(...)"
// Add this:

if !dryRun {
	// Write modified content back to file
	err = os.WriteFile(validatedPath, []byte(modifiedContent), 0644)
	if err != nil {
		return createFilesystemErrorResult(fmt.Sprintf("failed to write modified file: %s", err.Error())), nil
	}

	// ✅ NEW: Auto-validate after writing
	go h.runPostWriteValidation(ctx, []string{validatedPath})

	result["message"] = "Patch applied successfully"
}
```

#### B. Add validation after file_write success

```go
// In handleFileWrite(), after line "err = file.Write(decodedContent)"
// Add this:

// Write content
bytesWritten, err := file.Write(decodedContent)
if err != nil {
	return createFilesystemErrorResult(fmt.Sprintf("failed to write file: %s", err.Error())), nil
}

// ✅ NEW: Auto-validate after writing
go h.runPostWriteValidation(ctx, []string{validatedPath})
```

#### C. Add the validation method to FilesystemToolHandler

```go
// Add these imports at top
import (
	"hyper/internal/validation"
	"hyper/internal/mcp/storage"
)

// Add validator field to struct
type FilesystemToolHandler struct {
	logger           *zap.Logger
	baseDir          string
	metadataRegistry *ToolMetadataRegistry
	validator        *validation.CodeValidator  // ✅ NEW
	taskStorage      storage.TaskStorage        // ✅ NEW
}

// Update NewFilesystemToolHandler
func NewFilesystemToolHandler(logger *zap.Logger, validator *validation.CodeValidator, taskStorage storage.TaskStorage) *FilesystemToolHandler {
	baseDir := tools.GetProjectRoot()
	return &FilesystemToolHandler{
		logger:      logger,
		baseDir:     baseDir,
		validator:   validator,
		taskStorage: taskStorage,
	}
}

// Add validation method
func (h *FilesystemToolHandler) runPostWriteValidation(ctx context.Context, files []string) {
	// Filter to only validate TypeScript/Go files
	var validatableFiles []string
	for _, file := range files {
		ext := filepath.Ext(file)
		if ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".go" {
			validatableFiles = append(validatableFiles, file)
		}
	}

	if len(validatableFiles) == 0 {
		return
	}

	h.logger.Info("Running post-write validation", zap.Strings("files", validatableFiles))

	// Run validation
	result, err := h.validator.ValidateFiles(ctx, validatableFiles)
	if err != nil {
		h.logger.Warn("Validation failed to run", zap.Error(err))
		return
	}

	if !result.Passed {
		h.logger.Warn("Validation errors detected",
			zap.Int("errorCount", len(result.Errors)),
			zap.Strings("files", validatableFiles))

		// Get current agent task from context (if available)
		if agentTaskID := getAgentTaskIDFromContext(ctx); agentTaskID != "" {
			h.injectValidationTodo(ctx, agentTaskID, result)
		}
	} else {
		h.logger.Info("Validation passed", zap.Int("filesChecked", len(validatableFiles)))
	}
}

// Inject mandatory validation TODO
func (h *FilesystemToolHandler) injectValidationTodo(ctx context.Context, agentTaskID string, validationResult *validation.ValidationResult) {
	errorSummary := h.validator.FormatErrorsForAgent(validationResult)

	todo := storage.TodoItemInput{
		Description: fmt.Sprintf("🔴 MANDATORY: Fix %d validation error(s)", len(validationResult.Errors)),
		Status:      storage.TodoStatusPending,
		IsMandatory: true,
		ContextHint: errorSummary,
	}

	err := h.taskStorage.AddTodoToAgentTask(agentTaskID, todo)
	if err != nil {
		h.logger.Error("Failed to inject validation TODO", zap.Error(err))
	} else {
		h.logger.Info("Injected mandatory validation TODO",
			zap.String("taskID", agentTaskID),
			zap.Int("errors", len(validationResult.Errors)))
	}
}
```

### Step 3: Enhance Task Storage with Mandatory TODOs

**Location:** `hyper/internal/mcp/storage/tasks.go`

**Add to TodoItem struct:**

```go
type TodoItem struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Description string    `bson:"description" json:"description"`
	Status      string    `bson:"status" json:"status"` // "pending", "in_progress", "completed"
	FilePath    string    `bson:"filePath,omitempty" json:"filePath,omitempty"`
	// ... existing fields ...

	// ✅ NEW FIELDS
	IsMandatory    bool   `bson:"isMandatory,omitempty" json:"isMandatory,omitempty"`
	ValidationCmd  string `bson:"validationCmd,omitempty" json:"validationCmd,omitempty"`
	ContextHint    string `bson:"contextHint,omitempty" json:"contextHint,omitempty"`
}

type TodoItemInput struct {
	Description string `json:"description"`
	// ... existing fields ...

	// ✅ NEW FIELDS
	IsMandatory   bool   `json:"isMandatory,omitempty"`
	ValidationCmd string `json:"validationCmd,omitempty"`
	ContextHint   string `json:"contextHint,omitempty"`
}
```

**Add method to inject TODOs:**

```go
// AddTodoToAgentTask adds a new TODO to an existing agent task
func (s *MongoTaskStorage) AddTodoToAgentTask(agentTaskID string, todo TodoItemInput) error {
	// Generate unique ID for todo
	todoID := uuid.New().String()

	todoItem := TodoItem{
		ID:             todoID,
		Description:    todo.Description,
		Status:         string(TodoStatusPending),
		FilePath:       todo.FilePath,
		FunctionName:   todo.FunctionName,
		IsMandatory:    todo.IsMandatory,
		ValidationCmd:  todo.ValidationCmd,
		ContextHint:    todo.ContextHint,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Append to todos array
	filter := bson.M{"taskId": agentTaskID}
	update := bson.M{
		"$push": bson.M{"todos": todoItem},
		"$set":  bson.M{"updatedAt": time.Now()},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.agentTasksColl.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to add todo: %w", err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("agent task not found: %s", agentTaskID)
	}

	s.logger.Info("Added TODO to agent task",
		zap.String("taskID", agentTaskID),
		zap.String("todoID", todoID),
		zap.String("description", todo.Description),
		zap.Bool("mandatory", todo.IsMandatory))

	return nil
}
```

### Step 4: Task Completion Blocker

**Location:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go`

**In UpdateTaskStatusTool.Execute(), add validation before allowing completion:**

```go
func (t *UpdateTaskStatusTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	agentTaskID, _ := input["agentTaskId"].(string)
	newStatus, _ := input["status"].(string)
	notes, _ := input["notes"].(string)

	// ✅ NEW: Block completion if mandatory TODOs incomplete
	if newStatus == "completed" {
		task, err := t.storage.GetAgentTask(agentTaskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get task: %w", err)
		}

		// Check for incomplete mandatory TODOs
		var incompleteMandatory []string
		for _, todo := range task.Todos {
			if todo.IsMandatory && todo.Status != string(storage.TodoStatusCompleted) {
				incompleteMandatory = append(incompleteMandatory, todo.Description)
			}
		}

		if len(incompleteMandatory) > 0 {
			return nil, fmt.Errorf(
				"❌ Cannot complete task - %d mandatory TODO(s) not completed:\n%s\n\nYou MUST complete these before marking the task as done.",
				len(incompleteMandatory),
				strings.Join(incompleteMandatory, "\n- "))
		}

		// Run final validation check
		if len(task.FilesModified) > 0 {
			validator := validation.NewCodeValidator(zap.L(), getProjectRoot())
			result, err := validator.ValidateFiles(ctx, task.FilesModified)

			if err == nil && !result.Passed {
				// Inject new validation TODO and block completion
				errorMsg := validator.FormatErrorsForAgent(result)
				return nil, fmt.Errorf(
					"❌ Cannot complete task - validation failed with %d error(s):\n\n%s",
					len(result.Errors), errorMsg)
			}
		}
	}

	// Original update logic continues...
	err := t.storage.UpdateAgentTaskStatus(agentTaskID, newStatus, notes)
	// ... rest of function
}
```

### Step 5: Tool Executor Enhancement

**Location:** `hyper/internal/ai-service/tool_executor.go`

**Add validation phase after tool execution:**

```go
// In StreamChatWithTools(), after tool execution loop, add:

// ✅ NEW: Validation Phase
if hasFileModifications(executedTools) {
	modifiedFiles := extractModifiedFiles(executedTools)

	validator := validation.NewCodeValidator(s.logger, getProjectRoot())
	validationResult, err := validator.ValidateFiles(ctx, modifiedFiles)

	if err == nil && !validationResult.Passed {
		// Format errors as a tool result for agent to see
		errorMessage := validator.FormatErrorsForAgent(validationResult)

		// Create synthetic tool result with validation errors
		validationToolResult := &StreamEvent{
			Type: StreamEventToolResult,
			ToolResult: &ToolResult{
				Name:   "validation_check",
				Output: errorMessage,
			},
		}

		// Send validation result to agent
		select {
		case resultChan <- validationToolResult:
		case <-ctx.Done():
			return
		}

		// Continue execution to allow agent to fix errors
		// Don't break the loop - let agent make corrections
	}
}
```

## Integration Checklist

- [x] 1. Create validation service (`internal/validation/code_validator.go`)
- [ ] 2. Add validator to FilesystemToolHandler
- [ ] 3. Add post-write validation hooks in apply_patch and file_write
- [ ] 4. Add IsMandatory field to TodoItem struct
- [ ] 5. Implement AddTodoToAgentTask method
- [ ] 6. Add completion blocker in UpdateTaskStatusTool
- [ ] 7. Add validation phase in tool_executor.go
- [ ] 8. Update agent task creation to include agentTaskID in context
- [ ] 9. Test with a task that introduces errors
- [ ] 10. Verify agent fixes errors before completion

## Configuration

Add to `.env.hyper`:

```bash
# Validation Settings
ENABLE_AUTO_VALIDATION=true
VALIDATION_TIMEOUT=30s  # Timeout for validation checks
VALIDATION_FAIL_FAST=false  # If true, stop immediately on error
```

## Testing Plan

### Test Case 1: Unused Import Error

**Setup:**
```bash
# Create task that adds unused import
curl -X POST http://localhost:5555/api/v1/chat/sessions/:sessionId/messages \
  -d '{"content": "Add useState import to SessionList.tsx"}'
```

**Expected Flow:**
1. Agent adds import
2. Validation runs → detects unused import error
3. Mandatory TODO injected: "Fix validation errors"
4. Agent reads error
5. Agent removes unused import
6. Validation passes
7. Task completed

### Test Case 2: Undefined Variable Error

**Setup:**
```bash
# Create task that uses undefined variable
curl -X POST http://localhost:5555/api/v1/chat/sessions/:sessionId/messages \
  -d '{"content": "Add onClick handler that calls handleClick"}'
```

**Expected Flow:**
1. Agent adds onClick={handleClick}
2. Validation runs → detects "handleClick is not defined"
3. Mandatory TODO injected
4. Agent defines handleClick function
5. Validation passes
6. Task completed

### Test Case 3: Completion Blocker

**Setup:**
```bash
# Try to complete task with pending mandatory TODO
```

**Expected:**
```json
{
  "error": "❌ Cannot complete task - 1 mandatory TODO(s) not completed:\n- Fix validation errors"
}
```

## Benefits

1. **🔒 Quality Gate:** No task completes with errors
2. **🔁 Auto-Fix Loop:** Agent automatically retries until clean
3. **📊 Visibility:** Clear error messages guide the agent
4. **⚡ Fast Feedback:** Errors caught immediately after write
5. **🎯 Targeted:** Only validates modified files
6. **🚫 No Silent Failures:** Completion blocked if errors exist

## Monitoring

Add Prometheus metrics:

```go
var (
	validationRunsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "validation_runs_total",
			Help: "Total number of validation runs",
		})

	validationErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "validation_errors_total",
			Help: "Total number of validation errors by type",
		},
		[]string{"file_type", "error_type"})

	validationDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "validation_duration_seconds",
			Help: "Time spent running validation",
		})
)
```

## Future Enhancements

1. **ESLint Integration:** Add ESLint checks alongside TypeScript
2. **Auto-Fix Suggestions:** Provide fix hints from ESLint --fix
3. **Custom Rules:** Allow project-specific validation rules
4. **Performance:** Cache validation results for unchanged files
5. **Parallel Validation:** Run TypeScript and Go checks concurrently
6. **Smart Retries:** If agent fails to fix after 3 attempts, escalate to human

## References

- TypeScript Compiler API: https://github.com/microsoft/TypeScript/wiki/Using-the-Compiler-API
- Go vet: https://pkg.go.dev/cmd/vet
- ESLint: https://eslint.org/docs/developer-guide/nodejs-api
