#!/bin/bash
# Hot-reload development script
# Runs Go backend with Air + Vite dev server with HMR simultaneously

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Ensure Go binaries are in PATH
export PATH=$HOME/go/bin:$GOPATH/bin:$PATH

# Track background processes
BACKEND_PID=""
FRONTEND_PID=""

# Cleanup function
cleanup() {
    echo ""
    echo -e "${YELLOW}🛑 Shutting down development servers...${NC}"

    # Kill frontend process
    if [ -n "$FRONTEND_PID" ]; then
        echo -e "${CYAN}Stopping Vite dev server (PID: $FRONTEND_PID)...${NC}"
        kill -TERM "$FRONTEND_PID" 2>/dev/null || true
        wait "$FRONTEND_PID" 2>/dev/null || true
    fi

    # Kill backend process and its children (Air spawns child processes)
    if [ -n "$BACKEND_PID" ]; then
        echo -e "${CYAN}Stopping Air/Go backend (PID: $BACKEND_PID)...${NC}"
        # Kill process group to ensure Air and its spawned processes are stopped
        kill -TERM -"$BACKEND_PID" 2>/dev/null || true
        wait "$BACKEND_PID" 2>/dev/null || true
    fi

    echo -e "${GREEN}✓ Development servers stopped${NC}"
    exit 0
}

# Set up signal handlers
trap cleanup SIGINT SIGTERM EXIT

# Prerequisites check
echo -e "${BLUE}🔍 Checking prerequisites...${NC}"

# Check Air installation
if ! command -v air &> /dev/null; then
    echo -e "${RED}✗ Error: Air not found${NC}"
    echo -e "${YELLOW}Install Air with: make install-air${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Air installed${NC}"

# Check Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}✗ Error: Node.js not found${NC}"
    echo -e "${YELLOW}Install Node.js from: https://nodejs.org/${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Node.js installed${NC}"

# Check node_modules
if [ ! -d "coordinator/ui/node_modules" ]; then
    echo -e "${YELLOW}⚠ node_modules not found. Installing...${NC}"
    cd coordinator/ui && npm install && cd ../..
fi
echo -e "${GREEN}✓ node_modules ready${NC}"

# Check .air.toml
if [ ! -f ".air.toml" ]; then
    echo -e "${RED}✗ Error: .air.toml not found at project root${NC}"
    exit 1
fi
echo -e "${GREEN}✓ .air.toml found${NC}"

# Check .env file
if [ -f ".env.hyper" ]; then
    echo -e "${GREEN}✓ .env.hyper found${NC}"
    set -a; source .env.hyper; set +a
elif [ -f ".env" ]; then
    echo -e "${YELLOW}⚠ Using .env (no .env.hyper found)${NC}"
    set -a; source .env; set +a
else
    echo -e "${YELLOW}⚠ No .env file found. Using system environment variables.${NC}"
fi

# Enable hot reload mode for UI (proxy to Vite instead of serving static files)
export HOT_RELOAD=true

echo ""
echo -e "${GREEN}🚀 Starting hot-reload development mode...${NC}"
echo ""

# Start Go backend with Air in background (using unified hyper binary)
echo -e "${BLUE}[Backend] Starting Air hot-reload for unified hyper binary...${NC}"
(
    # Run Air from project root (uses root .air.toml -> builds bin/hyper)
    # Prefix all backend output with [Backend]
    air 2>&1 | while IFS= read -r line; do
        echo -e "${CYAN}[Backend]${NC} $line"
    done
) &
BACKEND_PID=$!

# Wait for backend to start
echo -e "${YELLOW}⏳ Waiting for backend to initialize (5 seconds)...${NC}"
sleep 5

# Check if backend is still running
if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
    echo -e "${RED}✗ Backend failed to start. Check logs above.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Backend started (PID: $BACKEND_PID)${NC}"

# Start Vite dev server in background
echo ""
echo -e "${BLUE}[Frontend] Starting Vite dev server with HMR...${NC}"
(
    cd coordinator/ui
    # Prefix all frontend output with [Frontend]
    npm run dev 2>&1 | while IFS= read -r line; do
        echo -e "${GREEN}[Frontend]${NC} $line"
    done
) &
FRONTEND_PID=$!

# Wait for frontend to start
echo -e "${YELLOW}⏳ Waiting for Vite to initialize (3 seconds)...${NC}"
sleep 3

# Check if frontend is still running
if ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
    echo -e "${RED}✗ Frontend failed to start. Check logs above.${NC}"
    cleanup
    exit 1
fi
echo -e "${GREEN}✓ Frontend started (PID: $FRONTEND_PID)${NC}"

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Hot-reload development mode ready!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}Backend (Air):${NC}       http://localhost:7095"
echo -e "${BLUE}Frontend (Vite):${NC}     http://localhost:5173"
echo -e "${BLUE}Vite proxies API to:${NC} http://localhost:7095"
echo ""
echo -e "${YELLOW}Features:${NC}"
echo -e "  • Go backend: Air hot-reload on .go file changes"
echo -e "  • React UI: Vite HMR (instant updates on .tsx/.ts changes)"
echo -e "  • API proxy: Vite proxies /api/* to backend"
echo ""
echo -e "${CYAN}Press Ctrl+C to stop both servers${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Wait for both processes (they run indefinitely until Ctrl+C)
wait "$BACKEND_PID" "$FRONTEND_PID"
