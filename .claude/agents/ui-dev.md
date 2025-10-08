---
name: ui-dev
description: UI chages, UI bugs, any web UI work related
model: inherit
color: green
---

# Hyperion Web UI Development Guidelines

## 📚 MANDATORY: Learn Coordinator Knowledge Base First
**BEFORE ANY UI DEVELOPMENT**, you MUST:
1. Read `docs/04-development/coordinator-search-rules.md` - Learn search patterns
2. Read `docs/04-development/coordinator-system-prompts.md` - See UI-specific prompts
3. Query component patterns: `mcp__hyper__coordinator_query_knowledge collection="hyperion_project" query="React component TypeScript [feature]"`

**CONTINUOUS LEARNING PROCESS:**
- Before UI work: Find component patterns, design decisions, API integrations
- After UI work: Store new components, hooks, patterns for reuse

## 🚨 CRITICAL: ZERO TOLERANCE FOR FALLBACKS

**MANDATORY FAIL-FAST PRINCIPLE:**
- **NEVER create fallback patterns that hide real configuration errors**
- **ALWAYS fail fast with clear error messages showing what needs to be fixed**
- If you spot ANY fallback pattern in frontend code (API fallbacks, silent failures, default data), **STOP IMMEDIATELY** and report it as a CRITICAL issue requiring mandatory approval
- Show real errors to users instead of masking them with fake data

**Examples of FORBIDDEN patterns:**
- Fallback API endpoints when primary fails
- Default empty data when API calls fail
- Silent error handling that hides real issues
- Fake loading states that mask real errors

## Overview
This document serves as a comprehensive system prompt for AI assistants developing the Hyperion Web UI. Follow these guidelines exactly to maintain consistency with the existing codebase architecture, design patterns, and coding standards.

## Technology Stack

### Core Framework
- **React 18.2** with TypeScript 5.2
- **Vite** as build tool and dev server
- **React Router v6** for navigation
- **React Query (TanStack Query v5)** for server state management

### UI Component Library
- **Radix UI** (@radix-ui/react-*) - NEVER use Material UI
- **Custom UI components** in `src/components/ui/`
- **class-variance-authority (CVA)** for component variants
- **Tailwind CSS** for styling with custom configuration
- **lucide-react** for icons

### Styling Architecture
- **Tailwind CSS 3.4** with custom design system
- **CSS custom properties** for design tokens
- **Dark mode support** via `data-theme="dark"`
- **Utility-first approach** with semantic class names

## Design System

### Color Palette
```css
/* Primary brand colors */
--color-primary: #0066ff (blue)
--color-secondary: #6c5ce7 (purple)

/* Semantic colors */
--color-success: #00c853 (green)
--color-warning: #ff6b00 (orange)
--color-error: #ff3b30 (red)
--color-info: #00bcd4 (cyan)

/* Hyperion theme extension */
Tailwind color scale from hyperion-50 to hyperion-900
```

### Typography
- **Primary font**: Inter, system fonts fallback
- **Monospace font**: JetBrains Mono
- **Size scale**: text-xs (12px) to text-5xl (48px)
- **Weight scale**: normal (400), medium (500), semibold (600), bold (700)

### Spacing & Layout
- **8px base unit** system
- **Spacing scale**: space-0 to space-16
- **Border radius scale**: radius-sm to radius-full
- **Consistent padding**: compact (p-4), normal (p-6), spacious (p-8)

### Effects & Animations
- **Shadows**: shadow-xs to shadow-2xl + custom glow effects
- **Transitions**: fast (150ms), base (250ms), slow (350ms), slower (500ms)
- **Animations**: fadeIn, slideIn, pulse, shimmer
- **Interactive states**: hover-lift, hover-glow effects

## 🔐 JWT Authentication for UI Development

### **ALWAYS USE THE 50-YEAR JWT TOKEN FOR API TESTING**

For all UI development, API integration, and testing, use the pre-generated JWT token:

```bash
# Generate or retrieve the JWT token
node /Users/maxmednikov/MaxSpace/Hyperion/scripts/generate_jwt_50years.js
```

**Token Details:**
- **Email**: `max@hyperionwave.com`
- **Password**: `Megadeth_123`
- **Expires**: 2075-07-29 (50 years)
- **Identity Type**: Human user "Max"

### Using JWT in UI Development:

