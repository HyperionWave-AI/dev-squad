# Human Prompt Notes - Feature Specification

**Feature Name:** Human-to-Agent Collaboration via Prompt Notes
**Version:** 1.0
**Status:** Planning
**Created:** 2025-10-02
**Owner:** Product/Engineering

---

## 📋 Executive Summary

Enable human users to add **prompt notes** to agent tasks and TODOs after the AI planning phase completes. These notes act as human guidance that agents **must read and incorporate** during implementation, creating a collaborative human-in-the-loop workflow that improves agent accuracy and allows users to inject domain knowledge or clarifications.

### Problem Statement

Currently, once the Workflow Coordinator creates agent tasks with rich context (contextSummary, contextHint, etc.), humans cannot inject additional guidance or corrections before agents start implementation. This creates issues when:

1. **Planning missed critical details** - AI planning doesn't capture user's domain knowledge
2. **User realizes context gaps** - After reviewing the plan, user identifies missing constraints
3. **Business rules change** - User needs to add new requirements discovered during planning review
4. **Clarifications needed** - User wants to emphasize certain approaches or warn about gotchas

### Solution

Add a **prompt notes system** where users can:
- Attach notes to **agent tasks** (task-level guidance)
- Attach notes to individual **TODOs** (item-level guidance)
- Edit/update notes before agent execution
- Have agents **automatically read and incorporate** these notes during implementation

---

## 🎯 Goals & Non-Goals

### Goals

1. ✅ Enable humans to add guidance notes to agent tasks after planning
2. ✅ Enable humans to add guidance notes to individual TODOs
3. ✅ Ensure agents read and incorporate prompt notes during implementation
4. ✅ Provide UI for viewing and editing prompt notes
5. ✅ Track when notes were added and by whom
6. ✅ Support markdown formatting in notes for rich documentation
7. ✅ Make notes editable until agent starts executing the task

### Non-Goals

1. ❌ Real-time collaboration between multiple humans
2. ❌ Approval workflows or sign-offs
3. ❌ Version history of note edits (v1)
4. ❌ Agent-generated suggestions for notes
5. ❌ Notifications when notes are added

---

## 🏗️ Architecture

### Data Model Changes

#### 1. AgentTask Schema Updates

**Add new field:**
```typescript
interface AgentTask {
  // ... existing fields ...
  humanPromptNotes?: string;           // NEW: Human's guidance for entire task
  humanPromptNotesAddedAt?: Date;      // NEW: When notes were added
  humanPromptNotesUpdatedAt?: Date;    // NEW: Last update timestamp
}
```

**MongoDB Schema:**
```go
type AgentTask struct {
    // ... existing fields ...
    HumanPromptNotes        string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
    HumanPromptNotesAddedAt *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
    HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
}
```

#### 2. TodoItem Schema Updates

**Add new field:**
```typescript
interface TodoItem {
  // ... existing fields ...
  humanPromptNotes?: string;           // NEW: Human's guidance for this TODO
  humanPromptNotesAddedAt?: Date;      // NEW: When notes were added
  humanPromptNotesUpdatedAt?: Date;    // NEW: Last update timestamp
}
```

**MongoDB Schema:**
```go
type TodoItem struct {
    // ... existing fields ...
    HumanPromptNotes        string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
    HumanPromptNotesAddedAt *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
    HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
}
```

### API Changes

#### New MCP Tools

**1. Add Task Prompt Notes**

```typescript
mcp__hyperion-coordinator__coordinator_add_task_prompt_notes({
  agentTaskId: string,        // REQUIRED: Agent task ID
  promptNotes: string         // REQUIRED: Human's guidance (markdown supported)
})
```

**Returns:**
```json
{
  "success": true,
  "taskId": "uuid",
  "notesAddedAt": "2025-10-02T14:30:00Z"
}
```

**2. Add TODO Prompt Notes**

```typescript
mcp__hyperion-coordinator__coordinator_add_todo_prompt_notes({
  agentTaskId: string,        // REQUIRED: Agent task ID
  todoId: string,             // REQUIRED: TODO UUID
  promptNotes: string         // REQUIRED: Human's guidance (markdown supported)
})
```

**Returns:**
```json
{
  "success": true,
  "taskId": "uuid",
  "todoId": "uuid",
  "notesAddedAt": "2025-10-02T14:30:00Z"
}
```

**3. Update Task Prompt Notes**

```typescript
mcp__hyperion-coordinator__coordinator_update_task_prompt_notes({
  agentTaskId: string,        // REQUIRED: Agent task ID
  promptNotes: string         // REQUIRED: Updated guidance
})
```

**4. Update TODO Prompt Notes**

