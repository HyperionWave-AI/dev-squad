#!/bin/bash
# Smart UI Rebuild Script
# Only rebuilds UI if source files have changed since last build
# This saves ~3 seconds when only Go files changed

set -e  # Exit on error

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
UI_DIR="$PROJECT_ROOT/coordinator/ui"
MARKER_FILE="$UI_DIR/.last-ui-build"
DIST_DIR="$UI_DIR/dist"

# Check if UI source changed since last build
ui_needs_rebuild() {
    # If marker doesn't exist, we need to build
    if [ ! -f "$MARKER_FILE" ]; then
        return 0  # true - need rebuild
    fi

    # If dist doesn't exist, we need to build
    if [ ! -d "$DIST_DIR" ]; then
        return 0  # true - need rebuild
    fi

    # Check if any UI source file is newer than marker
    if [ -n "$(find "$UI_DIR/src" -type f -newer "$MARKER_FILE" 2>/dev/null)" ]; then
        return 0  # true - need rebuild
    fi

    # Check if package.json changed (dependencies)
    if [ "$UI_DIR/package.json" -nt "$MARKER_FILE" ]; then
        return 0  # true - need rebuild
    fi

    return 1  # false - no rebuild needed
}

# Main logic
if ui_needs_rebuild; then
    echo "📦 UI source changed - rebuilding..."
    cd "$UI_DIR"

    # Install dependencies if node_modules missing
    if [ ! -d "node_modules" ]; then
        echo "Installing UI dependencies..."
        npm install
    fi

    # Build UI
    npm run build

    # Update marker
    touch "$MARKER_FILE"

    echo "✓ UI rebuilt successfully"
else
    echo "⚡ UI unchanged - skipping rebuild (saves ~3s)"
fi

exit 0  # Always exit 0 to not block Air
