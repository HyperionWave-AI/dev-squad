#!/bin/bash

# Cross-Platform Build Script
# Creates a single self-contained binary with embedded UI
# Supports: macOS (darwin), Windows, Linux

set -e  # Exit on error

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Parse command-line arguments
TARGET_PLATFORM=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --platform)
            TARGET_PLATFORM="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [--platform PLATFORM]"
            echo ""
            echo "Platforms:"
            echo "  darwin-arm64    macOS Apple Silicon (M1/M2/M3)"
            echo "  darwin-amd64    macOS Intel"
            echo "  windows-amd64   Windows 64-bit"
            echo "  linux-amd64     Linux 64-bit"
            echo "  linux-arm64     Linux ARM64"
            echo ""
            echo "If --platform not specified, builds for current platform"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Configuration
PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
HYPER_DIR="$PROJECT_ROOT/hyper"
UI_DIR="$PROJECT_ROOT/.archive/coordinator/ui"
CMD_DIR="$HYPER_DIR/cmd/coordinator"
OUTPUT_DIR="$PROJECT_ROOT/bin"

# Platform detection
if [ -z "$TARGET_PLATFORM" ]; then
    # Build for current platform
    GOOS=$(go env GOOS)
    GOARCH=$(go env GOARCH)
    TARGET_PLATFORM="$GOOS-$GOARCH"
else
    # Parse target platform
    GOOS="${TARGET_PLATFORM%-*}"
    GOARCH="${TARGET_PLATFORM#*-}"
fi

# Set output binary name based on platform
if [ "$GOOS" = "windows" ]; then
    OUTPUT_BINARY="$OUTPUT_DIR/hyper.exe"
else
    OUTPUT_BINARY="$OUTPUT_DIR/hyper"
fi

# Platform display name
case "$TARGET_PLATFORM" in
    darwin-arm64)
        PLATFORM="macOS Apple Silicon (M1/M2/M3)"
        ;;
    darwin-amd64)
        PLATFORM="macOS Intel"
        ;;
    windows-amd64)
        PLATFORM="Windows 64-bit"
        ;;
    linux-amd64)
        PLATFORM="Linux 64-bit"
        ;;
    linux-arm64)
        PLATFORM="Linux ARM64"
        ;;
    *)
        PLATFORM="$GOOS ($GOARCH)"
        ;;
esac

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Hyper - Native Build                                     ║${NC}"
echo -e "${BLUE}║  Single Binary with Embedded UI                           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Platform:${NC} $PLATFORM"
echo -e "${GREEN}Go version:${NC} $(go version | awk '{print $3}')"
echo -e "${GREEN}Node version:${NC} $(node --version)"
echo ""

# Step 1: Build UI
echo -e "${BLUE}[1/4] Building UI...${NC}"
cd "$UI_DIR"

# Check if UI is already built
if [ -f "dist/index.html" ]; then
    echo -e "${GREEN}✓ UI already built (using existing dist)${NC}"
    echo -e "  Assets: $(find dist/assets -type f | wc -l | tr -d ' ') files"
    echo -e "  Size: $(du -sh dist | awk '{print $1}')"
else
    # Check if node_modules exists
    if [ ! -d "node_modules" ]; then
        echo -e "${YELLOW}Installing UI dependencies...${NC}"
        npm install
    fi

    echo -e "${YELLOW}Building production UI bundle...${NC}"
    npm run build

    # Verify UI build
    if [ ! -f "dist/index.html" ]; then
        echo -e "${RED}ERROR: UI build failed - dist/index.html not found${NC}"
        exit 1
    fi

    echo -e "${GREEN}✓ UI built successfully${NC}"
    echo -e "  Assets: $(find dist/assets -type f | wc -l | tr -d ' ') files"
    echo -e "  Size: $(du -sh dist | awk '{print $1}')"
fi
echo ""

# Step 2: Prepare UI for embedding
echo -e "${BLUE}[2/4] Preparing UI for embedding...${NC}"
cd "$HYPER_DIR"

