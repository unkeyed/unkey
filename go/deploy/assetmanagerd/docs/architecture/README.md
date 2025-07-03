# Assetmanagerd Architecture

This document provides a comprehensive overview of assetmanagerd's architecture, including component interactions, data flow, and design decisions.

## Service Overview

Assetmanagerd serves as the centralized asset management service in the Unkey Deploy platform, responsible for storing, tracking, and lifecycle management of VM assets (kernels, rootfs images). The service integrates tightly with builderd for automatic asset creation and provides reference counting for garbage collection.

## System Architecture

```
┌─────────────┐    ┌─────────────────┐    ┌─────────────┐
│   metald    │────│  assetmanagerd  │────│  builderd   │
│             │    │                 │    │             │
│ VM Assets   │    │ Asset Registry  │    │ Build Exec  │
│ Consumer    │    │ + Storage       │    │ + Register  │
└─────────────┘    └─────────────────┘    └─────────────┘
                            │
                    ┌───────────────────┐
                    │ Storage Backends  │
                    │ • Local FS        │
                    │ • S3 (planned)    │
                    │ • NFS (planned)   │
                    └───────────────────┘
```

## Core Components

### Asset Registry

SQLite-based metadata store providing ACID transactions and efficient querying.

**Implementation**: [registry.go](../../internal/registry/registry.go)

#### Database Schema

```sql
-- Asset metadata table
CREATE TABLE assets (
    id TEXT PRIMARY KEY,                -- ULID identifier
    name TEXT NOT NULL,                 -- Human-readable name
    type INTEGER NOT NULL,              -- AssetType enum
    status INTEGER NOT NULL,            -- AssetStatus enum
    backend INTEGER NOT NULL,           -- StorageBackend enum
    location TEXT NOT NULL,             -- Storage path/URL
    size_bytes INTEGER NOT NULL,        -- File size
    checksum TEXT NOT NULL,             -- SHA256 hash
    created_by TEXT NOT NULL,           -- Creator identifier
    created_at INTEGER NOT NULL,        -- Unix timestamp
    last_accessed_at INTEGER NOT NULL,  -- Last access time
    reference_count INTEGER DEFAULT 0,  -- Active references
    build_id TEXT,                      -- Associated build
    source_image TEXT                   -- Source Docker image
);

-- Asset labels (key-value pairs)
CREATE TABLE asset_labels (
    asset_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (asset_id, key),
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);

-- Reference counting leases
CREATE TABLE asset_leases (
    id TEXT PRIMARY KEY,                -- ULID identifier
    asset_id TEXT NOT NULL,
    acquired_by TEXT NOT NULL,          -- Lease holder identifier
    acquired_at INTEGER NOT NULL,       -- Acquisition timestamp
    expires_at INTEGER,                 -- Optional expiration
    FOREIGN KEY (asset_id) REFERENCES assets(id) ON DELETE CASCADE
);
```

**Source**: [registry.go:62-108](../../internal/registry/registry.go:62)

#### Indexing Strategy

Optimized indexes for common query patterns:
- Type-based filtering: `idx_assets_type`
- Status filtering: `idx_assets_status`  
- Temporal queries: `idx_assets_created_at`, `idx_assets_last_accessed_at`
- Reference counting: `idx_assets_reference_count`
- Build tracking: `idx_assets_build_id`
- Label searches: `idx_asset_labels_key_value`

### Storage Backends

Pluggable storage architecture supporting multiple backend types.

**Interface**: [storage.go](../../internal/storage/storage.go)

#### Backend Interface

```go
type Backend interface {
    Store(ctx context.Context, id string, reader io.Reader, size int64) (string, error)
    Retrieve(ctx context.Context, location string) (io.ReadCloser, error)
    Delete(ctx context.Context, location string) error
    Exists(ctx context.Context, location string) (bool, error)
    GetSize(ctx context.Context, location string) (int64, error)
    GetChecksum(ctx context.Context, location string) (string, error)
    EnsureLocal(ctx context.Context, location string, cacheDir string) (string, error)
    Type() string
}
```

#### Local Backend Implementation

Current production backend using local filesystem with sharding.

**Implementation**: [local.go](../../internal/storage/local.go)

##### File Organization
- Base path: Configurable via `UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH`
- Sharding: First 2 characters of asset ID create subdirectories
- Atomic writes: Temporary files with atomic rename
- Integrity: SHA256 checksum validation

**Sharding Logic**: [local.go:37-43](../../internal/storage/local.go:37)

#### Future Backends

Planned implementations for broader deployment scenarios:
- **S3 Backend**: AWS S3 and compatible object storage
- **NFS Backend**: Network filesystem for shared storage
- **HTTP Backend**: Read-only HTTP/HTTPS URLs

## Service Integration Patterns

### Builderd Integration

Automatic asset creation workflow when assets don't exist.

