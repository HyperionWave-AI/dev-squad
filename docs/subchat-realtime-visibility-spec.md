# Subchat Real-Time Visibility Feature - Implementation Specification

## Overview
Implement real-time subchat visibility to show what subagents are doing in real-time, including messages, tool calls, and results.

## Background
- Users currently see only subchat status ("active", "completed", "failed") with 5-second polling
- Cannot see actual messages, tool calls, and responses from subagents
- Need real-time streaming view using existing WebSocket infrastructure

## Architecture Approach
Reuse existing chat WebSocket endpoint (`/api/v1/chat/stream?sessionId=XXX`) since each subchat has its own session ID stored in the database.

## Implementation Tasks

### Task 1: Create SubchatDetailView Component
**File**: `ui/src/components/SubchatDetailView.tsx` (NEW)

**Requirements**:
- Accept `subchatId` prop
- Fetch subchat details from `/api/v1/subchats/{subchatId}` to get session ID
- Connect to WebSocket: `/api/v1/chat/stream?sessionId={subchatSessionId}`
- Reuse `ChatMessageView` component pattern from `ui/src/components/ChatMessageView.tsx`
- Show real-time messages, tool calls, and results
- Collapsible/expandable UI using MUI Accordion or Collapse
- Show subagent name and status in header
- Auto-scroll to latest message
- Handle loading, error, and empty states
- Manage WebSocket connection lifecycle (connect, disconnect, reconnect)

**Component Interface**:
```typescript
interface SubchatDetailViewProps {
  subchatId: string;
  onClose?: () => void;
}
```

**Key Implementation Points**:
1. Fetch subchat details on mount to get `sessionId`
2. Connect WebSocket only when sessionId is available
3. Reuse streaming state management from `CodeChatPage.tsx`
4. Display using `ChatMessageView` component
5. Clean up WebSocket on unmount
6. Show connection status (connecting, connected, error)

### Task 2: Modify SubchatCard Component
**File**: `ui/src/components/SubchatCard.tsx`

**Changes**:
- Add "View Details" button or expand icon (use `ExpandMore` icon from MUI)
- Add click handler that emits the subchat ID to parent
- Add visual indicator when details are being viewed (highlight/border)
- Maintain existing card layout and functionality

**New Props**:
```typescript
interface SubchatCardProps {
  subchat: Subchat;
  onClick?: (subchatId: string) => void;
  isExpanded?: boolean; // NEW: for visual feedback
  onToggleDetails?: (subchatId: string) => void; // NEW: for expand/collapse
}
```

### Task 3: Update SubchatList Component
**File**: `ui/src/components/SubchatList.tsx`

**Changes**:
- Add state to track which subchat is expanded: `const [expandedSubchatId, setExpandedSubchatId] = useState<string | null>(null)`
- Render `SubchatDetailView` below the selected subchat card
- Handle expand/collapse interactions
- Ensure only one subchat can be expanded at a time
- Add smooth expand/collapse animation

**Layout Structure**:
```tsx
{runningSubchats.map((subchat) => (
  <Box key={subchat.id}>
    <SubchatCard
      subchat={subchat}
      onClick={handleCardClick}
      isExpanded={expandedSubchatId === subchat.id}
      onToggleDetails={handleToggleDetails}
    />
    {expandedSubchatId === subchat.id && (
      <Collapse in={true}>
        <Box sx={{ mt: 2, ml: 4 }}>
          <SubchatDetailView
            subchatId={subchat.id}
            onClose={() => setExpandedSubchatId(null)}
          />
        </Box>
      </Collapse>
    )}
  </Box>
))}
```

### Task 4: Update Subchat TypeScript Interface
**File**: `ui/src/services/subchatService.ts`

**Change**:
Verify that `Subchat` interface includes `sessionId` field. If not, add it:

```typescript
export interface Subchat {
  id: string;
  parentChatId: string;
  sessionId?: string; // ADD THIS if missing
  subagentName: string;
  assignedTaskId?: string;
  assignedTodoId?: string;
  createdAt: string;
  updatedAt: string;
  status: 'active' | 'completed' | 'failed';
}
```

**Note**: Backend already has `SessionID` field in `Subchat` struct (verified in `hyper/internal/mcp/storage/subchat_storage.go:28`)

