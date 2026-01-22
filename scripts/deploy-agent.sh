#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
GITHUB_REPO="whywaita/shoes-vz"
VERSION="latest"
SERVER_ADDR=""
MAX_RUNNERS="2"
TEMPLATE_PATH="/opt/myshoes/vz/templates/macos-26"
RUNNERS_PATH="/opt/myshoes/vz/runners"
IP_NOTIFY_PORT="8081"
INSTALL_DIR="/usr/local/bin"
PLIST_DIR="/Library/LaunchDaemons"
PLIST_NAME="com.github.whywaita.shoes-vz-agent.plist"
LOG_DIR="/var/log"
WORK_DIR="/var/lib/shoes-vz"

# Parse command line arguments
usage() {
    cat <<EOF
Usage: $0 -s SERVER_ADDR [OPTIONS]

Required:
  -s SERVER_ADDR          Server gRPC address (e.g., localhost:50051)

Optional:
  -v VERSION              Version to install (default: latest)
  -m MAX_RUNNERS          Maximum number of concurrent runners (default: 2)
  -t TEMPLATE_PATH        Path to VM template (default: /opt/myshoes/vz/templates/macos-26)
  -r RUNNERS_PATH         Path to runners directory (default: /opt/myshoes/vz/runners)
  -p IP_NOTIFY_PORT       Port for IP notification server (default: 8081)
  -h                      Show this help message

Examples:
  # Install latest version
  $0 -s localhost:50051

  # Install specific version with custom settings
  $0 -s prod-server:50051 -v v1.0.0 -m 4 -t /custom/templates

EOF
    exit 1
}

while getopts "s:v:m:t:r:p:h" opt; do
    case $opt in
        s) SERVER_ADDR="$OPTARG" ;;
        v) VERSION="$OPTARG" ;;
        m) MAX_RUNNERS="$OPTARG" ;;
        t) TEMPLATE_PATH="$OPTARG" ;;
        r) RUNNERS_PATH="$OPTARG" ;;
        p) IP_NOTIFY_PORT="$OPTARG" ;;
        h) usage ;;
        *) usage ;;
    esac
done

# Validate required arguments
if [ -z "$SERVER_ADDR" ]; then
    echo -e "${RED}Error: Server address is required${NC}"
    usage
fi

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run as root (use sudo)${NC}"
    exit 1
fi

echo -e "${GREEN}=== shoes-vz-agent Deployment Script ===${NC}"
echo "Repository: $GITHUB_REPO"
echo "Version: $VERSION"
echo "Server Address: $SERVER_ADDR"
echo "Max Runners: $MAX_RUNNERS"
echo "Template Path: $TEMPLATE_PATH"
echo "Runners Path: $RUNNERS_PATH"
echo ""

# Determine download URL
echo -e "${YELLOW}Fetching release information...${NC}"
if [ "$VERSION" = "latest" ]; then
    RELEASE_URL="https://api.github.com/repos/$GITHUB_REPO/releases/latest"
else
    RELEASE_URL="https://api.github.com/repos/$GITHUB_REPO/releases/tags/$VERSION"
fi

# Get release info
RELEASE_INFO=$(curl -sL "$RELEASE_URL")
DOWNLOAD_URL=$(echo "$RELEASE_INFO" | grep -o "https://github.com/$GITHUB_REPO/releases/download/.*/shoes-vz-agent.*darwin.*arm64" | head -1)

if [ -z "$DOWNLOAD_URL" ]; then
    # Try alternative naming patterns
    DOWNLOAD_URL=$(echo "$RELEASE_INFO" | grep -o "https://github.com/$GITHUB_REPO/releases/download/.*/shoes-vz-agent" | head -1)
fi

if [ -z "$DOWNLOAD_URL" ]; then
    echo -e "${RED}Error: Could not find download URL for shoes-vz-agent${NC}"
    echo "Please check if the release exists and contains the binary"
    exit 1
fi

echo "Download URL: $DOWNLOAD_URL"

