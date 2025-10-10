#!/bin/bash

################################################################################
# Hyper Command Installer
#
# Installs the "hyper" command globally so you can run:
#   hyper                    # Interactive wizard
#   hyper --folder . --port 8080
#   hyper --stop --name my-project
################################################################################

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Hyper Command Installer${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COORDINATOR_SCRIPT="$SCRIPT_DIR/start-coordinator.sh"

# Verify the coordinator script exists
if [[ ! -f "$COORDINATOR_SCRIPT" ]]; then
    echo -e "${RED}Error: start-coordinator.sh not found at: $COORDINATOR_SCRIPT${NC}"
    exit 1
fi

echo -e "${BLUE}[INFO]${NC} Found coordinator script at: $COORDINATOR_SCRIPT"
echo ""

# Create wrapper script
echo -e "${BLUE}[INFO]${NC} Creating hyper wrapper..."

cat > /tmp/hyper << EOF
#!/bin/bash

# Hyperion Coordinator CLI wrapper
# Automatically finds and executes the start-coordinator.sh script

SCRIPT_PATH="$COORDINATOR_SCRIPT"

if [[ ! -f "\$SCRIPT_PATH" ]]; then
    echo "Error: Hyperion Coordinator script not found at: \$SCRIPT_PATH"
    echo "Please reinstall with: cd $SCRIPT_DIR && ./install-hyper.sh"
    exit 1
fi

# Execute the script with all arguments passed through
exec "\$SCRIPT_PATH" "\$@"
EOF

chmod +x /tmp/hyper

echo -e "${GREEN}[SUCCESS]${NC} Wrapper created"
echo ""

# Install to /usr/local/bin
echo -e "${BLUE}[INFO]${NC} Installing to /usr/local/bin/hyper (requires sudo)..."
echo ""

sudo mv /tmp/hyper /usr/local/bin/hyper
sudo chmod +x /usr/local/bin/hyper

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  Hyper Command Installed Successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}✨ You can now use the 'hyper' command from anywhere!${NC}"
echo ""
echo -e "${GREEN}Examples:${NC}"
echo -e "  ${YELLOW}hyper${NC}                                  # Interactive wizard"
echo -e "  ${YELLOW}hyper --folder . --port 8080${NC}          # Start with parameters"
echo -e "  ${YELLOW}hyper --folder ~/projects/my-app${NC}      # Start with specific folder"
echo -e "  ${YELLOW}hyper --stop --name my-project${NC}        # Stop specific instance"
echo -e "  ${YELLOW}hyper --clean --name my-project${NC}       # Clean specific instance"
echo -e "  ${YELLOW}hyper --help${NC}                          # Show help"
echo ""
echo -e "${BLUE}Location:${NC} /usr/local/bin/hyper"
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