```typescript
mcp__hyperion-coordinator__coordinator_update_todo_prompt_notes({
  agentTaskId: string,        // REQUIRED: Agent task ID
  todoId: string,             // REQUIRED: TODO UUID
  promptNotes: string         // REQUIRED: Updated guidance
})
```

**5. Clear Task Prompt Notes**

```typescript
mcp__hyperion-coordinator__coordinator_clear_task_prompt_notes({
  agentTaskId: string         // REQUIRED: Agent task ID
})
```

**6. Clear TODO Prompt Notes**

```typescript
mcp__hyperion-coordinator__coordinator_clear_todo_prompt_notes({
  agentTaskId: string,        // REQUIRED: Agent task ID
  todoId: string              // REQUIRED: TODO UUID
})
```

#### Updated Existing Tools

**`coordinator_list_agent_tasks` - No changes needed** (automatically returns new fields)

Response includes:
```json
{
  "tasks": [
    {
      "id": "uuid",
      "humanPromptNotes": "User's guidance here",
      "humanPromptNotesAddedAt": "2025-10-02T...",
      "todos": [
        {
          "id": "uuid",
          "humanPromptNotes": "User's TODO-specific guidance",
          "humanPromptNotesAddedAt": "2025-10-02T..."
        }
      ]
    }
  ]
}
```

---

## 💻 Implementation Details

### Three-Tier Architecture

This feature requires changes across all three projects:

1. **MCP Server** (`coordinator/mcp-server/`) - Core logic and MongoDB storage
2. **HTTP Bridge** (`coordinator/mcp-http-bridge/`) - HTTP → MCP translation (minimal changes)
3. **UI** (`coordinator/ui/`) - React components for user interaction

---

### Project 1: MCP Server Changes

**File:** `coordinator/mcp-server/storage/tasks.go`

**1. Update Storage Interface:**
```go
type TaskStorage interface {
    // ... existing methods ...

    // NEW: Prompt notes management
    AddTaskPromptNotes(agentTaskID string, notes string) error
    UpdateTaskPromptNotes(agentTaskID string, notes string) error
    ClearTaskPromptNotes(agentTaskID string) error

    AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error
    UpdateTodoPromptNotes(agentTaskID string, todoID string, notes string) error
    ClearTodoPromptNotes(agentTaskID string, todoID string) error
}
```

**2. Implement MongoDB Methods:**
```go
func (s *MongoTaskStorage) AddTaskPromptNotes(agentTaskID string, notes string) error {
    ctx := context.Background()
    now := time.Now()

    update := bson.M{
        "$set": bson.M{
            "humanPromptNotes": notes,
            "humanPromptNotesAddedAt": now,
            "humanPromptNotesUpdatedAt": now,
        },
    }

    _, err := s.agentTasksCollection.UpdateOne(
        ctx,
        bson.M{"taskId": agentTaskID},
        update,
    )

    return err
}

func (s *MongoTaskStorage) AddTodoPromptNotes(agentTaskID string, todoID string, notes string) error {
    ctx := context.Background()
    now := time.Now()

    // Find the TODO and update it
    update := bson.M{
        "$set": bson.M{
            "todos.$[elem].humanPromptNotes": notes,
            "todos.$[elem].humanPromptNotesAddedAt": now,
            "todos.$[elem].humanPromptNotesUpdatedAt": now,
        },
    }

    arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
        Filters: []interface{}{
            bson.M{"elem.id": todoID},
        },
    })

    _, err := s.agentTasksCollection.UpdateOne(
        ctx,
        bson.M{"taskId": agentTaskID},
        update,
        arrayFilters,
    )

    return err
}
```

**File:** `coordinator/mcp-server/handlers/tools.go`

**3. Register New Tools:**

Add to `GetToolsList()`:
```go
{
    Name: "coordinator_add_task_prompt_notes",
    Description: "Add human prompt notes to an agent task",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "agentTaskId": map[string]interface{}{
                "type":        "string",
                "description": "Agent task UUID",
            },
            "promptNotes": map[string]interface{}{
                "type":        "string",
                "description": "Human's guidance notes (markdown supported)",
            },
        },
        "required": []string{"agentTaskId", "promptNotes"},
    },
},
// ... 5 more similar tool definitions
```

