# Error Prevention Mode - Toggleable Plugin Design

**Date**: 2025-11-21
**Status**: Design Phase
**Branch**: megha/subagent-subchat
**Source Code**: megha/agent-task-splitting (validation system)

## Overview

Make the validation workflow from `megha/agent-task-splitting` a **toggleable plugin** that can be switched on/off via UI. When OFF, the code should behave as if the validation system doesn't exist (for easier debugging).

---

## Architecture Design

### 1. **Feature Flag Storage**

**Location**: `ChatSession` model (per-session basis)

```go
// In hyper/internal/models/chat.go
type ChatSession struct {
    ID                  primitive.ObjectID  `bson:"_id" json:"id"`
    UserID              string              `bson:"userId" json:"userId"`
    CompanyID           string              `bson:"companyId" json:"companyId"`
    Title               string              `bson:"title" json:"title"`
    ParentChatID        *string             `bson:"parentChatId,omitempty" json:"parentChatId,omitempty"`
    ActiveSubagentID    *primitive.ObjectID `bson:"activeSubagentId,omitempty" json:"activeSubagentId,omitempty"`
    ActiveSubagentName  *string             `bson:"activeSubagentName,omitempty" json:"activeSubagentName,omitempty"`

    // ✅ NEW: Error Prevention Mode toggle
    ErrorPreventionMode bool                `bson:"errorPreventionMode" json:"errorPreventionMode"`

    CreatedAt           time.Time           `bson:"createdAt" json:"createdAt"`
    UpdatedAt           time.Time           `bson:"updatedAt" json:"updatedAt"`
}
```

**Why per-session?**
- Different chat sessions may need different modes (debugging vs production)
- Easy to toggle on/off without affecting other chats
- Persists across page refreshes

---

### 2. **Validation Module Structure**

```
hyper/internal/validation/
├── code_validator.go       # Core validation logic (from megha/agent-task-splitting)
├── plugin.go               # Plugin interface and feature flag checks
└── integration.go          # Integration hooks (post-write, pre-complete)
```

**Plugin Interface** (`plugin.go`):
```go
package validation

type ValidationPlugin struct {
    validator *CodeValidator
    enabled   bool
}

func NewValidationPlugin(validator *CodeValidator) *ValidationPlugin {
    return &ValidationPlugin{
        validator: validator,
        enabled:   false, // Default: OFF
    }
}

func (p *ValidationPlugin) SetEnabled(enabled bool) {
    p.enabled = enabled
}

func (p *ValidationPlugin) IsEnabled() bool {
    return p.enabled
}

// ValidateIfEnabled runs validation only if plugin is enabled
func (p *ValidationPlugin) ValidateIfEnabled(ctx context.Context, files []string) (*ValidationResult, error) {
    if !p.enabled {
        // Plugin disabled - return success without running validation
        return &ValidationResult{Passed: true, Skipped: true}, nil
    }
    return p.validator.ValidateFiles(ctx, files)
}
```

---

### 3. **Backend Integration Points**

#### A. **Session Creation/Update**

**API Endpoint**: `PATCH /api/v1/chat/sessions/:id/error-prevention`

```go
// In hyper/internal/handlers/chat_handler.go

type UpdateErrorPreventionRequest struct {
    Enabled bool `json:"enabled"`
}

func (h *ChatHandler) UpdateErrorPreventionMode(c *gin.Context) {
    sessionID := c.Param("id")

    var req UpdateErrorPreventionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // Update session
    objID, _ := primitive.ObjectIDFromHex(sessionID)
    err := h.chatService.UpdateErrorPreventionMode(c.Request.Context(), objID, req.Enabled)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "success": true,
        "errorPreventionMode": req.Enabled,
    })
}
```

#### B. **File Write Hook** (Conditional Validation)

**Location**: `hyper/internal/mcp/handlers/filesystem_tools.go`

```go
// After file write succeeds
if !dryRun {
    err = os.WriteFile(validatedPath, content, 0644)
    if err != nil {
        return createErrorResult(err), nil
    }

    // ✅ NEW: Only run validation if Error Prevention Mode is ON
    if h.isErrorPreventionEnabled(ctx) {
        go h.runPostWriteValidation(ctx, []string{validatedPath})
    }
    // Otherwise, skip validation entirely
}
```

