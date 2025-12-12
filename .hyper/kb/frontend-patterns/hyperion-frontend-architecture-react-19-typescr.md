# HYPERION FRONTEND ARCHITECTURE (React 19 + TypeScr

**Collection:** frontend-patterns
**Created:** 2025-11-20

---

HYPERION FRONTEND ARCHITECTURE (React 19 + TypeScript)

COMPONENT HIERARCHY (Atomic Design):
- Atoms: Basic UI building blocks (Button, Input, Badge, Avatar, Icon, Label, Textarea)
- Molecules: Combinations of atoms (form groups, tool inputs, message bubbles)
- Organisms: Complex components (ChatInterface, TaskBoard, KnowledgeSearch)
- Templates: Page-level layouts (PageLayout with sidebar navigation)

ROUTING & PAGES:
Entry: /src/App.tsx (line 29-113) with React Router v7
Pages:
- /chat: CodeChatPage (AI conversation with streaming)
- /tasks: KanbanBoard (task management via drag-drop)
- /blog: BlogProgressPage (dashboard/analytics)
- /knowledge: KnowledgeBasePage (search/browse knowledge entries)
- /reflection: ReflectionPage (system metacognition)
- /code: CodeSearchPage (semantic code search)
- /mcp-servers: MCPServersPage (MCP server management)
- /tools: HTTPToolsPage (tool discovery/execution)
- /subagents: SubagentsPage (agent configuration)
- /settings: SettingsPage (user preferences)

STATE MANAGEMENT:
- React Context API for global state (ConversationModeContext at /src/contexts/ConversationModeContext.tsx)
- Conversation modes: debug (technical details) vs default (user-friendly)
- User settings persisted to backend via userSettingsService
- Optimistic updates with error recovery

CUSTOM HOOKS (/src/hooks/):
- useStreamingPerformance: FPS tracking, network stats during streaming (bytes/chunks/tokens)
- useKeyboardShortcuts: Keyboard event handling
- usePromptNotes: Human guidance note management

STYLING:
- Tailwind CSS with custom configuration
- PostCSS for processing
- Radix UI for accessible component primitives
- Lucide React for iconography
- Framer Motion for animations

DATA FLOW:
Services (/src/services/) handle API communication
- Real-time WebSocket for chat streaming
- REST endpoints for CRUD operations
- Error handling with user feedback
- Loading states and skeleton UI

PERFORMANCE:
- Component memoization via React.memo
- Streaming performance monitoring (FPS, throughput)
- Code splitting with React Router lazy loading
- Image optimization
- Tailwind CSS production builds (tree-shaking unused styles)
