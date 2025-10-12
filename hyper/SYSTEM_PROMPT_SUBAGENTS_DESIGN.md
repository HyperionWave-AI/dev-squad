# System Prompt & Sub-Agents Feature - Architecture Design Document (ADR)

**Date:** October 12, 2025
**Status:** Design Phase
**Decision:** Implement system prompt and sub-agent management for AI chat customization

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Current State Analysis](#current-state-analysis)
3. [Feature Requirements](#feature-requirements)
4. [Architecture Design](#architecture-design)
5. [Implementation Plan](#implementation-plan)
6. [API Contracts](#api-contracts)
7. [Database Schema](#database-schema)
8. [UI Design](#ui-design)
9. [Security & Authorization](#security--authorization)
10. [Testing Strategy](#testing-strategy)
11. [Rollout Plan](#rollout-plan)

---

## Executive Summary

This ADR documents the design and implementation plan for **System Prompt** and **Sub-Agent** management features in the Hyperion AI Platform. These features enable users to:

1. **System Prompts**: Customize the global AI behavior with user-defined instructions
2. **Sub-Agents**: Create specialized AI agents with custom system prompts and invoke them during chat sessions

**Key Benefits:**
- Personalized AI interactions
- Task-specific agents (e.g., "Code Reviewer", "Documentation Writer", "DevOps Helper")
- Company-level customization
- Enhanced productivity through context switching

---

## Current State Analysis

### ✅ What's Already Implemented

#### 1. Backend Data Models (`internal/models/ai_settings.go`)
```go
type SystemPrompt struct {
    ID        primitive.ObjectID
    UserID    string
    CompanyID string
    Prompt    string
    UpdatedAt time.Time
}

type Subagent struct {
    ID           primitive.ObjectID
    UserID       string
    CompanyID    string
    Name         string
    Description  string
    SystemPrompt string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

#### 2. Backend Service Layer (`internal/services/ai_settings_service.go`)
- `AISettingsService` with full CRUD operations:
  - `GetSystemPrompt(userID, companyID) string`
  - `UpdateSystemPrompt(userID, companyID, prompt)`
  - `ListSubagents(userID, companyID) []Subagent`
  - `GetSubagent(id, companyID) *Subagent`
  - `CreateSubagent(name, description, systemPrompt)`
  - `UpdateSubagent(id, name, description, systemPrompt)`
  - `DeleteSubagent(id)`
- MongoDB storage with proper indexes
- Company-level data isolation

#### 3. Backend API Handlers (`internal/handlers/ai_settings_handler.go`)
- REST endpoints:
  - `GET /api/v1/ai/system-prompt` - Get user's system prompt
  - `PUT /api/v1/ai/system-prompt` - Update system prompt
  - `GET /api/v1/ai/subagents` - List all subagents
  - `GET /api/v1/ai/subagents/:id` - Get specific subagent
  - `POST /api/v1/ai/subagents` - Create subagent
  - `PUT /api/v1/ai/subagents/:id` - Update subagent
  - `DELETE /api/v1/ai/subagents/:id` - Delete subagent
- JWT authentication integrated
- Proper error handling

#### 4. Route Registration (`internal/server/http_server.go:199-207`)
- API routes registered and active
- Logging configured

### ❌ What's Missing

#### 1. **Chat Service Integration**
- Chat WebSocket handler (`internal/handlers/chat_websocket.go:228-256`) does NOT fetch or use system prompt
- Current flow:
  ```go
  messages, err := h.chatService.GetSessionMessages(ctx, sessionID)
  langchainMessages := aiservice.ConvertToLangChainMessages(messages)
  aiStream, err := h.aiService.StreamChatWithTools(ctx, langchainMessages, maxToolCalls)
  ```
- **Missing:** System prompt injection before streaming

#### 2. **Sub-Agent Invocation Mechanism**
- No way to select/invoke a specific subagent during chat
- No UI controls to switch between default AI and custom subagents
- No subagent metadata in chat session

#### 3. **Frontend UI Components**
- No UI for managing system prompts
- No UI for creating/editing subagents
- No UI for invoking subagents in chat
- UI source code not in this repository (only prebuilt assets in `embed/ui`)

#### 4. **Integration Points**
- AI service doesn't accept system prompt parameter
- Chat sessions don't track active subagent
- No visual indicator of active system prompt/subagent

---

## Feature Requirements

### FR1: System Prompt Management

**User Stories:**
- As a user, I want to set a global system prompt that applies to all my chat sessions
- As a user, I want to see my current system prompt before editing
- As a user, I want to clear my system prompt to revert to default behavior

**Acceptance Criteria:**
- [x] Backend API for CRUD operations (DONE)
- [ ] System prompt is injected into every chat message
- [ ] System prompt persists across sessions
- [ ] UI to view/edit system prompt
- [ ] Validation: max 10,000 characters

### FR2: Sub-Agent Creation & Management

**User Stories:**
- As a user, I want to create specialized agents for specific tasks (e.g., "DevOps Helper", "Code Reviewer")
- As a user, I want to provide a custom system prompt for each agent
- As a user, I want to edit or delete my custom agents
- As a user, I want to see a list of all my agents

**Acceptance Criteria:**
- [x] Backend API for CRUD operations (DONE)
- [ ] UI to create/edit/delete subagents
- [ ] Name validation: 3-50 characters, alphanumeric + spaces
- [ ] Description: optional, max 200 characters
- [ ] System prompt: required, max 10,000 characters
- [ ] List view with search/filter

### FR3: Sub-Agent Invocation in Chat

**User Stories:**
- As a user, I want to select a specific subagent before sending a message
- As a user, I want to see which agent is currently active
- As a user, I want to switch between default AI and custom agents mid-conversation
- As a user, I want the chat history to show which agent responded

**Acceptance Criteria:**
- [ ] Dropdown/selector in chat UI to choose agent
- [ ] Visual indicator of active agent (badge, avatar, color)
- [ ] Session metadata tracks active subagent
- [ ] Chat history shows which agent responded to each message
- [ ] Subagent's system prompt overrides global system prompt

---

## Architecture Design

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (UI)                            │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────────┐ │
│  │ Chat Page   │  │ Settings Page│  │ Subagents Management   │ │
│  │ - Agent     │  │ - System     │  │ - Create/Edit/Delete   │ │
│  │   Selector  │  │   Prompt     │  │ - List View            │ │
│  └──────┬──────┘  └──────┬───────┘  └───────────┬────────────┘ │
└─────────┼─────────────────┼──────────────────────┼──────────────┘
          │                 │                      │
          │    WebSocket    │     REST API         │    REST API
          │                 │                      │
┌─────────▼─────────────────▼──────────────────────▼──────────────┐
│                    Backend (Coordinator)                         │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │          Chat WebSocket Handler (ENHANCED)               │   │
│  │  1. Fetch active subagent (or use default)              │   │
│  │  2. Fetch system prompt or subagent's prompt            │   │
│  │  3. Inject prompt as first "system" message              │   │
│  │  4. Stream AI response                                   │   │
│  └─────────────────────┬────────────────────────────────────┘   │
│                        │                                         │
│  ┌─────────────────────▼─────────────────┐                      │
│  │      AI Settings Service               │                      │
│  │  - GetSystemPrompt(userId, companyId)  │                      │
│  │  - GetSubagent(id, companyId)          │                      │
│  └────────────────────┬───────────────────┘                      │
│                       │                                          │
│  ┌────────────────────▼────────────────────┐                     │
│  │         MongoDB Collections             │                     │
│  │  - system_prompts (userId, prompt)      │                     │
│  │  - subagents (userId, name, prompt)     │                     │
│  │  - chat_sessions (ENHANCED: +subagentId)│                     │
│  └─────────────────────────────────────────┘                     │
└──────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

#### 1. Frontend (UI)
- **Chat Page**: Agent selector, visual indicator, send messages with subagent context
- **Settings Page**: System prompt editor (textarea, save button, character counter)
- **Subagents Page**: CRUD interface (table, forms, modals)

#### 2. Backend - Chat WebSocket Handler (ENHANCED)
**File:** `internal/handlers/chat_websocket.go`

**Current Flow:**
```go
func (h *ChatWebSocketHandler) streamAIResponse(ctx, conn, sessionID, userMessage, companyID) {
    messages := h.chatService.GetSessionMessages(ctx, sessionID)
    langchainMessages := aiservice.ConvertToLangChainMessages(messages)
    aiStream := h.aiService.StreamChatWithTools(ctx, langchainMessages, maxToolCalls)
    // Stream response...
}
```

**Enhanced Flow:**
```go
func (h *ChatWebSocketHandler) streamAIResponse(ctx, conn, session, userMessage, userID, companyID) {
    // NEW: Determine active agent and fetch prompt
    var systemPromptText string
    if session.ActiveSubagentID != nil {
        // Using custom subagent
        subagent, err := h.aiSettingsService.GetSubagent(ctx, *session.ActiveSubagentID, companyID)
        if err == nil {
            systemPromptText = subagent.SystemPrompt
        }
    } else {
        // Using default AI - fetch global system prompt
        systemPromptText, _ = h.aiSettingsService.GetSystemPrompt(ctx, userID, companyID)
    }

    // Retrieve conversation history
    messages := h.chatService.GetSessionMessages(ctx, sessionID)
    langchainMessages := aiservice.ConvertToLangChainMessages(messages)

    // NEW: Inject system prompt as first message (if exists)
    if systemPromptText != "" {
        systemMessage := langchain.SystemMessage{Content: systemPromptText}
        langchainMessages = append([]langchain.Message{systemMessage}, langchainMessages...)
    }

    // Stream AI response with system prompt context
    aiStream := h.aiService.StreamChatWithTools(ctx, langchainMessages, maxToolCalls)
    // Stream response...
}
```

#### 3. Backend - Chat Session Model (ENHANCED)
**File:** `internal/models/chat.go`

**Add Field:**
```go
type ChatSession struct {
    ID               primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
    UserID           string              `json:"userId" bson:"userId"`
    CompanyID        string              `json:"companyId" bson:"companyId"`
    Title            string              `json:"title" bson:"title"`
    ActiveSubagentID *primitive.ObjectID `json:"activeSubagentId,omitempty" bson:"activeSubagentId,omitempty"` // NEW
    CreatedAt        time.Time           `json:"createdAt" bson:"createdAt"`
    UpdatedAt        time.Time           `json:"updatedAt" bson:"updatedAt"`
}
```

#### 4. Backend - Chat Service (ENHANCED)
**File:** `internal/services/chat_service.go`

**New Method:**
```go
// SetSessionSubagent sets or clears the active subagent for a session
func (s *ChatService) SetSessionSubagent(ctx context.Context, sessionID primitive.ObjectID, subagentID *primitive.ObjectID, companyID string) error {
    filter := bson.M{
        "_id":       sessionID,
        "companyId": companyID,
    }
    update := bson.M{
        "$set": bson.M{
            "activeSubagentId": subagentID,
            "updatedAt":        time.Now().UTC(),
        },
    }
    result, err := s.sessionsCollection.UpdateOne(ctx, filter, update)
    if err != nil {
        return fmt.Errorf("failed to update session subagent: %w", err)
    }
    if result.MatchedCount == 0 {
        return fmt.Errorf("session not found or access denied")
    }
    return nil
}
```

---

## Implementation Plan

### Phase 1: Backend Integration (Priority: HIGH)

**Squad:** `go-dev`
**Estimated Time:** 2-3 hours

**Tasks:**
1. **Enhance ChatSession Model**
   - Add `ActiveSubagentID *primitive.ObjectID` field to `models.ChatSession`
   - File: `internal/models/chat.go`

2. **Enhance ChatService**
   - Add `SetSessionSubagent()` method
   - File: `internal/services/chat_service.go`
   - Add database migration if needed (MongoDB index)

3. **Enhance ChatWebSocketHandler**
   - Inject `aiSettingsService` dependency
   - Modify `streamAIResponse()` to fetch and inject system prompt
   - File: `internal/handlers/chat_websocket.go`
   - Logic:
     ```go
     if session.ActiveSubagentID != nil {
         subagent, _ := h.aiSettingsService.GetSubagent(...)
         systemPrompt = subagent.SystemPrompt
     } else {
         systemPrompt, _ = h.aiSettingsService.GetSystemPrompt(...)
     }
     ```

4. **Add REST Endpoint for Setting Session Subagent**
   - New endpoint: `PUT /api/v1/chat/sessions/:id/subagent`
   - Request body: `{"subagentId": "ObjectID or null"}`
   - Handler: `internal/handlers/chat_handler.go`

5. **Update HTTP Server Initialization**
   - Pass `aiSettingsService` to `ChatWebSocketHandler` constructor
   - File: `internal/server/http_server.go:124`

**Acceptance Criteria:**
- [ ] Chat sessions can have `activeSubagentId` field
- [ ] System prompt is fetched and injected as first "system" message
- [ ] Subagent's prompt overrides global system prompt
- [ ] REST endpoint to set/clear session subagent
- [ ] All existing tests pass
- [ ] Integration test: chat with system prompt works
- [ ] Integration test: chat with subagent works

---

### Phase 2: Frontend UI - System Prompt (Priority: HIGH)

**Squad:** `ui-dev`
**Estimated Time:** 3-4 hours

**Tasks:**
1. **Create Settings Page**
   - Route: `/ui/settings`
   - Components:
     - `SystemPromptEditor.tsx` - Textarea with save button
     - Character counter (max 10,000)
     - Success/error toast notifications

2. **API Client Integration**
   - Add API methods to `services/api.ts`:
     ```typescript
     export async function getSystemPrompt(): Promise<string>
     export async function updateSystemPrompt(prompt: string): Promise<void>
     ```

3. **Navigation**
   - Add "Settings" link to main navigation/sidebar

**Acceptance Criteria:**
- [ ] User can view current system prompt
- [ ] User can edit and save system prompt
- [ ] Character counter shows remaining characters
- [ ] Success toast on save
- [ ] Error handling for API failures
- [ ] Loading states

---

### Phase 3: Frontend UI - Subagents Management (Priority: HIGH)

**Squad:** `ui-dev`
**Estimated Time:** 4-5 hours

**Tasks:**
1. **Create Subagents Page**
   - Route: `/ui/subagents`
   - Components:
     - `SubagentsList.tsx` - Table/grid of subagents
     - `SubagentForm.tsx` - Create/edit modal
     - `SubagentCard.tsx` - Individual subagent display

2. **CRUD Operations**
   - List all subagents
   - Create new subagent (modal form)
   - Edit subagent (modal form)
   - Delete subagent (confirm dialog)
   - Search/filter subagents

3. **API Client Integration**
   - Add API methods:
     ```typescript
     export async function listSubagents(): Promise<Subagent[]>
     export async function getSubagent(id: string): Promise<Subagent>
     export async function createSubagent(data: CreateSubagentRequest): Promise<Subagent>
     export async function updateSubagent(id: string, data: UpdateSubagentRequest): Promise<Subagent>
     export async function deleteSubagent(id: string): Promise<void>
     ```

4. **Form Validation**
   - Name: 3-50 characters, required
   - Description: max 200 characters, optional
   - System prompt: required, max 10,000 characters

**Acceptance Criteria:**
- [ ] User can see list of all subagents
- [ ] User can create new subagent with name, description, prompt
- [ ] User can edit existing subagent
- [ ] User can delete subagent (with confirmation)
- [ ] Form validation works correctly
- [ ] Success/error toast notifications
- [ ] Loading states and error handling

---

### Phase 4: Frontend UI - Chat Integration (Priority: HIGH)

**Squad:** `ui-dev`
**Estimated Time:** 4-5 hours

**Tasks:**
1. **Add Agent Selector to Chat**
   - Component: `AgentSelector.tsx`
   - Dropdown/select to choose between:
     - "Default AI" (no subagent)
     - Custom subagents (fetched from API)
   - Position: Above chat input or in header

2. **Visual Indicator**
   - Show active agent name/avatar
   - Badge or chip with agent name
   - Color coding (default vs custom)

3. **API Integration**
   - Fetch subagents list for selector
   - Call `PUT /api/v1/chat/sessions/:id/subagent` when agent changes
   - Update WebSocket to pass subagent context

4. **Chat History Metadata**
   - Show which agent responded (icon, name)
   - Different styling for default vs subagent responses

**Acceptance Criteria:**
- [ ] User can select agent before sending message
- [ ] Visual indicator shows active agent
- [ ] Switching agent updates session metadata
- [ ] Chat history shows which agent responded
- [ ] Smooth UX (no page refresh needed)

---

### Phase 5: Testing & QA (Priority: MEDIUM)

**Squad:** `ui-tester`
**Estimated Time:** 3-4 hours

**Tasks:**
1. **Backend Integration Tests**
   - Test system prompt injection
   - Test subagent prompt injection
   - Test subagent switching mid-session
   - Test authorization (user can only access own subagents)

2. **Frontend E2E Tests**
   - Test system prompt CRUD
   - Test subagent CRUD
   - Test agent selection in chat
   - Test chat with system prompt
   - Test chat with subagent

3. **Security Tests**
   - Verify JWT authentication
   - Verify company-level data isolation
   - Test XSS prevention in prompts
   - Test SQL injection prevention

**Acceptance Criteria:**
- [ ] All integration tests passing
- [ ] E2E tests for critical user flows
- [ ] Security vulnerabilities addressed
- [ ] Performance benchmarks met

---

## API Contracts

### Existing APIs (Already Implemented)

#### 1. Get System Prompt
```
GET /api/v1/ai/system-prompt
Authorization: Bearer <JWT>

Response 200:
{
  "systemPrompt": "You are a helpful AI assistant..."
}
```

#### 2. Update System Prompt
```
PUT /api/v1/ai/system-prompt
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "systemPrompt": "You are a DevOps expert..."
}

Response 200:
{
  "success": true,
  "message": "System prompt updated successfully"
}
```

#### 3. List Subagents
```
GET /api/v1/ai/subagents
Authorization: Bearer <JWT>

Response 200:
{
  "subagents": [
    {
      "id": "507f1f77bcf86cd799439011",
      "userId": "user123",
      "companyId": "company456",
      "name": "Code Reviewer",
      "description": "Reviews code for best practices",
      "systemPrompt": "You are an expert code reviewer...",
      "createdAt": "2025-10-12T10:00:00Z",
      "updatedAt": "2025-10-12T10:00:00Z"
    }
  ],
  "count": 1
}
```

#### 4. Get Subagent
```
GET /api/v1/ai/subagents/:id
Authorization: Bearer <JWT>

Response 200:
{
  "subagent": {
    "id": "507f1f77bcf86cd799439011",
    ...
  }
}

Response 404:
{
  "error": "subagent not found or access denied"
}
```

#### 5. Create Subagent
```
POST /api/v1/ai/subagents
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "name": "DevOps Helper",
  "description": "Assists with deployment and infrastructure",
  "systemPrompt": "You are a DevOps expert specializing in Kubernetes..."
}

Response 201:
{
  "subagent": {
    "id": "507f1f77bcf86cd799439011",
    "userId": "user123",
    "companyId": "company456",
    "name": "DevOps Helper",
    ...
  }
}

Response 400:
{
  "error": "Invalid request: name is required"
}
```

#### 6. Update Subagent
```
PUT /api/v1/ai/subagents/:id
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "name": "Updated Name",
  "description": "Updated description",
  "systemPrompt": "Updated prompt..."
}

Response 200:
{
  "subagent": { ... }
}

Response 403:
{
  "error": "unauthorized: subagent does not belong to user"
}
```

#### 7. Delete Subagent
```
DELETE /api/v1/ai/subagents/:id
Authorization: Bearer <JWT>

Response 200:
{
  "success": true,
  "message": "Subagent deleted successfully"
}

Response 403:
{
  "error": "unauthorized: subagent does not belong to user"
}
```

### New APIs (To Be Implemented)

#### 8. Set Session Subagent (NEW)
```
PUT /api/v1/chat/sessions/:sessionId/subagent
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "subagentId": "507f1f77bcf86cd799439011"  // or null to use default AI
}

Response 200:
{
  "success": true,
  "message": "Session subagent updated",
  "session": {
    "id": "...",
    "activeSubagentId": "507f1f77bcf86cd799439011",
    ...
  }
}

Response 404:
{
  "error": "session not found or access denied"
}

Response 400:
{
  "error": "subagent not found"
}
```

---

## Database Schema

### Existing Collections

#### 1. `system_prompts` (MongoDB)
```json
{
  "_id": ObjectId,
  "userId": "string",
  "companyId": "string",
  "prompt": "string",
  "updatedAt": ISODate
}

// Indexes:
// - {userId: 1, companyId: 1}
```

#### 2. `subagents` (MongoDB)
```json
{
  "_id": ObjectId,
  "userId": "string",
  "companyId": "string",
  "name": "string",
  "description": "string",
  "systemPrompt": "string",
  "createdAt": ISODate,
  "updatedAt": ISODate
}

// Indexes:
// - {userId: 1, companyId: 1}
// - {companyId: 1}
```

### Enhanced Collections

#### 3. `chat_sessions` (MongoDB) - ENHANCED
```json
{
  "_id": ObjectId,
  "userId": "string",
  "companyId": "string",
  "title": "string",
  "activeSubagentId": ObjectId | null,  // NEW FIELD
  "createdAt": ISODate,
  "updatedAt": ISODate
}

// Indexes (existing):
// - {userId: 1, companyId: 1}
// - {companyId: 1}

// New Index (recommended):
// - {activeSubagentId: 1}  // For querying sessions using a specific subagent
```

**Migration:** No migration needed - MongoDB allows adding optional fields without breaking existing documents.

---

## UI Design

### 1. Settings Page - System Prompt Editor

```
┌──────────────────────────────────────────────────────────────┐
│ Settings                                                      │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  System Prompt                                                │
│  ─────────────────────────────────────────────────────────   │
│  Customize the AI's behavior with a global system prompt      │
│  that applies to all conversations.                           │
│                                                               │
│  ┌────────────────────────────────────────────────────────┐  │
│  │ You are a helpful AI assistant specialized in...      │  │
│  │                                                        │  │
│  │                                                        │  │
│  │                                                        │  │
│  │                                                        │  │
│  │                                                        │  │
│  └────────────────────────────────────────────────────────┘  │
│  Characters: 245 / 10,000                                     │
│                                                               │
│  [ Save ]  [ Clear ]                                          │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 2. Subagents Page - Management Interface

```
┌──────────────────────────────────────────────────────────────┐
│ Subagents                              [ + Create Subagent ]  │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  [ Search... ]                                                │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ 🤖 Code Reviewer                             [Edit] │     │
│  │ Reviews code for best practices and security        │     │
│  │ Created: Oct 12, 2025                       [Delete]│     │
│  └─────────────────────────────────────────────────────┘     │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐     │
│  │ 🚀 DevOps Helper                             [Edit] │     │
│  │ Assists with Kubernetes deployments                 │     │
│  │ Created: Oct 11, 2025                       [Delete]│     │
│  └─────────────────────────────────────────────────────┘     │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

### 3. Chat Page - Agent Selector

```
┌──────────────────────────────────────────────────────────────┐
│ Chat Session                                                  │
├──────────────────────────────────────────────────────────────┤
│  Active Agent: [ Default AI ▼ ]                              │
│                ┌────────────────────────┐                     │
│                │ Default AI             │                     │
│                ├────────────────────────┤                     │
│                │ 🤖 Code Reviewer       │                     │
│                │ 🚀 DevOps Helper       │                     │
│                └────────────────────────┘                     │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ User: How do I deploy to Kubernetes?                   │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 🚀 DevOps Helper:                                        │ │
│  │ To deploy to Kubernetes, follow these steps...          │ │
│  └─────────────────────────────────────────────────────────┘ │
│                                                               │
│  ┌───────────────────────────────────────────────┐           │
│  │ Type a message...                             │  [ Send ] │
│  └───────────────────────────────────────────────┘           │
└──────────────────────────────────────────────────────────────┘
```

---

## Security & Authorization

### 1. Authentication
- **JWT Required**: All API endpoints require valid JWT token
- **User Context**: Extract `userId` and `companyId` from JWT claims
- **Middleware**: `OptionalJWTMiddleware` in `internal/middleware/`

### 2. Authorization Rules

#### System Prompts
- Users can only view/edit their own system prompt
- Filter: `{userId: <user>, companyId: <company>}`

#### Subagents
- Users can only view/edit/delete their own subagents
- Filter: `{userId: <user>, companyId: <company>}`
- Update/Delete: Verify subagent belongs to user before modification

#### Chat Sessions
- Users can only access their own sessions
- Filter: `{userId: <user>, companyId: <company>}`
- Setting subagent: Verify both session and subagent belong to user

### 3. Data Isolation
- **Company-Level**: All queries filtered by `companyId`
- **User-Level**: System prompts and subagents isolated by `userId`
- **MongoDB Indexes**: Composite indexes on `{userId, companyId}` for fast queries

### 4. Input Validation
- **System Prompt**: Max 10,000 characters, sanitize HTML
- **Subagent Name**: 3-50 characters, alphanumeric + spaces
- **Subagent Description**: Max 200 characters
- **Subagent System Prompt**: Max 10,000 characters, sanitize HTML

### 5. Rate Limiting (Recommended)
- System prompt updates: 10 per minute
- Subagent creation: 20 per hour
- Subagent updates: 50 per hour

---

## Testing Strategy

### 1. Unit Tests

#### Backend (Go)
- `ai_settings_service_test.go`:
  - Test CRUD operations for system prompts
  - Test CRUD operations for subagents
  - Test authorization (user can only access own data)
  - Test company-level isolation
- `chat_websocket_test.go`:
  - Test system prompt injection
  - Test subagent prompt injection
  - Test prompt override logic (subagent > system prompt)

#### Frontend (TypeScript)
- `SystemPromptEditor.test.tsx`:
  - Test character counter
  - Test save/clear functionality
  - Test API error handling
- `SubagentsList.test.tsx`:
  - Test list rendering
  - Test CRUD operations
  - Test search/filter
- `AgentSelector.test.tsx`:
  - Test dropdown rendering
  - Test agent switching
  - Test visual indicator

### 2. Integration Tests

#### Backend (Go)
- Test full chat flow with system prompt
- Test full chat flow with subagent
- Test switching subagent mid-session
- Test authorization across all endpoints

#### Frontend (E2E with Playwright)
- Test system prompt management flow
- Test subagent management flow
- Test chat with agent selection
- Test agent switching during chat

### 3. Security Tests
- Test JWT validation
- Test XSS prevention in prompts
- Test SQL injection prevention
- Test rate limiting
- Test company-level data isolation

### 4. Performance Tests
- Benchmark system prompt injection overhead
- Test concurrent chat sessions with different agents
- Test WebSocket performance with prompt injection

---

## Rollout Plan

### Phase 1: Backend (Week 1, Days 1-2)
- [ ] Enhance ChatSession model (`ActiveSubagentID`)
- [ ] Enhance ChatService (`SetSessionSubagent`)
- [ ] Enhance ChatWebSocketHandler (prompt injection)
- [ ] Add REST endpoint for setting session subagent
- [ ] Unit tests
- [ ] Integration tests
- [ ] Deploy to dev environment
- [ ] Manual testing

**Dependencies:** None (backend APIs already exist)
**Risk:** Low (additive changes, no breaking changes)

### Phase 2: Frontend - Settings & Subagents (Week 1, Days 3-4)
- [ ] Create Settings page (system prompt editor)
- [ ] Create Subagents page (CRUD interface)
- [ ] API client integration
- [ ] Unit tests
- [ ] Deploy to dev environment
- [ ] Manual testing

**Dependencies:** Phase 1 (backend integration)
**Risk:** Medium (UI source code location unknown)

### Phase 3: Frontend - Chat Integration (Week 1, Day 5)
- [ ] Add agent selector to chat
- [ ] Visual indicator for active agent
- [ ] Update WebSocket integration
- [ ] Unit tests
- [ ] E2E tests
- [ ] Deploy to dev environment
- [ ] User acceptance testing

**Dependencies:** Phase 1, Phase 2
**Risk:** Medium (WebSocket integration complexity)

### Phase 4: Testing & QA (Week 2, Days 1-2)
- [ ] Full integration testing
- [ ] Security testing
- [ ] Performance testing
- [ ] Bug fixes
- [ ] Documentation updates

**Dependencies:** Phase 1, 2, 3
**Risk:** Low

### Phase 5: Production Deployment (Week 2, Day 3)
- [ ] Final code review
- [ ] Merge to main
- [ ] Deploy to staging
- [ ] Smoke tests
- [ ] Deploy to production
- [ ] Monitor logs and metrics
- [ ] User announcement

**Dependencies:** Phase 4 (all tests passing)
**Risk:** Low (feature is additive, no breaking changes)

---

## Success Metrics

### Feature Adoption
- % of users who set a system prompt (target: 30% in first month)
- % of users who create subagents (target: 20% in first month)
- Average number of subagents per user (target: 2-3)
- % of chat sessions using custom subagents (target: 15%)

### Performance
- System prompt injection latency: < 10ms
- No degradation in chat response time
- WebSocket throughput maintained

### Quality
- Zero critical bugs in production
- < 1% error rate on API endpoints
- 95% uptime for chat service

---

## Open Questions

1. **UI Source Code Location**: Where is the frontend source code? (Not in `coordinator/ui/src/`)
   - **Answer:** Check separate repository or confirm UI doesn't exist yet

2. **System Prompt Caching**: Should we cache system prompts in memory?
   - **Recommendation:** Yes, cache per-user with 5-minute TTL to reduce MongoDB queries

3. **Subagent Limits**: Should we limit the number of subagents per user?
   - **Recommendation:** Start with 20 subagents per user, increase if needed

4. **System Prompt Versioning**: Should we track history of system prompt changes?
   - **Recommendation:** Phase 2 feature - add versioning later if needed

5. **Subagent Sharing**: Should users be able to share subagents within their company?
   - **Recommendation:** Phase 2 feature - add company-level subagents later

---

## Appendix

### A. Related Code Files

**Backend:**
- `internal/models/ai_settings.go` - Data models (DONE)
- `internal/services/ai_settings_service.go` - Service layer (DONE)
- `internal/handlers/ai_settings_handler.go` - API handlers (DONE)
- `internal/server/http_server.go` - Route registration (DONE)
- `internal/models/chat.go` - Chat models (TO ENHANCE)
- `internal/services/chat_service.go` - Chat service (TO ENHANCE)
- `internal/handlers/chat_websocket.go` - WebSocket handler (TO ENHANCE)
- `internal/handlers/chat_handler.go` - Chat REST handler (TO ADD ENDPOINT)

**Frontend:**
- Location TBD (UI source code not found in repository)

### B. References
- LangChain Messages Documentation
- MongoDB Indexing Best Practices
- WebSocket Best Practices
- JWT Authentication Standards

---

**Document Version:** 1.0
**Last Updated:** October 12, 2025
**Authors:** Hyperion AI Platform Team
**Status:** Design Approved - Ready for Implementation
