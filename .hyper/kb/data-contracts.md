# Data Contracts and Type Definitions

**Collection:** data-architecture
**Tags:** types, interfaces, schemas, contracts, typescript, go
**Version:** 1.0
**Last Updated:** 2025-11-20

---

## Overview

This document defines all data contracts used across the Hyperion system, including TypeScript interfaces for the frontend and Go structs for the backend. These contracts ensure type safety and consistent data structures across API boundaries.

---

## Common Types

### Status Enums

#### TaskStatus (Go & TypeScript)

```typescript
// TypeScript
type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'blocked'
```

```go
// Go
type TaskStatus string

const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusBlocked    TaskStatus = "blocked"
)
```

#### TodoStatus (Go & TypeScript)

```typescript
// TypeScript
type TodoStatus = 'pending' | 'in_progress' | 'completed'
```

```go
// Go
type TodoStatus string

const (
    TodoStatusPending    TodoStatus = "pending"
    TodoStatusInProgress TodoStatus = "in_progress"
    TodoStatusCompleted  TodoStatus = "completed"
)
```

### Chunk Size Types

```typescript
// TypeScript
type ChunkSize = 'xs' | 's' | 'm' | 'l' | 'xl'

// Line count mapping
const CHUNK_SIZES = {
  xs: 50,
  s: 100,
  m: 200,    // default
  l: 400,
  xl: 800,
} as const
```

```go
// Go
type ChunkSize string

const (
    ChunkSizeXS ChunkSize = "xs"  // 50 lines
    ChunkSizeS  ChunkSize = "s"   // 100 lines
    ChunkSizeM  ChunkSize = "m"   // 200 lines (default)
    ChunkSizeL  ChunkSize = "l"   // 400 lines
    ChunkSizeXL ChunkSize = "xl"  // 800 lines
)
```

### Retrieve Modes

```typescript
// TypeScript
type RetrieveMode =
  | 'chunk-s'    // 100 lines
  | 'chunk-m'    // 200 lines (default)
  | 'chunk-l'    // 400 lines
  | 'chunk-xl'   // 800 lines
  | 'full'       // entire file
```

---

## Task Management Contracts

### HumanTask

**TypeScript:**
```typescript
interface HumanTask {
  taskId: string
  prompt: string
  summary?: string                    // AI-generated summary
  status: TaskStatus
  agentTaskIds?: string[]
  notes?: string
  createdAt: string                   // ISO8601
  updatedAt: string                   // ISO8601
}
```

**Go:**
```go
type HumanTask struct {
    ID           string     `json:"taskId" bson:"taskId"`
    Prompt       string     `json:"prompt" bson:"prompt"`
    Summary      string     `json:"summary,omitempty" bson:"summary,omitempty"`
    Status       TaskStatus `json:"status" bson:"status"`
    AgentTaskIDs []string   `json:"agentTaskIds,omitempty" bson:"agentTaskIds,omitempty"`
    Notes        string     `json:"notes,omitempty" bson:"notes,omitempty"`
    CreatedAt    time.Time  `json:"createdAt" bson:"createdAt"`
    UpdatedAt    time.Time  `json:"updatedAt" bson:"updatedAt"`
}
```

### AgentTask

**TypeScript:**
```typescript
interface AgentTask {
  taskId: string
  humanTaskId: string
  agentName: string
  role: string
  status: TaskStatus
  todos: TodoItem[]
  notes?: string
  contextSummary?: string
  filesModified?: string[]
  qdrantCollections?: string[]
  priorWorkSummary?: string
  humanPromptNotes?: string
  humanPromptNotesAddedAt?: string    // ISO8601
  humanPromptNotesUpdatedAt?: string  // ISO8601
  createdAt: string                   // ISO8601
  updatedAt: string                   // ISO8601
}
```