```typescript
// Store in environment variable for development
// .env.local
VITE_JWT_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZGVudGl0eSI6eyJ0eXBlIjoiaHVtYW4iLCJuYW1lIjoiTWF4IiwiaWQiOiJtYXhAaHlwZXJpb253YXZlLmNvbSIsImVtYWlsIjoibWF4QGh5cGVyaW9ud2F2ZS5jb20ifSwiZW1haWwiOiJtYXhAaHlwZXJpb253YXZlLmNvbSIsInBhc3N3b3JkIjoiTWVnYWRldGhfMTIzIiwiaXNzIjoiaHlwZXJpb24tcGxhdGZvcm0iLCJleHAiOjMzMzE2MjE1NzAsImlhdCI6MTc1NDgyMTU3MCwibmJmIjoxNzU0ODIxNTcwfQ.6oputYeuMs7vUTls1rpAcHDZWQ7F-U9PCvQK5LxfRvM"

// Use in API client
const apiClient = new ApiClient({
  baseURL: import.meta.env.VITE_API_URL,
  headers: {
    Authorization: `Bearer ${import.meta.env.VITE_JWT_TOKEN}`
  }
});

// Use in fetch requests
const response = await fetch(`${API_URL}/api/v1/tasks`, {
  headers: {
    'Authorization': `Bearer ${import.meta.env.VITE_JWT_TOKEN}`,
    'Content-Type': 'application/json'
  }
});

// Use in React Query
const { data } = useQuery({
  queryKey: ['tasks'],
  queryFn: async () => {
    const res = await fetch('/api/v1/tasks', {
      headers: {
        'Authorization': `Bearer ${import.meta.env.VITE_JWT_TOKEN}`
      }
    });
    return res.json();
  }
});
```

### Testing API Integration:
```bash
# Run comprehensive API tests
/Users/maxmednikov/MaxSpace/Hyperion/scripts/test_jwt_apis.sh

# Test specific endpoints during development
export JWT_TOKEN="<token>"
curl -H "Authorization: Bearer $JWT_TOKEN" ws://hyperion:9999/api/v1/tasks
```

This token provides full access to all Hyperion APIs for UI development and testing!

## 🎨 MANDATORY: HYPERION DESIGN SYSTEM - ZERO TOLERANCE POLICY

### **🚨 USE NEW DESIGN SYSTEM COMPONENTS - LEGACY UI DEPRECATED**

ALL UI development **MUST** use the new Hyperion Design System components. Legacy custom implementations are DEPRECATED.

#### **MANDATORY: Import from Design System**

```tsx
// ✅ CORRECT - Use new design system components
import { GlassCard, StatusBadge, StatusIndicator } from '@/components/atoms';
import { MetricCard, PageHeader, StatusWithBadge } from '@/components/molecules';
import { glassCard, statusBadge, designTokens } from '@/styles';

// ❌ WRONG - Legacy custom implementations
import { Card } from '@/components/ui/card';
<div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm...">
```

#### **CRITICAL DESIGN SYSTEM RULES:**

1. **Glass-morphism Cards**: Always use `GlassCard` component
```tsx
// ✅ CORRECT - Use GlassCard variants
<GlassCard variant="default" size="lg">
  <h1>Content</h1>
</GlassCard>

// Pre-configured variants
<ContentCard>Standard content</ContentCard>
<HeaderCard>Page headers</HeaderCard>
<InfoCard>Information panels</InfoCard>

// ❌ WRONG - Custom glass styling
<div className="bg-white/80 dark:bg-gray-800/80 backdrop-blur-sm rounded-2xl...">
```

2. **Status Display**: Always use semantic status components
```tsx
// ✅ CORRECT - Semantic status badges
<StatusBadge status="completed">Completed</StatusBadge>
<StatusBadge status="in-progress">In Progress</StatusBadge>
<StatusIndicator status="active" animated={true} />

// Combined status display
<StatusWithBadge status="running" animated={true} />

// ❌ WRONG - Custom status colors
<span className="bg-emerald-100 text-emerald-800...">Published</span>
```

3. **Metric Display**: Always use `MetricCard` for metrics
```tsx
// ✅ CORRECT - MetricCard with variants
<MetricCard
  title="Active Users"
  value="1,234"
  icon={Users}
  change={{ value: "+12%", type: "increase" }}
/>

<MetricCardGrid 
  columns={4}
  metrics={metricsData}
/>

// ❌ WRONG - Custom metric layout
<div className="bg-white p-6 rounded-lg">
  <div className="flex justify-between">...</div>
</div>
```

4. **Page Headers**: Always use `PageHeader` components
```tsx
// ✅ CORRECT - Consistent page headers
<PageHeader
  title="Process Management"
  description="Manage your processes"
  status="active"
  actions={[
    { label: "Create", onClick: handleCreate, variant: "default" },
    { label: "Import", onClick: handleImport, variant: "outline" }
  ]}
/>

// Pre-configured variants
<ProcessPageHeader 
  processName="User Onboarding"
  processStatus="published"
  onEdit={handleEdit}
  onExecute={handleExecute}
/>

// ❌ WRONG - Custom header layout
<div className="flex justify-between items-center mb-6">
  <h1>Title</h1>
  <div className="flex gap-2">...</div>
</div>
```

5. **Loading States**: Always use `LoadingSkeleton`
```tsx
// ✅ CORRECT - Consistent loading skeletons
<ProcessDetailSkeleton />
<MetricCardGrid loading={true} columns={4} />
<TextSkeleton lines={3} />
<CardSkeleton height="200px" />

// ❌ WRONG - Custom loading states
<div className="animate-pulse bg-gray-200 h-4 rounded">
```

#### **DESIGN TOKEN USAGE:**

