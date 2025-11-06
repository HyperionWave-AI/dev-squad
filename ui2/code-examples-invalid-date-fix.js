/**
 * Code Examples: Invalid Date Issue Fix
 * 
 * This file demonstrates the problematic code and provides
 * concrete implementation examples for the recommended fixes.
 */

// ============================================================================
// CURRENT PROBLEMATIC CODE
// ============================================================================

/**
 * PROBLEM 1: API Interface (chatService.ts)
 * The API returns sessions with createdAt/updatedAt but UI expects timestamp
 */
export interface ChatSession {
  id: string;
  userId: string;
  companyId: string;
  title: string;
  parentChatId?: string;
  createdAt: string;    // ← API provides this
  updatedAt: string;    // ← API provides this
  // timestamp: missing! ← UI expects this
}

/**
 * PROBLEM 2: SessionList Interface (SessionList.tsx)
 * Component expects different field name than API provides
 */
interface ChatSession {
  id: string;
  title: string;
  lastMessage?: string;
  timestamp: Date | string;  // ← Expected but not provided
  messageCount: number;
}

/**
 * PROBLEM 3: Date Formatting Logic (SessionList.tsx lines 158-169)
 * Tries to access undefined session.timestamp
 */
{(() => {
  try {
    const date = typeof session.timestamp === 'string'  // ← session.timestamp is undefined!
      ? new Date(session.timestamp)
      : session.timestamp;
    return isNaN(date.getTime())
      ? 'Invalid date'  // ← This is what users see
      : formatDistanceToNow(date, { addSuffix: true });
  } catch {
    return 'Invalid date';
  }
})()}

// ============================================================================
// RECOMMENDED FIX OPTION 1: Map Data in CodeChatPage (RECOMMENDED)
// ============================================================================

/**
 * FIX 1A: Update loadSessions function in CodeChatPage.tsx
 * Transform API response to match SessionList expectations
 */

// BEFORE (current problematic code):
const loadSessions = async () => {
  try {
    const fetchedSessions = await getSessions();
    setSessions(fetchedSessions); // ← Direct pass-through causes issue
    
    if (!activeSessionIdRef.current && fetchedSessions.length > 0) {
      setActiveSessionId(fetchedSessions[0].id);
      await loadMessages(fetchedSessions[0].id);
    }
  } catch (err) {
    console.error('[CodeChatPage] Error loading sessions:', err);
    setError(err instanceof Error ? err.message : 'Failed to load sessions');
  }
};

// AFTER (fixed version):
const loadSessions = async () => {
  try {
    const fetchedSessions = await getSessions();
    
    // ✅ FIX: Map API response to SessionList interface
    const mappedSessions = fetchedSessions.map(session => ({
      id: session.id,
      title: session.title,
      lastMessage: undefined, // TODO: Add from API when available
      timestamp: session.updatedAt || session.createdAt, // ← Use updatedAt for recency
      messageCount: 0, // TODO: Add from API when available
      // Preserve original fields for other uses
      userId: session.userId,
      companyId: session.companyId,
      parentChatId: session.parentChatId,
      createdAt: session.createdAt,
      updatedAt: session.updatedAt,
    }));
    
    setSessions(mappedSessions);
    
    if (!activeSessionIdRef.current && mappedSessions.length > 0) {
      setActiveSessionId(mappedSessions[0].id);
      await loadMessages(mappedSessions[0].id);
    }
  } catch (err) {
    console.error('[CodeChatPage] Error loading sessions:', err);
    setError(err instanceof Error ? err.message : 'Failed to load sessions');
  }
};

/**
 * FIX 1B: Update TypeScript interfaces for consistency
 * Create a unified interface that works for both API and UI
 */