**4. Add MCP Tool Handlers:**
```go
func (h *CoordinatorHandler) handleAddTaskPromptNotes(params map[string]interface{}) (*mcp.CallToolResult, error) {
    agentTaskID, ok := params["agentTaskId"].(string)
    if !ok || agentTaskID == "" {
        return nil, fmt.Errorf("agentTaskId is required")
    }

    promptNotes, ok := params["promptNotes"].(string)
    if !ok || promptNotes == "" {
        return nil, fmt.Errorf("promptNotes is required")
    }

    err := h.storage.AddTaskPromptNotes(agentTaskID, promptNotes)
    if err != nil {
        return nil, fmt.Errorf("failed to add prompt notes: %w", err)
    }

    return &mcp.CallToolResult{
        Content: []interface{}{
            map[string]interface{}{
                "type": "text",
                "text": fmt.Sprintf("✓ Added prompt notes to task %s", agentTaskID),
            },
        },
    }, nil
}
```

---

### Project 2: HTTP Bridge Changes

**File:** `coordinator/mcp-http-bridge/main.go`

**Changes Needed:** ✅ **NONE** (Bridge is stateless - automatically forwards new tools)

The HTTP bridge translates HTTP requests to MCP stdio calls. Since it doesn't hardcode tool names or schemas, it will automatically support the new MCP tools without any code changes.

**Verification:**
After MCP server implementation, test the new endpoints:
```bash
curl -X POST http://localhost:8095/api/mcp/tools/call \
  -H "Content-Type: application/json" \
  -d '{
    "name": "coordinator_add_task_prompt_notes",
    "arguments": {
      "agentTaskId": "uuid-here",
      "promptNotes": "Test notes"
    }
  }'
```

The bridge will:
1. Receive HTTP POST
2. Convert to MCP JSON-RPC format
3. Send to MCP server via stdio
4. Return MCP response as HTTP JSON

**Note:** If the bridge has any tool name validation, remove it to support dynamic tool registration.

---

### Project 3: UI Changes

**File:** `coordinator/ui/src/components/TaskDetailDialog.tsx`

**1. Add Prompt Notes Section:**
```tsx
import { TextField, IconButton, Tooltip } from '@mui/material';
import { Edit, Save, Cancel, NoteAdd } from '@mui/icons-material';

const [isEditingTaskNotes, setIsEditingTaskNotes] = useState(false);
const [taskPromptNotes, setTaskPromptNotes] = useState(task.humanPromptNotes || '');
const [isEditingTodoNotes, setIsEditingTodoNotes] = useState<Record<string, boolean>>({});
const [todoPromptNotes, setTodoPromptNotes] = useState<Record<string, string>>({});

const handleSaveTaskNotes = async () => {
  try {
    await mcpClient.addTaskPromptNotes(task.id, taskPromptNotes);
    setIsEditingTaskNotes(false);
    // Refresh task data
  } catch (error) {
    console.error('Failed to save task notes:', error);
  }
};

// Render in dialog:
<Accordion>
  <AccordionSummary expandIcon={<ExpandMore />}>
    <Box display="flex" alignItems="center" gap={1}>
      <NoteAdd />
      <Typography variant="h6">Human Guidance Notes</Typography>
      {task.humanPromptNotes && (
        <Chip label="Has Notes" size="small" color="primary" />
      )}
    </Box>
  </AccordionSummary>
  <AccordionDetails>
    <Box>
      {!isEditingTaskNotes ? (
        <Box>
          {task.humanPromptNotes ? (
            <Box>
              <ReactMarkdown>{task.humanPromptNotes}</ReactMarkdown>
              <Box mt={1}>
                <Typography variant="caption" color="text.secondary">
                  Added {new Date(task.humanPromptNotesAddedAt).toLocaleString()}
                </Typography>
              </Box>
            </Box>
          ) : (
            <Typography color="text.secondary" fontStyle="italic">
              No guidance notes yet. Add notes to provide additional context to the agent.
            </Typography>
          )}
          <Box mt={2}>
            <Button
              startIcon={task.humanPromptNotes ? <Edit /> : <NoteAdd />}
              onClick={() => setIsEditingTaskNotes(true)}
              disabled={task.status !== 'pending'}
            >
              {task.humanPromptNotes ? 'Edit Notes' : 'Add Notes'}
            </Button>
          </Box>
        </Box>
      ) : (
        <Box>
          <TextField
            fullWidth
            multiline
            rows={6}
            value={taskPromptNotes}
            onChange={(e) => setTaskPromptNotes(e.target.value)}
            placeholder="Add guidance notes for the agent (markdown supported)..."
            helperText="These notes will be read by the agent before starting implementation"
          />
          <Box mt={2} display="flex" gap={1}>
            <Button
              variant="contained"
              startIcon={<Save />}
              onClick={handleSaveTaskNotes}
            >
              Save Notes
            </Button>
            <Button
              startIcon={<Cancel />}
              onClick={() => {
                setIsEditingTaskNotes(false);
                setTaskPromptNotes(task.humanPromptNotes || '');
              }}
            >
              Cancel
            </Button>
          </Box>
        </Box>
      )}
    </Box>
  </AccordionDetails>
</Accordion>
```