#### Integration Architecture

```
┌─────────────────┐     Missing Asset     ┌─────────────────┐
│  assetmanagerd  │────────────────────→  │    builderd     │
│                 │                       │                 │
│ QueryAssets()   │                       │ CreateBuild()   │
│                 │                       │                 │
│                 │  ←────────────────────│ RegisterAsset() │
│ Auto-registered │     Built Asset       │                 │
└─────────────────┘                       └─────────────────┘
```

#### Trigger Conditions

Automatic builds are triggered when:
1. `QueryAssets` or `ListAssets` returns no results
2. Request includes `docker_image` label for rootfs type
3. Builderd integration is enabled
4. Builderd client is available

**Trigger Logic**: [service.go:228-255](../../internal/service/service.go:228)

#### Build Workflow

1. **Build Triggering**: [service.go:863-935](../../internal/service/service.go:863)
   - Extract docker image from labels
   - Create build request with tenant context
   - Submit to builderd via gRPC

2. **Build Monitoring**: [service.go:895-923](../../internal/service/service.go:895)
   - Poll build status every 5 seconds
   - Handle completion, failure, and cancellation
   - Timeout after configured duration

3. **Asset Registration**: Automatic via builderd post-build hooks
   - Builderd uploads rootfs to storage
   - Builderd calls `RegisterAsset` to add metadata
   - Asset becomes available for subsequent queries

### SPIFFE/SPIRE Integration

All service communication secured with mutual TLS via SPIFFE/SPIRE.

**TLS Configuration**: [main.go:102-124](../../cmd/assetmanagerd/main.go:102)

#### Authentication Flow

```
┌─────────────┐    mTLS + SPIFFE ID    ┌─────────────────┐
│   Client    │───────────────────────→│  assetmanagerd  │
│             │                       │                 │
│ SVID Cert   │                       │ Verify SPIFFE   │
│             │  ←───────────────────── │ ID + Issue SVID │
└─────────────┘    Authenticated      └─────────────────┘
```

#### Security Features
- Automatic certificate rotation
- Identity-based access control
- Encrypted communication
- Service identity verification

## Reference Counting & Lifecycle Management

### Lease-based Reference Counting

Assets use lease-based reference counting to prevent deletion of in-use resources.

#### Lease Management

**Creation**: [registry.go:350-393](../../internal/registry/registry.go:350)
```go
func (r *Registry) CreateLease(assetID, acquiredBy string, ttl time.Duration) (string, error)
```

**Release**: [registry.go:395-435](../../internal/registry/registry.go:395)
```go
func (r *Registry) ReleaseLease(leaseID string) error
```

#### Reference Count Updates

Atomic operations ensure consistency:
- `AcquireAsset`: Increment reference count, create lease
- `ReleaseAsset`: Decrement reference count, remove lease
- Database transactions prevent race conditions

### Garbage Collection

Automated cleanup of expired leases and unreferenced assets.

**GC Implementation**: [service.go:407-495](../../internal/service/service.go:407)

#### GC Process

1. **Expired Lease Cleanup**: [service.go:418-436](../../internal/service/service.go:418)
   - Query leases past expiration time
   - Release expired leases automatically
   - Decrement asset reference counts

2. **Unreferenced Asset Cleanup**: [service.go:440-483](../../internal/service/service.go:440)
   - Find assets with zero references
   - Filter by last accessed time threshold
   - Delete from storage and registry atomically

3. **Background GC**: [service.go:628-666](../../internal/service/service.go:628)
   - Runs on configurable interval (default: 1 hour)
   - Respects configured max age (default: 7 days)
   - Logs freed space and deleted asset counts

#### GC Configuration

```bash
# Garbage collection settings
UNKEY_ASSETMANAGERD_GC_ENABLED=true
UNKEY_ASSETMANAGERD_GC_INTERVAL=1h
UNKEY_ASSETMANAGERD_GC_MAX_AGE=168h  # 7 days
```

## Asset Preparation Pipeline

Optimized asset staging for VM deployment with Firecracker integration.

**Implementation**: [service.go:497-626](../../internal/service/service.go:497)

### Preparation Process

1. **Asset Resolution**: Verify all requested assets exist
2. **Local Staging**: Ensure assets are available locally via `EnsureLocal()`
3. **Target Preparation**: Create target directory structure
4. **File Operations**: Hardlink or copy assets to target locations
5. **Firecracker Naming**: Standardize filenames for Firecracker compatibility

### Firecracker Integration

Specific handling for Firecracker microVM requirements:

```go
// Standardized naming for Firecracker
switch asset.GetType() {
case assetv1.AssetType_ASSET_TYPE_KERNEL:
    filename = "vmlinux"           // Kernel binary
case assetv1.AssetType_ASSET_TYPE_ROOTFS:
    filename = "rootfs.ext4"       // Root filesystem
default:
    filename = filepath.Base(localPath)
}
```

