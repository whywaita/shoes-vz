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