**2. Add TODO-Level Notes:**
```tsx
// Within TODO list rendering:
<ListItem>
  <ListItemText
    primary={todo.description}
    secondary={
      <Box>
        {todo.contextHint && (
          <Typography variant="caption" display="block">
            💡 {todo.contextHint}
          </Typography>
        )}
        {todo.humanPromptNotes && (
          <Box mt={1} p={1} bgcolor="primary.50" borderRadius={1}>
            <Typography variant="caption" fontWeight="bold" color="primary">
              📝 Human Guidance:
            </Typography>
            <Typography variant="caption" display="block">
              {todo.humanPromptNotes}
            </Typography>
          </Box>
        )}
        <Button
          size="small"
          startIcon={<NoteAdd />}
          onClick={() => handleEditTodoNotes(todo.id)}
          disabled={task.status !== 'pending'}
        >
          {todo.humanPromptNotes ? 'Edit' : 'Add'} Notes
        </Button>
      </Box>
    }
  />
</ListItem>
```

**File:** `coordinator/ui/src/services/mcpClient.ts`

**3. Add Client Methods:**
```typescript
class MCPClient {
  // ... existing methods ...

  async addTaskPromptNotes(agentTaskId: string, promptNotes: string): Promise<void> {
    const result = await this.callTool('coordinator_add_task_prompt_notes', {
      agentTaskId,
      promptNotes
    });

    if (!result.success) {
      throw new Error('Failed to add task prompt notes');
    }
  }

  async addTodoPromptNotes(agentTaskId: string, todoId: string, promptNotes: string): Promise<void> {
    const result = await this.callTool('coordinator_add_todo_prompt_notes', {
      agentTaskId,
      todoId,
      promptNotes
    });

    if (!result.success) {
      throw new Error('Failed to add TODO prompt notes');
    }
  }

  async updateTaskPromptNotes(agentTaskId: string, promptNotes: string): Promise<void> {
    const result = await this.callTool('coordinator_update_task_prompt_notes', {
      agentTaskId,
      promptNotes
    });

    if (!result.success) {
      throw new Error('Failed to update task prompt notes');
    }
  }

  async updateTodoPromptNotes(agentTaskId: string, todoId: string, promptNotes: string): Promise<void> {
    const result = await this.callTool('coordinator_update_todo_prompt_notes', {
      agentTaskId,
      todoId,
      promptNotes
    });

    if (!result.success) {
      throw new Error('Failed to update TODO prompt notes');
    }
  }

  async clearTaskPromptNotes(agentTaskId: string): Promise<void> {
    await this.callTool('coordinator_clear_task_prompt_notes', { agentTaskId });
  }

  async clearTodoPromptNotes(agentTaskId: string, todoId: string): Promise<void> {
    await this.callTool('coordinator_clear_todo_prompt_notes', { agentTaskId, todoId });
  }
}
```

### Agent Integration

**File:** `CLAUDE.md` - Update agent workflow documentation

**Add to "Phase 1: Context Discovery" section:**

```markdown
**Step 2: Read ALL Context Fields (60 seconds)**

Read in this order:
1. **`contextSummary`** - Your complete briefing (WHY, WHAT, HOW, TESTING)
2. **`role`** - Your specific objective
3. **🆕 `humanPromptNotes`** - CRITICAL: Read human's additional guidance FIRST
4. **`filesModified`** - EXACT files to create/modify (no searching needed)
5. **`todos`** - Each has `description`, `contextHint`, `filePath`, `functionName`, **🆕 `humanPromptNotes`**
6. **`priorWorkSummary`** - What previous agent did (if multi-phase)
7. **`qdrantCollections`** - Where to find patterns (if you need them)
8. **`notes`** - Gotchas and shortcuts

**🚨 CRITICAL: Human Prompt Notes Priority**

If `humanPromptNotes` is present at task or TODO level, it takes **HIGHEST PRIORITY**:
- Read human notes BEFORE starting implementation
- Human notes may override or clarify AI-generated context
- Human notes may add constraints, gotchas, or preferences
- If conflict between AI context and human notes, **human notes win**

**Example:**
```
Task.contextSummary: "Use REST API for data fetching"
Task.humanPromptNotes: "Actually, use GraphQL - we switched to it last week"
→ Agent MUST use GraphQL (human notes override)
```

**For each TODO:**
1. Read `description` - What to do
2. **🆕 Read `humanPromptNotes` FIRST** - Human's specific guidance
3. Read `contextHint` - HOW to do it (unless overridden by human notes)
4. Read `filePath` - Where to write code
5. Read `functionName` - What to name it
```

