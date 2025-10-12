/**
 * Mock Knowledge Data for Enhanced Knowledge Base Testing
 *
 * Provides comprehensive mock data for E2E testing:
 * - 10 collections across Tech/Task/UI/Ops categories
 * - 20 search results with varied scores (0.3-0.99)
 * - Code blocks in text content for syntax highlighting tests
 * - MSW handlers for API mocking
 */

import { Page } from '@playwright/test';

export interface MockCollection {
  name: string;
  category: 'Tech' | 'Task' | 'UI' | 'Ops';
  count: number;
  description: string;
}

export interface MockSearchResult {
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
    author?: string;
    [key: string]: any;
  };
  createdAt: string;
}

/**
 * Mock Collections (10 total)
 */
export const mockCollections: MockCollection[] = [
  // Tech category (3 collections)
  {
    name: 'technical-knowledge',
    category: 'Tech',
    count: 45,
    description: 'Technical patterns, solutions, and best practices',
  },
  {
    name: 'code-patterns',
    category: 'Tech',
    count: 32,
    description: 'Reusable code patterns and implementation examples',
  },
  {
    name: 'adr',
    category: 'Tech',
    count: 12,
    description: 'Architecture Decision Records',
  },

  // Task category (2 collections)
  {
    name: 'team-coordination',
    category: 'Task',
    count: 28,
    description: 'Cross-squad coordination and handoffs',
  },
  {
    name: 'agent-coordination',
    category: 'Task',
    count: 15,
    description: 'Agent task assignments and progress',
  },

  // UI category (3 collections)
  {
    name: 'ui-component-patterns',
    category: 'UI',
    count: 22,
    description: 'UI component patterns and design systems',
  },
  {
    name: 'ui-test-strategies',
    category: 'UI',
    count: 18,
    description: 'Testing strategies for UI components',
  },
  {
    name: 'ui-accessibility-standards',
    category: 'UI',
    count: 14,
    description: 'Accessibility guidelines and WCAG compliance',
  },

  // Ops category (2 collections)
  {
    name: 'mcp-operations',
    category: 'Ops',
    count: 9,
    description: 'MCP server operations and monitoring',
  },
  {
    name: 'technical-debt-registry',
    category: 'Ops',
    count: 7,
    description: 'Technical debt tracking and resolution plans',
  },
];

/**
 * Mock Search Results (20 total) with varied scores and code blocks
 */
