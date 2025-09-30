# Hyperion Coordinator UI

Frontend UI for the Hyperion Task Coordinator - Task and Knowledge Management for the Parallel Squad System.

## Overview

This React + TypeScript UI provides visual access to:

- **Task Dashboard**: View human tasks and their child agent tasks with real-time status updates
- **Knowledge Browser**: Search and explore knowledge base collections
- **Task Details**: Deep dive into individual tasks with status management

## Technology Stack

- **React 18** with TypeScript
- **Vite** for fast development and building
- **Tailwind CSS** for styling
- **MCP SDK** for connecting to coordinator MCP server

## Getting Started

### Installation

```bash
npm install
```

### Development

```bash
npm run dev
```

The UI will start at `http://localhost:5173`

### Building for Production

```bash
npm run build
```

## Project Structure

```
ui/
├── src/
│   ├── components/          # React components
│   │   ├── TaskDashboard.tsx
│   │   ├── TaskCard.tsx
│   │   ├── AgentTaskCard.tsx
│   │   ├── TaskDetail.tsx
│   │   └── KnowledgeBrowser.tsx
│   ├── services/
│   │   └── mcpClient.ts     # MCP client
│   ├── types/
│   │   └── coordinator.ts   # Type definitions
│   ├── App.tsx
│   └── index.css
```

## Features

### Task Dashboard

- Human tasks with nested agent tasks
- Color-coded status (pending, in_progress, completed, blocked)
- Priority indicators (low, medium, high, urgent)
- Real-time updates (3-second polling)
- Click-to-detail view

### Knowledge Browser

- Collection filtering (task, adr, data-contracts, etc.)
- Full-text search
- Metadata display
- Tag-based categorization

### Task Detail Modal

- Full task information
- Status management
- Timeline view
- Dependencies and blockers

## Current Implementation

**MVP Status**: Using mock data for development

- HTTP polling simulation (no real MCP connection yet)
- Hardcoded sample tasks
- Local dev only (no authentication)

## Next Steps

- [ ] Connect to real coordinator-mcp server
- [ ] Add task creation forms
- [ ] Display and manage task TODOs
- [ ] Add agent role management UI
- [ ] Implement WebSocket for real-time updates

## Related Documentation

- [Coordinator MCP Server](../coordinator-mcp/CLAUDE.md)
- [Hyperion Parallel Squad System](/CLAUDE.md)

---

**Status**: MVP Complete - Mock Data Mode
**Maintainer**: AI & Experience Squad
