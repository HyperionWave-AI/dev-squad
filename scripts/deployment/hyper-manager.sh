#!/bin/bash
################################################################################
# Hyperion Unified Binary Process Manager
#
# Manages the unified hyper binary to prevent duplicate processes
# and ensure clean startup/shutdown
#
# Usage:
#   ./hyper-manager.sh start [http|mcp|both]  # Start unified binary (default: both)
#   ./hyper-manager.sh stop                    # Stop all coordinator processes
#   ./hyper-manager.sh restart [mode]          # Restart with specified mode
#   ./hyper-manager.sh status                  # Check running processes
#   ./hyper-manager.sh clean                   # Kill all and clean
################################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HYPER_BINARY="$SCRIPT_DIR/bin/hyper"
PID_FILE="$SCRIPT_DIR/.hyper.pid"

# Functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check if hyper binary exists
check_binary() {
    if [[ ! -f "$HYPER_BINARY" ]]; then
        log_error "Unified hyper binary not found: $HYPER_BINARY"
        log_info "Build it with: cd hyper && make build"
        exit 1
    fi

    if [[ ! -x "$HYPER_BINARY" ]]; then
        log_error "Hyper binary is not executable"
        log_info "Fix with: chmod +x $HYPER_BINARY"
        exit 1
    fi
}

# Find all coordinator-related processes
find_coordinator_processes() {
    # Find processes by pattern
    ps aux | grep -E "(coordinator|hyper).*-mode|bin/hyper|tmp/coordinator" | grep -v grep || true
}

# Get PIDs of running coordinators
get_coordinator_pids() {
    # Match both old coordinator and new hyper binary
    pgrep -f "(coordinator.*-mode|bin/hyper)" || true
}

# Check status
check_status() {
    log_info "Checking for running coordinator processes..."
    echo ""

    local processes=$(find_coordinator_processes)
    if [[ -z "$processes" ]]; then
        log_success "No coordinator processes running"
        return 1
    fi

    echo "$processes"
    echo ""

    # Check for old coordinator
    if echo "$processes" | grep -q "coordinator/tmp/coordinator"; then
        log_warning "OLD coordinator binary detected (should use unified hyper)"
    fi

    # Check for new hyper
    if echo "$processes" | grep -q "bin/hyper"; then
        log_success "Unified hyper binary running"
    fi

    # Check PID file
    if [[ -f "$PID_FILE" ]]; then
        local saved_pid=$(cat "$PID_FILE")
        if ps -p "$saved_pid" > /dev/null 2>&1; then
            log_info "PID file: $saved_pid (process exists)"
        else
            log_warning "Stale PID file: $saved_pid (process not found)"
            rm "$PID_FILE"
        fi
    fi

    return 0
}

# Stop all coordinator processes
stop_processes() {
    log_info "Stopping all coordinator processes..."

    local pids=$(get_coordinator_pids)
    if [[ -z "$pids" ]]; then
        log_success "No processes to stop"
        return 0
    fi

    echo "Found PIDs: $pids"

    # Try graceful shutdown first (SIGTERM)
    for pid in $pids; do
        log_info "Sending SIGTERM to PID $pid..."
        kill "$pid" 2>/dev/null || log_warning "Failed to kill $pid (may already be dead)"
    done

    # Wait up to 5 seconds for graceful shutdown
    log_info "Waiting for graceful shutdown..."
    sleep 2

    # Check if any are still running
    local remaining=$(get_coordinator_pids)
    if [[ -n "$remaining" ]]; then
        log_warning "Some processes still running, forcing shutdown (SIGKILL)..."
        for pid in $remaining; do
            kill -9 "$pid" 2>/dev/null || true
        done
        sleep 1
    fi

    # Final check
    if [[ -z "$(get_coordinator_pids)" ]]; then
        log_success "All processes stopped"
    else
        log_error "Failed to stop some processes:"
        find_coordinator_processes
        exit 1
    fi

    # Clean up PID file
    if [[ -f "$PID_FILE" ]]; then
        rm "$PID_FILE"
    fi
}

