# ## Overview

**Collection:** backend-services
**Created:** 2025-11-20

---

## Overview

Hyperion exposes a comprehensive REST API for task management, knowledge base operations, code search, and AI chat functionality. This document describes all endpoints, request/response patterns, and the service layer architecture.

## Base Configuration

### Base URL

**Development:** `http://localhost:7095/api/v1`
**Production:** `https://your-domain.com/api/v1`

### HTTP Framework

**Framework:** Gin v1.9.1 (Go)
**Location:** `/hyper/internal/handlers/`

### Common Headers

| Header | Required | Description |
|--------|----------|-------------|
| `Content-Type` | Yes | `application/json` for all POST/PUT requests |
| `Authorization` | Optional | `Bearer <JWT>` for authenticated endpoints |

### Standard Response Format

**Success (200-299):**
```json
{
  "data": { /* response payload */ },
  "message": "optional success message"
}
```

**Error (400-599):**
```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "details": { /* optional additional context */ }
}
```

---

## Chat API

### Create Chat Session

**Endpoint:** `POST /api/v1/chat/sessions`

**Request Body:**
```json
{
  "title": "New chat session"
}
```

**Response:**
```json
{
  "sessionId": "uuid",
  "title": "New chat session",
  "createdAt": "2025-11-20T10:00:00Z"
}
```

### List Chat Sessions

**Endpoint:** `GET /api/v1/chat/sessions`

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 50 | Maximum sessions to return |
| `offset` | int | 0 | Pagination offset |

**Response:**
```json
{
  "sessions": [
    {
      "sessionId": "uuid",
      "title": "Chat session title",
      "createdAt": "2025-11-20T10:00:00Z",
      "updatedAt": "2025-11-20T11:00:00Z",
      "messageCount": 12
    }
  ],
  "total": 100,
  "limit": 50,
  "offset": 0
}
```

### Get Chat Session

**Endpoint:** `GET /api/v1/chat/sessions/:id`

**Response:**
```json
{
  "sessionId": "uuid",
  "title": "Chat session title",
  "messages": [
    {
      "id": "msg-uuid",
      "role": "user",
      "content": "What is the architecture?",
      "timestamp": "2025-11-20T10:00:00Z"
    },
    {
      "id": "msg-uuid",
      "role": "assistant",
      "content": "The architecture consists of...",
      "timestamp": "2025-11-20T10:00:05Z"
    }
  ],
  "createdAt": "2025-11-20T10:00:00Z",
  "updatedAt": "2025-11-20T11:00:00Z"
}
```

### Delete Chat Session

**Endpoint:** `DELETE /api/v1/chat/sessions/:id`

**Response:**
```json
{
  "message": "Session deleted successfully"
}
```

---

## Coordinator API (Task Management)

### List Human Tasks

**Endpoint:** `GET /api/v1/tasks`

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `status` | string | - | Filter by status (pending, in_progress, completed, blocked) |
| `limit` | int | 50 | Maximum tasks to return |
| `offset` | int | 0 | Pagination offset |

**Response:**
```json
{
  "tasks": [
    {
      "taskId": "uuid",
      "prompt": "Create user authentication feature",
      "summary": "Implement JWT-based authentication with login and registration",
      "status": "in_progress",
      "agentTaskIds": ["agent-task-uuid-1", "agent-task-uuid-2"],
      "createdAt": "2025-11-20T10:00:00Z",
      "updatedAt": "2025-11-20T11:00:00Z",
      "notes": "Backend implementation complete, frontend in progress"
    }
  ],
  "total": 25,
  "limit": 50,
  "offset": 0
}
```

### Create Human Task

**Endpoint:** `POST /api/v1/tasks`

**Request Body:**
```json
{
  "prompt": "Add dark mode support to the application"
}
```

**Response:**
```json
{
  "taskId": "uuid",
  "prompt": "Add dark mode support to the application",
  "status": "pending",
  "createdAt": "2025-11-20T10:00:00Z"
}
```

### Update Task Status

**Endpoint:** `PUT /api/v1/tasks/:id/status`

**Request Body:**
```json
{
  "status": "completed",
  "notes": "All requirements implemented and tested"
}
```

**Response:**
```json
{
  "taskId": "uuid",
  "status": "completed",
  "updatedAt": "2025-11-20T12:00:00Z"
}
```

### List Agent Tasks

