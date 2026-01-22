# VM Template (Golden Image) Build Guide

[日本語版はこちら](image-build.ja.md)

This document explains the procedure for creating a macOS Tahoe VM template for shoes-vz using Tart.

## Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Installing Tart](#installing-tart)
4. [Creating the Template](#creating-the-template)
5. [Testing the Template](#testing-the-template)
6. [Troubleshooting](#troubleshooting)
7. [Customization](#customization)

## Overview

The Golden Template (template image) is the VM image that serves as the basis for ephemeral Runner VMs. Runner VMs are cloned quickly from this template using APFS CoW (Copy-on-Write).

### Why Use Tart

- **Fast**: Image download and setup completes in 30-60 minutes
- **Simple**: No macOS installer required, uses ready-to-use vanilla images
- **Lightweight**: Only essential tools installed (approximately 15-20GB)
- **Reproducible**: Can be automated with scripts

### Template Requirements

- macOS 13+
- SSH server enabled
- Dedicated user account (`runner`)
- shoes-vz-runner-agent
  - Runner state monitoring functionality
  - State exposure via HTTP API
- Basic development tools (Git, Homebrew)

**Important:** The GitHub Actions Runner is **not included in the template**. All Runner download, installation, and registration is performed by the setup_script.

**IP Detection Mechanism:** After VM startup, the host automatically discovers the VM's IP address by scanning the NAT range (192.168.64.0/24) and checking connectivity to the SSH port (22).

### Template Structure

```
template-name.bundle/
├── Disk.img              # VM disk image
├── AuxiliaryStorage      # Auxiliary storage for macOS VM
└── HardwareModel.json    # Hardware model information (extracted from Tart)
```

**HardwareModel.json Format:**
```json
{
  "hardwareModel": "YnBsaXN0MDDUAQIDBAUGBwpYJHZlcnNpb25..."
}
```

A JSON file containing base64-encoded hardware model data.

## Prerequisites

### Host Environment

- macOS 13+ (Apple Silicon)
- Sufficient storage capacity (30GB or more recommended)
- APFS volume
- Homebrew

### Time Requirements

- **Download**: 10-15 minutes (approximately 10GB)
- **Setup**: 15-30 minutes
- **Total**: 30-60 minutes

## Installing Tart

```bash
# Install via Homebrew
brew install cirruslabs/cli/tart

# Verify version
tart --version
```

## Creating the Template

### 1. Clone the Vanilla Image

Use the macOS Tahoe vanilla image provided by Cirrus Labs as the base.

```bash
# Clone macOS Tahoe
tart clone ghcr.io/cirruslabs/macos-tahoe-vanilla:latest shoes-vz-template
```

**Initial State of Vanilla Image:**
- User: `admin` / Password: `admin`
- SSH: Disabled
- Homebrew: Not installed

### 2. Start the VM

```bash
# Start the VM (GUI window will open)
tart run shoes-vz-template
```

After startup, log in with `admin` / `admin`.

### 3. Enable SSH

Execute the following in the VM console:

```bash
# Enable SSH
sudo systemsetup -setremotelogin on

# Verify
sudo systemsetup -getremotelogin
# Output: Remote Login: On
```

### 4. Check VM IP Address

**In a separate terminal (on the host side)**:

```bash
# Get VM IP address
IP=$(tart ip shoes-vz-template)
echo "VM IP: $IP"

# Test SSH connection
ssh admin@$IP
# Password: admin
```

### 5. Transfer Setup Files

**Execute on the host:**

```bash
cd /path/to/shoes-vz

# Build shoes-vz-runner-agent
make build

# Transfer necessary files to VM
IP=$(tart ip shoes-vz-template)

scp scripts/setup-minimal-image.sh admin@$IP:/tmp/
scp bin/shoes-vz-runner-agent admin@$IP:/tmp/

# Transfer SSH public key (optional)
scp ~/.ssh/id_ed25519.pub admin@$IP:/tmp/ssh_public_key
```

### 6. Execute Setup Script

**SSH into the VM:**

```bash
ssh admin@$IP
```

**Execute the script inside the VM:**

```bash
# Grant execution permissions
chmod +x /tmp/setup-minimal-image.sh

# Execute the script
/tmp/setup-minimal-image.sh
```

**What the script does:**
- Verify SSH is enabled
- Create runner user (UID: 502)
- Place SSH public key (if `/tmp/ssh_public_key` exists)
- Install Homebrew
- Install basic tools (git, curl, wget, jq, yq)
- Place shoes-vz-runner-agent (`/usr/local/bin/`)
- Configure LaunchAgent (auto-start, with IP notification)
- System cleanup (cache, log deletion)
- Disable Spotlight (for faster startup)

After completion, the following message will be displayed:

```
=== Setup complete ===
You can now shutdown the VM with: sudo shutdown -h now
```

### 7. Shutdown the VM

**Execute inside the VM:**

```bash
sudo shutdown -h now
```

### 8. Convert to Template Format

**Execute on the host:**

```bash
# Create template directory
sudo mkdir -p /opt/myshoes/vz/templates/macos-tahoe

# Copy Tart VM image to template format
sudo cp ~/.tart/vms/shoes-vz-template/disk.img /opt/myshoes/vz/templates/macos-tahoe/Disk.img
sudo cp ~/.tart/vms/shoes-vz-template/nvram.bin /opt/myshoes/vz/templates/macos-tahoe/AuxiliaryStorage

# Create HardwareModel.json (required)
if [ -f ~/.tart/vms/shoes-vz-template/config.json ]; then
    # Extract hardwareModel from Tart's config.json and save in JSON format
    jq '{hardwareModel: .hardwareModel}' ~/.tart/vms/shoes-vz-template/config.json | \
        sudo tee /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json > /dev/null
else
    echo "Error: Tart config.json not found. Cannot create HardwareModel.json"
    exit 1
fi

# Verify correct format
if ! jq -e '.hardwareModel' /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json > /dev/null 2>&1; then
    echo "Error: HardwareModel.json is not in the correct format"
    exit 1
fi

# Set permissions
sudo chown -R $(whoami):staff /opt/myshoes/vz/templates/macos-tahoe
chmod 644 /opt/myshoes/vz/templates/macos-tahoe/*

# Check disk size
ls -lh /opt/myshoes/vz/templates/macos-tahoe/

# Verify file structure (expected output)
# Disk.img            # About 20GB
# AuxiliaryStorage    # A few MB
# HardwareModel.json  # About 1KB

# Verify HardwareModel.json contents
cat /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
# Expected output: {"hardwareModel":"YnBsaXN0MDDUAQIDBAUGBwpY..."}
```

**Important Verification Items:**
- ✅ `Disk.img` exists and is approximately 20GB
- ✅ `AuxiliaryStorage` exists
- ✅ `HardwareModel.json` is in JSON format and contains the `hardwareModel` key

### 9. Create Template Metadata (Optional)

```bash
cat > /opt/myshoes/vz/templates/macos-tahoe/TemplateMetadata.json << 'EOF'
{
  "name": "macos-tahoe",
  "version": "15.x",
  "created_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "description": "macOS Tahoe vanilla template for GitHub Actions self-hosted runner",
  "base_image": "ghcr.io/cirruslabs/macos-tahoe-vanilla:latest",
  "cpu_count": 2,
  "memory_gb": 4,
  "disk_size_gb": 20,
  "features": [
    "SSH enabled",
    "runner user created",
    "shoes-vz-runner-agent installed",
    "HTTP API for monitoring",
    "Homebrew installed",
    "Basic tools (git, curl, wget, jq, yq)"
  ],
  "note": "GitHub Actions Runner will be installed via setup_script at runtime"
}
EOF
```

## Testing the Template

### 1. Test with shoes-vz-agent

**Start shoes-vz-server in a separate terminal:**

```bash
cd /path/to/shoes-vz
./bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090
```

**Start shoes-vz-agent:**

```bash
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname test-agent \
  -max-runners 1 \
  -template-path /opt/myshoes/vz/templates/macos-tahoe \
  -runners-path /tmp/test-runners \
  -ssh-key ~/.ssh/id_ed25519
```

**Expected logs:**

```
Starting shoes-vz-agent
Server: localhost:50051
Hostname: test-agent
Max runners: 1
Template path: /opt/myshoes/vz/templates/macos-tahoe
Runners path: /tmp/test-runners
Agent registered with ID: xxx
```

### 2. VM Creation Test

Run Go tests:

```bash
cd /path/to/shoes-vz

# Test with specified template path
TEST_VM_TEMPLATE=/opt/myshoes/vz/templates/macos-tahoe \
  go test -v ./internal/agent/vm/ -run TestVMManager_Create
```

**Expected output:**

```
=== RUN   TestVMManager_Create
Starting VM for runner xxx...
Waiting for VM to reach running state...
VM state: VirtualMachineStateRunning
VM is now running, discovering guest IP via TCP/IP...
IP discovery attempt 1...
Trying common NAT IPs...
Found guest IP: 192.168.64.2
Guest IP discovered: 192.168.64.2
--- PASS: TestVMManager_Create (30.00s)
PASS
```

### 3. Verify SSH Connection

After the VM starts, test SSH connection:

```bash
# Check IP address from runner-agent logs
ssh -i ~/.ssh/id_ed25519 runner@192.168.64.2 whoami
# Output: runner
```

### 4. Verify runner-agent Operation

```bash
# Check runner-agent logs
ssh -i ~/.ssh/id_ed25519 runner@192.168.64.2 tail -f /Users/runner/runner-agent.log
```

**Expected logs:**

```
Starting shoes-vz-runner-agent
Starting on TCP, listen_addr=:8080
Using runner path, path=/Users/runner/actions-runner
Starting HTTP server on :8080
```

## Troubleshooting

### Tart Download is Slow

**Symptom:**
Image download takes more than 30 minutes

**Solution:**

```bash
# Check download progress
tart list

# Clear cache
rm -rf ~/.tart/cache/

# Try a different mirror (if applicable)
```

### Cannot Connect via SSH

**Symptom:**
```
ssh: connect to host [IP] port 22: Connection refused
```

**Solution:**

1. Verify SSH is enabled (inside VM):
   ```bash
   sudo systemsetup -getremotelogin
   ```

2. Check firewall (inside VM):
   ```bash
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   ```

3. Restart SSH service (inside VM):
   ```bash
   sudo launchctl stop com.openssh.sshd
   sudo launchctl start com.openssh.sshd
   ```

### runner-agent Won't Start

**Symptom:**
```
Failed to start runner-agent
```

**Solution:**

1. Verify binary exists (inside VM):
   ```bash
   ls -la /usr/local/bin/shoes-vz-runner-agent
   ```

2. Check LaunchAgent logs (inside VM):
   ```bash
   tail -f /Users/runner/runner-agent.error.log
   ```

3. Test manual startup (inside VM):
   ```bash
   sudo -u runner /usr/local/bin/shoes-vz-runner-agent \
     -listen :8080 \
     -runner-path /Users/runner/actions-runner
   ```

4. Reload LaunchAgent (inside VM):
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-runner-agent.plist
   launchctl load ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-runner-agent.plist
   ```

### IP Address Not Detected

**Symptom:**
```
VM is now running, discovering guest IP via TCP/IP...
IP discovery attempt 1...
Trying common NAT IPs...
timeout discovering guest IP after 3 minutes
```

**Solution:**

1. Verify SSH server is running (inside VM):
   ```bash
   sudo launchctl list | grep sshd
   ```

2. Check network interfaces (inside VM):
   ```bash
   ifconfig | grep "inet "
   ```

3. Verify firewall allows SSH (inside VM):
   ```bash
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   ```

4. Try manual SSH connection from host:
   ```bash
   ssh -i ~/.ssh/id_ed25519 runner@192.168.64.2
   ```

### HardwareModel.json Not Found

**Symptom:**
```
hardware model not found in template: /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
```

**Cause:**
HardwareModel.json does not exist or is not created in the correct format.

**Solution:**

1. Verify file exists:
   ```bash
   ls -la /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
   ```

2. Check file contents:
   ```bash
   cat /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
   ```

   Expected format:
   ```json
   {
     "hardwareModel": "YnBsaXN0MDDUAQIDBAUGBwpY..."
   }
   ```

3. Recreate if missing or incorrect format:
   ```bash
   # Extract from Tart VM
   jq '{hardwareModel: .hardwareModel}' ~/.tart/vms/shoes-vz-template/config.json | \
       sudo tee /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json > /dev/null

   # Verify correct creation
   jq -e '.hardwareModel' /opt/myshoes/vz/templates/macos-tahoe/HardwareModel.json
   ```

4. If Tart VM does not exist:
   ```bash
   # Recreate Tart VM
   tart clone ghcr.io/cirruslabs/macos-tahoe-vanilla:latest shoes-vz-template

   # Execute the above procedure
   ```

### Template Size is Too Large

**Symptom:**
Disk.img is 30GB or more

**Solution:**

1. Delete unnecessary files (inside VM, execute before shutdown):
   ```bash
   # Delete Homebrew cache
   brew cleanup -s

   # Delete Xcode cache (if installed)
   rm -rf ~/Library/Developer/Xcode/DerivedData/*

   # Delete system logs
   sudo rm -rf /var/log/*
   sudo rm -rf ~/Library/Logs/*
   ```

2. Compress disk (inside VM):
   ```bash
   # Zero fill
   sudo dd if=/dev/zero of=/tmp/zero.dat bs=1m || true
   sudo rm /tmp/zero.dat
   ```

3. Optimize with Tart (on host):
   ```bash
   # After stopping VM
   tart prune shoes-vz-template
   ```

## Customization

### Installing Xcode Command Line Tools

If Xcode is needed, add the following to the setup script:

```bash
echo "=== Installing Xcode Command Line Tools ==="
xcode-select --install

# Wait for installation to complete
until xcode-select -p &> /dev/null; do
  sleep 5
done
```

### Installing Additional Tools

Add tools with Homebrew:

```bash
echo "=== Installing additional tools ==="
brew install \
  node \
  python@3.11 \
  go \
  rust \
  docker
```

### Adding Custom Users

Add users other than runner:

```bash
echo "=== Creating custom user ==="
sudo dscl . -create /Users/myuser
sudo dscl . -create /Users/myuser UserShell /bin/bash
sudo dscl . -create /Users/myuser RealName "My User"
sudo dscl . -create /Users/myuser UniqueID 503
sudo dscl . -create /Users/myuser PrimaryGroupID 20
sudo dscl . -create /Users/myuser NFSHomeDirectory /Users/myuser
sudo mkdir -p /Users/myuser
sudo chown myuser:staff /Users/myuser
```

### Customizing Setup Script

Copy `scripts/setup-minimal-image.sh` to create your own script:

```bash
cp scripts/setup-minimal-image.sh scripts/setup-custom.sh

# Edit the script
vim scripts/setup-custom.sh

# Transfer to VM and execute
scp scripts/setup-custom.sh admin@$IP:/tmp/
ssh admin@$IP 'chmod +x /tmp/setup-custom.sh && /tmp/setup-custom.sh'
```

## Best Practices

### 1. Template Version Management

```bash
# Include version in template name
/opt/myshoes/vz/templates/
├── macos-tahoe-v1/
├── macos-tahoe-v2/
└── macos-tahoe-latest -> macos-tahoe-v2  # Symbolic link

# Version management
ln -sf macos-tahoe-v2 /opt/myshoes/vz/templates/macos-tahoe-latest
```

### 2. Regular Template Updates

```bash
#!/bin/bash
# update-template.sh - Template update script

# Use current date as version
VERSION=$(date +%Y%m%d)
TEMPLATE_NAME="macos-tahoe-$VERSION"

# Clone new vanilla image
tart clone ghcr.io/cirruslabs/macos-tahoe-vanilla:latest $TEMPLATE_NAME

# Setup process...
# (Automate the above steps)

# Delete old templates (older than 3 generations)
# ...
```

### 3. Security Configuration

```bash
# Enable firewall (inside VM)
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on

# Enable Gatekeeper (inside VM)
sudo spctl --master-enable

# Disable FileVault (for performance, recommended)
# System Settings > Privacy & Security > FileVault
```

### 4. Test Automation

```bash
#!/bin/bash
# test-template.sh

TEMPLATE_PATH="/opt/myshoes/vz/templates/macos-sequoia"

# Check template existence
if [ ! -d "$TEMPLATE_PATH" ]; then
  echo "❌ Template not found: $TEMPLATE_PATH"
  exit 1
fi

# Check required files
for file in Disk.img AuxiliaryStorage; do
  if [ ! -f "$TEMPLATE_PATH/$file" ]; then
    echo "❌ Missing file: $file"
    exit 1
  fi
done

# VM creation test
echo "🧪 Testing VM creation..."
TEST_VM_TEMPLATE="$TEMPLATE_PATH" \
  go test -v ./internal/agent/vm/ -run TestVMManager_Create

if [ $? -eq 0 ]; then
  echo "✅ Template test passed"
else
  echo "❌ Template test failed"
  exit 1
fi
```

## Next Steps

After verifying the template works properly:

1. **Production Agent Launch**: Refer to [setup.md](./setup.md) to launch the Agent in production
2. **Integration with myshoes**: Use shoes-vz-client to integrate with myshoes
3. **Monitoring and Metrics**: Collect metrics with Prometheus and create dashboards with Grafana

## Related Documentation

- [setup.md](./setup.md) - shoes-vz setup procedure
- [README.md](../README.md) - Project overview
- [Tart Official Documentation](https://tart.run/)
- [Cirrus Labs VM Images](https://github.com/cirruslabs/macos-image-templates)
- [Apple Virtualization Framework Documentation](https://developer.apple.com/documentation/virtualization)
