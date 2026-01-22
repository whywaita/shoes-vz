#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

INSTALL_DIR="/usr/local/bin"
PLIST_DIR="/Library/LaunchDaemons"
PLIST_NAME="com.github.whywaita.shoes-vz-agent.plist"
LOG_DIR="/var/log"
WORK_DIR="/var/lib/shoes-vz"

# Parse command line arguments
REMOVE_DATA=false
REMOVE_RUNNERS=false

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Options:
  --remove-data      Remove working directory ($WORK_DIR)
  --remove-runners   Remove runners directory (specified in plist)
  -h, --help         Show this help message

Examples:
  # Uninstall service only
  $0

  # Uninstall and remove all data
  $0 --remove-data --remove-runners

EOF
    exit 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --remove-data)
            REMOVE_DATA=true
            shift
            ;;
        --remove-runners)
            REMOVE_RUNNERS=true
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            ;;
    esac
done

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

echo -e "${GREEN}=== shoes-vz-agent Uninstallation Script ===${NC}"
echo ""

# Stop and unload service
if [ -f "$PLIST_DIR/$PLIST_NAME" ]; then
    echo -e "${YELLOW}Stopping service...${NC}"
    if launchctl list | grep -q "com.github.whywaita.shoes-vz-agent"; then
        launchctl unload "$PLIST_DIR/$PLIST_NAME" 2>/dev/null || true
        echo -e "${GREEN}✓ Service stopped${NC}"
    else
        echo "Service not running"
    fi

    # Remove plist file
    echo -e "${YELLOW}Removing plist file...${NC}"
    rm -f "$PLIST_DIR/$PLIST_NAME"
    echo -e "${GREEN}✓ Plist file removed${NC}"
else
    echo "Plist file not found, skipping service stop"
fi

# Remove binary
if [ -f "$INSTALL_DIR/shoes-vz-agent" ]; then
    echo -e "${YELLOW}Removing binary...${NC}"
    rm -f "$INSTALL_DIR/shoes-vz-agent"
    echo -e "${GREEN}✓ Binary removed${NC}"
else
    echo "Binary not found, skipping"
fi

# Remove log files
echo -e "${YELLOW}Removing log files...${NC}"
rm -f "$LOG_DIR/shoes-vz-agent.log"
rm -f "$LOG_DIR/shoes-vz-agent-error.log"
echo -e "${GREEN}✓ Log files removed${NC}"

# Remove working directory if requested
if [ "$REMOVE_DATA" = true ]; then
    if [ -d "$WORK_DIR" ]; then
        echo -e "${YELLOW}Removing working directory...${NC}"
        rm -rf "$WORK_DIR"
        echo -e "${GREEN}✓ Working directory removed${NC}"
    fi
fi

# Remove runners directory if requested
if [ "$REMOVE_RUNNERS" = true ]; then
    # Try to extract runners path from plist if it was backed up
    if [ -f "$PLIST_DIR/$PLIST_NAME.bak" ]; then
        RUNNERS_PATH=$(grep -A1 "runners-path" "$PLIST_DIR/$PLIST_NAME.bak" | tail -1 | sed 's/.*<string>\(.*\)<\/string>.*/\1/')
    else
        # Default path
        RUNNERS_PATH="/opt/myshoes/vz/runners"
    fi

    if [ -n "$RUNNERS_PATH" ] && [ -d "$RUNNERS_PATH" ]; then
        echo -e "${YELLOW}Removing runners directory: $RUNNERS_PATH${NC}"
        echo -e "${RED}WARNING: This will delete all runner VMs!${NC}"
        read -p "Are you sure? (yes/no): " CONFIRM
        if [ "$CONFIRM" = "yes" ]; then
            rm -rf "$RUNNERS_PATH"
            echo -e "${GREEN}✓ Runners directory removed${NC}"
        else
            echo "Skipping runners directory removal"
        fi
    fi
fi

echo ""
echo -e "${GREEN}=== Uninstallation Complete ===${NC}"
echo ""

if [ "$REMOVE_DATA" = false ] || [ "$REMOVE_RUNNERS" = false ]; then
    echo "Note: Some data may still remain:"
    if [ "$REMOVE_DATA" = false ] && [ -d "$WORK_DIR" ]; then
        echo "  - Working directory: $WORK_DIR"
    fi
    if [ "$REMOVE_RUNNERS" = false ]; then
        echo "  - Runners directory (check your configuration)"
    fi
    echo ""
    echo "To remove all data, run:"
    echo "  sudo $0 --remove-data --remove-runners"
fi