**Source**: [service.go:535-544](../../internal/service/service.go:535)

### Container Metadata Handling

Rootfs assets include optional metadata for container initialization:

**Metadata Logic**: [service.go:576-612](../../internal/service/service.go:576)

- Looks for `.metadata.json` files alongside rootfs assets
- Copies metadata to `metadata.json` in target directory
- Graceful handling when metadata files don't exist

## Configuration Architecture

Environment-based configuration with comprehensive validation.

**Config Implementation**: [config.go](../../internal/config/config.go)

### Configuration Categories

#### Service Configuration
```bash
UNKEY_ASSETMANAGERD_PORT=8083           # Service port
UNKEY_ASSETMANAGERD_ADDRESS=0.0.0.0     # Bind address
```

#### Storage Configuration
```bash
UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local
UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/opt/vm-assets
UNKEY_ASSETMANAGERD_DATABASE_PATH=/opt/assetmanagerd/assets.db
UNKEY_ASSETMANAGERD_CACHE_DIR=/opt/assetmanagerd/cache
```

#### Builderd Integration
```bash
UNKEY_ASSETMANAGERD_BUILDERD_ENABLED=true
UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT=https://localhost:8082
UNKEY_ASSETMANAGERD_BUILDERD_TIMEOUT=30m
UNKEY_ASSETMANAGERD_BUILDERD_AUTO_REGISTER=true
```

#### Performance Tuning
```bash
UNKEY_ASSETMANAGERD_MAX_ASSET_SIZE=10737418240    # 10GB
UNKEY_ASSETMANAGERD_MAX_CACHE_SIZE=107374182400   # 100GB
UNKEY_ASSETMANAGERD_DOWNLOAD_CONCURRENCY=4
```

### Validation Logic

Comprehensive validation ensures service reliability:

**Validation**: [config.go:88-152](../../internal/config/config.go:88)

- Port range validation
- Storage backend compatibility checks
- Size limit consistency verification
- Builderd configuration validation
- TLS mode verification

## Observability Architecture

Comprehensive monitoring with OpenTelemetry integration.

**Observability**: [otel.go](../../internal/observability/otel.go)

### Telemetry Export

- **Traces**: OTLP HTTP to collector (default: localhost:4318)
- **Metrics**: Prometheus endpoint (:9467) + OTLP export
- **Sampling**: Configurable trace sampling rate

### Service Metrics

Key metrics exported for monitoring:

- Request duration and error rates
- Asset operation counters
- Storage backend performance
- GC effectiveness metrics
- Builderd integration success rates

### Health Checks

Comprehensive health monitoring:

**Health Handler**: [main.go:262](../../cmd/assetmanagerd/main.go:262)

- Service uptime tracking
- Version information
- Component health status

## Data Flow Patterns

### Asset Registration Flow

```
1. Asset Upload (builderd) → Storage Backend
2. Builderd → RegisterAsset() → assetmanagerd
3. Verify existence in storage
4. Calculate checksum if missing
5. Atomic registry transaction
6. Return asset metadata
```

### Asset Query Flow

```
1. Client → QueryAssets() → assetmanagerd
2. Query registry with filters
3. If empty results + docker_image label:
   a. Trigger builderd build
   b. Wait for completion (optional)
   c. Re-query registry
4. Return assets (+ build info)
```

### Asset Preparation Flow

```
1. Client → PrepareAssets() → assetmanagerd
2. Validate all asset IDs exist
3. For each asset:
   a. EnsureLocal() via storage backend
   b. Create target directory
   c. Hardlink or copy to target
   d. Handle metadata files
4. Return asset path mapping
```

## Security Considerations

### Authentication & Authorization

- **mTLS**: All service communication via SPIFFE/SPIRE
- **Tenant Isolation**: Request-scoped tenant context
- **Service Identity**: SPIFFE ID-based access control

### Data Protection

- **Integrity**: SHA256 checksums for all assets
- **Encryption**: TLS for data in transit
- **Access Logging**: Comprehensive audit trails

### Input Validation

- **Size Limits**: Configurable maximum asset sizes
- **Path Sanitization**: Prevent directory traversal attacks
- **Type Validation**: Strict protobuf schema enforcement

## Performance Characteristics

### Scalability

- **Concurrent Operations**: Configurable worker pools
- **Database Optimization**: Indexed queries and connection pooling
- **Storage Sharding**: Filesystem optimization via directory sharding

### Caching Strategy

- **Local Caching**: EnsureLocal() for remote backends
- **Metadata Caching**: SQLite with WAL mode for performance
- **Connection Pooling**: HTTP/2 for gRPC communication

### Resource Management

- **Memory Usage**: Streaming for large file operations
- **Disk Usage**: Automatic garbage collection
- **Network Usage**: Efficient gRPC streaming protocols