```tsx
// ✅ CORRECT - Use design tokens
import { designTokens, spacing, typography } from '@/styles';

// Access semantic spacing
<div className="p-6 space-y-6">  // Uses semanticSpacing
  <div className="grid grid-cols-auto gap-6">  // Uses grid patterns
    <GlassCard variant="highlight">
      Content with proper tokens
    </GlassCard>
  </div>
</div>

// ❌ WRONG - Arbitrary values
<div className="p-7 space-y-5 gap-5">  // Breaks 8px grid
```

#### **LAYOUT PATTERNS:**

```tsx
// ✅ CORRECT - Use layout utilities
import { pageContainer, contentContainer, grid } from '@/styles';

<div className={pageContainer({ background: 'default' })}>
  <div className={contentContainer({ maxWidth: 'lg' })}>
    <div className={grid({ cols: 'auto', gap: 'default' })}>
      <GlassCard>Content</GlassCard>
    </div>
  </div>
</div>

// ❌ WRONG - Manual layout classes
<div className="min-h-screen bg-gradient-to-br...">
  <div className="container mx-auto p-6">
```

#### **MIGRATION REQUIREMENTS:**

**IMMEDIATE ACTION REQUIRED**: When working on existing pages:

1. **Replace custom glass cards** with `GlassCard` components
2. **Replace custom status displays** with `StatusBadge`/`StatusIndicator`
3. **Replace custom metric layouts** with `MetricCard`
4. **Replace custom page headers** with `PageHeader` variants
5. **Replace custom loading states** with `LoadingSkeleton`

#### **DOCUMENTATION REFERENCES:**

**BEFORE starting UI work, READ:**
- `/docs/design-system/DESIGN_GUIDELINES.md` - Complete design system
- `/docs/design-system/COMPONENT_USAGE_GUIDE.md` - Usage examples
- `src/styles/` - Design tokens and utilities

#### **UI-DEV CHECKLIST (MANDATORY):**

- [ ] ✅ Using `GlassCard` instead of custom glass styling
- [ ] ✅ Using `StatusBadge`/`StatusIndicator` for all status displays  
- [ ] ✅ Using `MetricCard` for all metric displays
- [ ] ✅ Using `PageHeader` variants for page headers
- [ ] ✅ Using `LoadingSkeleton` for loading states
- [ ] ✅ Using design tokens from `@/styles`
- [ ] ✅ Following 8px grid system
- [ ] ✅ Supporting dark mode through variants
- [ ] ✅ Including accessibility features (ARIA, keyboard navigation)

## 🔗 MANDATORY: MCP Schema Standards

### **🚨 CAMEL CASE ENFORCEMENT - ZERO TOLERANCE POLICY**

ALL UI components, API integration, and data interfaces **MUST** use camelCase convention. No exceptions.

#### **Frontend Schema Requirements:**

1. **TypeScript Interfaces**: Always camelCase
```typescript
// ✅ CORRECT - camelCase properties
interface TaskRequest {
  personId: string;
  taskName: string;
  description: string;
  dueDate?: string;
}

// ❌ WRONG - snake_case properties
interface TaskRequest {
  person_id: string;
  task_name: string;
  task_description: string;
  due_date?: string;
}
```

2. **API Client Usage**: Ensure camelCase parameters
```typescript
// ✅ CORRECT - API calls use camelCase
const response = await tasksApi.createTask({
  personId: "123",
  taskName: "Review PR",
  description: "Code review task",
  dueDate: "2024-12-31"
});

// ❌ WRONG - snake_case parameters
const response = await tasksApi.createTask({
  person_id: "123",
  task_name: "Review PR",
  task_description: "Code review task"
});
```

3. **Form Data Handling**: camelCase form fields
```typescript
// ✅ CORRECT - Form state uses camelCase
const [formData, setFormData] = useState({
  taskName: '',
  description: '', 
  dueDate: '',
  personId: ''
});

// Handle form submission with camelCase
const handleSubmit = async (e: React.FormEvent) => {
  await apiClient.createTask({
    taskName: formData.taskName,
    personId: formData.personId,
    description: formData.description
  });
};
```

4. **Component Props**: camelCase prop names
```typescript
// ✅ CORRECT - camelCase props
interface TaskCardProps {
  taskId: string;
  taskName: string;
  dueDate?: string;
  personId: string;
}

const TaskCard: React.FC<TaskCardProps> = ({ 
  taskId, 
  taskName, 
  dueDate, 
  personId 
}) => {
  // Component implementation
};
```

#### **UI Validation Checklist:**
- [ ] All TypeScript interfaces use camelCase
- [ ] API client calls use camelCase parameters
- [ ] Form fields use camelCase names
- [ ] Component props follow camelCase
- [ ] State variables use camelCase
- [ ] No snake_case in JSON payloads

#### **CRITICAL UI INTEGRATION ISSUES:**
- Form validation errors due to parameter name mismatches
- API calls failing due to incorrect parameter names
- Type errors from inconsistent naming conventions
- WebSocket events using wrong field names

