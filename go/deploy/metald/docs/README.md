# Metald Documentation

Welcome to the metald service documentation. Metald is the virtual machine provisioning control plane that provides unified VM management across different hypervisor backends with comprehensive lifecycle management, billing integration, and asset management.

## Documentation Navigation

### [API Documentation](api/README.md)
Complete reference for all VM lifecycle RPCs:
- Service endpoints and methods
- Request/response schemas and examples
- VM configuration and state management
- Error handling patterns
- Integration examples

### [Architecture Guide](architecture/README.md)
Deep dive into the service design:
- System architecture and backend abstractions
- Service dependencies and interaction patterns
- Database schema and state management
- Network isolation and security model
- Asset lifecycle integration

### [Operations Manual](operations/README.md)
Production deployment and management:
- Installation and configuration
- Monitoring and metrics collection
- Health checks and observability
- Security configuration (SPIFFE/mTLS)
- Performance tuning and troubleshooting

### [Development Setup](development/README.md)
Getting started with development:
- Build instructions and dependencies
- Local development environment
- Testing strategies and test setup
- Backend implementation patterns
- Contributing guidelines

## Quick Links

- [Service Overview](../) - Main README with key features
- [VM Proto Definition](../proto/vmprovisioner/v1/vm.proto) - Protocol buffer definitions
- [Configuration Reference](../internal/config/config.go) - Environment variables and settings

## Service Role

Metald is one of the four pillar services in Unkey Deploy, responsible for:
- **VM Lifecycle Management** - Create, boot, shutdown, pause, resume, and delete VMs
- **Multi-Backend Support** - Unified interface for Firecracker (CloudHypervisor planned)
- **Resource Isolation** - Secure VM execution with jailer integration
- **Billing Integration** - Real-time usage metrics collection and reporting
- **Asset Management** - Kernel, rootfs, and storage asset preparation

## Service Dependencies

### Core Dependencies
- **[assetmanagerd](../../assetmanagerd/docs/README.md)** - Asset management and distribution
- **[billaged](../../billaged/docs/README.md)** - Usage metrics aggregation and billing
- **SPIFFE/Spire** - mTLS authentication and authorization
- **SQLite** - VM state persistence and customer isolation

### Backend Dependencies
- **Firecracker** - Lightweight VM execution (primary backend)
- **Linux Networking** - TAP devices, bridges, and traffic control
- **Filesystem Isolation** - Jailer chroot environments and file permissions

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client Apps   │    │   assetmanagerd │    │    billaged     │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ ConnectRPC/gRPC       │ Asset APIs           │ Metrics APIs
         │                       │                       │
┌─────────────────────────────────────────────────────────────────┐
│                          metald                                 │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ VM Service    │  │ Auth Service │  │ Billing Collector    │ │
│  │ (ConnectRPC)  │  │ (SPIFFE)     │  │ (Background)         │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Backend       │  │ Database     │  │ Network Manager      │ │
│  │ (Firecracker) │  │ (SQLite)     │  │ (TAP/Bridge)         │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │                       │                       │
         │ Hypervisor APIs       │ File I/O             │ Network APIs
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Firecracker   │    │   Filesystem    │    │ Linux Network   │
│     Process     │    │   (Jailer)      │    │   Stack         │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Getting Started

1. **Quick Start**: See the [main README](../) for installation and basic usage
2. **API Integration**: Check the [API documentation](api/README.md) for RPC methods
3. **System Design**: Review [architecture docs](architecture/README.md) for service design
4. **Production Deploy**: Follow the [operations guide](operations/README.md)
5. **Contributing**: Read the [development guide](development/README.md)

## Key Features

- **Unified VM API**: Single interface for all VM lifecycle operations
- **Multi-Backend Support**: Pluggable hypervisor backends (Firecracker primary)
- **Security First**: Mandatory SPIFFE/mTLS and process isolation
- **Production Ready**: Comprehensive observability, metrics, and health checks
- **Billing Integrated**: Real-time usage tracking and customer isolation
- **Asset Automated**: Seamless integration with build and asset management

## Quick Start

### Installation

