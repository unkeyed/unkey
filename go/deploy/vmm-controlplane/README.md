# Cloud Hypervisor Control Plane

The Cloud Hypervisor Control Plane (CHCP) provides a controlled, observable frontend for creating and managing VMs on a host. It acts as an intermediary between the Deploy control plane and Cloud Hypervisor instances, offering a ConnectRPC-based API with comprehensive observability through OpenTelemetry.

## Features

- **ConnectRPC API** based on Cloud Hypervisor's [OpenAPI v3 spec](https://raw.githubusercontent.com/cloud-hypervisor/cloud-hypervisor/master/vmm/src/api/openapi/cloud-hypervisor.yaml)
- **Full OpenTelemetry instrumentation** with tracing and metrics
- **Configurable trace sampling** with parent-based sampling and always-on error capture
- **Dual metrics export** via OTLP push and Prometheus pull
- **Health checks** with comprehensive system metrics
- **Unix socket support** for Cloud Hypervisor communication
- **Graceful shutdown** handling

## Quick Start

```bash
# Build the API server
make build

# Start with default configuration
./build/chcp-api

# Or with OpenTelemetry enabled
export UNKEY_CHCP_OTEL_ENABLED=true
./build/chcp-api
```

## Configuration

All configuration is done via environment variables:

### Server Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `CHCP_PORT` | API server port | `8080` |
| `CHCP_ADDRESS` | Bind address | `0.0.0.0` |
| `CHCP_BACKEND` | Backend type | `cloudhypervisor` |
| `CHCP_CH_ENDPOINT` | Cloud Hypervisor endpoint | `unix:///tmp/ch.sock` |

### OpenTelemetry Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `UNKEY_CHCP_OTEL_ENABLED` | Enable OpenTelemetry | `false` |
| `UNKEY_CHCP_OTEL_SERVICE_NAME` | Service name for telemetry | `cloud-hypervisor-controlplane` |
| `UNKEY_CHCP_OTEL_SERVICE_VERSION` | Service version | `0.0.1` |
| `UNKEY_CHCP_OTEL_SAMPLING_RATE` | Trace sampling rate (0.0-1.0) | `1.0` |
| `UNKEY_CHCP_OTEL_ENDPOINT` | OTLP endpoint | `localhost:4318` |
| `UNKEY_CHCP_OTEL_PROMETHEUS_ENABLED` | Enable Prometheus metrics | `true` |
| `UNKEY_CHCP_OTEL_PROMETHEUS_PORT` | Prometheus metrics port | `9464` |

## Observability

CHCP includes comprehensive observability with OpenTelemetry:

### Start the Observability Stack

```bash
# Start Grafana LGTM stack (requires Podman)
make o11y

# Access Grafana at http://localhost:3000 (admin/admin)
# Stop the stack
make o11y-stop
```

### Trace Sampling

CHCP uses parent-based sampling with configurable ratios:
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
- `/cloudhypervisor.v1.VmService/CreateVm`
- `/cloudhypervisor.v1.VmService/DeleteVm`
- `/cloudhypervisor.v1.VmService/BootVm`
- `/cloudhypervisor.v1.VmService/ShutdownVm`
- `/cloudhypervisor.v1.VmService/PauseVm`
- `/cloudhypervisor.v1.VmService/ResumeVm`
- `/cloudhypervisor.v1.VmService/RebootVm`
- `/cloudhypervisor.v1.VmService/GetVmInfo`

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