# Remove old symlink and create directory structure for embedding
rm -rf embed/ui
mkdir -p embed/ui

# Copy UI dist to embed directory (Go embed doesn't support symlinks)
echo -e "${YELLOW}Copying UI to embed directory...${NC}"
cp -r "$UI_DIR/dist" embed/ui/

echo -e "${GREEN}✓ UI prepared for embedding${NC}"
echo ""

# Step 3: Build Go binary with embedded UI
echo -e "${BLUE}[3/4] Building Go binary with embedded UI...${NC}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build with embedded UI
echo -e "${YELLOW}Compiling for $PLATFORM...${NC}"

# Set build variables
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION="2.0.0-native"

# Go build with tags and optimizations (cross-platform)
# CGO is ENABLED for llama.cpp GPU acceleration
cd "$CMD_DIR"

# Platform-specific CGO flags for GPU acceleration
if [ "$GOOS" = "darwin" ]; then
    # macOS: Enable Metal GPU acceleration
    echo -e "${YELLOW}Enabling Metal GPU acceleration...${NC}"
    export CGO_ENABLED=1
    export CGO_CXXFLAGS="-std=c++17 -DGGML_USE_METAL -DGGML_METAL_NDEBUG"
    export CGO_LDFLAGS="-framework Foundation -framework Metal -framework MetalKit -framework MetalPerformanceShaders"
elif [ "$GOOS" = "windows" ]; then
    # Windows: Build with CPU only (CUDA/Vulkan requires Windows SDK)
    echo -e "${YELLOW}Building for Windows (CPU)...${NC}"
    echo -e "${YELLOW}For GPU support on Windows, build on Windows with CUDA/Vulkan SDK${NC}"
    export CGO_ENABLED=1
elif [ "$GOOS" = "linux" ]; then
    # Linux: Build with CPU only (CUDA/Vulkan requires Linux SDK)
    echo -e "${YELLOW}Building for Linux (CPU)...${NC}"
    echo -e "${YELLOW}For GPU support on Linux, build on Linux with CUDA/Vulkan SDK${NC}"
    export CGO_ENABLED=1
fi

GOOS=$GOOS GOARCH=$GOARCH go build \
    -tags "embed" \
    -ldflags="-s -w -X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GitCommit=$GIT_COMMIT" \
    -o "$OUTPUT_BINARY" \
    .

if [ ! -f "$OUTPUT_BINARY" ]; then
    echo -e "${RED}ERROR: Go build failed${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Binary built successfully${NC}"
echo -e "  Location: $OUTPUT_BINARY"
echo -e "  Size: $(du -sh "$OUTPUT_BINARY" | awk '{print $1}')"
echo ""

# Step 4: Verify binary
echo -e "${BLUE}[4/5] Verifying binary...${NC}"

# Check if binary is executable (skip for Windows when building on macOS/Linux)
if [ "$GOOS" != "windows" ]; then
    if [ ! -x "$OUTPUT_BINARY" ]; then
        chmod +x "$OUTPUT_BINARY"
    fi
fi

# Check binary architecture
if command -v file &> /dev/null; then
    file "$OUTPUT_BINARY"
fi

# Verify binary exists and has content
echo -e "${YELLOW}Verifying binary...${NC}"
if [ -f "$OUTPUT_BINARY" ] && [ -s "$OUTPUT_BINARY" ]; then
    echo -e "${GREEN}✓ Binary created successfully${NC}"
    if [ "$GOOS" != "windows" ] && [ -x "$OUTPUT_BINARY" ]; then
        echo -e "${GREEN}✓ Binary is executable${NC}"
    fi
else
    echo -e "${RED}ERROR: Binary is missing or empty${NC}"
    exit 1
fi

echo ""

