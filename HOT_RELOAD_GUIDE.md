# Native Binary Hot Reload Guide

## Quick Start

```bash
# 1. Install Air (one-time setup)
make install-air

# 2. Start hot reload development mode
make dev-native

# 3. Edit files and watch automatic rebuilds!
# - Edit Go files: bin/hyper rebuilds in ~2s
# - Edit UI files: UI + binary rebuilds in ~5s
```

## What Gets Watched

**Go Files:**
- `coordinator/cmd/**/*.go`
- `coordinator/internal/**/*.go`
- `coordinator/mcp-server/**/*.go`
- `coordinator/ai-service/**/*.go`

**UI Files:**
- `coordinator/ui/src/**/*` (React/TypeScript)

**Excluded:**
- `coordinator/tmp/`
- `coordinator/ui/node_modules/`
- `coordinator/ui/dist/`
- `*_test.go` files

## How It Works

### Smart UI Detection
The system uses a marker file (`.last-ui-build`) to track UI builds:

- **UI changed:** Rebuilds UI (~3.5s) → Rebuilds binary (~2s) = **~5s total**
- **UI unchanged:** Skips UI build → Rebuilds binary (~2s) = **~2s total**

### Build Process
1. Pre-build: `scripts/rebuild-ui.sh` checks if UI needs rebuild
2. Build: `scripts/air-build.sh` copies UI to embed/ and builds Go binary
3. Run: `bin/hyper --mode=http` starts with embedded UI

## Configuration

### Environment Variables
Hot reload uses `.env.hyper` for configuration:

```bash
MONGODB_URI="mongodb+srv://..."
QDRANT_URL="https://..."
EMBEDDING="ollama"
HTTP_PORT="7095"
```

### Air Configuration
Located at `.air.toml` (project root):

```toml
[build]
  pre_cmd = ["./scripts/rebuild-ui.sh"]  # Smart UI rebuild
  cmd = "./scripts/air-build.sh"          # Fast Go build
  bin = "./bin/hyper"
  args_bin = ["--mode=http"]
  delay = 1000                            # 1 second stability delay
```

## Performance Comparison

| Method | Time | Use Case |
|--------|------|----------|
| `make native` | 30-60s | Production builds |
| `make dev-native` (Go change) | ~2s | Development (Go only) |
| `make dev-native` (UI change) | ~5s | Development (UI + Go) |
| `make run-dev` (Docker) | ~3s | Coordinator service only |

## Troubleshooting

### Air not found
```bash
# Install Air
make install-air

# Or manually
go install github.com/air-verse/air@latest

# Verify installation
$(go env GOPATH)/bin/air -v
```

### Build fails
```bash
# Clean and rebuild
rm -rf coordinator/embed/ui
rm -rf coordinator/ui/dist
rm -f bin/hyper

# Manual build to see errors
./scripts/rebuild-ui.sh
./scripts/air-build.sh
```

### UI not updating
```bash
# Force UI rebuild
rm -f coordinator/ui/.last-ui-build
make dev-native
```

### Port already in use
```bash
# Kill existing hyper process
pkill -f bin/hyper

# Or change port in .env.hyper
echo "HTTP_PORT=7096" >> .env.hyper
```

## Advanced Usage

### Watch specific directories only
Edit `.air.toml` and modify `include_dir`:

```toml
include_dir = ["coordinator/internal", "coordinator/ai-service"]
```

### Adjust rebuild delay
Edit `.air.toml`:

```toml
delay = 2000  # 2 seconds instead of 1
```

### Run without UI rebuild
If working on Go only:

```bash
# Temporarily disable UI watch
mv .air.toml .air.toml.bak
# ... make changes ...
mv .air.toml.bak .air.toml
```

## Scripts Reference

### `scripts/rebuild-ui.sh`
Smart UI rebuild with timestamp detection:
- Checks if `coordinator/ui/src/**/*` changed
- Checks if `coordinator/ui/package.json` changed
- Only rebuilds if necessary
- Creates `.last-ui-build` marker

### `scripts/air-build.sh`
Fast Go binary build:
- Copies `coordinator/ui/dist` → `coordinator/embed/ui/`
- Builds with `-tags embed`
- Outputs to `bin/hyper` (15M)
- ~2 second build time

### `scripts/dev-native.sh`
Development mode wrapper:
- Sources `.env.hyper` environment
- Adds `GOPATH/bin` to PATH for Air
- Validates configuration
- Runs Air with project root config
- Handles Ctrl+C cleanup

## Integration with Existing Workflows

### Development Workflow
```bash
# Hot reload development (native binary)
make dev-native

# Docker development (services only)
make run-dev
```

### Production Workflow
```bash
# Build optimized native binary
make native

# Run production binary
./run-native.sh
```

### Testing Workflow
```bash
# Run tests (no rebuild)
make test

# Run specific tests
cd coordinator && go test ./internal/...
```

## What's Next?

After implementing changes with hot reload:

1. **Test:** Verify functionality at http://localhost:7095/ui
2. **Commit:** Commit your changes
3. **Build:** Run `make native` for production build
4. **Deploy:** Use production binary for deployment

## Tips

✅ **Do:**
- Use `make dev-native` for rapid iteration
- Keep terminal open to see rebuild logs
- Edit multiple files - Air batches changes (1s delay)
- Use browser DevTools for UI debugging

❌ **Don't:**
- Run `make native` during development (too slow)
- Edit files in `coordinator/embed/ui/` (gets overwritten)
- Commit `coordinator/ui/.last-ui-build` marker
- Run multiple `make dev-native` instances

## Support

If hot reload isn't working:

1. Check Air is installed: `$(go env GOPATH)/bin/air -v`
2. Check `.air.toml` exists at project root
3. Check `.env.hyper` has correct configuration
4. Check logs in `tmp/build-errors.log`
5. Try manual build: `./scripts/air-build.sh`

For issues, check:
- Air documentation: https://github.com/air-verse/air
- Project logs: `tmp/build-errors.log`
- Console output from `make dev-native`
