#!/bin/bash

# Hyperion Clean Install Script
# This script performs a complete clean installation of Hyperion

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_header() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_step() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC}  $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC}  $1"
}

# Check if running from project root
if [ ! -f "Makefile" ] || [ ! -d "hyper" ]; then
    print_error "Please run this script from the Hyperion project root directory"
    exit 1
fi

print_header "Hyperion Clean Install"

echo "This script will:"
echo "  1. Clean all build artifacts"
echo "  2. Remove node_modules"
echo "  3. Clean Go module cache (optional)"
echo "  4. Install dependencies"
echo "  5. Build the hyper binary"
echo ""
read -p "Continue? (yes/no): " -r
echo ""

if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    print_info "Installation cancelled"
    exit 0
fi

# Step 1: Clean build artifacts
print_header "Step 1: Cleaning Build Artifacts"

if [ -d "bin" ]; then
    print_step "Removing bin/ directory..."
    rm -rf bin/
fi

if [ -d "ui/dist" ]; then
    print_step "Removing ui/dist/..."
    rm -rf ui/dist/
fi

if [ -d "ui2/dist" ]; then
    print_step "Removing ui2/dist/..."
    rm -rf ui2/dist/
fi

if [ -d "hyper/embed/ui" ]; then
    print_step "Removing hyper/embed/ui/..."
    rm -rf hyper/embed/ui/
fi

if [ -d "hyper/embed/ui2" ]; then
    print_step "Removing hyper/embed/ui2/..."
    rm -rf hyper/embed/ui2/
fi

if [ -d "hyper/bin" ]; then
    print_step "Removing hyper/bin/..."
    rm -rf hyper/bin/
fi

print_step "Build artifacts cleaned"

# Step 2: Remove node_modules
print_header "Step 2: Cleaning Node Modules"

read -p "Remove node_modules? (yes/no): " -r
echo ""

if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    if [ -d "ui/node_modules" ]; then
        print_step "Removing ui/node_modules/..."
        rm -rf ui/node_modules/
    fi

    if [ -d "ui2/node_modules" ]; then
        print_step "Removing ui2/node_modules/..."
        rm -rf ui2/node_modules/
    fi

    print_step "Node modules removed"
else
    print_info "Skipping node_modules removal"
fi

# Step 3: Clean Go cache
print_header "Step 3: Cleaning Go Cache"

read -p "Clean Go module cache? (yes/no): " -r
echo ""

if [[ $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    print_step "Cleaning Go module cache..."
    cd hyper && go clean -modcache && cd ..
    print_step "Go cache cleaned"
else
    print_info "Skipping Go cache cleanup"
fi

# Step 4: Install dependencies
print_header "Step 4: Installing Dependencies"

print_info "Installing Go dependencies..."
cd hyper && go mod download && cd ..
print_step "Go dependencies installed"

print_info "Installing Node.js dependencies for UI..."
cd ui && npm install && cd ..
print_step "UI dependencies installed"

# Step 5: Build hyper binary
print_header "Step 5: Building Hyper Binary"

print_info "Building Go binary (without UI embedding)..."
cd hyper && go build -tags dev -o ../bin/hyper ./cmd/coordinator && cd ..

if [ -f "bin/hyper" ]; then
    print_step "Hyper binary built successfully!"

    # Get binary size
    SIZE=$(ls -lh bin/hyper | awk '{print $5}')
    print_info "Binary size: $SIZE"
    print_info "Location: $(pwd)/bin/hyper"
else
    print_error "Failed to build hyper binary"
    exit 1
fi

# Step 6: Verify installation
print_header "Step 6: Verifying Installation"

print_info "Testing hyper binary..."
if ./bin/hyper --help > /dev/null 2>&1 || [ $? -eq 0 ]; then
    print_step "Binary runs successfully"
else
    print_warning "Binary may have issues (this is ok if --help isn't implemented)"
fi

# Final summary
print_header "Installation Complete!"

echo "✅ Hyperion is ready to use!"
echo ""
echo "📦 What was installed:"
echo "   • Go binary: $(pwd)/bin/hyper"
echo "   • Go dependencies: hyper/go.mod"
echo "   • Node dependencies: ui/node_modules"
echo ""
echo "🚀 Next steps:"
echo ""
echo "   1. Initialize a new project:"
echo "      mkdir my-project && cd my-project"
echo "      $(pwd)/bin/hyper init"
echo ""
echo "   2. Or with a provider:"
echo "      $(pwd)/bin/hyper init -provider openai -token sk-..."
echo ""
echo "   3. Add to PATH (optional):"
echo "      export PATH=\"$(pwd)/bin:\$PATH\""
echo "      echo 'export PATH=\"$(pwd)/bin:\$PATH\"' >> ~/.bashrc"
echo ""
echo "   4. Configure Claude Code (optional):"
echo "      make configure-native"
echo ""
echo "📖 Documentation:"
echo "   • Quick start: QUICK_REFERENCE.md"
echo "   • Provider setup: HYPER_INIT_WITH_PROVIDER.md"
echo "   • Full guide: MAKEFILE_AND_DOCKER_GUIDE.md"
echo ""
print_header "Thank you for using Hyperion! 🎉"
