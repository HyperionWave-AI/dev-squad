# System Prompt & Sub-Agents - Implementation Plan

**Date:** October 12, 2025
**Full Design:** See `SYSTEM_PROMPT_SUBAGENTS_DESIGN.md`

---

## Quick Summary

**Goal:** Enable users to customize AI behavior with system prompts and create specialized sub-agents.

**Current State:**
- ✅ Backend API endpoints (CRUD) - **FULLY IMPLEMENTED**
- ✅ Database models and storage - **FULLY IMPLEMENTED**
- ✅ Routes registered - **FULLY IMPLEMENTED**
- ❌ Chat integration (prompt injection) - **NOT IMPLEMENTED**
- ❌ Frontend UI - **NOT IMPLEMENTED**

---

## Implementation Tasks

### Phase 1: Backend Chat Integration (Priority: HIGH)

**Assigned To:** `go-dev` sub-agent
**Estimated Time:** 2-3 hours
**Files to Modify:** 4 files

#### Task 1.1: Enhance ChatSession Model
**File:** `internal/models/chat.go`
```go
// Add this field to ChatSession struct:
ActiveSubagentID *primitive.ObjectID `json:"activeSubagentId,omitempty" bson:"activeSubagentId,omitempty"`
```

#### Task 1.2: Add SetSessionSubagent Method
**File:** `internal/services/chat_service.go`
```go
// Add new method:
func (s *ChatService) SetSessionSubagent(ctx context.Context, sessionID primitive.ObjectID, subagentID *primitive.ObjectID, companyID string) error {
    // Update session with activeSubagentId
    // Verify session exists and user has access
}
```

#### Task 1.3: Enhance ChatWebSocketHandler
**File:** `internal/handlers/chat_websocket.go`

**Changes:**
1. Add `aiSettingsService` field to struct
2. Update `streamAIResponse()` method:
   ```go
   func (h *ChatWebSocketHandler) streamAIResponse(ctx, conn, session, userMessage, userID, companyID) {
       // NEW: Fetch system prompt or subagent prompt
       var systemPromptText string
       if session.ActiveSubagentID != nil {
           subagent, _ := h.aiSettingsService.GetSubagent(ctx, *session.ActiveSubagentID, companyID)
           systemPromptText = subagent.SystemPrompt
       } else {
           systemPromptText, _ = h.aiSettingsService.GetSystemPrompt(ctx, userID, companyID)
       }

       // Get conversation history
       messages := h.chatService.GetSessionMessages(ctx, sessionID)
       langchainMessages := aiservice.ConvertToLangChainMessages(messages)

       // NEW: Inject system prompt as first message
       if systemPromptText != "" {
           systemMessage := langchain.SystemMessage{Content: systemPromptText}
           langchainMessages = append([]langchain.Message{systemMessage}, langchainMessages...)
       }

       // Stream AI response...
   }
   ```

#### Task 1.4: Add REST Endpoint
**File:** `internal/handlers/chat_handler.go`

**New Endpoint:**
```go
// PUT /api/v1/chat/sessions/:id/subagent
func (h *ChatHandler) SetSessionSubagent(c *gin.Context) {
    // Extract sessionID from URL
    // Parse request body: {"subagentId": "xxx" or null}
    // Call chatService.SetSessionSubagent()
    // Return success response
}
```

**Register Route:**
```go
func (h *ChatHandler) RegisterChatRoutes(r *gin.RouterGroup) {
    // ... existing routes ...
    r.PUT("/sessions/:id/subagent", h.SetSessionSubagent)
}
```

#### Task 1.5: Update HTTP Server Init
**File:** `internal/server/http_server.go`

**Change Line 124:**
```go
// BEFORE:
chatWebSocketHandler := handlers.NewChatWebSocketHandler(chatService, aiChatService, logger)

// AFTER:
chatWebSocketHandler := handlers.NewChatWebSocketHandler(chatService, aiChatService, aiSettingsService, logger)
```

**Testing Checklist:**
- [ ] Unit test: SetSessionSubagent method
- [ ] Unit test: System prompt injection logic
- [ ] Integration test: Chat with system prompt
- [ ] Integration test: Chat with subagent
- [ ] Integration test: Switch subagent mid-session

---

### Phase 2: Frontend UI - System Prompt (Priority: HIGH)

**Assigned To:** `ui-dev` sub-agent
**Estimated Time:** 3-4 hours