**Reference**: See `/Users/maxmednikov/MaxSpace/Hyperion/.claude/schema-standards.md` for complete standards.

## Component Architecture

### UI Component Pattern
```typescript
// Component with CVA variants
import { cva, type VariantProps } from 'class-variance-authority'

const componentVariants = cva(
  'base-classes',
  {
    variants: {
      variant: {
        primary: 'variant-specific-classes',
        secondary: 'variant-specific-classes',
      },
      size: {
        sm: 'size-specific-classes',
        md: 'size-specific-classes',
      }
    },
    defaultVariants: {
      variant: 'primary',
      size: 'md',
    },
  }
)

interface ComponentProps extends 
  React.HTMLAttributes<HTMLElement>,
  VariantProps<typeof componentVariants> {
  // Additional props
}

const Component = React.forwardRef<HTMLElement, ComponentProps>(
  ({ className, variant, size, ...props }, ref) => {
    return (
      <element
        ref={ref}
        className={componentVariants({ variant, size, className })}
        {...props}
      />
    )
  }
)
Component.displayName = 'Component'
```

### Common UI Components Available
- **Button**: Multiple variants (primary, secondary, success, danger, outline, ghost)
- **Card**: With CardHeader, CardTitle, CardContent, CardDescription
- **Input, Textarea, Label**: Form elements
- **Select**: Using Radix UI Select
- **Dialog**: Modal dialogs with Radix UI
- **Badge, StatusBadge**: For status indicators
- **Alert**: For notifications
- **Skeleton**: Loading states
- **Tabs**: Tab navigation
- **DropdownMenu**: Context menus

## API Architecture

### MANDATORY: Use API Clients Only
```typescript
// ✅ CORRECT - Always use typed API clients
import { staffApi, tasksApi, documentsApi } from '@/services/clients'

const loadData = async () => {
  const { people } = await staffApi.listPeople()
  const { tasks } = await tasksApi.listTasks()
}

// ❌ WRONG - Never use axios/fetch directly
import axios from 'axios'
const response = await axios.get('/api/people')
```

### API Client Structure
- **BaseApiClient**: Handles auth, retries, error transformation
- **Service-specific clients**: TasksApiClient, StaffApiClient, etc.
- **Consistent error handling**: All errors transformed to standard format
- **Automatic authentication**: JWT tokens handled transparently
- **Type safety**: Full TypeScript interfaces matching backend models

### API Response Types
```typescript
// Standard list response
interface ListResponse<T> {
  items: T[]
  total: number
  page?: number
  pageSize?: number
}

// Error response
interface ErrorResponse {
  error: string
  code?: string
  details?: any
}

// Identity type (used everywhere)
interface Identity {
  type: 'human' | 'agent' | 'system'
  name: string
  id: string
}
```

## Page Structure Pattern

### Standard Page Layout
```typescript
import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from 'react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/Card'
import { Button } from '@/components/ui/Button'
import { Skeleton } from '@/components/ui/Skeleton'
import { apiClient } from '@/services/clients'

const PageName = () => {
  const queryClient = useQueryClient()
  
  // Data fetching with React Query
  const { data, isLoading, error } = useQuery(
    ['queryKey'],
    () => apiClient.getData(),
    {
      refetchInterval: 30000, // Auto-refresh every 30s
      retry: 3,
    }
  )
  
  // Mutations with optimistic updates
  const mutation = useMutation(
    (data) => apiClient.updateData(data),
    {
      onSuccess: () => {
        queryClient.invalidateQueries(['queryKey'])
      }
    }
  )
  
  if (isLoading) return <LoadingState />
  if (error) return <ErrorState error={error} />
  
  return (
    <div className="space-y-6">
      <PageHeader />
      <ContentSection data={data} />
    </div>
  )
}
```

## State Management

### Server State (React Query)
- **Use for**: All API data fetching and caching
- **Query keys**: Hierarchical and consistent
- **Optimistic updates**: For better UX
- **Background refetching**: Keep data fresh
- **Error/loading states**: Always handle

### Local State (useState/useReducer)
- **Use for**: UI state (modals, forms, selections)
- **Form state**: Use controlled components
- **Complex state**: Consider useReducer

## Routing Patterns

### Route Structure
```typescript
// Routes should follow RESTful patterns
/dashboard                  // Overview page
/tasks                     // List view
/tasks/new                 // Create form
/tasks/:id                 // Detail view
/tasks/:id/edit           // Edit form
/agents                   // Agent management
/agents/:id/instances     // Nested resources
```

### Navigation
```typescript
import { useNavigate, Link } from 'react-router-dom'

// Programmatic navigation
const navigate = useNavigate()
navigate('/tasks/123')

// Declarative navigation
<Link to="/tasks" className="text-blue-600 hover:underline">
  View Tasks
</Link>
```

## Form Handling

### Controlled Components Pattern
```typescript
const [formData, setFormData] = useState({
  name: '',
  description: '',
  priority: 'medium'
})

const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault()
  try {
    await apiClient.createItem(formData)
    navigate('/success')
  } catch (error) {
    setError(error.message)
  }
}

<form onSubmit={handleSubmit} className="space-y-4">
  <Input
    value={formData.name}
    onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
    required
  />
</form>
```

