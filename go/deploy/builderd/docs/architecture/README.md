# Builderd Architecture Guide

This document provides a comprehensive overview of builderd's internal architecture, service interactions, and design patterns.

## System Architecture

Builderd follows a modular architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────────┐
│                          Builderd                               │
│                                                                 │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│ │   ConnectRPC    │  │   Tenant Auth   │  │   OpenTelemetry │ │
│ │   API Layer     │  │   Interceptor   │  │   Tracing       │ │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│           │                    │                    │         │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│ │   Builder       │  │   Config        │  │   Metrics       │ │
│ │   Service       │  │   Management    │  │   Collection    │ │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│           │                    │                    │         │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│ │   Executor      │  │   Asset         │  │   Tenant        │ │
│ │   Registry      │  │   Client        │  │   Manager       │ │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│           │                    │                    │         │
│ ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│ │   Docker        │  │   Storage       │  │   Build         │ │
│ │   Executor      │  │   Backend       │  │   Isolation     │ │
│ └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
          │                      │                      │
          │ Docker APIs          │ File I/O            │ SPIFFE mTLS
          │                      │                      │
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Docker        │  │   Local         │  │   Assetmanager  │
│   Runtime       │  │   Filesystem    │  │   Service       │
└─────────────────┘  └─────────────────┘  └─────────────────┘
```

## Core Components

### BuilderService

The main service component that orchestrates build operations with comprehensive lifecycle management.

**Implementation**: [internal/service/builder.go](../../internal/service/builder.go)

**Key Responsibilities**:
- Build job coordination and state management
- Tenant-scoped validation and authorization
- Asset registration with assetmanagerd integration
- Graceful shutdown coordination

**Shutdown Coordination**:
```go
func (s *BuilderService) Shutdown(ctx context.Context) error {
    s.shutdownCancel()  // Cancel all running builds
    s.buildWg.Wait()    // Wait for completion with timeout
}
```
Source: [internal/service/builder.go:617](../../internal/service/builder.go#L617)

### Executor Registry

Pluggable executor system that handles different source types with extensible patterns.

**Implementation**: [internal/executor/registry.go](../../internal/executor/registry.go)

**Architecture Pattern**:
```go
type Executor interface {
    Execute(ctx context.Context, request *builderv1.CreateBuildRequest) (*BuildResult, error)
    ExecuteWithID(ctx context.Context, request *builderv1.CreateBuildRequest, buildID string) (*BuildResult, error)
    GetSupportedSources() []string
    Cleanup(ctx context.Context, buildID string) error
}
```
Source: [internal/executor/types.go:11](../../internal/executor/types.go#L11)

**Current Executors**:
- **DockerExecutor**: Extracts Docker images to optimized rootfs
- **GitExecutor**: (Planned) Git repository builds
- **ArchiveExecutor**: (Planned) Archive extraction builds

### Docker Executor

Comprehensive Docker image processing with microVM optimization.

**Implementation**: [internal/executor/docker.go](../../internal/executor/docker.go)

**Build Pipeline**:
1. **Image Pull**: Download Docker image with timeout and authentication
2. **Container Creation**: Create container without running for filesystem access
3. **Metadata Extraction**: Extract runtime configuration (CMD, ENTRYPOINT, ENV)
4. **Filesystem Extraction**: Export container filesystem via docker export
5. **Rootfs Optimization**: Remove unnecessary files and caches
6. **Init Injection**: Inject metald-init for microVM execution
7. **Ext4 Creation**: Generate filesystem image for Firecracker VMs

**Critical Integration Points**:
```go
// Inject metald-init for VM execution
func (d *DockerExecutor) injectMetaldInit(ctx context.Context, logger *slog.Logger, rootfsDir string) error
```
Source: [internal/executor/docker.go:969](../../internal/executor/docker.go#L969)

```go
// Create container command file for metald-init
func (d *DockerExecutor) createContainerCmd(ctx context.Context, logger *slog.Logger, rootfsDir string, metadata *builderv1.ImageMetadata) error
```
Source: [internal/executor/docker.go:1027](../../internal/executor/docker.go#L1027)

## Service Dependencies

### Assetmanagerd Integration

Builderd integrates closely with assetmanagerd for artifact management and microVM provisioning.

**Client Implementation**: [internal/assetmanager/client.go](../../internal/assetmanager/client.go)

**Integration Flow**:
1. Build completion triggers automatic asset registration
2. Rootfs and kernel assets uploaded via streaming API
3. Tenant context preserved for access control
4. Asset IDs returned for metald VM creation

**Asset Registration**:
```go
assetID, err := s.assetClient.RegisterBuildArtifactWithID(
    ctx, 
    buildID, 
    rootfsPath, 
    assetv1.AssetType_ASSET_TYPE_ROOTFS, 
    labels, 
    suggestedAssetID
)
```
Source: [internal/service/builder.go:280](../../internal/service/builder.go#L280)

**Base Asset Initialization**:
Builderd automatically downloads and registers base VM assets (kernel, rootfs) on startup to ensure microVM infrastructure availability.

```go
func initializeBaseAssets(ctx context.Context, logger *slog.Logger, cfg *config.Config, assetClient *assetmanager.Client) error
```
Source: [cmd/builderd/main.go:615](../../cmd/builderd/main.go#L615)

### SPIFFE/SPIRE Authentication

All service communications use SPIFFE/SPIRE for mTLS authentication and authorization.

**TLS Provider Integration**: [pkg/tls](../../internal/config/config.go#L224)

**Configuration**:
```go
type TLSConfig struct {
    Mode             string // "disabled", "file", "spiffe"  
    SPIFFESocketPath string // "/var/lib/spire/agent/agent.sock"
    CertFile         string // For file-based TLS
    KeyFile          string // For file-based TLS
    CAFile           string // For file-based TLS
}
```
Source: [internal/config/config.go:134](../../internal/config/config.go#L134)

### Metald Integration

While builderd doesn't directly call metald, it produces assets that metald consumes for VM creation.

**Asset Labeling for Metald**:
```go
labels := map[string]string{
    "docker_image": dockerSource.GetImageUri(),  // Enables metald VM queries
    "tenant_id":    tenantID,
    "customer_id":  customerID,
}
```
Source: [internal/service/builder.go:261](../../internal/service/builder.go#L261)

## Multi-Tenant Architecture

### Tenant Isolation

Builderd enforces strict tenant isolation at multiple layers:

**Request-Level Isolation**:
- Tenant authentication via SPIFFE identity validation
- Resource quota enforcement per tenant tier
- Storage path isolation for build artifacts

**Tenant Context Flow**:
```go
type TenantContext struct {
    TenantID       string
    CustomerID     string  
    Tier           TenantTier
    Permissions    []string
    Metadata       map[string]string
}
```
Source: [proto/builder/v1/builder.proto:69](../../proto/builder/v1/builder.proto#L69)

### Resource Quotas

**Tenant Resource Limits**:
```go
type TenantResourceLimits struct {
    MaxMemoryBytes       int64
    MaxCPUCores         int32
    MaxDiskBytes        int64
    TimeoutSeconds      int32
    MaxConcurrentBuilds int32
    MaxDailyBuilds      int32
    MaxStorageBytes     int64
    MaxBuildTimeMinutes int32
}
```
Source: [proto/builder/v1/builder.proto:207](../../proto/builder/v1/builder.proto#L207)

**Default Limits by Tier**: [internal/config/config.go:188](../../internal/config/config.go#L188)

### Tenant Manager

**Implementation**: [internal/tenant/manager.go](../../internal/tenant/manager.go)

Handles tenant-specific configuration, quota enforcement, and usage tracking.

## Build Execution Pipeline

### Build Lifecycle States

```
PENDING → PULLING → EXTRACTING → BUILDING → OPTIMIZING → COMPLETED
                                      ↓
                                   FAILED
                                      ↓  
                                  CANCELLED
