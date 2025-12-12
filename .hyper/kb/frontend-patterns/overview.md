# ## Overview

**Collection:** frontend-patterns
**Created:** 2025-11-20

---

## Overview

Hyperion's frontend is built with modern React 19, Vite, and TypeScript, featuring Radix UI components, Tailwind CSS styling, and a service-oriented architecture for API communication.

## Core Technologies

### Frontend Framework

**Location:** `/ui/package.json`

| Technology | Version | Purpose |
|------------|---------|---------|
| React | 19.1.1 | UI framework with latest concurrent features |
| React DOM | 19.1.1 | DOM rendering |
| TypeScript | 5.8.3 | Type-safe development |
| Vite | 7.1.7 | Build tool and dev server |

### Why React 19?

- **Concurrent Rendering:** Improved performance with automatic batching
- **Server Components:** Prepare for future SSR capabilities
- **Transitions API:** Smooth UI updates without blocking
- **useOptimistic Hook:** Optimistic UI updates

## Build Configuration

### Vite Setup

**File:** `/ui/vite.config.ts`

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],

  // Always served through Go proxy at /ui/
  base: '/ui/',

  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@atoms': path.resolve(__dirname, './src/components/atoms'),
      '@molecules': path.resolve(__dirname, './src/components/molecules'),
      '@organisms': path.resolve(__dirname, './src/components/organisms'),
      '@templates': path.resolve(__dirname, './src/components/templates'),
      '@pages': path.resolve(__dirname, './src/pages'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@services': path.resolve(__dirname, './src/services'),
      '@types': path.resolve(__dirname, './src/types'),
      '@utils': path.resolve(__dirname, './src/utils'),
    },
  },

  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_BACKEND_URL || 'http://localhost:7095',
        changeOrigin: true,
      },
    },
  },

  build: {
    outDir: 'dist',
    sourcemap: true,
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['react', 'react-dom'],
          'ui': ['@radix-ui/react-dialog', '@radix-ui/react-dropdown-menu'],
        },
      },
    },
  },
})
```

### Path Aliases

**Benefits:**
- Clean imports: `import Button from '@atoms/Button'` instead of `'../../../components/atoms/Button'`
- Easier refactoring
- Atomic design structure enforcement

**Usage Example:**
```typescript
// Instead of relative paths
import { Button } from '../../../components/atoms/Button'
import { ChatInput } from '../../molecules/ChatInput'

// Use aliases
import { Button } from '@atoms/Button'
import { ChatInput } from '@molecules/ChatInput'
```

### TypeScript Configuration

**File:** `/ui/tsconfig.json`

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "paths": {
      "@/*": ["./src/*"],
      "@atoms/*": ["./src/components/atoms/*"],
      "@molecules/*": ["./src/components/molecules/*"],
      "@organisms/*": ["./src/components/organisms/*"],
      "@templates/*": ["./src/components/templates/*"],
      "@pages/*": ["./src/pages/*"],
      "@hooks/*": ["./src/hooks/*"],
      "@services/*": ["./src/services/*"],
      "@types/*": ["./src/types/*"],
      "@utils/*": ["./src/utils/*"]
    }
  }
}
```

## UI Component Library

### Radix UI

**Why Radix UI?**
- **Unstyled/Headless:** Full styling control
- **Accessibility:** WCAG 2.1 compliant out of the box
- **Composable:** Build complex components from primitives
- **TypeScript:** First-class TypeScript support

### Installed Components

| Component | Version | Purpose |
|-----------|---------|---------|
| `@radix-ui/react-dialog` | 1.1.4 | Modals and dialogs |
| `@radix-ui/react-dropdown-menu` | 2.1.4 | Dropdown menus |
| `@radix-ui/react-select` | 2.1.4 | Select dropdowns |
| `@radix-ui/react-tabs` | 1.1.2 | Tab navigation |
| `@radix-ui/react-tooltip` | 1.1.7 | Tooltips |
| `@radix-ui/react-accordion` | 1.2.2 | Accordions |
| `@radix-ui/react-popover` | 1.1.4 | Popovers |
| `@radix-ui/react-slider` | 1.2.2 | Range sliders |
| `@radix-ui/react-switch` | 1.1.3 | Toggle switches |
| `@radix-ui/react-checkbox` | 1.1.4 | Checkboxes |
| `@radix-ui/react-radio-group` | 1.2.2 | Radio buttons |

### Usage Example

```typescript
import * as Dialog from '@radix-ui/react-dialog'

export function ConfirmDialog({ open, onClose, onConfirm }) {
  return (
    <Dialog.Root open={open} onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 bg-black/50" />
        <Dialog.Content className="fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 bg-white p-6 rounded-lg">
          <Dialog.Title className="text-xl font-bold">
            Confirm Action
          </Dialog.Title>
          <Dialog.Description className="mt-2 text-gray-600">
            Are you sure you want to proceed?
          </Dialog.Description>
          <div className="mt-4 flex gap-2 justify-end">
            <button onClick={onClose} className="px-4 py-2 bg-gray-200 rounded">
              Cancel
            </button>
            <button onClick={onConfirm} className="px-4 py-2 bg-blue-500 text-white rounded">
              Confirm
            </button>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
```

