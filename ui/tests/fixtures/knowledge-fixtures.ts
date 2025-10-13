/**
 * Knowledge Base Test Fixtures
 *
 * Mock data and utilities for testing knowledge base UI components:
 * - Mock Qdrant collections across all categories
 * - Mock knowledge entries with varying scores and metadata
 * - Mock search results filtered by query
 * - API route interception helpers
 */

import { Page } from '@playwright/test';

/**
 * Knowledge Collection Interface
 */
export interface KnowledgeCollection {
  name: string;
  category: 'Tech' | 'Task' | 'UI' | 'Ops';
  count: number;
  description?: string;
}

/**
 * Knowledge Entry Interface
 */
export interface KnowledgeEntry {
  id: string;
  collection: string;
  text: string;
  score: number;
  metadata: {
    knowledgeType?: string;
    domain?: string;
    title?: string;
    tags?: string[];
    linkedTaskId?: string;
    createdAt?: string;
    [key: string]: any;
  };
}

/**
 * Mock Collections - 10 collections across all categories
 */
export const mockCollections: KnowledgeCollection[] = [
  // Tech category
  { name: 'technical-knowledge', category: 'Tech', count: 45, description: 'General technical patterns and solutions' },
  { name: 'code-patterns', category: 'Tech', count: 32, description: 'Reusable code patterns and examples' },
  { name: 'adr', category: 'Tech', count: 12, description: 'Architecture decision records' },

  // Task category
  { name: 'team-coordination', category: 'Task', count: 28, description: 'Cross-squad coordination messages' },
  { name: 'agent-coordination', category: 'Task', count: 15, description: 'Agent task coordination' },

  // UI category
  { name: 'ui-component-patterns', category: 'UI', count: 22, description: 'UI component patterns and best practices' },
  { name: 'ui-test-strategies', category: 'UI', count: 18, description: 'Testing strategies for UI components' },
  { name: 'ui-accessibility-standards', category: 'UI', count: 14, description: 'Accessibility guidelines and checks' },

  // Ops category
  { name: 'mcp-operations', category: 'Ops', count: 9, description: 'MCP server operations and monitoring' },
  { name: 'technical-debt-registry', category: 'Ops', count: 7, description: 'Technical debt tracking and resolution' },
];

/**
 * Mock Knowledge Entries - 20 entries with varying scores
 */