## Error Handling

### API Error Display
```typescript
// Consistent error display pattern
{error && (
  <Alert variant="error">
    <AlertCircle className="h-4 w-4" />
    <span>{error.message || 'An error occurred'}</span>
  </Alert>
)}
```

### Loading States
```typescript
// Skeleton loading pattern
{isLoading ? (
  <div className="space-y-4">
    <Skeleton className="h-12 w-full" />
    <Skeleton className="h-32 w-full" />
  </div>
) : (
  <ActualContent />
)}
```

## WebSocket Integration

### Real-time Updates
```typescript
import { io } from 'socket.io-client'

useEffect(() => {
  const socket = io(WS_URL, {
    auth: { token: getAuthToken() }
  })
  
  socket.on('update', (data) => {
    queryClient.setQueryData(['queryKey'], data)
  })
  
  return () => socket.disconnect()
}, [])
```

## Performance Optimization

### Code Splitting
```typescript
// Lazy load heavy components
const HeavyComponent = lazy(() => import('./HeavyComponent'))

<Suspense fallback={<Loading />}>
  <HeavyComponent />
</Suspense>
```

### Memoization
```typescript
// Memoize expensive computations
const expensiveValue = useMemo(
  () => computeExpensiveValue(data),
  [data]
)

// Memoize callbacks
const handleClick = useCallback(
  (id) => { /* handler */ },
  [dependency]
)
```

## Testing Requirements

### Component Testing
- Use Vitest for unit tests
- Test user interactions
- Mock API calls
- Test error states

### E2E Testing
- Use Playwright for E2E tests
- Test critical user flows
- Test against deployed UI

## Accessibility

### ARIA Requirements
- Proper semantic HTML
- ARIA labels for interactive elements
- Keyboard navigation support
- Focus management

### Contrast & Colors
- WCAG AA compliance minimum
- Test with dark mode
- Sufficient color contrast

## File Organization

```
src/
├── components/
│   ├── ui/              # Reusable UI components
│   ├── layout/          # Layout components
│   └── features/        # Feature-specific components
├── pages/               # Route components
├── services/
│   ├── base/           # Base classes
│   ├── clients/        # API clients
│   └── auth/           # Auth services
├── hooks/              # Custom React hooks
├── types/              # TypeScript types
├── utils/              # Utility functions
└── styles/             # Global styles
```

## Development Workflow

### Component Creation
1. Check if similar component exists
2. Use existing UI components as base
3. Follow CVA pattern for variants
4. Add proper TypeScript types
5. Include loading/error states

### API Integration
1. Use appropriate API client
2. Add types to `/types/api.ts`
3. Use React Query for data fetching
4. Handle errors consistently
5. Add loading skeletons

### Styling Approach
1. Use Tailwind utility classes
2. Follow existing color scheme
3. Maintain consistent spacing
4. Add hover/focus states
5. Support dark mode

## Common Pitfalls to Avoid

### ❌ DON'T
- Import from @mui/material
- Use axios/fetch directly
- Create inline styles
- Skip error handling
- Hardcode values
- Use any TypeScript type
- Create components without loading states

### ✅ DO
- Use Radix UI components
- Use typed API clients
- Use Tailwind classes
- Handle all error cases
- Use configuration/constants
- Define proper interfaces
- Include loading/error/empty states

## Code Quality Standards

### TypeScript
- Strict mode enabled
- No implicit any
- Proper interface definitions
- Use type imports

### Component Quality
- Single responsibility
- Props validation
- Proper naming
- Documentation for complex logic

### Performance
- Lazy load routes
- Optimize re-renders
- Use proper keys in lists
- Debounce user input

## Deployment Considerations

### Environment Variables
```typescript
// Use import.meta.env for Vite
const API_URL = import.meta.env.VITE_API_URL || 'http://hyperion'
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://hyperion'
```

### Build Optimization
- Tree shaking enabled
- Code splitting by route
- Asset optimization
- Source maps for debugging

## 🏗️ CRITICAL: MANDATORY ARCHITECTURE DOCUMENTATION

### **🚨 ZERO TOLERANCE POLICY - UI ARCHITECTURE DOCUMENTATION IS MANDATORY**

**EVERY UI component, page, feature, or integration MUST be documented and stored.**

### **MANDATORY UI ARCHITECTURE DOCUMENTATION STRUCTURE**

Each UI feature MUST maintain comprehensive architecture documentation in:
```
./docs/03-services/hyperion-web-ui/architecture/
├── README.md                    # UI overview and quick reference
├── component-architecture.md   # Component hierarchy and patterns
├── page-flows.md               # User flows and page interactions
├── api-integrations.md         # API client usage and data flows
├── state-management.md         # State management patterns and data flow
├── ui-component-catalog.md     # Complete component documentation
└── feature-specifications.md   # Feature requirements and implementation
```

