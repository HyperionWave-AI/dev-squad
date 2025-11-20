# Hyperion Architecture Documentation

This directory contains comprehensive architecture documentation exported from the Hyperion knowledge base. The documentation is organized by technical domain and covers all aspects of the system.

## Table of Contents

- [Hyperion Architecture](#hyperion-architecture)
- [Frontend Patterns](#frontend-patterns)
- [Infrastructure](#infrastructure)
- [Data Architecture](#data-architecture)
- [Event Systems](#event-systems)
- [Security Patterns](#security-patterns)
- [AI Integration](#ai-integration)
- [Backend Services](#backend-services)

---

## Hyperion Architecture

Core system architecture, microservices design, and integration patterns.

- [**System Overview**](hyperion-architecture/system-overview.md)
  High-level architecture of the Hyperion platform, including core components, data layers, deployment options, and key architectural patterns.

- [**Integration Map**](hyperion-architecture/integration-map.md)
  End-to-end data flow, service responsibilities, persistence tiers, real-time communication, security layers, and scaling considerations.

- [**Backend Microservices**](hyperion-architecture/backend-microservices.md)
  Detailed breakdown of internal services including AI service, MCP handlers, storage layer, code indexing, HTTP handlers, and middleware.

## Frontend Patterns

React-based frontend architecture and component design.

- [**React Architecture**](frontend-patterns/react-architecture.md)
  React 19 + TypeScript frontend structure using atomic design, routing, state management, custom hooks, and performance optimizations.

## Infrastructure

Deployment, containerization, and build pipeline configuration.

- [**Deployment Architecture**](infrastructure/deployment-architecture.md)
  Docker Compose setup, Makefile build targets, Kubernetes deployment, environment configuration, and health checks.

## Data Architecture

Database schemas, collections, and query patterns.

- [**MongoDB Collections**](data-architecture/mongodb-collections.md)
  Complete MongoDB schema documentation covering knowledge management, task coordination, reflection system, code indexing, and user settings collections.

## Event Systems

Real-time communication and event processing.

- [**File Watcher**](event-systems/file-watcher.md)
  Filesystem event processing with fsnotify, debouncing, worker pool pattern, and automatic code re-indexing.

- [**WebSocket Streaming**](event-systems/websocket-streaming.md)
  Real-time chat streaming architecture with graceful lifecycle management, concurrent goroutines, and write safety mechanisms.

## Security Patterns

Authentication, authorization, and rate limiting.

- [**Rate Limiting**](security-patterns/rate-limiting.md)
  Token bucket algorithm implementation for API protection, DDoS mitigation, and fair resource allocation.

- [**JWT Authentication**](security-patterns/jwt-authentication.md)
  JWT-based authentication and authorization middleware with flexible claims extraction and development/production modes.

## AI Integration

Claude API, MCP protocol, and task coordination.

- [**Task Coordination System**](ai-integration/task-coordination-system.md)
  AI-powered workflow orchestration with 6-step mandatory workflow, subagent delegation, task hierarchy, and traceability.

- [**Claude API Integration**](ai-integration/claude-api-integration.md)
  Anthropic Claude integration via LangChain, including streaming, tool calling, task summarization, and configuration.

- [**MCP Server Architecture**](ai-integration/mcp-server-architecture.md)
  Model Context Protocol implementation with dual server modes, tool registration (19+ tools), and metadata registry.

## Backend Services

Service implementations and storage patterns.

- [**Task Storage Patterns**](backend-services/task-storage-patterns.md)
  MongoDB collections and query patterns for human tasks, agent tasks, and TODO items with full traceability.

---

## Documentation Metadata

- **Total Collections:** 8
- **Total Entries:** 14
- **Export Date:** 2025-11-17
- **Version:** 1.0

## Contributing

This documentation is auto-generated from the Hyperion knowledge base. To update:

1. Store knowledge using the `knowledge_store` MCP tool
2. Organize entries into the appropriate collection
3. Re-export using the documentation export workflow

## Knowledge Base Collections

The source knowledge base contains the following collections:

- `hyperion-architecture` - System design and architecture (3 entries)
- `frontend-patterns` - React and UI patterns (1 entry)
- `infrastructure` - Deployment and DevOps (1 entry)
- `data-architecture` - Database schemas and patterns (1 entry)
- `event-systems` - Real-time and event processing (2 entries)
- `security-patterns` - Security implementations (2 entries)
- `ai-integration` - AI and MCP integration (3 entries)
- `backend-services` - Backend service patterns (1 entry)

---

For questions or updates, consult the Hyperion Coordinator system.
