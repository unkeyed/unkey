# Metald - VM Management Service

High-performance VM lifecycle management with integrated security isolation and real-time billing.

## Overview

Metald manages Firecracker microVMs with:
- **Integrated jailer** for security isolation (no external jailer binary needed)
- **Real-time billing** integration with 100ms precision
- **IPv6-first networking** with multi-tenant isolation
- **Production-ready** process management and observability

## Quick Start

```bash
# Build
make build

# Install with systemd
make install

# Run development server
./build/metald
```

## Documentation

- 📖 **[Development Guide](DEVELOPMENT.md)** - Start here for local development
- 🏗️ **[Architecture Overview](docs/architecture/overview.md)** - System design and components
- 🔒 **[Integrated Jailer](internal/jailer/README.md)** - Security isolation implementation
- 🔑 **[Authentication Guide](docs/development/authentication.md)** - Auth system overview
- 📋 **[Architecture Decisions](docs/adr/)** - Key design decisions

## Key Features

### Integrated Jailer
Unlike standard Firecracker deployments, metald includes jailer functionality directly in the binary. This solves network namespace issues and provides better control. See [ADR-001](docs/adr/001-integrated-jailer.md).

### Single Backend Support
Only Firecracker is supported. CloudHypervisor code exists but is incomplete. See [ADR-002](docs/adr/002-firecracker-only-backend.md).

### Development Authentication
Uses mock tokens (`Bearer dev_customer_123`) for development. Must be replaced in production. See the [Authentication Guide](docs/development/authentication.md).

## API Example

```bash
# Create a VM
curl -X POST http://localhost:8080/v1/vms \
  -H "Authorization: Bearer dev_customer_test" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "cpu": {"vcpu_count": 2},
      "memory": {"size_bytes": 1073741824},
      "kernel": {"image_id": "vmlinux-5.10"},
      "rootfs": {"image_id": "alpine-rootfs"}
    }
  }'

# List VMs
curl http://localhost:8080/v1/vms \
  -H "Authorization: Bearer dev_customer_test"
```

## Requirements

- Linux with KVM support
- Firecracker binary installed
- Go 1.21+ (for building)
- systemd (for production deployment)

## Security

Metald requires specific capabilities instead of running as root:
- `CAP_SYS_ADMIN` - Namespace operations
- `CAP_NET_ADMIN` - Network device creation
- `CAP_SYS_CHROOT` - Jail creation
- Additional capabilities for privilege dropping

The `make install` command sets these automatically.

## Contributing

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and guidelines.

## License

[Your License Here]