### **CRITICAL REQUIREMENTS FOR EVERY UI CHANGE**

#### **1. Component Architecture Documentation**
- **Component hierarchy**: How components compose together
- **Props interfaces**: All prop types with validation rules
- **State management**: Local state vs server state patterns
- **Event handling**: User interactions and event flows
- **Styling patterns**: CSS classes, variants, and theming
- **Accessibility**: ARIA compliance and keyboard navigation

#### **2. Page Flow Documentation**
- **User journeys**: Step-by-step user interactions
- **Navigation paths**: How users move between pages
- **Data loading**: When and how data is fetched
- **Error handling**: Error states and recovery paths
- **Loading states**: Progressive loading and skeleton patterns
- **Form workflows**: Form validation and submission flows

#### **3. API Integration Documentation**
- **API client usage**: Which clients are used where
- **Data transformations**: How API data maps to UI components
- **Error handling**: API error display and user feedback
- **Caching strategy**: React Query configuration and invalidation
- **Authentication**: JWT token handling in UI
- **Real-time updates**: WebSocket integration patterns

#### **4. State Management Documentation**
- **React Query usage**: Server state management patterns
- **Local state patterns**: useState vs useReducer decisions
- **State sharing**: Context usage and prop drilling avoidance
- **Performance optimizations**: Memoization and re-render prevention
- **State persistence**: Local storage and session management

#### **5. UI Component Catalog**
- **Component variants**: All CVA variants with examples
- **Usage examples**: Real implementation examples
- **Composition patterns**: How components work together
- **Accessibility features**: Screen reader support and keyboard navigation
- **Performance notes**: Rendering optimization and bundle size impact

### **UI-DEV AGENT MANDATORY CHECKLIST**

EVERY UI change MUST include:

- [ ] **🧩 Update component documentation** in `./docs/03-services/hyperion-web-ui/architecture/`
- [ ] **🔄 Document user flows** if navigation or interactions change
- [ ] **📡 Update API integration docs** if API calls change
- [ ] **🎨 Update component catalog** if new UI components are added
- [ ] **📊 Document state changes** if state management patterns change
- [ ] **♿ Document accessibility features** for new interactive elements
- [ ] **💾 Store in coordinator** using the coordinator_upsert_knowledge MCP tool

### **COORDINATOR STORAGE REQUIREMENTS**

After documenting UI changes, STORE the documentation in coordinator knowledge:

```bash
# Use the MCP coordinator_upsert_knowledge tool to store UI architecture documentation
mcp__hyper__coordinator_upsert_knowledge \
  collection="hyperion_ui_architecture" \
  text="UI: <change description with user experience impact>" \
  metadata='{"component": "hyperion-web-ui", "type": "ui_architecture", "feature": "<feature>", "impact": "<user_impact>"}'
```

### **DESIGN SYSTEM STORAGE REQUIREMENTS**

**MANDATORY**: Store design system usage patterns and discoveries:

```bash
# Store design system component usage
mcp__hyper__coordinator_upsert_knowledge \
  collection="hyperion_ui_architecture" \
  text="DESIGN SYSTEM [$(date +%Y-%m-%d)]: <component> implementation
COMPONENT: <GlassCard|MetricCard|PageHeader|StatusBadge>
USE CASE: <when and why this component was chosen>
IMPLEMENTATION:
\`\`\`tsx
<working code example>
\`\`\`
VARIANTS USED: <variant options and reasoning>
ACCESSIBILITY: <ARIA features, keyboard navigation>
RESPONSIVE: <mobile/tablet behavior>
INTEGRATION: <how it works with other components>
MIGRATION: <what it replaced and why>
"

# Store design system patterns
mcp__hyper__coordinator_upsert_knowledge \
  collection="hyperion_ui_architecture" \
  text="UI PATTERN [$(date +%Y-%m-%d)]: <pattern name> with Design System
BEFORE: <legacy implementation>
AFTER: <design system implementation>
BENEFITS: <consistency, accessibility, maintainability>
CODE EXAMPLE:
\`\`\`tsx
// Design system approach
import { GlassCard, StatusBadge } from '@/components/atoms';

<GlassCard variant='highlight' size='lg'>
  <StatusBadge status='completed'>Completed</StatusBadge>
</GlassCard>
\`\`\`
COMPONENTS USED: <list of design system components>
DESIGN TOKENS: <spacing, colors, typography used>
"
```

### **DOCUMENTATION UPDATE TRIGGERS**

Documentation MUST be updated when:

1. **New pages or routes** are added
2. **Component interfaces** change (new props, events)
3. **User flows** are modified or added
4. **API integrations** change (new endpoints, data structures)
5. **State management patterns** are updated
6. **Authentication/authorization** UI changes
7. **Performance optimizations** that affect user experience
8. **Accessibility features** are added or modified
9. **Design system changes** (colors, typography, spacing)
10. **Build or deployment processes** affecting UI delivery

### **NO EXCEPTIONS - UI ARCHITECTURE DOCUMENTATION IS NOT OPTIONAL**

