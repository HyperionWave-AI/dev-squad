# AI Band Manager - Project Specification

**Version:** 1.0
**Last Updated:** 2025-09-30
**Purpose:** Test project to validate full functionality of hyperion-coordinator MCP

---

## 🚨 **CRITICAL: Working Directory Requirement**

**MANDATORY: All work for this project MUST be done within the `examples/` directory.**

- ✅ Project root: `examples/ai-band-manager/`
- ✅ Backend code: `examples/ai-band-manager/backend/`
- ✅ Frontend code: `examples/ai-band-manager/frontend/`
- ✅ Documentation: `examples/ai-band-manager/README.md`
- ✅ Configuration: `examples/ai-band-manager/.env.example`

**❌ NO work should be done outside the `examples/` folder**

This ensures the project remains isolated as an example/test project and doesn't interfere with other repository code.

---

## 📋 Project Overview

### Concept
A fun, creative web application where users describe a music genre/vibe, and specialized AI agents collaborate in real-time to create a complete virtual band identity including lyrics, artwork, biography, and marketing materials.

### Primary Goal
Demonstrate and test all coordinator MCP operations through an engaging, visual interface that showcases parallel agent coordination, real-time status updates, and knowledge persistence.

### Target Audience
- MCP developers testing coordinator functionality
- Demonstrations of multi-agent orchestration
- Educational tool for understanding agent coordination patterns

---

## 🎯 Core Features

### 1. Band Creation Workflow

**User Input:**
- Simple form with textarea: "Describe your band idea"
- Example prompts provided:
  - "Create a punk rock band about cats"
  - "Jazz fusion group inspired by space exploration"
  - "K-pop boy band but they're all vampires"
- Optional fields: band name suggestion, preferred era (70s, 80s, 90s, 2000s, modern)

**System Response:**
1. Creates human task via `coordinator_create_human_task`
2. Spawns 4 agent tasks via `coordinator_create_agent_task`:
   - **Lyricist Agent**: Write song snippets and hooks
   - **Album Art Designer Agent**: Describe album cover and logo concepts
   - **Bio Writer Agent**: Create band backstory and member profiles
   - **Social Media Manager Agent**: Generate Instagram bio, tour poster copy, press release

### 2. Real-Time Creative Dashboard

**Visual Components:**

**Agent Status Cards** (4 cards in grid layout):
```
┌─────────────────────────────────────┐
│ 🎤 Lyricist Agent                   │
│ Status: in_progress                 │
│ ━━━━━━━━━━░░░░░░░░░░ 60%           │
│                                     │
│ TODOs:                              │
│ ✅ Analyze genre characteristics    │
│ ✅ Generate song themes             │
│ ⏳ Writing chorus for "Meow Chaos" │
│ ⬜ Writing verse 1                  │
│ ⬜ Writing bridge                   │
└─────────────────────────────────────┘
```

**Central Progress Tracker:**
- Overall band creation progress (aggregated from all agents)
- Estimated time remaining
- Current phase: "Brainstorming" → "Creating" → "Finalizing" → "Complete"

**Live Activity Feed:**
- Real-time stream of agent status updates
- Example entries:
  - "🎤 Lyricist Agent: Started writing chorus..."
  - "🎨 Album Art Designer: Analyzing punk aesthetics..."
  - "📝 Bio Writer: Crafting band origin story..."

### 3. Final Deliverables View

**Band Identity Package** (displayed when all agents complete):

**1. Band Profile:**
- Band name (AI-generated or user-suggested)
- Genre/subgenre
- Origin story (2-3 paragraphs)
- Band member profiles (4-5 members with names, instruments, personalities)

**2. Musical Content:**
- 3 song titles with lyric snippets (chorus + verse sample)
- Songwriting themes and motifs
- Musical style description

**3. Visual Identity:**
- Album cover concept (detailed text description)
- Logo design description
- Color palette and aesthetic notes
- Band photo concept

**4. Marketing Materials:**
- Instagram bio (150 chars)
- Press release (1 paragraph)
- Tour poster tagline
- 3 social media post ideas

### 4. Knowledge Base Viewer

**Collections Explorer:**
- View what agents learned during creation
- Query interface for:
  - `musical-styles` - Genre patterns and characteristics
  - `creative-patterns` - Successful creative combinations
  - `band-history` - All previously created bands
  - `coordination-insights` - Agent collaboration patterns

