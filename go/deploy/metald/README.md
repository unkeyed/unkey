# Metald - VM Lifecycle Management Service

High-performance VM lifecycle management with integrated security isolation and real-time billing.

## Overview

Metald is the central control plane for virtual machine lifecycle management in the Unkey Deploy platform. It provides a unified API for creating, managing, and monitoring microVMs using Firecracker.

**Key Features:**
- **Integrated jailer** for security isolation (no external jailer binary needed)
- **Real-time billing** integration with 100ms precision  
- **Dual-stack networking** with IPv4/IPv6 support and multi-tenant isolation
- **Asset management** integration for dynamic VM image distribution
- **Production-ready** with comprehensive observability and monitoring

## Documentation

For comprehensive documentation, see [**📚 Full Documentation**](./docs/README.md)

**Quick Links:**
- [API Reference](./docs/api/README.md) - Complete API documentation with examples
- [Architecture Guide](./docs/architecture/README.md) - System design and service interactions  
- [Operations Manual](./docs/operations/README.md) - Production deployment and monitoring
- [Development Setup](./docs/development/README.md) - Build instructions and contributing guide

## Quick Start

```bash
# Build from source
make build

# Install with systemd
sudo make install

# Run development server
export UNKEY_METALD_BILLING_MOCK_MODE=true
export UNKEY_METALD_ASSETMANAGER_ENABLED=false
./build/metald
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
      }
    }
  }'
```

## Service Dependencies

Metald integrates with other Unkey Deploy services:
- **[assetmanagerd](../assetmanagerd/docs/README.md)** - VM asset preparation and distribution
- **[billaged](../billaged/docs/README.md)** - Usage tracking and billing
- **builderd** - Indirect integration through assetmanagerd

## Requirements

- Linux with KVM support
- Firecracker binary installed
- Go 1.24+ (for building)
- systemd (for production deployment)

## Security

Metald runs as root to manage network namespaces, interfaces, and iptables operations. This is acceptable as metald is designed to be the sole application on dedicated VM hosts. The integrated jailer still drops privileges to specified UID/GID for individual VM processes, ensuring proper isolation.

The `make install` command configures the service with appropriate permissions automatically.

## Contributing

See [Development Setup](./docs/development/README.md) for contribution guidelines.

## Version

v0.2.0 (Integrated Jailer)