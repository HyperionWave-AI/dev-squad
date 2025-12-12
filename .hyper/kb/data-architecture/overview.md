# ## Overview

**Collection:** data-architecture
**Created:** 2025-11-20

---

## Overview

This document provides comprehensive schemas for all MongoDB collections in the Hyperion system, including field types, indexes, and query patterns.

---

## Knowledge Base Collections

### Collection: `collections`

**Purpose:** Knowledge collection metadata and organization

**Location:** `/hyper/internal/mcp/storage/knowledge.go` (Lines 33-44)

#### Schema

```go
type Collection struct {
    ID          primitive.ObjectID `json:"id" bson:"_id"`
    Name        string             `json:"name" bson:"name"`                   // Unique display name
    QdrantName  string             `json:"qdrantName" bson:"qdrantName"`       // Unique Qdrant identifier
    Category    string             `json:"category" bson:"category"`           // Grouping category
    Description string             `json:"description" bson:"description"`     // Human-readable description
    Tags        []string           `json:"tags" bson:"tags"`                   // Searchable tags
    EntryCount  int                `json:"entryCount" bson:"entryCount"`       // Cached entry count
    CreatedAt   time.Time          `json:"createdAt" bson:"createdAt"`
    UpdatedAt   time.Time          `json:"updatedAt" bson:"updatedAt"`
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key (auto-created) |
| `name_1` | `name: 1` | Unique | Enforce unique collection names |
| `qdrantName_1` | `qdrantName: 1` | Unique, Sparse | Map to Qdrant collections |
| `createdAt_-1` | `createdAt: -1` | Standard | Sort by creation date (newest first) |

#### Query Patterns

```go
// Find collection by name
filter := bson.M{"name": "technical-knowledge"}

// List all collections sorted by creation
opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})

// Find collections by category
filter := bson.M{"category": "architecture"}
```

---

### Collection: `knowledge_entries`

**Purpose:** Individual knowledge entries with content and metadata

**Location:** `/hyper/internal/mcp/storage/knowledge.go` (Lines 46-55)

#### Schema

```go
type KnowledgeEntry struct {
    ID           string                 `json:"id" bson:"entryId"`                        // Unique entry ID
    CollectionID primitive.ObjectID     `json:"collectionId,omitempty" bson:"collectionId,omitempty"` // FK to collections
    Collection   string                 `json:"collection" bson:"collection"`             // DEPRECATED: Legacy field
    TaskId       string                 `json:"taskId,omitempty" bson:"taskId,omitempty"` // Optional task scope
    Text         string                 `json:"text" bson:"text"`                         // Entry content
    Metadata     map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"` // Custom metadata
    CreatedAt    time.Time              `json:"createdAt" bson:"createdAt"`
}
```

#### Metadata Fields (Common)

```json
{
  "sourceFile": "string",
  "tags": ["array", "of", "tags"],
  "version": "string",
  "author": "string",
  "syncedAt": "ISO8601"
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `entryId_1` | `entryId: 1` | Unique | Enforce unique entry IDs |
| `collectionId_1` | `collectionId: 1` | Standard | Filter by collection |
| `collection_1` | `collection: 1` | Standard | DEPRECATED: Legacy queries |
| `taskId_1` | `taskId: 1` | Sparse | Task-scoped queries |
| `text_text` | `text: text` | Text | Full-text search |

#### Query Patterns

```go
// Find entry by ID
filter := bson.M{"entryId": entryID}

// List entries in collection
filter := bson.M{"collectionId": collectionID}
opts := options.Find().SetLimit(20).SetSkip(offset)

// Task-scoped query
filter := bson.M{
    "collectionId": collectionID,
    "taskId":       taskID,
}

// Full-text search
filter := bson.M{"$text": bson.M{"$search": "authentication patterns"}}
```

---

### Collection: `knowledge_votes`

**Purpose:** User votes on knowledge entries for quality ranking

**Location:** `/hyper/internal/mcp/storage/knowledge.go` (Lines 57-65)

#### Schema

```go
type Vote struct {
    EntryID   string    `json:"entryId" bson:"entryId"`   // FK to knowledge_entries
    UserID    string    `json:"userId" bson:"userId"`     // User identifier from JWT
    Vote      string    `json:"vote" bson:"vote"`         // "+" for upvote, "-" for downvote
    Reason    string    `json:"reason" bson:"reason"`     // Required explanation
    CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
    UpdatedAt time.Time `json:"updatedAt" bson:"updatedAt"`
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `entryId_1_userId_1` | `entryId: 1, userId: 1` | Compound, Unique | One vote per user per entry |
| `entryId_1` | `entryId: 1` | Standard | Vote aggregation queries |

#### Query Patterns

```go
// Get vote statistics for entry
pipeline := []bson.M{
    {"$match": bson.M{"entryId": entryID}},
    {"$group": bson.M{
        "_id":       nil,
        "upvotes":   bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []string{"$vote", "+"}}, 1, 0}}},
        "downvotes": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$eq": []string{"$vote", "-"}}, 1, 0}}},
    }},
}

