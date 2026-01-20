# shoes-vz

[日本語版 README はこちら](README.ja.md)

shoes-vz is a tool suite for creating, running, and destroying **ephemeral macOS VMs** as GitHub Actions self-hosted runners on macOS 26+ (Apple Silicon) using **Code-Hex/vz (Go bindings for Apple Virtualization Framework)**.

## Overview

shoes-vz minimizes VM startup time to SSH-ready state by leveraging **APFS Copy-on-Write (clone)** for instant runner replication from templates.

### Key Features

- **Fast SSH Ready**: Unified startup completion condition based on successful SSH connection
- **Ephemeral Runners**: Fully isolated VM instances per runner
- **CoW-based Fast Cloning**: Instant template replication using APFS clone
- **macOS 26+ Native**: Leverages latest Virtualization.framework capabilities
- **gRPC-centric Design**: GUI-independent, gRPC integration with myshoes

## System Components

shoes-vz consists of three components:

### shoes-vz-server (single instance)
- gRPC integration with myshoes
- Agent management (registration, health monitoring)
- Runner scheduling
- Aggregated runner state management

### shoes-vz-agent (one per macOS host)
- Virtualization.framework control (via vz)
- Template management (cloning)
- VM lifecycle management
- State synchronization with server

### shoes-vz-runner-agent (inside each guest macOS VM)
- GitHub Actions Runner state monitoring
- HTTP API for state exposure
- Automatic IP address notification to host
- Command execution via HTTP requests from host

## Documentation

- **[Setup Guide](docs/setup.md)** - Installation and configuration steps for each component
- **[Image Build Guide](docs/image-build.md)** - Instructions for creating VM templates (Golden Images)
- **[Design Document](docs/design.md)** - Detailed architecture and design explanation

## Build

```bash
# Install dependencies
make deps

# Generate Go code from Proto files
make proto-generate

# Build (with entitlement signing)
make build
```

**Note**: Since we use Virtualization Framework, the build process automatically signs binaries with `com.apple.security.virtualization` entitlement using adhoc signature (`codesign -s -`). No developer certificate required.

## Running

### Server

```bash
# Start gRPC and metrics server
./bin/shoes-vz-server -grpc-addr :50051 -metrics-addr :9090

# Check metrics
curl http://localhost:9090/metrics
```

### Agent

```bash
# Basic startup
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname my-host \
  -max-runners 2 \
  -template-path /opt/myshoes/vz/templates/macos-26 \
  -runners-path /opt/myshoes/vz/runners \
  -ip-notify-port 8081

# Enable graphics for debugging (opens GUI window)
./bin/shoes-vz-agent \
  -server localhost:50051 \
  -hostname my-host \
  -max-runners 2 \
  -template-path /opt/myshoes/vz/templates/macos-26 \
  -runners-path /opt/myshoes/vz/runners \
  -ip-notify-port 8081 \
  -enable-graphics
```

### Runner Agent (inside guest VM)

```bash
# Basic startup (automatically reads runner ID from .runner file)
./bin/shoes-vz-runner-agent \
  -listen :8080 \
  -runner-path /tmp/runner \
  -host-ip 192.168.64.1 \
  -agent-port 8081

# Manually specify runner ID
./bin/shoes-vz-runner-agent \
  -listen :8080 \
  -runner-path /tmp/runner \
  -runner-id my-runner-001 \
  -host-ip 192.168.64.1 \
  -agent-port 8081
```

## Testing

```bash
# Run all tests
make test

# VM Manager tests (requires template)
TEST_VM_TEMPLATE=/path/to/template go test -v ./internal/agent/vm/
```

## Project Structure

```
shoes-vz/
├── apis/proto/           # Protocol Buffers definitions
├── cmd/                  # Entry points
│   ├── shoes-vz-server/
│   ├── shoes-vz-agent/
│   ├── shoes-vz-runner-agent/
│   └── shoes-vz-client/
├── internal/             # Internal implementations
│   ├── server/          # Server implementation
│   ├── agent/           # Agent implementation
│   ├── monitor/         # Runner Agent implementation
│   └── client/          # myshoes plugin client implementation
├── pkg/model/           # Shared models
└── gen/                 # Generated code
```

## API

### myshoes Integration API (gRPC)

- `AddInstance`: Create a runner instance
- `DeleteInstance`: Delete a runner instance

### Agent-Server API (gRPC)

- `RegisterAgent`: Register an agent
- `Sync`: Bidirectional streaming for state synchronization

See `.proto` files under `apis/proto/` for details.

## Implementation Status

### Completed

- ✅ Protocol Buffers definitions and code generation
- ✅ shoes-vz-server basic implementation
  - gRPC handlers
  - Agent management (registration, state sync)
  - Runner scheduling
  - State management (Store)
- ✅ shoes-vz-agent basic implementation
  - Runner Manager
  - VM Manager (using Code-Hex/vz)
  - Bidirectional sync with server
- ✅ shoes-vz-runner-agent implementation
  - Automatic IP address notification on VM boot (via HTTP)
  - Automatic runner ID detection from .runner file
  - Runner state monitoring
  - HTTP API
- ✅ VM operations implementation
  - Template cloning via APFS clone
  - VM create/start/stop/delete
  - SSH connection check
- ✅ Network implementation
  - NAT network configuration
  - IP address notification from guest VM to host (HTTP POST)
  - SSH connection (IP address based)
  - HTTP-based host-guest communication (port 8080)
  - IP notification server (host side, port 8081)
- ✅ runner-agent HTTP API
  - Command execution endpoint (`/exec`)
  - State retrieval endpoint (`/status`)
  - Health check endpoint (`/health`)
- ✅ Prometheus metrics
  - Runner state metrics (total, idle, busy, error)
  - Agent state metrics (online count, capacity)
  - Capacity metrics (utilization rate, available count)
  - Performance metrics (startup time, request processing time)
  - `/metrics` endpoint (port 9090)
- ✅ Basic tests

### Next Steps

1. **VM Template Creation**
   - Install macOS in VM
   - Configure SSH and create user
   - Install GitHub Actions Runner
   - Deploy shoes-vz-runner-agent and LaunchDaemon configuration

2. **Error Handling Improvements**
   - More detailed error messages
   - Recovery procedures
   - Timeout optimization

3. **Real-world Testing**
   - Integration tests with templates
   - Integration tests with myshoes
   - HTTP communication performance tests

## Requirements

### Host (Agent execution environment)
- macOS 26+ (Apple Silicon)
- APFS filesystem
- Virtualization.framework permissions
- Go 1.21+
- buf CLI (for Proto code generation)

### Guest (inside VM)
- macOS 13+
- GitHub Actions Runner
- shoes-vz-runner-agent (state monitoring and HTTP API)

## Major Dependencies

- [Code-Hex/vz](https://github.com/Code-Hex/vz) - Go bindings for Apple Virtualization Framework
- [prometheus/client_golang](https://github.com/prometheus/client_golang) - Prometheus metrics
- [grpc/grpc-go](https://github.com/grpc/grpc-go) - gRPC implementation

## License

MIT