**Add to "Implementation Stage - Before starting" section:**

```markdown
**Before starting ANY implementation:**

1. **Check for human prompt notes:**
   ```typescript
   if (task.humanPromptNotes) {
     console.log("🚨 HUMAN GUIDANCE PRESENT - Read before proceeding:");
     console.log(task.humanPromptNotes);
     // Incorporate guidance into your implementation plan
   }
   ```

2. **For each TODO, check for human notes:**
   ```typescript
   for (const todo of task.todos) {
     if (todo.humanPromptNotes) {
       console.log(`🚨 Human guidance for "${todo.description}":`);
       console.log(todo.humanPromptNotes);
       // This may override or clarify the contextHint
     }
   }
   ```

3. **Update coordinator with acknowledgment:**
   ```typescript
   coordinator_update_task_status({
     taskId: task.id,
     status: "in_progress",
     notes: task.humanPromptNotes
       ? "Starting implementation. Reviewed and incorporating human guidance notes."
       : "Starting implementation with context from planning phase"
   });
   ```
```

---

## 🎨 UI/UX Design

### Task Detail View

**Layout:**
```
┌─────────────────────────────────────────────┐
│ Task: Backend Services Specialist           │
│ ─────────────────────────────────────────── │
│                                             │
│ [Task Details Section]                      │
│                                             │
│ ▼ Human Guidance Notes          [Has Notes]│
│   ┌───────────────────────────────────────┐ │
│   │ 📝 Human's markdown notes here...     │ │
│   │                                       │ │
│   │ - Use the new authentication flow    │ │
│   │ - Don't forget to add rate limiting  │ │
│   │                                       │ │
│   │ Added: 2025-10-02 14:30              │ │
│   └───────────────────────────────────────┘ │
│   [Edit Notes] [Clear Notes]                │
│                                             │
│ ▼ TODOs (4)                                 │
│   ☐ Create export handler                  │
│      💡 Mirror documents-api pattern       │
│      📝 Human: Make sure to use streaming  │
│      [Add/Edit Notes]                      │
│                                             │
│   ☐ Implement CSV formatting               │
│      [Add Notes]                           │
└─────────────────────────────────────────────┘
```

### States

1. **No notes added:**
   - Shows placeholder text
   - "Add Notes" button visible (only if task is pending)

2. **Notes present:**
   - Renders markdown content
   - Shows timestamp when added
   - "Edit Notes" and "Clear Notes" buttons (only if task is pending)

3. **Task in progress or completed:**
   - Notes are read-only
   - No edit/add buttons
   - Shows "Notes locked - task in progress" message

### Visual Indicators

- **Task card:** Show 📝 icon if task has human notes
- **TODO item:** Show 📝 badge if TODO has human notes
- **Kanban board:** Add "Has Human Guidance" chip to cards with notes
- **Priority indicator:** Add yellow border if human notes present (draws attention)

---

## 🔒 Security & Permissions

### Access Control

**Who can add/edit notes:**
- ✅ Task creator (human who initiated the request)
- ✅ System administrators
- ❌ Agents (agents can only read)

**Implementation:**
```go
func (h *CoordinatorHandler) handleAddTaskPromptNotes(params map[string]interface{}) (*mcp.CallToolResult, error) {
    // Get user identity from JWT context
    identity, err := auth.GetIdentityFromContext(h.ctx)
    if err != nil {
        return nil, fmt.Errorf("unauthorized: no user identity")
    }

    // Verify user has permission
    task, err := h.storage.GetAgentTask(agentTaskID)
    if err != nil {
        return nil, err
    }

    // Check if user is task creator or admin
    // (Implementation depends on auth system)

    // ... proceed with adding notes
}
```

### Data Validation

**Input validation:**
- Maximum length: 5000 characters (prevent abuse)
- Sanitize markdown (prevent XSS)
- Strip dangerous HTML tags
- Allow safe markdown: headers, lists, code blocks, bold, italic, links

**Implementation:**
```go
import "github.com/microcosm-cc/bluemonday"

func validatePromptNotes(notes string) (string, error) {
    if len(notes) > 5000 {
        return "", fmt.Errorf("prompt notes exceed maximum length of 5000 characters")
    }

    // Sanitize HTML but allow safe markdown
    p := bluemonday.UGCPolicy()
    sanitized := p.Sanitize(notes)

    return sanitized, nil
}
```

### Audit Trail

