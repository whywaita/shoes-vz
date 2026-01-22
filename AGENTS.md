# AGENTS.md - AI Agent Development Guide

This document provides essential information for AI agents working on the shoes-vz project. It focuses on practical implementation details and common development patterns.

## Documentation Policy

**English is the primary language for all code and documentation.** However, we also provide Japanese translations (`.ja.md`) for key documents to support Japanese-speaking contributors.

When creating or updating documentation:
- **Primary**: Always write/update English version first (e.g., `README.md`, `docs/setup.md`)
- **Secondary**: Provide Japanese translation when appropriate (e.g., `README.ja.md`, `docs/setup.ja.md`)
- **Code**: All code comments, commit messages, and PR descriptions must be in English
- **Consistency**: Keep both language versions synchronized when making updates

## Project Overview

**shoes-vz** is a tool suite for creating, running, and destroying ephemeral macOS VMs as GitHub Actions self-hosted runners on macOS 26+ (Apple Silicon) using Apple's Virtualization Framework via [Code-Hex/vz](https://github.com/Code-Hex/vz) Go bindings.

### Core Concept

- **Fast VM Cloning**: Uses APFS Copy-on-Write (CoW) to instantly replicate VM templates
- **Ephemeral Runners**: Each GitHub Actions job runs in a fresh, isolated VM
- **SSH-Ready Metric**: VM startup is complete when SSH connection succeeds
- **gRPC Architecture**: Server-Agent communication via gRPC bidirectional streaming

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                          myshoes                                 │
│                    (External Orchestrator)                       │
└───────────────────────────┬─────────────────────────────────────┘
                            │ gRPC (AddInstance/DeleteInstance)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    shoes-vz-server                               │
│  - Runner scheduling                                             │
│  - Agent management                                              │
│  - State aggregation                                             │
└───────────────────────────┬─────────────────────────────────────┘
                            │ gRPC (RegisterAgent/Sync)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                    shoes-vz-agent (macOS host)                   │
│  - VM lifecycle (create/start/stop/delete)                       │
│  - Template cloning (APFS CoW)                                   │
│  - IP notification server (port 8081)                            │
└───────────────────────────┬─────────────────────────────────────┘
                            │ HTTP (IP notification + command exec)
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│              shoes-vz-runner-agent (inside VM)                   │
│  - Runner state monitoring                                       │
│  - HTTP API (port 8080)                                          │
│  - Automatic IP notification                                     │
└─────────────────────────────────────────────────────────────────┘
```

## Three Main Components

### 1. shoes-vz-server (Single Instance)
- **Location**: `cmd/shoes-vz-server/`
- **Purpose**: Central coordinator, integrates with myshoes
- **Key Features**:
  - Registers and monitors agents
  - Schedules runners across agents
  - Exposes gRPC API for myshoes
  - Provides Prometheus metrics (port 9090)
- **State Management**: In-memory store (`internal/server/store/`)

### 2. shoes-vz-agent (One per macOS Host)
- **Location**: `cmd/shoes-vz-agent/`
- **Purpose**: VM management on Apple Silicon hosts
- **Key Features**:
  - Controls Virtualization.framework via Code-Hex/vz
  - Clones templates using APFS CoW
  - Manages VM lifecycle
  - Bidirectional sync with server
  - Runs IP notification server (port 8081)
- **Requirements**: macOS 26+, Apple Silicon, APFS, `com.apple.security.virtualization` entitlement

### 3. shoes-vz-runner-agent (Inside Guest VM)
- **Location**: `cmd/shoes-vz-runner-agent/`
- **Purpose**: Monitor runner state inside VM
- **Key Features**:
  - HTTP API for state queries (port 8080)
  - Automatic IP notification to host on boot
  - Reads runner ID from `.runner` file
  - Executes commands via HTTP
- **Deployment**: Runs as LaunchDaemon inside guest VM

## Directory Structure

```
shoes-vz/
├── apis/proto/                    # Protocol Buffers definitions
│   ├── agent/v1/                  # Agent-Server API
│   └── plugin/v1/                 # myshoes integration API
├── cmd/                           # Entry points
│   ├── shoes-vz-server/          # Server binary
│   ├── shoes-vz-agent/           # Agent binary
│   ├── shoes-vz-runner-agent/    # Runner monitor binary
│   └── shoes-vz-client/          # Client binary (testing)
├── internal/                      # Internal packages
│   ├── server/                   # Server implementation
│   │   ├── grpc/                # gRPC handlers
│   │   ├── store/               # State management
│   │   └── scheduler/           # Runner scheduling
│   ├── agent/                    # Agent implementation
│   │   ├── runner/              # Runner manager
│   │   ├── vm/                  # VM manager (vz wrapper)
│   │   ├── sync/                # Server sync client
│   │   └── ipnotify/            # IP notification server
│   ├── monitor/                  # Runner agent implementation
│   └── client/                   # myshoes plugin client
├── pkg/                          # Public packages
│   ├── model/                   # Shared models
│   └── logging/                 # Logging utilities
├── gen/                          # Generated code (from protobuf)
├── scripts/                      # Deployment/setup scripts
│   ├── deploy-agent.sh          # Agent deployment from GitHub Release
│   ├── uninstall-agent.sh       # Agent uninstallation
│   ├── setup-minimal-image.sh   # VM template setup
│   ├── deploy-tart-to-template.sh # Tart VM conversion
│   └── README.md                # Scripts documentation
├── docs/                         # Documentation
│   ├── setup.md                 # Setup guide (English)
│   ├── setup.ja.md              # Setup guide (Japanese)
│   ├── design.md                # Architecture design (English)
│   ├── design.ja.md             # Architecture design (Japanese)
│   ├── image-build.md           # VM template creation (English)
│   └── image-build.ja.md        # VM template creation (Japanese)
├── README.md                     # Project README (English)
└── README.ja.md                  # Project README (Japanese)
```

## Key Technologies

### Go Libraries
- **Code-Hex/vz**: Go bindings for Apple Virtualization Framework
- **grpc-go**: gRPC implementation
- **protobuf**: Protocol Buffers
- **prometheus/client_golang**: Metrics collection
- **slog**: Structured logging (standard library)

### macOS Features
- **Virtualization.framework**: VM management
- **APFS CoW**: Fast disk cloning via `cp -c`
- **LaunchDaemon**: System service management

## Development Workflow

### 1. Build Process

```bash
# Install dependencies (buf, protoc-gen-go, protoc-gen-go-grpc)
make deps