// Upsert user vote (change vote if already exists)
filter := bson.M{"entryId": entryID, "userId": userID}
update := bson.M{
    "$set": bson.M{
        "vote":      "+",
        "reason":    "Helpful pattern for authentication",
        "updatedAt": time.Now(),
    },
    "$setOnInsert": bson.M{"createdAt": time.Now()},
}
opts := options.Update().SetUpsert(true)
```

---

## Task Management Collections

### Collection: `human_tasks`

**Purpose:** User-created tasks and prompts

**Location:** `/hyper/internal/mcp/storage/tasks.go` (Lines 67-77)

#### Schema

```go
type HumanTask struct {
    ID           string     `json:"taskId" bson:"taskId"`                         // UUID
    Prompt       string     `json:"prompt" bson:"prompt"`                         // Original user request
    Summary      string     `json:"summary,omitempty" bson:"summary,omitempty"`   // AI-generated summary
    AgentTaskIDs []string   `json:"agentTaskIds,omitempty" bson:"agentTaskIds,omitempty"` // Child tasks
    CreatedAt    time.Time  `json:"createdAt" bson:"createdAt"`
    UpdatedAt    time.Time  `json:"updatedAt" bson:"updatedAt"`
    Status       TaskStatus `json:"status" bson:"status"`                         // pending|in_progress|completed|blocked
    Notes        string     `json:"notes,omitempty" bson:"notes,omitempty"`       // Progress notes
}
```

#### TaskStatus Values

```go
const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusCompleted  TaskStatus = "completed"
    TaskStatusBlocked    TaskStatus = "blocked"
)
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `taskId_1` | `taskId: 1` | Unique | Query by task ID |
| `createdAt_-1` | `createdAt: -1` | Standard | Sort by creation (newest first) |
| `status_1` | `status: 1` | Standard | Filter by status |

#### Query Patterns

```go
// List recent tasks
opts := options.Find().
    SetSort(bson.D{{Key: "createdAt", Value: -1}}).
    SetLimit(50)

// Find pending tasks
filter := bson.M{"status": "pending"}

// Update task status with notes
update := bson.M{
    "$set": bson.M{
        "status":    "completed",
        "notes":     "All agent tasks completed successfully",
        "updatedAt": time.Now(),
    },
}
```

---

### Collection: `agent_tasks`

**Purpose:** AI agent task assignments with TODOs and context

**Location:** `/hyper/internal/mcp/storage/tasks.go` (Lines 79-98)

#### Schema

```go
type AgentTask struct {
    ID                        string     `json:"taskId" bson:"taskId"`                     // UUID
    HumanTaskID               string     `json:"humanTaskId" bson:"humanTaskId"`           // FK to human_tasks
    AgentName                 string     `json:"agentName" bson:"agentName"`               // Agent identifier
    Role                      string     `json:"role" bson:"role"`                         // Agent's mission
    Todos                     []TodoItem `json:"todos" bson:"todos"`                       // Task breakdown
    CreatedAt                 time.Time  `json:"createdAt" bson:"createdAt"`
    UpdatedAt                 time.Time  `json:"updatedAt" bson:"updatedAt"`
    Status                    TaskStatus `json:"status" bson:"status"`
    Notes                     string     `json:"notes,omitempty" bson:"notes,omitempty"`
    ContextSummary            string     `json:"contextSummary,omitempty" bson:"contextSummary,omitempty"`
    FilesModified             []string   `json:"filesModified,omitempty" bson:"filesModified,omitempty"`
    QdrantCollections         []string   `json:"qdrantCollections,omitempty" bson:"qdrantCollections,omitempty"`
    PriorWorkSummary          string     `json:"priorWorkSummary,omitempty" bson:"priorWorkSummary,omitempty"`
    HumanPromptNotes          string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
    HumanPromptNotesAddedAt   *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
    HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
}
```