```

**State Definitions**: [proto/builder/v1/builder.proto:38](../../proto/builder/v1/builder.proto#L38)

### Async Build Execution

Builds execute asynchronously to avoid blocking API calls:

```go
s.buildWg.Add(1)
go func() {
    defer s.buildWg.Done()
    buildCtx := s.shutdownCtx  // Shutdown-aware context
    
    // Build execution with coordinated cancellation
    buildResult, err := s.executors.ExecuteWithID(buildCtx, req.Msg, buildJob.BuildId)
}()
```
Source: [internal/service/builder.go:177](../../internal/service/builder.go#L177)

### Build Result Processing

After successful builds:
1. Update build job state and metadata
2. Register rootfs with assetmanagerd  
3. Register appropriate kernel with assetmanagerd
4. Record build metrics and telemetry

## Storage Architecture

### Storage Backends

**Current Implementation**: Local filesystem storage with configurable backends.

**Configuration**: [internal/config/config.go:42](../../internal/config/config.go#L42)

**Future Backends**:
- **S3**: AWS S3 and S3-compatible storage
- **GCS**: Google Cloud Storage
- **NFS**: Network filesystem storage

### Build Artifacts

**Directory Structure**:
```
/opt/builderd/
├── workspace/           # Temporary build workspaces
│   └── build-<id>/     # Per-build working directories
├── rootfs/             # Built rootfs outputs
│   ├── build-<id>.ext4     # Ext4 filesystem images
│   ├── build-<id>.metadata.json # Container metadata
│   └── base/               # Base VM assets (kernel, rootfs)
└── scratch/            # Temporary storage during builds
```

**Path Configuration**: [internal/config/config.go:32](../../internal/config/config.go#L32)

## Observability Architecture

### OpenTelemetry Integration

**Implementation**: [internal/observability/otel.go](../../internal/observability/otel.go)

**Tracing Instrumentation**:
- Build operation spans with step-level detail
- Service-to-service request tracing  
- Docker executor step tracing

**Span Hierarchy**:
```
builderd.docker.build
├── builderd.docker.pull_image
├── builderd.docker.create_container  
├── builderd.docker.extract_filesystem
├── builderd.docker.optimize_rootfs
└── builderd.docker.create_ext4_image
```

### Metrics Collection

**Implementation**: [internal/observability/metrics.go](../../internal/observability/metrics.go)

**Key Metrics**:
- `builderd_builds_total` - Build counters by type and tenant
- `builderd_build_duration_seconds` - Build completion times
- `builderd_active_builds` - Currently running builds
- `builderd_build_errors_total` - Failure counters by error type

**High-Cardinality Metrics**: Optional tenant-specific metrics when enabled.

### Health Checks

**Health Endpoint**: `/health` with rate limiting and comprehensive validation.

**Implementation**: [cmd/builderd/main.go:497](../../cmd/builderd/main.go#L497)

## Security Architecture

### Process Isolation

**Build Sandbox**: Each build executes in isolated environment with:
- Temporary working directories with restricted permissions
- Resource limits enforced via cgroups (planned)
- Network isolation for registry access only

### Path Validation

**Security Validation**: [internal/executor/docker.go:799](../../internal/executor/docker.go#L799)

```go
func validateAndSanitizePath(rootfsDir, targetPath string) error {
    // Prevent directory traversal attacks
    // Validate path length and dangerous characters
    // Ensure paths remain within rootfs directory
}
```

### Registry Security

**Docker Registry Authentication**: Support for private registries with credentials.

**Configuration**: [internal/config/config.go:69](../../internal/config/config.go#L69)

## Configuration Management

### Environment Variables

All configuration uses the `UNKEY_BUILDERD_*` prefix pattern:

**Core Configuration**:
- `UNKEY_BUILDERD_PORT` - Server port (default: 8082)
- `UNKEY_BUILDERD_ADDRESS` - Bind address (default: 0.0.0.0)
- `UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS` - Build concurrency (default: 5)
- `UNKEY_BUILDERD_BUILD_TIMEOUT` - Build timeout (default: 15m)

**Full Configuration**: [internal/config/config.go:147](../../internal/config/config.go#L147)

### Configuration Validation

**Implementation**: [internal/config/config.go:250](../../internal/config/config.go#L250)

Validates all configuration values on startup with clear error messages for misconfigurations.

## Error Handling

### Build Error Types

**Error Classification**: [internal/executor/types.go:99](../../internal/executor/types.go#L99)

- `source_not_found` - Image/repository not accessible
- `source_too_large` - Exceeds size limits
- `extraction_failed` - Filesystem extraction error
- `permission_denied` - Insufficient access rights
- `quota_exceeded` - Tenant limits exceeded
- `timeout` - Build timeout exceeded
- `internal_error` - Server-side processing error

### Error Propagation

Errors are consistently propagated with context and structured logging throughout the build pipeline.

## Future Architecture Considerations

### Planned Enhancements

1. **Database Integration**: Persistent build job storage with PostgreSQL/SQLite
2. **Build Caching**: Intermediate build artifact caching for performance
3. **Git Integration**: Direct Git repository build support
4. **Multi-Backend Executors**: Support for additional build strategies
5. **Enhanced Resource Management**: cgroups-based resource isolation

### Scalability Patterns

- **Horizontal Scaling**: Multi-instance deployment with shared storage
- **Work Distribution**: Queue-based build distribution (planned)
- **Cache Layer**: Redis-based build result caching (planned)