export const mockSearchResults: MockSearchResult[] = [
  {
    id: 'result-1',
    collection: 'technical-knowledge',
    text: `JWT authentication middleware implementation:

\`\`\`go
func ValidateJWT(c *gin.Context) {
    token := c.GetHeader("Authorization")
    claims, err := jwt.Parse(token)
    if err != nil {
        c.JSON(401, gin.H{"error": "invalid_token"})
        return
    }
    c.Set("userId", claims.Subject)
}
\`\`\`

Uses HS256 algorithm with expiration validation.`,
    score: 0.99,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'JWT Authentication Middleware',
      tags: ['jwt', 'authentication', 'security', 'go'],
      author: 'backend-agent',
    },
    createdAt: '2025-10-01T10:00:00Z',
  },
  {
    id: 'result-2',
    collection: 'code-patterns',
    text: `React hooks pattern for state management with TypeScript:

\`\`\`typescript
const useTaskState = () => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(false);

  const addTask = useCallback((task: Task) => {
    setTasks(prev => [...prev, task]);
  }, []);

  return { tasks, loading, addTask };
};
\`\`\``,
    score: 0.95,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'frontend',
      title: 'React Hooks State Management',
      tags: ['react', 'hooks', 'typescript'],
    },
    createdAt: '2025-10-02T14:30:00Z',
  },
  {
    id: 'result-3',
    collection: 'ui-accessibility-standards',
    text: 'WCAG 2.1 AA compliance checklist: Color contrast 4.5:1 for normal text, 3:1 for large text. All interactive elements keyboard accessible. ARIA labels on non-semantic elements. Focus indicators visible with 2px outline or box-shadow.',
    score: 0.92,
    metadata: {
      knowledgeType: 'standard',
      domain: 'ui',
      title: 'WCAG 2.1 AA Checklist',
      tags: ['accessibility', 'wcag', 'compliance'],
    },
    createdAt: '2025-09-20T09:00:00Z',
  },
  {
    id: 'result-4',
    collection: 'ui-test-strategies',
    text: 'Playwright accessibility testing with axe-core: Run automated scans on all pages, test keyboard navigation (Tab/Enter/Escape), verify screen reader attributes (aria-label, role), check color contrast programmatically.',
    score: 0.89,
    metadata: {
      knowledgeType: 'strategy',
      domain: 'testing',
      title: 'Playwright Accessibility Testing',
      tags: ['playwright', 'accessibility', 'axe-core'],
    },
    createdAt: '2025-10-03T11:00:00Z',
  },
  {
    id: 'result-5',
    collection: 'team-coordination',
    text: 'Breaking change: Authentication API updated. New endpoint /api/v2/auth requires userId parameter. Migration deadline: 2025-10-15. All frontend components must update to new format.',
    score: 0.87,
    metadata: {
      knowledgeType: 'coordination',
      domain: 'api',
      title: 'Auth API Breaking Change Notice',
      tags: ['api', 'breaking-change', 'migration'],
      linkedTaskId: 'task-auth-migration-123',
    },
    createdAt: '2025-10-04T08:00:00Z',
  },
  {
    id: 'result-6',
    collection: 'adr',
    text: 'ADR-001: Dual-MCP Architecture. Decision: Use MongoDB-backed coordinator for task tracking, Qdrant for semantic knowledge search. Rationale: Separation of concerns, optimized access patterns, real-time UI updates vs semantic discovery.',
    score: 0.85,
    metadata: {
      knowledgeType: 'decision',
      domain: 'architecture',
      title: 'ADR-001: Dual-MCP Architecture',
      tags: ['adr', 'architecture', 'mcp'],
    },
    createdAt: '2025-09-15T10:00:00Z',
  },
  {
    id: 'result-7',
    collection: 'technical-knowledge',
    text: 'Debounced search implementation with 300ms delay prevents excessive API calls during typing. Use setTimeout, clear previous timeout on each keystroke.',
    score: 0.82,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'frontend',
      title: 'Debounced Search Pattern',
      tags: ['search', 'debounce', 'performance'],
    },
    createdAt: '2025-10-02T11:00:00Z',
  },
  {
    id: 'result-8',
    collection: 'ui-component-patterns',
    text: `Optimistic UI update pattern for task status changes:

\`\`\`typescript
const updateTaskStatus = async (taskId: string, status: string) => {
  // Update UI immediately
  setTasks(prev => prev.map(t => t.id === taskId ? {...t, status} : t));

  try {
    await api.updateTask(taskId, status);
  } catch (error) {
    // Revert on error
    setTasks(prev => prev.map(t => t.id === taskId ? {...t, status: oldStatus} : t));
    showError("Update failed");
  }
};
\`\`\``,
    score: 0.79,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'ui',
      title: 'Optimistic UI Updates',
      tags: ['ui', 'performance', 'state-management'],
    },
    createdAt: '2025-10-01T15:00:00Z',
  },
  {
    id: 'result-9',
    collection: 'mcp-operations',
    text: 'MCP health monitoring: Check /health endpoint every 30s, alert if >3 consecutive failures or response time >2s. Log metrics to Prometheus for observability dashboard.',
    score: 0.76,
    metadata: {
      knowledgeType: 'operation',
      domain: 'ops',
      title: 'MCP Health Monitoring',
      tags: ['mcp', 'monitoring', 'health-check'],
    },
    createdAt: '2025-09-25T11:00:00Z',
  },
  {
    id: 'result-10',
    collection: 'technical-debt-registry',
    text: 'DEBT-045: UserService.go exceeds 400 line limit (currently 523 lines). Impact: Hard to maintain and test. Recommendation: Split into UserService (auth) and UserProfileService (profile).',
    score: 0.72,
    metadata: {
      knowledgeType: 'debt',
      domain: 'backend',
      title: 'UserService File Size Violation',
      tags: ['technical-debt', 'refactoring'],
      severity: 'high',
      filePath: 'backend/services/UserService.go',
    },
    createdAt: '2025-10-02T09:00:00Z',
  },
  {
    id: 'result-11',
    collection: 'agent-coordination',
    text: 'Task handoff: Backend completed /api/tasks endpoint. Returns {tasks: [{id, title, status, assignee}]}. Frontend should implement TaskList component with MUI DataGrid.',
    score: 0.68,
    metadata: {
      knowledgeType: 'handoff',
      domain: 'coordination',
      title: 'Backend-Frontend Task API Handoff',
      tags: ['handoff', 'api', 'coordination'],
      linkedTaskId: 'task-456',
    },
    createdAt: '2025-10-03T13:00:00Z',
  },
  {
    id: 'result-12',
    collection: 'code-patterns',
    text: 'Repository pattern for data access: Interface-based abstraction enables testing with mocks. Inject repository into services via dependency injection.',
    score: 0.65,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'Repository Pattern',
      tags: ['pattern', 'data-access', 'architecture'],
    },
    createdAt: '2025-09-28T10:00:00Z',
  },
  {
    id: 'result-13',
    collection: 'ui-accessibility-standards',
    text: 'Keyboard navigation: Tab for sequential navigation, Enter/Space for activation, Arrow keys for directional movement, Escape to close modals/dropdowns. Focus order must be logical.',
    score: 0.62,
    metadata: {
      knowledgeType: 'standard',
      domain: 'ui',
      title: 'Keyboard Navigation Standards',
      tags: ['accessibility', 'keyboard', 'navigation'],
    },
    createdAt: '2025-09-22T14:00:00Z',
  },
  {
    id: 'result-14',
    collection: 'technical-knowledge',
    text: 'MongoDB secure client: Extract user identity from JWT context using auth.GetIdentityFromContext. Pass to SecureMongoClient with IdentityProvider. Never use system service identities.',
    score: 0.58,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'MongoDB Secure Client',
      tags: ['mongodb', 'security', 'authentication'],
    },
    createdAt: '2025-10-01T12:00:00Z',
  },
  {
    id: 'result-15',
    collection: 'ui-test-strategies',
    text: 'Visual regression testing: Capture screenshots of key UI states (empty, loading, success, error), compare with baseline using Percy. Flag differences >0.1% for manual review.',
    score: 0.54,
    metadata: {
      knowledgeType: 'strategy',
      domain: 'testing',
      title: 'Visual Regression Testing',
      tags: ['testing', 'visual-regression', 'percy'],
    },
    createdAt: '2025-10-01T14:00:00Z',
  },
  {
    id: 'result-16',
    collection: 'adr',
    text: 'ADR-002: Material-UI component library adoption. Decision: Use MUI v7 for all UI components. Rationale: Comprehensive component set, built-in accessibility, active maintenance, TypeScript support.',
    score: 0.50,
    metadata: {
      knowledgeType: 'decision',
      domain: 'frontend',
      title: 'ADR-002: Material-UI Adoption',
      tags: ['adr', 'ui', 'material-ui'],
    },
    createdAt: '2025-09-10T10:00:00Z',
  },
  {
    id: 'result-17',
    collection: 'team-coordination',
    text: 'Database migration scheduled: User table adds emailVerified, lastLoginAt columns on 2025-10-15. Backend migration script ready. Frontend should handle new fields gracefully with fallbacks.',
    score: 0.46,
    metadata: {
      knowledgeType: 'coordination',
      domain: 'database',
      title: 'User Schema Migration Notice',
      tags: ['database', 'migration', 'coordination'],
    },
    createdAt: '2025-10-04T08:00:00Z',
  },
  {
    id: 'result-18',
    collection: 'mcp-operations',
    text: 'MCP deployment checklist: Build Docker image, push to registry, update K8s deployment, verify health endpoint, check logs, monitor metrics for 1 hour post-deployment.',
    score: 0.42,
    metadata: {
      knowledgeType: 'operation',
      domain: 'ops',
      title: 'MCP Deployment Checklist',
      tags: ['deployment', 'mcp', 'operations'],
    },
    createdAt: '2025-09-30T10:00:00Z',
  },
  {
    id: 'result-19',
    collection: 'code-patterns',
    text: 'Error handling: Never use silent fallbacks. Return explicit errors with context. Log with structured fields. Display user-friendly messages in UI while preserving error details for debugging.',
    score: 0.38,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'backend',
      title: 'Error Handling Best Practices',
      tags: ['error-handling', 'api', 'logging'],
    },
    createdAt: '2025-10-02T10:00:00Z',
  },
  {
    id: 'result-20',
    collection: 'ui-component-patterns',
    text: 'Form validation pattern: Real-time validation with Formik, schema validation with Yup. Display inline errors below fields. Include ARIA attributes for screen readers: aria-invalid, aria-describedby.',
    score: 0.34,
    metadata: {
      knowledgeType: 'pattern',
      domain: 'ui',
      title: 'Form Validation Pattern',
      tags: ['forms', 'validation', 'accessibility'],
    },
    createdAt: '2025-10-02T16:00:00Z',
  },
];

