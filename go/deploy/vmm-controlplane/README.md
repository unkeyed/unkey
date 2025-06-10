# VMM Control Plane

The VMM Control Plane (VMCP) provides a controlled, observable frontend for creating and managing VMs on a host. It acts as an intermediary between the Deploy control plane and VMM instances (Cloud Hypervisor, Firecracker), offering a ConnectRPC-based API with comprehensive observability through OpenTelemetry.

## Features

- **Unified ConnectRPC API** supporting multiple hypervisor backends
- **Multi-backend support** for Cloud Hypervisor, Firecracker, and future VMMs
- **Full OpenTelemetry instrumentation** with tracing and metrics
- **Configurable trace sampling** with parent-based sampling and always-on error capture
- **Dual metrics export** via OTLP push and Prometheus pull
- **Health checks** with comprehensive system metrics
- **Unix socket support** for VMM communication
- **Graceful shutdown** handling

## Quick Start

```bash
# Build the API server
make build

# Start with default configuration (Cloud Hypervisor backend)
./build/vmcp-api

# Or with OpenTelemetry enabled
export UNKEY_VMCP_OTEL_ENABLED=true
./build/vmcp-api
```

## Configuration

All configuration is done via environment variables:

### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `UNKEY_VMCP_PORT` | API server port | `8080` |
| `UNKEY_VMCP_ADDRESS` | Bind address | `0.0.0.0` |
| `UNKEY_VMCP_BACKEND` | Backend type (cloudhypervisor/firecracker) | `cloudhypervisor` |
| `UNKEY_VMCP_CH_ENDPOINT` | Cloud Hypervisor endpoint | `unix:///tmp/ch.sock` |
| `UNKEY_VMCP_FC_ENDPOINT` | Firecracker endpoint | `unix:///tmp/firecracker.sock` |

### OpenTelemetry Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `UNKEY_VMCP_OTEL_ENABLED` | Enable OpenTelemetry | `false` |
| `UNKEY_VMCP_OTEL_SERVICE_NAME` | Service name for telemetry | `vmm-controlplane` |
| `UNKEY_VMCP_OTEL_SERVICE_VERSION` | Service version | `0.0.1` |
| `UNKEY_VMCP_OTEL_SAMPLING_RATE` | Trace sampling rate (0.0-1.0) | `1.0` |
| `UNKEY_VMCP_OTEL_ENDPOINT` | OTLP endpoint | `localhost:4318` |
| `UNKEY_VMCP_OTEL_PROMETHEUS_ENABLED` | Enable Prometheus metrics | `true` |
| `UNKEY_VMCP_OTEL_PROMETHEUS_PORT` | Prometheus metrics port | `9464` |

## Observability

VMCP includes comprehensive observability with OpenTelemetry:

### Start the Observability Stack

```bash
# Start Grafana LGTM stack (requires Podman)
make o11y

# Access Grafana at http://localhost:3000 (admin/admin)
# Stop the stack
make o11y-stop
```

### Trace Sampling

VMCP uses parent-based sampling with configurable ratios:
- **Parent-based**: Honors upstream sampling decisions for distributed traces
- **Configurable ratio**: Set `UNKEY_VMCP_OTEL_SAMPLING_RATE` between 0.0 and 1.0
- **Always-on for errors**: Errors are always captured regardless of sampling rate

### Metrics

Metrics are exported via two mechanisms:
1. **OTLP Push**: Sends metrics to the configured OTLP endpoint
2. **Prometheus Pull**: Exposes metrics on `http://0.0.0.0:9464/metrics`

Available metrics include:
- `rpc_server_requests_total`: Total RPC requests by method and status
- `rpc_server_request_duration_seconds`: Request duration histogram
- `rpc_server_active_requests`: Currently active requests
- System metrics (CPU, memory, uptime) via health endpoint

## API Endpoints

### VM Management (ConnectRPC)
- `/vmm.v1.VmService/CreateVm`
- `/vmm.v1.VmService/DeleteVm`
- `/vmm.v1.VmService/BootVm`
- `/vmm.v1.VmService/ShutdownVm`
- `/vmm.v1.VmService/PauseVm`
- `/vmm.v1.VmService/ResumeVm`
- `/vmm.v1.VmService/RebootVm`
- `/vmm.v1.VmService/GetVmInfo`

### Health & Metrics
- `/health` - Simple health check
- `/_/health` - Comprehensive health with system metrics
- `/metrics` - Prometheus metrics (when OTEL enabled)

## Development

```bash
# Install development tools
make install-tools

# Generate protobuf code
make generate

# Run tests
make test

# Run with auto-reload (requires air)
make dev

# Run linter
make lint
```

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Deploy Control │────▶│      VMCP        │────▶│ Cloud Hypervisor│
│     Plane       │     │  (ConnectRPC)    │     │   or Firecracker│
└─────────────────┘     └──────────────────┘     │   (Unix Socket) │
                               │                 └─────────────────┘
                               ▼                   
                        ┌──────────────────┐
                        │   Backend Layer  │
                        │ (VMM Abstraction)│
                        └──────────────────┘
                               │
                               ▼
                        ┌──────────────────┐
                        │   Observability  │
                        │  (OTLP/Grafana)  │
                        └──────────────────┘
```

## Monitoring

When running with the observability stack:

1. **Traces**: View in Grafana Tempo
2. **Metrics**: View in Grafana Prometheus/Mimir
3. **Logs**: Application logs are structured JSON to stdout

## License

[License details here]
