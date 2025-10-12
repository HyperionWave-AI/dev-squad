# Hyperion Coordinator MCP Server - Developer Guide

## Overview

This MCP server provides task coordination and knowledge management for the Hyperion Parallel Squad System. It integrates with MongoDB for task storage and Qdrant for vector-based semantic search.

## Architecture

### Storage Layers
- **MongoDB**: Task and knowledge entry storage with text indexes
- **Qdrant**: Vector embeddings for semantic similarity search
- **Embedding Services**: Supports local (TEI/Ollama), OpenAI, and Voyage AI

### Key Components
- **Task Management**: Human tasks, agent tasks, TODOs with status tracking
- **Knowledge Base**: Collection-based knowledge storage with semantic search
- **MCP Resources**: URI-based access to tasks and knowledge
- **MCP Tools**: 17 coordinator tools + 2 Qdrant tools + 5 code indexing tools
- **Health Checks**: Service availability monitoring

## Common Errors & Recovery

### 1. Ollama Embedding Service Down

**Symptom:**
```
Error: dial tcp [::1]:11434: connect: connection refused
Error: Qdrant embedding service unavailable
```

**Diagnosis:**
- Check if Ollama is running: `ollama list`
- Verify Ollama URL: `echo $OLLAMA_URL` (default: http://localhost:11434)
- Check model availability: `ollama list | grep nomic-embed-text`

**Recovery Steps:**

1. **Start Ollama (if not running):**
   ```bash
   # Ollama runs as a service, check status:
   ps aux | grep ollama

   # If not running, start it:
   ollama serve
   ```

2. **Pull the required model (if missing):**
   ```bash
   ollama pull nomic-embed-text
   ```

3. **Test Ollama connectivity:**
   ```bash
   curl http://localhost:11434/api/tags
   # Should return: {"models": [...]}
   ```

4. **Use health check endpoint:**
   ```bash
   curl http://localhost:7778/health/ollama
   # Returns: {"status": "up", "models": ["nomic-embed-text:latest"], ...}
   ```

**Fallback Options:**
- Use `coordinator_query_knowledge` instead of `knowledge_find` for task-specific knowledge
- Switch to TEI embedding service (set `EMBEDDING=local`)
- Use OpenAI embeddings (set `EMBEDDING=openai` with `OPENAI_API_KEY`)

---

### 2. Qdrant Service Timeout/Unavailable

**Symptom:**
```
Error: Qdrant search unavailable
Error: context deadline exceeded
```

**Diagnosis:**
- Check Qdrant URL: `echo $QDRANT_URL` (default: http://qdrant:6333)
- Test connectivity: `curl http://qdrant:6333/collections`
- Check health: `curl http://localhost:7778/health/qdrant`

**Recovery Steps:**

1. **Verify Qdrant is running:**
   ```bash
   docker ps | grep qdrant
   # Or for native: ps aux | grep qdrant
   ```

2. **Check Qdrant logs:**
   ```bash
   docker logs qdrant  # If running in Docker
   ```

3. **Restart Qdrant (if needed):**
   ```bash
   docker restart qdrant
   ```

4. **Test collection access:**
   ```bash
   curl http://qdrant:6333/collections/dev_squad_knowledge
   ```

**Fallback Options:**
- Use `coordinator_query_knowledge` for task-specific searches (uses MongoDB text search)
- Use `coordinator_upsert_knowledge` for storing knowledge (MongoDB only, no vector embeddings)
- Reduces to keyword-based search instead of semantic similarity

---

### 3. Null Returns from coordinator_get_popular_collections

**Symptom:**
```json
{"collections": null}
```

**Root Cause:**
- Empty knowledge base (no entries stored yet)
- Database connection issue

**Expected Behavior (Fixed):**
```json
{
  "collections": [],
  "message": "No collections with entries yet. Check hyperion://knowledge/collections resource for available collections.",
  "totalDefined": 14
}
```

**Recovery:**
- This is now a non-error: Empty array indicates knowledge base needs initial data
- Use `coordinator_upsert_knowledge` to add entries
- Check predefined collections via `hyperion://knowledge/collections` resource

**Prevention:**
- Never return null from Go functions - always initialize slices with `make([]Type, 0)`
- Provide descriptive messages in empty responses

---

### 4. MongoDB Connection Failures

**Symptom:**
```
Error: Failed to connect to MongoDB
Error: server selection error
```

**Diagnosis:**
- Check MongoDB URI: `echo $MONGODB_URI`
- Verify network connectivity
- Check MongoDB Atlas status

**Recovery Steps:**

1. **Verify MongoDB URI is set:**
   ```bash
   echo $MONGODB_URI
   # Should not be empty
   ```

2. **Test MongoDB connection:**
   ```bash
   mongosh "$MONGODB_URI" --eval "db.adminCommand('ping')"
   ```

3. **Check MongoDB Atlas dashboard:**
   - Verify cluster is running
   - Check IP whitelist (add 0.0.0.0/0 for dev)
   - Verify credentials

4. **Check health endpoint:**
   ```bash
   curl http://localhost:7778/health
   # Returns: {"status": "healthy", "services": {"mongodb": {"status": "up", ...}}}
   ```

**No Fallback:**
- MongoDB is required for task storage
- Service will fail to start without MongoDB connection
- This is **intentional** - fail-fast is better than silent fallback

---

### 5. Environment Variable Issues

**Symptom:**
```
Warning: Could not load .env.hyper
Error: MONGODB_URI environment variable is required
```

**Diagnosis:**
- Check if `.env.hyper` exists in project root
- Verify environment variables are exported

**Recovery Steps:**

1. **Check .env.hyper file:**
   ```bash
   ls -la /Users/maxmednikov/MaxSpace/dev-squad/.env.hyper
   ```

2. **Verify key variables:**
   ```bash
   source .env.hyper
   echo $MONGODB_URI
   echo $EMBEDDING
   echo $OLLAMA_URL
   echo $QDRANT_URL
   ```

3. **Use run script (recommended):**
   ```bash
   ./run-native.sh
   # Automatically sources .env.hyper
   ```

**Configuration Priority:**
1. Explicit environment variables (highest priority)
2. `.env.hyper` file (auto-loaded by mcp-server)
3. Hardcoded defaults (lowest priority)

---

### 6. Health Check Endpoints

**Pre-Flight Checks (Before Using MCP Tools):**

```bash
# Check all services
curl http://localhost:7778/health
# Returns overall status: healthy/unhealthy

# Check Qdrant specifically
curl http://localhost:7778/health/qdrant
# Returns: {"status": "up", "service": "qdrant", "url": "http://qdrant:6333", "responseMs": 5}

# Check Ollama specifically
curl http://localhost:7778/health/ollama
# Returns: {"status": "up", "service": "ollama", "url": "http://localhost:11434",
#           "models": ["nomic-embed-text:latest"], "responseMs": 12}
```

**Response Codes:**
- `200 OK`: Service is up and responding
- `503 Service Unavailable`: Service is down or unreachable

**Use Cases:**
- Run health checks before starting work sessions
- Diagnose embedding/search issues quickly
- Monitor service availability in scripts

---

## Error Handling Philosophy

### Fail-Fast Principle
- **NEVER create silent fallbacks** that hide configuration errors
- **ALWAYS return descriptive errors** with recovery suggestions
- **MongoDB failures = service failure** (no fallback)
- **Qdrant/Ollama failures = suggest coordinator knowledge fallback**

### Error Message Format
All error messages follow this pattern:
```
{Primary Error Message}. {Recovery Suggestion}. Original error: {technical details}
```

Example:
```
Qdrant embedding service unavailable. For task-specific knowledge, use coordinator_query_knowledge with task URI. Original error: dial tcp: connection refused
```

### When to Use Fallbacks
- ✅ **Qdrant unavailable** → Suggest `coordinator_query_knowledge` (MongoDB text search)
- ✅ **Empty results** → Return empty array with helpful message
- ❌ **MongoDB unavailable** → Fail immediately (no fallback)
- ❌ **Configuration errors** → Fail immediately with clear message

---

## Troubleshooting Checklist

### Before Starting Development

- [ ] Run health checks: `curl http://localhost:7778/health`
- [ ] Verify Ollama: `curl http://localhost:7778/health/ollama`
- [ ] Verify Qdrant: `curl http://localhost:7778/health/qdrant`
- [ ] Check MongoDB: `curl http://localhost:7778/health | jq .services.mongodb`
- [ ] Verify environment: `source .env.hyper && env | grep -E "(MONGODB|QDRANT|OLLAMA|EMBEDDING)"`

### When Errors Occur

1. **Read the error message completely** - it includes recovery steps
2. **Check health endpoints** for affected service
3. **Follow recovery steps** in error message
4. **Use fallback options** (coordinator knowledge) if Qdrant/Ollama down
5. **Report persistent issues** - don't work around silently

### Common Mistakes to Avoid

- ❌ Ignoring "connection refused" errors (service is down!)
- ❌ Using qdrant tools when Ollama is down (use coordinator instead)
- ❌ Expecting null returns to be errors (empty arrays are valid)
- ❌ Skipping health checks before long work sessions
- ✅ Reading error messages fully (recovery steps are embedded)
- ✅ Using health endpoints to diagnose issues quickly
- ✅ Switching to coordinator knowledge when Qdrant unavailable

---

## Quick Reference

### Service URLs (Default)
- MCP Server: `http://localhost:7778/mcp`
- Health Check: `http://localhost:7778/health`
- Qdrant Health: `http://localhost:7778/health/qdrant`
- Ollama Health: `http://localhost:7778/health/ollama`
- MongoDB: `mongodb+srv://dev:***@devdb.yqf8f8r.mongodb.net`
- Qdrant: `http://qdrant:6333`
- Ollama: `http://localhost:11434`

### Critical Environment Variables
- `MONGODB_URI`: MongoDB connection string (REQUIRED)
- `EMBEDDING`: "ollama"|"local"|"openai"|"voyage" (default: "local")
- `OLLAMA_URL`: Ollama service URL (default: "http://localhost:11434")
- `OLLAMA_MODEL`: Ollama embedding model (default: "nomic-embed-text")
- `QDRANT_URL`: Qdrant service URL (default: "http://qdrant:6333")

### MCP Tools Summary
- **Task Management**: 10 tools (create, list, update, status, notes)
- **Knowledge Base**: 3 tools (upsert, query, get_popular_collections)
- **Qdrant Direct**: 2 tools (knowledge_find, knowledge_store)
- **Code Indexing**: 5 tools (add folder, scan, search, status, remove)

---

## See Also

- `README.md` - Installation and usage guide
- `INSTALLATION.md` - Detailed setup instructions
- `DOCKER_FILE_WATCHER.md` - Docker volume mapping for code indexing
- `TEST_RESULTS.md` - Test coverage and validation