// In a shared types file (e.g., types/chat.ts):
export interface ApiChatSession {
  id: string;
  userId: string;
  companyId: string;
  title: string;
  parentChatId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UiChatSession extends ApiChatSession {
  timestamp: string;      // Computed from updatedAt || createdAt
  lastMessage?: string;   // From last message in session
  messageCount: number;   // Count of messages in session
}

// ============================================================================
// ALTERNATIVE FIX OPTION 2: Update SessionList Component
// ============================================================================

/**
 * FIX 2A: Modify SessionList to use createdAt/updatedAt directly
 * Change the component to work with API structure
 */

// Update SessionList interface:
interface ChatSession {
  id: string;
  title: string;
  lastMessage?: string;
  createdAt: string;    // ← Use API field names
  updatedAt: string;    // ← Use API field names
  messageCount: number;
}

// Update date formatting logic in SessionList.tsx:
<div className="flex items-center gap-1">
  <Clock className="w-3 h-3" />
  {(() => {
    try {
      // ✅ FIX: Use updatedAt (or createdAt as fallback) instead of timestamp
      const dateString = session.updatedAt || session.createdAt;
      const date = new Date(dateString);
      return isNaN(date.getTime())
        ? 'Invalid date'
        : formatDistanceToNow(date, { addSuffix: true });
    } catch {
      return 'Invalid date';
    }
  })()}
</div>

// ============================================================================
// ALTERNATIVE FIX OPTION 3: Update API Service Layer
// ============================================================================

/**
 * FIX 3A: Modify getSessions to add computed timestamp field
 * Transform data at the service layer
 */

// Update getSessions function in chatService.ts:
export async function getSessions(): Promise<ChatSession[]> {
  const response = await fetchJSON<{ sessions: ChatSession[]; count: number }>('/chat/sessions', {
    method: 'GET',
  });

  // ✅ FIX: Add computed fields for UI compatibility
  const enhancedSessions = (response.sessions || []).map(session => ({
    ...session,
    timestamp: session.updatedAt || session.createdAt,
    messageCount: 0, // TODO: Compute from messages when API supports it
  }));

  return enhancedSessions;
}

/**
 * FIX 3B: Update ChatSession interface to include UI fields
 */
export interface ChatSession {
  id: string;
  userId: string;
  companyId: string;
  title: string;
  parentChatId?: string;
  createdAt: string;
  updatedAt: string;
  // ✅ FIX: Add computed fields for UI
  timestamp: string;      // Computed from updatedAt || createdAt
  messageCount: number;   // Count of messages (TODO: from API)
}

// ============================================================================
// ENHANCED ERROR HANDLING (BONUS IMPROVEMENT)
// ============================================================================

/**
 * BONUS: Improved date formatting with better error handling and logging
 */
const formatSessionDate = (session: ChatSession): string => {
  try {
    // Try multiple date sources in order of preference
    const dateString = session.timestamp || session.updatedAt || session.createdAt;
    
    if (!dateString) {
      console.warn(`[SessionList] No date available for session ${session.id}`);
      return 'No date';
    }
    
    const date = new Date(dateString);
    
    if (isNaN(date.getTime())) {
      console.warn(`[SessionList] Invalid date for session ${session.id}:`, dateString);
      return 'Invalid date';
    }
    
    return formatDistanceToNow(date, { addSuffix: true });
  } catch (error) {
    console.error(`[SessionList] Date formatting error for session ${session.id}:`, error);
    return 'Date error';
  }
};

// Usage in SessionList component:
<div className="flex items-center gap-1">
  <Clock className="w-3 h-3" />
  {formatSessionDate(session)}
</div>

// ============================================================================
// TESTING EXAMPLES
// ============================================================================

/**
 * Unit test examples for the fixes
 */

// Test data mapping function:
describe('Session Data Mapping', () => {
  it('should map API session to UI session correctly', () => {
    const apiSession = {
      id: 'test-123',
      userId: 'user-456',
      companyId: 'company-789',
      title: 'Test Chat',
      createdAt: '2024-01-15T10:30:00Z',
      updatedAt: '2024-01-15T11:45:00Z',
    };
    
    const uiSession = mapApiSessionToUi(apiSession);
    
    expect(uiSession.timestamp).toBe('2024-01-15T11:45:00Z');
    expect(uiSession.title).toBe('Test Chat');
    expect(uiSession.messageCount).toBe(0);
  });
  
  it('should handle missing updatedAt gracefully', () => {
    const apiSession = {
      id: 'test-123',
      userId: 'user-456',
      companyId: 'company-789',
      title: 'Test Chat',
      createdAt: '2024-01-15T10:30:00Z',
      updatedAt: null,
    };
    
    const uiSession = mapApiSessionToUi(apiSession);
    
    expect(uiSession.timestamp).toBe('2024-01-15T10:30:00Z');
  });
});

// Test date formatting function:
describe('Date Formatting', () => {
  it('should format valid dates correctly', () => {
    const session = {
      id: 'test',
      timestamp: '2024-01-15T10:30:00Z',
    };
    
    const result = formatSessionDate(session);
    expect(result).toMatch(/\d+ (minute|hour|day)s? ago/);
  });
  
  it('should handle invalid dates gracefully', () => {
    const session = {
      id: 'test',
      timestamp: 'invalid-date',
    };
    
    const result = formatSessionDate(session);
    expect(result).toBe('Invalid date');
  });
});

export {
  // Export the fix functions for use in the application
  mapApiSessionToUi,
  formatSessionDate,
};