**Go:**
```go
type AgentTask struct {
    ID                        string     `json:"taskId" bson:"taskId"`
    HumanTaskID               string     `json:"humanTaskId" bson:"humanTaskId"`
    AgentName                 string     `json:"agentName" bson:"agentName"`
    Role                      string     `json:"role" bson:"role"`
    Status                    TaskStatus `json:"status" bson:"status"`
    Todos                     []TodoItem `json:"todos" bson:"todos"`
    Notes                     string     `json:"notes,omitempty" bson:"notes,omitempty"`
    ContextSummary            string     `json:"contextSummary,omitempty" bson:"contextSummary,omitempty"`
    FilesModified             []string   `json:"filesModified,omitempty" bson:"filesModified,omitempty"`
    QdrantCollections         []string   `json:"qdrantCollections,omitempty" bson:"qdrantCollections,omitempty"`
    PriorWorkSummary          string     `json:"priorWorkSummary,omitempty" bson:"priorWorkSummary,omitempty"`
    HumanPromptNotes          string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
    HumanPromptNotesAddedAt   *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
    HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
    CreatedAt                 time.Time  `json:"createdAt" bson:"createdAt"`
    UpdatedAt                 time.Time  `json:"updatedAt" bson:"updatedAt"`
}
```

### TodoItem

**TypeScript:**
```typescript
interface TodoItem {
  id: string
  description: string
  status: TodoStatus
  filePath?: string
  functionName?: string
  contextHint?: string
  notes?: string
  humanPromptNotes?: string
  humanPromptNotesAddedAt?: string
  humanPromptNotesUpdatedAt?: string
  createdAt: string                   // ISO8601
  completedAt?: string                // ISO8601
}
```

**Go:**
```go
type TodoItem struct {
    ID                        string     `json:"id" bson:"id"`
    Description               string     `json:"description" bson:"description"`
    Status                    TodoStatus `json:"status" bson:"status"`
    FilePath                  string     `json:"filePath,omitempty" bson:"filePath,omitempty"`
    FunctionName              string     `json:"functionName,omitempty" bson:"functionName,omitempty"`
    ContextHint               string     `json:"contextHint,omitempty" bson:"contextHint,omitempty"`
    Notes                     string     `json:"notes,omitempty" bson:"notes,omitempty"`
    HumanPromptNotes          string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
    HumanPromptNotesAddedAt   *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
    HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
    CreatedAt                 time.Time  `json:"createdAt" bson:"createdAt"`
    CompletedAt               *time.Time `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
}
```

### API Request/Response Types

#### CreateHumanTaskRequest

```typescript
// TypeScript
interface CreateHumanTaskRequest {
  prompt: string
}

interface CreateHumanTaskResponse {
  taskId: string
  prompt: string
  status: TaskStatus
  createdAt: string
}
```

```go
// Go
type CreateHumanTaskRequest struct {
    Prompt string `json:"prompt" binding:"required"`
}

type CreateHumanTaskResponse struct {
    TaskID    string     `json:"taskId"`
    Prompt    string     `json:"prompt"`
    Status    TaskStatus `json:"status"`
    CreatedAt time.Time  `json:"createdAt"`
}
```

#### CreateAgentTaskRequest

```typescript
// TypeScript
interface CreateAgentTaskRequest {
  humanTaskId: string
  agentName: string
  role: string
  todos: Array<{
    description: string
    filePath?: string
    functionName?: string
    contextHint?: string
  }>
  contextSummary?: string
  filesModified?: string[]
  priorWorkSummary?: string
  qdrantCollections?: string[]
}
```

```go
// Go
type CreateAgentTaskRequest struct {
    HumanTaskID      string              `json:"humanTaskId" binding:"required"`
    AgentName        string              `json:"agentName" binding:"required"`
    Role             string              `json:"role" binding:"required"`
    Todos            []TodoItemCreate    `json:"todos" binding:"required"`
    ContextSummary   string              `json:"contextSummary,omitempty"`
    FilesModified    []string            `json:"filesModified,omitempty"`
    PriorWorkSummary string              `json:"priorWorkSummary,omitempty"`
    QdrantCollections []string           `json:"qdrantCollections,omitempty"`
}

type TodoItemCreate struct {
    Description  string `json:"description" binding:"required"`
    FilePath     string `json:"filePath,omitempty"`
    FunctionName string `json:"functionName,omitempty"`
    ContextHint  string `json:"contextHint,omitempty"`
}
```

---

## Knowledge Base Contracts

### Collection

**TypeScript:**
```typescript
interface Collection {
  id: string
  name: string
  qdrantName: string
  category: string
  description: string
  tags: string[]
  entryCount: number
  createdAt: string                   // ISO8601
  updatedAt: string                   // ISO8601
}
```

**Go:**
```go
type Collection struct {
    ID          primitive.ObjectID `json:"id" bson:"_id"`
    Name        string             `json:"name" bson:"name"`
    QdrantName  string             `json:"qdrantName" bson:"qdrantName"`
    Category    string             `json:"category" bson:"category"`
    Description string             `json:"description" bson:"description"`
    Tags        []string           `json:"tags" bson:"tags"`
    EntryCount  int                `json:"entryCount" bson:"entryCount"`
    CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
    UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}