#### Task 2.1: Create Settings Page
**New Files:**
- `ui/src/pages/SettingsPage.tsx`
- `ui/src/components/SystemPromptEditor.tsx`

**Features:**
- Textarea for system prompt (max 10,000 chars)
- Character counter
- Save button
- Success/error toasts

#### Task 2.2: API Client
**File:** `ui/src/services/api.ts`

**Add Methods:**
```typescript
export async function getSystemPrompt(): Promise<string> {
    const response = await fetch('/api/v1/ai/system-prompt', {
        headers: { 'Authorization': `Bearer ${getToken()}` }
    });
    const data = await response.json();
    return data.systemPrompt;
}

export async function updateSystemPrompt(prompt: string): Promise<void> {
    await fetch('/api/v1/ai/system-prompt', {
        method: 'PUT',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${getToken()}`
        },
        body: JSON.stringify({ systemPrompt: prompt })
    });
}
```

#### Task 2.3: Add Navigation Link
**File:** `ui/src/components/Navigation.tsx`

**Add:**
```tsx
<Link to="/settings">Settings</Link>
```

**Testing Checklist:**
- [ ] Component test: SystemPromptEditor
- [ ] E2E test: View system prompt
- [ ] E2E test: Edit and save system prompt
- [ ] E2E test: Character counter
- [ ] E2E test: Error handling

---

### Phase 3: Frontend UI - Subagents Management (Priority: HIGH)

**Assigned To:** `ui-dev` sub-agent
**Estimated Time:** 4-5 hours

#### Task 3.1: Create Subagents Page
**New Files:**
- `ui/src/pages/SubagentsPage.tsx`
- `ui/src/components/SubagentsList.tsx`
- `ui/src/components/SubagentForm.tsx`
- `ui/src/components/SubagentCard.tsx`

**Features:**
- List view with all subagents
- Create button → modal form
- Edit button → modal form
- Delete button → confirm dialog
- Search/filter

#### Task 3.2: API Client
**File:** `ui/src/services/api.ts`

**Add Methods:**
```typescript
export async function listSubagents(): Promise<Subagent[]>
export async function getSubagent(id: string): Promise<Subagent>
export async function createSubagent(data: CreateSubagentRequest): Promise<Subagent>
export async function updateSubagent(id: string, data: UpdateSubagentRequest): Promise<Subagent>
export async function deleteSubagent(id: string): Promise<void>
```

#### Task 3.3: Form Validation
- Name: 3-50 chars, required
- Description: max 200 chars, optional
- System Prompt: required, max 10,000 chars

**Testing Checklist:**
- [ ] Component test: SubagentsList
- [ ] Component test: SubagentForm
- [ ] E2E test: List subagents
- [ ] E2E test: Create subagent
- [ ] E2E test: Edit subagent
- [ ] E2E test: Delete subagent
- [ ] E2E test: Form validation

---

### Phase 4: Frontend UI - Chat Integration (Priority: HIGH)

**Assigned To:** `ui-dev` sub-agent
**Estimated Time:** 4-5 hours

#### Task 4.1: Add Agent Selector
**File:** `ui/src/components/ChatPage.tsx`

**New Component:**
- `ui/src/components/AgentSelector.tsx`

**Features:**
- Dropdown with: "Default AI" + list of user's subagents
- Call API to fetch subagents
- Call `PUT /api/v1/chat/sessions/:id/subagent` on change
- Visual indicator (badge, icon) showing active agent

#### Task 4.2: Chat History Metadata
**Update:**
- Show which agent responded (icon, name)
- Different styling for default vs subagent

**Testing Checklist:**
- [ ] Component test: AgentSelector
- [ ] E2E test: Select agent
- [ ] E2E test: Send message with agent
- [ ] E2E test: Switch agent mid-session
- [ ] E2E test: Visual indicator

---

### Phase 5: Testing & QA (Priority: MEDIUM)

**Assigned To:** `ui-tester` sub-agent
**Estimated Time:** 3-4 hours

#### Task 5.1: Backend Tests
- [ ] Integration test: Full chat flow with system prompt
- [ ] Integration test: Full chat flow with subagent
- [ ] Security test: JWT validation
- [ ] Security test: Data isolation

#### Task 5.2: Frontend Tests
- [ ] E2E test: System prompt management flow
- [ ] E2E test: Subagent management flow
- [ ] E2E test: Chat with agent selection
- [ ] E2E test: Agent switching

#### Task 5.3: Performance Tests
- [ ] Benchmark system prompt injection
- [ ] Test concurrent sessions

---

## Coordination via Hyperion MCP

### For Backend Work (go-dev):
```typescript
// Create human task
const humanTask = await mcp__hyper__coordinator_create_human_task({
  prompt: "Integrate system prompts and subagents into chat service"
});

