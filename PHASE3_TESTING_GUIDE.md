# Phase 3: Task Hierarchy System - Testing Guide

## Overview
Phase 3 adds 5 new MCP tools for task complexity analysis, task splitting, hierarchy management, and progress tracking.

## New Tools Available

### 1. `coordinator_analyze_task_complexity`
**Purpose:** Analyzes task complexity using multi-heuristic scoring (0.0-1.0)

**Parameters:**
- `title` (string): Task title
- `contextSummary` (string): Detailed task description
- `todos` (array): List of TODO items
- `filesModified` (array): Files that will be modified

**Returns:**
- `score` (float): Complexity score 0.0-1.0
- `level` (string): LOW/MEDIUM/HIGH/VERY_HIGH
- `recommendation` (string): Whether to split or not
- `estimatedTimeMinutes` (int): Estimated completion time
- `factors` (object): Breakdown of complexity factors

---

### 2. `coordinator_split_agent_task`
**Purpose:** Splits a complex parent task into smaller child tasks with dependency management

**Parameters:**
- `parentTaskId` (string): ID of parent task to split
- `splittingStrategy` (string): "SEQUENTIAL" or "PARALLEL"
- `childTasks` (array): Array of child task definitions
  - `title` (string): Child task title
  - `contextSummary` (string): Child task context
  - `todos` (array): Child task TODOs
  - `filesModified` (array): Files for this child
  - `dependsOn` (array, optional): Task titles this depends on
  - `priority` (string, optional): HIGH/MEDIUM/LOW
  - `estimatedMinutes` (int, optional): Time estimate
- `createIntegrationDoc` (bool, optional): Generate integration documentation

**Features:**
- Validates circular dependencies using DFS graph traversal
- Automatically sets sequential dependencies if strategy = "SEQUENTIAL"
- Marks parent task as BLOCKED
- Creates child tasks with proper ordering

---

### 3. `coordinator_get_task_hierarchy`
**Purpose:** Retrieves complete task hierarchy tree with parent-child relationships

**Parameters:**
- `rootTaskId` (string, optional): Start from specific task
- `includeProgress` (bool): Include progress calculations
- `maxDepth` (int): Maximum depth to traverse

**Returns:**
- Tree structure with nested children
- Progress calculations at each level
- Dependency information

---

### 4. `coordinator_get_next_executable_task`
**Purpose:** Finds tasks that can be executed now (all dependencies met)

**Parameters:**
- `parentTaskId` (string, optional): Filter by parent
- `priority` (string, optional): Filter by priority
- `maxResults` (int): Maximum tasks to return
- `includeDetails` (bool): Include full task details
- `sortBy` (string): "priority", "createdAt", "estimatedTime", "orderIndex"

**Returns:**
- List of executable tasks
- Dependency status for each
- Sorting applied

---

### 5. `coordinator_update_child_task_progress`
**Purpose:** Updates child task status and propagates progress to parent

**Parameters:**
- `childTaskId` (string): Child task to update
- `status` (string): PENDING/IN_PROGRESS/COMPLETED/BLOCKED
- `progress` (float, optional): Progress 0.0-1.0 (auto-calculated from TODOs if omitted)
- `notes` (string, optional): Progress notes
- `propagateToParent` (bool, default: true): Update parent status

**Auto-calculates parent status:**
- All children completed → Parent = COMPLETED
- Any child in progress → Parent = IN_PROGRESS
- All children blocked → Parent = BLOCKED

---

## Manual Testing Steps

### Step 1: Start Hyperion
```bash
cd /Users/meghaneelamana/dev-squad
./bin/hyper
```

### Step 2: Test Complexity Analysis

In the Hyperion chat interface, try:

```
Analyze the complexity of this task using coordinator_analyze_task_complexity:
{
  "title": "Implement Real-time Notification System",
  "contextSummary": "Build a real-time notification system using WebSockets, including backend event streaming, frontend notifications UI, MongoDB storage, and integration with existing authentication",
  "todos": [
    "Set up WebSocket server in Go",
    "Create notification event types and schemas",
    "Implement MongoDB notification storage",
    "Build React notification center component",
    "Add real-time event streaming",
    "Integrate with existing auth system",
    "Add notification preferences UI",
    "Implement read/unread tracking"
  ],
  "filesModified": [
    "server/websocket.go",
    "models/notification.go",
    "storage/notifications.go",
    "handlers/notification_handler.go",
    "frontend/src/components/NotificationCenter.tsx",
    "frontend/src/hooks/useNotifications.ts"
  ]
}
```

**Expected Output:**
- Score should be > 0.6 (HIGH complexity)
- Recommendation: "SPLIT - Consider breaking this task into smaller subtasks"
- Estimated time: 180-300 minutes

### Step 3: Create a Parent Task

First create a parent task to split:

```
Create an agent task for "Implement Real-time Notification System" with the details above.
```

Note the task ID returned.

### Step 4: Test Task Splitting

