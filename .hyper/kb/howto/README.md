# HowTo Guides

**Collection:** howto
**Last Updated:** 2025-11-21

---

## Overview

This directory contains practical, step-by-step guides for implementing common patterns and workflows in the Hyperion AI Platform. Each guide is tutorial-style and provides general templates that can be adapted to your specific use case.

---

## Available Guides

### Backend Development

#### [Go Microservice Scaffolding](./go-microservice-scaffolding.md)
Learn how to scaffold a new Go 1.25 microservice from scratch with proper structure, configuration, and error handling.

**Topics:** Project structure, main entry point, graceful shutdown, Makefile automation

**Use when:** Starting a new service, understanding Hyperion backend architecture

---

#### [JWT Authentication Middleware](./jwt-authentication-middleware.md)
Implement JWT token validation middleware for protecting API endpoints with user identity extraction.

**Topics:** Token validation, claims extraction, context injection, dev mode

**Use when:** Securing APIs, implementing user authentication, multi-tenant apps

---

#### [MongoDB Secure Connection](./mongodb-secure-connection.md)
Connect to MongoDB using user JWT identity for secure, multi-tenant data access patterns.

**Topics:** User-scoped clients, automatic filtering, connection pooling, security patterns

**Use when:** Building multi-tenant apps, ensuring data isolation, following zero-trust principles

**Security Note:** Hyperion REQUIRES all MongoDB operations use user JWT identity. System-level access is prohibited.

---

#### [Qdrant Client Initialization](./qdrant-client-initialization.md)
Set up and use Qdrant vector database for semantic search and embedding storage.

**Topics:** Client setup, collection management, embedding generation, similarity search

**Use when:** Implementing semantic search, building knowledge bases, vector similarity queries

---

#### [MCP Tool Registration](./mcp-tool-registration.md)
Register and implement Model Context Protocol (MCP) tools for AI agent integration.

**Topics:** Tool schema definition, handler implementation, parameter validation, response formatting

**Use when:** Creating AI tools, extending agent capabilities, integrating with LLMs

---

#### [REST API Endpoint Patterns](./rest-api-endpoint-patterns.md)
Design and implement REST API endpoints following Hyperion conventions.

**Topics:** Route registration, request validation, error handling, response formatting

**Use when:** Creating new API endpoints, understanding API patterns

---

#### [WebSocket Connection Handling](./websocket-connection-handling.md)
Implement WebSocket connections for real-time, bidirectional communication.

**Topics:** Connection upgrade, message handling, broadcasting, graceful disconnect

**Use when:** Building chat interfaces, real-time updates, streaming responses

---

### Build & Deploy

#### [Makefile Build Commands](./makefile-build-deploy.md)
Use Makefile targets for building, testing, and deploying services.

**Topics:** Build automation, hot reload, testing, CI/CD integration

**Use when:** Building services, setting up dev environment, understanding build pipeline

**Important:** Production deploys MUST use CI/CD (GitHub Actions), not manual commands.

---

### Frontend Development

#### [React Component Structuring](./react-component-structuring.md)
Organize React components using Atomic Design principles (atoms, molecules, organisms).

**Topics:** Component hierarchy, file organization, reusability, TypeScript integration

**Use when:** Building UI components, structuring frontend codebase

---

## How to Use These Guides

### 1. Choose Your Guide
Select a guide based on your current task. Use the "Use when" section to determine relevance.

### 2. Review Prerequisites
Check the "Prerequisites" section to ensure you have necessary knowledge and tools.

### 3. Follow Steps Sequentially
Work through steps in order. Code examples are general templates - adapt to your needs.

### 4. Understand Best Practices
Pay special attention to "Best Practices" and "Common Pitfalls" sections.

### 5. Troubleshoot Issues
Use the "Troubleshooting" section for common errors and solutions.

### 6. Consult Related Docs
Review "Related Documentation" links for deeper understanding.

---

## Guide Structure

Each guide follows this template:

```markdown
# How to [Task Name]

**Collection:** howto
**Tags:** relevant, tags, here
**Version:** 1.0
**Last Updated:** YYYY-MM-DD

## Overview
Brief description of what this guide covers.

## Prerequisites
- Required knowledge
- Tools needed
- Related concepts

## When to Use This Guide
- Specific use cases
- When to apply this pattern

## Steps
### Step 1: [First Action]
Explanation and code examples...

### Step 2: [Second Action]
...

## Best Practices
Key recommendations and patterns.

## Common Pitfalls
Mistakes to avoid.

## Related Documentation
Links to other KB articles.

## Troubleshooting
Common issues and solutions.
```

---

## Contributing New Guides

When creating new HowTo guides:

1. **Use General Templates:** Provide reusable patterns, not project-specific code
2. **Explain Why:** Don't just show what - explain why decisions are made
3. **Include Examples:** Show both good (✅) and bad (❌) patterns
4. **Add Troubleshooting:** Document common errors you encountered
5. **Link Related Docs:** Cross-reference other KB articles
6. **Keep Updated:** Update guides when patterns or tools change

---

## Related Documentation

### Architecture & Design
- [Component Architecture](../component-architecture.md) - System overview
- [Data Contracts](../data-contracts.md) - Type definitions
- [API Service Layer](../api-service-layer.md) - REST API documentation

### Infrastructure & Data
- [MongoDB Integration](../mongodb-integration.md) - Database patterns
- [Qdrant Integration](../qdrant-integration.md) - Vector database
- [Configuration Reference](../configuration-reference.md) - Environment variables

### Security
- [JWT Authentication](../security-patterns/jwt-authentication.md) - Auth architecture
- [Rate Limiting](../security-patterns/rate-limiting.md) - API protection

### Frontend
- [UI Client Stack](../ui-client-stack.md) - Frontend technologies
- [React Architecture](../frontend-patterns/react-architecture.md) - Component patterns

---

## Feedback & Improvements

Found an issue or have suggestions?
- Update the guide directly (with version bump)
- Add to "Common Pitfalls" if you discover new issues
- Expand "Troubleshooting" with solutions you found

These guides are living documents - keep them accurate and useful!