**Track changes:**
```go
type PromptNotesAudit struct {
    AgentTaskID string    `bson:"agentTaskId"`
    TodoID      string    `bson:"todoId,omitempty"`
    Action      string    `bson:"action"` // "added", "updated", "cleared"
    Notes       string    `bson:"notes"`
    UserID      string    `bson:"userId"`
    Timestamp   time.Time `bson:"timestamp"`
}
```

**Optional: Store audit log in separate collection** (not in v1, but design for it)

---

## 📊 Metrics & Analytics

### Track Usage

**Metrics to collect:**
1. Percentage of tasks with human notes
2. Percentage of TODOs with human notes
3. Average note length
4. Time between task creation and notes being added
5. Correlation between notes and task success rate

**Implementation:**
```go
type PromptNotesMetrics struct {
    TasksWithNotes       int64
    TodosWithNotes       int64
    TotalTasks           int64
    TotalTodos           int64
    AverageNoteLength    float64
    NotesPercentage      float64
}

func (s *MongoTaskStorage) GetPromptNotesMetrics() (*PromptNotesMetrics, error) {
    // Aggregate query to calculate metrics
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**Backend:**
```go
func TestAddTaskPromptNotes(t *testing.T) {
    // Test adding notes to task
    // Test updating notes
    // Test clearing notes
    // Test validation (max length)
    // Test unauthorized access
}

func TestAddTodoPromptNotes(t *testing.T) {
    // Test adding notes to specific TODO
    // Test updating TODO notes
    // Test clearing TODO notes
    // Test invalid TODO ID
}
```

**Frontend:**
```typescript
describe('TaskDetailDialog - Prompt Notes', () => {
  it('should show add notes button when task is pending', () => {});
  it('should allow editing existing notes', () => {});
  it('should disable notes editing when task is in progress', () => {});
  it('should render markdown in notes display', () => {});
  it('should show visual indicator when notes present', () => {});
});
```

### Integration Tests

```go
func TestEndToEndPromptNotes(t *testing.T) {
    // 1. Create agent task
    // 2. Add task-level prompt notes
    // 3. Add TODO-level prompt notes
    // 4. Retrieve task and verify notes present
    // 5. Update notes
    // 6. Clear notes
    // 7. Verify notes cleared
}
```

### Agent Behavior Tests

**Verify agents read notes:**
```typescript
// Create test task with human notes
const task = await createAgentTask({
  agentName: "test-agent",
  role: "Test implementation",
  todos: [...],
  humanPromptNotes: "CRITICAL: Use method X, not method Y"
});

// Execute agent
const result = await executeAgent(task);

// Verify agent acknowledged notes
expect(result.notes).toContain("Reviewed and incorporating human guidance");

