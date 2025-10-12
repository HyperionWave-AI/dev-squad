# System Prompt & Sub-Agents - Design Summary

**Date:** October 12, 2025
**Status:** ✅ DESIGN COMPLETE - Ready for Implementation

---

## 🎯 What You Asked For

> "add ability to add system prompt and sub agents. design first, use hyper and the team."

---

## ✅ What's Already Built (70% Complete!)

**Great news!** Most of the backend is already implemented:

### Backend API (100% Complete)
- ✅ MongoDB models for `SystemPrompt` and `Subagent`
- ✅ Full CRUD service layer (`AISettingsService`)
- ✅ REST API handlers with JWT authentication
- ✅ Routes registered at `/api/v1/ai/*`
- ✅ Company-level data isolation

**Available Endpoints:**
```
GET    /api/v1/ai/system-prompt       # Get user's system prompt
PUT    /api/v1/ai/system-prompt       # Update system prompt

GET    /api/v1/ai/subagents            # List all subagents
GET    /api/v1/ai/subagents/:id        # Get specific subagent
POST   /api/v1/ai/subagents            # Create subagent
PUT    /api/v1/ai/subagents/:id        # Update subagent
DELETE /api/v1/ai/subagents/:id        # Delete subagent
```

---

## ❌ What's Missing (30% of Work)

### 1. Backend Chat Integration (20%)
The chat WebSocket handler doesn't use system prompts yet:
- Need to inject system prompt as first "system" message
- Need to support subagent selection per chat session
- Need new endpoint: `PUT /api/v1/chat/sessions/:id/subagent`

### 2. Frontend UI (80% of remaining work)
No UI exists yet (source code not found in repository):
- Settings page for system prompt editor
- Subagents management page (CRUD interface)
- Agent selector in chat page
- Visual indicators for active agent

---

## 📋 Architecture Overview

```
User → Chat UI → WebSocket → Backend Chat Handler
                                    ↓
                      Fetch System Prompt OR Subagent Prompt
                                    ↓
                      Inject as first "system" message
                                    ↓
                           Stream AI Response
```

**Key Design Decisions:**
1. **System Prompt**: Global default for all chats (user-level)
2. **Subagent**: Custom agent with own system prompt (overrides global)
3. **Chat Session**: Tracks active subagent (optional field)
4. **Priority**: Subagent prompt > System prompt > No prompt

---

## 📄 Documentation Created

### 1. **SYSTEM_PROMPT_SUBAGENTS_DESIGN.md** (Comprehensive ADR)
- 400+ lines of detailed architecture design
- Database schema with MongoDB collections
- API contracts (existing + new endpoints)
- UI wireframes and mockups
- Security & authorization rules
- Testing strategy
- Rollout plan

### 2. **IMPLEMENTATION_PLAN_SYSTEM_PROMPT_SUBAGENTS.md** (Implementation Guide)
- Phase-by-phase breakdown
- 5 implementation phases
- Agent assignments (go-dev, ui-dev, ui-tester)
- Exact files to modify with code examples
- Testing checklists
- Timeline: 2 weeks

### 3. **PROJECT_STATUS.md** (Updated)
- All filesystem tools verified ✅
- All 4 binaries building successfully ✅

---

## 🚀 Implementation Plan Summary

### Phase 1: Backend Chat Integration (2-3 hours)
**Agent:** `go-dev`
**Files:** 5 files (chat.go, chat_service.go, chat_websocket.go, chat_handler.go, http_server.go)
**Goal:** Inject system prompts into chat

### Phase 2: Frontend - System Prompt Editor (3-4 hours)
**Agent:** `ui-dev`
**Files:** SettingsPage.tsx, SystemPromptEditor.tsx, api.ts
**Goal:** UI to manage system prompt

### Phase 3: Frontend - Subagents Management (4-5 hours)
**Agent:** `ui-dev`
**Files:** SubagentsPage.tsx, SubagentsList.tsx, SubagentForm.tsx
**Goal:** CRUD interface for subagents

