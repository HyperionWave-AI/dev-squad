# Phantom Task Completion Bug - Fix Specification

## Problem Statement

Subagents can mark themselves as "completed" without actually doing any work. This creates phantom completions where:
- Task shows "completed" in the UI
- No code changes were made
- No files were modified
- Expected deliverables don't exist

## Root Cause

**File**: `hyper/internal/ai-service/tools/mcp/coordinator_tools.go`
**Function**: `executeSubagentInBackground` (lines 1658-1925)
**Issue**: Lines 1904-1912 mark task/subchat as completed WITHOUT validation

```go
// CURRENT CODE (BROKEN):
err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusCompleted, summaryNotes)
err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusCompleted)
```

## Proposed Fix

Add **3-layer validation** before allowing completion:

### Layer 1: File Modification Validation
```go
// Verify at least one expected file was actually modified
func (t *ExecuteSubagentTool) validateFileModifications(agentTask *storage.AgentTask) (bool, []string, error) {
    // Run: git diff --name-only HEAD
    cmd := exec.Command("git", "diff", "--name-only", "HEAD")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return false, nil, fmt.Errorf("git diff failed: %w", err)
    }

    modifiedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")

    // Check if any expected files were modified
    expectedFiles := make(map[string]bool)
    for _, file := range agentTask.FilesModified {
        expectedFiles[file] = true
    }

    matchedFiles := []string{}
    for _, modFile := range modifiedFiles {
        if modFile != "" && expectedFiles[modFile] {
            matchedFiles = append(matchedFiles, modFile)
        }
    }

    // Require at least 1 expected file to be modified
    if len(matchedFiles) == 0 && len(agentTask.FilesModified) > 0 {
        return false, matchedFiles, fmt.Errorf("expected files not modified: wanted %v, got nothing", agentTask.FilesModified)
    }

    return true, matchedFiles, nil
}
```

### Layer 2: TODO Completion Verification
```go
// ENFORCE all TODOs must be completed
if completedTodos < len(agentTask.Todos) {
    t.logger.Warn("Not all TODOs completed - marking as in_progress instead",
        zap.Int("completedTodos", completedTodos),
        zap.Int("totalTodos", len(agentTask.Todos)))

    summaryNotes := fmt.Sprintf("Partial completion: %d/%d TODOs done, %d tool calls",
        completedTodos, len(agentTask.Todos), toolCallCount)

    err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusInProgress, summaryNotes)
    err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusActive)
    return // Don't mark as completed
}
```

### Layer 3: Combined Validation with Proof Logging
```go
// Before marking as completed (line 1894), add:

// VALIDATION STEP 1: Check file modifications
if len(agentTask.FilesModified) > 0 {
    filesOK, modifiedFiles, err := t.validateFileModifications(agentTask)
    if !filesOK {
        t.logger.Warn("❌ File modification validation FAILED",
            zap.String("subchatId", subchatID),
            zap.String("agentTaskId", agentTask.ID),
            zap.Error(err))

        // Mark as BLOCKED instead of completed
        blockReason := fmt.Sprintf("Validation failed: %v. Tool calls: %d, Claimed TODOs: %d/%d",
            err, toolCallCount, completedTodos, len(agentTask.Todos))

        err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusBlocked, blockReason)
        err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusFailed)

        t.logger.Error("🚨 PHANTOM COMPLETION PREVENTED",
            zap.String("subchatId", subchatID),
            zap.String("reason", "no files modified"))
        return
    }

    // Log proof of work
    t.logger.Info("✅ File modification validation PASSED",
        zap.String("subchatId", subchatID),
        zap.Strings("modifiedFiles", modifiedFiles))
}

// VALIDATION STEP 2: Check TODO completion
if completedTodos < len(agentTask.Todos) {
    t.logger.Warn("❌ TODO completion validation FAILED",
        zap.Int("completed", completedTodos),
        zap.Int("total", len(agentTask.Todos)))

    summaryNotes := fmt.Sprintf("Incomplete: %d/%d TODOs done, %d tool calls. Files modified: %v",
        completedTodos, len(agentTask.Todos), toolCallCount, modifiedFiles)

    err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusInProgress, summaryNotes)
    err = t.subchatStorage.UpdateSubchatStatus(subchatID, storage.SubchatStatusActive)
    return
}

// ALL VALIDATIONS PASSED - Safe to mark as completed
summaryNotes := fmt.Sprintf("✅ VALIDATED completion: %d/%d TODOs, %d tool calls, %d files modified: %v",
    completedTodos, len(agentTask.Todos), toolCallCount, len(modifiedFiles), modifiedFiles)

t.logger.Info("🎉 Task completion validated successfully",
    zap.String("subchatId", subchatID),
    zap.String("agentTaskId", agentTask.ID),
    zap.Int("toolCalls", toolCallCount),
    zap.Int("completedTodos", completedTodos),
    zap.Strings("filesModified", modifiedFiles))

err = t.taskStorage.UpdateTaskStatus(agentTask.ID, storage.TaskStatusCompleted, summaryNotes)
// ... rest of completion logic
```

## Implementation Steps

1. **Add helper function** `validateFileModifications` to ExecuteSubagentTool struct
2. **Import `os/exec`** package for git commands
3. **Insert validation logic** before line 1894 (before marking as completed)
4. **Add comprehensive logging** for audit trail
5. **Test with failing case** (delete button task) to verify prevention

## Expected Behavior After Fix

### ✅ Valid Completion (Allowed)
- All TODOs marked completed: 5/5 ✓
- Expected files modified: TaskCard.tsx, TaskList.tsx ✓
- Git shows changes ✓
- **Result**: Task marked "completed" with proof logged

### ❌ Phantom Completion (BLOCKED)
- TODOs claimed completed: 5/5
- Expected files modified: NONE ✗
- Git shows no changes ✗
- **Result**: Task marked "blocked" with validation error

## Testing Plan

1. Run delete button task again
2. Verify it gets marked as "blocked" instead of "completed"
3. Check logs for validation failure message
4. Confirm git status shows no changes
5. Verify task notes explain why it failed

## Security Considerations

- Git commands run with proper error handling
- File paths validated against expected list
- No arbitrary command execution
- All validation logged for audit

## Backwards Compatibility

- Existing completed tasks unaffected
- Only new completions validated
- Graceful degradation if git unavailable (log warning, allow completion)
- Can be disabled via feature flag if needed

## Related Files

- `hyper/internal/ai-service/tools/mcp/coordinator_tools.go` (PRIMARY FIX)
- `hyper/internal/mcp/storage/tasks.go` (task status updates)
- `hyper/internal/mcp/storage/subchat_storage.go` (subchat status updates)
