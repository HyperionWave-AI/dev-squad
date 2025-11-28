# How to Structure React Components with Atomic Design

**Collection:** howto
**Tags:** react, typescript, atomic-design, frontend, components
**Version:** 1.0
**Last Updated:** 2025-11-21

---

## Overview

This guide explains how to organize React components using Atomic Design principles - building UIs from smallest building blocks (atoms) to complete pages (templates). You'll learn component hierarchy, file organization, and reusability patterns used in Hyperion.

## Prerequisites

- React 18+ and TypeScript
- Understanding of component composition
- Familiarity with [UI Client Stack](../ui-client-stack.md)
- Knowledge of [React Architecture](../frontend-patterns/react-architecture.md)

## When to Use This Guide

- Organizing component structure for new React projects
- Refactoring existing component hierarchy
- Building reusable component libraries
- Maintaining consistent UI patterns

---

## Atomic Design Hierarchy

```
Atoms → Molecules → Organisms → Templates → Pages
  ↓         ↓           ↓            ↓         ↓
Single   Combined   Complex      Layouts    Routes
Purpose   Atoms     Features
```

---

## Steps

### Step 1: Set Up Directory Structure

Create organized component hierarchy:

```bash
src/
├── components/
│   ├── atoms/          # Basic building blocks
│   │   ├── Button.tsx
│   │   ├── Input.tsx
│   │   ├── Badge.tsx
│   │   ├── Icon.tsx
│   │   └── Spinner.tsx
│   ├── molecules/      # Combinations of atoms
│   │   ├── SearchBar.tsx
│   │   ├── TaskCard.tsx
│   │   ├── FormField.tsx
│   │   └── UserAvatar.tsx
│   ├── organisms/      # Complex features
│   │   ├── TaskList.tsx
│   │   ├── ChatWindow.tsx
│   │   ├── Navigation.tsx
│   │   └── Header.tsx
│   └── templates/      # Page layouts
│       ├── MainLayout.tsx
│       └── DashboardLayout.tsx
├── pages/              # Route components
│   ├── TasksPage.tsx
│   ├── ChatPage.tsx
│   └── SettingsPage.tsx
├── hooks/              # Custom hooks
├── services/           # API clients
├── types/              # TypeScript types
└── utils/              # Utility functions
```

### Step 2: Create Atoms (Basic Elements)

Build foundational, single-purpose components:

```tsx
// components/atoms/Button.tsx
import { ButtonHTMLAttributes, ReactNode } from 'react'
import { cn } from '@/utils/cn'

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  children: ReactNode
}

export function Button({ 
  variant = 'primary', 
  size = 'md', 
  className,
  children,
  ...props 
}: ButtonProps) {
  const baseStyles = 'rounded font-medium transition-colors focus:outline-none focus:ring-2'
  
  const variants = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700',
    secondary: 'bg-gray-200 text-gray-900 hover:bg-gray-300',
    danger: 'bg-red-600 text-white hover:bg-red-700',
    ghost: 'bg-transparent hover:bg-gray-100',
  }
  
  const sizes = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg',
  }
  
  return (
    <button
      className={cn(baseStyles, variants[variant], sizes[size], className)}
      {...props}
    >
      {children}
    </button>
  )
}
```

**Atom Characteristics:**
- Single purpose (one job)
- No dependencies on other components
- Highly reusable
- Minimal internal state

### Step 3: Create Molecules (Combined Atoms)

Combine atoms into functional units:

```tsx
// components/molecules/SearchBar.tsx
import { useState } from 'react'
import { Input } from '@/components/atoms/Input'
import { Button } from '@/components/atoms/Button'
import { Icon } from '@/components/atoms/Icon'

interface SearchBarProps {
  placeholder?: string
  onSearch: (query: string) => void
}

export function SearchBar({ placeholder = 'Search...', onSearch }: SearchBarProps) {
  const [query, setQuery] = useState('')
  
  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSearch(query)
  }
  
  return (
    <form onSubmit={handleSubmit} className="flex gap-2">
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={placeholder}
        className="flex-1"
      />
      <Button type="submit" size="md">
        <Icon name="search" size={20} />
      </Button>
    </form>
  )
}
```

