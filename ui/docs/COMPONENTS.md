# Hyperion Coordinator UI - Component Library Catalog

## Table of Contents

1. [Overview](#overview)
2. [Pages](#pages)
3. [Feature Components](#feature-components)
4. [Layout Components](#layout-components)
5. [UI Components](#ui-components)
6. [Component Patterns](#component-patterns)
7. [Material-UI Integration](#material-ui-integration)
8. [Component Testing](#component-testing)

---

## Overview

The Hyperion Coordinator UI contains **42 feature components** and **9 page components**, all built with React 19, TypeScript 5.8, and Material-UI 7.3.

### Component Categories

| Category | Count | Location |
|----------|-------|----------|
| **Pages** | 9 | `src/pages/` |
| **Feature Components** | 29 | `src/components/` |
| **Code Components** | 4 | `src/components/code/` |
| **Knowledge Components** | 5 | `src/components/knowledge/` |
| **Layout Components** | 1 | `src/App.tsx` |

### Design System

All components follow the Hyperion design system:
- **Material-UI 7.3.2** for base components
- **Tailwind CSS 4.1.13** for custom styling
- **@hello-pangea/dnd 18.0.1** for drag-and-drop
- **Custom theme** with blue/purple gradient branding

---

## Pages

Pages are top-level route components that compose multiple feature components.

### CodeChatPage.tsx

**Route**: `/chat`
**Purpose**: Real-time AI chat interface with streaming responses

**Features**:
- WebSocket-based streaming chat
- Tool call visualization
- Subchat management
- Session history
- Markdown rendering with syntax highlighting

**Key Components Used**:
- `ChatSessionList` - Session sidebar
- `ChatMessageView` - Message rendering
- `ChatInputBox` - Input with send button
- `ToolCallCard` - Tool execution display
- `SubchatCard` - Subchat visualization

**State Management**:
```typescript
const [sessions, setSessions] = useState<ChatSession[]>([]);
const [currentSession, setCurrentSession] = useState<ChatSession | null>(null);
const [messages, setMessages] = useState<ChatMessage[]>([]);
const [wsConnection, setWsConnection] = useState<ChatStreamConnection | null>(null);
```

**API Integration**:
- `chatService.getSessions()` - Fetch sessions
- `chatService.getMessages(sessionId)` - Load history
- `chatService.connectChatStream(sessionId, callbacks)` - WebSocket connection

---

### KanbanBoard Component (also serves as /tasks page)

**Route**: `/tasks`
**Purpose**: Drag-and-drop task management board

**Features**:
- 4-column Kanban layout (Pending, In Progress, Blocked, Completed)
- Drag-and-drop task cards with `@hello-pangea/dnd`
- Optimistic UI updates
- Real-time task synchronization
- Search and filter functionality
- Auto-refresh every 30 seconds

**Key Components**:
- `KanbanBoard` - Main board container
- `KanbanTaskCard` - Individual task card
- `TaskDetailDialog` - Task detail modal

**Columns**:
```typescript
const columns = [
  { id: 'pending', title: 'Pending', color: '#64748b' },
  { id: 'in_progress', title: 'In Progress', color: '#2563eb' },
  { id: 'blocked', title: 'Blocked', color: '#dc2626' },
  { id: 'completed', title: 'Completed', color: '#16a34a' },
];
```

**Drag-and-Drop Implementation**:
```typescript
<DragDropContext onDragEnd={handleDragEnd}>
  {columns.map(column => (
    <Droppable droppableId={column.id} key={column.id}>
      {(provided, snapshot) => (
        <div ref={provided.innerRef} {...provided.droppableProps}>
          {tasksInColumn.map((task, index) => (
            <Draggable draggableId={task.id} index={index} key={task.id}>
              {(provided, snapshot) => (
                <div ref={provided.innerRef} {...provided.draggableProps}>
                  <KanbanTaskCard task={task} />
                </div>
              )}
            </Draggable>
          ))}
          {provided.placeholder}
        </div>
      )}
    </Droppable>
  ))}
</DragDropContext>
```

**Responsive Design**:
- Desktop (1920px): 4 columns side-by-side
- Tablet (768px): 2 columns in 2 rows
- Mobile (375px): 1 column, vertical scrolling

---

### KnowledgeBasePage.tsx

**Route**: `/knowledge`
**Purpose**: Browse and search knowledge base

**Features**:
- Collection browser with categories
- Semantic search
- Entry creation and editing
- Collection management
- Metadata display
- Review and compaction tools

**Key Components**:
- `CollectionBrowser` - Collection selection
- `KnowledgeSearch` - Search interface
- `SearchResults` - Results display
- `KnowledgeCreate` - Entry creation
- `CollectionSettingsDialog` - Settings modal

**Layout Structure**:
```typescript
<KnowledgeLayout>
  <CollectionSidebar collections={collections} />
  <KnowledgeSearch onSearch={handleSearch} />
  <SearchResults results={results} />
</KnowledgeLayout>
```

---

### CodeSearchPage.tsx

**Route**: `/code`
**Purpose**: Semantic code search and indexing

**Features**:
- Code indexing status
- Folder management
- Semantic search
- Syntax-highlighted results
- Index configuration

**Key Components**:
- `CodeSearch` - Search interface
- `CodeResults` - Results display with syntax highlighting
- `IndexStatus` - Indexing progress
- `CodeIndexConfig` - Configuration panel

**Code Highlighting**:
```typescript
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

<SyntaxHighlighter
  language="typescript"
  style={vscDarkPlus}
  showLineNumbers
>
  {code}
</SyntaxHighlighter>
```

---

### HTTPToolsPage.tsx

**Route**: `/tools`
**Purpose**: HTTP tool testing and execution

**Features**:
- Tool discovery
- Parameter input
- Tool execution
- Response display
- Error handling

**Key Components**:
- Tool selection dropdown
- Parameter form with JSON validation
- Execute button with loading state
- Response viewer with JSON formatting

---

### MCPServersPage.tsx

**Route**: `/mcp-servers`
**Purpose**: MCP server registry management

**Features**:
- Server list with status indicators
- Add/edit/remove servers
- Server health checks
- Tool discovery per server
- Configuration management

**Key Components**:
- `AddMCPServerDialog` - Add server modal
- `EditMCPServerDialog` - Edit server modal
- `ServerToolsList` - Tools list per server

**Server Display**:
```typescript
interface MCPServer {
  name: string;
  transport: 'stdio' | 'sse';
  command: string;
  args: string[];
  env?: Record<string, string>;
  status: 'online' | 'offline' | 'error';
}
```

---

### SubagentsPage.tsx

**Route**: `/subagents`
**Purpose**: Sub-agent management interface

**Features**:
- Agent selection
- Status monitoring
- Task assignment
- Performance metrics

**Key Components**:
- `AgentSelector` - Agent dropdown
- Agent status cards
- Task list per agent

---

### SettingsPage.tsx

**Route**: `/settings`
**Purpose**: Application settings and preferences

**Features**:
- Theme switching (Light/Dark)
- AI configuration
- User preferences
- System information

**Key Components**:
- `DarkModeToggle` - Theme switcher
- Settings forms
- Save/reset buttons

---

### AISettingsPage.tsx

**Route**: `/settings/ai` (nested)
**Purpose**: AI model configuration

**Features**:
- Model selection
- API key management
- Temperature/top_p settings
- Token limits

---

## Feature Components

### Chat Components

#### ChatSessionList.tsx

**Purpose**: Display and manage chat sessions

**Features**:
- Session list with timestamps
- Active session highlighting
- Create new session button
- Delete session action
- Session title editing

**Props**:
```typescript
interface ChatSessionListProps {
  sessions: ChatSession[];
  currentSessionId?: string;
  onSelectSession: (session: ChatSession) => void;
  onCreateSession: () => void;
  onDeleteSession: (sessionId: string) => void;
}
```

**MUI Components Used**:
- `List`, `ListItem`, `ListItemButton`
- `IconButton` (for delete)
- `TextField` (for session title editing)
- `Button` (for new session)

---

#### ChatMessageView.tsx

**Purpose**: Render individual chat messages with rich content

**Features**:
- Markdown rendering with `react-markdown`
- Code syntax highlighting
- Tool call display
- Tool result display
- Role-based styling (user/assistant/system)
- Timestamp display
- Copy message button

**Message Types**:
```typescript
type MessageRole = 'user' | 'assistant' | 'system' | 'tool_call' | 'tool_result';

interface ChatMessage {
  id: string;
  role: MessageRole;
  content: string;
  timestamp: string;
  toolCall?: ToolCall;
  toolResult?: ToolResult;
}
```

**Rendering Example**:
```typescript
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

<ReactMarkdown
  remarkPlugins={[remarkGfm]}
  components={{
    code: ({ node, inline, className, children, ...props }) => {
      const match = /language-(\w+)/.exec(className || '');
      return !inline && match ? (
        <SyntaxHighlighter language={match[1]} {...props}>
          {String(children).replace(/\n$/, '')}
        </SyntaxHighlighter>
      ) : (
        <code className={className} {...props}>
          {children}
        </code>
      );
    },
  }}
>
  {message.content}
</ReactMarkdown>
```

---

#### ChatInputBox.tsx

**Purpose**: Chat message input with send button

**Features**:
- Multi-line textarea with auto-resize
- Send button with loading state
- Enter to send (Shift+Enter for new line)
- Character count (optional)
- Disabled state during streaming

**Props**:
```typescript
interface ChatInputBoxProps {
  onSend: (message: string) => void;
  disabled?: boolean;
  placeholder?: string;
}
```

**MUI Components**:
- `TextField` (multiline)
- `IconButton` with `Send` icon
- `CircularProgress` (loading state)

---

#### ToolCallCard.tsx

**Purpose**: Display tool execution information

**Features**:
- Tool name badge
- Arguments display (JSON formatted)
- Execution status indicator
- Collapsible args view
- Copy args button

**Display Example**:
```typescript
<Card>
  <CardHeader
    avatar={<Build />}
    title={toolCall.tool}
    subheader="Executing..."
  />
  <CardContent>
    <Typography variant="body2" color="text.secondary">
      Arguments:
    </Typography>
    <pre>{JSON.stringify(toolCall.args, null, 2)}</pre>
  </CardContent>
</Card>
```

---

#### ToolResultCard.tsx

**Purpose**: Display tool execution results

**Features**:
- Result/error display
- Execution duration
- Success/error color coding
- Collapsible result view
- Copy result button

**Props**:
```typescript
interface ToolResultCardProps {
  toolResult: ToolResult;
}

interface ToolResult {
  id: string;
  tool: string;
  result: any;
  error: string | null;
  durationMs: number;
}
```

---

#### SubchatCard.tsx

**Purpose**: Display subchat information in parent chat

**Features**:
- Subchat title
- Message count
- Status indicator
- Click to open subchat
- Parent-child relationship visualization

---

### Task Components

#### TaskCard.tsx

**Purpose**: Display human task in dashboard view

**Features**:
- Task title and description
- Status badge with color
- Priority indicator
- Created date
- Tags display
- Click to expand details

**Status Colors**:
```typescript
const statusColors = {
  pending: '#64748b',     // Gray
  in_progress: '#2563eb', // Blue
  completed: '#16a34a',   // Green
  blocked: '#dc2626',     // Red
};
```

---

#### AgentTaskCard.tsx

**Purpose**: Display agent task with todos

**Features**:
- Agent name badge
- Role description
- Status indicator
- Todo list with checkboxes
- Progress bar
- Files modified list
- Prompt notes display
- Expand/collapse todos

**Props**:
```typescript
interface AgentTaskCardProps {
  task: AgentTask;
  onUpdateStatus: (status: TaskStatus) => void;
  onUpdateTodo: (todoId: string, status: TodoStatus) => void;
}
```

**Layout**:
```typescript
<Card>
  <CardHeader
    avatar={<SmartToy />}
    title={task.agentName}
    subheader={task.role}
    action={<StatusBadge status={task.status} />}
  />
  <CardContent>
    <Typography variant="body2">{task.contextSummary}</Typography>
    <TodoList todos={task.todos} />
    <LinearProgress value={progressPercentage} />
  </CardContent>
</Card>
```

---

#### TaskDetailDialog.tsx

**Purpose**: Full task detail modal

**Features**:
- Full task information
- Status update dropdown
- Todo management
- Prompt notes editor
- Timeline view
- Files modified list
- Agent tasks list (for human tasks)

**MUI Components**:
- `Dialog`, `DialogTitle`, `DialogContent`, `DialogActions`
- `Select` (status dropdown)
- `List` (todos)
- `Timeline` (optional)

---

#### KanbanTaskCard.tsx

**Purpose**: Task card for Kanban board

**Features**:
- Compact card design for board view
- Priority badge
- Status indicator
- Truncated description (2 lines)
- Tag chips
- Drag handle
- Hover effects

**Drag Styling**:
```typescript
sx={{
  opacity: isDragging ? 0.5 : 1,
  transform: isDragging ? 'rotate(2deg)' : 'none',
  transition: 'transform 0.2s ease',
  '&:hover': {
    boxShadow: 3,
    transform: 'translateY(-2px)',
  }
}}
```

---

### Knowledge Components

#### CollectionBrowser.tsx

**Purpose**: Browse knowledge collections

**Features**:
- Collection list with counts
- Category filtering
- Search filter
- Create collection button
- Delete collection action

**Props**:
```typescript
interface CollectionBrowserProps {
  collections: CollectionInfo[];
  selectedCollection?: string;
  onSelectCollection: (collection: string) => void;
  onCreateCollection: () => void;
}
```

---

#### KnowledgeSearch.tsx

**Purpose**: Knowledge search interface

**Features**:
- Search input with debounce
- Collection filter
- Limit selector
- Search button
- Clear filters button

**Search Implementation**:
```typescript
const [query, setQuery] = useState('');
const [debouncedQuery] = useDebounce(query, 500);

useEffect(() => {
  if (debouncedQuery) {
    handleSearch(debouncedQuery);
  }
}, [debouncedQuery]);
```

---

#### SearchResults.tsx

**Purpose**: Display knowledge search results

**Features**:
- Result list with scores
- Metadata display
- Content preview
- Copy content button
- Expand/collapse metadata

**Result Card**:
```typescript
<Card>
  <CardHeader
    title={entry.collection}
    subheader={`Score: ${entry.score?.toFixed(3)}`}
  />
  <CardContent>
    <Typography variant="body2">{entry.text}</Typography>
    <Chip label={`Created: ${formatDate(entry.createdAt)}`} />
  </CardContent>
</Card>
```

---

#### KnowledgeCreate.tsx

**Purpose**: Create knowledge entries

**Features**:
- Collection selector
- Text input (multi-line)
- Metadata JSON editor
- Submit button with validation
- Success/error messages

---

#### CollectionSettingsDialog.tsx

**Purpose**: Configure collection settings

**Features**:
- Rename collection
- Set default limit
- Configure metadata schema
- Compaction settings
- Delete collection

---

### Dialogs & Modals

#### AddMCPServerDialog.tsx

**Purpose**: Add new MCP server

**Form Fields**:
- Server name
- Transport type (stdio/sse)
- Command
- Arguments (array)
- Environment variables (JSON)

**Validation**:
```typescript
const [errors, setErrors] = useState<Record<string, string>>({});

const validate = () => {
  const newErrors: Record<string, string> = {};

  if (!formData.name.trim()) {
    newErrors.name = 'Name is required';
  }

  if (!formData.command.trim()) {
    newErrors.command = 'Command is required';
  }

  setErrors(newErrors);
  return Object.keys(newErrors).length === 0;
};
```

---

#### SubchatCreationDialog.tsx

**Purpose**: Create subchat from parent chat

**Features**:
- Subchat title input
- Context selection (which messages to include)
- Create button with validation

---

#### CompactionDialog.tsx

**Purpose**: Knowledge collection compaction

**Features**:
- Collection selector
- Compaction strategy options
- Preview changes
- Confirm/cancel buttons
- Progress indicator

---

#### ReviewResultDialog.tsx

**Purpose**: Review knowledge search results before actions

**Features**:
- Result list
- Select/deselect entries
- Bulk actions (delete, update)
- Confirm button

---

### Code Components

#### CodeSearch.tsx

**Purpose**: Semantic code search interface

**Features**:
- Query input
- Language filter
- File path filter
- Search button
- Recent searches

---

#### CodeResults.tsx

**Purpose**: Display code search results with syntax highlighting

**Features**:
- Result list with file paths
- Line numbers
- Syntax highlighted code blocks
- Copy code button
- Open in editor link
- Score display

**Syntax Highlighting**:
```typescript
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

<SyntaxHighlighter
  language={detectLanguage(result.filePath)}
  style={vscDarkPlus}
  showLineNumbers
  startingLineNumber={result.lineNumber}
  wrapLines
>
  {result.code}
</SyntaxHighlighter>
```

---

#### IndexStatus.tsx

**Purpose**: Display code indexing status

**Features**:
- Indexed folder list
- Index progress bar
- File count
- Last indexed timestamp
- Re-index button

**Status Display**:
```typescript
<Card>
  <CardHeader title="Indexing Status" />
  <CardContent>
    <LinearProgress variant="determinate" value={progress} />
    <Typography>{filesIndexed} / {totalFiles} files</Typography>
    <Typography variant="caption" color="text.secondary">
      Last indexed: {formatDate(lastIndexed)}
    </Typography>
  </CardContent>
  <CardActions>
    <Button onClick={handleReindex}>Re-index</Button>
  </CardActions>
</Card>
```

---

#### CodeIndexConfig.tsx

**Purpose**: Configure code indexing

**Features**:
- Add/remove folders
- File extension filters
- Ignore patterns
- Index settings

---

### Utility Components

#### DarkModeToggle.tsx

**Purpose**: Toggle between light and dark themes

**Implementation**:
```typescript
import { IconButton } from '@mui/material';
import { LightMode, DarkMode } from '@mui/icons-material';

const DarkModeToggle: React.FC = () => {
  const [themeMode, setThemeMode] = useState<'light' | 'dark'>('light');

  const toggleTheme = () => {
    const newMode = themeMode === 'light' ? 'dark' : 'light';
    setThemeMode(newMode);
    localStorage.setItem('theme-mode', newMode);
  };

  return (
    <IconButton onClick={toggleTheme} aria-label="Toggle dark mode">
      {themeMode === 'light' ? <DarkMode /> : <LightMode />}
    </IconButton>
  );
};
```

---

#### PromptNotesEditor.tsx

**Purpose**: Edit task/todo prompt notes

**Features**:
- Multi-line editor
- Save/cancel buttons
- Last updated timestamp
- Markdown preview (optional)

---

#### TodoPromptNotes.tsx

**Purpose**: Display and edit todo-level prompt notes

**Features**:
- Notes display
- Edit button
- Timestamp display
- Clear notes action

---

## Component Patterns

### Standard Component Structure

```typescript
import React, { useState, useEffect } from 'react';
import {
  Card,
  CardContent,
  CardHeader,
  Typography,
  Button,
} from '@mui/material';
import type { YourType } from '@/types/coordinator';

interface YourComponentProps {
  data: YourType;
  onAction: (id: string) => void;
  optional?: boolean;
}

export const YourComponent: React.FC<YourComponentProps> = ({
  data,
  onAction,
  optional = false,
}) => {
  // State
  const [localState, setLocalState] = useState<string>('');

  // Effects
  useEffect(() => {
    // Side effects
  }, [dependency]);

  // Handlers
  const handleClick = () => {
    onAction(data.id);
  };

  // Render
  return (
    <Card>
      <CardHeader title={data.title} />
      <CardContent>
        <Typography>{data.description}</Typography>
        <Button onClick={handleClick}>Action</Button>
      </CardContent>
    </Card>
  );
};

export default YourComponent;
```

### Common Patterns

#### 1. **Loading State Pattern**

```typescript
const [loading, setLoading] = useState(true);
const [error, setError] = useState<Error | null>(null);

useEffect(() => {
  const fetchData = async () => {
    try {
      setLoading(true);
      const result = await apiClient.getData();
      setData(result);
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Unknown error'));
    } finally {
      setLoading(false);
    }
  };

  fetchData();
}, []);

if (loading) return <CircularProgress />;
if (error) return <Alert severity="error">{error.message}</Alert>;
```

#### 2. **Modal Pattern**

```typescript
const [open, setOpen] = useState(false);

<Dialog open={open} onClose={() => setOpen(false)} maxWidth="md" fullWidth>
  <DialogTitle>Title</DialogTitle>
  <DialogContent>Content</DialogContent>
  <DialogActions>
    <Button onClick={() => setOpen(false)}>Cancel</Button>
    <Button onClick={handleSave} variant="contained">Save</Button>
  </DialogActions>
</Dialog>
```

#### 3. **Form Pattern**

```typescript
const [formData, setFormData] = useState({ field1: '', field2: '' });
const [errors, setErrors] = useState<Record<string, string>>({});

const handleChange = (field: string) => (e: React.ChangeEvent<HTMLInputElement>) => {
  setFormData(prev => ({ ...prev, [field]: e.target.value }));
  // Clear error on change
  if (errors[field]) {
    setErrors(prev => ({ ...prev, [field]: '' }));
  }
};

const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault();

  // Validate
  const newErrors = validate(formData);
  if (Object.keys(newErrors).length > 0) {
    setErrors(newErrors);
    return;
  }

  // Submit
  await apiClient.submitData(formData);
};
```

---

## Material-UI Integration

### Theme Customization

All components use the custom Hyperion theme defined in `src/theme.ts`:

```typescript
import { ThemeProvider } from '@mui/material/styles';
import { getTheme } from './theme';

<ThemeProvider theme={getTheme(themeMode)}>
  <YourComponent />
</ThemeProvider>
```

### Custom Colors

```typescript
const theme = createTheme({
  palette: {
    primary: {
      main: '#2563eb', // Blue-600
      light: '#60a5fa', // Blue-400
      dark: '#1e40af', // Blue-700
    },
    secondary: {
      main: '#9333ea', // Purple-600
    },
    success: {
      main: '#16a34a', // Green-600
    },
    // ... other colors
  },
});
```

### Responsive Breakpoints

```typescript
import { useTheme, useMediaQuery } from '@mui/material';

const theme = useTheme();
const isMobile = useMediaQuery(theme.breakpoints.down('sm')); // <600px
const isTablet = useMediaQuery(theme.breakpoints.between('sm', 'md')); // 600-900px
const isDesktop = useMediaQuery(theme.breakpoints.up('lg')); // >1200px
```

### Common MUI Components Used

| Component | Usage | Import |
|-----------|-------|--------|
| `Card`, `CardContent`, `CardHeader` | Content containers | `@mui/material` |
| `Button`, `IconButton` | Actions | `@mui/material` |
| `TextField` | Form inputs | `@mui/material` |
| `Dialog`, `DialogTitle`, `DialogContent` | Modals | `@mui/material` |
| `List`, `ListItem`, `ListItemButton` | Lists | `@mui/material` |
| `Chip` | Tags, badges | `@mui/material` |
| `AppBar`, `Toolbar` | Navigation | `@mui/material` |
| `CircularProgress`, `LinearProgress` | Loading | `@mui/material` |
| `Alert` | Notifications | `@mui/material` |

---

## Component Testing

### Testing Strategy

All major components have corresponding test files using:
- **Playwright** for E2E tests
- **Vitest** for unit tests
- **@testing-library/react** for component tests

### Example Component Test

```typescript
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { YourComponent } from './YourComponent';

describe('YourComponent', () => {
  it('renders correctly', () => {
    const mockData = { id: '1', title: 'Test' };
    render(<YourComponent data={mockData} onAction={vi.fn()} />);

    expect(screen.getByText('Test')).toBeInTheDocument();
  });

  it('calls onAction when button clicked', () => {
    const mockOnAction = vi.fn();
    const mockData = { id: '1', title: 'Test' };

    render(<YourComponent data={mockData} onAction={mockOnAction} />);

    fireEvent.click(screen.getByRole('button', { name: /action/i }));

    expect(mockOnAction).toHaveBeenCalledWith('1');
  });
});
```

### Playwright E2E Tests

See [Testing Guide](./TESTING.md) for comprehensive testing documentation.

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture
- [API Integration Guide](./API_INTEGRATION.md) - API usage patterns
- [UI/UX Patterns](./UI_UX_PATTERNS.md) - Design patterns
- [Testing Guide](./TESTING.md) - Testing strategies
- [Developer Guide](./DEVELOPER_GUIDE.md) - Getting started

---

**Component Count**: 42 feature components + 9 pages = **51 total**
**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