## Styling System

### Tailwind CSS

**Version:** 3.4.18

**Configuration File:** `/ui/tailwind.config.js`

```javascript
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#3B82F6',
        secondary: '#10B981',
        danger: '#EF4444',
        // ... custom colors
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
    },
  },
  plugins: [],
  darkMode: 'class', // Enable dark mode via class
}
```

### Utility Libraries

| Package | Version | Purpose |
|---------|---------|---------|
| `tailwind-merge` | 2.5.5 | Merge Tailwind classes without conflicts |
| `class-variance-authority` | 0.7.1 | Type-safe component variants |
| `clsx` | 2.1.1 | Conditional className utility |

### Component Variant Pattern

**Using CVA (Class Variance Authority):**

```typescript
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@utils/cn'

const buttonVariants = cva(
  // Base styles
  "inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none disabled:opacity-50",
  {
    variants: {
      variant: {
        primary: "bg-blue-500 text-white hover:bg-blue-600",
        secondary: "bg-gray-200 text-gray-900 hover:bg-gray-300",
        danger: "bg-red-500 text-white hover:bg-red-600",
      },
      size: {
        sm: "h-8 px-3 text-sm",
        md: "h-10 px-4",
        lg: "h-12 px-6 text-lg",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  }
)

interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return (
    <button
      className={cn(buttonVariants({ variant, size }), className)}
      {...props}
    />
  )
}
```

### Utility Function: cn()

**File:** `/ui/src/utils/cn.ts`

```typescript
import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

**Usage:**
```typescript
<div className={cn(
  "base-class",
  condition && "conditional-class",
  anotherCondition ? "true-class" : "false-class"
)} />
```

## Icon Library

### Lucide React

**Version:** 0.552.0

**Why Lucide?**
- Extensive icon set (1000+ icons)
- Tree-shakeable (only import what you use)
- Consistent design
- TypeScript support

**Usage:**
```typescript
import { FileText, Search, Settings, User } from 'lucide-react'

function Toolbar() {
  return (
    <div className="flex gap-2">
      <button><FileText size={20} /></button>
      <button><Search size={20} /></button>
      <button><Settings size={20} /></button>
      <button><User size={20} /></button>
    </div>
  )
}
```

## Animation Library

### Framer Motion

**Version:** 12.23.24

**Why Framer Motion?**
- Declarative animations
- Spring physics
- Layout animations
- Gesture support

**Usage Example:**
```typescript
import { motion } from 'framer-motion'

function AnimatedCard() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.3 }}
      className="card"
    >
      Content
    </motion.div>
  )
}
```

## State Management

### No Global State Library

**Philosophy:** Use React's built-in features for state management

**Patterns Used:**

#### 1. Component State (useState)

```typescript
function SearchInput() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])

  return (
    <input
      value={query}
      onChange={(e) => setQuery(e.target.value)}
    />
  )
}
```

#### 2. Context API for Shared State

```typescript
// contexts/ConversationModeContext.tsx
const ConversationModeContext = createContext<{
  mode: 'chat' | 'code';
  setMode: (mode: 'chat' | 'code') => void;
}>({ mode: 'chat', setMode: () => {} })

export function ConversationModeProvider({ children }) {
  const [mode, setMode] = useState<'chat' | 'code'>('chat')

  return (
    <ConversationModeContext.Provider value={{ mode, setMode }}>
      {children}
    </ConversationModeContext.Provider>
  )
}

export const useConversationMode = () => useContext(ConversationModeContext)
```

#### 3. Custom Hooks for Logic

```typescript
// hooks/useStreamingPerformance.ts
export function useStreamingPerformance() {
  const [fps, setFps] = useState(0)
  const [latency, setLatency] = useState(0)

  useEffect(() => {
    // Performance monitoring logic
  }, [])

  return { fps, latency }
}
```

### Custom Hooks

**Location:** `/ui/src/hooks/`

| Hook | Purpose |
|------|---------|
| `useStreamingPerformance` | Monitor WebSocket streaming performance |
| `useKeyboardShortcuts` | Global keyboard shortcut management |
| `useLocalStorage` | Persist state to localStorage |
| `useDebounce` | Debounce input values |
| `useMediaQuery` | Responsive design breakpoints |

## API Client Architecture

### Service Layer Pattern

**Location:** `/ui/src/services/`

| Service | Purpose |
|---------|---------|
| `restClient.ts` | Coordinator API (tasks, knowledge) |
| `codeClient.ts` | Code search and indexing API |
| `knowledgeService.ts` | Knowledge base operations |
| `mcpService.ts` | MCP protocol communication |
| `chatService.ts` | Chat and WebSocket |

### REST Client Example

**File:** `/ui/src/services/restClient.ts`

```typescript
const BASE_URL = '/api/v1'