**Endpoint:** `GET /api/v1/agent-tasks`

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `humanTaskId` | string | - | Filter by parent human task |
| `agentName` | string | - | Filter by agent name |
| `status` | string | - | Filter by status |
| `limit` | int | 50 | Maximum tasks to return (max: 50) |
| `offset` | int | 0 | Pagination offset |

**Response:**
```json
{
  "tasks": [
    {
      "taskId": "uuid",
      "humanTaskId": "parent-uuid",
      "agentName": "go-dev",
      "role": "Implement backend authentication service",
      "status": "in_progress",
      "todos": [
        {
          "id": "todo-uuid",
          "description": "Create JWT middleware",
          "status": "completed",
          "filePath": "hyper/internal/middleware/jwt_auth.go",
          "notes": "Implemented at lines 15-45",
          "createdAt": "2025-11-20T10:00:00Z",
          "completedAt": "2025-11-20T10:30:00Z"
        }
      ],
      "contextSummary": "Build JWT-based authentication...",
      "filesModified": ["hyper/internal/middleware/jwt_auth.go"],
      "createdAt": "2025-11-20T10:00:00Z",
      "updatedAt": "2025-11-20T11:00:00Z"
    }
  ],
  "total": 10,
  "limit": 50,
  "offset": 0
}
```

### Get Agent Task (Full Details)

**Endpoint:** `GET /api/v1/agent-tasks/:id`

**Response:** Same schema as list item, but with full untruncated fields

### Create Agent Task

**Endpoint:** `POST /api/v1/agent-tasks`

**Request Body:**
```json
{
  "humanTaskId": "parent-uuid",
  "agentName": "go-dev",
  "role": "Implement backend authentication service",
  "todos": [
    {
      "description": "Create JWT middleware",
      "filePath": "hyper/internal/middleware/jwt_auth.go",
      "functionName": "JWTAuth",
      "contextHint": "Use gin.HandlerFunc, extract token from Authorization header"
    },
    {
      "description": "Add login endpoint",
      "filePath": "hyper/internal/handlers/auth_handler.go",
      "functionName": "Login"
    }
  ],
  "contextSummary": "Build JWT-based authentication with login/register endpoints...",
  "filesModified": [
    "hyper/internal/middleware/jwt_auth.go",
    "hyper/internal/handlers/auth_handler.go"
  ],
  "qdrantCollections": ["backend-services"]
}
```

**Response:**
```json
{
  "taskId": "uuid",
  "humanTaskId": "parent-uuid",
  "agentName": "go-dev",
  "status": "pending",
  "createdAt": "2025-11-20T10:00:00Z"
}
```

### Update TODO Status

**Endpoint:** `PUT /api/v1/agent-tasks/:taskId/todos/:todoId/status`

**Request Body:**
```json
{
  "status": "completed",
  "notes": "JWT middleware implemented with token validation and claims extraction"
}
```

**Response:**
```json
{
  "message": "TODO status updated successfully",
  "taskStatus": "in_progress"
}
```

**Note:** When all TODOs are completed, the agent task is automatically marked as completed.

### Add Task Prompt Notes

**Endpoint:** `POST /api/v1/agent-tasks/:id/prompt-notes`

**Request Body:**
```json
{
  "promptNotes": "Please ensure proper error handling and add unit tests for edge cases"
}
```

**Response:**
```json
{
  "message": "Prompt notes added successfully"
}
```

### Update Task Prompt Notes

**Endpoint:** `PUT /api/v1/agent-tasks/:id/prompt-notes`

**Request Body:** Same as add

### Clear Task Prompt Notes

**Endpoint:** `DELETE /api/v1/agent-tasks/:id/prompt-notes`

**Response:**
```json
{
  "message": "Prompt notes cleared successfully"
}
```

---

## Knowledge Base API

### Query Knowledge

**Endpoint:** `POST /api/v1/knowledge/query`

**Request Body:**
```json
{
  "collection": "technical-knowledge",
  "query": "golang error handling patterns",
  "limit": 5,
  "taskId": "optional-task-uuid"
}
```

**Response:**
```json
{
  "results": [
    {
      "id": "entry-uuid",
      "text": "In Go, errors are values. Always check errors explicitly...",
      "score": 0.92,
      "metadata": {
        "sourceFile": "docs/kb/golang-patterns.md",
        "tags": ["golang", "error-handling"]
      },
      "createdAt": "2025-11-20T10:00:00Z"
    }
  ]
}
```

### Upsert Knowledge

**Endpoint:** `POST /api/v1/knowledge`