**Search Functionality:**
- Search by genre: "Show me all punk rock bands"
- Search by theme: "Find bands about animals"
- Search by agent: "What has the Lyricist Agent created?"

---

## 🏗️ Technical Architecture

### Frontend (React + TypeScript)

**Tech Stack:**
- React 18 with TypeScript
- State management: React Context + hooks
- Real-time updates: Server-Sent Events (SSE) or WebSocket
- UI Framework: Tailwind CSS + shadcn/ui components
- Build tool: Vite

**Key Components:**
```
examples/ai-band-manager/frontend/src/
├── components/
│   ├── BandCreationForm.tsx        # User input form
│   ├── AgentStatusCard.tsx         # Individual agent progress
│   ├── ProgressTracker.tsx         # Overall completion status
│   ├── ActivityFeed.tsx            # Live agent updates
│   ├── BandDeliverable.tsx         # Final output display
│   └── KnowledgeBaseViewer.tsx     # Qdrant search interface
├── hooks/
│   ├── useCoordinator.ts           # Coordinator MCP interactions
│   ├── useRealtimeUpdates.ts      # SSE/WebSocket connection
│   └── useKnowledgeSearch.ts      # Qdrant queries
├── services/
│   ├── coordinatorApi.ts           # Backend API client
│   └── eventStream.ts              # SSE/WebSocket client
└── types/
    ├── coordinator.ts              # MCP type definitions
    └── band.ts                     # Band domain types
```

**State Management:**
```typescript
interface AppState {
  humanTask: HumanTask | null;
  agentTasks: AgentTask[];
  activityLog: ActivityEntry[];
  finalDeliverable: BandIdentity | null;
  connectionStatus: 'connected' | 'disconnected' | 'error';
}
```

### Backend (Go)

**Tech Stack:**
- Go 1.25
- Web framework: Gin
- MCP integration: Official MCP Go SDK
- Real-time: Server-Sent Events
- Configuration: Environment variables

**Project Structure:**
```
examples/ai-band-manager/backend/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point
├── internal/
│   ├── handlers/
│   │   ├── band.go                # Band creation endpoints
│   │   ├── coordinator.go         # MCP proxy handlers
│   │   └── events.go              # SSE stream handler
│   ├── services/
│   │   ├── coordinator_client.go  # MCP client wrapper
│   │   ├── agent_orchestrator.go  # Agent task management
│   │   └── event_broadcaster.go   # SSE broadcast service
│   ├── models/
│   │   ├── band.go                # Domain models
│   │   └── coordinator.go         # MCP types
│   └── config/
│       └── config.go              # Configuration
├── go.mod
└── go.sum
```

**Key API Endpoints:**
```
POST   /api/v1/bands              # Create new band (human task)
GET    /api/v1/bands/:id          # Get band status
GET    /api/v1/bands/:id/agents   # List agent tasks
GET    /api/v1/bands/:id/stream   # SSE event stream
POST   /api/v1/knowledge/search   # Query Qdrant via coordinator
GET    /api/v1/health             # Health check
```

**Agent Orchestration Flow:**
```go
// 1. Create human task
humanTask := coordinator.CreateHumanTask(userPrompt)

// 2. Spawn agent tasks in parallel
agents := []AgentConfig{
    {Name: "lyricist", Role: "Song lyric creation", TODOs: [...]},
    {Name: "album-art-designer", Role: "Visual identity design", TODOs: [...]},
    {Name: "bio-writer", Role: "Band biography and profiles", TODOs: [...]},
    {Name: "social-media-manager", Role: "Marketing content", TODOs: [...]},
}

for _, agent := range agents {
    go coordinator.CreateAgentTask(humanTask.ID, agent)
}

// 3. Simulate agent work with status updates
for each agent {
    for each TODO {
        coordinator.UpdateTodoStatus(agentTaskID, todoID, "in_progress")
        // Simulate creative work (sleep 2-5s)
        coordinator.UpdateTodoStatus(agentTaskID, todoID, "completed")
        // Broadcast SSE event
    }
}

// 4. Store knowledge in Qdrant via coordinator
coordinator.UpsertKnowledge("musical-styles", ...)
coordinator.UpsertKnowledge("creative-patterns", ...)
```

