# Duplicate Task Detection - Feature Demo

## Overview

The `coordinator_create_human_task` MCP tool now automatically checks for similar existing tasks before creating a new one using semantic similarity search.

## How It Works

1. **Before Creation**: When you call `coordinator_create_human_task`, the system:
   - Searches the knowledge base for similar task prompts using vector embeddings
   - Filters results by similarity score (default: 0.75 threshold)
   - Returns similar tasks if found

2. **Automatic Indexing**: Every human task is automatically indexed in the knowledge base collection `human_tasks_search` for future similarity searches.

3. **Semantic Matching**: Uses Qdrant vector database with embedding models (Ollama, Voyage, OpenAI) to find semantically similar tasks.

## Usage Examples

### Example 1: Creating a task when no similar tasks exist

```json
{
  "tool": "coordinator_create_human_task",
  "input": {
    "prompt": "Implement user authentication with JWT tokens"
  }
}
```

**Response** (no duplicates found):
```json
{
  "similarTasksFound": false,
  "taskId": "abc-123-def",
  "status": "pending",
  "prompt": "Implement user authentication with JWT tokens",
  "createdAt": "2025-10-20T08:00:00Z"
}
```

### Example 2: Attempting to create a similar task

```json
{
  "tool": "coordinator_create_human_task",
  "input": {
    "prompt": "Add JWT authentication to the application"
  }
}
```

**Response** (similar task found):
```json
{
  "similarTasksFound": true,
  "similarTasks": [
    {
      "taskId": "abc-123-def",
      "prompt": "Implement user authentication with JWT tokens",
      "status": "pending",
      "createdAt": "2025-10-20T08:00:00Z",
      "similarity": 0.89
    }
  ],
  "message": "Found 1 similar task(s). Set forceCreate=true to create anyway, or use an existing task."
}
```

### Example 3: Force creating despite similar tasks

```json
{
  "tool": "coordinator_create_human_task",
  "input": {
    "prompt": "Add JWT authentication to the application",
    "forceCreate": true
  }
}
```

**Response** (creates new task):
```json
{
  "similarTasksFound": false,
  "taskId": "xyz-789-abc",
  "status": "pending",
  "prompt": "Add JWT authentication to the application",
  "createdAt": "2025-10-20T08:05:00Z"
}
```

## Configuration

### Similarity Threshold

The similarity threshold is hardcoded to `0.75` (75% similarity). Tasks with similarity scores below this threshold are not returned.

### Search Limit

The system searches for up to 5 similar tasks by default.

### Embedding Models

Similarity detection works with any configured embedding model:
- **Ollama** (default): `nomic-embed-text` (768 dimensions)
- **Voyage AI**: `voyage-3` (1024 dimensions)
- **OpenAI**: `text-embedding-3-small` (1536 dimensions)

## Benefits

1. **Prevents Duplicate Work**: Alerts coordinators when similar tasks already exist
2. **Improves Task Management**: Encourages reuse of existing tasks
3. **Semantic Understanding**: Finds similar tasks even with different wording
4. **Flexible Override**: Can force creation when intentional duplicates are needed

## Implementation Details

### Modified Files

1. **`hyper/internal/mcp/storage/tasks.go`**:
   - Added `SearchSimilarHumanTasks` method to TaskStorage interface
   - Updated `MongoTaskStorage` to include `KnowledgeStorage` dependency
   - Modified `CreateHumanTask` to index tasks in knowledge base
   - Implemented semantic similarity search

2. **`hyper/internal/ai-service/tools/mcp/coordinator_tools.go`**:
   - Updated `CreateHumanTaskTool` input schema to include `forceCreate` parameter
   - Modified `Execute` method to check for similar tasks before creation
   - Returns formatted similar tasks with similarity scores

3. **`hyper/cmd/coordinator/main.go`**:
   - Updated initialization order to create KnowledgeStorage before TaskStorage
   - Passed KnowledgeStorage to TaskStorage constructor

### Database Schema

Tasks are indexed in Qdrant collection: `human_tasks_search`

**Metadata stored**:
```json
{
  "taskId": "abc-123-def",
  "status": "pending",
  "createdAt": "2025-10-20T08:00:00Z"
}
```

## Testing

Run tests with:
```bash
go test ./internal/mcp/storage -v -run TestSearchSimilarHumanTasks
```

Integration tests require `MONGODB_TEST_URL` environment variable.

## Future Enhancements

Potential improvements:
1. Configurable similarity threshold via environment variable
2. UI integration to show similar tasks before submission
3. Automatic task merging suggestions
4. Similarity analytics and reporting
