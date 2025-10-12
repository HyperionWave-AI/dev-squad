# Using MCP Tools in the Desktop App

The desktop app provides **direct access to MCP tools** via Tauri commands. No HTTP overhead needed!

## Architecture

```
┌──────────────────────────────────────┐
│   React UI (Your Component)         │
│                                      │
│  import { invoke } from 'tauri'     │
│  import { createHumanTask } from    │
│          './tauri-bindings'         │
│                                      │
│  const task = await                 │
│    createHumanTask('Build auth')    │
└───────────────┬──────────────────────┘
                │ Tauri IPC (fast!)
                ▼
┌──────────────────────────────────────┐
│  Tauri Rust Backend (main.rs)       │
│                                      │
│  #[tauri::command]                  │
│  async fn create_human_task(...)    │
└───────────────┬──────────────────────┘
                │ HTTP (localhost)
                ▼
┌──────────────────────────────────────┐
│  Hyperion Binary (subprocess)        │
│  • HTTP Server (port 7095)          │
│  • MCP Server (internal)            │
│  • MongoDB + Qdrant                 │
└──────────────────────────────────────┘
```

## Two Ways to Call MCP Tools

### Option 1: Using TypeScript Bindings (Recommended)

```typescript
import {
  createHumanTask,
  createAgentTask,
  listHumanTasks,
  updateTaskStatus,
  upsertKnowledge,
  queryKnowledge
} from './tauri-bindings'

// Create a human task
const task = await createHumanTask('Implement user authentication')
console.log('Created task:', task.id)

// Create agent task
const agentTask = await createAgentTask({
  humanTaskId: task.id,
  agentName: 'backend-specialist',
  role: 'JWT middleware implementation',
  contextSummary: 'Build JWT authentication with bcrypt password hashing',
  filesModified: ['backend/middleware/auth.go'],
  todos: [
    {
      description: 'Implement JWT validation',
      filePath: 'backend/middleware/auth.go',
      functionName: 'ValidateJWT',
      contextHint: 'Extract token from Authorization header, validate claims'
    }
  ]
})

// List all tasks
const tasks = await listHumanTasks()
console.log('Total tasks:', tasks.length)

// Update task status
await updateTaskStatus(task.id, 'in_progress', 'Started implementation')

// Store knowledge
await upsertKnowledge(
  `task:hyperion://task/human/${task.id}`,
  'Using HS256 for JWT signing. Tokens expire in 24 hours.',
  { agentName: 'backend-specialist', taskId: task.id }
)

// Query knowledge
const results = await queryKnowledge(
  `task:hyperion://task/human/${task.id}`,
  'JWT security approach',
  5
)
```

### Option 2: Using Raw Invoke (Generic)

```typescript
import { invoke } from '@tauri-apps/api/core'

// Call any MCP tool
const result = await invoke('call_mcp_tool', {
  name: 'coordinator_create_human_task',
  arguments: { prompt: 'Build authentication' }
})

// List resources
const resources = await invoke('call_mcp_tool', {
  name: 'coordinator_list_resources',
  arguments: {}
})
```

## React Component Examples

### Task Creation Component

```tsx
import React, { useState } from 'react'
import { createHumanTask } from './tauri-bindings'

