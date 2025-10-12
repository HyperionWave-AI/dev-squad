# Kanban Board Test Suite

Comprehensive Playwright test suite for the Hyperion Coordinator MUI Kanban board.

## Test Coverage

### 1. Rendering Tests (`kanban/rendering.spec.ts`)
- Column structure validation (pending, in_progress, completed, blocked)
- Task card rendering
- MUI component presence (Cards, AppBar, Chips)
- Loading states
- Error handling
- Empty states
- Task metadata display

### 2. Drag-and-Drop Tests (`kanban/drag-drop.spec.ts`)
- Move tasks between columns
- Drag visual feedback
- Drop target highlighting
- Task position updates
- API integration (status updates)
- Edge cases (invalid drops, rapid movements, ESC cancellation)
- Persistence after page reload

### 3. Responsive Design Tests (`kanban/responsive.spec.ts`)
- Desktop layout (1920px)
- Tablet layout (768px)
- Mobile layout (375px)
- Touch interactions
- Horizontal scrolling
- Viewport transitions
- Layout adaptation

### 4. Accessibility Tests (`kanban/accessibility.spec.ts`)
- Axe-core WCAG 2.1 AA compliance
- Keyboard navigation (Tab, Enter, Space, Arrow keys)
- Screen reader compatibility (ARIA labels, roles, live regions)
- Color contrast validation
- Focus management
- Drag-and-drop keyboard accessibility
- Heading hierarchy

### 5. Visual Regression Tests (`kanban/visual-regression.spec.ts`)
- Screenshot comparison for full board
- Column visual consistency
- Task card styling
- Priority badge colors (red, yellow, green)
- Responsive layout screenshots
- MUI Card elevation shadows
- Empty/error/loading states

### 6. MUI Component Tests (`kanban/mui-components.spec.ts`)
- MUI AppBar and Toolbar
- MUI Card and CardContent
- MUI Chip (priority badges)
- MUI Button and IconButton
- MUI Grid layout
- MUI Typography
- MUI Paper elevation
- MUI Avatar (if present)
- Theme integration

### 7. Concurrent Updates Tests (`kanban/concurrent-updates.spec.ts`)
- UI polling (3-second intervals)
- Real-time data synchronization
- Concurrent user actions
- Optimistic UI updates
- Network failure recovery
- Debouncing
- Scroll position maintenance

### 8. Filtering and Search Tests (`kanban/filtering-search.spec.ts`)
- Search by task prompt/description
- Filter by status
- Filter by priority
- Filter by agent name
- Combined filters
- Clear filters
- Search results highlighting
- No results message

## Running Tests

### All Tests
```bash
npm test
```

### Headed Mode (see browser)
```bash
npm run test:headed
```

### Interactive UI Mode
```bash
npm run test:ui
```

### Debug Mode (step through tests)
```bash
npm run test:debug
```

### Accessibility Tests Only
```bash
npm run test:accessibility
```

### Mobile Tests Only
```bash
npm run test:mobile
```

### Tablet Tests Only
```bash
npm run test:tablet
```

### Desktop Tests Only
```bash
npm run test:desktop
```

### View Test Report
```bash
npm run test:report
```

## Test Projects

Tests run across multiple browser/viewport configurations:

1. **chromium-desktop** - Desktop Chrome (1920x1080)
2. **webkit-desktop** - Desktop Safari (1920x1080)
3. **tablet** - iPad Pro (768x1024)
4. **mobile** - iPhone 13 (375x812)
5. **accessibility** - Chromium with axe-core (tests tagged with @accessibility)

## Test Structure

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
│   └── mockTasks.ts           # Mock task data
├── utils/                      # Test utilities
│   └── accessibility.ts       # Accessibility helpers
└── README.md                   # This file
```

## Test Data

Mock data is provided in `fixtures/mockTasks.ts`:
- 4 human tasks (one per status)
- 4 agent tasks (linked to human tasks)
- Priority test cases (high, medium, low)
- Column definitions

## Accessibility Testing

Accessibility tests use:
- **@axe-core/playwright** for automated WCAG audits
- Custom keyboard navigation utilities
- Screen reader attribute verification
- Color contrast validation

All accessibility tests are tagged with `@accessibility` for easy filtering.

## Visual Regression Testing

Visual tests create baseline screenshots in `test-results/screenshots/`:
- Full board layouts (desktop, tablet, mobile)
- Individual column screenshots
- Task card styling
- Priority badge colors
- Empty/error/loading states

## Continuous Integration

Tests are configured for CI/CD:
- Retry failed tests 2 times in CI
- Single worker in CI for stability
- HTML and JSON reports generated
- Screenshots and videos on failure

## Requirements

- Node.js 18+
- Playwright browsers (automatically installed)
- UI server running at http://localhost:5173 (auto-started by Playwright)

## Test Configuration

Configuration in `playwright.config.ts`:
- 30-second test timeout
- Traces on first retry
- Screenshots on failure
- Videos retained on failure
- Auto-start dev server
- Multiple browser projects

## Debugging Tests

### VSCode Extension
Install the Playwright Test for VSCode extension for:
- Test explorer UI
- Run/debug individual tests
- Step through test execution
- View traces visually

### Playwright Inspector
```bash
npm run test:debug
```

This opens the Playwright Inspector to:
- Step through test actions
- Inspect locators
- View console logs
- Examine network requests

### Trace Viewer
After a test failure:
```bash
npx playwright show-trace test-results/trace.zip
```

## Best Practices

1. **Wait for elements** - Use `waitForSelector` with reasonable timeouts
2. **Test isolation** - Each test is independent and can run in any order
3. **Data attributes** - Use `data-testid` attributes for stable selectors
4. **Mock responses** - Use `page.route()` to mock API responses when needed
5. **Screenshot comparison** - Allow small pixel differences for visual tests
6. **Accessibility first** - Run accessibility tests on every PR

## Coverage Report

After running tests, view the HTML report:
```bash
npm run test:report
```

Report includes:
- Test pass/fail status
- Execution duration
- Screenshots on failure
- Trace files for debugging
- Accessibility violations

## Contributing

When adding new tests:
1. Follow existing naming conventions
2. Add comprehensive test documentation
3. Use appropriate test fixtures
4. Tag accessibility tests with `@accessibility`
5. Include visual regression baselines
6. Test across all viewport sizes

## Known Limitations

- Visual regression tests may have minor pixel differences across machines
- Drag-and-drop keyboard simulation has limited browser support
- WebSocket testing requires additional setup
- Some MUI components may render differently in Webkit

## Support

For issues or questions:
- Check test output and screenshots
- Review trace files for failures
- Consult Playwright documentation: https://playwright.dev
- Check MUI component docs: https://mui.com