```bash
# Build from source
cd metald
make build

# Install with systemd
sudo make install
```

### Basic Configuration

```bash
# Minimal configuration for development
export UNKEY_METALD_PORT=8080
export UNKEY_METALD_ADDRESS=0.0.0.0
export UNKEY_METALD_BILLING_MOCK_MODE=true
export UNKEY_METALD_ASSETMANAGER_ENABLED=false
export UNKEY_METALD_JAILER_UID=$(id -u)
export UNKEY_METALD_JAILER_GID=$(id -g)
export UNKEY_METALD_TLS_MODE=disabled

./build/metald
```

### Create Your First VM

```bash
# Using the example client
cd contrib/example-client
go run main.go

# Or via direct ConnectRPC call
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 2},
      "memory": {"size_bytes": 1073741824},
      "boot": {
        "kernel_path": "/opt/vm-assets/vmlinux",
        "kernel_args": "console=ttyS0 reboot=k panic=1"
      },
      "storage": [{
        "id": "rootfs",
        "path": "/opt/vm-assets/rootfs.ext4",
        "is_root_device": true
      }]
    },
    "customer_id": "test-customer"
  }'
```

## API Highlights

The service exposes a ConnectRPC API with the following main operations:

- [`CreateVm`](../proto/vmprovisioner/v1/vm.proto#L9-10) - Create a new VM with specified configuration
- [`BootVm`](../proto/vmprovisioner/v1/vm.proto#L15-16) - Start a created VM
- [`ShutdownVm`](../proto/vmprovisioner/v1/vm.proto#L18-19) - Gracefully stop a running VM
- [`DeleteVm`](../proto/vmprovisioner/v1/vm.proto#L12-13) - Remove a VM and cleanup resources
- [`GetVmInfo`](../proto/vmprovisioner/v1/vm.proto#L30-31) - Get VM status and metrics
- [`ListVms`](../proto/vmprovisioner/v1/vm.proto#L33-34) - List VMs with filtering and pagination

See [API Documentation](./api/README.md) for complete reference.

## Production Deployment

### System Requirements

- **OS**: Linux with KVM support and firecracker binary
- **CPU**: 4+ cores recommended for multi-VM workloads
- **Memory**: 8GB+ for running multiple VMs
- **Storage**: SSD with 100GB+ free space for VM assets
- **Network**: CAP_NET_ADMIN capability for TAP devices and bridge management

### Security Considerations

1. **Jailer Configuration**: Always run with jailer enabled in production ([config.go:243](../internal/config/config.go#L243))
2. **TLS/mTLS**: Enable SPIFFE for service-to-service authentication ([config.go:356](../internal/config/config.go#L356))
3. **Customer Isolation**: Enforced at API and resource levels ([service/auth.go](../internal/service/auth.go))
4. **Resource Limits**: Configure appropriate VM quotas per customer

### Key Configuration

```bash
# Production environment variables
export UNKEY_METALD_TLS_MODE=spiffe                    # Enable mTLS
export UNKEY_METALD_OTEL_ENABLED=true                  # Enable observability
export UNKEY_METALD_BILLING_ENABLED=true               # Enable billing integration
export UNKEY_METALD_ASSETMANAGER_ENABLED=true          # Enable asset management
export UNKEY_METALD_JAILER_UID=1000                    # Jailer process isolation
export UNKEY_METALD_JAILER_GID=1000                    # Jailer group isolation
export UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer      # Chroot base directory
```

## Monitoring

Key metrics to monitor in production:

- `vm_operation_duration_seconds` - Operation latency by type
- `vm_create_total` - VM creation rate and failures
- `billing_metrics_sent_total` - Billing integration health
- `firecracker_vm_error_total` - Backend failures and error types

Source: [observability/metrics.go](../internal/observability/metrics.go)

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

# Specific backend tests
go test ./internal/backend/firecracker/...

# Service layer tests
go test ./internal/service/...
```

See [Development Setup](./development/README.md) for detailed instructions.

## Version

Current version: **v0.3.0** (Enhanced version management with automatic asset building)

Source: [cmd/api/main.go:39](../cmd/api/main.go#L39)