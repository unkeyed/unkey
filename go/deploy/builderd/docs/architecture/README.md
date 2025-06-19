# Builderd Architecture Guide

## System Architecture

Builderd is a multi-tenant build execution service designed to transform various source types (Docker images, Git repositories, archives) into optimized rootfs images for Firecracker microVM deployment.

### High-Level Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    Clients      │     │   API Gateway   │     │     metald      │
│  (Direct API)   │     │    (Future)     │     │    (Future)     │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                        │
         └───────────────────────┴────────────────────────┘
                                 │
                                 v
                     ┌───────────────────────┐
                     │      builderd         │
                     │  (ConnectRPC/gRPC)    │
                     └───────────┬───────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        v                        v                        v
┌───────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ assetmanagerd │     │  Docker Daemon   │     │ Storage Backend │
│  (Artifact    │     │ (Image Pulling)  │     │ (Local/S3/GCS)  │
│  Registration)│     └──────────────────┘     └─────────────────┘
└───────────────┘
```

### Core Components

#### 1. Service Layer ([service/builder.go](../../../internal/service/builder.go))
- **BuilderService**: Main ConnectRPC service implementation
- Handles all RPC methods (CreateBuild, GetBuild, etc.)
- Validates requests and enforces tenant permissions
- Coordinates with executors for build processing

#### 2. Executor System ([executor/](../../../internal/executor/))
- **Registry** ([registry.go](../../../internal/executor/registry.go)): Manages executor instances
- **DockerExecutor** ([docker.go](../../../internal/executor/docker.go)): Handles Docker image extraction
- Future executors: GitExecutor, ArchiveExecutor
- Implements strategy pattern for extensible build types

#### 3. Multi-Tenant Management ([tenant/](../../../internal/tenant/))
- **Manager** ([manager.go](../../../internal/tenant/manager.go)): Tenant lifecycle management
- **Isolation** ([isolation.go](../../../internal/tenant/isolation.go)): Security boundaries
- **Storage** ([storage.go](../../../internal/tenant/storage.go)): Tenant-specific storage
- Enforces resource quotas and access controls

#### 4. Observability ([observability/](../../../internal/observability/))
- **Metrics** ([metrics.go](../../../internal/observability/metrics.go)): Build and resource metrics
- **Interceptor** ([interceptor.go](../../../internal/observability/interceptor.go)): Request logging and auth
- **OpenTelemetry** ([otel.go](../../../internal/observability/otel.go)): Tracing and metrics export

## Build Execution Pipeline

### Build Lifecycle

1. **Request Validation**
   - Tenant authentication via interceptor
   - Build configuration validation
   - Quota checks and resource limits

2. **Build Scheduling** (Currently synchronous, async planned)
   - Build job creation and persistence
   - Resource allocation
   - Executor selection based on source type

3. **Source Processing**
   - Docker: Pull image from registry
   - Git: Clone repository (future)
   - Archive: Download and extract (future)

4. **Rootfs Generation**
   - Extract layers/files to workspace
   - Apply optimizations (strip debug, remove docs)
   - Configure runtime (init strategy, environment)
   - Package as rootfs archive

5. **Artifact Management**
   - Store rootfs in configured backend
   - Register with assetmanagerd
   - Update build metadata

6. **Cleanup**
   - Remove temporary workspace
   - Release allocated resources
   - Record metrics

### Build States

State transitions managed in [proto definition](../../../proto/builder/v1/builder.proto:38):

```
PENDING → PULLING → EXTRACTING → BUILDING → OPTIMIZING → COMPLETED
   ↓         ↓          ↓            ↓           ↓            ↓
   └─────────┴──────────┴────────────┴───────────┴─→ FAILED
                                                         ↓
                                                    CANCELLED
