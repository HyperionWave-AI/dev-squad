# Hyperion Coordinator UI - Testing Documentation

## Table of Contents

1. [Testing Overview](#testing-overview)
2. [Playwright E2E Tests](#playwright-e2e-tests)
3. [Test Projects](#test-projects)
4. [Test Commands](#test-commands)
5. [Unit Tests (Vitest)](#unit-tests-vitest)
6. [Test Structure](#test-structure)
7. [Test Data & Fixtures](#test-data--fixtures)
8. [Writing Tests](#writing-tests)
9. [Debugging Tests](#debugging-tests)
10. [Continuous Integration](#continuous-integration)
11. [Best Practices](#best-practices)

---

## Testing Overview

The Hyperion Coordinator UI has a comprehensive testing infrastructure covering:

- **E2E Testing**: Playwright for full user interaction flows
- **Unit Testing**: Vitest for component and service logic
- **Accessibility Testing**: Axe-core for WCAG 2.1 AA compliance
- **Visual Regression**: Screenshot comparison across viewports
- **Cross-Browser**: Chromium, WebKit testing
- **Responsive**: Mobile, tablet, desktop viewports

### Test Coverage Summary

| Category | Test Suites | Coverage |
|----------|------------|----------|
| **Kanban Board** | 8 suites | Rendering, drag-drop, responsive, accessibility |
| **Accessibility** | Dedicated suite | WCAG 2.1 AA compliance |
| **Visual Regression** | Screenshot tests | Layout consistency across viewports |
| **MUI Components** | Component tests | Material-UI integration validation |
| **Real-time** | Concurrent updates | WebSocket polling, data sync |
| **Search/Filter** | Filtering tests | Search and filter functionality |

---

## Playwright E2E Tests

### Test Suites (in `tests/kanban/`)

#### 1. Rendering Tests (`rendering.spec.ts`)

Tests basic rendering and display functionality:

- ✅ Column structure validation (pending, in_progress, completed, blocked)
- ✅ Task card rendering with correct data
- ✅ MUI component presence (Cards, AppBar, Chips)
- ✅ Loading states during data fetch
- ✅ Error handling and error displays
- ✅ Empty states for columns without tasks
- ✅ Task metadata display (dates, status)

**Example Test:**
```typescript
test('should render all four Kanban columns', async ({ page }) => {
  await page.goto('/tasks');

  await expect(page.locator('[data-testid="kanban-column-pending"]')).toBeVisible();
  await expect(page.locator('[data-testid="kanban-column-in-progress"]')).toBeVisible();
  await expect(page.locator('[data-testid="kanban-column-completed"]')).toBeVisible();
  await expect(page.locator('[data-testid="kanban-column-blocked"]')).toBeVisible();
});
```

#### 2. Drag-and-Drop Tests (`drag-drop.spec.ts`)

Tests drag-and-drop functionality:

- ✅ Move tasks between columns
- ✅ Drag visual feedback (opacity, cursor)
- ✅ Drop target highlighting
- ✅ Task position updates after drop
- ✅ API integration (status updates sent to backend)
- ✅ Edge cases (invalid drops, rapid movements, ESC cancellation)
- ✅ Persistence after page reload

**Example Test:**
```typescript
test('should drag task from pending to in_progress', async ({ page }) => {
  await page.goto('/tasks');

  const taskCard = page.locator('[data-task-id="human-task-1"]');
  const targetColumn = page.locator('[data-testid="kanban-column-in-progress"]');

  await taskCard.dragTo(targetColumn);

  // Verify task moved
  await expect(targetColumn.locator('[data-task-id="human-task-1"]')).toBeVisible();
});
```

#### 3. Responsive Design Tests (`responsive.spec.ts`)

Tests responsive layouts across viewports:

- ✅ Desktop layout (1920x1080) - 4 columns side-by-side
- ✅ Tablet layout (768x1024) - 2 columns per row
- ✅ Mobile layout (375x812) - Single column, horizontal scroll
- ✅ Touch interactions on mobile
- ✅ Horizontal scrolling behavior
- ✅ Viewport transitions (resize handling)
- ✅ Layout adaptation and reflow

**Viewport Breakpoints:**
```typescript
const viewports = {
  mobile: { width: 375, height: 812 },
  tablet: { width: 768, height: 1024 },
  desktop: { width: 1920, height: 1080 },
};
```

#### 4. Accessibility Tests (`accessibility.spec.ts`)

Tests WCAG 2.1 AA compliance:

- ✅ Axe-core automated WCAG audits
- ✅ Keyboard navigation (Tab, Enter, Space, Arrow keys)
- ✅ Screen reader compatibility (ARIA labels, roles, live regions)
- ✅ Color contrast validation (4.5:1 for text)
- ✅ Focus management and focus traps
- ✅ Drag-and-drop keyboard accessibility
- ✅ Heading hierarchy (h1 → h6 structure)

**Example Test:**
```typescript
import { injectAxe, checkA11y } from 'axe-playwright';

test('should have no accessibility violations @accessibility', async ({ page }) => {
  await page.goto('/tasks');
  await injectAxe(page);

  await checkA11y(page, null, {
    detailedReport: true,
    detailedReportOptions: { html: true },
  });
});
```

#### 5. Visual Regression Tests (`visual-regression.spec.ts`)

Tests visual consistency with screenshots:

- ✅ Screenshot comparison for full board
- ✅ Column visual consistency
- ✅ Task card styling and shadows
- ✅ Priority badge colors (red, yellow, green)
- ✅ Responsive layout screenshots
- ✅ MUI Card elevation shadows
- ✅ Empty/error/loading states

**Screenshot Storage:** `test-results/screenshots/`

**Example Test:**
```typescript
test('should match full Kanban board screenshot', async ({ page }) => {
  await page.goto('/tasks');
  await page.waitForSelector('[data-testid="kanban-board"]');

  await expect(page).toHaveScreenshot('kanban-board-full.png', {
    fullPage: true,
    maxDiffPixels: 100, // Allow small differences
  });
});
```

#### 6. MUI Component Tests (`mui-components.spec.ts`)

Tests Material-UI integration:

- ✅ MUI AppBar and Toolbar
- ✅ MUI Card and CardContent
- ✅ MUI Chip (priority badges)
- ✅ MUI Button and IconButton
- ✅ MUI Grid layout
- ✅ MUI Typography
- ✅ MUI Paper elevation
- ✅ MUI Avatar (if present)
- ✅ Theme integration (light/dark modes)

**Example Test:**
```typescript
test('should render MUI Cards with correct elevation', async ({ page }) => {
  await page.goto('/tasks');

  const cards = page.locator('.MuiCard-root');
  await expect(cards.first()).toHaveCSS('box-shadow', /rgba/);
});
```

#### 7. Concurrent Updates Tests (`concurrent-updates.spec.ts`)

Tests real-time data synchronization:

- ✅ UI polling (3-second intervals)
- ✅ Real-time data synchronization
- ✅ Concurrent user actions (multiple updates)
- ✅ Optimistic UI updates
- ✅ Network failure recovery
- ✅ Debouncing repeated requests
- ✅ Scroll position maintenance during updates

**Example Test:**
```typescript
test('should auto-refresh every 3 seconds', async ({ page }) => {
  await page.goto('/tasks');

  const initialTasks = await page.locator('[data-testid="task-card"]').count();

  // Wait for refresh interval
  await page.waitForTimeout(3500);

  // Verify data was refetched
  const updatedTasks = await page.locator('[data-testid="task-card"]').count();
  expect(updatedTasks).toBeGreaterThanOrEqual(initialTasks);
});
```

#### 8. Filtering and Search Tests (`filtering-search.spec.ts`)

Tests search and filter functionality:

- ✅ Search by task prompt/description
- ✅ Filter by status (pending, in_progress, completed, blocked)
- ✅ Filter by priority (high, medium, low)
- ✅ Filter by agent name
- ✅ Combined filters (multiple filters active)
- ✅ Clear filters functionality
- ✅ Search results highlighting
- ✅ No results message display

**Example Test:**
```typescript
test('should filter tasks by status', async ({ page }) => {
  await page.goto('/tasks');

  await page.click('[data-testid="filter-status"]');
  await page.click('[data-value="in_progress"]');

  const visibleTasks = page.locator('[data-testid="task-card"]:visible');
  await expect(visibleTasks).toHaveCount(await countInProgressTasks());
});
```

---

## Test Projects

Tests run across multiple browser and viewport configurations:

### Project Configurations

| Project | Browser | Viewport | Use Case |
|---------|---------|----------|----------|
| **chromium-desktop** | Chrome | 1920x1080 | Primary desktop testing |
| **webkit-desktop** | Safari | 1920x1080 | Safari compatibility |
| **tablet** | iPad Pro | 768x1024 | Tablet responsive |
| **mobile** | iPhone 13 | 375x812 | Mobile responsive |
| **accessibility** | Chrome | 1920x1080 | WCAG 2.1 AA tests (tagged with @accessibility) |

### Configuration Example

```typescript
// playwright.config.ts
export default defineConfig({
  projects: [
    {
      name: 'chromium-desktop',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 }
      },
    },
    {
      name: 'accessibility',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1920, height: 1080 }
      },
      grep: /@accessibility/,
    },
  ],
});
```

---

## Test Commands

### Running Tests

```bash
# Run all tests
npm test

# Run all tests (explicit)
npm run test
```

### Development Testing

```bash
# Run tests with browser UI visible
npm run test:headed

# Interactive UI mode (Playwright UI)
npm run test:ui

# Debug mode (step through tests)
npm run test:debug
```

### Filtered Testing

```bash
# Run only accessibility tests
npm run test:accessibility

# Run only mobile tests
npm run test:mobile

# Run only tablet tests
npm run test:tablet

# Run only desktop tests
npm run test:desktop
```

### Reporting

```bash
# View HTML test report
npm run test:report

# Report location: test-results/html/index.html
```

### Command Details

| Command | Description | Playwright Options |
|---------|-------------|--------------------|
| `npm test` | All tests, all projects | Default config |
| `npm run test:headed` | See browser UI | `--headed` |
| `npm run test:ui` | Interactive UI | `--ui` |
| `npm run test:debug` | Step-through debugging | `--debug` |
| `npm run test:accessibility` | WCAG tests only | `--grep @accessibility` |
| `npm run test:mobile` | Mobile viewport | `--project=mobile` |
| `npm run test:tablet` | Tablet viewport | `--project=tablet` |
| `npm run test:desktop` | Desktop viewports | `--project=chromium-desktop --project=webkit-desktop` |
| `npm run test:report` | HTML report | `npx playwright show-report` |

---

## Unit Tests (Vitest)

### Unit Test Structure

```
src/
├── components/
│   └── __tests__/
│       ├── TaskCard.test.tsx
│       ├── AgentTaskCard.test.tsx
│       └── KanbanBoard.test.tsx
├── services/
│   └── __tests__/
│       ├── restClient.test.ts
│       └── chatService.test.ts
└── utils/
    └── __tests__/
        └── formatters.test.ts
```

### Running Unit Tests

```bash
# Run unit tests
npm run test:unit

# Run in watch mode
npm run test:unit:ui

# Run with coverage
npm run test:unit -- --coverage
```

### Unit Test Example

```typescript
import { render, screen } from '@testing-library/react';
import { TaskCard } from '../TaskCard';
import { mockHumanTasks } from '../../../tests/fixtures/mockTasks';

describe('TaskCard', () => {
  it('should render task prompt', () => {
    const task = mockHumanTasks[0];

    render(<TaskCard task={task} />);

    expect(screen.getByText(task.prompt)).toBeInTheDocument();
  });

  it('should display correct status badge', () => {
    const task = mockHumanTasks[0];

    render(<TaskCard task={task} />);

    const badge = screen.getByText(task.status);
    expect(badge).toHaveClass('MuiChip-root');
  });
});
```

---

## Test Structure

### Directory Layout

```
tests/
├── kanban/                     # Kanban board test suites
│   ├── rendering.spec.ts       # Basic rendering tests
│   ├── drag-drop.spec.ts       # Drag-and-drop functionality
│   ├── responsive.spec.ts      # Responsive design
│   ├── accessibility.spec.ts   # WCAG 2.1 AA compliance
│   ├── visual-regression.spec.ts # Screenshot comparisons
│   ├── mui-components.spec.ts  # MUI component validation
│   ├── concurrent-updates.spec.ts # Real-time updates
│   └── filtering-search.spec.ts # Filtering and search
├── fixtures/                   # Test data
│   ├── mockTasks.ts           # Mock task data
│   ├── mockKnowledgeData.ts   # Mock knowledge base data
│   └── knowledge-fixtures.ts  # Knowledge test fixtures
├── utils/                      # Test utilities
│   └── accessibility.ts       # Accessibility helpers
└── README.md                   # Test documentation
```

---

## Test Data & Fixtures

### Mock Task Data

Mock data is provided in `tests/fixtures/mockTasks.ts`:

```typescript
export const mockHumanTasks: HumanTask[] = [
  {
    id: 'human-task-1',
    prompt: 'Implement user authentication system',
    status: 'in_progress',
    createdAt: '2025-09-30T10:00:00.000Z',
    updatedAt: '2025-09-30T11:30:00.000Z',
  },
  // ... 3 more tasks (pending, completed, blocked)
];

export const mockAgentTasks: AgentTask[] = [
  {
    id: 'agent-task-1',
    humanTaskId: 'human-task-1',
    agentName: 'Backend Services Specialist',
    role: 'Implement JWT authentication',
    status: 'in_progress',
    todos: [
      { id: 'todo-1-1', description: 'Setup JWT library', status: 'completed' },
      { id: 'todo-1-2', description: 'Create auth middleware', status: 'in_progress' },
    ],
    // ...
  },
];
```

### Using Fixtures

```typescript
import { mockHumanTasks, mockAgentTasks } from '../fixtures/mockTasks';

test.beforeEach(async ({ page }) => {
  // Mock API responses
  await page.route('/api/v1/tasks', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ tasks: mockHumanTasks }),
    });
  });
});
```

### Column Definitions

```typescript
export const kanbanColumns = [
  { id: 'pending', title: 'Pending', dataTestId: 'kanban-column-pending' },
  { id: 'in_progress', title: 'In Progress', dataTestId: 'kanban-column-in-progress' },
  { id: 'completed', title: 'Completed', dataTestId: 'kanban-column-completed' },
  { id: 'blocked', title: 'Blocked', dataTestId: 'kanban-column-blocked' },
];
```

---

## Writing Tests

### Test Pattern

```typescript
import { test, expect } from '@playwright/test';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    // Setup (navigate, mock APIs, etc.)
    await page.goto('/tasks');
  });

  test('should do something specific', async ({ page }) => {
    // Arrange
    const element = page.locator('[data-testid="my-element"]');

    // Act
    await element.click();

    // Assert
    await expect(page.locator('[data-testid="result"]')).toBeVisible();
  });

  test('should handle error case', async ({ page }) => {
    // Test error scenarios
  });
});
```

### Best Practices

#### 1. Use Data Test IDs

```typescript
// ✅ GOOD - Stable selector
await page.locator('[data-testid="kanban-column-pending"]').click();

// ❌ BAD - Brittle selector
await page.locator('.MuiCard-root:nth-child(1)').click();
```

#### 2. Wait for Elements

```typescript
// ✅ GOOD - Wait for element
await page.waitForSelector('[data-testid="task-card"]');
await expect(page.locator('[data-testid="task-card"]')).toBeVisible();

// ❌ BAD - No waiting
await page.click('[data-testid="task-card"]'); // Might fail if not loaded
```

#### 3. Mock API Responses

```typescript
// Mock API for consistent testing
await page.route('/api/v1/tasks', async (route) => {
  await route.fulfill({
    status: 200,
    body: JSON.stringify({ tasks: mockHumanTasks }),
  });
});
```

#### 4. Test Isolation

Each test should be independent and not rely on other tests:

```typescript
test.beforeEach(async ({ page, context }) => {
  // Clear cookies, local storage, etc.
  await context.clearCookies();
  await page.goto('/tasks');
});
```

#### 5. Accessibility Testing

```typescript
test('should be keyboard accessible @accessibility', async ({ page }) => {
  await page.goto('/tasks');

  // Tab through interactive elements
  await page.keyboard.press('Tab');
  const focused = await page.evaluate(() => document.activeElement?.tagName);
  expect(focused).toBeTruthy();
});
```

---

## Debugging Tests

### Playwright Inspector

```bash
npm run test:debug
```

**Features:**
- Step through test actions
- Inspect locators in real-time
- View console logs
- Examine network requests
- Pause execution at breakpoints

### Trace Viewer

After a test failure:

```bash
npx playwright show-trace test-results/trace.zip
```

**Trace Contents:**
- Timeline of all actions
- Screenshots at each step
- Network activity
- Console logs
- DOM snapshots

### VS Code Extension

Install **Playwright Test for VSCode**:

1. Search "Playwright Test" in VS Code extensions
2. Install the extension
3. Features:
   - Test explorer sidebar
   - Run/debug individual tests
   - Step through test execution
   - View traces visually

### Browser DevTools

Run tests in headed mode to use browser DevTools:

```bash
npm run test:headed
```

### Console Logging

Add debug output in tests:

```typescript
test('debug test', async ({ page }) => {
  console.log('Current URL:', page.url());

  const text = await page.locator('[data-testid="task-card"]').textContent();
  console.log('Task text:', text);

  await page.screenshot({ path: 'debug-screenshot.png' });
});
```

### Test Configuration Debugging

```typescript
// playwright.config.ts
export default defineConfig({
  use: {
    trace: 'on-first-retry',       // Capture trace on retry
    screenshot: 'only-on-failure', // Screenshot on failure
    video: 'retain-on-failure',    // Video on failure
  },
});
```

---

## Continuous Integration

### CI Configuration

```typescript
// playwright.config.ts
export default defineConfig({
  retries: process.env.CI ? 2 : 0,          // Retry failed tests in CI
  workers: process.env.CI ? 1 : undefined,  // Single worker in CI
  reporter: [
    ['html', { outputFolder: 'test-results/html' }],
    ['json', { outputFile: 'test-results/results.json' }],
    ['list'], // Console output
  ],
});
```

### CI Features

- **Retry Failed Tests**: 2 retries in CI for flaky tests
- **Single Worker**: Prevents race conditions
- **Multiple Reporters**: HTML + JSON + console
- **Artifacts**: Screenshots, videos, traces on failure

### Running Tests in CI

```yaml
# Example GitHub Actions workflow
- name: Run Playwright tests
  run: npm test
  env:
    CI: true

- name: Upload test results
  if: always()
  uses: actions/upload-artifact@v3
  with:
    name: playwright-report
    path: test-results/
```

---

## Best Practices

### 1. Test Naming

Use descriptive test names:

```typescript
// ✅ GOOD - Clear intent
test('should move task from pending to in_progress column', async ({ page }) => {
  // ...
});

// ❌ BAD - Vague
test('drag test', async ({ page }) => {
  // ...
});
```

### 2. Test Organization

Group related tests:

```typescript
test.describe('Task Creation', () => {
  test('should create task with valid data', async ({ page }) => {});
  test('should show error with invalid data', async ({ page }) => {});
  test('should clear form after creation', async ({ page }) => {});
});
```

### 3. Avoid Hardcoded Waits

```typescript
// ✅ GOOD - Wait for condition
await page.waitForSelector('[data-testid="task-card"]');

// ❌ BAD - Arbitrary timeout
await page.waitForTimeout(5000);
```

### 4. Screenshot Comparisons

Allow small pixel differences:

```typescript
await expect(page).toHaveScreenshot('board.png', {
  maxDiffPixels: 100, // Allow minor rendering differences
});
```

### 5. Tag Tests Appropriately

```typescript
test('should pass WCAG audit @accessibility', async ({ page }) => {
  // Accessibility-specific test
});

test('should load quickly @performance', async ({ page }) => {
  // Performance-specific test
});
```

### 6. Clean Up Resources

```typescript
test.afterEach(async ({ page }) => {
  // Clean up test data
  await page.evaluate(() => localStorage.clear());
});
```

---

## Related Documentation

- [Architecture Overview](./ARCHITECTURE.md) - System architecture
- [Component Catalog](./COMPONENTS.md) - Component reference
- [Developer Guide](./DEVELOPER_GUIDE.md) - Development setup
- [UI/UX Patterns](./UI_UX_PATTERNS.md) - Design system
- [Troubleshooting](./TROUBLESHOOTING.md) - Common issues

---

**Last Updated**: 2025-11-04
**Version**: ui2 (Hyperion Coordinator UI)
**Maintainer**: Hyperion Platform Team
