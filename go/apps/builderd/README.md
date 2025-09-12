# Builderd - Multi-Tenant Build Service

Builderd transforms various source types into optimized rootfs images for Firecracker microVM execution with comprehensive multi-tenant isolation and resource management.

## Quick Links

- [API Documentation](./docs/api/README.md) - Complete API reference with examples
- [Architecture & Dependencies](./docs/architecture/README.md) - Service design and integrations
- [Operations Guide](./docs/operations/README.md) - Production deployment and monitoring
- [Development Setup](./docs/development/README.md) - Build, test, and local development

## Service Overview

**Purpose**: Multi-tenant build execution service that processes Docker images, Git repositories, and archives to produce optimized ext4 rootfs images for microVM deployment.

**Implementation**: [BuilderService](internal/service/builder.go:23) with [DockerExecutor](internal/executor/docker.go:25) for Docker image processing and [tenant manager](internal/tenant/manager.go:14) for multi-tenant isolation.

### Key Features

- **Multi-Tenant Isolation**: Linux namespaces, cgroups, and tenant-specific resource limits
- **Docker Image Processing**: Pull, extract, and optimize Docker images to rootfs
- **Asset Registration**: Automatic registration with [assetmanagerd](../assetmanagerd/README.md) for VM deployment
- **Real-time Monitoring**: OpenTelemetry tracing, build metrics, and streaming logs
- **Resource Management**: Per-tenant quotas for CPU, memory, disk, and concurrent builds
- **Optimization**: Rootfs size reduction through layer flattening and cleanup
- **Security**: SPIFFE/mTLS authentication and sandboxed build execution

### Dependencies

- [assetmanagerd](../assetmanagerd/README.md) - Registers built artifacts for VM provisioning ([client implementation](internal/assetmanager/client.go:63))
- [metald](../metald/README.md) - Consumes registered assets for VM creation
- SPIFFE/Spire - Service authentication and mTLS ([TLS provider](cmd/builderd/main.go:147))
- Docker Engine - Image pulling and container operations ([executor implementation](internal/executor/docker.go:25))
- OpenTelemetry - Observability and metrics collection ([metrics setup](internal/observability/otel.go:1))

// AIDEV-NOTE: Documentation updated with source code references for easy navigation

## Quick Start

### Installation

```bash
# Build from source
cd builderd
make build

# Install with systemd
sudo make install
```

### Basic Configuration

```bash
# Minimal configuration for development
export UNKEY_BUILDERD_PORT=8082
export UNKEY_BUILDERD_STORAGE_BACKEND=local
export UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/opt/builderd/rootfs
export UNKEY_BUILDERD_TLS_MODE=spiffe
export UNKEY_BUILDERD_ASSETMANAGER_ENABLED=true

./builderd
```

### Create Your First Build

```bash
# Submit a Docker image build
curl -X POST http://localhost:8082/builder.v1.BuilderService/CreateBuild \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "tenant": {
        "tenant_id": "example-tenant",
        "tier": "TENANT_TIER_FREE"
      },
      "source": {
        "docker_image": {
          "image_uri": "nginx:1.21-alpine"
        }
      },
      "target": {
        "microvm_rootfs": {
          "init_strategy": "INIT_STRATEGY_TINI"
        }
      },
      "strategy": {
        "docker_extract": {}
      }
    }
  }'
```

## Overview

builderd is designed to handle the complexities of multi-tenant build execution with a focus on:

- **Multi-Tenant Isolation**: Secure build environments with resource quotas per tenant
- **Flexible Source Support**: Docker images, Git repositories, and archive formats
- **Build Optimization**: Automatic rootfs optimization for microVM deployment
- **Resource Management**: CPU, memory, disk, and time limits per tenant tier
- **Comprehensive Observability**: OpenTelemetry integration with metrics and tracing
- **High Performance**: Concurrent build execution with efficient caching

### Key Features

- **Source Types**:
  - Docker image extraction with registry authentication
  - Git repository builds (planned)
  - Archive extraction (planned)

- **Build Targets**:
  - MicroVM rootfs with init strategies (tini, direct, custom)
  - Container images (planned)
  - WebAssembly modules (planned)

- **Tenant Management**:
  - Service tiers (Free, Pro, Enterprise, Dedicated)
  - Resource quotas and limits
  - Build history and statistics
  - Cost tracking for billing integration

- **Security**:
  - SPIFFE/mTLS for service communication
  - Tenant isolation with namespaces and cgroups
  - Registry access controls
  - Build-time security scanning (planned)

## Service Endpoints

- **gRPC/ConnectRPC**: `<host>:8082/builder.v1.BuilderService/*`
- **Health Check**: `<host>:8082/health` (rate limited)
- **Prometheus Metrics**: `<host>:9466/metrics` (when enabled)

## Configuration

builderd uses environment variables following the `UNKEY_BUILDERD_*` pattern:

### Core Settings
```bash
UNKEY_BUILDERD_PORT=8082                          # Service port
UNKEY_BUILDERD_ADDRESS=0.0.0.0                    # Bind address
UNKEY_BUILDERD_SHUTDOWN_TIMEOUT=15s               # Graceful shutdown timeout
UNKEY_BUILDERD_RATE_LIMIT=100                     # Health endpoint rate limit/sec
```