#### C. **Completion Check Hook** (Conditional Blocking)

**Location**: `hyper/internal/mcp/handlers/task_handler.go`

```go
func (h *TaskHandler) canCompleteTask(ctx context.Context, taskID string) (bool, error) {
    // ✅ NEW: Only enforce validation if Error Prevention Mode is ON
    if !h.isErrorPreventionEnabled(ctx) {
        return true, nil // Skip validation checks
    }

    // Run validation checks only when enabled
    result, err := h.validationPlugin.ValidateIfEnabled(ctx, task.FilesModified)
    if err != nil {
        return false, err
    }

    return result.Passed, nil
}
```

---

### 4. **UI Toggle Button**

**Location**: `ui/src/pages/CodeChatPage.tsx`

**Design**:
```tsx
// Add to page header, next to other controls
<IconButton
  onClick={toggleErrorPrevention}
  color={errorPreventionMode ? "primary" : "default"}
  title={errorPreventionMode ? "Error Prevention: ON" : "Error Prevention: OFF"}
>
  <Badge
    color={errorPreventionMode ? "success" : "default"}
    variant="dot"
  >
    {errorPreventionMode ? <ShieldCheckIcon /> : <ShieldOffIcon />}
  </Badge>
</IconButton>
```

**State Management**:
```tsx
const [errorPreventionMode, setErrorPreventionMode] = useState(false);

// Load from session on mount
useEffect(() => {
  if (activeSessionId) {
    loadSession(activeSessionId).then(session => {
      setErrorPreventionMode(session.errorPreventionMode || false);
    });
  }
}, [activeSessionId]);

// Toggle handler
const toggleErrorPrevention = async () => {
  const newMode = !errorPreventionMode;

  try {
    await updateErrorPreventionMode(activeSessionId, newMode);
    setErrorPreventionMode(newMode);

    // Show notification
    showNotification(
      newMode
        ? "Error Prevention Mode: ON - AI will validate code before completion"
        : "Error Prevention Mode: OFF - Validation disabled for debugging"
    );
  } catch (error) {
    console.error("Failed to toggle error prevention mode", error);
  }
};
```

**API Service** (`ui/src/services/chatService.ts`):
```ts
export async function updateErrorPreventionMode(
  sessionId: string,
  enabled: boolean
): Promise<void> {
  await fetchJSON(`/chat/sessions/${sessionId}/error-prevention`, {
    method: 'PATCH',
    body: JSON.stringify({ enabled }),
  });
}
```

---

### 5. **Context Propagation**

**Problem**: How does the validation plugin know if mode is ON or OFF?

**Solution**: Pass session context through the request chain

```go
// 1. At WebSocket handler entry point
sessionID := getSessionIDFromRequest(req)
session, _ := h.chatService.GetSession(ctx, sessionID)

// 2. Store in context
ctx = context.WithValue(ctx, "errorPreventionMode", session.ErrorPreventionMode)

// 3. Retrieve in validation hooks
func (h *FilesystemToolHandler) isErrorPreventionEnabled(ctx context.Context) bool {
    if mode, ok := ctx.Value("errorPreventionMode").(bool); ok {
        return mode
    }
    return false // Default: OFF
}
```

---

## Implementation Plan

### Phase 1: Backend Infrastructure ✅
1. Copy validation code from `megha/agent-task-splitting`
   - `internal/validation/code_validator.go`
   - `internal/validation/plugin.go` (new wrapper)
2. Add `ErrorPreventionMode` field to `ChatSession` model
3. Create API endpoint for toggling mode
4. Add context propagation helpers

### Phase 2: Hook Integration 🔄
5. Add conditional validation to file write hooks
6. Add conditional blocking to task completion
7. Ensure all validation code is wrapped in `if (enabled)` checks

### Phase 3: UI Toggle 🎨
8. Add toggle button to `CodeChatPage.tsx`
9. Add API service method
10. Implement state management
11. Add visual feedback (badge, notifications)

### Phase 4: Testing ✅
12. Test with mode ON: validation runs, errors block completion
13. Test with mode OFF: validation skipped, no blocking
14. Verify no performance impact when OFF

---

## Behavior Comparison

### When Error Prevention Mode is **ON** ✅

