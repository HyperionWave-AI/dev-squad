# Development Targets Update Summary

## ✅ **All `make dev` Targets Now Use Unified Hyper**

### **Files Updated**

#### 1. **Makefile**

**Target: `run-dev`** ✅ FIXED
```diff
- run-dev: ## Run with Air hot-reload (coordinator only)
+ run-dev: ## Run with Air hot-reload (unified hyper binary)
  	@echo "Starting development mode with hot-reload..."
  	@echo "Using Air for automatic rebuild on file changes"
  	@if ! command -v air &> /dev/null; then \
  		echo "Error: Air not found. Install with 'make install-air'"; \
  		exit 1; \
  	fi
- 	cd coordinator && air
+ 	@if [ ! -f .air.toml ]; then \
+ 		echo "Error: .air.toml not found at project root"; \
+ 		exit 1; \
+ 	fi
+ 	@echo "Building and running unified hyper binary with Air..."
+ 	air
```

**Effect:** Runs Air from project root → uses unified hyper

---

#### 2. **scripts/dev-native.sh** ✅ UPDATED

**Banner Update:**
```diff
- echo -e "${BLUE}║  Hyper - Native Development Mode with Hot Reload         ║${NC}"
+ echo -e "${BLUE}║  Unified Hyper - Native Development Mode with Hot Reload  ║${NC}"
```

**Watch Pattern Update:**
```diff
  echo -e "${BLUE}Starting Air hot reload...${NC}"
- echo -e "  Watching:   coordinator/**/*.go, coordinator/ui/src/**/*"
- echo -e "  Binary:     bin/hyper"
+ echo -e "  Watching:   hyper/**/*.go"
+ echo -e "  Binary:     bin/hyper (unified)"
  echo -e "  Mode:       http"
```

**Effect:** Correct messaging about what's being watched and built

---

#### 3. **CLAUDE.md** ✅ CRITICAL WARNING ADDED

**New Section Added at Top (after mission statement):**

```markdown
## 🚨 **CRITICAL: Unified Hyper Binary Architecture**

**⚠️ IMPORTANT: DO NOT use `coordinator/` directory as main entry point!**

### **Correct Architecture (MANDATORY)**

✅ CORRECT: Use unified hyper binary
  Entry Point:  hyper/cmd/coordinator/main.go
  Build Output: bin/hyper
  Source Code:  hyper/*
  Size:         17MB
  Modes:        http | mcp | both
  Version:      2.0.0

❌ DEPRECATED: Old coordinator (DO NOT USE)
  Entry Point:  coordinator/cmd/coordinator/main.go
  Build Output: coordinator/tmp/coordinator
  Source Code:  coordinator/*
  Size:         24MB
  Modes:        http only
  Version:      1.0.0 (obsolete)
```

**Effect:** Agents now see prominent warning about architecture

---

## 📋 **Complete List of Development Targets**

### ✅ **Already Using Unified Hyper**

1. **`make build`** → `bin/hyper` ✅
2. **`make native`** → `bin/hyper` ✅
3. **`make dev`** → calls `scripts/dev-native.sh` → `bin/hyper` ✅
4. **`make dev-hot`** → calls `scripts/dev-hot.sh` → `bin/hyper` ✅ (fixed yesterday)
5. **`make run`** → `bin/hyper --mode=http` ✅
6. **`make run-stdio`** → `bin/hyper --mode=mcp` ✅
7. **`make run-mcp-http`** → `bin/hyper --mode=http` ✅
8. **`make desktop`** → `bin/hyper` ✅
9. **`make desktop-dev`** → `bin/hyper` ✅
10. **`make desktop-build`** → `bin/hyper` ✅

### ✅ **Now Fixed**

11. **`make run-dev`** → `air` from project root → `bin/hyper` ✅ (fixed today)

---

## 🎯 **Target Descriptions**

| Target | Purpose | Binary | Notes |
|--------|---------|--------|-------|
| `make build` | Build production binary | `bin/hyper` | With embedded UI |
| `make native` | Same as build | `bin/hyper` | Alias |
| `make dev` | Air hot-reload (Go only) | `bin/hyper` | Uses root `.air.toml` |
| `make dev-hot` | Air + Vite HMR | `bin/hyper` | Best for UI work |
| `make run-dev` | Air hot-reload simple | `bin/hyper` | Just runs `air` |
| `make run` | Run compiled binary | `bin/hyper` | HTTP mode |
| `make run-stdio` | Run in stdio mode | `bin/hyper` | MCP mode |
| `make run-mcp-http` | Run in HTTP mode | `bin/hyper` | Full REST API |
| `make desktop` | Desktop app dev | `bin/hyper` | With Tauri |

