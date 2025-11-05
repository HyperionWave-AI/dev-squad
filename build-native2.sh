#!/bin/bash

# Cross-Platform Build Script for UI2
# Creates a single self-contained binary with embedded UI2
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
UI_DIR="$PROJECT_ROOT/ui2"
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
    OUTPUT_BINARY="$OUTPUT_DIR/hyper2.exe"
else
    OUTPUT_BINARY="$OUTPUT_DIR/hyper2"
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
echo -e "${BLUE}║  Hyper2 - Native Build (UI2)                              ║${NC}"
echo -e "${BLUE}║  Single Binary with Embedded UI2                          ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Platform:${NC} $PLATFORM"
echo -e "${GREEN}Go version:${NC} $(go version | awk '{print $3}')"
echo -e "${GREEN}Node version:${NC} $(node --version)"
echo ""

# Step 1: Build UI2
echo -e "${BLUE}[1/4] Building UI2...${NC}"
cd "$UI_DIR"

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing UI2 dependencies...${NC}"
    npm install
fi

echo -e "${YELLOW}Building production UI2 bundle (skipping TypeScript check)...${NC}"
# Use build:prod to skip TypeScript checking
npm run build:prod

# Verify UI2 build
if [ ! -f "dist/index.html" ]; then
    echo -e "${RED}ERROR: UI2 build failed - dist/index.html not found${NC}"
    exit 1
fi

echo -e "${GREEN}✓ UI2 built successfully${NC}"
echo -e "  Assets: $(find dist/assets -type f | wc -l | tr -d ' ') files"
echo -e "  Size: $(du -sh dist | awk '{print $1}')"
echo ""

# Step 2: Prepare UI2 for embedding
echo -e "${BLUE}[2/4] Preparing UI2 for embedding...${NC}"
cd "$HYPER_DIR"

# Remove old symlink and create directory structure for embedding
rm -rf embed/ui2
mkdir -p embed/ui2

# Copy UI2 dist to embed directory (Go embed doesn't support symlinks)
echo -e "${YELLOW}Copying UI2 to embed directory...${NC}"
cp -r "$UI_DIR/dist" embed/ui2/

echo -e "${GREEN}✓ UI2 prepared for embedding${NC}"
echo ""

# Step 3: Build Go binary with embedded UI2
echo -e "${BLUE}[3/4] Building Go binary with embedded UI2...${NC}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Build with embedded UI2
echo -e "${YELLOW}Compiling for $PLATFORM...${NC}"

# Set build variables
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
VERSION="2.0.0-native2"

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

if [ "$GOOS" != "windows" ]; then
    echo -e "1. Run the binary:"
    echo -e "   ${YELLOW}$OUTPUT_BINARY --mode=http${NC}"
    echo ""
    echo -e "2. Access the UI2:"
    echo -e "   ${YELLOW}http://localhost:7095/ui${NC}"
    echo ""
fi
echo -e "${GREEN}The binary is fully self-contained with embedded UI2!${NC}"
echo ""