#### Embedded: TodoItem

```go
type TodoItem struct {
    ID                        string     `json:"id" bson:"id"`                             // UUID
    Description               string     `json:"description" bson:"description"`
    Status                    TodoStatus `json:"status" bson:"status"`                     // pending|in_progress|completed
    CreatedAt                 time.Time  `json:"createdAt" bson:"createdAt"`
    CompletedAt               *time.Time `json:"completedAt,omitempty" bson:"completedAt,omitempty"`
    Notes                     string     `json:"notes,omitempty" bson:"notes,omitempty"`
    FilePath                  string     `json:"filePath,omitempty" bson:"filePath,omitempty"`
    FunctionName              string     `json:"functionName,omitempty" bson:"functionName,omitempty"`
    ContextHint               string     `json:"contextHint,omitempty" bson:"contextHint,omitempty"`
    HumanPromptNotes          string     `json:"humanPromptNotes,omitempty" bson:"humanPromptNotes,omitempty"`
    HumanPromptNotesAddedAt   *time.Time `json:"humanPromptNotesAddedAt,omitempty" bson:"humanPromptNotesAddedAt,omitempty"`
    HumanPromptNotesUpdatedAt *time.Time `json:"humanPromptNotesUpdatedAt,omitempty" bson:"humanPromptNotesUpdatedAt,omitempty"`
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `taskId_1` | `taskId: 1` | Unique | Query by task ID |
| `humanTaskId_1` | `humanTaskId: 1` | Standard | Find tasks for human task |
| `agentName_1` | `agentName: 1` | Standard | Filter by agent |
| `status_1` | `status: 1` | Standard | Filter by status |

#### Query Patterns

```go
// Find tasks by agent
filter := bson.M{"agentName": "go-dev"}

// Find tasks for human task
filter := bson.M{"humanTaskId": humanTaskID}

// Update TODO status
filter := bson.M{"taskId": taskID}
update := bson.M{
    "$set": bson.M{
        "todos.$[elem].status":      "completed",
        "todos.$[elem].completedAt": time.Now(),
        "todos.$[elem].notes":       "Implementation complete at line 345",
        "updatedAt":                 time.Now(),
    },
}
arrayFilters := options.Update().SetArrayFilters(options.ArrayFilters{
    Filters: []interface{}{
        bson.M{"elem.id": todoID},
    },
})

