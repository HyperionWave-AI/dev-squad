# Hyperion Coordinator UI - Quick Start

## Run the UI (Development Mode)

```bash
cd /Users/maxmednikov/MaxSpace/Hyperion/development/coordinator/ui
npm run dev
```

Open your browser to: **http://localhost:5173**

## What You'll See

### 1. Task Dashboard (Default View)

- **Human Tasks**: Top-level tasks with colored status indicators
  - Gray = Pending
  - Blue = In Progress
  - Green = Completed
  - Red = Blocked

- **Agent Tasks**: Nested under human tasks, showing:
  - Agent name
  - Priority indicator (⚪ 🟡 🟠 🔴)
  - Status badge
  - Blockers and dependencies

- **Auto-refresh**: Dashboard updates every 3 seconds

### 2. Knowledge Browser

- Click **🧠 Knowledge** button in header
- Select collection or search all
- Enter search query and press Enter or click **🔍 Search**
- Results show:
  - Collection name
  - Content text
  - Tags
  - Metadata (expandable)
  - Created date and author

### 3. Task Details (Coming Soon)

Click any task card to open detail modal with:
- Full task information
- Status dropdown (update status)
- Timeline (created, updated, completed)
- Dependencies and blockers
- Tags

## Current Status

**MVP with Mock Data**

The UI currently uses hardcoded sample data for demonstration. Real MCP integration coming next.

## Next Steps

To connect to real coordinator-mcp server:

1. Start MongoDB:
```bash
mongod --dbpath /path/to/data
```

2. Build and run coordinator-mcp server:
```bash
cd ../coordinator-mcp
make build
MONGODB_URI=mongodb://localhost:27017 MONGODB_DATABASE=hyperion_coordinator ./coordinator-mcp
```

3. Update `mcpClient.ts` to use real MCP transport (stdio or HTTP)

## Features Demonstrated

✅ Task Dashboard with hierarchical view
✅ Status color coding
✅ Priority indicators
✅ Auto-refresh (polling)
✅ Knowledge search interface
✅ Collection filtering
✅ Responsive design
✅ Clean, modern UI

## Development

### Project Structure
```
src/
├── components/
│   ├── TaskDashboard.tsx      # Main task view
│   ├── TaskCard.tsx            # Human task card
│   ├── AgentTaskCard.tsx       # Agent task card
│   ├── TaskDetail.tsx          # Task detail modal
│   └── KnowledgeBrowser.tsx    # Knowledge search
├── services/
│   └── mcpClient.ts            # MCP client (mock data)
├── types/
│   └── coordinator.ts          # TypeScript types
└── App.tsx                     # Main app with routing
```

### Technologies
- React 18 + TypeScript
- Vite (fast dev server)
- Tailwind CSS (utility-first styling)
- MCP SDK (Model Context Protocol)

## Troubleshooting

### Port Already in Use
```bash
# Kill process on port 5173
lsof -ti:5173 | xargs kill -9
npm run dev
```

### Build Errors
```bash
# Clean and reinstall
rm -rf node_modules package-lock.json
npm install
npm run build
```

### Styles Not Showing
```bash
# Rebuild Tailwind
npm run build
npm run dev
```

## What's Working

- ✅ Task visualization
- ✅ Status color coding
- ✅ Priority display
- ✅ Knowledge search UI
- ✅ Auto-refresh
- ✅ Responsive layout

## What's Coming Next

- [ ] Real MCP connection to coordinator-mcp server
- [ ] Task creation forms
- [ ] TODO item display and management
- [ ] Agent role management UI
- [ ] Task timeline view
- [ ] WebSocket for real-time updates
- [ ] Filtering and sorting
- [ ] Export/import capabilities

---

**Enjoy exploring the Hyperion Coordinator UI!** 🚀