## Technical Requirements

### Component Dependencies
- MUI: `Box`, `Paper`, `Collapse`, `Accordion`, `CircularProgress`, `Alert`, `Typography`, `IconButton`
- MUI Icons: `ExpandMore`, `Close`
- React hooks: `useState`, `useEffect`, `useRef`, `useCallback`
- Services: `subchatService`, `connectChatStream` from `chatService`

### WebSocket Pattern (from chatService.ts)
```typescript
const connection = connectChatStream(sessionId, {
  onMessage: (content: string, done: boolean) => { /* handle streaming */ },
  onToolCall: (tool, args, id) => { /* handle tool call */ },
  onToolResult: (id, tool, result, error, durationMs) => { /* handle result */ },
  onError: (error) => { /* handle error */ },
  onOpen: () => { /* handle connection */ },
  onClose: () => { /* handle disconnect */ }
});

// Cleanup
return () => connection.disconnect();
```

### State Management Pattern
Follow the same pattern as `CodeChatPage.tsx`:
- `messages` - stored messages
- `isStreaming` - streaming state
- `streamingContent` - current streaming text
- `streamingToolCalls` - real-time tool calls
- `streamingToolResults` - real-time tool results
- `pendingToolCalls` - set of pending tool IDs

### Error Handling
- Show error alert if subchat fetch fails
- Show error alert if WebSocket connection fails
- Implement reconnect logic (3 second timeout)
- Show "connecting..." state while establishing connection
- Handle case where sessionId is null/undefined

### UI/UX Requirements
- Smooth expand/collapse animation (MUI Collapse with 300ms transition)
- Visual feedback on expanded card (border highlight or shadow)
- Auto-scroll to latest message in detail view
- Loading skeleton while fetching subchat details
- Connection status indicator (dot: green=connected, yellow=connecting, red=error)
- Close button in detail view header

## Testing Checklist
1. ✅ Create a subchat from main chat
2. ✅ Click "View Details" on an active subchat
3. ✅ Verify messages stream in real-time
4. ✅ Verify tool calls appear immediately
5. ✅ Verify tool results appear when complete
6. ✅ Verify collapse/expand works smoothly
7. ✅ Test with multiple subchats (only one expands at a time)
8. ✅ Test connection error handling
9. ✅ Test with subchat that has no sessionId yet
10. ✅ Test WebSocket cleanup on collapse

## Reference Files
- `ui/src/components/ChatMessageView.tsx` - Message rendering pattern
- `ui/src/components/ChatInputBox.tsx` - (Actually doesn't have WebSocket, but shows component patterns)
- `ui/src/pages/CodeChatPage.tsx` - WebSocket connection pattern (lines 103-200)
- `ui/src/services/chatService.ts` - WebSocket service (lines 224-318)
- `ui/src/components/SubchatCard.tsx` - Current card component
- `ui/src/components/SubchatList.tsx` - Current list component

## Success Criteria
✅ SubchatDetailView component created and renders messages
✅ WebSocket connection established using subchat's session ID
✅ Real-time streaming of messages, tool calls, and results
✅ Smooth UI integration with expand/collapse
✅ No new backend endpoints needed (reuses existing infrastructure)
✅ Follows existing code patterns and design system
✅ Handles all error cases gracefully
✅ Auto-scrolls to latest content
✅ Only one subchat expanded at a time
✅ WebSocket properly cleaned up on collapse/unmount

## Implementation Notes
- **DO NOT** create new backend endpoints - reuse `/api/v1/chat/stream`
- **DO** reuse `ChatMessageView` component exactly as-is
- **DO** follow the WebSocket pattern from `CodeChatPage.tsx`
- **DO** use MUI components consistently with existing code
- **DO** handle loading and error states
- **DO** implement proper cleanup in useEffect
- **VERIFY** that backend returns sessionId in subchat response

## Estimated Complexity
- SubchatDetailView: Medium (150-200 lines)
- SubchatCard modifications: Simple (20-30 lines)
- SubchatList modifications: Simple (30-40 lines)
- Type updates: Trivial (1-5 lines)

Total: ~200-275 lines of new/modified code
