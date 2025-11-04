# UI Visual QA Report – Knowledge Features (BLOCKED)

- **Date**: 2025-11-03
- **Environment**: Local Dev (http://localhost:4097)
- **Viewport Tested**: Desktop 1440x900
- **Overall Status**: ❌ BLOCKED - Critical Bug Found

---

## ❌ CRITICAL BUG BLOCKING ALL TESTS

### API Path Mismatch in knowledgeApi.ts

**File**: `/Users/maxmednikov/MaxSpace/hyper/ui/src/services/knowledgeApi.ts`
**Line**: 64

**Current (WRONG)**:
```typescript
async listCollections(): Promise<{ collections: KnowledgeCollection[] }> {
  return fetchWithAuth(`${API_BASE}/api/knowledge/collections`);
}
```

**Should Be**:
```typescript
async listCollections(): Promise<{ collections: KnowledgeCollection[] }> {
  return fetchWithAuth(`${API_BASE}/api/v1/knowledge/collections`);
}
```

**Impact**:
- CollectionBrowser component cannot load collections
- 404 errors repeated 4 times in console
- Red "Not found" error message displayed at bottom of page
- Create Collection button not visible (because CollectionBrowser doesn't render)
- All collection-related features non-functional

**Evidence**:
- Screenshot: `knowledge-page-404-error.png`
- Console errors: `Failed to load resource: the server responded with a status of 404 (Not Found) @ http://localhost:4097/api/knowledge/collections`
- Backend verified working correctly (returns 12 collections with metadata at `/api/v1/knowledge/collections`)

---

## 🧪 Test Results

### TODO 1: Navigate and Verify CollectionBrowser Loads
- **Status**: ❌ FAILED
- **Expected**: 12 collections displayed in grid layout
- **Actual**: CollectionBrowser not rendered, only KnowledgeSearch component visible
- **Root Cause**: API 404 error preventing component render

### TODO 2-10: Cannot Test
- **Status**: ⏸️ BLOCKED
- All remaining tests require CollectionBrowser to be functional
- Cannot test Create Collection button (not visible)
- Cannot test modal, metadata, filtering, or any collection features

---

## ✅ What Works (Partial Success)

### KnowledgeSearch Component
- Semantic/Browse toggle renders correctly
- Toggle button group displays both options
- Semantic mode is selected by default (pressed state visible)
- Form fields render: Collection dropdown, Search Query input, Result Limit slider
- Search and Clear buttons visible
- Keyboard shortcut hints displayed

### Page Layout
- Navigation works (Knowledge tab highlighted correctly)
- Header renders properly
- Main content area structured correctly
- Theme toggle and refresh buttons functional

---

## 🔧 Required Fix

**Assignee**: ui-dev agent
**Task**: Fix API path in knowledgeApi.ts line 64

**Change Required**:
```diff
- return fetchWithAuth(`${API_BASE}/api/knowledge/collections`);
+ return fetchWithAuth(`${API_BASE}/api/v1/knowledge/collections`);
```

**Note**: Check if any other API calls in knowledgeApi.ts are missing `/v1/`:
- Line 42: `searchKnowledge` uses `/api/knowledge/search` (may also need `/v1/`)
- Line 46: `queryKnowledge` correctly uses `/api/v1/knowledge/query` ✓
- Line 53: `createKnowledge` uses `/api/knowledge` (may also need `/v1/`)
- Line 68: `createCollection` correctly uses `/api/v1/knowledge/collections` ✓

---

## 📋 Testing Blocked Until Fixed

Cannot proceed with:
1. ✅ CollectionBrowser load verification
2. ⏸️ Create Collection button visibility test
3. ⏸️ Create Collection modal functionality
4. ⏸️ Collection submission flow
5. ⏸️ Metadata display verification
6. ⏸️ Category filtering tabs
7. ⏸️ Semantic search toggle behavior
8. ⏸️ Semantic search API call
9. ⏸️ Browse search API call
10. ⏸️ Comprehensive test suite creation

---

## 💡 Recommendations

1. **Immediate**: Fix API path in knowledgeApi.ts line 64
2. **Follow-up**: Audit all API endpoints in knowledgeApi.ts for `/v1/` consistency
3. **Testing**: Re-run ui-tester after fix to complete all 10 TODOs
4. **Prevention**: Add API path constants or OpenAPI spec to prevent path mismatches

---

## 🎯 Next Steps

1. Coordinator assigns ui-dev to fix knowledgeApi.ts
2. ui-dev updates line 64 (and audits other endpoints)
3. ui-dev verifies fix in browser console (no 404 errors)
4. ui-tester re-runs full test suite
5. Create comprehensive Playwright test file

---

**Test Session ID**: 6f29ac57-c1ed-47cd-ba46-889505d8b494
**Tester**: ui-tester agent
**Status**: BLOCKED awaiting ui-dev fix