# Stop existing service if running
if launchctl list | grep -q "$PLIST_NAME"; then
    echo -e "${YELLOW}Stopping existing service...${NC}"
    launchctl unload "$PLIST_DIR/$PLIST_NAME" 2>/dev/null || true
fi

# Download binary
echo -e "${YELLOW}Downloading shoes-vz-agent...${NC}"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl -sL -o "$TMP_DIR/shoes-vz-agent" "$DOWNLOAD_URL"
chmod +x "$TMP_DIR/shoes-vz-agent"

# Install binary
echo -e "${YELLOW}Installing binary to $INSTALL_DIR...${NC}"
mv "$TMP_DIR/shoes-vz-agent" "$INSTALL_DIR/shoes-vz-agent"

# Create necessary directories
echo -e "${YELLOW}Creating directories...${NC}"
mkdir -p "$WORK_DIR"
mkdir -p "$RUNNERS_PATH"
mkdir -p "$(dirname "$TEMPLATE_PATH")"

# Create plist file
echo -e "${YELLOW}Creating LaunchDaemon plist...${NC}"
cat > "$PLIST_DIR/$PLIST_NAME" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-agent</string>

    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/shoes-vz-agent</string>
        <string>-server</string>
        <string>$SERVER_ADDR</string>
        <string>-max-runners</string>
        <string>$MAX_RUNNERS</string>
        <string>-template-path</string>
        <string>$TEMPLATE_PATH</string>
        <string>-runners-path</string>
        <string>$RUNNERS_PATH</string>
        <string>-ip-notify-port</string>
        <string>$IP_NOTIFY_PORT</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <dict>
        <key>Crashed</key>
        <true/>
        <key>SuccessfulExit</key>
        <false/>
    </dict>

    <key>WorkingDirectory</key>
    <string>$WORK_DIR</string>

    <key>StandardOutPath</key>
    <string>$LOG_DIR/shoes-vz-agent.log</string>

    <key>StandardErrorPath</key>
    <string>$LOG_DIR/shoes-vz-agent-error.log</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
    </dict>

    <key>ProcessType</key>
    <string>Adaptive</string>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>ExitTimeOut</key>
    <integer>30</integer>

    <key>HardResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>4096</integer>
    </dict>

    <key>SoftResourceLimits</key>
    <dict>
        <key>NumberOfFiles</key>
        <integer>2048</integer>
    </dict>

    <key>Nice</key>
    <integer>0</integer>

    <key>Umask</key>
    <integer>22</integer>

    <key>Debug</key>
    <false/>
</dict>
</plist>
EOF

# Set correct permissions
chmod 644 "$PLIST_DIR/$PLIST_NAME"

# Load and start service
echo -e "${YELLOW}Loading and starting service...${NC}"
launchctl load "$PLIST_DIR/$PLIST_NAME"

# Wait a moment for service to start
sleep 2

# Check if service is running
if launchctl list | grep -q "com.github.whywaita.shoes-vz-agent"; then
    echo -e "${GREEN}✓ Service successfully started${NC}"
else
    echo -e "${RED}✗ Service failed to start${NC}"
    echo "Check logs at $LOG_DIR/shoes-vz-agent-error.log"
    exit 1
fi

echo ""
echo -e "${GREEN}=== Deployment Complete ===${NC}"
echo ""
echo "Service: com.github.whywaita.shoes-vz-agent"
echo "Binary: $INSTALL_DIR/shoes-vz-agent"
echo "Plist: $PLIST_DIR/$PLIST_NAME"
echo "Logs: $LOG_DIR/shoes-vz-agent.log"
echo "Error Logs: $LOG_DIR/shoes-vz-agent-error.log"
echo ""
echo "Useful commands:"
echo "  Check status:  sudo launchctl list | grep shoes-vz-agent"
echo "  Stop service:  sudo launchctl unload $PLIST_DIR/$PLIST_NAME"
echo "  Start service: sudo launchctl load $PLIST_DIR/$PLIST_NAME"
echo "  View logs:     tail -f $LOG_DIR/shoes-vz-agent.log"
echo "  View errors:   tail -f $LOG_DIR/shoes-vz-agent-error.log"
echo ""
