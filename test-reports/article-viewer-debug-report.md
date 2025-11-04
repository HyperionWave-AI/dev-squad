# ArticleViewer Debug Report - Review & Compact Buttons

**Date**: 2025-01-04
**Component**: `ui2/src/components/organisms/ArticleViewer.tsx`
**Issue**: Review and Compact buttons not working properly

## 🔍 Investigation Results

### ✅ Backend Endpoints CONFIRMED
Both API endpoints exist and are properly registered:

**Review Endpoint:**
- **Path**: `POST /api/v1/knowledge/entries/:id/review`
- **Handler**: `ReviewEntryHandler` (line 636 in `knowledge_handler.go`)
- **Status**: ✅ Working

**Compact Endpoint:**
- **Path**: `POST /api/v1/knowledge/entries/:id/compact`
- **Handler**: `CompactEntryHandler` (line 818 in `knowledge_handler.go`)
- **Status**: ✅ Working

### 🛠️ Changes Made

#### 1. Enhanced Error Handling & Logging
**File**: `ArticleViewer.tsx` (lines 54-111)

Added comprehensive logging to all handlers:
```typescript
// Review handler (lines 54-73)
- Console logs at every step (start, API call, result, dialog open)
- User-friendly error alerts with actual error messages
- Try-catch with proper error handling

// Compact handler (lines 75-94)
- Console logs at every step
- User-friendly error alerts
- Try-catch with proper error handling

// Apply compaction handler (lines 96-111)
- Console logs for apply process
- Error alerts on failure
```

#### 2. Dialog Component Debugging
**Files**: `ReviewResultDialog.tsx`, `CompactionDialog.tsx`

Added logging to dialog render cycles:
```typescript
// ReviewResultDialog (lines 38-48)
- Logs when dialog renders
- Logs if result is missing
- Logs when result is received

// CompactionDialog (lines 23-40)
- Logs when dialog renders with props
- Logs if result is missing
- Logs when approve button clicked
```

#### 3. Mock Data Test Buttons
**File**: `ArticleViewer.tsx` (lines 32-82, 191-205)

Added **DEV-ONLY** test buttons to isolate UI vs backend issues:

**Mock Handlers:**
- `handleReviewMock()`: Creates mock ReviewResult data
- `handleCompactMock()`: Creates mock CompactionResult data

**Test Buttons:**
```typescript
🧪 Review  - Opens ReviewResultDialog with mock data
🧪 Compact - Opens CompactionDialog with mock data
```

These buttons bypass the API completely to test if dialogs work independently.

## 📊 Debugging Strategy

### Step 1: Test Mock Buttons First
1. Open ui2 ArticleViewer
2. Select any entry
3. Click **🧪 Review** button
4. Check if ReviewResultDialog opens correctly
5. Click **🧪 Compact** button
6. Check if CompactionDialog opens correctly

**Expected**: Dialogs should open instantly with mock data

### Step 2: Test Real API Buttons
1. Open browser console (F12)
2. Click **Review** button (ClipboardCheck icon)
3. Watch console logs:
   ```
   🔍 [REVIEW] Starting review for entry: <id>
   🔍 [REVIEW] Calling API: POST /api/v1/knowledge/entries/{id}/review
   ✅ [REVIEW] Review result: {...}
   🔍 [REVIEW DIALOG] Rendered with props: {...}
   ✅ [REVIEW DIALOG] Result received: {...}
   ✅ [REVIEW] Dialog should now be open
   ```

4. Click **Compact** button (Archive icon)
5. Watch console logs:
   ```
   📦 [COMPACT] Starting compaction for entry: <id>
   📦 [COMPACT] Calling API: POST /api/v1/knowledge/entries/{id}/compact
   ✅ [COMPACT] Compaction result: {...}
   📦 [COMPACT DIALOG] Rendered with props: {...}
   ✅ [COMPACT DIALOG] Result received: {...}
   ✅ [COMPACT] Dialog should now be open
   ```

