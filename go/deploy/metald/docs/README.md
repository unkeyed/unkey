# Metald Documentation

Welcome to the metald service documentation. Metald is the Virtual Machine Manager (VMM) control plane that provides unified VM lifecycle management across multiple hypervisor backends, with multi-tenant isolation and comprehensive resource management.

## Documentation Navigation

### [API Documentation](api/README.md)
Complete reference for all VmService RPCs:
- Service endpoints and methods
- Request/response schemas with examples
- VM lifecycle operations (create, boot, shutdown, delete)
- Resource configuration and networking
- Multi-tenant authentication and authorization

### [Architecture Guide](architecture/README.md)
Deep dive into the service design:
- VM lifecycle state management and backend abstraction
- Integration with assetmanagerd, builderd, and billaged services
- SPIFFE/SPIRE mTLS authentication and multi-tenant isolation
- Network management and port allocation
- Database state consistency and reconciliation

### [Operations Manual](operations/README.md)
Production deployment and management:
- Configuration and environment variables
- Monitoring and metrics collection via OpenTelemetry
- Health checks and system observability
- Performance tuning and resource management
- Troubleshooting and debugging guides

### [Development Setup](development/README.md)
Getting started with development:
- Build instructions and dependencies
- Local development environment setup
- Testing strategies with mock backends
- Client library usage examples
- Contributing guidelines and code patterns

## Quick Links

- [Service Overview](../) - Main README with key features
- [Proto Definition](../proto/vmprovisioner/v1/vm.proto) - Protocol buffer definitions
- [Configuration Reference](../internal/config/config.go) - Environment variables and settings
- [Client Library](../client/README.md) - Go client library documentation

## Service Role

Metald is one of the four pillar services in Unkey Deploy, responsible for:
- **VM Lifecycle Management** - Create, boot, shutdown, pause, resume, and delete operations
- **Multi-Hypervisor Support** - Unified API across Firecracker and Cloud Hypervisor backends
- **Resource Management** - CPU, memory, storage, and network configuration
- **Multi-Tenant Isolation** - Secure customer separation with authentication and authorization
- **State Consistency** - Database persistence with automatic reconciliation
- **Billing Integration** - Real-time metrics collection for accurate usage tracking

## Service Dependencies

### Core Dependencies
- **[assetmanagerd](../../assetmanagerd/docs/README.md)** - VM asset management (kernels, rootfs images) with automatic build triggering
- **[billaged](../../billaged/docs/README.md)** - VM usage metrics collection and billing aggregation
- **SPIFFE/Spire** - mTLS authentication and service authorization
- **SQLite** - VM state persistence and customer isolation

### Optional Dependencies
- **[builderd](../../builderd/docs/README.md)** - Automatic rootfs building via assetmanagerd integration
- **OpenTelemetry** - Observability, metrics export, and distributed tracing
- **Prometheus** - Metrics collection and monitoring

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client Apps   │    │  assetmanagerd  │    │    billaged     │
│  (metald-cli)   │    │   (Assets)      │    │   (Billing)     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ ConnectRPC/gRPC       │ Asset APIs           │ Metrics/Events
         │                       │                       │
┌─────────────────────────────────────────────────────────────────┐
│                          metald                                 │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ VM Service    │  │ Auth Service │  │ Billing Integration  │ │
│  │ (ConnectRPC)  │  │ (SPIFFE)     │  │ (Metrics Collector)  │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ VM Repository │  │ Network      │  │ Observability        │ │
│  │ (SQLite)      │  │ Manager      │  │ (OpenTelemetry)      │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Firecracker   │  │ VM           │  │ State Reconciler     │ │
│  │ Backend       │  │ Reconciler   │  │ (Background)         │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │                       │                       │
         │ VM Processes          │ Network Setup        │ File I/O
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Firecracker   │    │   Linux         │    │   SQLite        │
│   Hypervisor    │    │   Networking    │    │   Database      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Getting Started

### Installation

```bash
# Build from source
cd metald
make build

# Install with systemd (requires root for networking)
sudo make install
```

### Basic Configuration

```bash
# Minimal configuration for development
export UNKEY_METALD_PORT=8080
export UNKEY_METALD_ADDRESS=0.0.0.0
export UNKEY_METALD_BACKEND=firecracker
export UNKEY_METALD_TLS_MODE=disabled
export UNKEY_METALD_OTEL_ENABLED=false
export UNKEY_METALD_BILLING_ENABLED=false
export UNKEY_METALD_ASSETMANAGER_ENABLED=false

sudo ./build/metald
```

### Create Your First VM

```bash
# Using the CLI tool
cd client/cmd/metald-cli
go run main.go create-vm --config ../examples/configs/minimal.json

# Or using the client library
cd client/examples
go run basic_usage.go
```

