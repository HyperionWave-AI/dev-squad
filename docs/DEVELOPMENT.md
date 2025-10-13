# Hyperion Development Guide

## Quick Start

### Prerequisites
```bash
# Install dependencies
make install        # Install Go + Node.js dependencies
make install-air    # Install Air hot-reload tool
```

### Development Modes

#### 1. Backend Development Only (Recommended for Go work)
```bash
make dev
```
**What it does:**
- Hot reloads Go backend on file changes (Air)
- **No UI compilation** - saves ~3 seconds per reload
- API available at `http://localhost:7095/api/v1`
- Health check at `http://localhost:7095/api/v1/health`

**Use when:**
- Working on Go backend code
- Testing API endpoints
- Don't need UI changes

#### 2. Full-Stack Development (Go + React)
```bash
make dev-hot
```
**What it does:**
- Hot reloads Go backend (Air)
- Vite dev server with HMR for React UI
- Instant UI updates on `.tsx`/`.ts` changes
- Backend: `http://localhost:7095`
- Frontend: `http://localhost:5173`
- Vite proxies API requests to backend

**Use when:**
- Working on both backend and frontend
- Testing UI + API integration
- Need instant React component updates

#### 3. Production Build
```bash
make build    # or: make native
```
**What it does:**
- Compiles React UI to static files
- Embeds UI into Go binary
- Creates single `bin/hyper` binary
- Ready for deployment

## Configuration

### Environment Variables
Create `.env.hyper` in project root:
```bash
# MongoDB
MONGODB_URI=mongodb://localhost:27017
DATABASE_NAME=hyper_coordinator_db

# Qdrant
QDRANT_URL=http://localhost:6333
QDRANT_API_KEY=your-api-key

# Embedding Mode (ollama, openai, local)
EMBEDDING=ollama

# Server
HTTP_PORT=7095
```

### Air Configuration
Located in `.air.toml` at project root:
- Watches: `hyper/cmd`, `hyper/internal`
- Builds: `cd hyper && go build -o ../tmp/hyper ./cmd/coordinator`
- Binary: `tmp/hyper --mode=http`
- No UI compilation during development

## Architecture

### Directory Structure
```
dev-squad/
├── hyper/              # Go backend source
│   ├── cmd/
│   │   └── coordinator/
│   │       └── main.go  # Entry point
│   ├── internal/       # Internal packages
│   ├── go.mod          # Go module definition
│   └── go.sum
├── ui/                 # React frontend source
│   ├── src/
│   ├── package.json
│   └── vite.config.ts
├── bin/                # Compiled binaries
│   └── hyper           # Production binary
├── tmp/                # Development binaries
│   └── hyper           # Dev binary (Air)
├── .air.toml           # Air configuration
├── .env.hyper          # Environment config
└── Makefile            # Build commands
```

### Build Process

#### Development (`make dev`)
1. Air watches Go files in `hyper/`
2. On change: `cd hyper && go build -o ../tmp/hyper ./cmd/coordinator`
3. Runs: `tmp/hyper --mode=http`
4. No UI compilation (use Vite dev server for UI work)

#### Production (`make build`)
1. UI: `cd ui && npm run build` → `ui/dist/`
2. Embed: Copy `ui/dist/` → `hyper/embed/ui/`
3. Go: `cd hyper && go build -o ../bin/hyper ./cmd/coordinator`
4. Result: Single `bin/hyper` with embedded UI

## Troubleshooting

### Air not found
```bash
make install-air
# or manually:
go install github.com/air-verse/air@latest
```

### Go module errors
```bash
cd hyper && go mod tidy
cd hyper && go mod download
```

### UI build errors
```bash
cd ui && npm install
cd ui && npm run build
```

### Port conflicts
Edit `.env.hyper`:
```bash
HTTP_PORT=7096  # Change to different port
```

### Air not detecting changes
- Check `.air.toml` include_dir paths
- Verify excluded directories don't contain your files
- Restart Air: Ctrl+C, then `make dev`

## Tips

### Fast Go Development
- Use `make dev` for backend-only work (no UI compilation)
- Edit code, Air rebuilds in ~2 seconds
- Test API with `curl` or Postman

### Fast UI Development
- Use `make dev-hot` for frontend work
- Edit React components, see changes instantly (HMR)
- Vite proxies API calls to Go backend

### Testing API
```bash
# Health check
curl http://localhost:7095/api/v1/health

# With authentication
export JWT_TOKEN="your-jwt-token"
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:7095/api/v1/tasks
```

### Debugging
- Go logs: Check Air output
- UI logs: Check browser console (F12)
- Backend errors: See `tmp/build-errors.log`

## Common Workflows

### Backend Feature Development
```bash
# 1. Start backend hot reload
make dev

# 2. Edit Go files in hyper/internal/
# 3. Air rebuilds automatically
# 4. Test API endpoints
curl http://localhost:7095/api/v1/...

# 5. Commit when ready
git add .
git commit -m "feat: add new endpoint"
```

### UI Component Development
```bash
# 1. Start full-stack hot reload
make dev-hot

# 2. Open frontend in browser
open http://localhost:5173

# 3. Edit React components in ui/src/
# 4. See changes instantly (HMR)

# 5. Commit when ready
git add .
git commit -m "feat: add new component"
```

### Full Feature (Backend + Frontend)
```bash
# 1. Start full-stack mode
make dev-hot

# 2. Backend work:
#    - Edit Go files in hyper/
#    - Air rebuilds backend

# 3. Frontend work:
#    - Edit React files in ui/
#    - Vite hot reloads instantly

# 4. Test integration:
#    - Frontend calls API endpoints
#    - Vite proxies to Go backend

# 5. Build for production
make build

# 6. Commit
git add .
git commit -m "feat: complete feature"
```

## Performance

### Development Mode
- Go rebuild: ~2 seconds (Air)
- UI HMR: <100ms (Vite)
- No UI compilation during Go development (saves ~3s per change)

### Production Build
- UI build: ~10 seconds
- Go build: ~5 seconds
- Total: ~15 seconds
- Result: Single 17MB binary with embedded UI

## See Also
- [Air Documentation](https://github.com/air-verse/air)
- [Vite Documentation](https://vitejs.dev/)
- [Go Documentation](https://golang.org/doc/)