**Request Body:**
```json
{
  "collectionName": "technical-knowledge",
  "information": "MongoDB indexing best practices: Always create indexes on frequently queried fields...",
  "metadata": {
    "author": "system",
    "tags": ["mongodb", "performance"]
  },
  "taskId": "optional-task-uuid"
}
```

**Response:**
```json
{
  "id": "entry-uuid",
  "collectionId": "collection-object-id",
  "message": "Knowledge entry created successfully"
}
```

### List Collections

**Endpoint:** `GET /api/v1/knowledge/collections`

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 5 | Maximum collections to return |

**Response:**
```json
{
  "collections": [
    {
      "id": "collection-object-id",
      "name": "technical-knowledge",
      "category": "engineering",
      "description": "Technical patterns and best practices",
      "tags": ["patterns", "best-practices"],
      "entryCount": 142,
      "createdAt": "2025-11-01T00:00:00Z",
      "updatedAt": "2025-11-20T10:00:00Z"
    }
  ]
}
```

### Browse Knowledge Entries

**Endpoint:** `GET /api/v1/knowledge/browse`

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `collectionId` | string | - | Filter by collection (ObjectID) |
| `taskId` | string | - | Filter by task scope |
| `limit` | int | 20 | Maximum entries to return |
| `offset` | int | 0 | Pagination offset |

**Response:**
```json
{
  "entries": [
    {
      "id": "entry-uuid",
      "collectionId": "collection-object-id",
      "text": "Entry content...",
      "metadata": {},
      "createdAt": "2025-11-20T10:00:00Z"
    }
  ],
  "total": 142,
  "limit": 20,
  "offset": 0
}
```

### Vote on Knowledge Entry

**Endpoint:** `POST /api/v1/knowledge/vote`

**Request Body:**
```json
{
  "entryId": "entry-uuid",
  "vote": "+",
  "reason": "This pattern solved my authentication issue"
}
```

**Response:**
```json
{
  "message": "Vote recorded successfully",
  "entryId": "entry-uuid",
  "upvotes": 12,
  "downvotes": 2,
  "netScore": 10
}
```

### Sync Markdown KB

**Endpoint:** `POST /api/v1/knowledge/sync-markdown-kb`

**Request Body:** None (uses `KB_DOCS_PATH` environment variable)

**Response:**
```json
{
  "filesProcessed": 14,
  "entriesCreated": 8,
  "entriesUpdated": 6,
  "collections": ["infrastructure", "backend-services", "frontend-patterns"],
  "errors": []
}
```

---

## Code Search API

### Add Folder for Indexing

**Endpoint:** `POST /api/v1/code/folders`

**Request Body:**
```json
{
  "path": "/path/to/code",
  "description": "Main application code",
  "includePatterns": ["*.go", "*.ts", "*.tsx"],
  "excludePatterns": ["node_modules", ".git", "dist"],
  "chunkSize": "m"
}
```

**Response:**
```json
{
  "folderId": "uuid",
  "path": "/path/to/code",
  "status": "scanning",
  "addedAt": "2025-11-20T10:00:00Z"
}
```

### List Indexed Folders

**Endpoint:** `GET /api/v1/code/folders`

**Response:**
```json
{
  "folders": [
    {
      "id": "uuid",
      "path": "/path/to/code",
      "description": "Main application code",
      "status": "active",
      "fileCount": 342,
      "lastScanned": "2025-11-20T09:00:00Z",
      "addedAt": "2025-11-15T10:00:00Z"
    }
  ]
}
```

### Remove Folder

**Endpoint:** `DELETE /api/v1/code/folders/:id`

**Response:**
```json
{
  "message": "Folder and all associated files removed successfully"
}
```

### Scan Folder

**Endpoint:** `POST /api/v1/code/folders/:id/scan`

**Response:**
```json
{
  "message": "Scan initiated",
  "folderId": "uuid",
  "status": "scanning"
}
```

### Reindex All

**Endpoint:** `POST /api/v1/code/reindex`

**Response:**
```json
{
  "message": "Reindexing initiated for all folders",
  "folderCount": 3
}
```

### Clear All Index Data

**Endpoint:** `DELETE /api/v1/code/clear-all`

**Response:**
```json
{
  "message": "All index data cleared successfully",
  "foldersRemoved": 3,
  "filesRemoved": 342,
  "chunksRemoved": 5420
}
```

### Search Code

**Endpoint:** `POST /api/v1/code/search`

**Request Body:**
```json
{
  "query": "authentication middleware implementation",
  "folderPath": "/path/to/code",
  "limit": 10,
  "minScore": 0.7,
  "retrieve": "chunk-m",
  "fileTypes": [".go", ".ts"]
}
```