# Generate Go code from .proto files
make generate

# Build all binaries (with entitlement signing)
make build

# Run tests
make test
```

**Important**: The build process automatically signs binaries with `com.apple.security.virtualization` entitlement using adhoc signature (`codesign -s -`).

### 2. Protocol Buffers

When modifying `.proto` files:

```bash
# Lint proto files
make proto-lint

# Check for breaking changes
make proto-breaking

# Generate Go code
make generate
```

### 3. Code Style

- **Go**: Follow [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- **Comments**: All code comments must be in English
- **Formatting**: Run `go fmt` before committing
- **Testing**: Include both success and failure test cases
- **Documentation**:
  - Write English documentation first (e.g., `docs/feature.md`)
  - Add Japanese translation when appropriate (e.g., `docs/feature.ja.md`)
  - Keep both versions synchronized

### 4. Testing Strategy

```bash
# Unit tests
go test ./...

# VM manager tests (requires template)
TEST_VM_TEMPLATE=/path/to/template go test -v ./internal/agent/vm/

# Integration tests (manual)
# 1. Start server
./bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090

# 2. Start agent (requires VM template)
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -template-path /path/to/template \
  -runners-path /tmp/runners
```

## VM Template Requirements

Templates must contain:
- `Disk.img` - macOS disk image (APFS)
- `AuxiliaryStorage` - NVRAM data
- `HardwareModel.json` - Hardware model configuration

**Template Creation Options**:
1. Use `scripts/setup-minimal-image.sh` inside a VM
2. Convert from Tart VM using `scripts/deploy-tart-to-template.sh`

**SSH Requirements**:
- SSH server enabled (`sudo systemsetup -setremotelogin on`)
- User `runner` with password `runner`
- Public key in `/Users/runner/.ssh/authorized_keys`

## Network Architecture

### Host → Guest Communication
- **IP Notification**: Guest POSTs to host `http://192.168.64.1:8081/notify` on boot
- **Command Execution**: Host sends commands to guest `http://<guest-ip>:8080/exec`
- **Health Check**: Host queries `http://<guest-ip>:8080/health`

### NAT Configuration
- Default gateway: `192.168.64.1` (host)
- DHCP range: `192.168.64.2` - `192.168.64.254`
- Guest VMs get dynamic IPs in this range

## Common Development Tasks

### Adding a New gRPC Method

1. Update `.proto` file in `apis/proto/`
2. Run `make generate`
3. Implement handler in `internal/server/grpc/` or `internal/agent/sync/`
4. Add tests
5. Update documentation

### Modifying VM Behavior

1. Edit `internal/agent/vm/manager.go`
2. Update vz configuration in `createVM()` or related methods
3. Test with actual VM template
4. Consider impact on startup time

### Adding Metrics

1. Define metric in appropriate package (server/agent)
2. Register with Prometheus registry
3. Update metric in relevant code paths
4. Test via `/metrics` endpoint

### Adding Documentation

1. Create English version first (e.g., `docs/feature.md`)
2. Write clear, concise content with code examples
3. Add Japanese translation (e.g., `docs/feature.ja.md`)
4. Link from README.md and README.ja.md
5. Keep both versions synchronized when updating