- ❌ UI changes without documentation updates are INCOMPLETE
- ❌ Missing component documentation blocks releases
- ❌ Outdated user flow documentation causes UX regression
- ✅ Documentation-first UI development is the only acceptable approach

### **DOCUMENTATION QUALITY STANDARDS**

- **User flow diagrams**: Use Mermaid syntax for user journey diagrams
- **Component examples**: Include working code examples and screenshots
- **API integration**: Document request/response flows with examples
- **Error scenarios**: Document error states and user recovery paths
- **Performance impact**: Document loading times and bundle size impact
- **Mobile responsiveness**: Document mobile-specific behavior and breakpoints

### **CRITICAL UI REQUIREMENTS**

#### **User Experience Documentation**
Every UI change must document:

```markdown
# ✅ CORRECT - Complete UX documentation
## User Flow: Task Creation
1. User clicks "New Task" button → Opens modal dialog
2. User fills required fields → Real-time validation
3. User clicks "Create" → Loading state shown
4. Success → Modal closes, list updates, success toast
5. Error → Error message inline, form stays open

## Accessibility
- Modal has proper focus management
- Form fields have ARIA labels
- Keyboard navigation fully supported
- Screen reader announcements for state changes

## Performance
- Modal lazy-loaded on first use
- Form validation debounced 300ms
- API call includes optimistic update
```

```markdown
# ❌ WRONG - Incomplete documentation
## Task Creation
Added a modal for creating tasks. Uses API to save.
```

#### **Component Integration Documentation**
- **Composition patterns**: How components work together
- **Prop drilling**: Document data flow through component trees
- **Event bubbling**: How events propagate through UI
- **Side effects**: Document API calls and state changes
- **Error boundaries**: How errors are caught and displayed

## **REMEMBER: UI DOCUMENTATION SHAPES USER EXPERIENCE**

## Summary

When developing for Hyperion Web UI:
1. Always use Radix UI, never Material UI
2. Always use API clients, never direct HTTP calls
3. Follow the CVA pattern for components
4. Use Tailwind for styling
5. Handle loading and error states
6. Follow TypeScript best practices
7. Maintain consistency with existing patterns
8. **MANDATORY: Document all UI changes in architecture docs**

This guide ensures all AI-generated code matches the established patterns and maintains the high quality standards of the Hyperion platform.

## 🧠 Knowledge Management Protocol

### **🚨 MANDATORY: QUERY COORDINATOR KNOWLEDGE BEFORE ANY WORK - ZERO TOLERANCE POLICY**

**CRITICAL: You MUST query coordinator knowledge BEFORE starting ANY UI development work. NO EXCEPTIONS!**

### **BEFORE Starting Work (MANDATORY):**
```bash
# 1. Query for existing UI patterns
mcp__hyper__coordinator_query_knowledge collection="hyperion_ui_architecture" query="<component> pattern implementation"

# 2. Query for UI bugs and fixes
mcp__hyper__coordinator_query_knowledge collection="hyperion_bugs" query="UI <error or issue>"

# 3. Query for component implementations
mcp__hyper__coordinator_query_knowledge collection="hyperion_project" query="React <feature> component example"

# 4. Query for API integration patterns
mcp__hyper__coordinator_query_knowledge collection="hyperion_ui_architecture" query="<service> API integration React Query"
```

**❌ FAILURE TO QUERY = DUPLICATED WORK OR INCONSISTENT UI**

### **DURING Work (MANDATORY):**
Store information IMMEDIATELY after discovering:
- UI component patterns that work well
- Failed UI approaches and why they failed
- Performance optimizations for React components
- API integration patterns with React Query
- Accessibility improvements

```bash
# Store UI bug fix
mcp__hyper__coordinator_upsert_knowledge collection="hyperion_bugs" text="
UI BUG FIX [$(date +%Y-%m-%d)]: <component> - <issue>
SYMPTOM: <what was broken in UI>
ROOT CAUSE: <why it was broken>
SOLUTION: 
\`\`\`tsx
// Fixed code
\`\`\`
FILES: <list of changed files>
TESTING: <how to verify in browser>
"

# Store UI pattern
mcp__hyper__coordinator_upsert_knowledge collection="hyperion_ui_architecture" text="
UI PATTERN [$(date +%Y-%m-%d)]: <pattern name>
USE CASE: <when to use this pattern>
IMPLEMENTATION:
\`\`\`tsx
<working code example>
\`\`\`
BENEFITS: <why this pattern works>
EXAMPLES: <where it's used in codebase>
"
```

### **AFTER Completing Work (MANDATORY):**
```bash
# Store comprehensive UI solution
mcp__hyper__coordinator_upsert_knowledge collection="hyperion_project" text="
UI COMPLETED [$(date +%Y-%m-%d)]: [UI Development] <feature description>
COMPONENTS CREATED/MODIFIED:
- <component1>: <what it does>
- <component2>: <what it does>
CODE EXAMPLE:
\`\`\`tsx
<key implementation code>
\`\`\`
USER FLOW:
1. <step 1>
2. <step 2>
ACCESSIBILITY: <ARIA features added>
PERFORMANCE: <optimizations applied>
TESTING: <how to test in browser>
"
```

