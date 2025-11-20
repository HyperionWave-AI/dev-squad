# Hyperion System Architecture Overview

**Collection:** hyperion-architecture
**Tags:** architecture, system-design, microservices
**Version:** 1.0

---

HYPERION SYSTEM ARCHITECTURE OVERVIEW

Hyperion is an AI-powered development coordination platform with a distributed microservices architecture designed for intelligent task orchestration, code intelligence, and real-time collaboration.

CORE COMPONENTS:
1. Unified Coordinator Service (Go 1.25): Central orchestrator managing all system operations
   - Modes: HTTP server (port 8080), MCP server (stdio), or both
   - Entry point: /hyper/cmd/coordinator/main.go
   - Configuration: .env.hyper or environment variables

2. Backend Infrastructure (Go):
   - HTTP/REST API via Gin framework
   - WebSocket for real-time updates
   - MCP (Model Context Protocol) integration for external AI tools
   - JWT-based authentication/authorization

3. Frontend (React 19 + TypeScript + Tailwind CSS):
   - Component library using atomic design (atoms/molecules/organisms/templates)
   - Real-time chat interface with streaming support
   - Task management (Kanban board), knowledge base UI
   - Code search and MCP server management

4. Data Layers:
   - MongoDB: Primary persistent store (JWT-secured via user identity)
   - Qdrant: Vector database for semantic search across code and knowledge
   - Redis: Optional caching layer

5. Messaging & Events:
   - NATS JetStream for event-driven pub/sub patterns
   - File watcher for real-time code indexing

6. External Integrations:
   - Claude API for AI-powered features
   - Multiple embedding providers: Ollama (local), OpenAI, Voyage (Anthropic-recommended)
   - MCP protocol support for tool discovery and execution

DEPLOYMENT:
- Docker/Compose for local dev (docker-compose.yml)
- Kubernetes (GKE) for production (dev/prod namespaces)
- CI/CD: GitHub Actions for automated build/test/deploy

KEY PATTERNS:
- Service locator pattern for dependency injection
- Event-driven architecture with message streaming
- Semantic search via embeddings + vector similarity
- Task coordination with TODO item tracking
- Reflection/metacognitive layer for system learning