## Troubleshooting Guide

### Build Issues

**Problem**: `command not found: buf`
```bash
# Solution: Install buf
brew install buf
# Or use make deps
make deps
```

**Problem**: Code generation fails
```bash
# Solution: Clean and regenerate
rm -rf gen/
make generate
```

### Runtime Issues

**Problem**: Agent fails to start VM
- Check entitlement: `codesign -d --entitlements - bin/shoes-vz-agent`
- Verify template exists: `ls -la /path/to/template`
- Check macOS version: Must be 26+ (macOS 15+)
- Verify Apple Silicon: `uname -m` should show `arm64`

**Problem**: VM doesn't get IP address
- Check IP notification server is running on host (port 8081)
- Verify runner-agent LaunchDaemon in guest: `sudo launchctl list | grep shoes-vz`
- Check guest logs: `/tmp/runner-agent.log`, `/tmp/runner-agent.error.log`

**Problem**: SSH connection fails
- Verify SSH is enabled in guest: `sudo systemsetup -getremotelogin`
- Check runner user exists: `id runner`
- Test manual SSH: `ssh runner@<guest-ip>`

## Deployment

### Development Deployment

Use local binaries:
```bash
./bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090
./bin/shoes-vz-agent -server localhost:50051 -template-path /path/to/template
```

### Production Deployment

Use deployment script:
```bash
# Download from GitHub Release and install as LaunchDaemon
sudo ./scripts/deploy-agent.sh -s server:50051 -v v1.0.0

# Check service status
sudo launchctl list | grep shoes-vz-agent

# View logs
tail -f /var/log/shoes-vz-agent.log
```

### Uninstallation

```bash
# Remove service only
sudo ./scripts/uninstall-agent.sh

# Remove everything including data
sudo ./scripts/uninstall-agent.sh --remove-data --remove-runners
```

## API Reference

### Server gRPC API (myshoes integration)

**Service**: `PluginService` (port 50051)
- `AddInstance(AddInstanceRequest) → AddInstanceResponse`
  - Creates a new runner instance
- `DeleteInstance(DeleteInstanceRequest) → DeleteInstanceResponse`
  - Deletes a runner instance

### Agent gRPC API (Server-Agent sync)

**Service**: `AgentService` (port 50051)
- `RegisterAgent(RegisterAgentRequest) → RegisterAgentResponse`
  - Registers agent with server
- `Sync(stream SyncRequest) → stream SyncResponse`
  - Bidirectional streaming for state synchronization

### Runner Agent HTTP API

**Endpoints** (port 8080):
- `POST /notify` - IP notification from guest to host
- `POST /exec` - Execute command in guest
- `GET /status` - Get runner state
- `GET /health` - Health check

## Important Implementation Details

### APFS CoW Cloning

Template cloning uses `cp -c` for Copy-on-Write:
```go
// Fast path: APFS clone (CoW)
cmd := exec.Command("cp", "-c", templatePath, targetPath)
```

This is instant and space-efficient. Only diffs consume additional space.

### VM Startup Flow

1. Clone template (APFS CoW) → instant
2. Create VM configuration (vz) → ~100ms
3. Start VM → ~5-10s
4. Guest boots and notifies IP → ~10-20s
5. SSH ready check → ~1s
6. **Total**: ~15-30s to SSH-ready

### State Synchronization

Agent and Server maintain eventual consistency via bidirectional gRPC streaming:
- Agent sends local state changes
- Server sends scheduling commands
- Both sides reconcile state periodically

### Error Handling Patterns

```go
// Prefer early returns
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Use structured logging
logger.Error("failed to start VM", "vm_id", vmID, "error", err)

// Cleanup on error
defer func() {
    if err != nil {
        cleanup()
    }
}()
```

## CI/CD

- **Linting**: golangci-lint (see `.golangci.yml`)
- **Testing**: GitHub Actions on macOS runners
- **Build**: Automated binary builds with entitlement signing

## Getting Help

- **Documentation**: See `docs/` directory
  - English: `docs/setup.md`, `docs/design.md`, `docs/image-build.md`
  - Japanese: `docs/setup.ja.md`, `docs/design.ja.md`, `docs/image-build.ja.md`
- **Issues**: Check existing issues on GitHub
- **Design**: Read `docs/design.md` for architecture rationale
- **README**: `README.md` (English) or `README.ja.md` (Japanese)

## Quick Reference

```bash
# Build
make build

# Test
make test

# Format
make fmt

# Lint
make lint

# Clean
make clean

# Deploy agent
sudo scripts/deploy-agent.sh -s server:50051

# Check agent status
sudo launchctl list | grep shoes-vz-agent

# View logs
tail -f /var/log/shoes-vz-agent.log

# Uninstall
sudo scripts/uninstall-agent.sh
```

---

**Last Updated**: 2026-01-22
**Maintained by**: whywaita
**License**: MIT
