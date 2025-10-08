# Knowledge UI Components - Quick Implementation Guide

## For ui-dev Agent

The specifications from Frontend Experience Specialist didn't persist to disk. Use existing patterns instead:

### Existing Patterns to Follow:
- **Reference Component**: `coordinator/ui/src/components/KnowledgeBrowser.tsx` (Tailwind CSS patterns)
- **API Client**: `coordinator/ui/src/services/mcpClient.ts` (MCP client patterns)
- **Types**: `coordinator/ui/src/types/coordinator.ts` and `knowledge.ts`

### Quick Implementation (Use contextHint from TODOs):

1. **KnowledgeSearch.tsx** - Search interface with Tailwind
2. **KnowledgeCreate.tsx** - Form component
3. **CollectionBrowser.tsx** - Grid view of collections

### API Endpoints (Backend Ready):
- GET `/api/knowledge/search?collectionName=X&query=Y&limit=N`
- POST `/api/knowledge` - body: `{collectionName, information, metadata}`
- GET `/api/knowledge/collections`

### Use Tailwind CSS (not Material-UI)
The existing codebase uses Tailwind. Follow KnowledgeBrowser.tsx patterns.