export const mockKnowledgeEntries: KnowledgeEntry[] = [
  {
    id: 'entry-1',
    collection: 'technical-knowledge',
    text: 'JWT authentication implementation using HS256 algorithm with token validation and error handling. Middleware extracts Bearer token from Authorization header, validates signature, expiration, and claims.',
    score: 0.95,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'JWT Authentication Middleware',
      tags: ['jwt', 'authentication', 'security'],
      createdAt: '2025-10-01T10:00:00Z',
    },
  },
  {
    id: 'entry-2',
    collection: 'code-patterns',
    text: 'React component pattern using hooks for state management. useEffect for side effects, useState for local state, useCallback for memoized callbacks. Follows single responsibility principle.',
    score: 0.92,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'frontend',
      title: 'React Hooks Pattern',
      tags: ['react', 'hooks', 'state-management'],
      createdAt: '2025-10-02T14:30:00Z',
    },
  },
  {
    id: 'entry-3',
    collection: 'ui-component-patterns',
    text: 'Material-UI form validation pattern with real-time error feedback. Uses Formik for form state, Yup for validation schema. Displays inline errors below each field with proper ARIA attributes.',
    score: 0.89,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'ui',
      title: 'MUI Form Validation',
      tags: ['mui', 'forms', 'validation'],
      createdAt: '2025-10-02T16:00:00Z',
    },
  },
  {
    id: 'entry-4',
    collection: 'team-coordination',
    text: 'Cross-squad API change notification: Backend squad updated user authentication endpoint to require new userId parameter. All frontend components must be updated by 2025-10-10.',
    score: 0.88,
    metadata: {
      knowledgeType: 'coordination',
      domain: 'backend-frontend',
      title: 'Auth API Breaking Change',
      tags: ['api', 'breaking-change', 'authentication'],
      linkedTaskId: 'task-123',
      createdAt: '2025-10-03T09:00:00Z',
    },
  },
  {
    id: 'entry-5',
    collection: 'ui-test-strategies',
    text: 'Playwright accessibility testing strategy using @axe-core/playwright. Run automated axe audit on each component, test keyboard navigation with Tab/Enter/Space, verify ARIA labels and roles.',
    score: 0.87,
    metadata: {
      knowledgeType: 'strategy',
      domain: 'testing',
      title: 'Accessibility Testing with Playwright',
      tags: ['playwright', 'accessibility', 'testing'],
      createdAt: '2025-10-03T11:00:00Z',
    },
  },
  {
    id: 'entry-6',
    collection: 'adr',
    text: 'ADR-001: Adopt dual-MCP architecture. Decision: Use MongoDB-backed coordinator for task tracking and Qdrant for semantic knowledge search. Rationale: Separation of concerns, optimized for different access patterns.',
    score: 0.85,
    metadata: {
      knowledgeType: 'decision',
      domain: 'architecture',
      title: 'ADR-001: Dual-MCP Architecture',
      tags: ['adr', 'architecture', 'mcp'],
      createdAt: '2025-09-15T10:00:00Z',
    },
  },
  {
    id: 'entry-7',
    collection: 'technical-knowledge',
    text: 'MongoDB secure client implementation with user identity from JWT context. Extract identity using auth.GetIdentityFromContext, pass to SecureMongoClient with IdentityProvider. Never use system service identities.',
    score: 0.84,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'MongoDB Secure Client Pattern',
      tags: ['mongodb', 'security', 'authentication'],
      createdAt: '2025-10-01T12:00:00Z',
    },
  },
  {
    id: 'entry-8',
    collection: 'ui-accessibility-standards',
    text: 'WCAG 2.1 AA compliance checklist: Color contrast 4.5:1 for text, keyboard navigation for all interactive elements, ARIA labels on non-semantic elements, focus indicators visible, heading hierarchy logical.',
    score: 0.83,
    metadata: {
      knowledgeType: 'standard',
      domain: 'ui',
      title: 'WCAG 2.1 AA Checklist',
      tags: ['accessibility', 'wcag', 'standards'],
      createdAt: '2025-09-20T09:00:00Z',
    },
  },
  {
    id: 'entry-9',
    collection: 'code-patterns',
    text: 'Error handling pattern for API calls: Use try-catch with specific error types, return descriptive error messages, log errors with context, display user-friendly messages in UI. Never silent fallbacks.',
    score: 0.82,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'API Error Handling Pattern',
      tags: ['error-handling', 'api', 'backend'],
      createdAt: '2025-10-02T10:00:00Z',
    },
  },
  {
    id: 'entry-10',
    collection: 'ui-component-patterns',
    text: 'Optimistic UI update pattern: Update local state immediately, show loading indicator, revert on error with error message, sync with server response. Improves perceived performance.',
    score: 0.81,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'ui',
      title: 'Optimistic UI Updates',
      tags: ['ui', 'performance', 'state-management'],
      createdAt: '2025-10-01T15:00:00Z',
    },
  },
  {
    id: 'entry-11',
    collection: 'mcp-operations',
    text: 'MCP server health monitoring: Check /health endpoint every 30s, log response time and status, alert if >3 consecutive failures or response time >2s. Use Prometheus metrics.',
    score: 0.79,
    metadata: {
      knowledgeType: 'operation',
      domain: 'ops',
      title: 'MCP Health Monitoring',
      tags: ['mcp', 'monitoring', 'health-check'],
      createdAt: '2025-09-25T11:00:00Z',
    },
  },
  {
    id: 'entry-12',
    collection: 'technical-debt-registry',
    text: 'DEBT-045: UserService.go exceeds 400 line limit (currently 523 lines). Impact: Hard to maintain and test. Recommended: Split into UserService (auth) and UserProfileService (profile management).',
    score: 0.78,
    metadata: {
      knowledgeType: 'debt',
      domain: 'backend',
      title: 'UserService File Size Violation',
      tags: ['technical-debt', 'refactoring', 'backend'],
      filePath: 'backend/services/UserService.go',
      severity: 'high',
      createdAt: '2025-10-02T09:00:00Z',
    },
  },
  {
    id: 'entry-13',
    collection: 'agent-coordination',
    text: 'Task handoff from Backend Agent to Frontend Agent: API endpoint /api/tasks completed. Returns JSON with tasks array, each task has id, title, status, assignee. Frontend should implement TaskList component.',
    score: 0.77,
    metadata: {
      knowledgeType: 'handoff',
      domain: 'coordination',
      title: 'Backend-Frontend Task API Handoff',
      tags: ['handoff', 'api', 'coordination'],
      linkedTaskId: 'task-456',
      createdAt: '2025-10-03T13:00:00Z',
    },
  },
  {
    id: 'entry-14',
    collection: 'ui-test-strategies',
    text: 'Visual regression testing with Percy: Capture screenshots of key UI states (empty, loading, success, error), compare with baseline, flag differences >0.1%. Run on PRs before merge.',
    score: 0.76,
    metadata: {
      knowledgeType: 'strategy',
      domain: 'testing',
      title: 'Visual Regression with Percy',
      tags: ['testing', 'visual-regression', 'percy'],
      createdAt: '2025-10-01T14:00:00Z',
    },
  },
  {
    id: 'entry-15',
    collection: 'technical-knowledge',
    text: 'Debounced search implementation: Use setTimeout to delay API call by 300ms after last keystroke. Clear previous timeout on each keystroke. Prevents excessive API calls during typing.',
    score: 0.75,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'frontend',
      title: 'Debounced Search Pattern',
      tags: ['search', 'debounce', 'performance'],
      createdAt: '2025-10-02T11:00:00Z',
    },
  },
  {
    id: 'entry-16',
    collection: 'adr',
    text: 'ADR-002: Use Material-UI for component library. Decision: Adopt MUI v7 for all UI components. Rationale: Comprehensive component set, accessibility built-in, active maintenance, TypeScript support.',
    score: 0.74,
    metadata: {
      knowledgeType: 'decision',
      domain: 'frontend',
      title: 'ADR-002: Material-UI Adoption',
      tags: ['adr', 'ui', 'material-ui'],
      createdAt: '2025-09-10T10:00:00Z',
    },
  },
  {
    id: 'entry-17',
    collection: 'code-patterns',
    text: 'Repository pattern for data access: Separate data access logic into repository layer, inject repository into services via dependency injection. Enables easy testing with mock repositories.',
    score: 0.72,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'Repository Pattern',
      tags: ['pattern', 'data-access', 'architecture'],
      createdAt: '2025-09-28T10:00:00Z',
    },
  },
  {
    id: 'entry-18',
    collection: 'ui-accessibility-standards',
    text: 'Keyboard navigation best practices: Tab for sequential navigation, Enter/Space for activation, Arrow keys for directional navigation within components, Escape to close modals/dropdowns.',
    score: 0.71,
    metadata: {
      knowledgeType: 'standard',
      domain: 'ui',
      title: 'Keyboard Navigation Standards',
      tags: ['accessibility', 'keyboard', 'navigation'],
      createdAt: '2025-09-22T14:00:00Z',
    },
  },
  {
    id: 'entry-19',
    collection: 'team-coordination',
    text: 'Database schema migration scheduled for 2025-10-15: User table adds new columns (emailVerified, lastLoginAt). Backend migration script ready. Frontend should handle new fields gracefully.',
    score: 0.70,
    metadata: {
      knowledgeType: 'coordination',
      domain: 'database',
      title: 'User Schema Migration Notice',
      tags: ['database', 'migration', 'coordination'],
      createdAt: '2025-10-04T08:00:00Z',
    },
  },
  {
    id: 'entry-20',
    collection: 'mcp-operations',
    text: 'MCP server deployment checklist: Build Docker image, push to registry, update Kubernetes deployment, verify health endpoint, check logs for errors, monitor metrics for 1 hour.',
    score: 0.68,
    metadata: {
      knowledgeType: 'operation',
      domain: 'ops',
      title: 'MCP Deployment Checklist',
      tags: ['deployment', 'mcp', 'operations'],
      createdAt: '2025-09-30T10:00:00Z',
    },
  },
];