### Build Configuration
```bash
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=5            # Concurrent build limit
UNKEY_BUILDERD_BUILD_TIMEOUT=15m                  # Maximum build duration
UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd          # Temporary build directory
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/opt/builderd/rootfs  # Output directory
UNKEY_BUILDERD_WORKSPACE_DIR=/opt/builderd/workspace   # Build workspace
```

### Storage Backend
```bash
UNKEY_BUILDERD_STORAGE_BACKEND=local              # Backend type: local, s3, gcs
UNKEY_BUILDERD_STORAGE_RETENTION_DAYS=30          # Artifact retention period
UNKEY_BUILDERD_STORAGE_MAX_SIZE_GB=100            # Maximum storage size
UNKEY_BUILDERD_STORAGE_CACHE_ENABLED=true         # Enable build cache
UNKEY_BUILDERD_STORAGE_CACHE_MAX_SIZE_GB=50       # Cache size limit
```

### Docker Registry
```bash
UNKEY_BUILDERD_DOCKER_REGISTRY_AUTH=true          # Enable registry authentication
UNKEY_BUILDERD_DOCKER_MAX_IMAGE_SIZE_GB=5         # Maximum image size
UNKEY_BUILDERD_DOCKER_PULL_TIMEOUT=10m            # Image pull timeout
UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR=""          # Optional registry mirror
```

### Multi-Tenancy
```bash
UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED=true      # Enable tenant isolation
UNKEY_BUILDERD_TENANT_DEFAULT_TIER=free           # Default service tier
UNKEY_BUILDERD_TENANT_QUOTA_CHECK_INTERVAL=5m     # Quota check frequency

# Default resource limits
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_MEMORY_BYTES=2147483648  # 2GB
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_CPU_CORES=2
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_DISK_BYTES=10737418240   # 10GB
UNKEY_BUILDERD_TENANT_DEFAULT_TIMEOUT_SECONDS=900          # 15min
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_CONCURRENT_BUILDS=3
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_DAILY_BUILDS=100
```

### AssetManagerd Integration
```bash
UNKEY_BUILDERD_ASSETMANAGER_ENABLED=true          # Enable asset registration
UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT=https://localhost:8083  # AssetManagerd endpoint
```

### OpenTelemetry
```bash
UNKEY_BUILDERD_OTEL_ENABLED=false                 # Enable observability
UNKEY_BUILDERD_OTEL_SERVICE_NAME=builderd         # Service identifier
UNKEY_BUILDERD_OTEL_ENDPOINT=localhost:4318       # OTLP endpoint
UNKEY_BUILDERD_OTEL_SAMPLING_RATE=1.0             # Trace sampling rate
UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED=true       # Enable metrics
UNKEY_BUILDERD_OTEL_PROMETHEUS_PORT=9466          # Metrics port
```

### TLS/SPIFFE
```bash
UNKEY_BUILDERD_TLS_MODE=spiffe                    # TLS mode: disabled, file, spiffe
UNKEY_BUILDERD_SPIFFE_SOCKET=/run/spire/sockets/agent.sock  # SPIFFE socket
```

## Integration Examples

### Creating a Build

```go
import (
    builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/builder/v1"
    "github.com/unkeyed/unkey/go/deploy/builderd/gen/builder/v1/builderv1connect"
)

// Create a Docker image build
req := &builderv1.CreateBuildRequest{
    Config: &builderv1.BuildConfig{
        Tenant: &builderv1.TenantContext{
            TenantId:   "example-tenant",
            CustomerId: "example-customer",
            Tier:       builderv1.TenantTier_TENANT_TIER_PRO,
        },
        Source: &builderv1.BuildSource{
            SourceType: &builderv1.BuildSource_DockerImage{
                DockerImage: &builderv1.DockerImageSource{
                    ImageUri: "ghcr.io/myorg/myapp:v1.0.0",
                },
            },
        },
        Target: &builderv1.BuildTarget{
            TargetType: &builderv1.BuildTarget_MicrovmRootfs{
                MicrovmRootfs: &builderv1.MicroVMRootfs{
                    InitStrategy: builderv1.InitStrategy_INIT_STRATEGY_TINI,
                },
            },
        },
        Strategy: &builderv1.BuildStrategy{
            StrategyType: &builderv1.BuildStrategy_DockerExtract{
                DockerExtract: &builderv1.DockerExtractStrategy{
                    FlattenFilesystem: true,
                },
            },
        },
    },
}

resp, err := client.CreateBuild(ctx, connect.NewRequest(req))
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Build started: %s\n", resp.Msg.BuildId)
fmt.Printf("Rootfs will be at: %s\n", resp.Msg.RootfsPath)
```

### Monitoring Build Progress

```go
// Stream build logs
stream, err := client.StreamBuildLogs(ctx, connect.NewRequest(&builderv1.StreamBuildLogsRequest{
    BuildId:  buildId,
    TenantId: tenantId,
    Follow:   true,
}))

for stream.Receive() {
    log := stream.Msg()
    fmt.Printf("[%s] %s: %s\n", log.Timestamp.AsTime(), log.Level, log.Message)
}
```

## Version

Current version: **0.1.0** ([proto definition](proto/builder/v1/builder.proto))

## Related Documentation

- [Service Pillar Overview](../docs/PILLAR_SERVICES.md)
- [Multi-Tenant Architecture](../docs/architecture/multi-tenancy.md)
- [SPIFFE/mTLS Setup](../docs/tls-implementation.md)
- [Observability Guide](../docs/telemetry-migration-guide.md)