```

### KnowledgeEntry

**TypeScript:**
```typescript
interface KnowledgeEntry {
  id: string
  collectionId?: string
  collection?: string                 // DEPRECATED
  taskId?: string
  text: string
  metadata?: Record<string, any>
  createdAt: string                   // ISO8601
}
```

**Go:**
```go
type KnowledgeEntry struct {
    ID           string                 `json:"id" bson:"entryId"`
    CollectionID primitive.ObjectID     `json:"collectionId,omitempty" bson:"collectionId,omitempty"`
    Collection   string                 `json:"collection" bson:"collection"` // DEPRECATED
    TaskID       string                 `json:"taskId,omitempty" bson:"taskId,omitempty"`
    Text         string                 `json:"text" bson:"text"`
    Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
    CreatedAt    time.Time              `json:"createdAt" bson:"createdAt"`
}
```

### Vote

**TypeScript:**
```typescript
interface Vote {
  entryId: string
  userId: string
  vote: '+' | '-'
  reason: string
  createdAt: string                   // ISO8601
  updatedAt: string                   // ISO8601
}
```

**Go:**
```go
type Vote struct {
    EntryID   string    `json:"entryId" bson:"entryId"`
    UserID    string    `json:"userId" bson:"userId"`
    Vote      string    `json:"vote" bson:"vote"` // "+" or "-"
    Reason    string    `json:"reason" bson:"reason"`
    CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
```

### API Request/Response Types

#### QueryKnowledgeRequest

```typescript
// TypeScript
interface QueryKnowledgeRequest {
  collection: string
  query: string
  limit?: number                      // default: 5
  taskId?: string
}

interface QueryKnowledgeResponse {
  results: Array<{
    id: string
    text: string
    score: number
    metadata?: Record<string, any>
    createdAt: string
  }>
}
```

```go
// Go
type QueryKnowledgeRequest struct {
    Collection string  `json:"collection" binding:"required"`
    Query      string  `json:"query" binding:"required"`
    Limit      int     `json:"limit,omitempty"`
    TaskID     *string `json:"taskId,omitempty"`
}

type KnowledgeResult struct {
    ID        string                 `json:"id"`
    Text      string                 `json:"text"`
    Score     float64                `json:"score"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt time.Time              `json:"createdAt"`
}

type QueryKnowledgeResponse struct {
    Results []KnowledgeResult `json:"results"`
}
```

#### UpsertKnowledgeRequest

```typescript
// TypeScript
interface UpsertKnowledgeRequest {
  collectionName: string
  information: string
  metadata?: Record<string, any>
  taskId?: string
}

interface UpsertKnowledgeResponse {
  id: string
  collectionId: string
  message: string
}
```

```go
// Go
type UpsertKnowledgeRequest struct {
    CollectionName string                 `json:"collectionName" binding:"required"`
    Information    string                 `json:"information" binding:"required"`
    Metadata       map[string]interface{} `json:"metadata,omitempty"`
    TaskID         *string                `json:"taskId,omitempty"`
}

type UpsertKnowledgeResponse struct {
    ID           string `json:"id"`
    CollectionID string `json:"collectionId"`
    Message      string `json:"message"`
}
```

---

## Code Search Contracts

### IndexedFolder

**TypeScript:**
```typescript
interface IndexedFolder {
  id: string
  path: string
  description?: string
  status: 'active' | 'scanning' | 'error'
  error?: string
  fileCount: number
  includePatterns?: string[]
  excludePatterns?: string[]
  chunkSize?: ChunkSize
  addedAt: string                     // ISO8601
  lastScanned?: string                // ISO8601
}
```

**Go:**
```go
type IndexedFolder struct {
    ID              string    `bson:"_id,omitempty" json:"id"`
    Path            string    `bson:"path" json:"path"`
    Description     string    `bson:"description,omitempty" json:"description"`
    Status          string    `bson:"status" json:"status"`
    Error           string    `bson:"error,omitempty" json:"error,omitempty"`
    FileCount       int       `bson:"fileCount" json:"fileCount"`
    IncludePatterns []string  `bson:"includePatterns,omitempty" json:"includePatterns"`
    ExcludePatterns []string  `bson:"excludePatterns,omitempty" json:"excludePatterns"`
    ChunkSize       string    `bson:"chunkSize,omitempty" json:"chunkSize"`
    AddedAt         time.Time `bson:"addedAt" json:"addedAt"`
    LastScanned     time.Time `bson:"lastScanned,omitempty" json:"lastScanned"`
}
```

### SearchResult

**TypeScript:**
```typescript
interface SearchResult {
  fileId: string
  filePath: string
  content: string
  lineStart: number
  lineEnd: number
  score: number
  language?: string
  // AST metadata (optional)
  chunkType?: 'ast' | 'line-based'
  nodeType?: 'function' | 'class' | 'method' | 'interface' | 'struct'
  nodeName?: string
  signature?: string
  symbols?: string[]
  hasDocstring?: boolean
  docContent?: string
}
```

**Go:**
```go
type SearchResult struct {
    FileID    string  `json:"fileId"`
    FilePath  string  `json:"filePath"`
    Content   string  `json:"content"`
    LineStart int     `json:"lineStart"`
    LineEnd   int     `json:"lineEnd"`
    Score     float64 `json:"score"`
    Language  string  `json:"language,omitempty"`
    // AST metadata
    ChunkType    string   `json:"chunkType,omitempty"`
    NodeType     string   `json:"nodeType,omitempty"`
    NodeName     string   `json:"nodeName,omitempty"`
    Signature    string   `json:"signature,omitempty"`
    Symbols      []string `json:"symbols,omitempty"`
    HasDocstring bool     `json:"hasDocstring,omitempty"`
    DocContent   string   `json:"docContent,omitempty"`
}
```

### API Request/Response Types

#### SearchCodeRequest

```typescript
// TypeScript
interface SearchCodeRequest {
  query: string
  folderPath?: string
  limit?: number                      // default: 10, max: 50
  minScore?: number                   // default: 0.0
  retrieve?: RetrieveMode             // default: 'chunk-m'
  fileTypes?: string[]                // e.g., ['.go', '.ts']
}

interface SearchCodeResponse {
  results: SearchResult[]
}
```

```go
// Go
type SearchCodeRequest struct {
    Query      string   `json:"query" binding:"required"`
    FolderPath string   `json:"folderPath,omitempty"`
    Limit      int      `json:"limit,omitempty"`
    MinScore   float64  `json:"minScore,omitempty"`
    Retrieve   string   `json:"retrieve,omitempty"`
    FileTypes  []string `json:"fileTypes,omitempty"`
}

type SearchCodeResponse struct {
    Results []SearchResult `json:"results"`
}
```

#### AddFolderRequest

```typescript
// TypeScript
interface AddFolderRequest {
  path: string
  description?: string
  includePatterns?: string[]
  excludePatterns?: string[]
  chunkSize?: ChunkSize
}

interface AddFolderResponse {
  folderId: string
  path: string
  status: string
  addedAt: string
}
```

```go
// Go
type AddFolderRequest struct {
    Path            string   `json:"path" binding:"required"`
    Description     string   `json:"description,omitempty"`
    IncludePatterns []string `json:"includePatterns,omitempty"`
    ExcludePatterns []string `json:"excludePatterns,omitempty"`
    ChunkSize       string   `json:"chunkSize,omitempty"`
}

type AddFolderResponse struct {
    FolderID string    `json:"folderId"`
    Path     string    `json:"path"`
    Status   string    `json:"status"`
    AddedAt  time.Time `json:"addedAt"`
}
```

---

## Chat Contracts

### Message

**TypeScript:**
```typescript
interface Message {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  timestamp: string                   // ISO8601
  metadata?: Record<string, any>
}
```

**Go:**
```go
type Message struct {
    ID        string                 `json:"id"`
    Role      string                 `json:"role"` // user|assistant|system
    Content   string                 `json:"content"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}
```

### ChatSession

**TypeScript:**
```typescript
interface ChatSession {
  sessionId: string
  title: string
  messages?: Message[]
  messageCount?: number
  createdAt: string                   // ISO8601
  updatedAt: string                   // ISO8601
}
```

**Go:**
```go
type ChatSession struct {
    SessionID    string    `json:"sessionId"`
    Title        string    `json:"title"`
    Messages     []Message `json:"messages,omitempty"`
    MessageCount int       `json:"messageCount,omitempty"`
    CreatedAt    time.Time `json:"createdAt"`
    UpdatedAt    time.Time `json:"updatedAt"`
}
```

### WebSocket Message Types

**TypeScript:**
```typescript
// Client → Server
interface UserMessage {
  type: 'user_message'
  sessionId: string
  content: string
}

// Server → Client
interface AssistantMessage {
  type: 'assistant_message'
  sessionId: string
  message: {
    id: string
    role: 'assistant'
    content: string
    timestamp: string
  }
}

interface StreamToken {
  type: 'stream_token'
  sessionId: string
  token: string
  done: boolean
}

interface ErrorMessage {
  type: 'error'
  error: string
}

type WebSocketMessage =
  | UserMessage
  | AssistantMessage
  | StreamToken
  | ErrorMessage
```

---

## Pagination Contracts

### PaginationParams

**TypeScript:**
```typescript
interface PaginationParams {
  limit?: number                      // default varies by endpoint
  offset?: number                     // default: 0
}

interface PaginatedResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}
```

**Go:**
```go
type PaginationParams struct {
    Limit  int `json:"limit,omitempty" form:"limit"`
    Offset int `json:"offset,omitempty" form:"offset"`
}