### **Coordinator Knowledge Collections for UI Development:**

1. **`hyperion_ui_architecture`** - UI patterns, component designs, React patterns
2. **`hyperion_bugs`** - UI bugs, rendering issues, browser compatibility
3. **`hyperion_project`** - General UI implementations, features
4. **`hyperion_performance`** - React performance, bundle size, rendering
5. **`hyperion_accessibility`** - ARIA patterns, keyboard navigation, screen readers

### **UI-Specific Query Patterns:**

```bash
# Before creating new component
mcp__hyper__coordinator_query_knowledge collection="hyperion_ui_architecture" query="React <component type> Radix UI pattern"

# Before fixing UI bug
mcp__hyper__coordinator_query_knowledge collection="hyperion_bugs" query="React <exact error> console error"

# Before API integration
mcp__hyper__coordinator_query_knowledge collection="hyperion_ui_architecture" query="React Query <service> API hooks"

# Before styling work
mcp__hyper__coordinator_query_knowledge collection="hyperion_ui_architecture" query="Tailwind <component> styling CVA variants"

# For accessibility
mcp__hyper__coordinator_query_knowledge collection="hyperion_accessibility" query="ARIA <component type> keyboard navigation"
```

### **UI Development Storage Requirements:**

#### **ALWAYS Store After:**
- ✅ Creating new React components
- ✅ Fixing UI bugs or console errors
- ✅ Implementing new UI patterns
- ✅ Solving performance issues
- ✅ Adding accessibility features
- ✅ Creating custom hooks
- ✅ Integrating with new APIs
- ✅ Implementing complex state management

#### **Storage Format for UI Components:**
```
COMPONENT [date]: <ComponentName>
PURPOSE: <what it does>
PROPS:
\`\`\`typescript
interface ComponentProps {
  // prop definitions
}
\`\`\`
IMPLEMENTATION:
\`\`\`tsx
// Component code
\`\`\`
USAGE:
\`\`\`tsx
<ComponentName prop1="value" />
\`\`\`
LOCATION: src/components/<path>
DEPENDENCIES: <Radix UI components used>
```

#### **Storage Format for UI Bugs:**
```
UI BUG [date]: <page/component> - <error>
BROWSER: <Chrome/Firefox/Safari>
SYMPTOM: <visual issue or console error>
ROOT CAUSE: <technical cause>
FIX:
\`\`\`tsx
// Fixed code with explanation
\`\`\`
PREVENTION: <how to avoid similar issues>
```

### **UI-DEV AGENT CHECKLIST (UPDATED):**
- [ ] ✅ Query coordinator knowledge for existing UI patterns BEFORE starting
- [ ] ✅ Query for similar components already built
- [ ] ✅ Store new component patterns with examples
- [ ] ✅ Store failed UI approaches with reasons
- [ ] ✅ Document API integration patterns
- [ ] ✅ Store accessibility improvements
- [ ] ✅ Query before implementing similar features
- [ ] ✅ Store performance optimizations

### **CRITICAL REMINDERS:**
1. **Include working code** - Full TSX examples, not descriptions
2. **Store Radix UI patterns** - Document how Radix components are used
3. **Document React Query usage** - Include hook patterns and caching
4. **Store Tailwind patterns** - Include CVA variant definitions
5. **Cross-browser testing** - Note any browser-specific fixes

### **Component Pattern Storage Example:**
```bash
mcp__hyper__coordinator_upsert_knowledge collection="hyperion_ui_architecture" text="
UI COMPONENT PATTERN [2025-01-10]: Modal Dialog with Form
USE CASE: Creating/editing entities with validation
IMPLEMENTATION:
\`\`\`tsx
import * as Dialog from '@radix-ui/react-dialog'
import { useForm } from 'react-hook-form'
import { useMutation, useQueryClient } from '@tanstack/react-query'

export function EntityModal({ onClose }) {
  const queryClient = useQueryClient()
  const { register, handleSubmit, formState: { errors } } = useForm()
  
  const mutation = useMutation({
    mutationFn: createEntity,
    onSuccess: () => {
      queryClient.invalidateQueries(['entities'])
      onClose()
    }
  })
  
  return (
    <Dialog.Root open onOpenChange={onClose}>
      <Dialog.Portal>
        <Dialog.Overlay className='fixed inset-0 bg-black/50' />
        <Dialog.Content className='fixed top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2'>
          <form onSubmit={handleSubmit(data => mutation.mutate(data))}>
            {/* Form fields */}
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}
\`\`\`
KEY PATTERNS:
- Radix Dialog for accessibility
- React Hook Form for validation
- React Query for server state
- Tailwind for styling
LOCATION: Used in TaskModal, PersonModal, DocumentModal
"
```

## **NO EXCEPTIONS - COORDINATOR USAGE IS MANDATORY FOR ALL UI DEVELOPMENT WORK**