```
Split the parent task (ID: <task-id-from-step-3>) using coordinator_split_agent_task:
{
  "parentTaskId": "<task-id-from-step-3>",
  "splittingStrategy": "SEQUENTIAL",
  "childTasks": [
    {
      "title": "Backend WebSocket Infrastructure",
      "contextSummary": "Set up WebSocket server, event types, and MongoDB storage",
      "todos": [
        "Set up WebSocket server in Go",
        "Create notification event types and schemas",
        "Implement MongoDB notification storage"
      ],
      "filesModified": [
        "server/websocket.go",
        "models/notification.go",
        "storage/notifications.go"
      ],
      "priority": "HIGH"
    },
    {
      "title": "Frontend Notification UI",
      "contextSummary": "Build React notification center and real-time updates",
      "todos": [
        "Build React notification center component",
        "Add real-time event streaming",
        "Add notification preferences UI",
        "Implement read/unread tracking"
      ],
      "filesModified": [
        "frontend/src/components/NotificationCenter.tsx",
        "frontend/src/hooks/useNotifications.ts"
      ],
      "priority": "MEDIUM",
      "dependsOn": ["Backend WebSocket Infrastructure"]
    },
    {
      "title": "Authentication Integration",
      "contextSummary": "Integrate notification system with existing auth",
      "todos": [
        "Integrate with existing auth system"
      ],
      "filesModified": [
        "handlers/notification_handler.go"
      ],
      "priority": "HIGH",
      "dependsOn": ["Backend WebSocket Infrastructure"]
    }
  ],
  "createIntegrationDoc": true
}
```

**Expected Output:**
- 3 child tasks created
- Parent task marked as BLOCKED
- Sequential dependencies set automatically
- Integration documentation generated

### Step 5: Test Task Hierarchy

```
Get the task hierarchy using coordinator_get_task_hierarchy:
{
  "includeProgress": true,
  "maxDepth": 5
}
```

**Expected Output:**
- Tree structure showing parent with 3 children
- Each child shows dependencies
- Progress: 0% (no tasks started)

### Step 6: Test Next Executable Task

```
Find the next executable task using coordinator_get_next_executable_task:
{
  "maxResults": 3,
  "includeDetails": true,
  "sortBy": "priority"
}
```

**Expected Output:**
- "Backend WebSocket Infrastructure" is executable (no dependencies)
- "Frontend Notification UI" is NOT executable (depends on backend)
- "Authentication Integration" is NOT executable (depends on backend)

### Step 7: Test Progress Updates

Start the first child task:

```
Update task progress using coordinator_update_child_task_progress:
{
  "childTaskId": "<backend-task-id>",
  "status": "IN_PROGRESS",
  "progress": 0.5,
  "notes": "WebSocket server implemented, working on MongoDB storage",
  "propagateToParent": true
}
```

**Expected Output:**
- Child task status: IN_PROGRESS
- Parent task status: IN_PROGRESS (because a child is in progress)

Complete the first child task:

```
{
  "childTaskId": "<backend-task-id>",
  "status": "COMPLETED",
  "progress": 1.0,
  "notes": "All backend infrastructure completed",
  "propagateToParent": true
}
```

**Expected Output:**
- Child task status: COMPLETED
- Parent task status: IN_PROGRESS (other children still pending)
- Now "Frontend Notification UI" becomes executable

### Step 8: Verify Executable Tasks Again

```
coordinator_get_next_executable_task with same parameters as Step 6
```

**Expected Output:**
- NOW "Frontend Notification UI" is executable
- NOW "Authentication Integration" is executable
- Both can run in parallel since backend is done

---

## Automated Testing

### Run Go Tests

```bash
cd /Users/meghaneelamana/dev-squad
go test -v ./hyper/internal/ai-service/tools/mcp -run TestPhase3
```

Note: Tests are currently skipped because they require a real MongoDB connection. To enable:

1. Set up test database connection in the test file
2. Remove `t.Skip()` calls
3. Run tests

---

## What to Watch For

### ✅ Expected Behavior

1. **Complexity Analysis:**
   - Score increases with more files, todos, and cross-system dependencies
   - TypeScript files add extra complexity weight
   - Clear recommendations based on thresholds

2. **Task Splitting:**
   - Circular dependency detection prevents invalid splits
   - Sequential strategy auto-sets dependencies
   - Parent task gets blocked when split

3. **Hierarchy:**
   - Correct tree structure
   - Progress aggregation works
   - Nested tasks display properly

4. **Executable Tasks:**
   - Only returns tasks with met dependencies
   - Sorting works correctly
   - Filtering by priority/parent works

5. **Progress Updates:**
   - Status propagates to parent correctly
   - Progress auto-calculated from TODO completion
   - Child completion unblocks dependent tasks

### ⚠️ Known Limitations

1. **Metadata Fields:** Priority and estimated time are not stored in AgentTask struct (removed during Phase 1→3 implementation)
2. **Test Function:** The `testPhase3ToolsIntegration` function exists but isn't wired up as a CLI command
3. **Storage Methods:** Some storage layer methods referenced in Phase 3 spec may need implementation

---

## Troubleshooting

**Problem:** "Parent task not found" error when splitting
- **Solution:** Create the parent task first using `coordinator_create_agent_task`

**Problem:** Task splitting fails with "circular dependency detected"
- **Solution:** Check your `dependsOn` arrays - Task A can't depend on Task B if Task B depends on Task A

**Problem:** No executable tasks found
- **Solution:** Check task dependencies - all parent dependencies must be COMPLETED first

**Problem:** Progress doesn't propagate to parent
- **Solution:** Set `propagateToParent: true` in update call

---

## Success Criteria

✅ Phase 3 is working correctly if:

1. Complexity analysis returns scores between 0.0-1.0 with clear recommendations
2. Task splitting creates child tasks with correct dependencies
3. Circular dependencies are detected and rejected
4. Task hierarchy shows correct parent-child relationships
5. Only tasks with met dependencies are marked as executable
6. Child task completion propagates status to parent
7. Parent status updates based on aggregate child status
8. All 5 new MCP tools are registered and callable

---

## Next Steps: Phase 4

After verifying Phase 3 works:
- Phase 4 will add LLM-powered smart task splitting with Claude integration
- Auto-analysis of task descriptions to suggest optimal split strategies
- Context-aware child task generation