type PaginatedResponse struct {
    Data   interface{} `json:"data"`
    Total  int         `json:"total"`
    Limit  int         `json:"limit"`
    Offset int         `json:"offset"`
}
```

---

## Error Response Contract

**TypeScript:**
```typescript
interface ErrorResponse {
  error: string
  code?: string
  details?: Record<string, any>
}
```

**Go:**
```go
type ErrorResponse struct {
    Error   string                 `json:"error"`
    Code    string                 `json:"code,omitempty"`
    Details map[string]interface{} `json:"details,omitempty"`
}
```

---

## Validation Rules

### Field Constraints

| Field | Type | Constraints |
|-------|------|-------------|
| `prompt` | string | required, min: 1, max: 10000 |
| `query` | string | required, min: 1, max: 1000 |
| `limit` | number | min: 1, max: 50 (varies by endpoint) |
| `offset` | number | min: 0 |
| `minScore` | number | min: 0.0, max: 1.0 |
| `taskId` | string | UUID format |
| `vote` | string | enum: ['+', '-'] |

### Date Format

**All dates use ISO 8601 format:**
```
2025-11-20T10:30:45Z
2025-11-20T10:30:45.123Z
```

**Go serialization:**
```go
time.Now().UTC().Format(time.RFC3339)
```

**TypeScript parsing:**
```typescript
new Date(isoString).toISOString()
```

---

## Type Guards (TypeScript)

```typescript
// Type guard functions
export function isTaskStatus(value: string): value is TaskStatus {
  return ['pending', 'in_progress', 'completed', 'blocked'].includes(value)
}

export function isTodoStatus(value: string): value is TodoStatus {
  return ['pending', 'in_progress', 'completed'].includes(value)
}

export function isChunkSize(value: string): value is ChunkSize {
  return ['xs', 's', 'm', 'l', 'xl'].includes(value)
}

// Validation functions
export function validateHumanTask(data: unknown): data is HumanTask {
  const task = data as HumanTask
  return (
    typeof task.taskId === 'string' &&
    typeof task.prompt === 'string' &&
    isTaskStatus(task.status) &&
    typeof task.createdAt === 'string'
  )
}
```

---

## Related Documents

- [API Service Layer](./api-service-layer.md) - REST endpoints using these contracts
- [MongoDB Schemas](./mongodb-schemas.md) - Database schema mappings
- [UI Client Stack](./ui-client-stack.md) - TypeScript usage in frontend
- [Component Architecture](./component-architecture.md) - How contracts flow through system

## Version Compatibility

| Backend (Go) | Frontend (TypeScript) | Compatible |
|--------------|----------------------|------------|
| 1.0 | 1.0 | ✅ |

**Breaking Changes Policy:**
- Major version bump for breaking changes
- Backward compatible changes in minor versions
- Field additions are non-breaking (use omitempty)
- Field removals are breaking (deprecate first)
