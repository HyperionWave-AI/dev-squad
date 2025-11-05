# UI Test Report: Drag and Drop Not Working - Root Cause Analysis

**Date**: 2025-11-04
**Environment**: Local Development (http://localhost:4589)
**Tester**: UI-Tester Agent
**Overall Status**: ❌ **CRITICAL BUG - WRONG UI BEING SERVED**

---

## Executive Summary

**Drag and drop does NOT work because the backend is serving the WRONG UI.**

- ✅ The NEW Material UI with drag-and-drop (`ui2`) exists and is correctly implemented
- ✅ @hello-pangea/dnd library is installed (v18.0.1)
- ✅ KanbanBoard component correctly uses DragDropContext and Draggable
- ❌ **Backend serves OLD UI (`ui/dist`) instead of NEW UI (`ui2`)**
- ❌ User sees Tailwind CSS UI without drag-and-drop functionality

---

## Root Cause Analysis

### 🔴 Critical Issue: Backend Routing Configuration

**File**: `/Users/maxmednikov/MaxSpace/hyper/hyper/embed/ui.go`

```go
//go:embed all:ui/dist
var UI embed.FS

func GetUIFileSystem() (http.FileSystem, error) {
    // The embedded FS has structure: ui/dist/index.html, ui/dist/assets/...
    // Strip the "ui/dist" prefix to serve files from root
    stripped, err := fs.Sub(UI, "ui/dist")
    if err != nil {
        return nil, err
    }
    return http.FS(stripped), nil
}
```

**Problem**: The backend is hardcoded to serve `ui/dist` (OLD UI with Tailwind) instead of `ui2` (NEW UI with Material UI + drag-and-drop).

### Evidence

#### 1. Two UI Directories Exist
```bash
/Users/maxmednikov/MaxSpace/hyper/ui/     # OLD UI (Tailwind CSS)
/Users/maxmednikov/MaxSpace/hyper/ui2/    # NEW UI (Material UI + drag-and-drop)
```

#### 2. KanbanBoard Location
```bash
# NEW UI with drag-and-drop
/Users/maxmednikov/MaxSpace/hyper/ui2/src/pages/KanbanBoard.tsx

# OLD UI has NO KanbanBoard
/Users/maxmednikov/MaxSpace/hyper/ui/src/components/KanbanBoard.tsx (Material UI version - not used)
```

#### 3. What's Actually Being Served

**URL**: `http://localhost:4589/ui/tasks`

**Rendered HTML**:
```html
<div class="flex h-screen w-full overflow-hidden">
  <div class="w-80 h-screen overflow-y-auto z-50 bg-[#0e1e3e] border-r border-[#0a1628]">
    <!-- OLD Tailwind UI -->
  </div>
</div>
```

**Expected (Material UI)**:
```html
<div class="MuiBox-root">
  <div data-rbd-droppable-context-id="...">
    <!-- Material UI + drag-and-drop -->
  </div>
</div>
```

#### 4. Missing Drag-and-Drop Attributes

**Found**: 0 elements with `data-rbd-draggable-id`
**Found**: 0 elements with `data-rbd-droppable-id`
**Found**: 0 elements with `data-rbd-droppable-context-id`

**Expected**: DragDropContext, Droppable, and Draggable components should render these attributes.

#### 5. Vite Dev Server IS Running ui2

```bash
$ ps -p 74267 -o command=
node /Users/maxmednikov/MaxSpace/hyper/ui2/node_modules/.bin/vite

$ lsof -i :4589
node 74267 ... TCP localhost:4589 (LISTEN)
```

The Vite dev server for `ui2` is running, but the backend routes traffic to `ui/dist` instead of proxying to the Vite dev server.

---

## Testing Results

### ✅ What Works

1. **Port 4589 is accessible** - Backend HTTP server is running
2. **No JavaScript errors** - Console is clean (Vite connection messages only)
3. **Tasks page loads** - Shows task list (OLD UI version)
4. **API endpoints working** - Network requests to `/api/v1/` succeed

### ❌ What Doesn't Work

1. **Drag-and-drop not functional** - @hello-pangea/dnd not initialized
2. **Wrong UI served** - Tailwind CSS UI instead of Material UI
3. **KanbanBoard not loaded** - Material UI KanbanBoard component not rendered
4. **No drag handles visible** - Task cards are not draggable

---

## Fix Recommendations

### Option 1: Change Backend to Serve ui2 (Recommended)

**File**: `hyper/embed/ui.go` (line 16)

**Change**:
```go
// OLD
//go:embed all:ui/dist

// NEW
//go:embed all:ui2/dist
```

**Also update**:
```go
// Line 25
stripped, err := fs.Sub(UI, "ui2/dist")

// Line 35
_, err := UI.ReadFile("ui2/dist/index.html")
```

### Option 2: Proxy to Vite Dev Server in Development

**File**: `hyper/internal/server/http_server.go`

Add Vite dev server proxy for development mode:

```go
// In development, proxy UI requests to Vite dev server
if os.Getenv("DEV_MODE") == "true" {
    proxy := httputil.NewSingleHostReverseProxy(
        &url.URL{Scheme: "http", Host: "localhost:5173"},
    )
    r.PathPrefix("/ui/").Handler(proxy)
}
```

### Option 3: Rename ui2 to ui (Clean Approach)

1. Backup old ui: `mv ui ui_old`
2. Rename ui2: `mv ui2 ui`
3. Rebuild: `cd ui && npm run build`
4. Backend will automatically serve the new UI

---

## Implementation Notes

### Why This Happened

1. **Two parallel UIs**: Development created a new Material UI version (`ui2`) while keeping the old Tailwind version (`ui`)
2. **Backend not updated**: The embed configuration still points to the old UI
3. **Port confusion**: User accessed port 4588 (nothing running) vs 4589 (serves old UI)

### Correct URL (After Fix)

- ✅ `http://localhost:4589/ui/tasks` - Should serve Material UI with drag-and-drop
- ❌ `http://localhost:4588/ui/tasks` - Port 4588 has nothing running

---

## Screenshots

### What User Currently Sees
![tasks-page-initial-load.png](/Users/maxmednikov/MaxSpace/hyper/.playwright-mcp/tasks-page-initial-load.png)

**Status**: OLD Tailwind UI with task list (no drag-and-drop)

---

## Verification Steps (After Fix)

1. ✅ Navigate to `http://localhost:4589/ui/tasks`
2. ✅ Verify Material UI components render (MuiCard, MuiBox)
3. ✅ Check console: `document.querySelectorAll('[data-rbd-draggable-id]').length > 0`
4. ✅ Drag a task card from "Pending" to "In Progress"
5. ✅ Verify API call to `PUT /api/v1/tasks/:id/status`
6. ✅ Confirm task moves to new column
7. ✅ Check for visual feedback during drag (card rotation, shadow)

---

## Related Files

### Backend
- `/Users/maxmednikov/MaxSpace/hyper/hyper/embed/ui.go` - **NEEDS FIX**
- `/Users/maxmednikov/MaxSpace/hyper/hyper/internal/server/http_server.go` - HTTP routing

### Frontend (NEW - ui2)
- `/Users/maxmednikov/MaxSpace/hyper/ui2/src/pages/KanbanBoard.tsx` - Drag-and-drop implementation
- `/Users/maxmednikov/MaxSpace/hyper/ui2/src/components/TaskCard.tsx` - Draggable task cards
- `/Users/maxmednikov/MaxSpace/hyper/ui2/package.json` - Dependencies (@hello-pangea/dnd)

### Frontend (OLD - ui)
- `/Users/maxmednikov/MaxSpace/hyper/ui/dist/` - Currently being served
- `/Users/maxmednikov/MaxSpace/hyper/ui/src/` - Tailwind CSS implementation

---

## Recommendations

### Immediate Actions

1. ✅ **Update backend embed path** to `ui2/dist`
2. ✅ **Rebuild ui2**: `cd ui2 && npm run build`
3. ✅ **Restart backend**: Reload Go binary with new embedded UI
4. ✅ **Test drag-and-drop** at `http://localhost:4589/ui/tasks`

### Long-term Actions

1. 📋 **Remove old UI**: Delete `ui` directory after confirming ui2 works
2. 📋 **Rename ui2 to ui**: Clean up directory structure
3. 📋 **Update documentation**: Note the UI migration
4. 📋 **Add dev mode proxy**: Support Vite HMR in development

---

## Conclusion

**The drag-and-drop functionality is correctly implemented but not accessible because the backend serves the wrong UI directory.**

**Severity**: 🔴 **CRITICAL**
**Impact**: User cannot access new Material UI features including drag-and-drop
**Fix Complexity**: 🟢 **LOW** (3-line change in embed/ui.go)
**Time to Fix**: ⏱️ **5 minutes** (change + rebuild + restart)

---

**Test Completion**: 2025-11-04 23:35 UTC
**Report Generated By**: UI-Tester Agent (Playwright MCP)