/**
 * Filter search results by query and collection
 */
export function filterSearchResults(
  query: string,
  collectionFilter?: string,
  limit: number = 10
): MockSearchResult[] {
  let results = [...mockSearchResults];

  // Filter by collection
  if (collectionFilter && collectionFilter !== '' && collectionFilter !== 'all') {
    results = results.filter(r => r.collection === collectionFilter);
  }

  // Filter by query (case-insensitive)
  if (query.trim()) {
    const queryLower = query.toLowerCase();
    results = results.filter(r =>
      r.text.toLowerCase().includes(queryLower) ||
      r.metadata.title?.toLowerCase().includes(queryLower) ||
      r.metadata.tags?.some(tag => tag.toLowerCase().includes(queryLower))
    );
  }

  // Sort by score descending
  results.sort((a, b) => b.score - a.score);

  // Apply limit
  return results.slice(0, limit);
}

/**
 * Setup MSW-style handlers for knowledge API
 * Uses Playwright's route interception
 */
export async function setupKnowledgeMocks(page: Page): Promise<void> {
  // GET /api/v1/knowledge/collections
  await page.route('**/api/v1/knowledge/collections', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ collections: mockCollections }),
    });
  });

  // POST /api/v1/knowledge/search
  await page.route('**/api/v1/knowledge/search', async (route) => {
    const method = route.request().method();

    if (method === 'POST') {
      const postData = route.request().postDataJSON();
      const { query = '', collection = '', limit = 10 } = postData || {};

      const results = filterSearchResults(query, collection, limit);

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          results,
          total: results.length,
        }),
      });
    } else {
      await route.continue();
    }
  });

  // GET /api/knowledge/search (query params)
  await page.route('**/api/knowledge/search**', async (route) => {
    const url = new URL(route.request().url());
    const query = url.searchParams.get('query') || '';
    const collection = url.searchParams.get('collection') || '';
    const limit = parseInt(url.searchParams.get('limit') || '10');

    const results = filterSearchResults(query, collection, limit);

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        results,
        total: results.length,
      }),
    });
  });

  // POST /api/v1/knowledge (create knowledge entry)
  await page.route('**/api/v1/knowledge', async (route) => {
    if (route.request().method() === 'POST') {
      const postData = route.request().postDataJSON();

      // Validate required fields
      if (!postData.collection || !postData.text || postData.text.length < 10) {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Validation failed',
            message: 'Collection and text (min 10 chars) required',
          }),
        });
        return;
      }

      // Success response
      await route.fulfill({
        status: 201,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          id: `entry-${Date.now()}`,
          message: 'Knowledge created successfully',
        }),
      });
    } else {
      await route.continue();
    }
  });
}

/**
 * Setup error responses for error handling tests
 */
export async function setupKnowledgeMocksWithErrors(page: Page): Promise<void> {
  await page.route('**/api/v1/knowledge/collections', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        error: 'Internal server error',
        message: 'Failed to fetch collections',
      }),
    });
  });

  await page.route('**/api/v1/knowledge/search', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        error: 'Internal server error',
        message: 'Failed to search knowledge',
      }),
    });
  });

  await page.route('**/api/knowledge/search**', async (route) => {
    await route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({
        error: 'Internal server error',
        message: 'Failed to search knowledge',
      }),
    });
  });
}

/**
 * Helper to wait for debounce (300ms delay)
 */
export async function waitForDebounce(page: Page): Promise<void> {
  await page.waitForTimeout(350); // 300ms debounce + 50ms buffer
}
