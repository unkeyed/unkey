# AssetManagerService API Documentation

This document provides comprehensive documentation for the AssetManagerService gRPC API, including all RPCs, message types, and integration patterns.

## Service Overview

AssetManagerService manages VM assets (kernels, rootfs images) with automatic build triggering, reference counting, and lifecycle management. The service supports streaming uploads, multi-backend storage, and tight integration with builderd for on-demand asset creation.

**Proto Definition**: [asset.proto](../../proto/asset/v1/asset.proto)  
**Implementation**: [service.go](../../internal/service/service.go)

## Asset Types

Assets are categorized by type for proper handling and Firecracker compatibility:

```protobuf
enum AssetType {
  ASSET_TYPE_UNSPECIFIED = 0;
  ASSET_TYPE_KERNEL = 1;      // vmlinux kernels
  ASSET_TYPE_ROOTFS = 2;      // Root filesystem images
  ASSET_TYPE_INITRD = 3;      // Initial ramdisk images  
  ASSET_TYPE_DISK_IMAGE = 4;  // Additional disk images
}
```

**Source**: [asset.proto:42-48](../../proto/asset/v1/asset.proto:42)

## Asset Status Lifecycle

Assets progress through different states during their lifecycle:

```protobuf
enum AssetStatus {
  ASSET_STATUS_UNSPECIFIED = 0;
  ASSET_STATUS_UPLOADING = 1;   // Currently being uploaded
  ASSET_STATUS_AVAILABLE = 2;   // Ready for use
  ASSET_STATUS_DELETING = 3;    // Being deleted
  ASSET_STATUS_ERROR = 4;       // Upload/processing failed
}
```

**Source**: [asset.proto:50-56](../../proto/asset/v1/asset.proto:50)

## Storage Backends

Multiple storage backends support different deployment scenarios:

```protobuf
enum StorageBackend {
  STORAGE_BACKEND_UNSPECIFIED = 0;
  STORAGE_BACKEND_LOCAL = 1;    // Local filesystem
  STORAGE_BACKEND_S3 = 2;       // AWS S3 or compatible
  STORAGE_BACKEND_HTTP = 3;     // HTTP/HTTPS URLs
  STORAGE_BACKEND_NFS = 4;      // Network File System
}
```

**Source**: [asset.proto:58-64](../../proto/asset/v1/asset.proto:58)

## Core Data Structures

### Asset

The primary asset metadata structure:

```protobuf
message Asset {
  string id = 1;                    // Unique asset identifier
  string name = 2;                  // Human-readable name
  AssetType type = 3;               // Asset type classification
  AssetStatus status = 4;           // Current lifecycle status
  
  // Storage information
  StorageBackend backend = 5;       // Storage backend type
  string location = 6;              // Path or URL
  int64 size_bytes = 7;             // File size in bytes
  string checksum = 8;              // SHA256 checksum
  
  // Metadata
  map<string, string> labels = 9;   // Key-value labels
  string created_by = 10;           // Creator identifier
  int64 created_at = 11;            // Unix timestamp
  int64 last_accessed_at = 12;      // Last access time
  
  // Reference counting for GC
  int32 reference_count = 13;       // Active reference count
  
  // Build information (if created by builderd)
  string build_id = 14;             // Associated build ID
  string source_image = 15;         // Source Docker image
}
```

**Source**: [asset.proto:66-90](../../proto/asset/v1/asset.proto:66)

## RPC Methods

### RegisterAsset

Registers a pre-stored asset in the registry. Used by builderd after uploading to storage.

```protobuf
rpc RegisterAsset(RegisterAssetRequest) returns (RegisterAssetResponse);
```

**Request**: [asset.proto:114-130](../../proto/asset/v1/asset.proto:114)  
**Implementation**: [service.go:46-141](../../internal/service/service.go:46)

#### Key Features
- Verifies asset existence in storage before registration
- Auto-calculates size and checksum if not provided
- Generates ULID for asset ID if not specified
- Atomic transaction for metadata consistency

### UploadAsset (Streaming)

Handles streaming asset uploads with metadata. First message contains metadata, subsequent messages contain data chunks.

```protobuf
rpc UploadAsset(stream UploadAssetRequest) returns (UploadAssetResponse);
```

**Request**: [asset.proto:92-97](../../proto/asset/v1/asset.proto:92)  
**Implementation**: [service.go:711-822](../../internal/service/service.go:711)

#### Streaming Protocol
1. Client sends `UploadAssetMetadata` in first message
2. Client streams data chunks in subsequent messages  
3. Server stores data and registers asset atomically
4. Auto-cleanup on failure

### GetAsset

Retrieves asset information with optional local staging.

```protobuf
rpc GetAsset(GetAssetRequest) returns (GetAssetResponse);
```

**Implementation**: [service.go:143-185](../../internal/service/service.go:143)

#### Features
- `ensure_local` flag downloads remote assets to cache
- Updates `last_accessed_at` timestamp for GC
- Returns local path when asset is staged locally

### ListAssets

Lists assets with filtering and pagination support.

```protobuf
rpc ListAssets(ListAssetsRequest) returns (ListAssetsResponse);
```

**Implementation**: [service.go:187-268](../../internal/service/service.go:187)

#### Automatic Build Triggering
When no rootfs assets match a docker_image label and builderd is enabled:
- Automatically triggers build via builderd integration
- Re-queries after successful build completion
- Graceful fallback on build failures

**Source**: [service.go:228-255](../../internal/service/service.go:228)

### QueryAssets (Enhanced)

Enhanced version of ListAssets with comprehensive build triggering support.

```protobuf
rpc QueryAssets(QueryAssetsRequest) returns (QueryAssetsResponse);
```

