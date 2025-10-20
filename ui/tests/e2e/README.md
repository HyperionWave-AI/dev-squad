# E2E Tests for Hyperion Tasks UI

## Overview

Comprehensive Playwright E2E test suite for the Tasks Management UI at `http://localhost:7095/ui/tasks`.

## Test Coverage

### Fully Implemented Tests (37 passing tests)

1. **Page Load & Initial State** (3 tests)
   - Page loads with correct title and header
   - Footer displays platform information
   - Search box is visible and accessible

2. **Kanban Board Rendering** (6 tests)
   - All 4 columns display (Pending, In Progress, Blocked, Completed)
   - Task counts shown per column
   - Empty state handling
   - Task cards display complete information
   - Human vs Agent task visual distinction
   - Priority badges displayed

3. **Task Detail Dialog** (6 tests)
   - Opens on card click
   - Displays task metadata
   - Shows description section
   - Agent tasks section with expandable details
   - Close button functionality
   - TODO list display

4. **Search & Filtering** (4 tests)
   - Filter by search query
   - Result count display
   - Clear search functionality
   - Cross-field search (title + description)

5. **Status Updates** (2 tests)
   - Status display on cards
   - Status display in dialog

6. **Error States & Console** (2 tests)
   - No critical console errors
   - KanbanBoard logging verification

7. **Responsive Behavior** (5 tests)
   - Mobile viewport (375x667)
   - Tablet viewport (768x1024)
   - Desktop viewport (1920x1080)
   - Viewport resizing
   - Dialog on mobile

8. **Loading States** (2 tests)
   - Loading indicator on page load
   - Loading indicator hides after tasks load

9. **Navigation** (3 tests)
   - Navigate to other sections
   - Navigate to Knowledge section
   - Stay on Tasks page

10. **Accessibility** (4 tests)
    - Proper heading hierarchy
    - Accessible buttons
    - Accessible search input
    - Keyboard navigation

### Placeholder Tests (Awaiting UI Implementation)

The following tests are marked as `.skip` and will be enabled when the UI implements these features:

- **Task Creation** (3 tests) - Create task form, validation, success flow
- **Task Status Updates** (1 test) - Status transitions and persistence
- **Task Editing** (2 tests) - Edit mode, validation
- **Task Deletion** (2 tests) - Confirmation dialog, deletion flow
- **API Error Handling** (2 tests) - API failures, network timeouts

## Prerequisites

1. **Backend Running**: The backend must be running on port 7095
   ```bash
   # From project root
   make dev
   # OR
   make dev-hot
   ```

2. **Node Dependencies**: Install Playwright and dependencies
   ```bash
   cd ui
   npm install
   npx playwright install
   ```

## Running Tests

### Run All Tests
```bash
cd ui
npm test
```

### Run Specific Test File
```bash
npm test tests/e2e/tasks-ui.spec.ts
```

### Run with UI Mode (Interactive)
```bash
npm run test:ui
```

### Run in Headed Mode (See Browser)
```bash
npm run test:headed
```

### Run in Debug Mode (Step Through)
```bash
npm run test:debug
```

### Run Specific Viewport
```bash
# Mobile only
npm run test:mobile

# Tablet only
npm run test:tablet

# Desktop browsers
npm run test:desktop
```

### Skip Web Server Check
If you already have the backend running and want to skip the web server check:
```bash
SKIP_WEB_SERVER=1 npm test
```

## Test Reports

After running tests, view the HTML report:
```bash
npm run test:report
```

Reports are saved to:
- HTML: `ui/test-results/html/`
- JSON: `ui/test-results/results.json`

## Test Structure

```
ui/tests/e2e/tasks-ui.spec.ts
├── Page Load & Initial State
├── Kanban Board Rendering
├── Task Detail Dialog
├── Search and Filtering
├── Status Updates
├── Task Creation (skipped)
├── Task Editing (skipped)
├── Task Deletion (skipped)
├── Error States and Console Errors
├── Responsive Behavior
├── Loading States
├── Navigation
├── Accessibility
└── Data Persistence
```

## Key Test Patterns

### Accessibility-First Selectors
```typescript
page.getByRole('button', { name: 'Tasks' })
page.getByRole('heading', { name: 'Pending', level: 6 })
page.getByRole('textbox', { name: /Search tasks/i })
```

### Proper Waiting
```typescript
// Wait for specific element
await page.waitForSelector('h6:has-text("Pending")');

// Wait for URL change
await page.waitForURL(/\/ui\/chat/);

// Small delay for debounce
await page.waitForTimeout(500);
```

### Viewport Testing
```typescript
await page.setViewportSize({ width: 375, height: 667 }); // Mobile
await page.setViewportSize({ width: 768, height: 1024 }); // Tablet
await page.setViewportSize({ width: 1920, height: 1080 }); // Desktop
```

### Console Monitoring
```typescript
page.on('console', (msg) => {
  if (msg.type() === 'error') {
    consoleErrors.push(msg.text());
  }
});
```

## Known Issues

1. **Non-Critical Console Error**: `vite.svg` returns 404 (cosmetic issue, filtered in tests)
2. **Skipped Tests**: Several CRUD operation tests are skipped pending UI implementation

## Continuous Integration

The tests are configured for CI with:
- Retries: 2 (only in CI)
- Workers: 1 (only in CI)
- Screenshots on failure
- Video retention on failure
- Trace on first retry

## Troubleshooting

### Backend Not Running
```
Error: Target page, context or browser has been closed
```
**Solution**: Start the backend with `make dev` or `make dev-hot`

### Port Conflict
```
Error: connect ECONNREFUSED 127.0.0.1:7095
```
**Solution**: Ensure nothing else is using port 7095, or change the port in `.env.hyper`

### Slow Tests
```
Timeout of 30000ms exceeded
```
**Solution**: Increase timeout in `playwright.config.ts` or check backend performance

## Future Enhancements

1. Enable skipped tests when UI implements CRUD features
2. Add API mocking for error scenario testing
3. Add visual regression testing with screenshot comparison
4. Add performance testing (Core Web Vitals)
5. Add network request validation

## Contributing

When adding new tests:
1. Use descriptive test names
2. Group related tests in `test.describe` blocks
3. Use accessibility-first selectors
4. Avoid hardcoded waits (use proper wait conditions)
5. Add comments for complex interactions
6. Update this README with new test categories

## Resources

- [Playwright Documentation](https://playwright.dev)
- [Playwright Best Practices](https://playwright.dev/docs/best-practices)
- [MUI Testing Guide](https://mui.com/material-ui/guides/testing/)
- [Accessibility Testing](https://playwright.dev/docs/accessibility-testing)