**Molecule Characteristics:**
- Combines 2-5 atoms
- Single responsibility
- Reusable across contexts
- Minimal business logic

### Step 4: Create Organisms (Complex Features)

Build feature-rich components with business logic:

```tsx
// components/organisms/TaskList.tsx
import { useState, useEffect } from 'react'
import { TaskCard } from '@/components/molecules/TaskCard'
import { SearchBar } from '@/components/molecules/SearchBar'
import { Spinner } from '@/components/atoms/Spinner'
import { taskService } from '@/services/taskService'
import { Task } from '@/types/tasks'

interface TaskListProps {
  status?: string
}

export function TaskList({ status }: TaskListProps) {
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  
  useEffect(() => {
    loadTasks()
  }, [status])
  
  const loadTasks = async () => {
    setLoading(true)
    try {
      const data = await taskService.list({ status })
      setTasks(data)
    } catch (error) {
      console.error('Failed to load tasks:', error)
    } finally {
      setLoading(false)
    }
  }
  
  const filteredTasks = tasks.filter(task =>
    task.title.toLowerCase().includes(searchQuery.toLowerCase())
  )
  
  const handleTaskUpdate = async (taskId: string, updates: Partial<Task>) => {
    try {
      await taskService.update(taskId, updates)
      await loadTasks()
    } catch (error) {
      console.error('Failed to update task:', error)
    }
  }
  
  if (loading) {
    return (
      <div className="flex justify-center p-8">
        <Spinner size="lg" />
      </div>
    )
  }
  
  return (
    <div className="space-y-4">
      <SearchBar onSearch={setSearchQuery} placeholder="Search tasks..." />
      
      {filteredTasks.length === 0 ? (
        <p className="text-gray-500 text-center py-8">No tasks found</p>
      ) : (
        <div className="space-y-3">
          {filteredTasks.map(task => (
            <TaskCard
              key={task.id}
              task={task}
              onUpdate={(updates) => handleTaskUpdate(task.id, updates)}
            />
          ))}
        </div>
      )}
    </div>
  )
}
```

**Organism Characteristics:**
- Contains business logic
- Manages state and side effects
- Composes molecules and atoms
- Feature-specific

### Step 5: Create Templates (Page Layouts)

Define reusable page structures:

```tsx
// components/templates/MainLayout.tsx
import { ReactNode } from 'react'
import { Header } from '@/components/organisms/Header'
import { Sidebar } from '@/components/organisms/Sidebar'

interface MainLayoutProps {
  children: ReactNode
}

export function MainLayout({ children }: MainLayoutProps) {
  return (
    <div className="min-h-screen bg-gray-50">
      <Header />
      
      <div className="flex">
        <Sidebar />
        
        <main className="flex-1 p-6 max-w-7xl mx-auto">
          {children}
        </main>
      </div>
    </div>
  )
}
```

**Template Characteristics:**
- Defines page structure
- No business logic
- Receives content via children
- Composable layouts

### Step 6: Create Pages (Route Components)

Connect routes to templates with data:

```tsx
// pages/TasksPage.tsx
import { MainLayout } from '@/components/templates/MainLayout'
import { TaskList } from '@/components/organisms/TaskList'
import { Button } from '@/components/atoms/Button'
import { useNavigate } from 'react-router-dom'

export function TasksPage() {
  const navigate = useNavigate()
  
  return (
    <MainLayout>
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h1 className="text-3xl font-bold">Tasks</h1>
          <Button onClick={() => navigate('/tasks/new')}>
            Create Task
          </Button>
        </div>
        
        <TaskList />
      </div>
    </MainLayout>
  )
}
```