/**
 * Mock search results - filters entries by query and collection
 */
export function mockSearchResults(
  query: string,
  collectionFilter?: string
): KnowledgeEntry[] {
  let results = [...mockKnowledgeEntries];

  // Filter by collection if specified
  if (collectionFilter && collectionFilter !== 'all') {
    results = results.filter(entry => entry.collection === collectionFilter);
  }

  // Filter by query (case-insensitive text search)
  if (query.trim()) {
    const queryLower = query.toLowerCase();
    results = results.filter(entry =>
      entry.text.toLowerCase().includes(queryLower) ||
      entry.metadata.title?.toLowerCase().includes(queryLower) ||
      entry.metadata.tags?.some(tag => tag.toLowerCase().includes(queryLower))
    );
  }

  // Sort by score descending
  return results.sort((a, b) => b.score - a.score);
}

/**
 * Setup API route interception for knowledge endpoints
 */
export async function setupKnowledgeAPI(page: Page): Promise<void> {
  // Mock GET /api/knowledge/collections
  await page.route('**/api/knowledge/collections', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ collections: mockCollections }),
    });
  });

  // Mock GET /api/knowledge/search
  await page.route('**/api/knowledge/search**', async (route) => {
    const url = new URL(route.request().url());
    const query = url.searchParams.get('query') || '';
    const collection = url.searchParams.get('collection') || 'all';

    const results = mockSearchResults(query, collection);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results,
        total: results.length,
      }),
    });
  });

  // Mock POST /api/knowledge
  await page.route('**/api/knowledge', async (route) => {
    if (route.request().method() === 'POST') {
      const postData = route.request().postDataJSON();

      // Simulate validation
      if (!postData.collection || !postData.text || postData.text.length < 10) {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Validation failed',
            message: 'Collection and text (min 10 chars) are required',
          }),
        });
        return;
      }

      // Simulate successful creation
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          id: `entry-${Date.now()}`,
          message: 'Knowledge created successfully',
        }),
      });
    }
  });
}

/**
 * Setup API with error responses for error handling tests
 */
export async function setupKnowledgeAPIWithErrors(page: Page): Promise<void> {
  // Mock search endpoint with error
  await page.route('**/api/knowledge/search**', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        error: 'Internal server error',
        message: 'Failed to query Qdrant',
      }),
    });
  });

  // Mock create endpoint with error
  await page.route('**/api/knowledge', async (route) => {
    if (route.request().method() === 'POST') {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({
          error: 'Internal server error',
          message: 'Failed to store knowledge in Qdrant',
        }),
      });
    }
  });

  // Mock collections endpoint with error
  await page.route('**/api/knowledge/collections', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        error: 'Internal server error',
        message: 'Failed to fetch collections',
      }),
    });
  });
}

/**
 * Reset knowledge data to initial state
 */
export function resetKnowledgeData(): void {
  // Reset mock data to original state
  // In a real implementation, this would clear test database or reset mocks
  // For now, since we're using route interception, no state to reset
}

/**
 * Helper to wait for debounced search (300ms)
 */
export async function waitForDebounce(page: Page): Promise<void> {
  await page.waitForTimeout(350); // 300ms debounce + 50ms buffer
}
