# shoes-vz Setup Guide

[日本語版はこちら](setup.ja.md)

This document explains the setup procedure for each component of shoes-vz.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Build](#build)
3. [Server Setup](#server-setup)
4. [Agent Setup](#agent-setup)
5. [Verification](#verification)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### Hardware Requirements

- **Apple Silicon Mac** (M1, M2, M3 or later)
- **RAM**: 16GB or more recommended (for VM execution)
- **Storage**: 100GB or more free space (for VM templates + runners)

### Software Requirements

#### Server Execution Environment

- macOS 13.0+ or Linux
- Go 1.21+
- Protocol Buffers compiler (development only)
  - `brew install buf`

#### Agent Execution Environment

- **macOS 26+ (Apple Silicon)**
- **APFS filesystem** (for CoW functionality)
- **Virtualization.framework permissions**
- Go 1.21+

#### Guest VM Requirements

- macOS 13+
- SSH server enabled
- GitHub Actions Runner
- shoes-vz-runner-agent

## Build

### 1. Clone Repository

```bash
git clone https://github.com/whywaita/shoes-vz.git
cd shoes-vz
```

### 2. Install Dependencies

```bash
make deps
```

Or manually:

```bash
go mod download
```

### 3. Generate Protocol Buffers Code

```bash
make proto-generate
```

### 4. Build

```bash
make build
```

Built binaries will be placed in the `bin/` directory:

- `bin/shoes-vz-server`
- `bin/shoes-vz-agent`
- `bin/shoes-vz-runner-agent`
- `bin/shoes-vz-client`

### 5. Verify Build

```bash
./bin/shoes-vz-server -h
./bin/shoes-vz-agent -h
./bin/shoes-vz-runner-agent -h
./bin/shoes-vz-client -h
```

## Server Setup

### Basic Configuration

The server serves as the integration point with myshoes and manages multiple agents.

#### 1. Prepare Configuration File (Optional)

Specify settings via environment variables or startup options.

#### 2. Start Server

```bash
./bin/shoes-vz-server \
  -grpc-addr :50051 \
  -metrics-addr :9090
```

**Options:**

- `-grpc-addr`: gRPC server listen address (default: `:50051`)
- `-metrics-addr`: Prometheus metrics listen address (default: `:9090`)

#### 3. Verification

**Check gRPC:**

```bash
# Using grpcurl (install: brew install grpcurl)
grpcurl -plaintext localhost:50051 list
```

**Check metrics:**

```bash
curl http://localhost:9090/metrics
```

### Running with systemd (Linux)

`/etc/systemd/system/shoes-vz-server.service`:

```ini
[Unit]
Description=shoes-vz Server
After=network.target

[Service]
Type=simple
User=shoesvz
ExecStart=/opt/shoes-vz/bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shoes-vz-server
sudo systemctl start shoes-vz-server
```

### Running with launchd (macOS)

`~/Library/LaunchAgents/com.github.whywaita.shoes-vz-server.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/shoes-vz/bin/shoes-vz-server</string>
        <string>-grpc-addr</string>
        <string>:50051</string>
        <string>-metrics-addr</string>
        <string>:9090</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/shoes-vz-server.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/shoes-vz-server.error.log</string>
</dict>
</plist>
```

Start:

```bash
launchctl load ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-server.plist
```

## Agent Setup

The agent runs on macOS hosts and manages VM creation and lifecycle.

### Verify Prerequisites

#### 1. Check Virtualization.framework Permissions

```bash
# Check permissions
csrutil status
```

Should show `System Integrity Protection status: disabled` or specific developer mode enabled.

#### 2. Verify APFS Volume

```bash
diskutil list
```

Ensure the volume where templates and runners will be placed is APFS.

### Create Directory Structure

```bash
# Template directory
sudo mkdir -p /opt/myshoes/vz/templates

# Runner directory
sudo mkdir -p /opt/myshoes/vz/runners

# Set permissions
sudo chown -R $(whoami):staff /opt/myshoes
```

### Prepare SSH Keys

Create SSH keys for accessing runner VMs.

```bash
ssh-keygen -t ed25519 -f ~/.ssh/shoes-vz-runner -N ""
```

The public key (`~/.ssh/shoes-vz-runner.pub`) will be used later during template creation.

### Start Agent

```bash
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname $(hostname) \
  -max-runners 2 \
  -template-path /opt/myshoes/vz/templates/macos-26 \
  -runners-path /opt/myshoes/vz/runners \
  -ssh-key ~/.ssh/shoes-vz-runner
```

**Options:**

- `-server`: Server gRPC address (default: `localhost:50051`)
- `-hostname`: Agent hostname (default: system hostname)
- `-max-runners`: Maximum number of concurrent runners (default: `2`, limit: `2`)
- `-template-path`: VM template path
- `-runners-path`: Directory for runner VMs
- `-ssh-key`: SSH private key path (optional)

### Running with launchd

`~/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>/opt/shoes-vz/bin/shoes-vz-agent</string>
        <string>-server</string>
        <string>server.example.com:50051</string>
        <string>-hostname</string>
        <string>mac-agent-1</string>
        <string>-max-runners</string>
        <string>2</string>
        <string>-template-path</string>
        <string>/opt/myshoes/vz/templates/macos-26</string>
        <string>-runners-path</string>
        <string>/opt/myshoes/vz/runners</string>
        <string>-ssh-key</string>
        <string>/Users/runner/.ssh/shoes-vz-runner</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/shoes-vz-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/shoes-vz-agent.error.log</string>
</dict>
</plist>
```

Start:

```bash
launchctl load ~/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist
```

## Agent Provisioning

This section describes automated provisioning methods for deploying shoes-vz-agent to multiple macOS hosts.

### Download from GitHub Releases

Binaries are available from GitHub Releases:

```bash
# Set version
VERSION=v0.1.0

# Download binary
curl -L -o shoes-vz-agent.tar.gz \
  "https://github.com/whywaita/shoes-vz/releases/download/${VERSION}/shoes-vz_Darwin_arm64.tar.gz"

# Extract
tar xzf shoes-vz-agent.tar.gz

# Install
sudo mkdir -p /opt/shoes-vz/bin
sudo mv shoes-vz-agent /opt/shoes-vz/bin/
sudo chmod +x /opt/shoes-vz/bin/shoes-vz-agent
```

### Automated Setup Script

Create a provisioning script for automated deployment:

**`setup-agent.sh`:**

```bash
#!/bin/bash
set -euo pipefail

# Configuration
VERSION="${VERSION:-v0.1.0}"
SERVER_ADDR="${SERVER_ADDR:-server.example.com:50051}"
MAX_RUNNERS="${MAX_RUNNERS:-2}"
TEMPLATE_PATH="${TEMPLATE_PATH:-/opt/myshoes/vz/templates/macos-26}"
RUNNERS_PATH="${RUNNERS_PATH:-/opt/myshoes/vz/runners}"
INSTALL_DIR="/opt/shoes-vz"

echo "Installing shoes-vz-agent ${VERSION}..."

# Download binary
curl -L -o /tmp/shoes-vz.tar.gz \
  "https://github.com/whywaita/shoes-vz/releases/download/${VERSION}/shoes-vz_Darwin_arm64.tar.gz"

# Extract and install
mkdir -p "${INSTALL_DIR}/bin"
tar xzf /tmp/shoes-vz.tar.gz -C /tmp
mv /tmp/shoes-vz-agent "${INSTALL_DIR}/bin/"
chmod +x "${INSTALL_DIR}/bin/shoes-vz-agent"
rm /tmp/shoes-vz.tar.gz

# Create directories
mkdir -p "${TEMPLATE_PATH}"
mkdir -p "${RUNNERS_PATH}"

# Generate SSH key if not exists
if [ ! -f ~/.ssh/shoes-vz-runner ]; then
  echo "Generating SSH key..."
  ssh-keygen -t ed25519 -f ~/.ssh/shoes-vz-runner -N ""
fi

# Create launchd plist
PLIST_PATH="${HOME}/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist"
cat > "${PLIST_PATH}" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/bin/shoes-vz-agent</string>
        <string>-server</string>
        <string>${SERVER_ADDR}</string>
        <string>-hostname</string>
        <string>$(hostname)</string>
        <string>-max-runners</string>
        <string>${MAX_RUNNERS}</string>
        <string>-template-path</string>
        <string>${TEMPLATE_PATH}</string>
        <string>-runners-path</string>
        <string>${RUNNERS_PATH}</string>
        <string>-ssh-key</string>
        <string>${HOME}/.ssh/shoes-vz-runner</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/shoes-vz-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/shoes-vz-agent.error.log</string>
</dict>
</plist>
EOF

# Load launchd service
launchctl unload "${PLIST_PATH}" 2>/dev/null || true
launchctl load "${PLIST_PATH}"

echo "Installation complete!"
echo ""
echo "SSH public key for template configuration:"
cat ~/.ssh/shoes-vz-runner.pub
echo ""
echo "Check logs: tail -f /var/log/shoes-vz-agent.log"
```

Usage:

```bash
# Download and run
curl -fsSL https://raw.githubusercontent.com/whywaita/shoes-vz/main/scripts/setup-agent.sh | \
  VERSION=v0.1.0 \
  SERVER_ADDR=server.example.com:50051 \
  bash
```

### Remote Deployment with SSH

Deploy to multiple hosts using SSH:

```bash
# hosts.txt
mac-agent-1.example.com
mac-agent-2.example.com
mac-agent-3.example.com
```

**`deploy-agents.sh`:**

```bash
#!/bin/bash
set -euo pipefail

HOSTS_FILE="hosts.txt"
VERSION="v0.1.0"
SERVER_ADDR="server.example.com:50051"

while IFS= read -r host; do
  echo "Deploying to ${host}..."

  ssh "${host}" "bash -s" < setup-agent.sh <<-ENV
VERSION=${VERSION}
SERVER_ADDR=${SERVER_ADDR}
ENV

  echo "Deployed to ${host}"
done < "${HOSTS_FILE}"

echo "All deployments complete!"
```

### Using Ansible

Create an Ansible playbook for agent deployment:

**`playbook.yml`:**

```yaml
---
- name: Deploy shoes-vz-agent
  hosts: mac_agents
  vars:
    shoes_vz_version: "v0.1.0"
    shoes_vz_server: "server.example.com:50051"
    shoes_vz_max_runners: 2
    shoes_vz_install_dir: "/opt/shoes-vz"
    shoes_vz_template_path: "/opt/myshoes/vz/templates/macos-26"
    shoes_vz_runners_path: "/opt/myshoes/vz/runners"

  tasks:
    - name: Create installation directory
      file:
        path: "{{ shoes_vz_install_dir }}/bin"
        state: directory
        mode: '0755'
      become: yes

    - name: Download shoes-vz binary
      get_url:
        url: "https://github.com/whywaita/shoes-vz/releases/download/{{ shoes_vz_version }}/shoes-vz_Darwin_arm64.tar.gz"
        dest: "/tmp/shoes-vz.tar.gz"

    - name: Extract binary
      unarchive:
        src: "/tmp/shoes-vz.tar.gz"
        dest: "/tmp"
        remote_src: yes

    - name: Install binary
      copy:
        src: "/tmp/shoes-vz-agent"
        dest: "{{ shoes_vz_install_dir }}/bin/shoes-vz-agent"
        mode: '0755'
        remote_src: yes
      become: yes

    - name: Create template directory
      file:
        path: "{{ shoes_vz_template_path }}"
        state: directory
        mode: '0755'

    - name: Create runners directory
      file:
        path: "{{ shoes_vz_runners_path }}"
        state: directory
        mode: '0755'

    - name: Generate SSH key
      openssh_keypair:
        path: "{{ ansible_env.HOME }}/.ssh/shoes-vz-runner"
        type: ed25519
        comment: "shoes-vz-runner"

    - name: Deploy launchd plist
      template:
        src: templates/shoes-vz-agent.plist.j2
        dest: "{{ ansible_env.HOME }}/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist"

    - name: Load launchd service
      shell: |
        launchctl unload {{ ansible_env.HOME }}/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist || true
        launchctl load {{ ansible_env.HOME }}/Library/LaunchAgents/com.github.whywaita.shoes-vz-agent.plist
```

**`templates/shoes-vz-agent.plist.j2`:**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.github.whywaita.shoes-vz-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{ shoes_vz_install_dir }}/bin/shoes-vz-agent</string>
        <string>-server</string>
        <string>{{ shoes_vz_server }}</string>
        <string>-hostname</string>
        <string>{{ ansible_hostname }}</string>
        <string>-max-runners</string>
        <string>{{ shoes_vz_max_runners }}</string>
        <string>-template-path</string>
        <string>{{ shoes_vz_template_path }}</string>
        <string>-runners-path</string>
        <string>{{ shoes_vz_runners_path }}</string>
        <string>-ssh-key</string>
        <string>{{ ansible_env.HOME }}/.ssh/shoes-vz-runner</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/shoes-vz-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/shoes-vz-agent.error.log</string>
</dict>
</plist>
```

Run:

```bash
ansible-playbook -i inventory.ini playbook.yml
```

### Server Deployment with Docker

For shoes-vz-server, use Docker:

```bash
# Pull image
docker pull ghcr.io/whywaita/shoes-vz-server:latest

# Run server
docker run -d \
  --name shoes-vz-server \
  -p 50051:50051 \
  -p 9090:9090 \
  ghcr.io/whywaita/shoes-vz-server:latest
```

**`docker-compose.yml`:**

```yaml
version: '3.8'

services:
  shoes-vz-server:
    image: ghcr.io/whywaita/shoes-vz-server:latest
    ports:
      - "50051:50051"
      - "9090:9090"
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "grpcurl", "-plaintext", "localhost:50051", "list"]
      interval: 30s
      timeout: 10s
      retries: 3
```

Run:

```bash
docker-compose up -d
```

## Verification

### 1. Check Server Logs

```bash
# systemd
sudo journalctl -u shoes-vz-server -f

# launchd
tail -f /var/log/shoes-vz-server.log
```

Expected logs:

```
Starting shoes-vz-server
gRPC address: :50051
Metrics address: :9090
Metrics server listening on :9090
gRPC server listening on :50051
```

### 2. Check Agent Logs

```bash
# launchd
tail -f /var/log/shoes-vz-agent.log
```

Expected logs:

```
Starting shoes-vz-agent
Server: localhost:50051
Hostname: mac-agent-1
Max runners: 4
Template path: /opt/myshoes/vz/templates/macos-26
Runners path: /opt/myshoes/vz/runners
Connected to server
Agent registered: agent-id-xxx
```

### 3. Check Metrics

```bash
curl http://localhost:9090/metrics | grep shoesvz
```

Expected output:

```
shoesvz_agents_online 1
shoesvz_capacity_total_runners 4
shoesvz_runners_total{state="creating"} 0
shoesvz_runners_total{state="running"} 0
```

### 4. Verify gRPC Connection

Use grpcurl to verify server connection:

```bash
# List services
grpcurl -plaintext localhost:50051 list

# Check agent status (adjust to actual gRPC methods)
grpcurl -plaintext localhost:50051 shoes.vz.agent.v1.AgentService/...
```

## Troubleshooting

### Agent Cannot Connect to Server

**Symptom:**
```
Failed to connect to server: connection refused
```

**Solution:**

1. Check if server is running:
   ```bash
   ps aux | grep shoes-vz-server
   ```

2. Check if port is open:
   ```bash
   lsof -i :50051
   ```

3. Check firewall settings:
   ```bash
   # macOS
   sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate
   ```

### VM Template Not Found

**Symptom:**
```
Template not found: /opt/myshoes/vz/templates/macos-26
```

**Solution:**

1. Check if template directory exists:
   ```bash
   ls -la /opt/myshoes/vz/templates/
   ```

2. Check if required files are present:
   ```bash
   ls -la /opt/myshoes/vz/templates/macos-26/
   # Required: Disk.img, AuxiliaryStorage
   ```

3. If template not created, refer to [image-build.md](./image-build.md)

### APFS Clone Fails

**Symptom:**
```
Failed to clone disk: operation not supported
```

**Solution:**

1. Verify APFS volume:
   ```bash
   diskutil info /opt/myshoes | grep "Type (Bundle)"
   # Should be APFS
   ```

2. Check if template and runners are on same volume:
   ```bash
   df /opt/myshoes/vz/templates
   df /opt/myshoes/vz/runners
   # Should be same mount point
   ```

### Virtualization.framework Error

**Symptom:**
```
Failed to create VM: not entitled
```

**Solution:**

1. Check code signature and entitlements:
   ```bash
   codesign -d --entitlements - ./bin/shoes-vz-agent
   ```

2. Check if developer mode is enabled:
   ```bash
   DevToolsSecurity -status
   ```

3. Enable developer mode if needed:
   ```bash
   sudo DevToolsSecurity -enable
   ```

### Out of Memory Error

**Symptom:**
```
Failed to start VM: insufficient memory
```

**Solution:**

1. Check system memory usage:
   ```bash
   vm_stat
   ```

2. Reduce `-max-runners`:
   ```bash
   # Example: 2 → 1
   -max-runners 1
   ```

3. Adjust VM memory allocation (requires code modification):
   ```go
   // internal/agent/vm/vm.go
   // Change 4GB → 2GB
   2*1024*1024*1024, // 2GB memory
   ```

## Next Steps

1. **Create VM Template**: Refer to [image-build.md](./image-build.md) to create a Golden Template.
2. **Integrate with myshoes**: Configure myshoes and register shoes-vz as a plugin.
3. **Setup Monitoring**: Collect metrics with Prometheus and create dashboards with Grafana.

## Related Documentation

- [image-build.md](./image-build.md) - VM template creation procedure
- [README.md](../README.md) - Project overview
- [plans/](../plans/) - Detailed design documents