// Create agent task
const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "go-dev",
  role: "Integrate system prompt and subagent selection in chat WebSocket handler",
  contextSummary: `
    Backend APIs exist. Need to integrate prompt injection into chat flow.
    Files: chat.go, chat_service.go, chat_websocket.go, chat_handler.go, http_server.go
    Pattern: Fetch prompt from aiSettingsService, inject as first "system" message.
  `,
  filesModified: [
    "internal/models/chat.go",
    "internal/services/chat_service.go",
    "internal/handlers/chat_websocket.go",
    "internal/handlers/chat_handler.go",
    "internal/server/http_server.go"
  ],
  todos: [
    {
      description: "Add ActiveSubagentID field to ChatSession model",
      filePath: "internal/models/chat.go",
      contextHint: "Add optional field after CompanyID. Use pointer for nullable ObjectID."
    },
    {
      description: "Add SetSessionSubagent method to ChatService",
      filePath: "internal/services/chat_service.go",
      functionName: "SetSessionSubagent",
      contextHint: "Update MongoDB with $set. Filter by sessionID and companyID. Return error if not found."
    },
    // ... more TODOs
  ]
});
```

### For Frontend Work (ui-dev):
```typescript
const agentTask = await mcp__hyper__coordinator_create_agent_task({
  humanTaskId: humanTask.taskId,
  agentName: "ui-dev",
  role: "Create UI for system prompt management and subagent CRUD",
  contextSummary: `
    Backend APIs exist at /api/v1/ai/*. Need React components for CRUD.
    Pattern: React 18 + TypeScript, atomic design, API client in services/api.ts
  `,
  filesModified: [
    "ui/src/pages/SettingsPage.tsx",
    "ui/src/components/SystemPromptEditor.tsx",
    "ui/src/pages/SubagentsPage.tsx",
    "ui/src/services/api.ts"
  ],
  todos: [
    // ... TODOs
  ]
});
```

---

## Dependencies

### Phase 1: Backend Chat Integration
- **Depends On:** None (APIs already exist)
- **Blocks:** Phase 4 (chat integration)

### Phase 2: Frontend - System Prompt
- **Depends On:** None (APIs already exist)
- **Blocks:** None

### Phase 3: Frontend - Subagents
- **Depends On:** None (APIs already exist)
- **Blocks:** Phase 4 (chat integration)

### Phase 4: Frontend - Chat Integration
- **Depends On:** Phase 1 (backend), Phase 3 (subagents list)
- **Blocks:** None

### Phase 5: Testing & QA
- **Depends On:** All phases
- **Blocks:** Production deployment

---

## Timeline (2 Weeks)

```
Week 1:
  Mon-Tue: Phase 1 (Backend Integration) - go-dev
  Wed-Thu: Phase 2 + 3 (Frontend UI) - ui-dev
  Fri: Phase 4 (Chat Integration) - ui-dev

Week 2:
  Mon-Tue: Phase 5 (Testing & QA) - ui-tester
  Wed: Bug fixes and polishing
  Thu: Production deployment
```

---

## Success Criteria

- [x] Backend APIs exist (DONE)
- [ ] System prompts are injected into chat
- [ ] Subagents can be created and managed
- [ ] Users can select subagent in chat
- [ ] All tests passing
- [ ] Zero critical bugs
- [ ] Documentation complete

---

## Next Steps

1. **User Approval**: Get user confirmation to proceed with implementation
2. **Launch Phase 1**: Create agent task for go-dev to integrate backend
3. **Launch Phase 2**: Create agent task for ui-dev to build settings UI
4. **Launch Phase 3**: Create agent task for ui-dev to build subagents UI
5. **Launch Phase 4**: Create agent task for ui-dev to integrate chat
6. **Launch Phase 5**: Create agent task for ui-tester for QA

---

**Document Version:** 1.0
**Last Updated:** October 12, 2025
**Status:** Ready for Implementation
