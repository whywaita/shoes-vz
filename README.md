# shoes-vz

[Japanese](README.ja.md)

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
- **[AGENTS.md](AGENTS.md)** - Developer guide with build instructions, API reference, and troubleshooting

## Getting Started

```bash
# Install dependencies and build
make deps
make build

# See AGENTS.md for detailed build instructions, running examples, and development workflow
```

## Requirements

### Host (Agent execution environment)
- macOS 26+ (Apple Silicon)
- APFS filesystem
- Virtualization.framework permissions

### Guest (inside VM)
- macOS 13+
- GitHub Actions Runner
- shoes-vz-runner-agent

## Major Dependencies

- [Code-Hex/vz](https://github.com/Code-Hex/vz) - Go bindings for Apple Virtualization Framework
- [prometheus/client_golang](https://github.com/prometheus/client_golang) - Prometheus metrics
- [grpc/grpc-go](https://github.com/grpc/grpc-go) - gRPC implementation

## License

MIT