**Page Characteristics:**
- Mapped to routes
- Uses templates for layout
- Coordinates multiple organisms
- Minimal styling (delegates to components)

### Step 7: Add TypeScript Types

Define shared interfaces:

```tsx
// types/tasks.ts
export interface Task {
  id: string
  title: string
  status: 'pending' | 'in_progress' | 'completed'
  createdAt: string
  updatedAt: string
}

export interface TaskCreateInput {
  title: string
}

export interface TaskUpdateInput {
  status?: Task['status']
  title?: string
}
```

### Step 8: Create Custom Hooks

Extract reusable logic:

```tsx
// hooks/useTasks.ts
import { useState, useEffect } from 'react'
import { taskService } from '@/services/taskService'
import { Task } from '@/types/tasks'

export function useTasks(status?: string) {
  const [tasks, setTasks] = useState<Task[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)
  
  useEffect(() => {
    loadTasks()
  }, [status])
  
  const loadTasks = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await taskService.list({ status })
      setTasks(data)
    } catch (err) {
      setError(err as Error)
    } finally {
      setLoading(false)
    }
  }
  
  const createTask = async (input: TaskCreateInput) => {
    await taskService.create(input)
    await loadTasks()
  }
  
  const updateTask = async (id: string, updates: TaskUpdateInput) => {
    await taskService.update(id, updates)
    await loadTasks()
  }
  
  return { tasks, loading, error, createTask, updateTask, refresh: loadTasks }
}

// Usage in organism
function TaskList() {
  const { tasks, loading, updateTask } = useTasks('pending')
  // ...
}
```

---

## Best Practices

### 1. Component Naming
- Use PascalCase for components
- Be descriptive: `UserProfileCard` not `Card`
- Indicate hierarchy: `Button` (atom), `SearchBar` (molecule), `TaskList` (organism)

### 2. Props Interface
Always define explicit prop interfaces:
```tsx
interface ButtonProps {
  variant: 'primary' | 'secondary'
  onClick: () => void
  children: ReactNode
}
```

### 3. Composition Over Configuration
```tsx
// ✅ GOOD - Flexible composition
<Card>
  <CardHeader>Title</CardHeader>
  <CardBody>Content</CardBody>
</Card>

// ❌ BAD - Too many props
<Card title="Title" content="Content" showHeader={true} />
```

### 4. Single Responsibility
Each component should do ONE thing well.

### 5. Prop Drilling Avoidance
Use Context or state management for deeply nested data:
```tsx
// Context for theme
const ThemeContext = createContext<Theme | null>(null)

function useTheme() {
  const theme = useContext(ThemeContext)
  if (!theme) throw new Error('useTheme must be within ThemeProvider')
  return theme
}
```

---

## Common Pitfalls

### 1. Over-Abstraction
Don't create atoms for every HTML element. Balance reusability with simplicity.

### 2. Business Logic in Atoms/Molecules
Keep atoms and molecules pure - no API calls or complex state.

### 3. God Components
Avoid organisms that do too much. Split into smaller organisms.

### 4. Inconsistent Naming
Stick to one naming convention throughout the project.

---

## Related Documentation

- [UI Client Stack](../ui-client-stack.md) - Frontend technologies
- [React Architecture](../frontend-patterns/react-architecture.md) - Component patterns
- [Component Architecture](../component-architecture.md) - Full system view

---

## Troubleshooting

### Issue: "Component re-renders too often"

**Solution:**
```tsx
// Use memo for expensive renders
export const TaskList = memo(function TaskList({ tasks }: Props) {
  // ...
})

// Use useCallback for functions
const handleUpdate = useCallback((id: string) => {
  // ...
}, [])
```

### Issue: "Props drilling through many levels"

**Solution:**
- Use Context API for global state
- Consider state management (Zustand, Jotai)
- Colocate state closer to where it's used
