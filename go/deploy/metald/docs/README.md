# Metald - VM Lifecycle Management Service

Metald is the central control plane for virtual machine lifecycle management in the Unkey Deploy platform. It provides a unified API for creating, managing, and monitoring microVMs using Firecracker or Cloud Hypervisor backends.

## Quick Links

- [API Documentation](./api/README.md) - Complete API reference with examples
- [Architecture & Dependencies](./architecture/README.md) - Service interactions and system design
- [Operations Guide](./operations/README.md) - Metrics, monitoring, and production deployment
- [Development Setup](./development/README.md) - Build, test, and local development

## Service Overview

**Purpose**: Centralized virtual machine lifecycle management with integrated security isolation and multi-tenant resource control.

### Key Features

- **Unified VM Management**: Single API for VM lifecycle operations across hypervisor backends
- **Security Isolation**: Integrated jailer with chroot, cgroups, and network namespace isolation
- **Multi-tenant Support**: Customer-level isolation and resource quotas
- **Dual-stack Networking**: IPv4/IPv6 support with TAP device management
- **Usage Tracking**: Integration with billing service for resource accounting
- **Asset Management**: Dynamic VM image distribution (kernels, rootfs)
- **High Observability**: OpenTelemetry tracing, Prometheus metrics, structured logging

### Dependencies

- [assetmanagerd](../../assetmanagerd/docs/README.md) - VM image storage and distribution
- [billaged](../../billaged/docs/README.md) - Usage tracking and billing integration

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
export UNKEY_METALD_SERVER_PORT=8080
export UNKEY_METALD_BILLING_MOCK_MODE=true
export UNKEY_METALD_ASSETMANAGER_ENABLED=false
export UNKEY_METALD_JAILER_UID=$(id -u)
export UNKEY_METALD_JAILER_GID=$(id -g)

./metald
```

### Create Your First VM

```bash
# Using the example client
cd contrib/example-client
go run main.go -action create-and-boot

# Or via direct API call
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test123" \
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
    }
  }'
```

## Architecture Overview

```mermaid
graph TB
    subgraph "API Layer"
        API[ConnectRPC API<br/>:8080]
        AUTH[Auth Middleware]
    end
    
    subgraph "Core Services"
        SVC[VM Service]
        DB[(SQLite DB)]
        METRICS[Metrics Collector]
    end
    
    subgraph "Backends"
        FC[Firecracker Backend]
        CH[Cloud Hypervisor<br/>Backend]
    end
    
    subgraph "External Services"
        ASSET[AssetManagerd]
        BILL[Billaged]
    end
    
    API --> AUTH --> SVC
    SVC --> DB
    SVC --> METRICS
    SVC --> FC
    SVC --> CH
    SVC -.-> ASSET
    METRICS --> BILL
```

## Production Deployment

### System Requirements

- **OS**: Linux with KVM support
- **CPU**: 4+ cores recommended
- **Memory**: 8GB+ for running multiple VMs
- **Storage**: SSD with 100GB+ free space
- **Network**: CAP_NET_ADMIN capability for TAP devices

### Security Considerations

1. **Jailer Configuration**: Always run with jailer enabled in production
2. **TLS/mTLS**: Enable SPIFFE for service-to-service authentication
3. **Customer Isolation**: Enforced at API and resource levels
4. **Resource Limits**: Configure appropriate VM quotas per customer

### High Availability

While metald itself is stateless (using SQLite for persistence), consider:
- Running multiple instances behind a load balancer
- Regular SQLite backups
- Monitoring backend health and failover

## API Highlights

The service exposes a ConnectRPC API with the following main operations:

- `CreateVm` - Create a new VM with specified configuration
- `BootVm` - Start a created VM
- `ShutdownVm` - Gracefully stop a running VM
- `DeleteVm` - Remove a VM and cleanup resources
- `GetVmInfo` - Get VM status and metrics
- `ListVms` - List VMs with filtering and pagination

See [API Documentation](./api/README.md) for complete reference.

## Monitoring

Key metrics to monitor in production:

- `vm_operation_duration_seconds` - Operation latency
- `vm_create_total` - VM creation rate and failures
- `billing_metrics_sent_total` - Billing integration health
- `firecracker_vm_error_total` - Backend failures

See [Operations Guide](./operations/README.md) for complete monitoring setup.

## Development

### Building from Source

```bash
git clone https://github.com/unkeyed/unkey
cd go/deploy/metald
make test
make build
```

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires Docker)
make test-integration

# Benchmark tests
make bench
```

See [Development Setup](./development/README.md) for detailed instructions.

## Support

- **Issues**: [GitHub Issues](https://github.com/unkeyed/unkey/issues)
- **Documentation**: [Full Documentation](./README.md)
- **Version**: v0.2.0 (Integrated Jailer)