// Verify agent followed guidance (method X used)
const code = await readFile(result.filesModified[0]);
expect(code).toContain("methodX");
expect(code).not.toContain("methodY");
```

---

## 📈 Success Metrics

### KPIs

1. **Adoption Rate:**
   - Target: 40% of tasks have human notes within 1 month
   - Measure: Percentage of tasks with `humanPromptNotes` populated

2. **Task Success Rate:**
   - Target: 15% improvement in task completion without rework
   - Measure: Compare tasks with notes vs without notes

3. **Agent Accuracy:**
   - Target: 20% reduction in "blocked" status due to misunderstanding
   - Measure: Count blocked tasks with notes vs without

4. **User Satisfaction:**
   - Target: 8/10 rating for "ability to guide agents"
   - Measure: User survey after 2 weeks

### Monitoring

**Dashboard metrics:**
- Notes usage over time (trend line)
- Average notes length per task
- Most common note keywords (word cloud)
- Tasks with vs without notes (pie chart)
- Note edit frequency (how often notes are updated)

---

## 🚀 Rollout Plan

### Phase 1: MCP Server Implementation (Week 1)

**Project:** `coordinator/mcp-server/`

**Tasks:**
1. Update data models (AgentTask, TodoItem) in `storage/tasks.go`
2. Implement storage layer methods (6 new methods)
3. Register new tools in `handlers/tools.go` GetToolsList()
4. Add MCP tool handlers (6 handlers)
5. Add input validation and sanitization
6. Write unit tests for storage layer
7. Write unit tests for tool handlers

**Deliverables:**
- MongoDB schema updated with 3 new fields per entity
- 6 new MCP tools registered and working
- Input validation (5000 char limit, markdown sanitization)
- 90% test coverage for new code
- Updated HYPERION_COORDINATOR_MCP_REFERENCE.md with new tools

### Phase 2: HTTP Bridge Verification (Day 1 of Week 2)

**Project:** `coordinator/mcp-http-bridge/`

**Tasks:**
1. Verify bridge automatically forwards new tools (should require NO code changes)
2. Test HTTP endpoints with curl
3. Verify JSON-RPC conversion works correctly
4. Test error handling for new tools
5. Update bridge documentation if needed

**Deliverables:**
- Confirmation that bridge works without modifications
- HTTP endpoint tests passing
- curl test scripts in documentation

**Note:** This should be very quick (few hours) since the bridge is designed to be stateless and tool-agnostic.

---

### Phase 3: UI Implementation (Week 2)

**Project:** `coordinator/ui/`

**Tasks:**
1. Update TypeScript types in `src/types/coordinator.ts`
2. Add client methods to `src/services/mcpClient.ts` (6 new methods)
3. Update `TaskDetailDialog.tsx` with notes section
4. Add TODO-level notes UI
5. Add visual indicators (📝 badges) to task cards
6. Implement edit/save/cancel workflows
7. Add markdown rendering
8. Write React component tests
9. Write Playwright E2E tests

**Deliverables:**
- Full UI for adding/editing/clearing notes (task and TODO level)
- Markdown rendering with ReactMarkdown
- Visual indicators throughout UI
- Component tests with React Testing Library
- E2E tests with Playwright

### Phase 4: Agent Integration (Week 3)

**Tasks:**
1. Update CLAUDE.md documentation
2. Update agent workflow prompts
3. Test agent note reading
4. Verify agents follow guidance

**Deliverables:**
- Agents read and acknowledge notes
- Documentation updated
- Behavior tests passing

### Phase 5: Beta Testing (Week 4)

**Tasks:**
1. Internal testing with 3-5 users
2. Gather feedback
3. Bug fixes
4. Performance optimization

**Deliverables:**
- Bug fixes applied
- Performance report
- User feedback incorporated

### Phase 6: Production Release (Week 5)

**Tasks:**
1. Deploy to production
2. Monitor metrics
3. User onboarding guide
4. Support documentation

**Deliverables:**
- Feature live in production
- Metrics dashboard
- User guide published

---

## 🎓 User Guide

### How to Use Prompt Notes

**Step 1: Review the Plan**
After the AI planning phase completes, review all agent tasks and TODOs in the task detail view.

**Step 2: Identify Gaps**
Look for:
- Missing context the AI couldn't know
- Business rules or constraints
- Preferences for implementation approach
- Known gotchas or edge cases

**Step 3: Add Task-Level Notes**
Click "Add Notes" in the "Human Guidance Notes" section to provide overall guidance for the entire task.

**Example:**
```markdown
## Important Context

We recently switched from REST to GraphQL. All new endpoints should use GraphQL.

## Security Note

This feature handles PII data - make sure to:
- Log all access attempts
- Use encryption at rest
- Add rate limiting (100 req/min per user)

## Testing

Test with the staging environment first (API key: staging_xyz...)
```

**Step 4: Add TODO-Level Notes**
For specific TODOs, click "Add Notes" to provide detailed guidance.

**Example:**
```markdown
Use the `StreamDataPipe` utility from utils/streaming.go - it handles backpressure automatically.

Don't use bufio.Writer directly, it doesn't play well with our rate limiter.
```

**Step 5: Review Before Agent Starts**
Once notes are added, the agent will automatically read them before starting implementation. Notes are locked once the agent status changes to "in_progress".

---

## 🔮 Future Enhancements

### V2 Features (Not in Scope for V1)

1. **Note Templates:**
   - Pre-defined templates for common scenarios
   - "Security Notes", "Performance Notes", "Testing Notes"
   - One-click template insertion

2. **Collaborative Notes:**
   - Multiple humans can add notes
   - Show contributor avatars
   - Thread-based discussions

3. **AI-Suggested Notes:**
   - AI analyzes task and suggests what notes might be helpful
   - "Did you consider mentioning X?"
   - Learn from past successful notes

4. **Version History:**
   - Track all note edits
   - Rollback to previous versions
   - Diff view for changes

5. **Note Validation:**
   - AI reviews notes for clarity
   - Suggests improvements
   - Warns about ambiguous guidance

6. **Linked Notes:**
   - Reference other tasks/TODOs in notes
   - Automatic linking when mentioning IDs
   - Cross-task knowledge sharing

7. **Voice Notes:**
   - Record audio notes
   - Auto-transcription to text
   - Faster for long explanations

8. **Note Analytics:**
   - Most effective note patterns
   - Correlation with task success
   - Recommendations for note content

---

## ❓ FAQ

**Q: Can agents edit or add their own notes?**
A: No. The `humanPromptNotes` field is exclusively for human input. Agents can only read these notes. Agents use the existing `notes` field for their own progress updates.

**Q: What happens if human notes conflict with AI-generated context?**
A: Human notes take priority. Agents are instructed to treat human notes as highest priority and override AI context if there's a conflict.

**Q: Can I add notes after an agent has started?**
A: No. Notes are locked once the task status changes from "pending" to "in_progress". This prevents confusion mid-execution.

**Q: Is there a character limit for notes?**
A: Yes, 5000 characters per note (task-level or TODO-level). This is sufficient for detailed guidance while preventing abuse.

**Q: Can I use markdown formatting?**
A: Yes! Full markdown support including headers, lists, code blocks, bold, italic, and links. HTML is sanitized for security.

**Q: Will old tasks without notes still work?**
A: Yes. The `humanPromptNotes` field is optional. If not present, agents follow the existing workflow without any changes.

**Q: How do I know if an agent read my notes?**
A: Agents are required to acknowledge human notes in their status update when starting: "Reviewed and incorporating human guidance notes."

**Q: Can I clear notes once added?**
A: Yes, use the "Clear Notes" button. However, once the task status changes to "in_progress", notes become read-only.

---

## 📝 Appendix

### Example Use Cases

**Use Case 1: Security Constraint**

**Scenario:** AI planning didn't know about new security policy

**Human Note:**
```markdown
🔒 SECURITY REQUIREMENT (New policy as of 2025-10-01):