**Implementation**: [service.go:937-1140](../../internal/service/service.go:937)

#### Build Options
- `enable_auto_build`: Trigger builds for missing assets
- `wait_for_completion`: Block until build completes
- `build_timeout_seconds`: Maximum build wait time
- `suggested_asset_id`: Pre-allocate asset ID for builds

### Reference Management

#### AcquireAsset

Creates a lease for an asset, incrementing its reference count.

```protobuf
rpc AcquireAsset(AcquireAssetRequest) returns (AcquireAssetResponse);
```

**Implementation**: [service.go:270-317](../../internal/service/service.go:270)

#### ReleaseAsset

Releases an asset lease, decrementing its reference count.

```protobuf
rpc ReleaseAsset(ReleaseAssetRequest) returns (ReleaseAssetResponse);
```

**Implementation**: [service.go:319-349](../../internal/service/service.go:319)

### Asset Preparation

#### PrepareAssets

Pre-stages assets for VM deployment, creating hardlinks or copies in target directories.

```protobuf
rpc PrepareAssets(PrepareAssetsRequest) returns (PrepareAssetsResponse);
```

**Implementation**: [service.go:497-626](../../internal/service/service.go:497)

#### Firecracker Integration
- Kernel assets → `vmlinux` filename
- Rootfs assets → `rootfs.ext4` filename  
- Automatic metadata file handling for containers
- Efficient hardlinking when possible, fallback to copying

**Source**: [service.go:535-544](../../internal/service/service.go:535)

### Lifecycle Management

#### DeleteAsset

Deletes an asset if reference count is zero (or forced).

```protobuf
rpc DeleteAsset(DeleteAssetRequest) returns (DeleteAssetResponse);
```

**Implementation**: [service.go:351-405](../../internal/service/service.go:351)

#### GarbageCollect

Performs garbage collection of expired leases and unreferenced assets.

```protobuf
rpc GarbageCollect(GarbageCollectRequest) returns (GarbageCollectResponse);
```

**Implementation**: [service.go:407-495](../../internal/service/service.go:407)

##### GC Process
1. Clean up expired leases first
2. Identify unreferenced assets older than threshold
3. Delete from storage and registry atomically
4. Return freed space statistics

## Builderd Integration

### Automatic Asset Creation

When assets don't exist, assetmanagerd automatically triggers builderd:

**Trigger Logic**: [service.go:863-935](../../internal/service/service.go:863)

#### Integration Flow
1. Client queries for rootfs with `docker_image` label
2. No matching assets found in registry
3. Assetmanagerd calls builderd to create rootfs
4. Builderd builds and registers asset automatically
5. Assetmanagerd re-queries and returns new asset

### Build Client

Builderd integration via gRPC client with SPIFFE mTLS:

**Client Implementation**: [builderd/client.go](../../internal/builderd/client.go)

#### Client Features
- SPIFFE/mTLS authentication
- OpenTelemetry trace propagation
- Retry logic with exponential backoff
- Tenant context preservation

## Error Handling

### Connect Error Codes

Standard ConnectRPC error codes are used:

- `InvalidArgument` - Missing required fields
- `NotFound` - Asset or lease not found
- `Internal` - Storage or registry failures
- `FailedPrecondition` - Reference count violations

### Error Propagation

Builderd errors are handled gracefully:
- Build failures logged but don't break queries
- Timeout errors return empty results
- Automatic retries for transient failures

## Multi-tenant Support

### Tenant Authentication

All RPCs support tenant isolation via interceptors:

**Interceptor Config**: [main.go:197-211](../../cmd/assetmanagerd/main.go:197)

#### Exempt Operations
- Health checks (`/health.v1.HealthService/Check`)
- System GC (`/asset.v1.AssetManagerService/GarbageCollect`)

### Tenant Context

Builderd integration preserves tenant context:
- `X-Tenant-ID` header propagation
- Customer ID extraction from context
- Tenant-scoped build requests

**Source**: [service.go:1037-1046](../../internal/service/service.go:1037)

## Usage Examples

### Basic Asset Registration

```go
// Register a kernel asset
req := &assetv1.RegisterAssetRequest{
    Name:      "vmlinux-5.10",
    Type:      assetv1.AssetType_ASSET_TYPE_KERNEL,
    Backend:   assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
    Location:  "vm/vmlinux",
    SizeBytes: 8388608,
    Checksum:  "sha256:abcd...",
    Labels: map[string]string{
        "version": "5.10",
        "arch":    "x86_64",
    },
    CreatedBy: "manual",
}

resp, err := client.RegisterAsset(ctx, connect.NewRequest(req))
```

### Query with Auto-build

```go
// Query for rootfs, trigger build if missing
req := &assetv1.QueryAssetsRequest{
    Type: assetv1.AssetType_ASSET_TYPE_ROOTFS,
    LabelSelector: map[string]string{
        "docker_image": "nginx:latest",
    },
    BuildOptions: &assetv1.BuildOptions{
        EnableAutoBuild:    true,
        WaitForCompletion:  true,
        BuildTimeoutSeconds: 1800, // 30 minutes
    },
}

resp, err := client.QueryAssets(ctx, connect.NewRequest(req))
```

### Asset Preparation

```go
// Prepare assets for VM deployment
req := &assetv1.PrepareAssetsRequest{
    AssetIds:   []string{"kernel-id", "rootfs-id"},
    TargetPath: "/opt/firecracker/vm-123",
    PreparedFor: "vm-123",
}

resp, err := client.PrepareAssets(ctx, connect.NewRequest(req))
// Assets now available as:
// /opt/firecracker/vm-123/vmlinux
// /opt/firecracker/vm-123/rootfs.ext4
```