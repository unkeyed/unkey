# CLAUDE.md

This file provides guidance to Claude Code when working with the metald repository.

## Project Overview

metald is a high-performance VM management service providing ConnectRPC APIs for Firecracker and Cloud Hypervisor backends with built-in billing metrics collection.

## Development Commands

```bash
# Build
go build -o build/metald ./cmd/api
go build -o build/stress-test ./cmd/stress-test

# Test
go test ./...
go vet ./...
go mod tidy

# Generate protobuf
buf generate

# Format
goimports -w .
```

## Architecture

- **API Layer**: ConnectRPC/protobuf-based API
- **Service Layer**: VM lifecycle management with billing integration
- **Backend Layer**: Abstracted interface for Firecracker/Cloud Hypervisor
- **Billing Layer**: 100ms precision metrics collection with FIFO streaming

## Key Features

- **Multi-VMM Support**: Unified API for Firecracker and Cloud Hypervisor
- **Billing Integration**: Real-time metrics collection (100ms precision)
- **Process Management**: Dedicated Firecracker processes per VM
- **OpenTelemetry**: Full observability with traces and metrics
- **Stress Testing**: Built-in load testing tools

## Configuration

Primary configuration via environment variables:
- `UNKEY_METALD_BACKEND`: `firecracker` (recommended) or `cloudhypervisor`
- `UNKEY_METALD_OTEL_ENABLED`: Enable telemetry (`true`/`false`)
- `UNKEY_METALD_PORT`: API port (default: `8080`)

## Documentation Structure

- `README.md` - Quick start and overview
- `docs/api-reference.md` - API endpoints and examples
- `docs/backend-support.md` - VMM backend comparison and setup
- `docs/vm-configuration.md` - VM configuration guide
- `docs/observability.md` - OpenTelemetry and monitoring
- `docs/glossary.md` - Key terms and definitions
- `docs/firecracker-process-manager.md` - Process manager architecture
- `docs/firecracker-process-flows.md` - Process lifecycle flows and diagrams
- `docs/jailer-integration.md` - AWS production jailer integration
- `docs/billing-*.md` - Comprehensive billing system documentation

## Quick Start

```bash
# Start with Firecracker (recommended)
UNKEY_METALD_BACKEND=firecracker ./build/metald

# Create a VM
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{"config":{"cpu":{"vcpu_count":1},"memory":{"size_bytes":134217728},"boot":{"kernel_path":"/opt/vm-assets/vmlinux","kernel_args":"console=ttyS0 reboot=k panic=1 pci=off"},"storage":[{"path":"/opt/vm-assets/rootfs.ext4","readonly":false}]}}'

# Run stress test
./build/stress-test -intervals 1 -interval-duration 20s -max-vms 3
```

## Implementation Notes

- **Module Path**: `github.com/unkeyed/unkey/go/deploy/metald`
- **Backend Interface**: All VMMs implement `types.Backend` interface
- **Billing**: Automatic 100ms metrics collection with FIFO streaming (Firecracker)
- **Process Isolation**: One Firecracker process per VM for security
- **Error Handling**: ConnectRPC error model with proper status codes

## Memory Notes

- Every file MUST end with a newline
- Use `goimports -w .` for consistent formatting
- OpenTelemetry requires careful version matching to avoid schema conflicts
- FIFO metrics provide real-time streaming vs file polling
- Backend abstraction enables easy VMM switching
- Process manager handles dedicated Firecracker processes per VM

## Recent Changes

- ✅ Fixed stress test panic with low maxVMs values
- ✅ Implemented proper VM state tracking and cleanup
- ✅ Updated module path to full GitHub path
- ✅ Consolidated and cleaned documentation
- ✅ All package errors resolved (go vet, build, imports)
- ✅ FIFO metrics streaming for real-time billing data
- ✅ **Jailer integration for AWS production deployment**
- ✅ Enhanced process manager with security isolation
- ✅ Comprehensive jailer documentation and configuration