All endpoints handling user data MUST:
1. Check user's company-level permissions via `checkCompanyAccess(identity, resourceId)`
2. Log access attempts to audit_log collection
3. Return 403 (not 404) when unauthorized

See: docs/security/data-access-policy.md
```

**Use Case 2: Performance Optimization**

**Scenario:** User knows a performance gotcha

**Human Note:**
```markdown
⚡ PERFORMANCE CRITICAL:

The staff collection has 500K+ documents. MUST use pagination:
- Limit: 100 records per page
- Use cursor-based pagination (not offset)
- Add index on `createdAt` field

DO NOT load entire collection into memory.
```

**Use Case 3: Code Pattern Reference**

**Scenario:** User wants specific pattern followed

**Human Note:**
```markdown
📐 FOLLOW THIS PATTERN:

See documents-api/handlers/export.go lines 45-120 for the exact pattern.

Key points:
- Use io.Pipe() for streaming
- Call w.Flush() after each row
- Wrap in goroutine with defer close()

Copy this pattern exactly - it's been tested with 1M+ rows.
```

**Use Case 4: Testing Requirements**

**Scenario:** Specific test cases needed

**Human Note:**
```markdown
🧪 TEST CASES REQUIRED:

Must test:
1. Empty dataset (0 results) - should return empty array, not error
2. Large dataset (10K+ rows) - verify streaming works, no memory spike
3. Invalid filters - should return 400 with clear error message
4. Company isolation - User A should NOT see User B's data

Run with: `go test -v -timeout 60s`
```

### Data Flow Diagram

```
┌─────────────┐
│   Human     │
│   User      │
└──────┬──────┘
       │
       │ 1. Reviews AI plan
       │
       ▼
┌─────────────────────────┐
│  Task Detail UI         │
│                         │
│  - Read task context    │
│  - Click "Add Notes"    │
│  - Enter guidance       │
│  - Save (MCP call)      │
└──────┬──────────────────┘
       │
       │ 2. Add notes via MCP
       │
       ▼
┌─────────────────────────┐
│  Coordinator MCP        │
│                         │
│  - Validate notes       │
│  - Save to MongoDB      │
│  - Add timestamp        │
└──────┬──────────────────┘
       │
       │ 3. Notes stored
       │
       ▼
┌─────────────────────────┐
│  MongoDB                │
│                         │
│  agent_tasks collection │
│  - humanPromptNotes ✅  │
└──────┬──────────────────┘
       │
       │ 4. Agent fetches task
       │
       ▼
┌─────────────────────────┐
│  Agent (Claude)         │
│                         │
│  1. List agent tasks    │
│  2. Read humanPromptNotes│
│  3. Incorporate guidance│
│  4. Start implementation│
└─────────────────────────┘
```

---

**End of Specification**

---

**Review Checklist:**
- [ ] Backend data model changes reviewed
- [ ] API design reviewed
- [ ] Frontend UX reviewed
- [ ] Security considerations addressed
- [ ] Testing strategy complete
- [ ] Rollout plan approved
- [ ] Documentation complete
- [ ] Metrics defined

**Approvals:**
- [ ] Product Manager: _______________
- [ ] Engineering Lead: _______________
- [ ] UX Designer: _______________
- [ ] Security Lead: _______________
