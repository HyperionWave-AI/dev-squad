# dev-hot Testing Guide

## Overview
This document provides testing instructions for the new `make dev-hot` target that enables hot-reload development with:
- Go backend hot-reload via Air
- React UI hot-reload via Vite HMR
- Parallel execution with clean shutdown

## Prerequisites Check

### 1. Install Air (if not installed)
```bash
make install-air
# Verify installation
air -v
```

### 2. Install Node.js dependencies
```bash
cd coordinator/ui
npm install
cd ../..
```

### 3. Verify configuration files
```bash
# Check .air.toml at project root
ls -la .air.toml

# Check coordinator Air config
ls -la coordinator/.air.toml

# Check environment file
ls -la .env.hyper

# Check Vite config
ls -la coordinator/ui/vite.config.ts
```

## Testing Procedure

### Test 1: Basic Startup
**Purpose:** Verify both servers start successfully

```bash
# Start dev-hot mode
make dev-hot

# Expected output:
# ✓ Air installed
# ✓ Node.js installed
# ✓ node_modules ready
# ✓ .air.toml found
# ✓ .env.hyper found (or .env)
# [Backend] Starting Air hot-reload for Go...
# [Frontend] Starting Vite dev server with HMR...
# ✅ Hot-reload development mode ready!
# Backend (Air):       http://localhost:7095
# Frontend (Vite):     http://localhost:5173
```

**Verify:**
- [ ] No errors in console output
- [ ] Backend shows "[Backend]" prefix in logs
- [ ] Frontend shows "[Frontend]" prefix in logs
- [ ] Both processes start within 10 seconds

### Test 2: Backend Accessibility
**Purpose:** Verify Go backend is running and serving API

```bash
# In a new terminal (while dev-hot is running)
curl http://localhost:7095/health

# Expected response:
# {"status":"ok"}
```

**Verify:**
- [ ] Backend responds to HTTP requests
- [ ] Health endpoint returns 200 OK

### Test 3: Frontend Accessibility
**Purpose:** Verify Vite dev server is running

```bash
# Open in browser
open http://localhost:5173

# Or use curl to check if Vite is serving
curl -I http://localhost:5173
```

**Verify:**
- [ ] Browser opens the UI
- [ ] Vite dev server responds (200 OK)
- [ ] UI loads without errors
- [ ] Browser console shows no errors

### Test 4: Vite HMR (Hot Module Replacement)
**Purpose:** Verify instant UI updates without full reload

```bash
# While dev-hot is running and browser is open at localhost:5173
# Edit a React component
nano coordinator/ui/src/App.tsx

# Make a visible change (e.g., change a heading text)
# Save the file

# Check browser console
# Expected: "HMR update" or "[vite] hot updated"
```

**Verify:**
- [ ] UI updates INSTANTLY without full page reload
- [ ] Browser console shows HMR messages
- [ ] No errors in terminal or browser console
- [ ] Change is visible in browser immediately

### Test 5: Backend Hot-Reload (Air)
**Purpose:** Verify Go backend reloads on code changes

```bash
# While dev-hot is running
# Edit a Go file (e.g., add a log statement)
nano coordinator/internal/server/http_server.go

# Add a log statement in a handler
# Save the file

# Check terminal output
# Expected:
# [Backend] Building...
# [Backend] Running...
```

**Verify:**
- [ ] Air detects file change within 1 second
- [ ] Backend rebuilds automatically
- [ ] New binary starts without errors
- [ ] API still responds after reload

### Test 6: API Proxy
**Purpose:** Verify Vite proxies API calls to backend

```bash
# While dev-hot is running and browser is open
# Open browser DevTools → Network tab
# Make an API call from UI (e.g., load tasks page)

# Check Network tab
# API calls should go to localhost:5173 but proxied to localhost:7095
```

**Verify:**
- [ ] API calls succeed (200 OK)
- [ ] No CORS errors in console
- [ ] Requests proxied correctly (/api/mcp, /api/tasks, etc.)

### Test 7: Clean Shutdown
**Purpose:** Verify both processes stop cleanly

```bash
# While dev-hot is running, press Ctrl+C

# Expected output:
# 🛑 Shutting down development servers...
# Stopping Vite dev server (PID: xxxx)...
# Stopping Air/Go backend (PID: xxxx)...
# ✓ Development servers stopped
```

**Verify:**
- [ ] Both processes stop within 2 seconds
- [ ] No zombie processes left behind
- [ ] Ports 7095 and 5173 are freed

```bash
# Verify no processes on ports
lsof -i :7095
lsof -i :5173
# Should return nothing
```

### Test 8: Error Handling - Missing Air
**Purpose:** Verify graceful error when Air not installed