### Real-Time Updates

**Server-Sent Events (SSE) Stream:**
```javascript
// Frontend connection
const eventSource = new EventSource('/api/v1/bands/123/stream');

eventSource.addEventListener('agent_status', (event) => {
  const update = JSON.parse(event.data);
  // Update UI with agent progress
});

eventSource.addEventListener('todo_completed', (event) => {
  const update = JSON.parse(event.data);
  // Animate TODO checkmark
});

eventSource.addEventListener('band_completed', (event) => {
  const deliverable = JSON.parse(event.data);
  // Show final band identity
});
```

**Event Types:**
- `agent_status` - Agent task status changed
- `todo_started` - TODO marked in_progress
- `todo_completed` - TODO marked completed
- `activity_log` - New activity feed entry
- `band_completed` - All agents finished
- `error` - Error occurred

---

## 🧪 Coordinator MCP Test Coverage

### Operations Tested

**✅ Task Creation:**
- `coordinator_create_human_task` - Create band creation request
- `coordinator_create_agent_task` - Spawn 4 specialized agents

**✅ Task Management:**
- `coordinator_list_human_tasks` - Display all band requests
- `coordinator_list_agent_tasks` - Filter by agentName, humanTaskId
- `coordinator_update_task_status` - Track agent progress (pending → in_progress → completed)

**✅ TODO Management:**
- `coordinator_update_todo_status` - Mark individual TODOs as completed
- Auto-completion - Verify task completes when all TODOs done

**✅ Knowledge Operations:**
- `coordinator_upsert_knowledge` - Store creative patterns, musical styles
- `coordinator_query_knowledge` - Search previous bands and patterns

**✅ Edge Cases:**
- Blocked agent (simulate dependency wait)
- Failed TODO (simulate creative block, require retry)
- Concurrent agent updates (parallel execution)
- Large knowledge payloads (full band deliverable)

### Success Metrics

**Functional:**
- ✅ All 4 agents complete tasks successfully
- ✅ Real-time UI updates reflect coordinator status changes
- ✅ Knowledge stored and retrievable via Qdrant
- ✅ Task hierarchy maintained (human → agent → TODOs)

**Performance:**
- ⏱️ Band creation completes in <60 seconds
- ⏱️ SSE latency <100ms for status updates
- ⏱️ Knowledge queries return <500ms
- ⏱️ UI remains responsive during agent work

**User Experience:**
- 🎨 Visual progress indicators for each agent
- 🎨 Smooth animations for TODO completions
- 🎨 Clear error messages if coordinator fails
- 🎨 Export band identity as JSON or shareable link

---

## 🎨 UI/UX Design Guidelines

