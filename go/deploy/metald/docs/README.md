# Metald Service Documentation

## Service Description

Metald is a high-performance VM lifecycle management service that orchestrates Firecracker microVMs with integrated security isolation, real-time billing integration, and IPv6-first networking.

### Business Purpose

Metald enables secure, multi-tenant VM hosting with:
- Sub-second VM startup times using Firecracker
- Real-time usage tracking for accurate billing
- Strong security isolation through integrated jailer
- Production-grade IPv6 networking with dual-stack support
- Comprehensive observability and monitoring

## Quick Start

### Prerequisites

- Linux system with KVM support
- Firecracker binary installed at `/usr/local/bin/firecracker`
- Go 1.21+ for building from source
- systemd for production deployment
- Root access or appropriate capabilities for VM management

### Basic Setup

```bash
# Clone the repository
git clone https://github.com/unkeyed/unkey
cd unkey/go/deploy/metald

# Build the service
make build

# Install with systemd (sets required capabilities)
sudo make install

# Start the service
sudo systemctl start metald

# Check service status
sudo systemctl status metald
```

### Development Mode

For local development without systemd:

```bash
# Build and run directly
make build
sudo ./build/metald

# Or with custom configuration
UNKEY_METALD_PORT=8080 \
UNKEY_METALD_LOG_LEVEL=debug \
sudo ./build/metald
```

## Dependencies

### Required Services

- **AssetManager** (Optional): Manages VM kernels and root filesystems
  - Default endpoint: `http://localhost:8083`
  - Can be disabled for development

- **Billaged** (Optional): Real-time billing and metrics collection
  - Default endpoint: `http://localhost:8081`
  - Can run in mock mode for development

### System Dependencies

- **Firecracker**: MicroVM hypervisor
- **KVM**: Kernel-based Virtual Machine support
- **Linux capabilities**: CAP_SYS_ADMIN, CAP_NET_ADMIN, etc.

## Basic Configuration

Key environment variables:

```bash
# Server configuration
UNKEY_METALD_PORT=8080
UNKEY_METALD_ADDRESS=0.0.0.0

# Backend selection (only firecracker supported)
UNKEY_METALD_BACKEND=firecracker

# Integrated jailer configuration
UNKEY_METALD_JAILER_UID=977
UNKEY_METALD_JAILER_GID=976
UNKEY_METALD_JAILER_CHROOT_DIR=/srv/jailer

# Service integration
UNKEY_METALD_BILLING_ENABLED=true
UNKEY_METALD_BILLING_ENDPOINT=http://localhost:8081
UNKEY_METALD_ASSETMANAGER_ENABLED=true
UNKEY_METALD_ASSETMANAGER_ENDPOINT=http://localhost:8083

# Observability
UNKEY_METALD_OTEL_ENABLED=true
UNKEY_METALD_OTEL_ENDPOINT=localhost:4318
```

## Documentation Index

### Core Documentation

- **[API Reference](api-reference.md)** - Complete API documentation with endpoints, schemas, and examples
- **[Architecture](architecture.md)** - System design, components, and technical decisions
- **[Diagrams](diagrams.md)** - Visual documentation including architecture and flow diagrams
- **[Operations](operations.md)** - Deployment, monitoring, and troubleshooting procedures

### Specialized Guides

- **[IPv6 Networking](ipv6-networking-implementation.md)** - Comprehensive IPv6 implementation guide
- **[IPv6 API Reference](ipv6-api-reference.md)** - IPv6-specific API documentation
- **[IPv6 Deployment](ipv6-deployment-guide.md)** - Production IPv6 deployment procedures
- **[IPv6 Examples](ipv6-examples-troubleshooting.md)** - Common scenarios and troubleshooting

### Architecture Decisions

- **[ADR-001: Integrated Jailer](adr/001-integrated-jailer.md)** - Why we integrate jailer functionality
- **[ADR-002: Firecracker Only](adr/002-firecracker-only-backend.md)** - Single backend support rationale

### Development Resources

- **[Authentication Guide](development/authentication.md)** - Development auth and production considerations
- **[Integrated Jailer Details](../internal/jailer/README.md)** - Technical details of jailer implementation

## Version Information

- **Current Version**: 0.2.0
- **Last Updated**: 2024-01-18
- **API Version**: v1 (ConnectRPC)

## Getting Help

### Common Issues

1. **Permission Denied**: Ensure metald has required capabilities (run `make install`)
2. **VM Creation Failed**: Check firecracker binary exists and KVM is available
3. **Network Issues**: Verify CAP_NET_ADMIN capability and namespace permissions

### Support Channels

- GitHub Issues: [github.com/unkeyed/unkey/issues](https://github.com/unkeyed/unkey/issues)
- Documentation: This directory
- Logs: `journalctl -u metald -f`

## Next Steps

1. Read the [API Reference](api-reference.md) to understand available endpoints
2. Review the [Architecture](architecture.md) for system design details
3. Follow [Operations](operations.md) for production deployment
4. Check [IPv6 guides](ipv6-networking-implementation.md) for networking setup