## Key Features

- **Unified VM API**: Single interface across multiple hypervisor backends
- **Multi-Tenant Security**: Customer isolation with SPIFFE/SPIRE authentication
- **Production Ready**: Comprehensive observability, metrics, and health monitoring
- **Asset Integration**: Automatic VM asset preparation via assetmanagerd
- **Real-time Billing**: VM usage metrics collection for accurate cost tracking
- **State Consistency**: Database persistence with automatic reconciliation
- **Network Management**: IPv4/IPv6 support with port forwarding and rate limiting

## Quick Configuration Examples

### Production Environment

```bash
# Production environment variables
export UNKEY_METALD_TLS_MODE=spiffe                    # Enable mTLS
export UNKEY_METALD_OTEL_ENABLED=true                  # Enable observability
export UNKEY_METALD_BILLING_ENABLED=true               # Enable billing
export UNKEY_METALD_ASSETMANAGER_ENABLED=true          # Enable assets
export UNKEY_METALD_OTEL_PROMETHEUS_ENABLED=true       # Metrics export
export UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED=false # Limit cardinality
```

### Development Environment

```bash
# Development environment variables  
export UNKEY_METALD_TLS_MODE=disabled                  # Disable TLS
export UNKEY_METALD_BILLING_MOCK_MODE=true             # Mock billing
export UNKEY_METALD_OTEL_ENABLED=false                 # Disable telemetry
export UNKEY_METALD_DATA_DIR=/tmp/metald                # Temp database
```

## API Highlights

The service exposes a ConnectRPC API with the following main operations:

- [`CreateVm`](../proto/vmprovisioner/v1/vm.proto#L9) - Create new VM instances with resource configuration
- [`BootVm`](../proto/vmprovisioner/v1/vm.proto#L15) - Start created VMs and begin billing collection
- [`ShutdownVm`](../proto/vmprovisioner/v1/vm.proto#L18) - Gracefully stop running VMs
- [`DeleteVm`](../proto/vmprovisioner/v1/vm.proto#L12) - Remove VM instances and cleanup resources
- [`GetVmInfo`](../proto/vmprovisioner/v1/vm.proto#L30) - Retrieve VM status, configuration, and metrics
- [`ListVms`](../proto/vmprovisioner/v1/vm.proto#L33) - List customer VMs with filtering support

See [API Documentation](./api/README.md) for complete reference with examples.

## Production Deployment

### System Requirements

- **OS**: Linux with systemd support and root privileges (for networking)
- **CPU**: 4+ cores for concurrent VM management
- **Memory**: 8GB+ for VM state management and network allocation
- **Storage**: 50GB+ for VM database and asset caching
- **Network**: Low-latency connection to assetmanagerd and billaged

### Key Configuration

```bash
# Critical production settings
export UNKEY_METALD_JAILER_UID=1000                    # Jailer user
export UNKEY_METALD_JAILER_GID=1000                    # Jailer group  
export UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer       # Isolation directory
export UNKEY_METALD_DATA_DIR=/opt/metald/data           # Database location
export UNKEY_METALD_NETWORK_ENABLED=true               # Enable networking
export UNKEY_METALD_NETWORK_HOST_PROTECTION=true       # Protect host routes
```

## Monitoring

Key metrics to monitor in production:

- `metald_vm_operations_total` - VM operation counts by type and result
- `metald_vm_operation_duration_seconds` - VM operation performance histograms
- `metald_active_vms` - Current VM count by state and customer
- `metald_backend_errors_total` - Backend failure rates by error type
- `metald_billing_metrics_collected_total` - Billing data collection rate

Source: [internal/observability/metrics.go](../internal/observability/metrics.go)

See [Operations Guide](./operations/README.md) for complete monitoring setup.

## Development

### Building from Source

```bash
git clone https://github.com/unkeyed/unkey
cd go/deploy/metald
make build  # Builds to build/metald
```

### Running Tests

```bash
# Unit tests
make test

# Service integration tests
go test ./internal/service/...

# Backend tests
go test ./internal/backend/...
```

See [Development Setup](./development/README.md) for detailed instructions.

## Security Considerations

- **Root Privileges**: Required for Linux networking operations (TAP devices, bridge management)
- **Jailer Isolation**: Firecracker VMs run in isolated chroot environments with dedicated UIDs
- **mTLS Authentication**: SPIFFE/SPIRE provides service-to-service authentication
- **Customer Isolation**: Database-level tenant separation with authentication validation

## Version

Current version: **Enhanced VMM control plane with multi-tenant support and comprehensive observability**

Source: [cmd/metald/main.go:40](../cmd/metald/main.go#L40)