```

## Multi-Tenant Isolation

### Security Boundaries

1. **Process Isolation**
   - Separate Linux namespaces per build
   - Unprivileged build processes
   - No network access during builds

2. **Filesystem Isolation**
   - Tenant-specific workspace directories
   - Read-only bind mounts for shared resources
   - Separate storage buckets per tenant

3. **Resource Isolation**
   - CPU and memory limits via cgroups
   - Disk quota enforcement
   - Build timeout enforcement

### Tenant Tiers

Defined in [proto](../../../proto/builder/v1/builder.proto:52):

- **FREE**: Limited resources, shared infrastructure
- **PRO**: Standard resources, basic isolation
- **ENTERPRISE**: Higher limits, enhanced isolation
- **DEDICATED**: Dedicated infrastructure, full isolation

Each tier has configurable limits for:
- Maximum concurrent builds
- Daily build quota
- CPU cores and memory
- Storage capacity
- Build timeout

## Storage Architecture

### Storage Backends

1. **Local Filesystem**
   - Development and single-node deployments
   - Path: `{rootfs_output_dir}/{tenant_id}/{build_id}/`
   - Retention via cron cleanup

2. **S3/S3-Compatible**
   - Production cloud deployments
   - Bucket structure: `{bucket}/{tenant_id}/builds/{build_id}/`
   - Lifecycle policies for retention

3. **Google Cloud Storage**
   - GCP deployments
   - Similar structure to S3
   - IAM-based access control

### Caching Strategy

Build caching implemented at multiple levels:

1. **Layer Cache** (Docker builds)
   - Cache key: `{tenant_id}/{image_digest}/{layer_digest}`
   - LRU eviction policy
   - Shared across tenant builds

2. **Artifact Cache**
   - Completed rootfs images
   - Cache key: `{source_hash}_{target_config_hash}`
   - Tenant-isolated

3. **Registry Mirror** (Optional)
   - Local Docker registry mirror
   - Reduces external bandwidth
   - Improves pull performance

## Service Interactions

### Current Integrations

1. **assetmanagerd**
   - Register built artifacts via [client](../../../internal/assetmanager/client.go)
   - Provides centralized asset tracking
   - Enables artifact discovery for VMs

2. **Docker Daemon**
   - Local daemon for image operations
   - Authentication for private registries
   - Layer extraction and manipulation

3. **SPIFFE/SPIRE**
   - Service identity and authentication
   - mTLS for service communication
   - Dynamic certificate rotation

### Future Integrations

1. **metald** (Planned)
   - Will consume builderd for on-demand builds
   - Rootfs provisioning for new VMs
   - Build status callbacks

2. **billaged** (Planned)
   - Resource usage metrics
   - Build time and storage tracking
   - Cost allocation per tenant

3. **Message Queue** (Planned)
   - Async build job queue
   - Build status notifications
   - Event-driven workflows

## Concurrency Model

### Request Handling
- Each RPC handled in separate goroutine
- Context propagation for cancellation
- Timeout enforcement at multiple levels

### Build Execution
- Currently synchronous in request handler
- Future: Worker pool with job queue
- Configurable concurrency limits

### Resource Management
- Semaphore for concurrent build limits
- Mutex protection for shared state
- Atomic operations for metrics

## Data Model

### Build Job Structure

Primary entity defined in [proto](../../../proto/builder/v1/builder.proto:295):

```
BuildJob
├── build_id (UUID)
├── config (BuildConfig)
│   ├── tenant (TenantContext)
│   ├── source (BuildSource)
│   ├── target (BuildTarget)
│   └── strategy (BuildStrategy)
├── state (BuildState)
├── timestamps
│   ├── created_at
│   ├── started_at
│   └── completed_at
├── results
│   ├── rootfs_path
│   ├── rootfs_size_bytes
│   └── rootfs_checksum
├── metrics (BuildMetrics)
└── logs (array)
```

### Database Schema (Future)

```sql
-- Builds table
CREATE TABLE builds (
    id UUID PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    customer_id TEXT,
    state TEXT NOT NULL,
    config JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    rootfs_path TEXT,
    error_message TEXT,
    metrics JSONB,
    INDEX idx_tenant_created (tenant_id, created_at DESC)
);

-- Build logs table
CREATE TABLE build_logs (
    build_id UUID REFERENCES builds(id),
    timestamp TIMESTAMP NOT NULL,
    level TEXT NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB,
    INDEX idx_build_time (build_id, timestamp)
);
```

## Error Handling

### Error Categories

1. **Validation Errors**
   - Invalid configuration
   - Missing required fields
   - Returns `INVALID_ARGUMENT`

2. **Resource Errors**
   - Quota exceeded
   - No available executors
   - Returns `RESOURCE_EXHAUSTED`

3. **Execution Errors**
   - Docker pull failures
   - Build process crashes
   - Returns `INTERNAL`

4. **External Errors**
   - Registry unavailable
   - Storage backend issues
   - Retry with backoff

### Error Propagation

```
Client Request
    ↓
Interceptor (Auth/Logging)
    ↓
Service Handler
    ↓
Executor
    ↓
External Service (Docker/Storage)
```

Errors bubble up with context preservation and are logged at each level.

## Performance Considerations

### Bottlenecks

1. **Docker Operations**
   - Image pulls are network-bound
   - Layer extraction is I/O intensive
   - Mitigation: Registry mirrors, parallel pulls

2. **Storage I/O**
   - Large rootfs writes
   - Concurrent access patterns
   - Mitigation: SSD storage, write batching

3. **Memory Usage**
   - Large images consume significant RAM
   - Multiple concurrent builds
   - Mitigation: Streaming processing, limits

### Optimization Strategies

1. **Caching**
   - Layer-level Docker cache
   - Completed build cache
   - Registry response cache

2. **Parallelization**
   - Concurrent layer downloads
   - Parallel optimization steps
   - Async artifact registration

3. **Resource Pooling**
   - Reusable workspace directories
   - Connection pooling for external services
   - Executor instance pooling

## Security Model

### Authentication
- SPIFFE identities for services
- mTLS for all communication
- Token-based tenant authentication

### Authorization
- Tenant-scoped operations
- Resource-based access control
- Audit logging for all actions

### Build Security
- Unprivileged build processes
- No network access during builds
- Input validation and sanitization
- Resource limits enforcement

## Monitoring and Debugging

### Key Metrics
- Build success/failure rates
- Build duration percentiles
- Resource utilization
- Queue depths and wait times

### Health Indicators
- External service connectivity
- Storage availability
- Memory/CPU pressure
- Error rates by category

### Debug Tools
- Structured JSON logs
- OpenTelemetry traces
- Build artifact inspection
- Tenant usage reports

## Future Architecture

### Planned Enhancements

1. **Async Job Queue**
   - Redis/NATS-based queue
   - Priority scheduling
   - Job dependencies

2. **Distributed Builds**
   - Multiple builderd instances
   - Shared storage backend
   - Consistent hashing for distribution

3. **Build Plugins**
   - Custom optimization steps
   - Language-specific builders
   - Post-processing hooks

4. **Advanced Caching**
   - Content-addressable storage
   - Cross-tenant cache sharing
   - Predictive cache warming