# Hyperion Coordinator UI - Developer Guide

## Table of Contents

1. [Getting Started](#getting-started)
2. [Prerequisites](#prerequisites)
3. [Installation](#installation)
4. [Development Workflow](#development-workflow)
5. [Project Structure](#project-structure)
6. [Coding Standards](#coding-standards)
7. [Common Tasks](#common-tasks)
8. [Debugging](#debugging)
9. [Best Practices](#best-practices)

---

## Getting Started

This guide will help you set up and start developing the Hyperion Coordinator UI.

### Quick Start

```bash
# Clone the repository
cd /path/to/dev-squad/ui

# Install dependencies
npm install

# Start development server
npm run dev

# Open browser to http://localhost:5173
```

---

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| **Node.js** | 18+ | JavaScript runtime |
| **npm** | 8+ | Package manager |
| **Git** | 2.0+ | Version control |

### Recommended Tools

| Tool | Purpose |
|------|---------|
| **VS Code** | Code editor with TypeScript support |
| **React Developer Tools** | Browser extension for React debugging |
| **Playwright VS Code Extension** | E2E test debugging |

### VS Code Extensions

Install these extensions for the best development experience:

```json
{
  "recommendations": [
    "dbaeumer.vscode-eslint",
    "esbenp.prettier-vscode",
    "bradlc.vscode-tailwindcss",
    "ms-playwright.playwright",
    "formulahendry.auto-rename-tag",
    "christian-kohler.path-intellisense"
  ]
}
```

---

## Installation

### Step 1: Install Dependencies

```bash
cd ui
npm install
```

This installs all dependencies from `package.json`:
- React 19.1.1
- TypeScript 5.8.3
- Vite 7.1.7
- Material-UI 7.3.2
- Playwright 1.55.1
- And 40+ other packages

### Step 2: Verify Installation

```bash
# Check Node.js version
node --version  # Should be v18+

# Check npm version
npm --version   # Should be v8+

# Verify TypeScript
npx tsc --version  # Should be 5.8.3
```

### Step 3: Install Playwright Browsers

```bash
# Install Playwright browsers for testing
npx playwright install

# Or install specific browsers
npx playwright install chromium
```

---

## Development Workflow

### Starting Development Server

```bash
npm run dev
```

**Output**:
```
  VITE v7.1.7  ready in 423 ms

  ➜  Local:   http://localhost:5173/ui/
  ➜  Network: use --host to expose
  ➜  press h + enter to show help
```

**Features**:
- Hot Module Replacement (HMR) for instant updates
- TypeScript compilation
- API proxy to MCP Bridge (configured in `vite.config.ts`)

### Running Tests

```bash
# Run all Playwright tests
npm test

# Run in headed mode (see browser)
npm run test:headed

# Run interactive UI mode
npm run test:ui

# Run unit tests (Vitest)
npm run test:unit

# Run unit tests in watch mode
npm run test:unit:ui
```

### Linting

```bash
# Check for linting errors
npm run lint

# Auto-fix linting errors
npm run lint -- --fix
```

### Building for Production

```bash
# TypeScript check + Vite build
npm run build

# Output goes to dist/
```

### Preview Production Build

```bash
# Build and preview
npm run build
npm run preview
```

---

## Project Structure

```
ui/
├── src/
│   ├── components/          # React components
│   │   ├── code/           # Code search components
│   │   └── knowledge/      # Knowledge base components
│   ├── pages/              # Page components (routes)
│   ├── services/           # API clients
│   ├── types/              # TypeScript type definitions
│   ├── theme.ts            # Material-UI theme
│   ├── App.tsx             # Main app component
│   └── main.tsx            # Entry point
├── tests/                  # Playwright E2E tests
│   ├── kanban/            # Kanban board tests
│   └── fixtures/          # Test data
├── public/                 # Static assets
├── dist/                   # Build output (gitignored)
├── docs/                   # Documentation
├── vite.config.ts         # Vite configuration
├── tsconfig.json          # TypeScript configuration
├── playwright.config.ts   # Playwright configuration
├── tailwind.config.js     # Tailwind CSS configuration
├── eslint.config.js       # ESLint configuration
└── package.json           # Dependencies and scripts
```

### Key Files

| File | Purpose |
|------|---------|
| `src/App.tsx` | Main application with routing and layout |
| `src/theme.ts` | Material-UI theme (light/dark modes) |
| `src/services/restClient.ts` | Primary REST API client |
| `src/services/chatService.ts` | Chat and WebSocket service |
| `src/types/coordinator.ts` | Type definitions for tasks, knowledge |
| `vite.config.ts` | Vite dev server and build configuration |
| `tailwind.config.js` | Tailwind CSS customization |

---

## Coding Standards

### TypeScript

**Always use TypeScript**. No `.js` or `.jsx` files.

```typescript
// ✅ CORRECT - Typed component
interface MyComponentProps {
  title: string;
  count: number;
  onAction: (id: string) => void;
}

const MyComponent: React.FC<MyComponentProps> = ({ title, count, onAction }) => {
  return <div>{title}: {count}</div>;
};

// ❌ WRONG - Untyped props
const MyComponent = ({ title, count, onAction }) => {  // TypeScript error
  return <div>{title}: {count}</div>;
};
```

### Component File Structure

```typescript
// 1. Imports (React first, then libraries, then local)
import React, { useState, useEffect } from 'react';
import { Card, CardContent, Button } from '@mui/material';
import { restClient } from '@/services/restClient';

// 2. Type definitions
interface MyComponentProps {
  // Props
}

interface LocalState {
  // Local state types
}

// 3. Component definition
export const MyComponent: React.FC<MyComponentProps> = ({
  prop1,
  prop2,
}) => {
  // 4. State hooks
  const [state, setState] = useState<LocalState>({});

  // 5. Effect hooks
  useEffect(() => {
    // Side effects
  }, [dependencies]);

  // 6. Event handlers
  const handleClick = () => {
    // Handler logic
  };

  // 7. Render logic
  return (
    <Card>
      <CardContent>
        {/* JSX */}
      </CardContent>
    </Card>
  );
};

// 8. Default export (optional)
export default MyComponent;
```

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| **Components** | PascalCase | `TaskCard`, `ChatMessageView` |
| **Files** | PascalCase.tsx | `TaskCard.tsx` |
| **Functions** | camelCase | `handleClick`, `fetchTasks` |
| **Variables** | camelCase | `userName`, `taskList` |
| **Constants** | UPPER_SNAKE_CASE | `MAX_RETRIES`, `API_BASE_URL` |
| **Types/Interfaces** | PascalCase | `TaskData`, `UserProfile` |

### Import Order

1. React imports
2. Third-party libraries
3. Material-UI components
4. Local services
5. Local components
6. Local types
7. Local utilities

```typescript
// ✅ CORRECT - Organized imports
import React, { useState } from 'react';
import { Card, CardContent } from '@mui/material';
import { restClient } from '@/services/restClient';
import { TaskCard } from './TaskCard';
import type { Task } from '@/types/coordinator';
import { formatDate } from '@/utils/date';

// ❌ WRONG - Random order
import { formatDate } from '@/utils/date';
import type { Task } from '@/types/coordinator';
import React, { useState } from 'react';
```

### API Integration Rules

**CRITICAL**: Always use service layer, never direct MCP calls.

```typescript
// ✅ CORRECT - Use REST client
import { restClient } from '@/services/restClient';
const tasks = await restClient.listHumanTasks();

// ❌ FORBIDDEN - Direct MCP import (ESLint error)
import { mcpClient } from '@/services/mcpClient';
const tasks = await mcpClient.listHumanTasks();  // ESLint error!
```

---

## Common Tasks

### Creating a New Page

1. Create page component in `src/pages/`:

```typescript
// src/pages/MyNewPage.tsx
import React from 'react';
import { Box, Typography } from '@mui/material';

export const MyNewPage: React.FC = () => {
  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4">My New Page</Typography>
    </Box>
  );
};

export default MyNewPage;
```

2. Add route in `src/App.tsx`:

```typescript
import { MyNewPage } from './pages/MyNewPage';

// Inside <Routes>
<Route path="/my-new-page" element={<MyNewPage />} />
```

3. Add navigation in `App.tsx`:

```typescript
const navigationItems = [
  // ... existing items
  { path: '/my-new-page', label: 'My Page', icon: <Dashboard /> },
];
```

### Creating a New Component

1. Create component file:

```typescript
// src/components/MyComponent.tsx
import React from 'react';
import { Card, CardContent, Typography } from '@mui/material';

interface MyComponentProps {
  title: string;
  description: string;
}

export const MyComponent: React.FC<MyComponentProps> = ({
  title,
  description,
}) => {
  return (
    <Card>
      <CardContent>
        <Typography variant="h6">{title}</Typography>
        <Typography variant="body2">{description}</Typography>
      </CardContent>
    </Card>
  );
};

export default MyComponent;
```

2. Import and use:

```typescript
import { MyComponent } from '@/components/MyComponent';

<MyComponent title="Hello" description="World" />
```

### Adding a New API Method

1. Add method to appropriate service:

```typescript
// src/services/restClient.ts
async myNewMethod(param: string): Promise<ResultType> {
  return await this.fetchJSON<ResultType>(`/my-endpoint/${param}`, {
    method: 'POST',
    body: JSON.stringify({ data: param }),
  });
}
```

2. Define types in `src/types/coordinator.ts`:

```typescript
export interface ResultType {
  id: string;
  value: string;
}
```

3. Use in component:

```typescript
const result = await restClient.myNewMethod('param-value');
```

### Adding Material-UI Component

Material-UI is already configured. Just import and use:

```typescript
import {
  Card,
  CardHeader,
  CardContent,
  CardActions,
  Button,
  Typography,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';

<Card>
  <CardHeader title="My Card" />
  <CardContent>
    <Typography>Content here</Typography>
  </CardContent>
  <CardActions>
    <Button startIcon={<AddIcon />}>Add</Button>
  </CardActions>
</Card>
```

### Adding Tailwind CSS Classes

Tailwind CSS is configured with custom utilities:

```typescript
// Use Tailwind classes directly
<div className="flex flex-col gap-4 p-6 bg-primary-50 dark:bg-gray-900">
  <h1 className="text-2xl font-bold text-primary-600">Title</h1>
  <p className="text-base text-gray-700 dark:text-gray-300">Text</p>
</div>

// Use mobile-specific utilities
<div className="touch-target mobile-only">
  Mobile-only button with touch-friendly size
</div>
```

---

## Debugging

### React Developer Tools

1. Install [React DevTools](https://react.dev/learn/react-developer-tools)
2. Open DevTools → Components tab
3. Inspect component props and state

### Vite Dev Server Debugging

```typescript
// Add console.log in components
useEffect(() => {
  console.log('Component mounted', { props, state });
}, []);

// View logs in browser console
```

### Network Debugging

1. Open DevTools → Network tab
2. Filter by Fetch/XHR
3. Inspect API requests/responses
4. Look for `/api/v1/*` calls

### TypeScript Errors

```bash
# Check TypeScript errors
npx tsc --noEmit

# Or use VS Code
# Errors show in Problems panel
```

### ESLint Errors

```bash
# Check linting errors
npm run lint

# View errors and fix automatically
npm run lint -- --fix
```

### Playwright Test Debugging

```bash
# Run tests in debug mode
npm run test:debug

# Or use VS Code extension:
# 1. Install "Playwright Test for VSCode"
# 2. Click "Debug Test" in test file
```

### WebSocket Debugging

```typescript
// Enable WebSocket logging in chatService.ts
ws.onopen = () => {
  console.log('[WebSocket] Connected');
};

ws.onmessage = (event) => {
  console.log('[WebSocket] Message:', event.data);
};

ws.onerror = (error) => {
  console.error('[WebSocket] Error:', error);
};
```

---

## Best Practices

### 1. Component Design

**Keep components small and focused**:

```typescript
// ✅ GOOD - Single responsibility
const TaskList = ({ tasks }) => (
  <div>
    {tasks.map(task => (
      <TaskCard key={task.id} task={task} />
    ))}
  </div>
);

// ❌ BAD - Too many responsibilities
const TaskList = ({ tasks, onEdit, onDelete, filters, sorting }) => {
  // Too much logic here
};
```

### 2. State Management

**Use local state when possible**:

```typescript
// ✅ GOOD - Local state for UI
const [isOpen, setIsOpen] = useState(false);

// ✅ GOOD - API data fetching
const [tasks, setTasks] = useState<Task[]>([]);

useEffect(() => {
  restClient.listHumanTasks().then(setTasks);
}, []);
```

### 3. API Calls

**Always handle loading and error states**:

```typescript
// ✅ GOOD - Complete handling
const [loading, setLoading] = useState(true);
const [error, setError] = useState<Error | null>(null);

useEffect(() => {
  const fetchData = async () => {
    try {
      setLoading(true);
      const result = await restClient.listHumanTasks();
      setTasks(result);
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

### 4. Type Safety

**Always define types for props and state**:

```typescript
// ✅ GOOD - Explicit types
interface TaskCardProps {
  task: Task;
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
}

const [selectedTask, setSelectedTask] = useState<Task | null>(null);

// ❌ BAD - Implicit any
const TaskCard = ({ task, onEdit }) => {  // task and onEdit are 'any'
  // ...
};
```

### 5. Performance

**Optimize re-renders with useMemo and useCallback**:

```typescript
// Expensive computation
const filteredTasks = useMemo(() => {
  return tasks.filter(task => task.status === 'pending');
}, [tasks]);

// Callback passed to child
const handleEdit = useCallback((id: string) => {
  // Edit logic
}, [dependency]);
```

### 6. Accessibility

**Always include ARIA labels and keyboard support**:

```typescript
<IconButton
  onClick={handleDelete}
  aria-label="Delete task"
>
  <Delete />
</IconButton>

<TextField
  label="Task Title"
  aria-describedby="task-title-help"
/>
<Typography id="task-title-help" variant="caption">
  Enter a descriptive title for your task
</Typography>
```

### 7. Error Boundaries

**Wrap components in error boundaries for production**:

```typescript
import { ErrorBoundary } from 'react-error-boundary';

<ErrorBoundary
  fallback={<div>Something went wrong</div>}
  onError={(error, errorInfo) => {
    console.error('Error caught:', error, errorInfo);
  }}
>
  <MyComponent />
</ErrorBoundary>
```

### 8. Code Organization

**Group related files together**:

```
components/
├── tasks/
│   ├── TaskCard.tsx
│   ├── TaskList.tsx
│   ├── TaskDetail.tsx
│   └── index.ts  # Re-export all
```

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture
- [Component Catalog](./COMPONENTS.md) - Component reference
- [API Integration Guide](./API_INTEGRATION.md) - API usage
- [Testing Guide](./TESTING.md) - Testing strategies
- [Deployment Guide](./DEPLOYMENT.md) - Build and deploy
- [Troubleshooting](./TROUBLESHOOTING.md) - Common issues

---

**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