class RestClient {
  private async fetchJSON<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const response = await fetch(`${BASE_URL}${endpoint}`, {
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
      ...options,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: 'Unknown error' }))
      throw new Error(error.message || `HTTP ${response.status}`)
    }

    return response.json()
  }

  // Human Tasks
  async listHumanTasks(): Promise<HumanTask[]> {
    return this.fetchJSON<HumanTask[]>('/tasks')
  }

  async createHumanTask(prompt: string): Promise<{ taskId: string }> {
    return this.fetchJSON('/tasks', {
      method: 'POST',
      body: JSON.stringify({ prompt }),
    })
  }

  // Agent Tasks
  async listAgentTasks(agentName?: string): Promise<AgentTask[]> {
    const query = agentName ? `?agentName=${encodeURIComponent(agentName)}` : ''
    return this.fetchJSON<AgentTask[]>(`/agent-tasks${query}`)
  }

  async updateTodoStatus(
    taskId: string,
    todoId: string,
    status: TodoStatus,
    notes?: string
  ): Promise<void> {
    return this.fetchJSON(`/agent-tasks/${taskId}/todos/${todoId}/status`, {
      method: 'PUT',
      body: JSON.stringify({ status, notes }),
    })
  }
}

export const restClient = new RestClient()
```

### Error Handling Pattern

```typescript
// utils/errorHandler.ts
export function handleApiError(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }

  if (typeof error === 'string') {
    return error
  }

  return 'An unexpected error occurred'
}

// Usage in component
try {
  await restClient.createHumanTask(prompt)
} catch (error) {
  const message = handleApiError(error)
  toast.error(message)
}
```

## Development Tools

### Package Manager

**npm** (comes with Node.js)

### Development Scripts

**File:** `/ui/package.json`

```json
{
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build",
    "preview": "vite preview",
    "lint": "eslint . --ext ts,tsx --report-unused-disable-directives --max-warnings 0",
    "type-check": "tsc --noEmit"
  }
}
```

### Testing

**Framework:** Playwright v1.42.1

**Location:** `/ui/tests/`

**Configuration:** `/ui/playwright.config.ts`

```typescript
export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:4097/ui',
    trace: 'on-first-retry',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
  ],
})
```

## Performance Optimization

### Code Splitting

```typescript
// Lazy load heavy components
const CodeEditor = lazy(() => import('@organisms/CodeEditor'))
const FileInspector = lazy(() => import('@organisms/FileInspector'))

function App() {
  return (
    <Suspense fallback={<LoadingSpinner />}>
      <CodeEditor />
    </Suspense>
  )
}
```

### Memoization

```typescript
import { memo, useMemo, useCallback } from 'react'

// Memoize expensive computations
const ExpensiveComponent = memo(function ExpensiveComponent({ data }) {
  const processedData = useMemo(() => {
    return data.map(item => complexProcessing(item))
  }, [data])

  const handleClick = useCallback(() => {
    // Handler logic
  }, [])

  return <div onClick={handleClick}>{processedData}</div>
})
```

### Virtual Scrolling

**For large lists, consider using:**
- `react-window` - Windowing for large lists
- `react-virtualized` - Advanced virtualization

## Related Documents

- [API Service Layer](./api-service-layer.md) - REST endpoints and client patterns
- [Component Architecture](./component-architecture.md) - Atomic design structure
- [Data Contracts](./data-contracts.md) - TypeScript type definitions
- [Configuration Reference](./configuration-reference.md) - Environment variables

## Best Practices

### 1. Component Structure

```typescript
// ✅ GOOD - Clear, focused component
interface ButtonProps {
  variant?: 'primary' | 'secondary'
  size?: 'sm' | 'md' | 'lg'
  children: React.ReactNode
  onClick?: () => void
}

export function Button({ variant = 'primary', size = 'md', children, onClick }: ButtonProps) {
  return (
    <button className={cn(buttonVariants({ variant, size }))} onClick={onClick}>
      {children}
    </button>
  )
}
```

### 2. Type Safety

```typescript
// ✅ GOOD - Explicit types
interface User {
  id: string
  name: string
  email: string
}

function UserProfile({ user }: { user: User }) {
  // Type-safe access
  return <div>{user.name}</div>
}

// ❌ BAD - Any types
function UserProfile({ user }: { user: any }) {
  return <div>{user.name}</div>
}
```

### 3. Error Boundaries

```typescript
class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { hasError: boolean }
> {
  state = { hasError: false }

  static getDerivedStateFromError() {
    return { hasError: true }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('Error caught by boundary:', error, errorInfo)
  }

  render() {
    if (this.state.hasError) {
      return <ErrorFallback />
    }

    return this.props.children
  }
}
```

### 4. Accessibility

```typescript
// ✅ GOOD - Accessible button
<button
  aria-label="Close dialog"
  onClick={onClose}
  className="..."
>
  <X size={20} aria-hidden="true" />
</button>

// ✅ GOOD - Semantic HTML
<nav aria-label="Main navigation">
  <ul role="list">
    <li><a href="/tasks">Tasks</a></li>
    <li><a href="/knowledge">Knowledge</a></li>
  </ul>
</nav>
```
