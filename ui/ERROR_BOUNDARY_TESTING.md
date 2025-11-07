# Error Boundary Testing Guide

## Overview
React error boundaries have been successfully implemented to prevent cascade failures in the UI. This document describes how to test and verify the error boundary functionality.

## Implementation Summary

### Files Created/Modified
1. **ErrorBoundary.tsx** - `/ui2/src/components/organisms/ErrorBoundary.tsx`
   - React class component with `getDerivedStateFromError` and `componentDidCatch`
   - Glassmorphic design matching Hyperion UI
   - Dark mode support
   - Recovery options: Reload Page, Go Back, Try Again (dev only)
   - Stack trace display in development mode only

2. **ErrorBoundaryTest.tsx** - `/ui2/src/components/organisms/ErrorBoundaryTest.tsx`
   - Test component to trigger intentional errors
   - Floating widget in bottom-right corner
   - Button to throw test error

3. **CodeSearchPage.tsx** - `/ui2/src/pages/CodeSearchPage.tsx`
   - Wrapped with ErrorBoundary
   - Named export preserved for testing
   - Includes temporary ErrorBoundaryTest component

4. **KanbanBoard.tsx** - `/ui2/src/pages/KanbanBoard.tsx`
   - Wrapped with ErrorBoundary
   - Named export preserved for testing

## How to Test Error Boundaries

### Method 1: Using ErrorBoundaryTest Component (Recommended)

The ErrorBoundaryTest component has been temporarily added to CodeSearchPage for testing purposes.

1. **Start the dev server:**
   ```bash
   cd /Users/maxmednikov/MaxSpace/hyper/ui2
   npm run dev
   ```

2. **Navigate to Code Search page:**
   - Open browser to `http://localhost:5173` (or your configured port)
   - Go to Code Search page

3. **Test error boundary:**
   - Look for yellow test widget in bottom-right corner
   - Click "🧪 Trigger Render Error" button
   - **Expected behavior:**
     - Error boundary catches the error
     - Fallback UI appears with glassmorphic design
     - Error message shown: "Test error from ErrorBoundaryTest component - ErrorBoundary is working!"
     - Stack trace visible in development mode (can expand "Show technical details")
     - Three action buttons appear:
       - "Reload Page" - reloads the entire page
       - "Go Back" - navigates back in browser history
       - "Try Again" - resets error state and attempts to re-render (dev only)

4. **Verify recovery:**
   - Click "Reload Page" - page should reload successfully
   - OR click "Go Back" - should navigate to previous page
   - OR click "Try Again" (dev only) - attempts to re-render (will error again if component still throws)

### Method 2: Manual Error Injection

If you want to test with real component errors:

1. **Temporarily add an error to a component:**
   ```typescript
   // In any component inside CodeSearchPage or KanbanBoard
   const SomeComponent = () => {
     throw new Error('Test error - remove this line after testing');
     return <div>Content</div>;
   };
   ```

2. **Load the page** - error boundary should catch it

3. **Remove the test error** after verification

### Method 3: Console Testing

Test programmatically from browser console:

```javascript
// This won't be caught by error boundary (event handlers aren't caught)
// but you can use it to test other error handling
throw new Error('Console test error');
```

Note: Error boundaries do NOT catch:
- Errors in event handlers (use try-catch instead)
- Asynchronous code (setTimeout, promises)
- Server-side rendering errors
- Errors in the error boundary itself

## Verification Checklist

- [ ] Error boundary catches component render errors
- [ ] Fallback UI displays with glassmorphic design
- [ ] Error message is clearly visible
- [ ] Stack trace is visible in development mode
- [ ] Stack trace is hidden in production mode
- [ ] "Reload Page" button works and reloads the page
- [ ] "Go Back" button works and navigates back
- [ ] "Try Again" button appears in dev mode only
- [ ] Dark mode styling works correctly
- [ ] Error doesn't crash the entire application
- [ ] Other pages remain functional
- [ ] Error is logged to console in development

## Production Behavior

In production build:
- Stack trace is **hidden** from users
- Only "Reload Page" and "Go Back" buttons shown
- "Try Again" button **not shown**
- Error message simplified for end users
- Errors logged to console for debugging

To test production behavior:
```bash
npm run build
npm run preview
```

## Cleanup After Testing

**IMPORTANT:** Remove the ErrorBoundaryTest component after testing:

1. **Remove from CodeSearchPage.tsx:**
   ```typescript
   // Remove this import:
   import { ErrorBoundaryTest } from '../components/organisms/ErrorBoundaryTest';

   // Remove this component from JSX:
   <ErrorBoundaryTest />
   ```

2. **(Optional) Delete ErrorBoundaryTest.tsx:**
   ```bash
   rm /Users/maxmednikov/MaxSpace/hyper/ui2/src/components/organisms/ErrorBoundaryTest.tsx
   ```

## Future Enhancements

1. **Add error logging service integration:**
   - Send errors to Sentry, LogRocket, or similar service
   - Track error frequency and patterns
   - Alert developers for critical errors

2. **Granular error boundaries:**
   - Wrap individual organisms for more specific error handling
   - Prevent single component failure from breaking entire page section

3. **Custom fallback UI per page:**
   - Different fallback designs for different pages
   - Context-specific recovery options

4. **Error boundary testing utilities:**
   - Automated tests for error boundary behavior
   - E2E tests with Playwright

## Architecture Notes

### Why Class Components?

Error boundaries **must** use class components because:
- `getDerivedStateFromError` is a static lifecycle method
- `componentDidCatch` is a class lifecycle method
- React does not support error boundaries with hooks (yet)

### Error Boundary Placement

Error boundaries are placed at:
1. **Page level** - Wraps entire page (CodeSearchPage, KanbanBoard)
2. **Future: Organism level** - Can wrap individual complex components

### Error Propagation

Errors propagate up the component tree until caught by nearest error boundary:
```
App
└── ErrorBoundary (page level)
    └── CodeSearchPage
        ├── CodeSearchForm ❌ error thrown here
        ├── CodeResultsList
        └── FolderManager
```
Error caught at page-level boundary → show fallback UI for entire page.

## Developer Notes

- Error boundaries are React class components (not functional)
- Must implement `getDerivedStateFromError` and/or `componentDidCatch`
- Do not catch errors in event handlers, async code, or SSR
- Use try-catch for event handler errors
- Use `.catch()` or async/await try-catch for promise errors

## Testing Completed

✅ ErrorBoundary component created with proper lifecycle methods
✅ CodeSearchPage wrapped with ErrorBoundary
✅ KanbanBoard wrapped with ErrorBoundary
✅ ErrorBoundaryTest component created for verification
✅ TypeScript compilation successful (error boundary files)
✅ Dark mode support implemented
✅ Glassmorphic design matching Hyperion UI
✅ Recovery options (reload, go back, try again)
✅ Development-only stack traces
✅ Named exports preserved for testing

## Contact

For questions or issues with error boundaries, contact the ui-dev team or check coordinator task logs.