### Phase 4: Frontend - Chat Integration (4-5 hours)
**Agent:** `ui-dev`
**Files:** ChatPage.tsx, AgentSelector.tsx
**Goal:** Agent selection in chat

### Phase 5: Testing & QA (3-4 hours)
**Agent:** `ui-tester`
**Goal:** Full integration and E2E testing

**Total Estimated Time:** 16-21 hours over 2 weeks

---

## 🔐 Security Features

- ✅ JWT authentication on all endpoints
- ✅ Company-level data isolation (`companyId` in all queries)
- ✅ User-level authorization (users can only access their own data)
- ✅ Input validation (max 10,000 chars for prompts)
- ✅ MongoDB indexes for fast queries
- ⚠️ XSS prevention (HTML sanitization needed in prompts)

---

## 📊 Expected User Experience

### Creating a Subagent
1. Navigate to `/ui/subagents`
2. Click "Create Subagent"
3. Fill form:
   - Name: "DevOps Helper"
   - Description: "Kubernetes deployment expert"
   - System Prompt: "You are a DevOps expert specializing in..."
4. Click "Save"
5. Subagent appears in list

### Using a Subagent in Chat
1. Open chat session
2. Click agent selector dropdown
3. Select "DevOps Helper"
4. Type message: "How do I deploy to Kubernetes?"
5. AI responds with DevOps expertise
6. Visual indicator shows "🚀 DevOps Helper" is active

### Setting Global System Prompt
1. Navigate to `/ui/settings`
2. Edit textarea: "You are a helpful assistant specialized in..."
3. Click "Save"
4. All future chats use this prompt (unless subagent selected)

---

## 🎯 Success Metrics

**Feature Adoption (Target for Month 1):**
- 30% of users set a system prompt
- 20% of users create subagents
- 15% of chat sessions use custom subagents
- Average 2-3 subagents per user

**Performance:**
- System prompt injection latency: < 10ms
- No degradation in chat response time
- 95% uptime

**Quality:**
- Zero critical bugs
- < 1% error rate
- All tests passing

---

## ⚠️ Open Questions (Need Your Input)

### 1. UI Source Code Location
**Issue:** Couldn't find UI source code in `coordinator/ui/src/` or anywhere else.
**Question:** Where is the frontend source code? Or does it need to be created from scratch?

### 2. UI Framework
**Question:** What UI framework should we use? (React, Vue, Svelte?)
**Recommendation:** React 18 + TypeScript (based on embed/ui structure)

### 3. Subagent Limits
**Question:** Should we limit the number of subagents per user?
**Recommendation:** Start with 20 subagents per user

### 4. System Prompt Caching
**Question:** Should we cache system prompts in memory?
**Recommendation:** Yes, with 5-minute TTL to reduce MongoDB queries

### 5. Feature Priority
**Question:** Which phase should we start with?
**Recommendation:** Start with Phase 1 (backend integration) since it's quick and unblocks everything else

---

## 🎉 Next Steps

**Ready to implement!** Here's what to do:

### Option A: Start with Backend (Recommended)
```bash
# Use Hyperion coordinator to delegate to go-dev agent
# Implement Phase 1 (backend chat integration)
# Takes 2-3 hours, unblocks everything else
```

### Option B: Start with Frontend
```bash
# If UI source code location is known
# Use ui-dev agent to create Settings and Subagents pages
# Can work in parallel with backend
```

### Option C: Full Parallel Execution
```bash
# Launch both go-dev and ui-dev agents simultaneously
# Backend integration + UI creation in parallel
# Fastest path to completion (4-5 days total)
```

---

## 📞 Questions for You

1. **Where is the UI source code?** (Not found in repository)
2. **Which phase should we start with?** (Backend first? UI first? Both in parallel?)
3. **Any specific UI framework preference?**
4. **Do you want me to launch the implementation agents now?**

---

**Design Status:** ✅ COMPLETE
**Implementation Status:** ⏳ AWAITING YOUR APPROVAL TO START

Let me know how you'd like to proceed! I can launch the specialist agents (go-dev, ui-dev) right away using the Hyperion coordinator workflow.