### Visual Theme
- **Style:** Modern, playful, music-festival inspired
- **Color Palette:**
  - Primary: Electric purple (#8B5CF6)
  - Secondary: Hot pink (#EC4899)
  - Accent: Neon cyan (#06B6D4)
  - Background: Dark slate (#0F172A)
  - Text: White/light gray
- **Typography:**
  - Headers: Bold, rock-poster style
  - Body: Clean, readable sans-serif
  - Code/data: Monospace

### Animations
- Agent cards "pulse" when in_progress
- TODOs get checkmark animation when completed
- Progress bars fill smoothly
- Final deliverable "reveals" with fade-in effect
- Activity feed entries slide in from right

### Responsive Design
- Desktop: 4-column agent grid
- Tablet: 2-column agent grid
- Mobile: Single column, collapsible agent cards

### Accessibility
- WCAG 2.1 AA compliance
- Keyboard navigation support
- Screen reader friendly status updates
- High contrast mode option

---

## 🚀 Implementation Phases

### Phase 1: Foundation (Days 1-2)
- ✅ Go backend with Gin server
- ✅ Coordinator MCP client integration
- ✅ React frontend scaffolding with Vite
- ✅ Basic API endpoints (create band, get status)
- ✅ Simple form UI

### Phase 2: Agent Orchestration (Days 3-4)
- ✅ Agent task creation logic
- ✅ TODO status progression simulation
- ✅ SSE stream implementation
- ✅ Real-time UI updates
- ✅ Agent status cards

### Phase 3: Creative Content (Days 5-6)
- ✅ Agent-specific TODO lists and outputs
- ✅ Final deliverable aggregation
- ✅ Band identity display UI
- ✅ Knowledge storage (Qdrant)

### Phase 4: Polish & Testing (Days 7-8)
- ✅ Knowledge base viewer UI
- ✅ Search functionality
- ✅ Error handling and edge cases
- ✅ UI animations and polish
- ✅ Documentation and demo video

---

## 📦 Deliverables

### 1. Source Code
- All code in `examples/ai-band-manager/` directory
- Clear README at `examples/ai-band-manager/README.md`
- Docker Compose setup for local development
- Environment variable templates (.env.example files)

### 2. Documentation
- API documentation (endpoints, request/response examples)
- MCP integration guide (how coordinator is used)
- Deployment guide (run locally, deploy to cloud)

### 3. Demo Materials
- 3-5 minute demo video showing:
  - Band creation flow
  - Real-time agent coordination
  - Knowledge base exploration
- Screenshot gallery
- Example band outputs (JSON exports)

### 4. Test Report
- Coordinator MCP operation coverage checklist
- Performance benchmarks
- Edge case validation results

---

## 🔧 Configuration

### Environment Variables

**Backend (.env):**
```bash
# Server
PORT=8080
ENVIRONMENT=development

# Coordinator MCP
COORDINATOR_MCP_URL=ws://localhost:9999/coordinator
COORDINATOR_AUTH_TOKEN=your_jwt_token_here

# CORS
ALLOWED_ORIGINS=http://localhost:5173

# Simulation (for testing)
AGENT_WORK_DELAY_MS=2000  # Simulate agent thinking time
ENABLE_MOCK_MODE=false     # Use mock responses instead of real MCP
```

**Frontend (.env):**
```bash
VITE_API_BASE_URL=http://localhost:8080
VITE_SSE_RECONNECT_DELAY=3000
```

---

## 🐛 Testing Strategy

### Unit Tests
- Go services: Agent orchestrator logic
- React components: Status card rendering
- API endpoints: Request/response validation

### Integration Tests
- End-to-end band creation flow
- SSE connection stability
- Coordinator MCP operation sequences

### Manual Testing Scenarios
1. **Happy Path**: Create band, watch agents work, view deliverable
2. **Slow Network**: Simulate delayed SSE updates
3. **Agent Failure**: One agent encounters error, system recovers
4. **Concurrent Requests**: Multiple users creating bands simultaneously
5. **Knowledge Search**: Query for previous bands, verify results

---

## 📊 Success Criteria

### MVP Complete When:
- ✅ User can input band idea and receive full band identity
- ✅ All 4 agents execute in parallel with visible progress
- ✅ Real-time updates reflect coordinator task status changes
- ✅ Knowledge stored in Qdrant and searchable
- ✅ UI is responsive, animated, and visually appealing
- ✅ All coordinator MCP operations successfully invoked

### Bonus Features (If Time Permits):
- 🎁 User authentication (JWT) to save favorite bands
- 🎁 "Remix" button to regenerate specific parts (re-run single agent)
- 🎁 Export band as shareable link or PDF
- 🎁 "Battle of the Bands" mode (create 2 bands, vote on favorite)
- 🎁 Integration with AI image generation for actual album art

---

## 🎯 Key Learnings Goals

By completing this project, developers will learn:
1. **Coordinator MCP Patterns**: Task creation, status management, knowledge storage
2. **Real-Time Agent Coordination**: SSE streaming, parallel execution visualization
3. **Knowledge Management**: Qdrant integration for persistent context
4. **User-Facing Multi-Agent Systems**: How to make agent coordination visible and fun
5. **Production-Ready Practices**: Error handling, configuration, deployment

---

## 📞 Support & Resources

### Coordinator MCP Documentation
- [MCP Protocol Specification](link-to-docs)
- [Hyperion Coordinator API Reference](link-to-coordinator-docs)

### Related Examples
- tasks-api refactoring (god file elimination pattern)
- Parallel squad coordination examples from CLAUDE.md

### Questions & Issues
- File GitHub issues in test project repository
- Reference this spec document for context

---

**Ready to Build?** 🎸

This specification provides everything needed to implement AI Band Manager. Start with Phase 1 (foundation), then iterate through each phase. Remember: the goal is testing coordinator MCP functionality while creating something genuinely fun and engaging!

**Have fun building! 🚀🎶**