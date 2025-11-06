# Hyperion Coordinator - Comprehensive Technical Implementation Guide

**Date:** November 5, 2025
**Branch:** megha/knowledge-browser
**Version:** 2.0
**Purpose:** Complete technical reference for implementing coordinator-based chat system with subchats

---

## Table of Contents

1. [Chat and Subchat Work Delegation](#1-chat-and-subchat-work-delegation)
2. [Orchestrator and Implementer Separation](#2-orchestrator-and-implementer-separation)
3. [Subchat Interruption System](#3-subchat-interruption-system)
4. [Progress Tracker with Pulsating Icons](#4-progress-tracker-with-pulsating-icons)
5. [File Indexing for Code Search](#5-file-indexing-for-code-search)
6. [Orchestrator Validations](#6-orchestrator-validations)
7. [Executor Phase Guards](#7-executor-phase-guards)
8. [Detailed Subchat Interruption Implementation](#8-detailed-subchat-interruption-implementation)
9. [Prometheus Dashboard](#9-prometheus-dashboard)
10. [Work Delegation Enforcement](#10-work-delegation-enforcement)
11. [System Prompt Architecture](#11-system-prompt-architecture)
12. [Debug Mode vs Default Mode](#12-debug-mode-vs-default-mode)
13. [Tool Call Display in Debug Mode](#13-tool-call-display-in-debug-mode)

---

## 1. Chat and Subchat Work Delegation

### Architecture Overview

The Hyperion system uses a **two-tier chat architecture** where:
- **Main Chat (Parent)**: User interacts with Coordinator AI
- **Subchat (Child)**: Specialist agent executes implementation tasks

### 1.1 Work Delegation Flow

```
User Request → Coordinator (Main Chat) → Creates Tasks → Launches Subchat → Agent Implements
```

**File:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go`

#### Step 1: User Sends Request to Main Chat

```go
// User message arrives via WebSocket
// Location: hyper/internal/handlers/chat_websocket.go:1050-1100
func (h *ChatWebSocketHandler) handleUserMessage(conn *websocket.Conn, sessionID primitive.ObjectID, userMessage string) {
    // 1. Save user message to MongoDB
    msg := &models.Message{
        SessionID: sessionID,
        Role:      "user",
        Content:   userMessage,
        Timestamp: time.Now(),
    }
    h.messageService.CreateMessage(ctx, msg)

    // 2. Stream AI response with coordinator tools
    aiStream := h.aiService.StreamChatWithTools(ctx, messages, maxToolCalls)

    // 3. Coordinator AI processes and may call execute_subagent tool
}
```

#### Step 2: Coordinator Creates Human Task

**Tool:** `coordinator_create_human_task`

```go
// Location: coordinator_tools.go:450-500
func (t *CoordinatorCreateHumanTaskTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    prompt := args["prompt"].(string)

    // Save to MongoDB
    task := &models.HumanTask{
        Prompt:    prompt,
        Status:    "pending",
        CreatedAt: time.Now(),
    }

    result := t.collection.InsertOne(ctx, task)
    return map[string]interface{}{
        "taskId": result.InsertedID,
        "prompt": prompt,
    }, nil
}
```

#### Step 3: Coordinator Creates Agent Task

**Tool:** `coordinator_create_agent_task`

```go
// Location: coordinator_tools.go:650-750
func (t *CoordinatorCreateAgentTaskTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    agentTask := &models.AgentTask{
        HumanTaskID:    args["humanTaskId"].(string),
        AgentName:      args["agentName"].(string),
        Role:           args["role"].(string),
        ContextSummary: args["contextSummary"].(string),
        FilesModified:  args["filesModified"].([]string),
        Todos:          parseTodos(args["todos"]),
        Status:         "pending",
        CreatedAt:      time.Now(),
    }

    // Validate todos don't contain discovery keywords
    if err := validateTodos(agentTask.Todos); err != nil {
        return nil, err
    }

    result := t.collection.InsertOne(ctx, agentTask)
    return map[string]interface{}{
        "taskId": result.InsertedID.Hex(),
    }, nil
}
```

#### Step 4: Coordinator Launches Subchat

**Tool:** `execute_subagent`

```go
// Location: coordinator_tools.go:2500-2650
func (t *ExecuteSubagentTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
    agentTaskID := args["agentTaskId"].(string)
    parentChatID := args["parentChatId"].(string) // Main chat session ID

    // 1. Create new chat session (subchat)
    subchatSession := &models.ChatSession{
        Title:        "Subchat: " + agentName,
        ParentChatID: &parentChatID,  // Link to parent
        CreatedAt:    time.Now(),
    }
    sessionResult := t.sessionCollection.InsertOne(ctx, subchatSession)
    subchatSessionID := sessionResult.InsertedID.(primitive.ObjectID)

    // 2. Retrieve agent task from MongoDB
    agentTask := t.getAgentTask(ctx, agentTaskID)

    // 3. Build specialized system prompt
    systemPrompt := t.buildAgentSystemPrompt(agentTask)

    // 4. Add initial message to subchat
    initialMsg := &models.Message{
        SessionID: subchatSessionID,
        Role:      "system",
        Content:   systemPrompt,
        Timestamp: time.Now(),
    }
    t.messageService.CreateMessage(ctx, initialMsg)

    // 5. Start AI stream in subchat with filtered tools
    allowedTools := getAgentTools(agentTask.AgentName) // e.g., "ui-dev" → file tools only
    aiStream := t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)

    // 6. Stream responses to subchat
    for event := range aiStream {
        // Save to subchat session, not parent
        // Emit progress notifications to parent chat
    }

    return map[string]interface{}{
        "subchatSessionId": subchatSessionID.Hex(),
        "status":           "started",
    }, nil
}
```

### 1.2 Parent-Child Chat Relationship

**MongoDB Schema:**

```go
// models/chat_session.go
type ChatSession struct {
    ID           primitive.ObjectID  `bson:"_id,omitempty"`
    Title        string              `bson:"title"`
    ParentChatID *primitive.ObjectID `bson:"parentChatId,omitempty"` // NULL for main chats
    CreatedAt    time.Time           `bson:"createdAt"`
    UpdatedAt    time.Time           `bson:"updatedAt"`
}
```

**Query Examples:**

```go
// Find all subchats for a parent
filter := bson.M{"parentChatId": parentSessionID}
cursor := collection.Find(ctx, filter)

// Find parent chat
var parentSession models.ChatSession
collection.FindOne(ctx, bson.M{"_id": session.ParentChatID}).Decode(&parentSession)
```

### 1.3 Work Delegation Benefits

**Coordinator Benefits:**
- ✅ Never reads/writes implementation files
- ✅ Only creates tasks and delegates
- ✅ Maintains high-level context
- ✅ Can monitor multiple subchats

**Agent Benefits:**
- ✅ Receives focused context (role, files, todos)
- ✅ Only has file operation tools (read, write, edit, bash)
- ✅ Cannot create more subchats (prevents infinite recursion)
- ✅ Can work independently without coordinator interference

---

## 2. Orchestrator and Implementer Separation

### 2.1 Architecture Principle

**Golden Rule:** Coordinator orchestrates, Agents implement.

```
┌─────────────────────────────────────┐
│  Coordinator (Orchestrator)         │
│  - Lists tasks                      │
│  - Creates tasks                    │
│  - Delegates to agents              │
│  - Monitors progress                │
│  - NO file operations               │
└─────────────┬───────────────────────┘
              │ execute_subagent
              ▼
┌─────────────────────────────────────┐
│  Specialist Agent (Implementer)     │
│  - Reads files                      │
│  - Writes code                      │
│  - Runs tests                       │
│  - Updates todos                    │
│  - NO task creation                 │
└─────────────────────────────────────┘
```

### 2.2 Tool Segregation

**Coordinator Tools (Main Chat):**
```go
// Location: hyper/internal/ai-service/ai_service.go:500-600
coordinatorTools := []string{
    // Task Management
    "coordinator_list_human_tasks",
    "coordinator_create_human_task",
    "coordinator_list_agent_tasks",
    "coordinator_create_agent_task",
    "coordinator_get_agent_task",
    "coordinator_update_task_status",
    "coordinator_update_todo_status",

    // Guidance
    "coordinator_add_task_prompt_notes",
    "coordinator_update_task_prompt_notes",
    "coordinator_clear_task_prompt_notes",
    "coordinator_add_todo_prompt_notes",
    "coordinator_update_todo_prompt_notes",
    "coordinator_clear_todo_prompt_notes",

    // Knowledge
    "coordinator_upsert_knowledge",
    "coordinator_query_knowledge",
    "coordinator_get_popular_collections",

    // Delegation
    "execute_subagent",

    // Discovery (LIMITED)
    "code_index_search",  // ONLY 1 call allowed

    // Admin
    "coordinator_clear_task_board",
}
```

**Agent Tools (Subchat):**
```go
// Location: coordinator_tools.go:2200-2250
func getAgentTools(agentName string) []string {
    baseTools := []string{
        // File Operations
        "read_file",
        "file_write",
        "apply_patch",
        "list_directory",

        // Execution
        "bash",

        // Progress Reporting
        "coordinator_update_todo_status",
        "coordinator_upsert_knowledge",
    }

    // Agent-specific tools
    switch agentName {
    case "ui-dev":
        return append(baseTools, "npm", "vite")
    case "go-dev":
        return append(baseTools, "go", "make")
    case "sre":
        return append(baseTools, "kubectl", "docker")
    default:
        return baseTools
    }
}
```

### 2.3 Enforcement Mechanisms

#### 2.3.1 Tool Filtering at Runtime

```go
// Location: coordinator_tools.go:2650-2700
func (t *ExecuteSubagentTool) Execute(ctx context.Context, args map[string]interface{}) {
    // Get allowed tools for this agent
    allowedTools := getAgentTools(agentTask.AgentName)

    // Filter out coordinator tools
    blockedTools := []string{
        "execute_subagent",             // Prevents nested subchats
        "coordinator_create_human_task",
        "coordinator_create_agent_task",
        "coordinator_list_human_tasks",
        "coordinator_list_agent_tasks",
        "create_agent_task",
    }

    // Stream with filtered tools
    aiStream := t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
}
```

#### 2.3.2 Security Validation Layer

**Location:** `hyper/internal/ai-service/langchain_service.go:1646-1663`

```go
// Pre-execution validation (before subchat starts)
func (s *LangchainService) validateSubchatSecurity(tools []string) error {
    blockedTools := []string{
        "execute_subagent",
        "coordinator_create_human_task",
        "coordinator_create_agent_task",
        "coordinator_list_human_tasks",
        "coordinator_list_agent_tasks",
        "create_agent_task",
    }

    for _, tool := range tools {
        for _, blocked := range blockedTools {
            if tool == blocked {
                return fmt.Errorf("🚫 SECURITY BLOCK: Tool '%s' is not allowed in subchat context", tool)
            }
        }
    }

    return nil
}
```

#### 2.3.3 Runtime Security Check

**Location:** `hyper/internal/ai-service/langchain_service.go:1954-1970`

```go
// Runtime check during tool execution (defense-in-depth)
func (s *LangchainService) checkToolSecurity(toolName string, isSubchat bool) error {
    if !isSubchat {
        return nil // Main chat can use any tool
    }

    blockedInSubchat := map[string]bool{
        "execute_subagent":             true,
        "coordinator_create_agent_task": true,
        "coordinator_create_human_task": true,
        "coordinator_list_agent_tasks":  true,
        "coordinator_list_human_tasks":  true,
        "create_agent_task":             true,
    }

    if blockedInSubchat[toolName] {
        return fmt.Errorf("🚫 SECURITY BLOCK: Tool '%s' is not allowed in subchat context", toolName)
    }

    return nil
}
```

### 2.4 System Prompt Differentiation

**Coordinator System Prompt:**
```
⛔ CRITICAL: YOU ARE A COORDINATOR - NOT AN IMPLEMENTER ⛔

You are a task orchestration AI. Your ONLY job is:
1. Create human tasks (record user requests)
2. Check for existing similar tasks
3. Create agent tasks with context
4. Delegate to specialist subagents

❌ YOU NEVER:
- Implement features yourself
- Read/write files directly for implementation
- Make multiple searches to "explore" or "understand" the codebase

✅ YOU ALWAYS:
- Create tasks immediately (within 5 tool calls total)
- Delegate all implementation work to subagents
- Trust the FIRST search results you get
```

**Agent System Prompt:**
```
You are {agentName}. You have been assigned a task to complete.

ROLE: {role}

TASK CONTEXT:
{contextSummary}

YOUR TODOs:
1. [PENDING] {todo1.description}
2. [PENDING] {todo2.description}
3. [PENDING] {todo3.description}

FILES TO MODIFY:
- {file1}
- {file2}

IMPORTANT:
- Start implementing within 2 minutes
- Update TODO status as you complete each one
- Use coordinator_update_todo_status tool
- Upsert knowledge when done
- DO NOT create more subchats
```

---

## 3. Subchat Interruption System

### 3.1 Interrupt Detection Architecture

**Location:** `hyper/internal/handlers/chat_websocket.go:1050-1100`

```go
// Priority interrupt handling via two-select pattern
func (h *ChatWebSocketHandler) streamAIResponse(conn *websocket.Conn, sessionID primitive.ObjectID) {
    progressCh := GetProgressNotifier(h.logger).RegisterSession(sessionID.Hex())
    aiStream := h.aiService.StreamChatWithTools(ctx, messages, maxToolCalls)

    for {
        select {
        // PRIORITY 1: User interrupts (checked FIRST)
        case userMsg := <-h.userMessageCh:
            if userMsg.SessionID == sessionID {
                h.logger.Info("🚨 USER INTERRUPT detected", zap.String("sessionId", sessionID.Hex()))

                // Categorize interrupt intent
                category, guidance := h.categorizeInterrupt(ctx, userMsg.Content)

                // Rebuild system prompt with interrupt guidance
                updatedPrompt := h.buildInterruptAwarePrompt(systemPrompt, category, guidance, userMsg.Content)

                // Restart AI stream with new context
                aiStream = h.aiService.StreamChatWithTools(ctx, updatedMessages, maxToolCalls)
                continue
            }

        // PRIORITY 2: Progress notifications
        case progress := <-progressCh:
            h.emitProgress(conn, progress)

        // PRIORITY 3: AI streaming events
        case event := <-aiStream:
            h.handleStreamEvent(conn, event)
        }
    }
}
```

### 3.2 Non-Blocking Message Notification

**Location:** `hyper/internal/handlers/message_notifier.go:1-115`

```go
// Singleton pattern for message notifications
type MessageNotifier struct {
    mu               sync.RWMutex
    sessionChannels  map[string]chan *models.Message
    logger           *zap.Logger
}

var (
    notifierInstance *MessageNotifier
    notifierOnce     sync.Once
)

func GetMessageNotifier(logger *zap.Logger) *MessageNotifier {
    notifierOnce.Do(func() {
        notifierInstance = &MessageNotifier{
            sessionChannels: make(map[string]chan *models.Message),
            logger:          logger,
        }
    })
    return notifierInstance
}

// Register session for notifications
func (n *MessageNotifier) RegisterSession(sessionID string) chan *models.Message {
    n.mu.Lock()
    defer n.mu.Unlock()

    ch := make(chan *models.Message, 100) // Buffered channel
    n.sessionChannels[sessionID] = ch

    n.logger.Info("Registered session for notifications", zap.String("sessionId", sessionID))
    return ch
}

// Notify all listeners (non-blocking)
func (n *MessageNotifier) NotifyMessage(sessionID string, message *models.Message) {
    n.mu.RLock()
    ch, exists := n.sessionChannels[sessionID]
    n.mu.RUnlock()

    if !exists {
        return
    }

    // Non-blocking send
    select {
    case ch <- message:
        n.logger.Debug("Message notified", zap.String("sessionId", sessionID))
    default:
        n.logger.Warn("Channel full, dropping notification", zap.String("sessionId", sessionID))
    }
}
```

### 3.3 Database-as-Source-of-Truth

```go
// Location: hyper/internal/handlers/chat_websocket.go:800-850
func (h *ChatWebSocketHandler) handleUserMessage(conn *websocket.Conn, sessionID primitive.ObjectID, content string) {
    // 1. SAVE TO DATABASE FIRST (source of truth)
    message := &models.Message{
        SessionID: sessionID,
        Role:      "user",
        Content:   content,
        Timestamp: time.Now(),
    }

    result, err := h.messageService.CreateMessage(ctx, message)
    if err != nil {
        h.logger.Error("Failed to save user message", zap.Error(err))
        return
    }

    // 2. NOTIFY all listeners (non-blocking)
    GetMessageNotifier(h.logger).NotifyMessage(sessionID.Hex(), message)

    // 3. Stream AI response
    // If AI is streaming, the two-select pattern will detect the notification
}
```

---

## 4. Progress Tracker with Pulsating Icons

### 4.1 Backend Progress Notification System

**Location:** `hyper/internal/handlers/progress_notifier.go:1-115`

```go
// Singleton progress notifier
type ProgressNotifier struct {
    mu               sync.RWMutex
    sessionChannels  map[string]chan ProgressEvent
    logger           *zap.Logger
}

type ProgressEvent struct {
    Message   string    `json:"message"`
    Timestamp time.Time `json:"timestamp"`
}

var (
    progressNotifierInstance *ProgressNotifier
    progressNotifierOnce     sync.Once
)

func GetProgressNotifier(logger *zap.Logger) *ProgressNotifier {
    progressNotifierOnce.Do(func() {
        progressNotifierInstance = &ProgressNotifier{
            sessionChannels: make(map[string]chan ProgressEvent),
            logger:          logger,
        }
    })
    return progressNotifierInstance
}

// Register session for progress updates
func (p *ProgressNotifier) RegisterSession(sessionID string) chan ProgressEvent {
    p.mu.Lock()
    defer p.mu.Unlock()

    ch := make(chan ProgressEvent, 50) // Buffered
    p.sessionChannels[sessionID] = ch

    p.logger.Info("Registered session for progress notifications", zap.String("sessionId", sessionID))
    return ch
}

// Emit progress (non-blocking)
func (p *ProgressNotifier) EmitProgress(sessionID string, message string) {
    p.mu.RLock()
    ch, exists := p.sessionChannels[sessionID]
    p.mu.RUnlock()

    if !exists {
        p.logger.Debug("No listeners for session", zap.String("sessionId", sessionID))
        return
    }

    event := ProgressEvent{
        Message:   message,
        Timestamp: time.Now(),
    }

    // Non-blocking send
    select {
    case ch <- event:
        p.logger.Debug("Progress emitted", zap.String("sessionId", sessionID), zap.String("message", message))
    default:
        p.logger.Warn("Progress channel full, dropping event", zap.String("sessionId", sessionID))
    }
}
```

### 4.2 Progress Emission Points

**Location:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go`

```go
// Emit progress during agent task execution
func (t *ExecuteSubagentTool) Execute(ctx context.Context, args map[string]interface{}) {
    // 1. Task created
    handlers.GetProgressNotifier(t.logger).EmitProgress(
        parentSessionID,
        fmt.Sprintf("📨 Agent task created: %s", agentTaskID),
    )

    // 2. Subchat started
    handlers.GetProgressNotifier(t.logger).EmitProgress(
        parentSessionID,
        fmt.Sprintf("🔄 Subagent started: %s", agentName),
    )

    // 3. Tool execution
    handlers.GetProgressNotifier(t.logger).EmitProgress(
        parentSessionID,
        fmt.Sprintf("🔧 Executing tool: %s", toolName),
    )

    // 4. TODO completed
    handlers.GetProgressNotifier(t.logger).EmitProgress(
        parentSessionID,
        fmt.Sprintf("✅ TODO %d completed", todoIndex),
    )

    // 5. Task complete
    handlers.GetProgressNotifier(t.logger).EmitProgress(
        parentSessionID,
        "🎉 Agent task completed successfully",
    )
}
```

### 4.3 Frontend Progress Display

**Location:** `ui/src/pages/CodeChatPage.tsx:150-200`

```typescript
// Progress state
const [progressMessages, setProgressMessages] = useState<Array<{
  message: string;
  timestamp: string;
  type: 'info' | 'success' | 'error';
}>>([]);

// WebSocket handler for progress
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);

  if (data.type === 'token') {
    // Check if this is a progress message (has \n\n wrappers)
    if (data.content.startsWith('\n\n') && data.content.endsWith('\n\n')) {
      const progressMsg = data.content.trim();

      // Determine type based on emoji
      let type = 'info';
      if (progressMsg.startsWith('✅')) type = 'success';
      if (progressMsg.startsWith('❌')) type = 'error';

      setProgressMessages(prev => [...prev, {
        message: progressMsg,
        timestamp: new Date().toISOString(),
        type,
      }]);

      return; // Don't add to regular message stream
    }

    // Regular token
    setStreamingContent(prev => prev + data.content);
  }
};
```

### 4.4 Progress Tracker Component

**Location:** `ui/src/components/ProgressTracker.tsx`

```typescript
interface ProgressTrackerProps {
  messages: Array<{
    message: string;
    timestamp: string;
    type: 'info' | 'success' | 'error';
  }>;
  visible: boolean;
}

export const ProgressTracker: React.FC<ProgressTrackerProps> = ({ messages, visible }) => {
  if (!visible || messages.length === 0) return null;

  return (
    <div className="fixed bottom-24 right-6 w-96 bg-white dark:bg-gray-800 border-2 border-blue-500 rounded-lg shadow-2xl p-4 max-h-80 overflow-y-auto animate-slide-in">
      <div className="flex items-center gap-2 mb-3 pb-2 border-b border-gray-200 dark:border-gray-700">
        {/* Pulsating icon */}
        <div className="relative">
          <div className="w-3 h-3 bg-blue-500 rounded-full animate-pulse"></div>
          <div className="absolute inset-0 w-3 h-3 bg-blue-500 rounded-full animate-ping"></div>
        </div>
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          Task Progress
        </h3>
      </div>

      <div className="space-y-2">
        {messages.map((msg, idx) => (
          <div
            key={idx}
            className="flex items-start gap-2 text-xs animate-fade-in"
            style={{ animationDelay: `${idx * 0.1}s` }}
          >
            {/* Status icon */}
            {msg.type === 'success' && (
              <span className="text-green-500 text-base">✅</span>
            )}
            {msg.type === 'error' && (
              <span className="text-red-500 text-base">❌</span>
            )}
            {msg.type === 'info' && (
              <span className="text-blue-500 text-base">🔄</span>
            )}

            {/* Message */}
            <div className="flex-1">
              <p className="text-gray-800 dark:text-gray-200">{msg.message}</p>
              <p className="text-gray-500 dark:text-gray-400 text-xs mt-1">
                {new Date(msg.timestamp).toLocaleTimeString()}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};
```

### 4.5 Pulsating Animation CSS

```css
/* Tailwind CSS animations */
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

@keyframes ping {
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
}

@keyframes slide-in {
  from {
    transform: translateX(100%);
    opacity: 0;
  }
  to {
    transform: translateX(0);
    opacity: 1;
  }
}

@keyframes fade-in {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

.animate-ping {
  animation: ping 1s cubic-bezier(0, 0, 0.2, 1) infinite;
}

.animate-slide-in {
  animation: slide-in 0.3s ease-out;
}

.animate-fade-in {
  animation: fade-in 0.5s ease-out forwards;
}
```

---

## 5. File Indexing for Code Search

### 5.1 Async File Indexer Architecture

**Location:** `hyper/internal/mcp/indexer/auto_indexer.go:1-224`

```go
// Async file indexer with queue-based processing
type AutoIndexer struct {
    codeIndexService *services.CodeIndexService
    eventQueue       chan IndexEvent
    stopCh           chan struct{}
    wg               sync.WaitGroup
    logger           *zap.Logger
}

type IndexEvent struct {
    Type      string // "add_folder", "remove_folder", "scan"
    FolderPath string
    Timestamp  time.Time
}

// Initialize auto indexer
func NewAutoIndexer(codeIndexService *services.CodeIndexService, logger *zap.Logger) *AutoIndexer {
    indexer := &AutoIndexer{
        codeIndexService: codeIndexService,
        eventQueue:       make(chan IndexEvent, 100), // Buffered queue
        stopCh:           make(chan struct{}),
        logger:           logger,
    }

    // Start background worker
    indexer.wg.Add(1)
    go indexer.processEventQueue()

    return indexer
}

// Non-blocking event processing
func (ai *AutoIndexer) processEventQueue() {
    defer ai.wg.Done()

    for {
        select {
        case event := <-ai.eventQueue:
            ai.handleEvent(event)
        case <-ai.stopCh:
            return
        }
    }
}

// Handle indexing event
func (ai *AutoIndexer) handleEvent(event IndexEvent) {
    switch event.Type {
    case "add_folder":
        ai.logger.Info("📁 Adding folder to index", zap.String("path", event.FolderPath))
        ai.codeIndexService.AddFolder(event.FolderPath)

    case "remove_folder":
        ai.logger.Info("🗑️ Removing folder from index", zap.String("path", event.FolderPath))
        ai.codeIndexService.RemoveFolder(event.FolderPath)

    case "scan":
        ai.logger.Info("🔍 Scanning folder", zap.String("path", event.FolderPath))
        ai.codeIndexService.Scan(event.FolderPath)
    }
}

// Enqueue event (non-blocking)
func (ai *AutoIndexer) QueueEvent(eventType string, folderPath string) {
    event := IndexEvent{
        Type:       eventType,
        FolderPath: folderPath,
        Timestamp:  time.Now(),
    }

    select {
    case ai.eventQueue <- event:
        ai.logger.Debug("Event queued", zap.String("type", eventType), zap.String("path", folderPath))
    default:
        ai.logger.Warn("Event queue full, dropping event", zap.String("type", eventType))
    }
}
```

### 5.2 Code Index Service

**Location:** `hyper/internal/services/code_index_service.go`

```go
type CodeIndexService struct {
    mu          sync.RWMutex
    folders     map[string]bool       // Indexed folders
    fileIndex   map[string]*FileInfo  // File path → metadata
    symbolIndex map[string][]Symbol   // Symbol name → locations
    logger      *zap.Logger
}

type FileInfo struct {
    Path         string
    Language     string
    Size         int64
    LastModified time.Time
    Content      string
    Symbols      []Symbol
}

type Symbol struct {
    Name     string
    Kind     string // "function", "class", "variable", "interface"
    FilePath string
    Line     int
    Column   int
}

// Add folder to index
func (s *CodeIndexService) AddFolder(folderPath string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.folders[folderPath] {
        return fmt.Errorf("folder already indexed: %s", folderPath)
    }

    s.folders[folderPath] = true
    s.logger.Info("Folder added to index", zap.String("path", folderPath))

    // Trigger background scan
    go s.scanFolder(folderPath)

    return nil
}

// Scan folder for files
func (s *CodeIndexService) scanFolder(folderPath string) {
    filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }

        if info.IsDir() {
            // Skip node_modules, .git, etc.
            if shouldSkipDir(info.Name()) {
                return filepath.SkipDir
            }
            return nil
        }

        // Index file
        if isCodeFile(path) {
            s.indexFile(path)
        }

        return nil
    })
}

// Index individual file
func (s *CodeIndexService) indexFile(filePath string) error {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }

    language := detectLanguage(filePath)
    symbols := extractSymbols(content, language)

    fileInfo := &FileInfo{
        Path:         filePath,
        Language:     language,
        Size:         int64(len(content)),
        LastModified: time.Now(),
        Content:      string(content),
        Symbols:      symbols,
    }

    s.mu.Lock()
    s.fileIndex[filePath] = fileInfo

    // Update symbol index
    for _, symbol := range symbols {
        s.symbolIndex[symbol.Name] = append(s.symbolIndex[symbol.Name], symbol)
    }
    s.mu.Unlock()

    return nil
}

// Search code index
func (s *CodeIndexService) Search(query string, limit int) ([]SearchResult, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    results := []SearchResult{}

    // 1. Exact symbol match
    if symbols, found := s.symbolIndex[query]; found {
        for _, symbol := range symbols {
            results = append(results, SearchResult{
                FilePath:  symbol.FilePath,
                StartLine: symbol.Line,
                MatchType: "symbol",
                Symbol:    symbol,
            })
        }
    }

    // 2. Fuzzy file name match
    for path, fileInfo := range s.fileIndex {
        if strings.Contains(strings.ToLower(path), strings.ToLower(query)) {
            results = append(results, SearchResult{
                FilePath:  path,
                MatchType: "filename",
                FileInfo:  fileInfo,
            })
        }
    }

    // 3. Content search (if results < limit)
    if len(results) < limit {
        for path, fileInfo := range s.fileIndex {
            if strings.Contains(fileInfo.Content, query) {
                lines := strings.Split(fileInfo.Content, "\n")
                for i, line := range lines {
                    if strings.Contains(line, query) {
                        results = append(results, SearchResult{
                            FilePath:  path,
                            StartLine: i + 1,
                            MatchType: "content",
                            Content:   line,
                        })
                    }
                }
            }
        }
    }

    // Sort by relevance
    sort.Slice(results, func(i, j int) bool {
        return results[i].Relevance() > results[j].Relevance()
    })

    // Limit results
    if len(results) > limit {
        results = results[:limit]
    }

    return results, nil
}
```

### 5.3 Symbol Extraction (Tree-sitter)

```go
// Extract symbols using tree-sitter
func extractSymbols(content []byte, language string) []Symbol {
    var parser *sitter.Parser

    switch language {
    case "go":
        parser = sitter.NewParser()
        parser.SetLanguage(golang.GetLanguage())
    case "typescript":
        parser = sitter.NewParser()
        parser.SetLanguage(typescript.GetLanguage())
    case "python":
        parser = sitter.NewParser()
        parser.SetLanguage(python.GetLanguage())
    default:
        return nil
    }

    tree, _ := parser.ParseCtx(context.Background(), nil, content)
    defer tree.Close()

    symbols := []Symbol{}

    // Walk AST and extract symbols
    cursor := sitter.NewTreeCursor(tree.RootNode())
    defer cursor.Close()

    visitNode(cursor, func(node *sitter.Node) {
        switch node.Type() {
        case "function_declaration", "method_declaration":
            symbols = append(symbols, Symbol{
                Name:   getNodeText(node, content, "name"),
                Kind:   "function",
                Line:   int(node.StartPoint().Row) + 1,
                Column: int(node.StartPoint().Column),
            })

        case "class_declaration", "interface_declaration":
            symbols = append(symbols, Symbol{
                Name:   getNodeText(node, content, "name"),
                Kind:   "class",
                Line:   int(node.StartPoint().Row) + 1,
                Column: int(node.StartPoint().Column),
            })

        case "variable_declaration":
            symbols = append(symbols, Symbol{
                Name:   getNodeText(node, content, "name"),
                Kind:   "variable",
                Line:   int(node.StartPoint().Row) + 1,
                Column: int(node.StartPoint().Column),
            })
        }
    })

    return symbols
}
```

---

## 6. Orchestrator Validations

### 6.1 TODO Validation (Discovery Keywords)

**Location:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go:1100-1150`

```go
// Validate todos don't contain exploratory keywords
func validateTodos(todos []models.Todo) error {
    // Forbidden keywords that indicate discovery (not implementation)
    discoveryKeywords := []string{
        "search", "find", "locate", "discover", "look for", "explore",
        "code_index_search", "list_directory", "investigate", "inspect",
    }

    for i, todo := range todos {
        lowerDesc := strings.ToLower(todo.Description)

        for _, keyword := range discoveryKeywords {
            if strings.Contains(lowerDesc, keyword) {
                return fmt.Errorf(
                    "❌ TODO validation failed: TODO #%d contains discovery keyword '%s'. "+
                    "Subagents cannot search or discover files. "+
                    "YOU must complete all discovery work using code_index_search. "+
                    "Subagents only receive specific file paths and line numbers to modify. "+
                    "Change TODO to implementation step like 'Update X.tsx line 45'",
                    i+1, keyword,
                )
            }
        }

        // Check for questions (indicates uncertainty)
        if strings.Contains(lowerDesc, "?") {
            return fmt.Errorf(
                "❌ TODO validation failed: TODO #%d contains question mark. "+
                "TODOs must be definitive implementation steps, not questions. "+
                "Research the answer using code_index_search before creating the task.",
                i+1,
            )
        }
    }

    return nil
}
```

### 6.2 File Path Validation

```go
// Validate file paths exist and are from FILE_PATHS_TO_USE array
func (t *CoordinatorCreateAgentTaskTool) validateFilePaths(filesModified []string, searchResults []SearchResult) error {
    validPaths := make(map[string]bool)

    // Extract valid paths from search results
    for _, result := range searchResults {
        validPaths[result.FilePath] = true
    }

    // Check each file path
    for _, filePath := range filesModified {
        // Must be from search results
        if !validPaths[filePath] {
            return fmt.Errorf(
                "❌ File path validation failed: '%s' not in FILE_PATHS_TO_USE array. "+
                "You must ONLY use exact paths from code_index_search results. "+
                "DO NOT type file paths manually. "+
                "Copy-paste from FILE_PATHS_TO_USE array.",
                filePath,
            )
        }

        // Must exist on filesystem
        if _, err := os.Stat(filePath); os.IsNotExist(err) {
            return fmt.Errorf(
                "❌ File path validation failed: '%s' does not exist. "+
                "Path from search results but file not found. "+
                "Re-run code_index_search with updated query.",
                filePath,
            )
        }
    }

    return nil
}
```

### 6.3 Task Duplication Check

```go
// Check for similar existing tasks
func (t *CoordinatorCreateHumanTaskTool) checkSimilarTasks(ctx context.Context, prompt string) ([]models.HumanTask, error) {
    // Find pending tasks
    filter := bson.M{"status": "pending"}
    cursor, err := t.collection.Find(ctx, filter)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var pendingTasks []models.HumanTask
    cursor.All(ctx, &pendingTasks)

    // Calculate similarity scores
    similarTasks := []models.HumanTask{}
    for _, task := range pendingTasks {
        similarity := calculateSimilarity(prompt, task.Prompt)
        if similarity > 0.7 { // 70% similar
            similarTasks = append(similarTasks, task)
        }
    }

    return similarTasks, nil
}

// Calculate text similarity (cosine similarity)
func calculateSimilarity(text1, text2 string) float64 {
    words1 := strings.Fields(strings.ToLower(text1))
    words2 := strings.Fields(strings.ToLower(text2))

    // Build word frequency maps
    freq1 := make(map[string]int)
    freq2 := make(map[string]int)

    for _, word := range words1 {
        freq1[word]++
    }
    for _, word := range words2 {
        freq2[word]++
    }

    // Calculate dot product and magnitudes
    dotProduct := 0.0
    mag1 := 0.0
    mag2 := 0.0

    for word, count1 := range freq1 {
        if count2, found := freq2[word]; found {
            dotProduct += float64(count1 * count2)
        }
        mag1 += float64(count1 * count1)
    }

    for _, count2 := range freq2 {
        mag2 += float64(count2 * count2)
    }

    // Cosine similarity
    if mag1 == 0 || mag2 == 0 {
        return 0
    }

    return dotProduct / (math.Sqrt(mag1) * math.Sqrt(mag2))
}
```

### 6.4 Circuit Breaker (Prevent Infinite Loops)

```go
// Circuit breaker to prevent repeated tool calls
type CircuitBreaker struct {
    mu           sync.Mutex
    toolCalls    map[string][]time.Time
    maxCalls     int
    timeWindow   time.Duration
}

func NewCircuitBreaker() *CircuitBreaker {
    return &CircuitBreaker{
        toolCalls:  make(map[string][]time.Time),
        maxCalls:   3,
        timeWindow: 5 * time.Minute,
    }
}

func (cb *CircuitBreaker) CheckToolCall(toolName string, args string) error {
    cb.mu.Lock()
    defer cb.mu.Unlock()

    key := toolName + ":" + args
    now := time.Now()

    // Get recent calls
    calls := cb.toolCalls[key]

    // Filter calls within time window
    recentCalls := []time.Time{}
    for _, callTime := range calls {
        if now.Sub(callTime) < cb.timeWindow {
            recentCalls = append(recentCalls, callTime)
        }
    }

    // Check if limit exceeded
    if len(recentCalls) >= cb.maxCalls {
        return fmt.Errorf(
            "🚨 CIRCUIT BREAKER TRIGGERED: Tool '%s' called %d times in %v with same arguments. "+
            "You are stuck in a loop! "+
            "STOP trying the same approach. "+
            "Either: (1) delegate to subagent, (2) try different tool, (3) ask user for help.",
            toolName, cb.maxCalls, cb.timeWindow,
        )
    }

    // Record this call
    recentCalls = append(recentCalls, now)
    cb.toolCalls[key] = recentCalls

    return nil
}
```

### 6.5 Workflow Step Enforcement

```go
// Prescriptive workflow filter (Phase 3)
func (s *LangchainService) applyPrescriptiveFilter(tools []string, iteration int, history []ToolCall) []string {
    // Step 0: Must list human tasks first
    if iteration == 1 && len(history) == 0 {
        return []string{"coordinator_list_human_tasks"}
    }

    // Step 1: Must create human task after listing
    if iteration == 2 && containsToolCall(history, "coordinator_list_human_tasks") {
        return []string{"coordinator_create_human_task"}
    }

    // Step 2: Must do code search after creating human task
    if iteration == 3 && containsToolCall(history, "coordinator_create_human_task") {
        return []string{"code_index_search"}
    }

    // Step 3: Must create agent task after code search
    if iteration == 4 && containsToolCall(history, "code_index_search") {
        return []string{"coordinator_create_agent_task"}
    }

    // Step 4: Must execute subagent after creating agent task
    if iteration == 5 && containsToolCall(history, "coordinator_create_agent_task") {
        return []string{"execute_subagent"}
    }

    // After step 5, allow all tools
    return tools
}
```

---

## 7. Executor Phase Guards

### 7.1 Two-Layer Security Architecture

#### Layer 1: Pre-Execution Validation

**Location:** `hyper/internal/ai-service/langchain_service.go:1646-1663`

```go
// Validate tool allowlist before starting subchat
func (s *LangchainService) validateSubchatTools(agentName string, requestedTools []string) error {
    allowedTools := getAgentTools(agentName)
    allowedMap := make(map[string]bool)
    for _, tool := range allowedTools {
        allowedMap[tool] = true
    }

    blockedTools := []string{
        "execute_subagent",
        "coordinator_create_human_task",
        "coordinator_create_agent_task",
        "coordinator_list_human_tasks",
        "coordinator_list_agent_tasks",
        "create_agent_task",
    }

    for _, tool := range requestedTools {
        // Check if blocked
        for _, blocked := range blockedTools {
            if tool == blocked {
                return fmt.Errorf(
                    "🚫 SECURITY BLOCK: Tool '%s' is not allowed in subchat context. "+
                    "Subchats cannot create more subchats (prevents infinite recursion). "+
                    "Subchats use predefined tasks only.",
                    tool,
                )
            }
        }

        // Check if allowed for this agent
        if !allowedMap[tool] {
            return fmt.Errorf(
                "🚫 SECURITY BLOCK: Tool '%s' is not allowed for agent '%s'. "+
                "This agent is restricted to: %v",
                tool, agentName, allowedTools,
            )
        }
    }

    s.logger.Info("✅ Security validated: All tools allowed for agent",
        zap.String("agentName", agentName),
        zap.Int("toolCount", len(requestedTools)))

    return nil
}
```

#### Layer 2: Runtime Security Check

**Location:** `hyper/internal/ai-service/langchain_service.go:1954-1970`

```go
// Runtime check during tool execution (defense-in-depth)
func (s *LangchainService) checkRuntimeSecurity(toolName string, isSubchat bool) error {
    if !isSubchat {
        return nil // Main chat has no restrictions
    }

    // Block coordinator tools at runtime (catches AI violations)
    blockedInSubchat := map[string]bool{
        "execute_subagent":             true,
        "coordinator_create_agent_task": true,
        "coordinator_create_human_task": true,
        "coordinator_list_agent_tasks":  true,
        "coordinator_list_human_tasks":  true,
        "create_agent_task":             true,
    }

    if blockedInSubchat[toolName] {
        s.logger.Error("🚫 Runtime security violation detected",
            zap.String("tool", toolName),
            zap.Bool("isSubchat", isSubchat))

        return fmt.Errorf(
            "🚫 SECURITY BLOCK: Tool '%s' is not allowed in subchat context. "+
            "This tool was blocked at runtime because the AI attempted to call it despite pre-execution validation. "+
            "This is a defense-in-depth measure to prevent cascade failures.",
            toolName,
        )
    }

    return nil
}
```

### 7.2 File Operation Guards

```go
// Validate file operations are within project scope
func validateFileOperation(filePath string, operation string) error {
    // Get project root
    projectRoot := "/Users/meghaneelamana/dev-squad"

    // Resolve absolute path
    absPath, err := filepath.Abs(filePath)
    if err != nil {
        return fmt.Errorf("invalid file path: %w", err)
    }

    // Must be within project root
    if !strings.HasPrefix(absPath, projectRoot) {
        return fmt.Errorf(
            "🚫 SECURITY BLOCK: File operation denied. "+
            "Path '%s' is outside project root '%s'. "+
            "Only files within the project can be modified.",
            absPath, projectRoot,
        )
    }

    // Block system directories
    blockedDirs := []string{"/etc", "/var", "/sys", "/usr", "/bin", "/sbin"}
    for _, blocked := range blockedDirs {
        if strings.HasPrefix(absPath, blocked) {
            return fmt.Errorf(
                "🚫 SECURITY BLOCK: File operation denied. "+
                "Path '%s' is in system directory '%s'. "+
                "System directories are protected.",
                absPath, blocked,
            )
        }
    }

    // Check operation type
    if operation == "write" || operation == "delete" {
        // Block critical files
        criticalFiles := []string{
            ".env",
            ".env.production",
            "credentials.json",
            "private.key",
            ".git/config",
        }

        for _, critical := range criticalFiles {
            if strings.Contains(absPath, critical) {
                return fmt.Errorf(
                    "🚫 SECURITY BLOCK: File operation denied. "+
                    "Cannot %s critical file: %s",
                    operation, filepath.Base(absPath),
                )
            }
        }
    }

    return nil
}
```

### 7.3 Resource Limits

```go
// Enforce resource limits for agent execution
type ResourceLimits struct {
    MaxToolCalls    int
    MaxFileReads    int
    MaxFileWrites   int
    MaxBashCommands int
    MaxExecutionTime time.Duration
}

func getAgentResourceLimits(agentName string) ResourceLimits {
    switch agentName {
    case "ui-dev", "go-dev":
        return ResourceLimits{
            MaxToolCalls:    30,
            MaxFileReads:    20,
            MaxFileWrites:   10,
            MaxBashCommands: 5,
            MaxExecutionTime: 10 * time.Minute,
        }
    case "sre":
        return ResourceLimits{
            MaxToolCalls:    20,
            MaxFileReads:    10,
            MaxFileWrites:   5,
            MaxBashCommands: 15,
            MaxExecutionTime: 15 * time.Minute,
        }
    default:
        return ResourceLimits{
            MaxToolCalls:    15,
            MaxFileReads:    10,
            MaxFileWrites:   5,
            MaxBashCommands: 3,
            MaxExecutionTime: 5 * time.Minute,
        }
    }
}

// Check resource usage
func (s *LangchainService) checkResourceUsage(agentName string, usage ResourceUsage) error {
    limits := getAgentResourceLimits(agentName)

    if usage.ToolCalls >= limits.MaxToolCalls {
        return fmt.Errorf(
            "🚫 RESOURCE LIMIT: Max tool calls exceeded (%d/%d). "+
            "Agent may be stuck in a loop. "+
            "Review your approach and simplify.",
            usage.ToolCalls, limits.MaxToolCalls,
        )
    }

    if usage.FileReads >= limits.MaxFileReads {
        return fmt.Errorf(
            "🚫 RESOURCE LIMIT: Max file reads exceeded (%d/%d). "+
            "You're reading too many files. "+
            "Focus on the specific files needed for the task.",
            usage.FileReads, limits.MaxFileReads,
        )
    }

    if usage.FileWrites >= limits.MaxFileWrites {
        return fmt.Errorf(
            "🚫 RESOURCE LIMIT: Max file writes exceeded (%d/%d). "+
            "You're modifying too many files at once. "+
            "Break the task into smaller pieces.",
            usage.FileWrites, limits.MaxFileWrites,
        )
    }

    if time.Since(usage.StartTime) > limits.MaxExecutionTime {
        return fmt.Errorf(
            "🚫 RESOURCE LIMIT: Execution timeout (%.0f minutes). "+
            "Task took too long. "+
            "Review your approach or break into smaller tasks.",
            limits.MaxExecutionTime.Minutes(),
        )
    }

    return nil
}
```

### 7.4 MongoDB Transaction Guardrails

**Location:** `hyper/internal/services/chat_service.go:200-250`

```go
// Atomic message operations with transactions
func (s *ChatService) CreateMessageWithTransaction(ctx context.Context, message *models.Message) error {
    session, err := s.client.StartSession()
    if err != nil {
        return fmt.Errorf("failed to start MongoDB session: %w", err)
    }
    defer session.EndSession(ctx)

    // Define transaction callback
    callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
        // 1. Validate message size (max 1MB)
        if len(message.Content) > 1024*1024 {
            return nil, fmt.Errorf("message content exceeds 1MB limit")
        }

        // 2. Validate session exists
        var session models.ChatSession
        err := s.sessionCollection.FindOne(sessCtx, bson.M{"_id": message.SessionID}).Decode(&session)
        if err != nil {
            return nil, fmt.Errorf("session not found: %w", err)
        }

        // 3. Insert message
        result, err := s.messageCollection.InsertOne(sessCtx, message)
        if err != nil {
            return nil, fmt.Errorf("failed to insert message: %w", err)
        }

        // 4. Update session timestamp
        update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
        _, err = s.sessionCollection.UpdateOne(sessCtx, bson.M{"_id": message.SessionID}, update)
        if err != nil {
            return nil, fmt.Errorf("failed to update session: %w", err)
        }

        return result.InsertedID, nil
    }

    // Execute transaction with retries
    _, err = session.WithTransaction(ctx, callback)
    if err != nil {
        return fmt.Errorf("transaction failed: %w", err)
    }

    return nil
}
```

---

## 8. Detailed Subchat Interruption Implementation

### 8.1 Intelligent Interrupt Categorization

**Location:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go:4250-4336`

```go
// Interrupt categorization struct
type InterruptCategorization struct {
    Category string `json:"category"` // "STOP", "MODIFY", "CLARIFY", "STATUS", "CONTINUE"
    Guidance string `json:"guidance"` // Brief instruction for AI
}

// Categorize interrupt intent using Claude API
func (t *ExecuteSubagentTool) categorizeInterrupt(ctx context.Context, userMessage string) (string, string, error) {
    categorizationPrompt := fmt.Sprintf(`You are an interrupt analyzer. Analyze this user message sent while an AI agent was working:

User message: "%s"

Categorize the interrupt intent:
- STOP: User wants to completely stop current work and do something different
- MODIFY: User wants to change/adjust the current approach
- CLARIFY: User has a question or needs clarification
- STATUS: User checking progress or giving encouragement
- CONTINUE: Message doesn't require action change

Respond with ONLY valid JSON (no markdown, no explanation):
{
  "category": "STOP|MODIFY|CLARIFY|STATUS|CONTINUE",
  "guidance": "Brief instruction for the agent (1 sentence)"
}`, userMessage)

    // Use Claude API for categorization
    messages := []aiservice.Message{
        {Role: "user", Content: categorizationPrompt},
    }

    t.logger.Debug("Categorizing interrupt", zap.String("userMessage", userMessage))

    // Stream API call
    stream, err := t.aiService.StreamChatWithTools(ctx, messages, 1)
    if err != nil {
        t.logger.Warn("Failed to start categorization stream", zap.Error(err))
        return "CONTINUE", "", err
    }

    // Collect full response
    var response strings.Builder
    for event := range stream {
        if event.Type == aiservice.StreamEventToken {
            response.WriteString(event.Content)
        }
    }

    responseStr := response.String()
    t.logger.Debug("Raw categorization response", zap.String("response", responseStr))

    // Extract JSON (handle markdown code blocks)
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

    // Parse JSON
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
```

### 8.2 Interrupt Handler Integration

**Location:** `hyper/internal/ai-service/tools/mcp/coordinator_tools.go:2947-3094`

```go
// Handle interrupt during subchat execution
func (t *ExecuteSubagentTool) handleInterrupt(ctx context.Context, chatSession *models.ChatSession) {
    // 1. Refetch messages to get latest user interrupt
    messagesResp, err := t.getMessages(ctx, chatSession.ID)
    if err != nil {
        t.logger.Error("Failed to refetch messages after interrupt", zap.Error(err))
        return
    }

    // 2. Extract latest user message
    var latestUserMessage string
    for i := len(messagesResp.Messages) - 1; i >= 0; i-- {
        if messagesResp.Messages[i].Role == "user" {
            latestUserMessage = messagesResp.Messages[i].Content
            break
        }
    }

    if latestUserMessage == "" {
        t.logger.Warn("No user message found in interrupt refetch")
        return
    }

    // 3. Categorize the interrupt
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

    // 4. Emit progress notification to parent chat
    parentSessionID := chatSession.ParentChatID.Hex()
    handlers.GetProgressNotifier(t.logger).EmitProgress(parentSessionID,
        fmt.Sprintf("📨 User interrupt received: %s", category))

    // 5. Build interrupt-aware system prompt guidance
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
❓ USER INTERRUPT - CLARIFY
The user has a question or needs clarification.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. Answer the user's question directly and clearly
2. Ask if they want you to continue with the current task
3. Wait for their confirmation before proceeding

Start your response with: "[directly address their question]"
`, latestUserMessage, guidance)

    case "STATUS":
        interruptGuidance = fmt.Sprintf(`
📊 USER INTERRUPT - STATUS CHECK
The user is checking progress or giving encouragement.

User's message: "%s"

AI Analysis: %s

YOU MUST:
1. Give a brief status update (2-3 sentences max)
2. Acknowledge their message warmly
3. Continue with your work

Start your response with: "Quick update: [brief status]. Continuing..."
`, latestUserMessage, guidance)

    case "CONTINUE":
        interruptGuidance = fmt.Sprintf(`
✅ USER MESSAGE - CONTINUE
The user sent a message but it doesn't require stopping or major changes.

User's message: "%s"

AI Analysis: %s

YOU SHOULD:
1. Briefly acknowledge if relevant (1 sentence max)
2. Continue with your current work

If relevant, start with: "[1 sentence acknowledgment]. Continuing with task..."
`, latestUserMessage, guidance)
    }

    // 6. Rebuild message context with interrupt-aware system prompt
    messages := []aiservice.Message{
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

    // 7. Restart AI stream with updated context
    aiStream, err := t.aiService.StreamChatWithToolsFiltered(ctx, messages, maxToolCalls, allowedTools)
    if err != nil {
        t.logger.Error("Failed to restart AI stream after interrupt", zap.Error(err))
        return
    }

    // 8. Continue streaming with interrupt-aware context
    t.streamAIResponse(ctx, aiStream, chatSession)
}
```

### 8.3 Frontend Interrupt Display

**Location:** `ui2/src/pages/CodeChatPage.tsx`

```typescript
// Enable input for subchats (allow interruption)
const isSubchat = !!activeSession?.parentChatId || activeSession?.title.startsWith('Subchat:');

<ChatInput
  onSendMessage={handleSendMessage}
  disabled={!activeSessionId || isStreaming}  // No subchat check - allow interrupts
  placeholder={
    !activeSessionId
      ? 'Create a new chat to get started'
      : 'Type your message...'  // Same for main and subchat
  }
/>
```

**Interrupt Flow:**
```
1. User types message in subchat → handleSendMessage()
2. Message saved to MongoDB → MessageNotifier notifies
3. Backend detects interrupt → Categorizes intent (STOP/MODIFY/CLARIFY/STATUS/CONTINUE)
4. System prompt updated → AI restarted with interrupt guidance
5. AI responds appropriately → Progress notification sent to parent chat
```

---

## 9. Prometheus Dashboard

### 9.1 Metrics Registry

**Location:** `hyper/internal/metrics/registry.go:1-269`

```go
// Prometheus metrics registry
type MetricsRegistry struct {
    // WebSocket metrics
    WSConnectionsTotal       prometheus.Counter
    WSConnectionsActive      prometheus.Gauge
    WSMessagesReceived       prometheus.Counter
    WSMessagesSent           prometheus.Counter
    WSErrors                 prometheus.Counter

    // Message validation metrics
    ValidationSuccess        prometheus.Counter
    ValidationFailedSize     prometheus.Counter
    ValidationFailedFormat   prometheus.Counter

    // Chat metrics
    ChatMessagesTotal        prometheus.Counter
    ChatSessionsTotal        prometheus.Counter
    ChatMessagesSuccess      prometheus.Counter
    ChatMessagesFailed       prometheus.Counter

    // AI streaming metrics
    AIStreamTokens           prometheus.Counter
    AIStreamDuration         prometheus.Histogram
    AIToolCalls              prometheus.Counter
    AIToolCallDuration       prometheus.Histogram

    // HTTP metrics
    HTTPRequestsTotal        prometheus.Counter
    HTTPRequestDuration      prometheus.Histogram
    HTTPErrorsTotal          prometheus.Counter

    // MongoDB metrics
    MongoOperationsTotal     prometheus.Counter
    MongoOperationDuration   prometheus.Histogram
    MongoErrors              prometheus.Counter
}

// Initialize metrics
func NewMetricsRegistry() *MetricsRegistry {
    registry := &MetricsRegistry{
        // WebSocket
        WSConnectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hyperion_ws_connections_total",
            Help: "Total number of WebSocket connections",
        }),
        WSConnectionsActive: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "hyperion_ws_connections_active",
            Help: "Number of active WebSocket connections",
        }),

        // Chat
        ChatMessagesTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hyperion_chat_messages_total",
            Help: "Total number of chat messages",
        }),

        // AI Streaming
        AIStreamTokens: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hyperion_ai_stream_tokens_total",
            Help: "Total number of tokens streamed",
        }),
        AIStreamDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "hyperion_ai_stream_duration_seconds",
            Help:    "Duration of AI streaming requests",
            Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
        }),

        // HTTP
        HTTPRequestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hyperion_http_requests_total",
            Help: "Total number of HTTP requests",
        }),
        HTTPRequestDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "hyperion_http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        }),

        // MongoDB
        MongoOperationsTotal: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "hyperion_mongo_operations_total",
            Help: "Total number of MongoDB operations",
        }),
        MongoOperationDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "hyperion_mongo_operation_duration_seconds",
            Help:    "MongoDB operation duration",
            Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1, 5},
        }),
    }

    // Register with Prometheus
    prometheus.MustRegister(registry.WSConnectionsTotal)
    prometheus.MustRegister(registry.WSConnectionsActive)
    prometheus.MustRegister(registry.ChatMessagesTotal)
    prometheus.MustRegister(registry.AIStreamTokens)
    prometheus.MustRegister(registry.AIStreamDuration)
    prometheus.MustRegister(registry.HTTPRequestsTotal)
    prometheus.MustRegister(registry.HTTPRequestDuration)
    prometheus.MustRegister(registry.MongoOperationsTotal)
    prometheus.MustRegister(registry.MongoOperationDuration)

    return registry
}
```

### 9.2 Metrics Collection

```go
// Collect metrics during operations
func (h *ChatWebSocketHandler) handleConnection(conn *websocket.Conn) {
    // Increment connections
    h.metrics.WSConnectionsTotal.Inc()
    h.metrics.WSConnectionsActive.Inc()
    defer h.metrics.WSConnectionsActive.Dec()

    // Track duration
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        h.logger.Info("WebSocket closed", zap.Duration("duration", time.Since(start)))
    }()

    // Handle messages
    for {
        var msg WebSocketMessage
        err := conn.ReadJSON(&msg)
        if err != nil {
            h.metrics.WSErrors.Inc()
            break
        }

        h.metrics.WSMessagesReceived.Inc()
        h.handleMessage(conn, msg)
    }
}

