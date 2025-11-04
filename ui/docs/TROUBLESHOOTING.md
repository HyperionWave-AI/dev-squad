# Hyperion Coordinator UI - Troubleshooting Guide

## Table of Contents

1. [Quick Diagnostic Checklist](#quick-diagnostic-checklist)
2. [Architecture Violations](#architecture-violations)
3. [Build Failures](#build-failures)
4. [Test Failures](#test-failures)
5. [Responsive Layout Issues](#responsive-layout-issues)
6. [WebSocket Connection Problems](#websocket-connection-problems)
7. [Performance Issues](#performance-issues)
8. [Drag-and-Drop Issues](#drag-and-drop-issues)
9. [Accessibility Violations](#accessibility-violations)
10. [Common Error Messages](#common-error-messages)
11. [Debugging Strategies](#debugging-strategies)
12. [FAQ](#faq)

---

## Quick Diagnostic Checklist

Before diving into specific issues, run through this checklist:

```bash
# 1. Check Node.js version
node --version  # Should be v18+

# 2. Check dependencies
npm install

# 3. Check TypeScript compilation
npx tsc --noEmit

# 4. Check linting
npm run lint

# 5. Check build
npm run build

# 6. Check tests
npm test

# 7. Check dev server
npm run dev
```

---

## Architecture Violations

### ESLint Error: Importing `mcpClient.ts`

**❌ Error:**
```
ESLint: Direct MCP calls are prohibited. Use restClient instead.
  'no-restricted-imports' rule violation
```

**🔍 Cause:**
Attempting to import and use `mcpClient.ts` directly from UI components, violating the architecture rule that all MCP calls must go through the REST API layer.

**✅ Solution:**

Replace direct MCP imports with REST client:

```typescript
// ❌ WRONG - Direct MCP import
import { mcpClient } from '@/services/mcpClient';
const tasks = await mcpClient.listHumanTasks();

// ✅ CORRECT - Use REST client
import { restClient } from '@/services/restClient';
const tasks = await restClient.listHumanTasks();
```

**📚 Reference:**
- [ARCHITECTURE.md - API Communication](./ARCHITECTURE.md#api-communication)
- [API_INTEGRATION.md](./API_INTEGRATION.md)

---

## Build Failures

### TypeScript Errors

**❌ Error:**
```
error TS2322: Type 'string' is not assignable to type 'TaskStatus'.
```

**🔍 Cause:**
Type mismatch between expected and actual types, often due to incorrect API response typing or missing type definitions.

**✅ Solution:**

1. Check type definitions in `src/types/coordinator.ts`:

```typescript
// Ensure types match API responses
export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'blocked';
```

2. Verify API response transformation:

```typescript
// Transform API response to match frontend types
const transformTask = (apiTask: APITask): Task => ({
  ...apiTask,
  status: apiTask.status as TaskStatus, // Type assertion if needed
});
```

3. Run type checking:

```bash
npx tsc --noEmit
```

**🛠️ Common TypeScript Fixes:**

```typescript
// Fix 1: Add proper type annotation
const [status, setStatus] = useState<TaskStatus>('pending');

// Fix 2: Use type guard
if (isValidStatus(status)) {
  // status is now narrowed to TaskStatus
}

// Fix 3: Add interface extension
interface MyComponentProps extends BaseProps {
  additionalProp: string;
}
```

### Build Command Fails

**❌ Error:**
```
npm run build
Build failed with 1 error
```

**🔍 Cause:**
Vite build errors, usually due to import issues, missing dependencies, or configuration problems.

**✅ Solution:**

```bash
# 1. Clean install
rm -rf node_modules package-lock.json
npm install

# 2. Clear Vite cache
rm -rf node_modules/.vite

# 3. Try build again
npm run build

# 4. Check for specific errors
npm run build -- --logLevel=error
```

**Common Build Issues:**

| Error | Cause | Solution |
|-------|-------|----------|
| `Cannot find module '@/...'` | Path alias not configured | Check `vite.config.ts` and `tsconfig.json` |
| `Unexpected token` | Syntax error | Run `npm run lint` to find syntax errors |
| `Out of memory` | Large bundle | Increase Node memory: `NODE_OPTIONS=--max-old-space-size=4096 npm run build` |

---

## Test Failures

### Playwright Tests Fail

**❌ Error:**
```
Test failed: page.locator('[data-testid="kanban-board"]') not found
```

**🔍 Cause:**
Element not rendered, timing issue, or incorrect selector.

**✅ Solution:**

1. **Check screenshots:**
```bash
# View test failure screenshots
ls test-results/*/test-failed-*.png
open test-results/*/test-failed-1.png
```

2. **View traces:**
```bash
# Open trace viewer
npm run test:report
# Or directly
npx playwright show-trace test-results/trace.zip
```

3. **Add explicit waits:**
```typescript
// ❌ WRONG - No wait
await page.click('[data-testid="button"]');

// ✅ CORRECT - Wait for element
await page.waitForSelector('[data-testid="button"]');
await page.click('[data-testid="button"]');
```

4. **Debug interactively:**
```bash
npm run test:debug
```

### Test Timeout

**❌ Error:**
```
Test timeout of 30000ms exceeded
```

**✅ Solution:**

```typescript
// Increase timeout for slow tests
test('slow test', async ({ page }) => {
  test.setTimeout(60000); // 60 seconds

  // ... test code
});

// Or in config
// playwright.config.ts
export default defineConfig({
  timeout: 60 * 1000, // 60 seconds
});
```

### Flaky Tests

**❌ Problem:**
Tests pass sometimes but fail randomly.

**✅ Solution:**

```typescript
// 1. Add proper waits
await page.waitForLoadState('networkidle');

// 2. Wait for specific conditions
await expect(page.locator('[data-testid="data"]')).toBeVisible();

// 3. Retry assertions
await expect(async () => {
  const count = await page.locator('.item').count();
  expect(count).toBeGreaterThan(0);
}).toPass({ timeout: 5000 });

// 4. Use data-testid for stable selectors
// ✅ GOOD
await page.click('[data-testid="submit-button"]');

// ❌ BAD - Fragile selector
await page.click('.MuiButton-root:nth-child(3)');
```

---

## Responsive Layout Issues

### Layout Breaks at Specific Viewport

**❌ Problem:**
Layout doesn't adapt correctly at tablet or mobile breakpoints.

**🔍 Cause:**
Incorrect breakpoint values or missing responsive props.

**✅ Solution:**

1. **Check Tailwind breakpoints** (`tailwind.config.js`):

```javascript
screens: {
  'sm': '600px',   // Material-UI sm
  'md': '900px',   // Material-UI md
  'lg': '1200px',  // Material-UI lg
}
```

2. **Use MUI responsive props:**

```typescript
// ✅ CORRECT - Responsive Grid
<Grid container spacing={{ xs: 2, md: 3 }}>
  <Grid item xs={12} md={6} lg={4}>
    Content
  </Grid>
</Grid>

// ❌ WRONG - Fixed layout
<Grid container spacing={3}>
  <Grid item xs={4}>Content</Grid>
</Grid>
```

3. **Test responsive layouts:**

```bash
# Run responsive tests
npm run test:mobile
npm run test:tablet
npm run test:desktop
```

4. **Use browser DevTools:**
   - Open DevTools (F12)
   - Toggle device toolbar (Ctrl+Shift+M)
   - Test at 375px, 768px, 1200px, 1920px

### Horizontal Scroll Issues

**❌ Problem:**
Unwanted horizontal scrollbar appears.

**✅ Solution:**

```typescript
// Fix container overflow
<Box sx={{ overflowX: 'hidden', width: '100%' }}>
  <Container maxWidth="lg">
    Content
  </Container>
</Box>

// For intentional horizontal scroll (mobile Kanban)
<Box
  sx={{
    display: 'flex',
    overflowX: 'auto',
    gap: 2,
    pb: 2,
    '&::-webkit-scrollbar': {
      height: 8,
    },
  }}
>
  <Box sx={{ minWidth: 300 }}>Column</Box>
</Box>
```

### Touch Target Too Small

**❌ Problem:**
Buttons are difficult to tap on mobile.

**✅ Solution:**

```typescript
// ✅ CORRECT - 44px minimum touch target
<Button
  sx={{
    minHeight: 44,
    minWidth: 44,
    px: 3,
  }}
>
  Button
</Button>

// Use Tailwind utility
<button className="touch-target">
  Button
</button>
```

---

## WebSocket Connection Problems

### Chat Not Working

**❌ Problem:**
Chat messages don't send or receive.

**🔍 Cause:**
WebSocket connection failure or incorrect URL configuration.

**✅ Solution:**

1. **Check WebSocket URL:**

```typescript
// src/services/chatService.ts
const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:7095';
console.log('Connecting to WebSocket:', WS_URL);
```

2. **Check browser console for errors:**
   - Open DevTools → Console tab
   - Look for WebSocket errors:
     - `WebSocket connection to 'ws://...' failed`
     - `Error during WebSocket handshake`

3. **Verify backend is running:**

```bash
# Check if backend is running on port 7095
curl http://localhost:7095/health

# Or check WebSocket endpoint
curl -i -N -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     http://localhost:7095/api/v1/chat
```

4. **Check network tab:**
   - Open DevTools → Network tab
   - Filter by WS (WebSocket)
   - Verify connection status and messages

5. **Add connection logging:**

```typescript
// Enhanced logging
const ws = new WebSocket(wsUrl);

ws.onopen = () => {
  console.log('[WebSocket] Connected successfully');
};

ws.onerror = (error) => {
  console.error('[WebSocket] Connection error:', error);
};

ws.onclose = (event) => {
  console.log('[WebSocket] Connection closed:', event.code, event.reason);
};

ws.onmessage = (event) => {
  console.log('[WebSocket] Received:', event.data);
};
```

### WebSocket Reconnection Issues

**❌ Problem:**
WebSocket doesn't reconnect after network interruption.

**✅ Solution:**

```typescript
// Implement reconnection logic
class WebSocketService {
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  connect(url: string) {
    const ws = new WebSocket(url);

    ws.onclose = () => {
      if (this.reconnectAttempts < this.maxReconnectAttempts) {
        setTimeout(() => {
          this.reconnectAttempts++;
          this.connect(url);
        }, this.reconnectDelay * Math.pow(2, this.reconnectAttempts));
      }
    };

    return ws;
  }
}
```

---

## Performance Issues

### Slow Rendering

**❌ Problem:**
UI feels sluggish or components take long to render.

**🔍 Diagnosis:**

1. **Use React DevTools Profiler:**
   - Install React Developer Tools extension
   - Open DevTools → Profiler tab
   - Record interaction
   - Analyze render times

2. **Check for unnecessary re-renders:**

```typescript
// Add logging to component
useEffect(() => {
  console.log('Component rendered:', { props, state });
});
```

**✅ Solution:**

```typescript
// 1. Memoize expensive computations
const filteredTasks = useMemo(() => {
  return tasks.filter(task => task.status === 'pending');
}, [tasks]);

// 2. Memoize callbacks
const handleClick = useCallback((id: string) => {
  updateTask(id);
}, [updateTask]);

// 3. Use React.memo for components
const TaskCard = React.memo<TaskCardProps>(({ task }) => {
  return <Card>{task.title}</Card>;
});

// 4. Lazy load heavy components
const HeavyComponent = lazy(() => import('./HeavyComponent'));

<Suspense fallback={<CircularProgress />}>
  <HeavyComponent />
</Suspense>
```

### Large Bundle Size

**❌ Problem:**
Initial page load is slow due to large JavaScript bundle.

**✅ Solution:**

1. **Analyze bundle size:**

```bash
npm run build
# Check dist/ folder sizes

# Use bundle analyzer
npm install -D rollup-plugin-visualizer
```

2. **Code splitting:**

```typescript
// Split by route
const TasksPage = lazy(() => import('./pages/TasksPage'));
const ChatPage = lazy(() => import('./pages/ChatPage'));

<Routes>
  <Route path="/tasks" element={
    <Suspense fallback={<Loading />}>
      <TasksPage />
    </Suspense>
  } />
</Routes>
```

3. **Import only what you need:**

```typescript
// ✅ GOOD - Named imports
import { Button, Card } from '@mui/material';

// ❌ BAD - Import everything
import * as MUI from '@mui/material';
```

### Memory Leaks

**❌ Problem:**
Memory usage increases over time, browser becomes slow.

**✅ Solution:**

```typescript
// Clean up WebSocket connections
useEffect(() => {
  const ws = connectWebSocket();

  return () => {
    ws.close(); // Cleanup on unmount
  };
}, []);

// Clean up intervals
useEffect(() => {
  const interval = setInterval(() => {
    fetchData();
  }, 3000);

  return () => {
    clearInterval(interval); // Cleanup
  };
}, []);

// Clean up event listeners
useEffect(() => {
  const handleResize = () => {
    updateLayout();
  };

  window.addEventListener('resize', handleResize);

  return () => {
    window.removeEventListener('resize', handleResize);
  };
}, []);
```

---

## Drag-and-Drop Issues

### Drag-and-Drop Not Working

**❌ Problem:**
Tasks cannot be dragged between columns.

**🔍 Cause:**
Drag-and-drop library not properly configured or event handlers missing.

**✅ Solution:**

1. **Check @hello-pangea/dnd setup:**

```typescript
import { DragDropContext, Droppable, Draggable } from '@hello-pangea/dnd';

<DragDropContext onDragEnd={handleDragEnd}>
  <Droppable droppableId="column-1">
    {(provided) => (
      <div ref={provided.innerRef} {...provided.droppableProps}>
        {items.map((item, index) => (
          <Draggable key={item.id} draggableId={item.id} index={index}>
            {(provided) => (
              <div
                ref={provided.innerRef}
                {...provided.draggableProps}
                {...provided.dragHandleProps}
              >
                {item.content}
              </div>
            )}
          </Draggable>
        ))}
        {provided.placeholder}
      </div>
    )}
  </Droppable>
</DragDropContext>
```

2. **Ensure unique IDs:**

```typescript
// ✅ GOOD - Unique IDs
<Draggable key={task.id} draggableId={task.id} index={index}>

// ❌ BAD - Non-unique IDs
<Draggable key={index} draggableId={`task-${index}`} index={index}>
```

### Touch Drag Issues on Mobile

**❌ Problem:**
Drag-and-drop doesn't work on touch devices.

**✅ Solution:**

```typescript
// @hello-pangea/dnd supports touch out of the box
// But ensure you're not preventing touch events

// Remove event.preventDefault() on touch events
// Don't use pointer-events: none on draggable elements
```

---

## Accessibility Violations

### WCAG Failures

**❌ Problem:**
Accessibility tests report WCAG 2.1 AA violations.

**✅ Solution:**

1. **Run accessibility tests:**

```bash
npm run test:accessibility
```

2. **Common violations and fixes:**

**Missing Alt Text:**
```typescript
// ❌ WRONG
<img src="logo.png" />

// ✅ CORRECT
<img src="logo.png" alt="Company logo" />
```

**Low Color Contrast:**
```typescript
// ❌ WRONG - Poor contrast
<Typography sx={{ color: '#ccc', backgroundColor: '#fff' }}>
  Low contrast text
</Typography>

// ✅ CORRECT - Good contrast (4.5:1)
<Typography sx={{ color: 'text.primary' }}>
  High contrast text
</Typography>
```

**Missing ARIA Labels:**
```typescript
// ❌ WRONG
<IconButton onClick={handleDelete}>
  <Delete />
</IconButton>

// ✅ CORRECT
<IconButton onClick={handleDelete} aria-label="Delete task">
  <Delete />
</IconButton>
```

**Improper Heading Hierarchy:**
```typescript
// ❌ WRONG - Skipped h2
<h1>Page Title</h1>
<h3>Section Title</h3>

// ✅ CORRECT
<h1>Page Title</h1>
<h2>Section Title</h2>
```

### Keyboard Navigation Issues

**❌ Problem:**
Cannot navigate with keyboard (Tab, Enter, Arrow keys).

**✅ Solution:**

```typescript
// Ensure focusable elements
<div
  role="button"
  tabIndex={0}
  onClick={handleClick}
  onKeyDown={(e) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleClick();
    }
  }}
>
  Click me
</div>

// Use visible focus indicators
<Button
  sx={{
    '&:focus-visible': {
      outline: '2px solid',
      outlineColor: 'primary.main',
      outlineOffset: 2,
    },
  }}
>
  Button
</Button>
```

---

## Common Error Messages

### `Cannot read property 'map' of undefined`

**✅ Solution:**
```typescript
// Add null check
{tasks?.map(task => <TaskCard key={task.id} task={task} />)}

// Or use default value
{(tasks || []).map(task => <TaskCard key={task.id} task={task} />)}
```

### `React Hook useEffect has a missing dependency`

**✅ Solution:**
```typescript
// Add dependency or use ESLint disable
useEffect(() => {
  fetchData();
}, [fetchData]); // Add missing dependency

// Or disable if intentional
useEffect(() => {
  fetchData();
  // eslint-disable-next-line react-hooks/exhaustive-deps
}, []);
```

### `Failed to fetch` / Network Error

**✅ Solution:**
```typescript
// Check API URL
console.log('API URL:', import.meta.env.VITE_MCP_BRIDGE_URL);

// Add error handling
try {
  const data = await restClient.listTasks();
} catch (error) {
  console.error('API Error:', error);
  // Show user-friendly error
}
```

---

## Debugging Strategies

### 1. Browser DevTools

**Console:**
- Check for JavaScript errors
- Log component state
- Inspect API responses

**Network:**
- Monitor API calls
- Check WebSocket connections
- Verify request/response data

**Elements:**
- Inspect DOM structure
- Check applied styles
- Verify ARIA attributes

### 2. React Developer Tools

- Inspect component hierarchy
- View props and state
- Profile render performance

### 3. Source Maps

```typescript
// Enable in vite.config.ts
export default defineConfig({
  build: {
    sourcemap: true,
  },
});
```

### 4. Logging Strategy

```typescript
// Development logging
if (import.meta.env.DEV) {
  console.log('[DEBUG]', 'Component state:', state);
}

// API logging
const logRequest = (method: string, url: string, data?: any) => {
  console.log(`[API] ${method} ${url}`, data);
};
```

---

## FAQ

### Q: Why is the dev server slow to start?

**A:** Vite performs optimization on first start. Subsequent starts are faster. Clear cache if it persists:

```bash
rm -rf node_modules/.vite
npm run dev
```

### Q: Why do tests fail in CI but pass locally?

**A:** Usually timing or environment differences. Add retries and explicit waits:

```typescript
test.setTimeout(60000);
await page.waitForLoadState('networkidle');
```

### Q: How do I test API integration without backend?

**A:** Mock API responses in tests:

```typescript
await page.route('/api/v1/tasks', async (route) => {
  await route.fulfill({
    status: 200,
    body: JSON.stringify({ tasks: mockTasks }),
  });
});
```

### Q: Why doesn't dark mode persist?

**A:** Ensure localStorage is being set:

```typescript
// Check localStorage
console.log(localStorage.getItem('theme-mode'));

// Verify setThemePreference is called
setThemePreference(newMode);
```

### Q: How do I debug WebSocket issues?

**A:** Enable verbose logging and check Network tab:

```typescript
// Add logging
ws.onmessage = (event) => {
  console.log('[WS] Received:', JSON.parse(event.data));
};

// Check DevTools → Network → WS
```

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture
- [Developer Guide](./DEVELOPER_GUIDE.md) - Development setup
- [Testing Guide](./TESTING.md) - Testing strategies
- [API Integration Guide](./API_INTEGRATION.md) - API usage
- [UI/UX Patterns](./UI_UX_PATTERNS.md) - Design system

---

## Getting Help

If you encounter an issue not covered here:

1. **Check existing documentation** in `ui/docs/`
2. **Search test files** in `ui/tests/` for examples
3. **Review source code** in `ui/src/`
4. **Check browser console** for error messages
5. **Run diagnostics** with the checklist above

---

**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
**Maintainer**: Hyperion Platform Team