# Step 5: Summary
echo -e "${BLUE}[5/5] Build Summary${NC}"
echo -e "${GREEN}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  Build completed successfully! ✓                          ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Binary location: ${YELLOW}$OUTPUT_BINARY${NC}"
echo -e "Binary size:     $(du -sh "$OUTPUT_BINARY" | awk '{print $1}')"
echo -e "Platform:        $PLATFORM"
echo -e "Version:         $VERSION"
echo -e "Git commit:      $GIT_COMMIT"
echo -e "Build time:      $BUILD_TIME"
echo ""
echo -e "${BLUE}Next steps:${NC}"
echo ""

if [ "$GOOS" = "windows" ]; then
    echo -e "1. Transfer binary to Windows machine:"
    echo -e "   ${YELLOW}scp $OUTPUT_BINARY user@windows-host:C:/path/to/hyper.exe${NC}"
    echo ""
    echo -e "2. Set environment variables on Windows (PowerShell):"
    echo -e "   ${YELLOW}\$env:MONGODB_URI = 'your-mongodb-uri'${NC}"
    echo -e "   ${YELLOW}\$env:MONGODB_DATABASE = 'coordinator_db'${NC}"
    echo -e "   ${YELLOW}\$env:QDRANT_URL = 'your-qdrant-url'${NC}"
    echo -e "   ${YELLOW}\$env:QDRANT_API_KEY = 'your-qdrant-key'${NC}"
    echo -e "   ${YELLOW}\$env:EMBEDDING = 'voyage'${NC}"
    echo -e "   ${YELLOW}\$env:VOYAGE_API_KEY = 'your-voyage-key'${NC}"
    echo -e "   ${YELLOW}\$env:HTTP_PORT = '7095'${NC}"
    echo ""
    echo -e "3. Run the binary:"
    echo -e "   ${YELLOW}.\\hyper.exe --mode=http${NC}"
    echo ""
    echo -e "4. Access the UI:"
    echo -e "   ${YELLOW}http://localhost:7095/ui${NC}"
else
    echo -e "1. Set environment variables (create .env.hyper or export):"
    echo -e "   ${YELLOW}export MONGODB_URI='your-mongodb-uri'${NC}"
    echo -e "   ${YELLOW}export MONGODB_DATABASE='coordinator_db'${NC}"
    echo -e "   ${YELLOW}export QDRANT_URL='your-qdrant-url'${NC}"
    echo -e "   ${YELLOW}export QDRANT_API_KEY='your-qdrant-key'${NC}"
    echo -e "   ${YELLOW}export EMBEDDING='voyage'${NC}"
    echo -e "   ${YELLOW}export VOYAGE_API_KEY='your-voyage-key'${NC}"
    echo -e "   ${YELLOW}export HTTP_PORT='7095'${NC}"
    echo ""
    echo -e "2. Run the binary:"
    echo -e "   ${YELLOW}$OUTPUT_BINARY --mode=http${NC}"
    echo ""
    echo -e "   Or use the convenience script:"
    echo -e "   ${YELLOW}./run-native.sh${NC}"
    echo ""
    echo -e "3. Access the UI:"
    echo -e "   ${YELLOW}http://localhost:7095/ui${NC}"
fi
echo ""
echo -e "${GREEN}The binary is fully self-contained with embedded UI!${NC}"
echo -e "${GREEN}No Docker, no file mappings, no external dependencies.${NC}"
echo ""

# Generate install command for non-Windows platforms
if [ "$GOOS" != "windows" ]; then
    echo -e "${BLUE}Optional: Install binary to system path:${NC}"
    if [ "$GOOS" = "darwin" ]; then
        echo -e "   ${YELLOW}sudo mv $OUTPUT_BINARY /usr/local/bin/${NC}"
    else
        echo -e "   ${YELLOW}sudo mv $OUTPUT_BINARY /usr/bin/${NC}"
    fi
    echo ""
    echo -e "${BLUE}Configure for Claude Code (MCP stdio mode):${NC}"
    echo -e "   ${YELLOW}make configure-native${NC}"
    echo -e "   Or manually:"
    echo -e "   ${YELLOW}claude mcp add hyper \"$OUTPUT_BINARY\" --args \"--mode=mcp\" --scope user${NC}"
    echo ""
fi