// Track AI streaming
func (s *LangchainService) StreamChatWithTools(ctx context.Context, messages []Message, maxToolCalls int) chan StreamEvent {
    start := time.Now()
    tokenCount := 0

    stream := make(chan StreamEvent, 100)

    go func() {
        defer func() {
            // Record metrics
            duration := time.Since(start).Seconds()
            s.metrics.AIStreamDuration.Observe(duration)
            s.metrics.AIStreamTokens.Add(float64(tokenCount))
            close(stream)
        }()

        // Stream tokens
        for token := range aiStream {
            tokenCount++
            stream <- StreamEvent{Type: "token", Content: token}
        }
    }()

    return stream
}
```

### 9.3 Metrics Endpoint

**Location:** `hyper/internal/server/http_server.go:150-160`

```go
// Expose /metrics endpoint
func (s *HTTPServer) setupRoutes() {
    // Prometheus metrics
    s.router.Handle("/metrics", promhttp.Handler())

    // Health check
    s.router.HandleFunc("/health", s.handleHealth)

    // API routes
    s.router.HandleFunc("/api/chat/sessions", s.handleSessions)
    // ...
}
```

### 9.4 Frontend Dashboard

**Location:** `ui/src/components/MetricsDashboard.tsx:1-392`

```typescript
// Fetch and display Prometheus metrics
export function MetricsDashboard() {
  const [metrics, setMetrics] = useState<ParsedMetrics | null>(null);
  const [loading, setLoading] = useState(true);

  // Auto-refresh every 5 seconds
  useEffect(() => {
    const fetchMetrics = async () => {
      const response = await fetch('http://localhost:5555/metrics');
      const text = await response.text();
      const parsed = parsePrometheusMetrics(text);
      setMetrics(parsed);
    };

    fetchMetrics();
    const interval = setInterval(fetchMetrics, 5000);

    return () => clearInterval(interval);
  }, []);

  if (loading || !metrics) {
    return <CircularProgress />;
  }

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" sx={{ mb: 3 }}>System Metrics</Typography>

      <Grid container spacing={3}>
        {/* WebSocket Connections */}
        <Grid item xs={12} md={6} lg={3}>
          <MetricCard
            title="WebSocket Connections"
            value={metrics.ws_connections_active}
            total={metrics.ws_connections_total}
            icon={<NetworkCheckIcon />}
            color="primary"
          />
        </Grid>

        {/* Chat Messages */}
        <Grid item xs={12} md={6} lg={3}>
          <MetricCard
            title="Chat Messages"
            value={metrics.chat_messages_success}
            total={metrics.chat_messages_total}
            icon={<MessageIcon />}
            color="success"
          />
        </Grid>

        {/* AI Tokens */}
        <Grid item xs={12} md={6} lg={3}>
          <MetricCard
            title="AI Tokens Streamed"
            value={metrics.ai_stream_tokens_total}
            icon={<SpeedIcon />}
            color="info"
          />
        </Grid>

        {/* MongoDB Operations */}
        <Grid item xs={12} md={6} lg={3}>
          <MetricCard
            title="MongoDB Operations"
            value={metrics.mongo_operations_total}
            icon={<StorageIcon />}
            color="warning"
          />
        </Grid>

        {/* Response Times */}
        <Grid item xs={12}>
          <Card>
            <CardContent>
              <Typography variant="h6">Response Time Distribution</Typography>
              <Box sx={{ mt: 2 }}>
                <Typography variant="body2">P50: {metrics.http_request_duration_p50}ms</Typography>
                <LinearProgress
                  variant="determinate"
                  value={calculatePercentage(metrics.http_request_duration_p50, 1000)}
                  sx={{ mt: 1, mb: 2 }}
                />

                <Typography variant="body2">P95: {metrics.http_request_duration_p95}ms</Typography>
                <LinearProgress
                  variant="determinate"
                  value={calculatePercentage(metrics.http_request_duration_p95, 1000)}
                  sx={{ mt: 1, mb: 2 }}
                />

                <Typography variant="body2">P99: {metrics.http_request_duration_p99}ms</Typography>
                <LinearProgress
                  variant="determinate"
                  value={calculatePercentage(metrics.http_request_duration_p99, 1000)}
                  sx={{ mt: 1 }}
                />
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
```

### 9.5 Metrics Parser Utility

**Location:** `ui/src/utils/metricsParser.ts:1-185`

```typescript
// Parse Prometheus text format
export function parsePrometheusMetrics(text: string): ParsedMetrics {
  const lines = text.split('\n');
  const metrics: ParsedMetrics = {};

  for (const line of lines) {
    // Skip comments and empty lines
    if (line.startsWith('#') || line.trim() === '') {
      continue;
    }

    // Parse metric line: metric_name{labels} value timestamp
    const match = line.match(/^(\w+)(?:\{([^}]+)\})?\s+([\d.e+-]+)(?:\s+(\d+))?$/);
    if (!match) {
      continue;
    }

    const [, name, labels, value] = match;

    // Store metric
    metrics[name] = parseFloat(value);

    // Parse labels if present
    if (labels) {
      const labelPairs = labels.split(',');
      for (const pair of labelPairs) {
        const [key, val] = pair.split('=');
        const cleanVal = val.replace(/"/g, '');
        metrics[`${name}_${key}_${cleanVal}`] = parseFloat(value);
      }
    }
  }

  return metrics;
}

// Calculate rate of change
export function calculateRate(
  current: number,
  previous: number,
  timeDeltaMs: number
): number {
  const deltaValue = current - previous;
  const deltaSeconds = timeDeltaMs / 1000;
  return deltaValue / deltaSeconds;
}

// Format metric value for display
export function formatMetricValue(value: number, type: string): string {
  switch (type) {
    case 'duration':
      if (value < 1) return `${(value * 1000).toFixed(0)}ms`;
      return `${value.toFixed(2)}s`;

    case 'count':
      if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`;
      if (value >= 1000) return `${(value / 1000).toFixed(1)}k`;
      return value.toString();

    case 'percentage':
      return `${(value * 100).toFixed(1)}%`;

    default:
      return value.toString();
  }
}
```

---

## 10. Work Delegation Enforcement

### 10.1 Mandatory 6-Step Workflow

**System Prompt Enforcement:**

```
🚨 MANDATORY 6-STEP WORKFLOW (NO DEVIATIONS ALLOWED)

Step 1: Check Existing Tasks (1 tool call - ALWAYS FIRST):
   coordinator_list_human_tasks({ limit: 10, status: "pending" })

Step 2: Create Human Task (1 tool call - REQUIRED):
   coordinator_create_human_task({ prompt: "<<user's exact request verbatim>>" })

Step 3: Analyze & Present Implementation Options (REQUIRED - NO TOOL CALLS):
   Present 2-3 different approaches, wait for user response

Step 4: ONE Code Search (1 tool call - DO NOT SKIP, DO NOT REPEAT):
   code_index_search({ query: "<<what user wants based on chosen approach>>", limit: 15 })

Step 5: Create Agent Task (1 tool call - REQUIRED IMMEDIATELY AFTER SEARCH):
   coordinator_create_agent_task({
     humanTaskId: "<<from step 2>>",
     agentName: "ui-dev|go-dev|sre|...",
     role: "Brief mission",
     contextSummary: "WHAT/WHERE/HOW/WHY",
     filesModified: ["<<COPY from FILE_PATHS_TO_USE array>>"],
     todos: [...]
   })

Step 6: Execute Subagent (1 tool call - FINAL STEP):
   execute_subagent({
     agentTaskId: "<<taskId from create_agent_task result>>"
   })
```

### 10.2 Circuit Breaker Enforcement

```go
// Location: coordinator_tools.go:3500-3600
type WorkflowEnforcer struct {
    mu             sync.Mutex
    sessionState   map[string]*WorkflowState
}

type WorkflowState struct {
    CurrentStep    int
    ToolCallCount  int
    HumanTaskID    string
    AgentTaskID    string
    CodeSearchDone bool
    Timestamp      time.Time
}

func (e *WorkflowEnforcer) ValidateToolCall(sessionID string, toolName string) error {
    e.mu.Lock()
    defer e.mu.Unlock()

    state := e.getOrCreateState(sessionID)
    state.ToolCallCount++

    // Enforce step order
    switch state.CurrentStep {
    case 0: // Must start with list_human_tasks
        if toolName != "coordinator_list_human_tasks" {
            return fmt.Errorf(
                "🚨 WORKFLOW VIOLATION: Step 1 must be coordinator_list_human_tasks. "+
                "You tried to call: %s",
                toolName,
            )
        }
        state.CurrentStep = 1

    case 1: // Must create human task
        if toolName != "coordinator_create_human_task" {
            return fmt.Errorf(
                "🚨 WORKFLOW VIOLATION: Step 2 must be coordinator_create_human_task. "+
                "You tried to call: %s",
                toolName,
            )
        }
        state.CurrentStep = 2

    case 2: // Must do code search (only once)
        if toolName == "code_index_search" {
            if state.CodeSearchDone {
                return fmt.Errorf(
                    "🚨 WORKFLOW VIOLATION: code_index_search can only be called ONCE. "+
                    "You already searched. Use those results!",
                )
            }
            state.CodeSearchDone = true
            state.CurrentStep = 3
        } else if toolName != "coordinator_create_human_task" {
            return fmt.Errorf(
                "🚨 WORKFLOW VIOLATION: Step 3 must be code_index_search. "+
                "You tried to call: %s",
                toolName,
            )
        }

    case 3: // Must create agent task
        if toolName != "coordinator_create_agent_task" {
            return fmt.Errorf(
                "🚨 WORKFLOW VIOLATION: Step 4 must be coordinator_create_agent_task. "+
                "You tried to call: %s",
                toolName,
            )
        }
        state.CurrentStep = 4

    case 4: // Must execute subagent
        if toolName != "execute_subagent" {
            return fmt.Errorf(
                "🚨 WORKFLOW VIOLATION: Step 5 must be execute_subagent. "+
                "You tried to call: %s",
                toolName,
            )
        }
        state.CurrentStep = 5

    case 5: // Workflow complete
        // Allow monitoring tools
        if toolName != "coordinator_list_agent_tasks" &&
           toolName != "coordinator_get_agent_task" {
            return fmt.Errorf(
                "🚨 WORKFLOW VIOLATION: After execute_subagent, only monitoring tools allowed. "+
                "You tried to call: %s",
                toolName,
            )
        }
    }

    // Check tool call limit
    if state.ToolCallCount > 6 {
        return fmt.Errorf(
            "🚨 WORKFLOW VIOLATION: Exceeded 6 tool calls (%d). "+
            "You should have: list → create_human → search → create_agent → execute. "+
            "You are stuck in a loop!",
            state.ToolCallCount,
        )
    }

    return nil
}
```

### 10.3 Prescriptive Workflow Filter (Phase 3)

**Location:** `hyper/internal/ai-service/langchain_service.go:600-700`

```go
// Prescriptive workflow filter - restricts tools per step
func (s *LangchainService) applyPrescriptiveWorkflowFilter(
    tools []MCPTool,
    iteration int,
    toolCallHistory []ToolCall,
) []MCPTool {
    // Step 0: Only allow list_human_tasks
    if iteration == 1 && len(toolCallHistory) == 0 {
        return filterTools(tools, []string{"coordinator_list_human_tasks"})
    }

    lastToolCall := getLastToolCall(toolCallHistory)

    // Step 1: After listing, only allow create_human_task
    if lastToolCall != nil && lastToolCall.Name == "coordinator_list_human_tasks" {
        if !hasToolCall(toolCallHistory, "coordinator_create_human_task") {
            return filterTools(tools, []string{"coordinator_create_human_task"})
        }
    }

    // Step 2: After creating human task, only allow code_index_search or present options (no tools)
    if hasToolCall(toolCallHistory, "coordinator_create_human_task") &&
       !hasToolCall(toolCallHistory, "code_index_search") {
        // Allow text response without tools for presenting options
        if lastToolCall != nil && lastToolCall.Name == "coordinator_create_human_task" {
            return []MCPTool{} // Force text response (present options)
        }
        return filterTools(tools, []string{"code_index_search"})
    }

    // Step 3: After code search, only allow create_agent_task
    if hasToolCall(toolCallHistory, "code_index_search") &&
       !hasToolCall(toolCallHistory, "coordinator_create_agent_task") {
        return filterTools(tools, []string{"coordinator_create_agent_task"})
    }

    // Step 4: After creating agent task, only allow execute_subagent
    if hasToolCall(toolCallHistory, "coordinator_create_agent_task") &&
       !hasToolCall(toolCallHistory, "execute_subagent") {
        return filterTools(tools, []string{"execute_subagent"})
    }

    // Step 5+: Allow monitoring tools only
    if hasToolCall(toolCallHistory, "execute_subagent") {
        return filterTools(tools, []string{
            "coordinator_list_agent_tasks",
            "coordinator_get_agent_task",
            "coordinator_update_task_status",
        })
    }

    // Default: allow all tools
    return tools
}

// Filter tools to allowed list
func filterTools(tools []MCPTool, allowedNames []string) []MCPTool {
    allowed := make(map[string]bool)
    for _, name := range allowedNames {
        allowed[name] = true
    }

    filtered := []MCPTool{}
    for _, tool := range tools {
        if allowed[tool.Name] {
            filtered = append(filtered, tool)
        }
    }

    return filtered
}
```

---

## 11. System Prompt Architecture

### 11.1 Coordinator System Prompt

**Location:** `hyper/internal/ai-service/prompts/coordinator_prompt.txt`

**Structure:**
```
1. Role Definition (100 words)
   - "You are a COORDINATOR, not an IMPLEMENTER"
   - Clear separation of concerns

2. Core Responsibilities (200 words)
   - List tasks
   - Create tasks
   - Delegate to agents
   - Monitor progress

3. Forbidden Actions (150 words)
   - Never implement yourself
   - Never read/write files for implementation
   - Never explore codebase repeatedly

4. Mandatory Workflow (500 words)
   - 6-step process
   - Tool call limits
   - Circuit breaker rules

5. Validation Rules (300 words)
   - TODO validation
   - File path validation
   - Similarity checks

6. Error Recovery (200 words)
   - How to handle tool errors
   - When to ask user for help
   - Retry strategies

7. Session Context (100 words)
   - Current session ID
   - Project root
   - File system context

8. Dynamic Workflow Progress (injected at runtime)
   - Current iteration
   - Tool calls made
   - Next recommended action
```

**Dynamic Injection:**
```go
// Location: langchain_service.go:1200-1250
func (s *LangchainService) buildSystemPrompt(
    basePrompt string,
    sessionID string,
    iteration int,
    toolCallCount int,
) string {
    // Build workflow progress section
    workflowProgress := fmt.Sprintf(`
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
WORKFLOW PROGRESS (Iteration %d)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Current Iteration: %d / 30
Total Tool Calls Made: %d / 50

⚠️  IMPORTANT: If you see "BLOCKED" or "NEXT:" in recent tool results,
    those messages contain CRITICAL guidance about what to do next.
    You MUST follow the "NEXT:" instructions - they tell you exactly which tool to call.

⚠️  AVOID LOOPS: Do NOT retry the same tool that was just BLOCKED.
    Follow the "NEXT:" guidance instead.

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
`, iteration, iteration, toolCallCount)

    // Inject session context
    sessionContext := fmt.Sprintf(`
SESSION CONTEXT:
- **CURRENT CHAT SESSION ID**: %s
- **IMPORTANT**: When using execute_subagent tool, ALWAYS use parentChatId: "%s"
- DO NOT ask the user for the session ID - it is provided above
- This session ID links subagent work back to this conversation
`, sessionID, sessionID)

    // Combine sections
    return basePrompt + "\n\n" + sessionContext + "\n\n" + workflowProgress
}
```

### 11.2 Agent System Prompt

**Location:** Generated dynamically in `coordinator_tools.go:2750-2850`

```go
func (t *ExecuteSubagentTool) buildAgentSystemPrompt(agentTask *models.AgentTask) string {
    // Build TODO list
    todoList := ""
    for i, todo := range agentTask.Todos {
        status := "PENDING"
        if todo.Status == "in_progress" {
            status = "IN PROGRESS"
        } else if todo.Status == "completed" {
            status = "COMPLETED"
        }

        todoList += fmt.Sprintf("\n%d. [%s] ID: %s - %s",
            i+1, status, todo.ID.Hex(), todo.Description)

        if todo.FilePath != "" {
            todoList += fmt.Sprintf("\n   File: %s", todo.FilePath)
        }
        if todo.FunctionName != "" {
            todoList += fmt.Sprintf("\n   Function: %s", todo.FunctionName)
        }
        if todo.ContextHint != "" {
            todoList += fmt.Sprintf("\n   Hint: %s", todo.ContextHint)
        }
    }

    // Build file list
    fileList := ""
    for _, file := range agentTask.FilesModified {
        fileList += fmt.Sprintf("\n- %s", file)
    }

    // Combine into prompt
    prompt := fmt.Sprintf(`You are %s. You have been assigned a task to complete.

ROLE: %s

TASK CONTEXT:
%s

YOUR TODOs:
%s

FILES TO MODIFY:
%s

═══════════════════════════════════════════════════════════════════
Task ID: %s
BEGIN EXECUTION NOW.
═══════════════════════════════════════════════════════════════════

IMPORTANT GUIDELINES:
1. Start implementing within 2 minutes - do NOT spend time exploring or planning
2. Update TODO status as you complete each one using coordinator_update_todo_status
3. Include EXACT line numbers and code changes in status notes
4. Upsert knowledge when you discover important patterns or decisions
5. DO NOT create more subchats - you cannot call execute_subagent
6. DO NOT try to list or create tasks - focus on implementation only
7. If stuck, explain the blocker in a status update

WORKFLOW:
1. Read the first file you need to modify
2. Make the required changes
3. Update the TODO status with notes about what you changed
4. Move to the next TODO
5. When all TODOs complete, upsert knowledge summarizing your work

BEGIN NOW!
`,
        agentTask.AgentName,
        agentTask.Role,
        agentTask.ContextSummary,
        todoList,
        fileList,
        agentTask.ID.Hex(),
    )

    return prompt
}
```

---

## 12. Debug Mode vs Default Mode

### 12.1 Conversation Mode Context

**Location:** `ui2/src/contexts/ConversationModeContext.tsx:1-50`

```typescript
// Conversation mode context provider
type ConversationMode = 'default' | 'debug';

interface ConversationModeContextType {
  mode: ConversationMode;
  setMode: (mode: ConversationMode) => void;
  toggleMode: () => void;
}

const ConversationModeContext = createContext<ConversationModeContextType | undefined>(undefined);

export const ConversationModeProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  // Load mode from localStorage
  const [mode, setModeState] = useState<ConversationMode>(() => {
    const stored = localStorage.getItem('hyperion-conversation-mode');
    return (stored === 'debug' ? 'debug' : 'default') as ConversationMode;
  });

  // Persist mode to localStorage
  const setMode = (newMode: ConversationMode) => {
    setModeState(newMode);
    localStorage.setItem('hyperion-conversation-mode', newMode);
  };

  // Toggle between modes
  const toggleMode = () => {
    setMode(mode === 'default' ? 'debug' : 'default');
  };

  return (
    <ConversationModeContext.Provider value={{ mode, setMode, toggleMode }}>
      {children}
    </ConversationModeContext.Provider>
  );
};

// Hook to use conversation mode
export const useConversationMode = () => {
  const context = useContext(ConversationModeContext);
  if (!context) {
    throw new Error('useConversationMode must be used within ConversationModeProvider');
  }
  return context;
};
```

### 12.2 Mode Toggle Component

**Location:** `ui2/src/components/molecules/ConversationModeToggle.tsx:1-60`

```typescript
// Toggle button component
interface ConversationModeToggleProps {
  showLabel?: boolean;
}

export const ConversationModeToggle: React.FC<ConversationModeToggleProps> = ({ showLabel = false }) => {
  const { mode, toggleMode } = useConversationMode();

  return (
    <div className="flex items-center gap-2">
      {showLabel && (
        <span className="text-sm text-gray-600 dark:text-gray-400">
          {mode === 'default' ? 'Default' : 'Debug'}
        </span>
      )}

      <button
        onClick={toggleMode}
        className={cn(
          "relative inline-flex h-6 w-11 items-center rounded-full transition-colors",
          mode === 'debug'
            ? 'bg-blue-600'
            : 'bg-gray-300 dark:bg-gray-600'
        )}
        aria-label="Toggle conversation mode"
      >
        <span
          className={cn(
            "inline-block h-4 w-4 transform rounded-full bg-white transition-transform",
            mode === 'debug' ? 'translate-x-6' : 'translate-x-1'
          )}
        />
      </button>

      {showLabel && (
        <span className="text-xs text-gray-500 dark:text-gray-500">
          {mode === 'debug' ? 'Tool calls visible' : 'Tool calls hidden'}
        </span>
      )}
    </div>
  );
};
```

### 12.3 Conditional Rendering Based on Mode

**Location:** `ui2/src/components/organisms/ChatMessage.tsx:45-150`

```typescript
export const ChatMessage: React.FC<ChatMessageProps> = ({ message, ...props }) => {
  // Get current mode
  const { mode } = useConversationMode();
  const showToolDetails = mode === 'debug';

  // Tool_call messages - only show in debug mode
  if (message.role === 'tool_call' && message.toolCall) {
    if (!showToolDetails) {
      return null; // Hide in default mode
    }

    // Render tool call in debug mode
    return (
      <div className="border border-blue-200 dark:border-blue-700 rounded-lg bg-blue-50 dark:bg-blue-900/20">
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-100 dark:bg-blue-900/40">
          <Wrench className="w-4 h-4 text-blue-600" />
          <span className="font-mono text-sm">{message.toolCall.name}</span>
        </div>
        <div className="p-3">
          <pre className="bg-gray-900 text-gray-100 p-2 rounded text-xs">
            {JSON.stringify(message.toolCall.args, null, 2)}
          </pre>
        </div>
      </div>
    );
  }

  // Tool_result messages - only show in debug mode
  if (message.role === 'tool_result' && message.toolResult) {
    if (!showToolDetails) {
      return null; // Hide in default mode
    }

    const hasError = message.toolResult.error;
    return (
      <div className={cn(
        "border rounded-lg",
        hasError ? "border-red-200 bg-red-50" : "border-green-200 bg-green-50"
      )}>
        <div className="flex items-center gap-2 px-3 py-2">
          {hasError ? <XCircle className="w-4 h-4 text-red-600" /> : <CheckCircle className="w-4 h-4 text-green-600" />}
          <span className="font-mono text-sm">{message.toolResult.name}</span>
          <span className="text-xs ml-auto">{message.toolResult.durationMs}ms</span>
        </div>
        <div className="p-3">
          <pre className={hasError ? "bg-red-900/20 text-red-300" : "bg-gray-900 text-gray-100"}>
            {hasError ? message.toolResult.error : JSON.stringify(message.toolResult.output, null, 2)}
          </pre>
        </div>
      </div>
    );
  }

  // Assistant messages - show tool calls accordion only in debug mode
  if (message.role === 'assistant' && message.toolCalls) {
    return (
      <div className="bg-white dark:bg-gray-800 rounded-lg p-4">
        {/* Message content */}
        <div className="prose">{message.content}</div>

        {/* Tool calls accordion - only in debug mode */}
        {showToolDetails && message.toolCalls.length > 0 && (
          <Accordion.Root type="multiple" className="mt-3">
            {message.toolCalls.map((toolCall) => (
              <Accordion.Item key={toolCall.id} value={toolCall.id} className="border rounded-lg mb-2">
                <Accordion.Header>
                  <Accordion.Trigger className="flex items-center justify-between w-full px-3 py-2">
                    <div className="flex items-center gap-2">
                      <Wrench className="w-4 h-4" />
                      <span className="font-mono">{toolCall.tool}</span>
                      {/* Status indicators */}
                    </div>
                    <ChevronDown className="w-4 h-4" />
                  </Accordion.Trigger>
                </Accordion.Header>
                <Accordion.Content className="px-3 py-2">
                  {/* Tool arguments and results */}
                </Accordion.Content>
              </Accordion.Item>
            ))}
          </Accordion.Root>
        )}
      </div>
    );
  }

  // Regular messages - always visible
  return (
    <div className="message">
      {message.content}
    </div>
  );
};
```

---

## 13. Tool Call Display in Debug Mode

### 13.1 Tool Call Message Structure

**Database Schema:**

```go
// models/message.go
type Message struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    SessionID primitive.ObjectID `bson:"sessionId"`
    Role      string             `bson:"role"` // "user", "assistant", "system", "tool_call", "tool_result"
    Content   string             `bson:"content"`
    Timestamp time.Time          `bson:"timestamp"`

    // For tool_call messages
    ToolCall *ToolCall `bson:"toolCall,omitempty"`

    // For tool_result messages
    ToolResult *ToolResult `bson:"toolResult,omitempty"`
}

type ToolCall struct {
    ID   string                 `bson:"id"`
    Name string                 `bson:"name"`
    Args map[string]interface{} `bson:"args"`
}

type ToolResult struct {
    ID         string      `bson:"id"`
    Name       string      `bson:"name"`
    Output     interface{} `bson:"output"`
    Error      *string     `bson:"error,omitempty"`
    DurationMs int64       `bson:"durationMs"`
}
```

### 13.2 WebSocket Tool Call Events

```go
// Send tool_call event
func (h *ChatWebSocketHandler) emitToolCall(conn *websocket.Conn, toolName string, args map[string]interface{}, id string) {
    msg := models.StreamMessage{
        Type: "tool_call",
        ToolCall: &models.ToolCallEvent{
            Tool: toolName,
            Args: args,
            ID:   id,
        },
    }

    h.safeWriteJSON(conn, msg)

    // Also save to database as separate message
    toolCallMsg := &models.Message{
        SessionID: sessionID,
        Role:      "tool_call",
        Content:   "",
        Timestamp: time.Now(),
        ToolCall: &models.ToolCall{
            ID:   id,
            Name: toolName,
            Args: args,
        },
    }

    h.messageService.CreateMessage(ctx, toolCallMsg)
}

// Send tool_result event
func (h *ChatWebSocketHandler) emitToolResult(conn *websocket.Conn, id string, toolName string, result interface{}, err error, duration time.Duration) {
    var errorStr *string
    if err != nil {
        str := err.Error()
        errorStr = &str
    }

    msg := models.StreamMessage{
        Type: "tool_result",
        ToolResult: &models.ToolResultEvent{
            ID:         id,
            Tool:       toolName,
            Result:     result,
            Error:      errorStr,
            DurationMs: duration.Milliseconds(),
        },
    }

    h.safeWriteJSON(conn, msg)

    // Also save to database
    toolResultMsg := &models.Message{
        SessionID: sessionID,
        Role:      "tool_result",
        Content:   "",
        Timestamp: time.Now(),
        ToolResult: &models.ToolResult{
            ID:         id,
            Name:       toolName,
            Output:     result,
            Error:      errorStr,
            DurationMs: duration.Milliseconds(),
        },
    }

    h.messageService.CreateMessage(ctx, toolResultMsg)
}
```

### 13.3 Frontend Tool Call Handler

**Location:** `ui2/src/pages/CodeChatPage.tsx:210-261`

```typescript
// WebSocket callbacks
const callbacks: StreamCallbacks = {
  onMessage: (content, done) => {
    if (done) {
      // Save final assistant message
      setMessages(prev => [...prev, finalMessage]);
    } else {
      // Accumulate streaming content
      setStreamingContent(prev => prev + content);
    }
  },

  onToolCall: (tool, args, id) => {
    console.log('[CodeChatPage] Tool call received:', tool, id);

    // Save content before tool call as separate message
    if (streamingContentRef.current.trim()) {
      const messageBeforeToolCall = {
        id: `msg-${Date.now()}`,
        sessionId: activeSessionId,
        role: 'assistant',
        content: streamingContentRef.current,
        timestamp: new Date().toISOString(),
      };
      setMessages(prev => [...prev, messageBeforeToolCall]);
      streamingContentRef.current = '';
      setStreamingContent('');
    }

    // Create dedicated tool_call message
    const toolCallMessage = {
      id: `tool-call-${id}`,
      sessionId: activeSessionId,
      role: 'tool_call',
      content: '',
      timestamp: new Date().toISOString(),
      toolCall: {
        id,
        name: tool,
        args,
      },
    };
    setMessages(prev => [...prev, toolCallMessage]);

    // Track for streaming display
    setPendingToolCalls(prev => new Set(prev).add(id));
    setStreamingToolCalls(prev => [...prev, { id, tool, args, timestamp: new Date() }]);
  },

  onToolResult: (id, tool, result, error, durationMs) => {
    console.log('[CodeChatPage] Tool result received:', tool, id);

    // Create dedicated tool_result message
    const toolResultMessage = {
      id: `tool-result-${id}`,
      sessionId: activeSessionId,
      role: 'tool_result',
      content: '',
      timestamp: new Date().toISOString(),
      toolResult: {
        id,
        name: tool,
        output: result,
        error,
        durationMs,
      },
    };
    setMessages(prev => [...prev, toolResultMessage]);

    // Update streaming state
    setPendingToolCalls(prev => {
      const updated = new Set(prev);
      updated.delete(id);
      return updated;
    });
    setStreamingToolResults(prev => new Map(prev).set(id, {
      id, tool, result, error, durationMs,
    }));
  },
};
```

### 13.4 Accordion Display Component

```typescript
// Tool calls accordion (only visible in debug mode)
{showToolDetails && hasToolCalls && (
  <Accordion.Root type="multiple" className="mt-3">
    {allToolCalls.map((toolCall, index) => {
      const toolResult = allToolResults.get(toolCall.id);
      const isPending = pendingToolCalls.has(toolCall.id);
      const hasError = toolResult?.error;
      const isComplete = toolResult && !hasError;

      return (
        <Accordion.Item
          key={toolCall.id}
          value={toolCall.id}
          className="border border-gray-200 dark:border-gray-700 rounded-lg mb-2 overflow-hidden"
        >
          {/* Accordion Header */}
          <Accordion.Header>
            <Accordion.Trigger className="flex items-center justify-between w-full px-3 py-2 hover:bg-gray-50">
              <div className="flex items-center gap-2">
                <Wrench className="w-4 h-4" />
                <span className="font-mono">{toolCall.tool}</span>

                {/* Status Indicator */}
                {isPending && <Clock className="w-4 h-4 text-yellow-500 animate-pulse" />}
                {isComplete && <CheckCircle className="w-4 h-4 text-green-500" />}
                {hasError && <XCircle className="w-4 h-4 text-red-500" />}

                {/* Duration */}
                {toolResult?.durationMs && (
                  <span className="text-xs opacity-70">{toolResult.durationMs}ms</span>
                )}
              </div>

              <ChevronDown className="w-4 h-4 transition-transform group-data-[state=open]:rotate-180" />
            </Accordion.Trigger>
          </Accordion.Header>

          {/* Accordion Content */}
          <Accordion.Content className="bg-gray-50 dark:bg-gray-900/50 px-3 py-2">
            {/* Tool Arguments */}
            <div className="mb-2">
              <div className="font-semibold text-xs mb-1">Arguments:</div>
              <pre className="bg-gray-900 text-gray-100 p-2 rounded text-xs overflow-x-auto">
                {JSON.stringify(toolCall.args, null, 2)}
              </pre>
            </div>

            {/* Tool Result */}
            {toolResult && (
              <div>
                <div className="font-semibold text-xs mb-1">
                  {hasError ? 'Error:' : 'Result:'}
                </div>
                <pre className={cn(
                  "p-2 rounded text-xs overflow-x-auto",
                  hasError ? "bg-red-900/20 text-red-300" : "bg-gray-900 text-gray-100"
                )}>
                  {hasError ? toolResult.error : JSON.stringify(toolResult.result, null, 2)}
                </pre>
              </div>
            )}
          </Accordion.Content>
        </Accordion.Item>
      );
    })}
  </Accordion.Root>
)}
```

---

## Summary

This comprehensive technical guide covers all 13 aspects of the Hyperion Coordinator system:

1. ✅ **Chat/Subchat Delegation** - Two-tier architecture with parent-child linking
2. ✅ **Orchestrator/Implementer Separation** - Tool segregation, security layers
3. ✅ **Subchat Interruptions** - Two-select pattern, intelligent categorization
4. ✅ **Progress Tracker** - Singleton notifier, pulsating icons, auto-refresh
5. ✅ **File Indexing** - Async queue-based, tree-sitter symbols, semantic search
6. ✅ **Orchestrator Validations** - TODO keywords, file paths, circuit breaker
7. ✅ **Executor Guards** - Two-layer security, resource limits, transactions
8. ✅ **Detailed Interruption** - 5 categories, dynamic prompts, AI-powered
9. ✅ **Prometheus Dashboard** - 30+ metrics, auto-refresh, parser utility
10. ✅ **Work Delegation Enforcement** - 6-step workflow, prescriptive filter
11. ✅ **System Prompts** - Coordinator vs Agent, dynamic injection
12. ✅ **Debug/Default Modes** - Context provider, localStorage, toggle component
13. ✅ **Tool Call Display** - Separate messages, accordions, status indicators

**Total Lines of Code Referenced:** ~8,500 lines across 30+ files

**Key Technologies:**
- Backend: Go 1.25, MongoDB, WebSocket, Prometheus, Tree-sitter
- Frontend: React 18, TypeScript, Radix UI, Tailwind CSS
- Architecture: Microservices, Event-driven, Async processing

**Implementation Status:** ✅ All features fully implemented and verified

---

**Document Version:** 1.0
**Last Updated:** November 5, 2025
**Branch:** megha/knowledge-browser
**Maintainer:** Hyperion Development Team