// Auto-complete task when all TODOs done
pipeline := []bson.M{
    {"$match": bson.M{"taskId": taskID}},
    {"$project": bson.M{
        "allComplete": bson.M{
            "$allElementsTrue": []interface{}{
                bson.M{"$map": bson.M{
                    "input": "$todos",
                    "as":    "todo",
                    "in":    bson.M{"$eq": []interface{}{"$$todo.status", "completed"}},
                }},
            },
        },
    }},
}
```

---

## Code Index Collections

### Collection: `indexed_folders`

**Purpose:** Folder index configuration and status

**Location:** `/hyper/internal/mcp/storage/code_index_models.go` (Lines 7-20)

#### Schema

```go
type IndexedFolder struct {
    ID              string    `bson:"_id,omitempty" json:"id"`                  // UUID or path-based
    Path            string    `bson:"path" json:"path"`                         // Absolute path
    Description     string    `bson:"description,omitempty" json:"description"`
    AddedAt         time.Time `bson:"addedAt" json:"addedAt"`
    LastScanned     time.Time `bson:"lastScanned,omitempty" json:"lastScanned"`
    FileCount       int       `bson:"fileCount" json:"fileCount"`
    Status          string    `bson:"status" json:"status"`                     // active|scanning|error
    Error           string    `bson:"error,omitempty" json:"error,omitempty"`
    IncludePatterns []string  `bson:"includePatterns,omitempty" json:"includePatterns"` // e.g., ["*.go", "*.ts"]
    ExcludePatterns []string  `bson:"excludePatterns,omitempty" json:"excludePatterns"` // e.g., ["node_modules", ".git"]
    ChunkSize       string    `bson:"chunkSize,omitempty" json:"chunkSize"`     // xs|s|m|l|xl
}
```

#### Status Values

- `active` - Folder indexed and ready
- `scanning` - Indexing in progress
- `error` - Indexing failed

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `path_1` | `path: 1` | Unique | Prevent duplicate folders |
| `status_1` | `status: 1` | Standard | Filter by status |

---

### Collection: `indexed_files`

**Purpose:** File metadata for indexed code files

**Location:** `/hyper/internal/mcp/storage/code_index_models.go` (Lines 22-36)

#### Schema

```go
type IndexedFile struct {
    ID           string    `bson:"_id,omitempty" json:"id"`                  // UUID
    FolderID     string    `bson:"folderId" json:"folderId"`                 // FK to indexed_folders
    Path         string    `bson:"path" json:"path"`                         // Absolute path
    RelativePath string    `bson:"relativePath" json:"relativePath"`         // Path relative to folder
    Language     string    `bson:"language" json:"language"`                 // go|typescript|javascript|python|etc
    SHA256       string    `bson:"sha256" json:"sha256"`                     // Content hash for change detection
    Size         int64     `bson:"size" json:"size"`                         // File size in bytes
    LineCount    int       `bson:"lineCount" json:"lineCount"`
    IndexedAt    time.Time `bson:"indexedAt" json:"indexedAt"`
    UpdatedAt    time.Time `bson:"updatedAt" json:"updatedAt"`
    VectorID     string    `bson:"vectorId,omitempty" json:"vectorId,omitempty"` // Qdrant point ID
    ChunkCount   int       `bson:"chunkCount" json:"chunkCount"`
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `path_1` | `path: 1` | Unique | Prevent duplicates |
| `folderId_1` | `folderId: 1` | Standard | Query files in folder |
| `sha256_1` | `sha256: 1` | Standard | Change detection |
| `language_1` | `language: 1` | Standard | Filter by language |

---

### Collection: `file_chunks`

**Purpose:** Code chunks with AST metadata for semantic search

**Location:** `/hyper/internal/mcp/storage/code_index_models.go` (Lines 38-60)

#### Schema

```go
type FileChunk struct {
    ID        string    `bson:"_id,omitempty" json:"id"`           // UUID
    FileID    string    `bson:"fileId" json:"fileId"`              // FK to indexed_files
    ChunkNum  int       `bson:"chunkNum" json:"chunkNum"`          // 0-based chunk index
    Content   string    `bson:"content" json:"content"`            // Chunk text content
    StartLine int       `bson:"startLine" json:"startLine"`        // Line number in file
    EndLine   int       `bson:"endLine" json:"endLine"`
    VectorID  string    `bson:"vectorId,omitempty" json:"vectorId"` // Qdrant point ID
    IndexedAt time.Time `bson:"indexedAt" json:"indexedAt"`

    // AST-based chunking metadata
    ChunkType    string   `bson:"chunkType,omitempty" json:"chunkType,omitempty"`       // "ast" or "line-based"
    NodeType     string   `bson:"nodeType,omitempty" json:"nodeType,omitempty"`         // function|class|method|interface
    NodeName     string   `bson:"nodeName,omitempty" json:"nodeName,omitempty"`         // Name of function/class
    Signature    string   `bson:"signature,omitempty" json:"signature,omitempty"`       // Function signature
    Symbols      []string `bson:"symbols,omitempty" json:"symbols,omitempty"`           // Imported/referenced symbols
    Imports      []string `bson:"imports,omitempty" json:"imports,omitempty"`           // Import statements
    HasDocstring bool     `bson:"hasDocstring,omitempty" json:"hasDocstring,omitempty"` // Has documentation
    DocContent   string   `bson:"docContent,omitempty" json:"docContent,omitempty"`     // Doc comment text
}
```

#### NodeType Values

- `function` - Function declaration
- `class` - Class definition
- `method` - Class method
- `interface` - Interface/type definition
- `struct` - Go struct definition

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `fileId_1_chunkNum_1` | `fileId: 1, chunkNum: 1` | Compound, Unique | Sequential chunk access |
| `nodeType_1` | `nodeType: 1` | Standard | Filter by AST node type |
| `vectorId_1` | `vectorId: 1` | Standard | Map to Qdrant points |

---

## Reflection (Metacognitive) Collections

### Collection: `reflections`

**Purpose:** Decision and outcome tracking for AI learning

**Location:** `/hyper/internal/mcp/storage/reflection_storage.go` (Lines 15-26)

#### Schema

```go
type Reflection struct {
    ID                 string                 `json:"id" bson:"_id"`                    // UUID
    Type               string                 `json:"type" bson:"type"`                 // "decision"|"outcome"|"lesson"|"causal_link"
    ChatID             string                 `json:"chatId" bson:"chatId"`             // Chat session ID
    TaskID             string                 `json:"taskId,omitempty" bson:"taskId,omitempty"` // Optional task scope
    Timestamp          time.Time              `json:"timestamp" bson:"timestamp"`
    Data               map[string]interface{} `json:"data" bson:"data"`                 // Type-specific data
    Confidence         float64                `json:"confidence,omitempty" bson:"confidence,omitempty"` // 0.0-1.0
    Tags               []string               `json:"tags" bson:"tags"`
    RelatedReflections []string               `json:"relatedReflections,omitempty" bson:"relatedReflections,omitempty"` // IDs
}
```

#### Type Values and Data Structures

**Type: "decision"**
```json
{
  "data": {
    "context": {
      "userRequest": "string",
      "availableInfo": "string",
      "uncertainty": "string"
    },
    "decision": {
      "action": "string",
      "reasoning": "string",
      "alternatives": ["array"],
      "confidence": 0.85
    },
    "predictions": {
      "successProbability": 0.9,
      "timeEstimate": "string",
      "risks": ["array"]
    }
  }
}
```

**Type: "outcome"**
```json
{
  "data": {
    "decisionId": "reflection-uuid",
    "outcome": {
      "success": true,
      "actualResult": "string",
      "userFeedback": "string",
      "rootCause": "string"
    },
    "analysis": {
      "predictionAccuracy": 0.8,
      "confidenceCalibration": "overconfident|underconfident|well-calibrated",
      "missedSignals": ["array"]
    }
  }
}
```

**Type: "lesson"**
```json
{
  "data": {
    "patternName": "string",
    "problem": "string",
    "solution": "string",
    "context": "string",
    "antipattern": "string",
    "applicableTo": ["situations"],
    "confidence": 0.9
  }
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `type_1` | `type: 1` | Standard | Filter by reflection type |
| `chatId_1` | `chatId: 1` | Standard | Chat-scoped queries |
| `taskId_1` | `taskId: 1` | Sparse | Task-scoped queries |
| `timestamp_-1` | `timestamp: -1` | Standard | Sort by time (newest first) |

---

### Collection: `error_patterns`

**Purpose:** Recurring error pattern detection for lesson extraction

**Location:** `/hyper/internal/mcp/storage/reflection_storage.go` (Lines 47-59)

#### Schema

```go
type ErrorPattern struct {
    ID              string          `json:"id" bson:"_id"`                    // UUID
    Signature       string          `json:"signature" bson:"signature"`       // Hash of error type + normalized message
    ErrorType       string          `json:"errorType" bson:"errorType"`       // TypeError|RuntimeError|etc
    MessagePattern  string          `json:"messagePattern" bson:"messagePattern"` // Normalized error message
    Occurrences     int             `json:"occurrences" bson:"occurrences"`   // Count of occurrences
    FirstSeen       time.Time       `json:"firstSeen" bson:"firstSeen"`
    LastSeen        time.Time       `json:"lastSeen" bson:"lastSeen"`
    RecentErrors    []ErrorInstance `json:"recentErrors" bson:"recentErrors"` // Keep last 5
    LessonExtracted bool            `json:"lessonExtracted" bson:"lessonExtracted"`
    RelatedLesson   string          `json:"relatedLesson,omitempty" bson:"relatedLesson,omitempty"` // Reflection ID
}

type ErrorInstance struct {
    Timestamp time.Time `json:"timestamp" bson:"timestamp"`
    Message   string    `json:"message" bson:"message"`
    Context   string    `json:"context" bson:"context"`
}
```

#### Indexes

| Index Name | Keys | Type | Purpose |
|------------|------|------|---------|
| `_id_` | `_id: 1` | Unique | Primary key |
| `signature_1` | `signature: 1` | Unique | Deduplication |
| `occurrences_-1` | `occurrences: -1` | Standard | Find frequent errors |
| `lessonExtracted_1` | `lessonExtracted: 1` | Standard | Find patterns needing lessons |

---

## Related Documents

- [MongoDB Integration](./mongodb-integration.md) - Connection and security
- [Qdrant Integration](./qdrant-integration.md) - Vector database usage
- [Data Contracts](./data-contracts.md) - API request/response schemas
- [Configuration Reference](./configuration-reference.md) - Environment variables
