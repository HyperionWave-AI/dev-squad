# Hyperion Data Architecture - MongoDB Collections & Schemas

**Collection:** data-architecture
**Tags:** MongoDB, data-model, collections, indexing
**Technology:** MongoDB 7.0
**Version:** 1.0

---

HYPERION DATA ARCHITECTURE - MONGODB COLLECTIONS & SCHEMAS

PRIMARY COLLECTIONS:

1. KNOWLEDGE MANAGEMENT:
knowledge_entries (KnowledgeEntry struct, storage/knowledge.go:46-55)
- Fields: entryId (unique, string), collectionId (ObjectID ref), collection (deprecated), taskId (sparse), text, metadata, createdAt
- Indexes: entryId (unique), collection (sorted), text (fulltext), taskId (sparse)
- Purpose: Stores all knowledge entries with semantic metadata
- Voting: Links to knowledge_votes collection for community rating

knowledge_votes (Vote struct, storage/knowledge.go:57-65)
- Fields: entryId, userId, vote (+/-), reason, createdAt, updatedAt
- Purpose: Track user votes for ranking knowledge quality
- Query pattern: Group by entryId to compute VoteSummary (upvotes, downvotes, net score)

collections (Collection struct, storage/knowledge.go:34-44)
- Fields: _id (ObjectID), name (unique), qdrantName, category, description, tags, entryCount, createdAt, updatedAt
- Purpose: Metadata for logical grouping of knowledge entries
- Indexes: name (unique)
- Categories: hyperion-architecture, frontend-patterns, backend-services, etc.

collection_metadata (CollectionMetadata struct, storage/knowledge.go:98-105)
- Fields: collectionName (unique), description, tags, category, createdAt, updatedAt
- Purpose: Enhanced metadata for collection organization
- Indexes: collectionName (unique)

2. TASK COORDINATION:
human_tasks (HumanTask struct, storage/tasks.go:67-77)
- Fields: taskId (unique), prompt, summary (AI-generated, ≤100 tokens), agentTaskIds[], status, notes, createdAt, updatedAt
- Statuses: pending, in_progress, completed, blocked
- Indexes: taskId (unique)
- Purpose: User-initiated work requests
- Bidirectional traceability to agent tasks

agent_tasks (AgentTask struct, storage/tasks.go:80-98)
- Fields: taskId (unique), humanTaskId, agentName, role, todos[], contextSummary, summary, filesModified[], qdrantCollections[], priorWorkSummary, status, notes, humanPromptNotes, timestamps
- TodoItem nested fields: id, description, status, createdAt, completedAt, notes, filePath, functionName, contextHint
- Indexes: taskId (unique), agentName (sorted), humanTaskId
- Purpose: Specialized tasks assigned to specific agents
- Contains: task breakdown, context, and progress tracking

3. REFLECTION & METACOGNITION:
reflection_entries (ReflectionEntry, storage/reflection_storage.go)
- Stores: decisions, outcomes, lessons learned
- Purpose: System self-awareness and pattern learning
- Indexed for semantic search via Qdrant

4. CODE INDEXING:
code_index_entries (CodeIndexEntry, storage/code_index_storage.go)
- Fields: fileId, filePath, startLine, endLine, language, category, semanticDescription, embedding
- Purpose: Map codebase structure and enable semantic search
- Storage: Both MongoDB (metadata) and Qdrant (vectors)

path_mappings (PathMapping, storage/code_index_storage.go)
- Maps: projectPath → qdrantCollection (for multi-project support)
- Purpose: Track which Qdrant collection contains code for each project

5. USER SETTINGS:
user_settings (UserSettings, storage/user_settings_storage.go)
- Fields: userId, conversationMode (debug|default), theme, preferences, createdAt, updatedAt
- Purpose: Persist user preferences across sessions

INDEXING STRATEGY:
- Unique indexes on identity fields (taskId, entryId, name)
- Sparse indexes on optional lookup fields (taskId in knowledge_entries)
- Text indexes for fulltext search
- All created with idempotent names to handle retries

QUERY PATTERNS:
- Task filtering: {humanTaskId, status, agentName}
- Knowledge search: Qdrant semantic + MongoDB fulltext
- Pagination: ListAgentTasks(offset, limit)
- Vote aggregation: Group knowledge_votes by entryId

SECURITY:
- JWT identity via MongoDB client (database.NewSecureMongoClient)
- No system service accounts - user identity throughout
- BSON tags for CRUD operations