# Start unified hyper binary
start_hyper() {
    local mode="${1:-both}"

    log_info "Starting unified hyper binary in '$mode' mode..."

    # Check for running processes first
    if [[ -n "$(get_coordinator_pids)" ]]; then
        log_error "Coordinator processes already running:"
        find_coordinator_processes
        echo ""
        log_info "Stop them first with: $0 stop"
        exit 1
    fi

    # Check binary exists
    check_binary

    # Verify .env.hyper exists
    if [[ ! -f "$SCRIPT_DIR/.env.hyper" ]]; then
        log_warning ".env.hyper not found in $SCRIPT_DIR"
        log_info "The binary will use system environment variables"
    fi

    # Start the binary in background
    log_info "Starting: $HYPER_BINARY -mode $mode"
    nohup "$HYPER_BINARY" -mode "$mode" > "$SCRIPT_DIR/hyper.log" 2>&1 &
    local pid=$!

    # Save PID
    echo "$pid" > "$PID_FILE"
    log_success "Started with PID: $pid"

    # Wait a moment and check if it's still running
    sleep 2
    if ! ps -p "$pid" > /dev/null 2>&1; then
        log_error "Process died immediately after start"
        log_info "Check logs: tail -f $SCRIPT_DIR/hyper.log"
        rm "$PID_FILE" 2>/dev/null || true
        exit 1
    fi

    log_success "Unified hyper binary is running (PID: $pid)"
    echo ""
    log_info "Logs: tail -f $SCRIPT_DIR/hyper.log"
    log_info "Status: $0 status"
    log_info "Stop: $0 stop"
}

# Clean everything
clean_all() {
    log_warning "Cleaning all coordinator processes and artifacts..."

    stop_processes

    # Clean log files
    if [[ -f "$SCRIPT_DIR/hyper.log" ]]; then
        rm "$SCRIPT_DIR/hyper.log"
        log_info "Removed log file"
    fi

    # Clean PID file
    if [[ -f "$PID_FILE" ]]; then
        rm "$PID_FILE"
        log_info "Removed PID file"
    fi

    log_success "Clean complete"
}

# Print usage
print_usage() {
    cat << EOF
${GREEN}Hyperion Unified Binary Process Manager${NC}

${BLUE}Usage:${NC}
    $0 start [http|mcp|both]    Start unified hyper binary (default: both)
    $0 stop                      Stop all coordinator processes
    $0 restart [mode]            Restart with specified mode
    $0 status                    Check running processes
    $0 clean                     Kill all and clean artifacts

${BLUE}Modes:${NC}
    http    HTTP-only mode (REST API + UI on port 7095)
    mcp     MCP-only mode (stdio protocol for Claude Code)
    both    Both HTTP and MCP (default - recommended)

${BLUE}Examples:${NC}
    # Start in dual mode (HTTP + MCP)
    $0 start

    # Start in HTTP-only mode
    $0 start http

    # Check what's running
    $0 status

    # Restart in different mode
    $0 restart mcp

    # Stop everything
    $0 stop

    # Clean shutdown and remove artifacts
    $0 clean

${BLUE}Files:${NC}
    Binary:     $HYPER_BINARY
    Config:     $SCRIPT_DIR/.env.hyper
    Logs:       $SCRIPT_DIR/hyper.log
    PID:        $PID_FILE

${YELLOW}Notes:${NC}
    - This script prevents duplicate processes
    - Old coordinator binaries will be stopped
    - Use 'both' mode for full functionality
    - Logs are written to hyper.log
    - PID is saved for process management

EOF
}

################################################################################
# Main
################################################################################

case "${1:-}" in
    start)
        start_hyper "${2:-both}"
        ;;
    stop)
        stop_processes
        ;;
    restart)
        stop_processes
        sleep 1
        start_hyper "${2:-both}"
        ;;
    status)
        check_status
        exit $?
        ;;
    clean)
        clean_all
        ;;
    --help|-h|help)
        print_usage
        exit 0
        ;;
    "")
        log_error "No command specified"
        echo ""
        print_usage
        exit 1
        ;;
    *)
        log_error "Unknown command: $1"
        echo ""
        print_usage
        exit 1
        ;;
esac
