# metald - VM Management Service

A high-performance VM management service providing ConnectRPC APIs for Firecracker and Cloud Hypervisor backends with built-in billing metrics collection.

## Features

- **Multi-VMM Support**: Firecracker and Cloud Hypervisor backends
- **ConnectRPC API**: Modern gRPC-compatible interface
- **Billing Metrics**: 100ms precision resource tracking with FIFO streaming
- **OpenTelemetry**: Full observability with traces and metrics
- **Health Monitoring**: Comprehensive system health checks
- **Stress Testing**: Built-in load testing tools

## Quick Start

```bash
# Build the service
go build -o build/metald ./cmd/api

# Start with Firecracker backend (recommended)
UNKEY_METALD_BACKEND=firecracker ./build/metald

# Or with Cloud Hypervisor
UNKEY_METALD_BACKEND=cloudhypervisor ./build/metald -socket /tmp/ch.sock
```

## API Endpoints

### VM Management
- `POST /vmprovisioner.v1.VmService/CreateVm` - Create a new VM
- `POST /vmprovisioner.v1.VmService/BootVm` - Boot a VM
- `POST /vmprovisioner.v1.VmService/ShutdownVm` - Shutdown a VM
- `POST /vmprovisioner.v1.VmService/DeleteVm` - Delete a VM
- `POST /vmprovisioner.v1.VmService/ListVms` - List all VMs
- `POST /vmprovisioner.v1.VmService/GetVmInfo` - Get VM details

### Health & Metrics
- `/metrics` - Prometheus metrics (port 9464)

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `UNKEY_METALD_BACKEND` | Backend type (`firecracker`/`cloudhypervisor`) | `cloudhypervisor` |
| `UNKEY_METALD_PORT` | API server port | `8080` |
| `UNKEY_METALD_ADDRESS` | Bind address | `0.0.0.0` |
| `UNKEY_METALD_OTEL_ENABLED` | Enable OpenTelemetry | `false` |
| `UNKEY_METALD_OTEL_SAMPLING_RATE` | Trace sampling rate (0.0-1.0) | `1.0` |

## VM Configuration Example

```bash
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 1},
      "memory": {"size_bytes": 134217728},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1 pci=off"
      },
      "storage": [{
        "path": "/opt/vm-assets/rootfs.ext4",
        "readonly": false
      }]
    }
  }'
```

## Stress Testing

```bash
# Build stress test tool
go build -o build/stress-test ./cmd/stress-test

# Run quick test (3 VMs, 20 seconds)
./build/stress-test -intervals 1 -interval-duration 20s -max-vms 3

# Run longer test (up to 10 VMs, 5 intervals of 2 minutes each)
./build/stress-test -intervals 5 -max-vms 10
```

## Observability

### OpenTelemetry
```bash
# Enable full telemetry
export UNKEY_METALD_OTEL_ENABLED=true
export UNKEY_METALD_OTEL_SAMPLING_RATE=1.0
./build/metald
```

### Prometheus Metrics
- VM lifecycle metrics
- API request metrics
- Billing collection metrics
- Available at `http://localhost:9464/metrics`

## Development

```bash
# Format and test
go fmt ./...
go vet ./...
go test ./...

# Generate protobuf code
buf generate

# Clean dependencies
go mod tidy
```

## Architecture

metald acts as a controlled frontend for VM management with billing integration:

```
┌─────────────┐   ConnectRPC   ┌─────────────┐   Unix Socket   ┌────────────────┐
│ Deploy CP   │───────────────▶│   metald    │────────────────▶│ Firecracker/   │
└─────────────┘                │             │                 │ Cloud Hypervisor│
                               │ • VM APIs   │                 └────────────────┘
                               │ • Billing   │
                               │ • Health    │                 ┌────────────────┐
                               │ • Metrics   │────────────────▶│ Billing Service│
                               └─────────────┘   FIFO Metrics  └────────────────┘
```

## Billing Integration

metald automatically collects VM resource usage metrics:
- **Frequency**: 100ms precision collection
- **Transport**: FIFO streaming for real-time data
- **Batching**: 1-minute batches sent to billing service
- **Reliability**: Write-ahead log for failure recovery

See `docs/billing-*.md` for detailed billing documentation.
