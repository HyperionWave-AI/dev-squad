# dev-hot Quick Reference

## 🚀 Quick Start

```bash
# 1. Install Air (one-time setup)
make install-air

# 2. Start hot-reload development
make dev-hot

# 3. Open browser
# Frontend: http://localhost:5173
# Backend:  http://localhost:7095

# 4. Stop servers
# Press Ctrl+C
```

## 📊 What You Get

| Feature | Benefit |
|---------|---------|
| **Vite HMR** | Instant UI updates (<200ms) |
| **Air Hot-Reload** | Backend rebuilds in ~2s |
| **Parallel Execution** | Both servers run simultaneously |
| **Clean Shutdown** | Ctrl+C stops both cleanly |
| **Colored Logs** | Backend (cyan) + Frontend (green) |

## 🔧 Development Workflow

### Edit React/TypeScript Files
```bash
# 1. Edit any .tsx or .ts file in coordinator/ui/src/
nano coordinator/ui/src/App.tsx

# 2. Save file
# 3. Browser updates INSTANTLY (no reload needed)
# 4. Check browser console for "HMR update" message
```

### Edit Go Files
```bash
# 1. Edit any .go file in coordinator/
nano coordinator/internal/server/http_server.go

# 2. Save file
# 3. Watch terminal for "[Backend] Building..." message
# 4. Backend rebuilds and restarts in ~2 seconds
# 5. API endpoints immediately available
```

## 🐛 Troubleshooting

### Port already in use
```bash
# Kill existing processes
lsof -ti:7095 | xargs kill -9  # Backend
lsof -ti:5173 | xargs kill -9  # Frontend
```

### Air not found
```bash
# Install Air
make install-air

# Verify installation
air -v
```

### Frontend not loading
```bash
# Reinstall node_modules
cd coordinator/ui
rm -rf node_modules package-lock.json
npm install
cd ../..

# Restart
make dev-hot
```

### Backend crashes
```bash
# Check logs
tail -f tmp/build-errors.log

# Verify environment
cat .env.hyper | grep HTTP_PORT
```

## 📋 When to Use What

| Command | Use Case | Startup | UI Updates |
|---------|----------|---------|------------|
| `make dev-hot` | **Active development** | 10s | Instant HMR |
| `make dev` | Pre-deployment testing | 30s | Full rebuild |
| `make run-dev` | Backend-only work | 5s | N/A |
| `make native` | Production build | 45s | N/A |

## 🎯 Common Tasks

### Check if servers are running
```bash
# Backend health check
curl http://localhost:7095/health

# Frontend (should return HTML)
curl http://localhost:5173
```

### View logs only
```bash
# Backend logs only
make dev-hot 2>&1 | grep "\[Backend\]"

# Frontend logs only
make dev-hot 2>&1 | grep "\[Frontend\]"
```

### Restart after error
```bash
# 1. Stop with Ctrl+C
# 2. Fix the error
# 3. Restart
make dev-hot
```

## ⚙️ Configuration

### Backend Port (Default: 7095)
Edit `.env.hyper`:
```bash
HTTP_PORT=7095
```

### Frontend Port (Default: 5173)
Edit `coordinator/ui/vite.config.ts`:
```typescript
server: {
  port: 5173,  // Change if needed
  // ...
}
```

### API Proxy
Vite automatically proxies these routes to backend:
- `/api/mcp` → `http://localhost:7095/api/mcp`
- `/api/knowledge` → `http://localhost:7095/api/knowledge`
- `/api/tasks` → `http://localhost:7095/api/tasks`
- `/api/agent-tasks` → `http://localhost:7095/api/agent-tasks`

## 💡 Pro Tips

1. **Keep terminal visible** - Watch colored logs for immediate feedback
2. **Use browser DevTools** - Network tab shows HMR updates
3. **Save often** - Air debounces rebuilds (1s delay)
4. **Check both ports** - Backend API + Frontend dev server
5. **Clean shutdown** - Always Ctrl+C (never kill -9)

## 📚 Full Documentation

- **Implementation Details:** `DEV_HOT_IMPLEMENTATION_SUMMARY.md`
- **Testing Guide:** `DEV_HOT_TESTING.md`
- **This Guide:** `DEV_HOT_QUICK_REFERENCE.md`

## 🆘 Still Having Issues?

1. Read error messages (they're helpful!)
2. Check `DEV_HOT_TESTING.md` section "Common Issues"
3. Review `DEV_HOT_IMPLEMENTATION_SUMMARY.md` section "Support"
4. Ask team for help

## ✅ Success Indicators

When everything works correctly, you'll see:
```
✅ Hot-reload development mode ready!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Backend (Air):       http://localhost:7095
Frontend (Vite):     http://localhost:5173
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Features:
  • Go backend: Air hot-reload on .go file changes
  • React UI: Vite HMR (instant updates)
  • API proxy: Vite proxies /api/* to backend
```

**Happy coding! 🎉**