```
1. User asks: "Add dark mode toggle"
2. AI modifies Settings.tsx
3. ✅ POST-WRITE: Auto-validate TypeScript
4. ❌ ERRORS FOUND: "Cannot find name 'darkModeState'"
5. 🔒 INJECT TODO: "Fix validation errors (mandatory)"
6. 🔄 AI fixes errors automatically
7. ✅ VALIDATION PASSES
8. ✅ Task marked complete
```

### When Error Prevention Mode is **OFF** 🔓

```
1. User asks: "Add dark mode toggle"
2. AI modifies Settings.tsx
3. ⏭️  SKIP: No validation run
4. ⏭️  SKIP: No mandatory TODOs injected
5. ✅ Task marked complete immediately
6. (User can debug manually if needed)
```

---

## Key Design Principles

1. **Zero Overhead When OFF**
   - No validation code runs
   - No database queries for validation results
   - No performance impact

2. **Graceful Degradation**
   - If plugin fails to load → mode stays OFF
   - If toggle API fails → keep current state
   - No crashes, just log warnings

3. **Per-Session Granularity**
   - Each chat session has independent mode
   - Useful for debugging specific conversations
   - Doesn't affect other sessions

4. **Explicit User Control**
   - Visible toggle button with clear state
   - Toast notifications on toggle
   - Badge indicator always visible

5. **Debuggability**
   - When OFF, behaves like validation code doesn't exist
   - No logs, no checks, no overhead
   - Clean separation of concerns

---

## Migration Strategy

### Bringing Code from `megha/agent-task-splitting`

```bash
# Cherry-pick validation files
git checkout megha/agent-task-splitting -- hyper/internal/validation/

# Review and adapt for plugin architecture
# Add feature flag checks
# Wrap all hooks with isEnabled() checks

# Test thoroughly before merging
```

### Database Migration

```go
// No migration needed!
// New field `errorPreventionMode` defaults to false (zero value)
// Existing sessions will have mode OFF by default
```

---

## Future Enhancements

1. **User Preferences**: Remember mode per user (not just per session)
2. **Auto-Enable**: Suggest enabling mode when errors detected
3. **Metrics**: Track how often mode prevents bad commits
4. **Team Settings**: Admin can enforce mode ON for all users
5. **Granular Control**: Toggle specific validators (TypeScript only, Go only, etc.)

---

## Files to Modify

### Backend (Go)
1. `hyper/internal/models/chat.go` - Add ErrorPreventionMode field
2. `hyper/internal/handlers/chat_handler.go` - Add toggle endpoint
3. `hyper/internal/services/chat_service.go` - Add UpdateErrorPreventionMode method
4. `hyper/internal/validation/code_validator.go` - Copy from other branch
5. `hyper/internal/validation/plugin.go` - NEW: Plugin wrapper
6. `hyper/internal/mcp/handlers/filesystem_tools.go` - Add conditional hooks
7. `hyper/internal/mcp/handlers/task_handler.go` - Add conditional completion checks
8. `hyper/internal/handlers/chat_websocket.go` - Add context propagation

### Frontend (TypeScript)
9. `ui/src/pages/CodeChatPage.tsx` - Add toggle button
10. `ui/src/services/chatService.ts` - Add API method
11. `ui/src/types/chat.ts` - Add errorPreventionMode field

### Documentation
12. `README.md` - Add Error Prevention Mode section
13. `docs/ERROR_PREVENTION_PLUGIN_DESIGN.md` - This file

---

## Success Criteria

✅ Toggle button visible and functional in UI
✅ Mode ON: Validation runs, errors block completion
✅ Mode OFF: No validation code executes at all
✅ State persists across page refreshes
✅ No performance impact when mode is OFF
✅ Clean code separation (plugin pattern)
✅ Works with existing subagent workflows

---

## Questions & Decisions

**Q**: Should mode default to ON or OFF?
**A**: OFF. User opt-in for error prevention, opt-out for debugging.

**Q**: Per-session or per-user setting?
**A**: Per-session for flexibility. Can add per-user later.

**Q**: What if validation crashes?
**A**: Catch errors, log warning, let task proceed. Never block on validation failures.

**Q**: Should validation run in background?
**A**: Yes (`go h.runValidation()`). Don't block writes waiting for validation.

---

**END OF DESIGN**