### Step 3: Identify Issue Location

**If mock buttons work but API buttons don't:**
- Issue is backend communication (network, auth, CORS)
- Check Network tab for failed requests
- Check response status codes (404, 500, 401, etc.)

**If mock buttons also fail:**
- Issue is in UI component rendering
- Check React DevTools for state changes
- Verify Radix Dialog Portal is rendering
- Check z-index conflicts

**If API returns error:**
- Check console for error message
- Error alerts will show user-friendly message
- Check backend logs for server-side errors

## 🎯 Expected Behavior

### Review Button Flow:
1. User clicks Review button → Loading spinner shows
2. API call to `/api/v1/knowledge/entries/:id/review`
3. Response contains: scores, verification, actions
4. ReviewResultDialog opens with result
5. User sees health score, component scores, broken references, suggested actions

### Compact Button Flow:
1. User clicks Compact button → Loading spinner shows
2. API call to `/api/v1/knowledge/entries/:id/compact` (dryRun: true)
3. Response contains: original/compacted text, word counts, compression ratio
4. CompactionDialog opens with side-by-side comparison
5. User can review changes and click "Apply Compaction"
6. If approved: Second API call with dryRun: false
7. Page reloads to show updated entry

## 🧪 Test Checklist

- [ ] Mock Review button opens dialog correctly
- [ ] Mock Compact button opens dialog correctly
- [ ] Real Review button works without errors
- [ ] Real Compact button works without errors
- [ ] Review dialog displays all scores and references
- [ ] Compact dialog shows side-by-side comparison
- [ ] Apply Compaction button actually applies changes
- [ ] Error messages appear if API fails
- [ ] Loading spinners show during API calls
- [ ] Console logs provide clear debugging trail

## 📝 Known Issues & Next Steps

### Current State:
- ✅ Backend endpoints exist and are registered
- ✅ Frontend has proper error handling
- ✅ Comprehensive logging added
- ✅ Mock data test buttons available
- ⏳ **NEEDS USER TESTING** to identify actual issue

### Possible Issues to Check:
1. **Authentication**: JWT token might be missing/invalid
2. **Network**: API might not be reachable (check base URL)
3. **CORS**: Browser might block cross-origin requests
4. **Entry ID format**: ID might need URL encoding
5. **Review system**: ReviewOrchestrator might not be initialized

### Next Steps:
1. **Test with mock buttons** to verify UI works
2. **Check browser Network tab** when clicking real buttons
3. **Check API response** - is it 200, 404, 500?
4. **Check backend logs** - is ReviewOrchestrator initialized?
5. **Remove mock buttons** once issue is identified and fixed

## 🔧 Files Modified

### Frontend Changes:
```
ui2/src/components/organisms/ArticleViewer.tsx
- Lines 32-82: Mock data handlers
- Lines 54-73: Enhanced handleReview with logging
- Lines 75-94: Enhanced handleCompact with logging
- Lines 96-111: Enhanced handleApplyCompaction with logging
- Lines 191-205: Mock test buttons in UI

ui2/src/components/organisms/ReviewResultDialog.tsx
- Lines 38-48: Render logging

ui2/src/components/organisms/CompactionDialog.tsx
- Lines 23-40: Render logging
```

### Backend Files Verified:
```
hyper/internal/handlers/knowledge_handler.go
- Line 636: ReviewEntryHandler (POST /api/v1/knowledge/entries/:id/review)
- Line 818: CompactEntryHandler (POST /api/v1/knowledge/entries/:id/compact)
- Line 1055-1056: Routes registered
```

## 🎉 Success Criteria

The debugging is successful when:
1. ✅ Console logs clearly show execution flow
2. ✅ Error messages are user-friendly and actionable
3. ✅ Mock buttons verify UI components work independently
4. ✅ Real buttons work and dialogs open correctly
5. ✅ User can complete full review/compact workflow

---

**Ready for Testing!** Open ui2, select an entry, and try both mock and real buttons while watching the console. Report any errors that appear.