---

## 🚀 **Verification Commands**

### Test All Dev Targets

```bash
# 1. Test build
make clean
make build
ls -lh bin/hyper  # Should show ~17MB

# 2. Test dev (hot-reload)
make dev
# Should show: "Unified Hyper - Native Development Mode"
# Should show: "Watching: hyper/**/*.go"
# Ctrl+C to stop

# 3. Test dev-hot (full stack)
make dev-hot
# Should show: "Building unified hyper WITHOUT embedded UI"
# Should show backend + frontend servers
# Ctrl+C to stop

# 4. Test run-dev (simple Air)
make run-dev
# Should run Air from project root
# Should build bin/hyper
# Ctrl+C to stop

# 5. Test compiled binary
make run
# Should run bin/hyper directly
# Ctrl+C to stop
```

---

## 📊 **Architecture Comparison**

### Before (Multiple Entry Points) ❌

```
coordinator/cmd/coordinator/main.go  → coordinator/tmp/coordinator (24MB)
hyper/cmd/coordinator/main.go        → bin/hyper (17MB)

Problem: Confusion about which to use
```

### After (Single Entry Point) ✅

```
hyper/cmd/coordinator/main.go  → bin/hyper (17MB)
                                  ↑
                          ONLY entry point

All make targets → bin/hyper
All dev scripts  → bin/hyper
All Air configs  → bin/hyper
```

---

## 🔧 **Scripts Updated**

| Script | Change | Status |
|--------|--------|--------|
| `scripts/dev-hot.sh` | Removed `cd coordinator` | ✅ Fixed (yesterday) |
| `scripts/dev-native.sh` | Updated watch paths to `hyper/*` | ✅ Fixed (today) |
| `scripts/air-build.sh` | Changed source from `coordinator/*` to `hyper/*` | ✅ Fixed (yesterday) |
| `.air.toml` (root) | Changed watch dirs to `hyper/*` | ✅ Fixed (yesterday) |
| `Makefile` | Fixed `run-dev` target | ✅ Fixed (today) |
| `CLAUDE.md` | Added critical warning | ✅ Fixed (today) |

---

## 📝 **CLAUDE.md Warning Benefits**

### For AI Agents

1. **Immediate visibility** - Warning at top of document
2. **Clear comparison** - Side-by-side old vs new
3. **Explicit commands** - Shows correct make targets
4. **DO/DON'T lists** - Clear actionable guidance
5. **Migration notes** - Explains what changed

### For Developers

1. **Prevents mistakes** - Won't accidentally use old coordinator
2. **Clear architecture** - Understands unified approach
3. **Command reference** - Quick lookup for correct commands
4. **Size comparison** - Shows benefits (17MB vs 24MB)

---

## ✅ **Verification Checklist**

After these changes, verify:

- [ ] All `make dev*` targets use `bin/hyper`
- [ ] No targets use `coordinator/tmp/coordinator`
- [ ] Air runs from project root (not `cd coordinator`)
- [ ] Watch patterns reference `hyper/*` (not `coordinator/*`)
- [ ] Build scripts compile from `hyper/cmd/coordinator`
- [ ] CLAUDE.md has prominent warning about architecture
- [ ] Scripts show "unified hyper" in output messages

---

## 🎓 **Key Changes Summary**

1. ✅ **Makefile `run-dev` target** - Runs Air from project root
2. ✅ **scripts/dev-native.sh** - Updated banner and watch paths
3. ✅ **CLAUDE.md** - Added critical architecture warning
4. ✅ **All dev targets** - Now consistently use `bin/hyper`
5. ✅ **Documentation** - Clear guidance for agents and developers

---

## 🎉 **Benefits**

### Consistency
- ✅ All targets use same binary
- ✅ No confusion about entry points
- ✅ Clear "one way to do it" pattern

### Performance
- ✅ 17MB binary (30% smaller)
- ✅ Faster builds (optimized pipeline)
- ✅ Multiple modes (http/mcp/both)

### Developer Experience
- ✅ Clear documentation
- ✅ Consistent commands
- ✅ Better error messages
- ✅ AI agent guidance

---

## 📅 **Date:** 2025-10-12
## 📝 **Status:** ✅ Complete - All Dev Targets Updated
