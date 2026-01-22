#!/bin/bash
# Setup script to run inside the VM
# Create minimal image for shoes-vz testing
set -euo pipefail

echo "=== Enabling SSH ==="
sudo systemsetup -setremotelogin on

echo "=== Creating runner user ==="
# Check if runner user exists
if id runner &>/dev/null; then
    echo "Runner user already exists, ensuring home directory is set up correctly..."
else
    # Create user with sysadminctl (macOS 10.10 and later)
    # -addUser: username
    # -fullName: full name
    # -password: password (plain text)
    # -home: home directory (default is /Users/username)
    # -admin: grant administrator privileges
    echo "Creating runner user with sysadminctl..."
    sudo sysadminctl -addUser runner \
      -fullName "GitHub Actions Runner" \
      -password "runner" \
      -admin \
      -home /Users/runner
fi

# Create home directory if it doesn't exist
if [ ! -d /Users/runner ]; then
    echo "Creating home directory for runner user..."
    sudo mkdir -p /Users/runner
fi

# Check if NFSHomeDirectory is correctly set
CURRENT_HOME=$(sudo dscl . -read /Users/runner NFSHomeDirectory 2>/dev/null | awk '{print $2}')
if [ "$CURRENT_HOME" != "/Users/runner" ]; then
    echo "Setting NFSHomeDirectory to /Users/runner..."
    sudo dscl . -create /Users/runner NFSHomeDirectory /Users/runner
fi

# Set permissions
echo "Setting permissions for home directory..."
sudo chown -R runner:staff /Users/runner
sudo chmod 755 /Users/runner

echo "Runner user setup completed successfully"

# Configure sudoers (allow sudo without password)
echo "=== Configuring sudoers for runner user ==="
echo "runner ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/runner
sudo chmod 440 /etc/sudoers.d/runner

echo "=== Setting up SSH for runner user ==="
sudo -u runner mkdir -p /Users/runner/.ssh
sudo -u runner chmod 700 /Users/runner/.ssh

# Add to authorized_keys if SSH key exists
if [ -f /tmp/ssh_public_key ]; then
    sudo -u runner cp /tmp/ssh_public_key /Users/runner/.ssh/authorized_keys
    sudo -u runner chmod 600 /Users/runner/.ssh/authorized_keys
    echo "SSH public key installed for runner user"
fi

echo "=== Installing Homebrew and basic tools as runner user ==="
# Install Homebrew and basic tools (run as runner user)
# -H option sets HOME environment variable to runner's home directory
sudo -H -u runner bash << 'HOMEBREW_SCRIPT'
# Install Homebrew (non-interactive)
NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Configure Homebrew PATH
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> /Users/runner/.zprofile
eval "$(/opt/homebrew/bin/brew shellenv)"

echo "=== Installing basic tools ==="
brew install git curl wget jq yq
HOMEBREW_SCRIPT

echo "=== Installing shoes-vz-runner-agent ==="
if [ -f /tmp/shoes-vz-runner-agent ]; then
    sudo mkdir -p /usr/local/bin
    sudo mv /tmp/shoes-vz-runner-agent /usr/local/bin/
    sudo chmod +x /usr/local/bin/shoes-vz-runner-agent
    echo "shoes-vz-runner-agent installed"
else
    echo "Warning: shoes-vz-runner-agent not found at /tmp/shoes-vz-runner-agent"
fi

echo "=== Setting up runner-monitor LaunchDaemon ==="
# Deploy as LaunchDaemon (runs at system boot without login)

# Create plist file
# shoes-vz-runner-agent watches .runner file at boot time,
# and automatically notifies IP when runner ID is found
sudo tee /Library/LaunchDaemons/com.github.whywaita.shoes-vz-runner-agent.plist > /dev/null << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-runner-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/shoes-vz-runner-agent</string>
        <string>-listen</string>
        <string>:8080</string>
        <string>-runner-path</string>
        <string>/tmp/runner</string>
        <string>-host-ip</string>
        <string>192.168.64.1</string>
        <string>-agent-port</string>
        <string>8081</string>
    </array>
    <key>UserName</key>
    <string>runner</string>
    <key>GroupName</key>
    <string>staff</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/runner-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/runner-agent.error.log</string>
</dict>
</plist>
EOF

sudo chmod 644 /Library/LaunchDaemons/com.github.whywaita.shoes-vz-runner-agent.plist
echo "LaunchDaemon plist created at /Library/LaunchDaemons/"
echo "shoes-vz-runner-agent will start at boot time (no login required)"
echo "It will run as 'runner' user and automatically watch for .runner file and notify IP when runner is registered"

echo "=== Cleanup ==="
# Delete cache (ignore errors)
sudo rm -rf /Library/Caches/* 2>/dev/null || true
sudo rm -rf /Users/runner/Library/Caches/* 2>/dev/null || true

# Delete logs (ignore errors)
sudo rm -rf /var/log/* 2>/dev/null || true
sudo rm -rf /Users/runner/Library/Logs/* 2>/dev/null || true

# Delete history
sudo rm -f /Users/runner/.bash_history /Users/runner/.zsh_history

# Disable Spotlight (for faster startup)
sudo mdutil -a -i off

echo "=== Setup complete ==="
echo "You can now shutdown the VM with: sudo shutdown -h now"
