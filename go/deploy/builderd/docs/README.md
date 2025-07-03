# Builderd Documentation

Welcome to the builderd service documentation. Builderd is a multi-tenant build service that processes various source types (Docker images, Git repositories, archives) and produces optimized rootfs images for microVM execution.

## Documentation Navigation

### [API Documentation](api/README.md)
Complete reference for all BuilderService RPCs, including:
- Service endpoints and methods  
- Request/response schemas with examples
- Build configuration and source types
- Streaming operations and error handling
- Multi-tenant authentication patterns

### [Architecture Guide](architecture/README.md)
Deep dive into the service design:
- Build execution pipeline and executor registry
- Integration with assetmanagerd and microVM infrastructure
- SPIFFE/SPIRE mTLS authentication
- Multi-tenant isolation and resource quotas
- Service interaction patterns

### [Operations Manual](operations/README.md)
Production deployment and management:
- Configuration and environment variables
- Monitoring and metrics collection
- Health checks and observability  
- Performance tuning and resource management
- Storage backend configuration

### [Development Setup](development/README.md)
Getting started with development:
- Build instructions and dependencies
- Local development environment
- Testing strategies and executor patterns
- Contributing guidelines and code patterns

## Quick Links

- [Service Overview](../) - Main README with key features
- [Proto Definition](../proto/builder/v1/builder.proto) - Protocol buffer definitions
- [Configuration Reference](../internal/config/config.go) - Environment variables and settings

## Service Role

Builderd is one of the four pillar services in Unkey Deploy, responsible for:
- **Multi-source Building** - Docker images, Git repositories, and archive extraction
- **Rootfs Generation** - Optimized ext4 filesystem images for microVMs
- **Asset Integration** - Automatic registration with assetmanagerd for VM creation
- **Tenant Isolation** - Secure multi-tenant build execution with resource quotas

## Service Dependencies

### Core Dependencies
- **[assetmanagerd](../../assetmanagerd/docs/README.md)** - Asset storage and registration for built artifacts
- **SPIFFE/Spire** - mTLS authentication and authorization for service communications
- **Docker** - Container runtime for Docker image extraction and building

### Integration Points
- **metald** - Consumes built rootfs assets for microVM creation
- **billaged** - // AIDEV: billaged documentation needed for complete interaction description

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client Apps   │    │   assetmanagerd │    │    metald       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ ConnectRPC/gRPC       │ Asset APIs           │ Consumes Assets
         │                       │                       │
┌─────────────────────────────────────────────────────────────────┐
│                          builderd                               │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Builder       │  │ Auth Service │  │ Asset Integration    │ │
│  │ Service       │  │ (SPIFFE)     │  │ (assetmanagerd)      │ │
│  │ (ConnectRPC)  │  └──────────────┘  └──────────────────────┘ │
│  └───────────────┘  ┌──────────────┐  ┌──────────────────────┐ │
│  ┌───────────────┐  │ Executor     │  │ Observability        │ │
│  │ Config        │  │ Registry     │  │ (OpenTelemetry)      │ │
│  │ Management    │  │ (Docker)     │  │                      │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │                       │                       │
         │ Docker APIs           │ File I/O             │ Network APIs  
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Docker        │    │   Filesystem    │    │ External        │
│   Runtime       │    │   Storage       │    │ Registries      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Getting Started

1. **Quick Start**: See the [main README](../) for installation and basic usage
2. **API Integration**: Check the [API documentation](api/README.md) for RPC methods
3. **System Design**: Review [architecture docs](architecture/README.md) for service design
4. **Production Deploy**: Follow the [operations guide](operations/README.md)
5. **Contributing**: Read the [development guide](development/README.md)

## Key Features

- **Multi-Source Support**: Docker images, Git repositories, and archives
- **MicroVM Optimized**: Produces ext4 rootfs images optimized for Firecracker
- **Multi-Tenant**: Secure tenant isolation with resource quotas and limits
- **Asset Integrated**: Automatic registration with assetmanagerd for seamless VM creation
- **Production Ready**: Comprehensive observability, metrics, and health checks
- **Highly Observable**: OpenTelemetry tracing and Prometheus metrics

## Build Pipeline Overview

1. **Source Acquisition** - Pull Docker images, clone Git repos, or download archives
2. **Extraction** - Extract filesystem contents using appropriate strategy
3. **Optimization** - Remove unnecessary files and optimize for microVM execution
4. **Rootfs Creation** - Generate ext4 filesystem image for Firecracker VMs
5. **Asset Registration** - Upload and register with assetmanagerd for VM provisioning

## Quick Configuration

```bash
# Minimal configuration for development
export UNKEY_BUILDERD_PORT=8082
export UNKEY_BUILDERD_ADDRESS=0.0.0.0
export UNKEY_BUILDERD_ASSETMANAGER_ENABLED=true
export UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT=https://localhost:8083
export UNKEY_BUILDERD_TLS_MODE=spiffe

./build/builderd
```

## Version

Current version: Enhanced multi-tenant build service with assetmanagerd integration

Source: [cmd/builderd/main.go:44](../cmd/builderd/main.go#L44)