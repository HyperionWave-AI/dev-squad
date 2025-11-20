# Hyperion HowTo Knowledge Base

This directory contains practical implementation patterns and best practices for building services in the Hyperion ecosystem.

## Overview

These guides provide step-by-step implementation patterns extracted from production code. Each guide includes working code examples, file references, and key considerations for maintainability and security.

## Available Patterns

### Database Patterns

#### [MongoDB Secure Connection](./mongodb-secure-connection.md)
Standard MongoDB connection pattern with JWT-based authentication and multi-tenant isolation.

**Key Topics:**
- Connection initialization with timeout and verification
- Service-level index creation
- JWT identity requirements for security
- Environment-driven configuration

**Technologies:** MongoDB 7.0, Go Driver, Zap Logger

---

### Vector Database Patterns

#### [Qdrant Client Initialization](./qdrant-client-initialization.md)
Initialize Qdrant vector database client with TEI embedding service integration.

**Key Topics:**
- Client configuration with embedding service
- Environment variable setup
- Default values and timeouts
- Vector dimension configuration (768 for nomic-embed-text-v1.5)

**Technologies:** Qdrant, TEI (Text Embeddings Inference)

#### [Qdrant Collection Management](./qdrant-collection-management.md)
Idempotent collection creation and dimension validation pattern.

**Key Topics:**
- Safe collection creation (won't recreate existing)
- Vector dimension validation
- Custom error types for dimension mismatches
- Cosine similarity configuration

**Technologies:** Qdrant REST API, Go HTTP Client

---

### Protocol Patterns

#### [MCP Tool Registration](./mcp-tool-registration.md)
Register Model Context Protocol tools with proper schema validation and handlers.

**Key Topics:**
- Tool definition with JSON Schema
- Parameter validation (required/optional)
- Handler function patterns
- Error result formatting
- Naming conventions (snake_case for tools, camelCase for params)

**Technologies:** MCP SDK, JSON Schema, Go

---

### Microservice Patterns

#### [Go Microservice Scaffolding](./go-microservice-scaffolding.md)
Standard HTTP microservice initialization with dependency injection and service layer architecture.

**Key Topics:**
- Dependency injection pattern
- Service layer initialization
- Router and middleware setup (Gin)
- Configuration loading from .env
- Error propagation and fail-fast design

**Technologies:** Go 1.25, Gin Framework, Zap Logger

#### [JWT Authentication Middleware](./jwt-authentication-middleware.md)
Gin middleware for JWT validation with dev/prod mode support.

**Key Topics:**
- Development mode with mock credentials
- Production JWT validation (HMAC)
- Flexible claim extraction (multiple formats)
- Context injection for handlers
- Security features (Bearer token, signature verification)

**Technologies:** Gin Framework, golang-jwt

---

### Build and Deploy Patterns

#### [Makefile Build and Deploy](./makefile-build-deploy.md)
Standard Makefile targets for consistent build, test, and deployment operations.

**Key Topics:**
- Development workflows (hot-reload, backend-only)
- Production builds (native binary, Docker)
- Testing and clean targets
- Environment variable management
- Common workflows and troubleshooting

**Technologies:** GNU Make, Go 1.25, Vite, Air

---

## Pattern Categories

### By Technology
- **Go**: All patterns use Go 1.25
- **Databases**: MongoDB, Qdrant
- **Frameworks**: Gin (HTTP), MCP (Protocol)
- **Build Tools**: Make, Vite, Air

### By Domain
- **Authentication**: JWT middleware
- **Database**: MongoDB connection, Qdrant initialization
- **Microservices**: Service scaffolding, routing
- **Protocol**: MCP tool registration
- **Build**: Makefile targets, development workflows

### By Pattern Type
- **Initialization**: MongoDB, Qdrant client, microservice setup
- **Middleware**: JWT authentication
- **Management**: Qdrant collections
- **Registration**: MCP tools
- **Build**: Makefile targets

## Usage Guidelines

### When to Use These Patterns

1. **Starting a New Service**: Begin with [Go Microservice Scaffolding](./go-microservice-scaffolding.md)
2. **Adding Database**: Use [MongoDB Secure Connection](./mongodb-secure-connection.md)
3. **Adding Vector Search**: Start with [Qdrant Client Initialization](./qdrant-client-initialization.md)
4. **Implementing Auth**: Follow [JWT Authentication Middleware](./jwt-authentication-middleware.md)
5. **Creating MCP Tools**: Reference [MCP Tool Registration](./mcp-tool-registration.md)
6. **Build Configuration**: Check [Makefile Build and Deploy](./makefile-build-deploy.md)

### Best Practices

All patterns follow these Hyperion standards:
- **Security First**: JWT identity for all operations, no system accounts
- **Fail Fast**: Early error returns during initialization
- **Structured Logging**: Zap logger with contextual fields
- **Environment Config**: .env files for configuration
- **Naming Conventions**: snake_case (tools), camelCase (JSON), PascalCase (Go)
- **Error Wrapping**: Use `fmt.Errorf("context: %w", err)` for error chains

### Contributing New Patterns

When adding new patterns to this knowledge base:

1. **Use the standard template**:
   - Overview
   - Technology
   - Use Case
   - Implementation (with file references)
   - Key Points
   - Metadata

2. **Include working code**: Extract real examples from production code

3. **Add file references**: Use `file:line` format for traceability

4. **Syntax highlighting**: Use proper language tags in code blocks

5. **Security notes**: Highlight security considerations prominently

6. **Update this README**: Add your pattern to the appropriate category

## Metadata

- **Collection**: howto
- **Last Updated**: 2025-11-20
- **Total Patterns**: 7
- **Pattern Format**: Markdown with code examples