**Response:**
```json
{
  "results": [
    {
      "fileId": "file-uuid",
      "filePath": "/path/to/code/middleware/auth.go",
      "content": "func JWTAuth() gin.HandlerFunc {\n  return func(c *gin.Context) {\n    ...\n  }\n}",
      "lineStart": 15,
      "lineEnd": 45,
      "score": 0.92,
      "language": "go",
      "chunkType": "ast",
      "nodeType": "function",
      "nodeName": "JWTAuth",
      "signature": "func JWTAuth() gin.HandlerFunc"
    }
  ]
}
```

### Get File Content

**Endpoint:** `GET /api/v1/code/file/:fileId`

**Response:**
```json
{
  "fileId": "uuid",
  "path": "/path/to/code/middleware/auth.go",
  "content": "package middleware\n\nimport (\n  \"github.com/gin-gonic/gin\"\n)\n\n// JWTAuth middleware...",
  "language": "go",
  "lineCount": 120,
  "size": 3580
}
```

### Get File Chunks

**Endpoint:** `GET /api/v1/code/file/:fileId/chunks`

**Response:**
```json
{
  "chunks": [
    {
      "chunkNum": 0,
      "content": "func JWTAuth() gin.HandlerFunc {...}",
      "startLine": 15,
      "endLine": 45,
      "nodeType": "function",
      "nodeName": "JWTAuth"
    }
  ]
}
```

### Get Index Status

**Endpoint:** `GET /api/v1/code/status`

**Response:**
```json
{
  "folderCount": 3,
  "fileCount": 342,
  "chunkCount": 5420,
  "totalSize": 12582912,
  "lastIndexed": "2025-11-20T09:00:00Z",
  "status": "active"
}
```

---

## Frontend Service Layer

### REST Client

**File:** `/ui/src/services/restClient.ts`

```typescript
const BASE_URL = '/api/v1'

class RestClient {
  private async fetchJSON<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${BASE_URL}${endpoint}`, {
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({
        message: 'Unknown error',
      }))
      throw new Error(error.message || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // Task methods
  async listHumanTasks(): Promise<HumanTask[]> {
    return this.fetchJSON<HumanTask[]>('/tasks')
  }

  async createHumanTask(prompt: string): Promise<{ taskId: string }> {
    return this.fetchJSON('/tasks', {
      method: 'POST',
      body: JSON.stringify({ prompt }),
    })
  }

  // Agent task methods
  async listAgentTasks(params?: {
    agentName?: string
    humanTaskId?: string
  }): Promise<AgentTask[]> {
    const query = new URLSearchParams()
    if (params?.agentName) query.set('agentName', params.agentName)
    if (params?.humanTaskId) query.set('humanTaskId', params.humanTaskId)

    const endpoint = `/agent-tasks${query.toString() ? `?${query}` : ''}`
    return this.fetchJSON<AgentTask[]>(endpoint)
  }

  // Knowledge methods
  async queryKnowledge(
    collection: string,
    query: string,
    limit = 5
  ): Promise<KnowledgeResult[]> {
    return this.fetchJSON('/knowledge/query', {
      method: 'POST',
      body: JSON.stringify({ collection, query, limit }),
    })
  }
}

export const restClient = new RestClient()
```

### Usage in Components

```typescript
import { restClient } from '@services/restClient'
import { useState, useEffect } from 'react'

function TaskList() {
  const [tasks, setTasks] = useState<HumanTask[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadTasks()
  }, [])

  async function loadTasks() {
    try {
      setLoading(true)
      const data = await restClient.listHumanTasks()
      setTasks(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load tasks')
    } finally {
      setLoading(false)
    }
  }

  if (loading) return <div>Loading...</div>
  if (error) return <div>Error: {error}</div>

  return (
    <div>
      {tasks.map(task => (
        <TaskCard key={task.taskId} task={task} />
      ))}
    </div>
  )
}
```

## Related Documents

- [Data Contracts](./data-contracts.md) - Request/response TypeScript interfaces
- [MongoDB Schemas](./mongodb-schemas.md) - Database schema reference
- [UI Client Stack](./ui-client-stack.md) - Frontend technologies
- [Component Architecture](./component-architecture.md) - System structure

## Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Request body validation failed |
| `UNAUTHORIZED` | 401 | Missing or invalid authentication |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `INTERNAL_ERROR` | 500 | Server error |
| `SERVICE_UNAVAILABLE` | 503 | Dependent service unavailable |