```bash
# Temporarily rename Air binary
sudo mv $(which air) $(which air).bak

# Run dev-hot
make dev-hot

# Expected output:
# ✗ Error: Air not found
# Install Air with: make install-air

# Restore Air
sudo mv $(which air).bak $(which air)
```

**Verify:**
- [ ] Clear error message
- [ ] Suggests installation command
- [ ] Exits gracefully (no hanging processes)

### Test 9: Error Handling - Missing node_modules
**Purpose:** Verify automatic npm install

```bash
# Temporarily rename node_modules
mv coordinator/ui/node_modules coordinator/ui/node_modules.bak

# Run dev-hot
make dev-hot

# Expected output:
# ⚠ node_modules not found. Installing...
# (npm install output)
# ✓ node_modules ready

# Restore node_modules
rm -rf coordinator/ui/node_modules
mv coordinator/ui/node_modules.bak coordinator/ui/node_modules
```

**Verify:**
- [ ] Detects missing node_modules
- [ ] Runs npm install automatically
- [ ] Continues to start servers after install

### Test 10: Parallel Execution
**Purpose:** Verify both processes run in parallel

```bash
# Start dev-hot
make dev-hot

# In another terminal, check running processes
ps aux | grep -E "(air|vite)" | grep -v grep

# Should see:
# - air process running
# - node/vite process running
```

**Verify:**
- [ ] Both processes run simultaneously
- [ ] Backend responds while frontend is running
- [ ] No blocking/waiting between processes

### Test 11: Environment Variables
**Purpose:** Verify environment variables are loaded

```bash
# Check .env.hyper exists
cat .env.hyper | grep HTTP_PORT

# Start dev-hot
make dev-hot

# Verify backend uses correct port
curl http://localhost:7095/health
```

**Verify:**
- [ ] HTTP_PORT=7095 is used by backend
- [ ] Vite proxies to correct backend URL
- [ ] Environment loaded from .env.hyper (or .env)

### Test 12: Makefile Help
**Purpose:** Verify help text is correct

```bash
make help | grep dev-hot

# Expected output:
# dev-hot              Start development with UI hot-reload (Vite dev server + Go Air)
```

**Verify:**
- [ ] Help text appears
- [ ] Description is clear and accurate

## Performance Expectations

- **Startup time:** Both servers ready within 10 seconds
- **Go rebuild:** <2 seconds on file change
- **UI HMR:** <200ms instant update
- **Shutdown:** Clean exit within 2 seconds
- **Memory usage:** Air ~50MB, Vite ~200MB (acceptable for dev)

## Common Issues & Solutions

### Issue: Port already in use
```bash
# Kill processes on ports
lsof -ti:7095 | xargs kill -9
lsof -ti:5173 | xargs kill -9
```

### Issue: Air not found after install
```bash
# Add GOPATH/bin to PATH
export PATH=$PATH:$(go env GOPATH)/bin
# Or restart terminal
```

### Issue: Vite fails to start
```bash
# Reinstall dependencies
cd coordinator/ui
rm -rf node_modules package-lock.json
npm install
```

### Issue: Backend crashes on startup
```bash
# Check logs
tail -f tmp/build-errors.log

# Verify .env.hyper configuration
cat .env.hyper
```

## Success Criteria

All tests must pass:
- ✅ Both servers start successfully
- ✅ Backend API responds on port 7095
- ✅ Frontend serves on port 5173
- ✅ Vite HMR updates UI instantly
- ✅ Air reloads backend on Go changes
- ✅ API proxy works (no CORS errors)
- ✅ Clean shutdown on Ctrl+C
- ✅ Prerequisites checked and installed
- ✅ Error messages are clear and helpful

## Comparison: dev vs dev-hot

| Feature | `make dev` | `make dev-hot` |
|---------|-----------|----------------|
| Backend | Air hot-reload | Air hot-reload |
| UI | Static build | Vite dev server |
| UI updates | Full rebuild | Instant HMR |
| Startup time | ~30s (UI build) | ~10s (no UI build) |
| Development speed | Slow (rebuild on each change) | Fast (instant updates) |
| Production-like | Yes (static assets) | No (dev server) |
| Use case | Pre-deployment testing | Active development |

## Recommended Workflow

1. **Active development:** Use `make dev-hot` for fastest iteration
2. **Pre-deployment testing:** Use `make dev` to test with static assets
3. **Production build:** Use `make native` for final binary

## Next Steps After Testing

If all tests pass:
1. ✅ Mark task as complete
2. ✅ Update project README with dev-hot usage
3. ✅ Add to onboarding documentation
4. ✅ Share with team