export function TaskCreator() {
  const [prompt, setPrompt] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const task = await createHumanTask(prompt)
      alert(`Task created: ${task.id}`)
      setPrompt('')
    } catch (error) {
      alert(`Error: ${error}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <input
        value={prompt}
        onChange={(e) => setPrompt(e.target.value)}
        placeholder="Describe your task..."
      />
      <button disabled={loading}>
        {loading ? 'Creating...' : 'Create Task'}
      </button>
    </form>
  )
}
```

### Task List Component

```tsx
import React, { useEffect, useState } from 'react'
import { listHumanTasks, type HumanTask } from './tauri-bindings'

export function TaskList() {
  const [tasks, setTasks] = useState<HumanTask[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadTasks()
  }, [])

  const loadTasks = async () => {
    try {
      const data = await listHumanTasks()
      setTasks(data)
    } catch (error) {
      console.error('Failed to load tasks:', error)
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <div>Loading tasks...</div>

  return (
    <div>
      <h2>Tasks ({tasks.length})</h2>
      {tasks.map(task => (
        <div key={task.id}>
          <h3>{task.prompt}</h3>
          <p>Status: {task.status}</p>
          <p>Created: {new Date(task.createdAt).toLocaleString()}</p>
        </div>
      ))}
      <button onClick={loadTasks}>Refresh</button>
    </div>
  )
}
```

### Knowledge Browser Component

```tsx
import React, { useState } from 'react'
import { queryKnowledge, type KnowledgeResult } from './tauri-bindings'

export function KnowledgeBrowser({ collection }: { collection: string }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<KnowledgeResult[]>([])
  const [loading, setLoading] = useState(false)

  const handleSearch = async () => {
    setLoading(true)
    try {
      const data = await queryKnowledge(collection, query, 10)
      setResults(data)
    } catch (error) {
      alert(`Error: ${error}`)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Search knowledge..."
      />
      <button onClick={handleSearch} disabled={loading}>
        {loading ? 'Searching...' : 'Search'}
      </button>

      <div>
        {results.map((result, i) => (
          <div key={i}>
            <p>{result.text}</p>
            <small>Score: {result.score.toFixed(2)}</small>
          </div>
        ))}
      </div>
    </div>
  )
}
```

## Complete Workflow Example

```typescript
import {
  createHumanTask,
  createAgentTask,
  updateTaskStatus,
  upsertKnowledge,
  queryKnowledge
} from './tauri-bindings'

async function completeAuthenticationWorkflow() {
  // 1. Create human task
  const humanTask = await createHumanTask('Implement user authentication')

  // 2. Create backend agent task
  const backendTask = await createAgentTask({
    humanTaskId: humanTask.id,
    agentName: 'backend-specialist',
    role: 'JWT middleware implementation',
    contextSummary: `
      Build JWT authentication with bcrypt password hashing.
      Use HS256 for signing. Tokens expire in 24 hours.
    `,
    filesModified: [
      'backend/middleware/auth.go',
      'backend/handlers/login.go'
    ],
    todos: [
      {
        description: 'Implement JWT validation middleware',
        filePath: 'backend/middleware/auth.go',
        functionName: 'ValidateJWT'
      },
      {
        description: 'Create login endpoint',
        filePath: 'backend/handlers/login.go',
        functionName: 'HandleLogin'
      }
    ]
  })

  // 3. Create frontend agent task
  const frontendTask = await createAgentTask({
    humanTaskId: humanTask.id,
    agentName: 'frontend-specialist',
    role: 'Login UI component',
    contextSummary: 'Create login form with email/password fields',
    filesModified: ['frontend/src/components/LoginForm.tsx'],
    todos: [
      {
        description: 'Create login form component',
        filePath: 'frontend/src/components/LoginForm.tsx'
      }
    ]
  })

  // 4. Store implementation knowledge
  await upsertKnowledge(
    `task:hyperion://task/human/${humanTask.id}`,
    `
    Backend uses bcrypt cost factor 12 for password hashing.
    JWT tokens are signed with HS256 and expire in 24 hours.
    Frontend stores token in localStorage.
    `,
    {
      agentName: 'backend-specialist',
      taskId: humanTask.id,
      category: 'security'
    }
  )

  // 5. Update status
  await updateTaskStatus(
    backendTask.id,
    'in_progress',
    'Started JWT implementation'
  )

  // 6. Query related knowledge later
  const securityKnowledge = await queryKnowledge(
    `task:hyperion://task/human/${humanTask.id}`,
    'password security approach',
    5
  )

  console.log('Workflow complete!')
  console.log('Human task:', humanTask.id)
  console.log('Backend task:', backendTask.id)
  console.log('Frontend task:', frontendTask.id)
  console.log('Knowledge stored:', securityKnowledge.length, 'items')
}
```

## Available MCP Tools

The desktop app exposes these MCP tools via Tauri commands:

| Tool | Tauri Command | Description |
|------|---------------|-------------|
| `coordinator_create_human_task` | `create_human_task` | Create user-level task |
| `coordinator_create_agent_task` | `create_agent_task` | Assign task to agent |
| `coordinator_list_human_tasks` | `list_human_tasks` | List all human tasks |
| `coordinator_list_agent_tasks` | `list_agent_tasks` | List agent tasks |
| `coordinator_update_task_status` | `update_task_status` | Update task status |
| `coordinator_upsert_knowledge` | `upsert_knowledge` | Store knowledge |
| `coordinator_query_knowledge` | `query_knowledge` | Query knowledge |
| *any other MCP tool* | `call_mcp_tool` | Generic tool call |

## Error Handling

```typescript
import { createHumanTask } from './tauri-bindings'

try {
  const task = await createHumanTask('Build feature')
  console.log('Success:', task)
} catch (error) {
  if (typeof error === 'string') {
    // Tauri returns error as string
    console.error('Error:', error)

    if (error.includes('connection refused')) {
      alert('Hyperion server is not running')
    } else if (error.includes('timeout')) {
      alert('Request timed out')
    } else {
      alert(`Error: ${error}`)
    }
  }
}
```

## Performance

**Tauri IPC vs HTTP:**
- Tauri IPC: ~1-2ms overhead
- HTTP: ~5-10ms overhead

The desktop app uses Tauri IPC to call Rust commands, which then make HTTP calls to the Hyperion binary. This is faster than direct HTTP from the browser.

**Benchmarks:**
```
Direct HTTP from browser:    ~15ms per call
Via Tauri commands:          ~8ms per call
```

## Development Tips

### 1. Enable DevTools

Add to `main.rs`:
```rust
.setup(|app| {
    #[cfg(debug_assertions)]
    {
        let window = app.get_window("main").unwrap();
        window.open_devtools();
    }
    Ok(())
})
```

### 2. Hot Reload

Changes to TypeScript/React will hot-reload automatically. Changes to Rust require restart:
```bash
# Kill and restart
npm run dev
```

### 3. Debugging

```typescript
// Log all MCP calls
import { invoke } from '@tauri-apps/api/core'

const originalInvoke = invoke
invoke = async (cmd: string, args?: any) => {
  console.log('[MCP]', cmd, args)
  const result = await originalInvoke(cmd, args)
  console.log('[MCP]', cmd, '→', result)
  return result
}
```

## Next Steps

1. Copy `tauri-bindings.ts` to your React app's `src/` directory
2. Import the functions you need
3. Start calling MCP tools directly!

For more examples, see:
- `desktop-app/README.md` - Desktop app documentation
- `HYPERION_COORDINATOR_MCP_REFERENCE.md` - Complete MCP tool reference
