# AssetManagerd API Reference

The AssetManagerd service exposes a ConnectRPC/gRPC API for managing VM-related assets. All RPCs require mTLS authentication via SPIFFE/SPIRE.

**Service Definition**: [proto/asset/v1/asset.proto](../../proto/asset/v1/asset.proto)  
**Generated Client**: [gen/asset/v1/assetv1connect/asset.connect.go](../../gen/asset/v1/assetv1connect/asset.connect.go)  
**Service Implementation**: [internal/service/service.go](../../internal/service/service.go)

## Service: AssetService

### RegisterAsset

Registers a new asset in the system. Typically called by builderd after creating VM images.

**Implementation**: [internal/service/service.go:103](../../internal/service/service.go:103)

```protobuf
rpc RegisterAsset(RegisterAssetRequest) returns (RegisterAssetResponse);
```

#### Request
```go
type RegisterAssetRequest struct {
    Name      string            // Human-readable asset name
    Type      AssetType         // KERNEL, ROOTFS, INITRD, or DISK_IMAGE
    Backend   StorageBackend    // Storage backend (default: LOCAL)
    Location  string            // Backend-specific location
    SizeBytes int64             // Asset size in bytes
    Checksum  string            // SHA256 checksum
    Labels    map[string]string // Metadata labels
    BuildId   string            // Optional builderd build ID
    SourceImage string          // Optional source image reference
}
```

#### Response
```go
type RegisterAssetResponse struct {
    Asset *Asset // Complete asset object with generated ID
}
```

#### Example
```go
resp, err := client.RegisterAsset(ctx, &assetv1.RegisterAssetRequest{
    Name: "ubuntu-24.04-kernel",
    Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
    Location: "/builds/kernels/ubuntu-24.04.kernel",
    SizeBytes: 12345678,
    Checksum: "sha256:abcdef0123456789",
    Labels: map[string]string{
        "os": "ubuntu",
        "version": "24.04",
        "arch": "amd64",
    },
    BuildId: "build-xyz123",
})
```

#### Downstream Calls
- Writes to local SQLite database
- Creates asset record with `UPLOADING` status
- No external service calls

### GetAsset

Retrieves asset metadata and optionally ensures the asset is available locally.

**Implementation**: [internal/service/service.go:161](../../internal/service/service.go:161)

```protobuf
rpc GetAsset(GetAssetRequest) returns (GetAssetResponse);
```

#### Request
```go
type GetAssetRequest struct {
    AssetId      string // Asset UUID
    EnsureLocal  bool   // Download if not locally cached
}
```

#### Response
```go
type GetAssetResponse struct {
    Asset *Asset // Complete asset metadata
}
```

#### Example
```go
resp, err := client.GetAsset(ctx, &assetv1.GetAssetRequest{
    AssetId: "550e8400-e29b-41d4-a716-446655440000",
    EnsureLocal: true, // Will download from S3 if not local
})
```

#### Downstream Calls
- Reads from SQLite database
- If `EnsureLocal=true` and asset not local:
  - Downloads from configured storage backend
  - Updates local cache

### ListAssets

Lists assets with optional filtering by type, status, and labels.

**Implementation**: [internal/service/service.go:184](../../internal/service/service.go:184)

```protobuf
rpc ListAssets(ListAssetsRequest) returns (ListAssetsResponse);
```

#### Request
```go
type ListAssetsRequest struct {
    Type   AssetType         // Filter by asset type (optional)
    Status AssetStatus       // Filter by status (optional)
    Labels map[string]string // Filter by labels (AND logic)
}
```

#### Response
```go
type ListAssetsResponse struct {
    Assets []*Asset // Matching assets
}
```

#### Example
```go
// List all available kernel images for Ubuntu 24.04
resp, err := client.ListAssets(ctx, &assetv1.ListAssetsRequest{
    Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
    Status: assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
    Labels: map[string]string{
        "os": "ubuntu",
        "version": "24.04",
    },
})
```

### AcquireAsset

Creates a lease on an asset, incrementing its reference count. Used by metald when creating VMs.

**Implementation**: [internal/service/service.go:218](../../internal/service/service.go:218)

```protobuf
rpc AcquireAsset(AcquireAssetRequest) returns (AcquireAssetResponse);
```

#### Request
```go
type AcquireAssetRequest struct {
    AssetId    string               // Asset to acquire
    AcquiredBy string               // Service/VM acquiring the asset
    TtlSeconds int64                // Optional lease duration (0 = infinite)
}
```

#### Response
```go
type AcquireAssetResponse struct {
    LeaseId string // Unique lease identifier
}
```

#### Example
```go
resp, err := client.AcquireAsset(ctx, &assetv1.AcquireAssetRequest{
    AssetId: "kernel-550e8400",
    AcquiredBy: "vm-123",
    TtlSeconds: 3600, // 1 hour lease
})
// Store resp.LeaseId for later release
```

#### Downstream Calls
- Updates asset reference count in SQLite (atomic increment)
- Creates lease record with expiration time

### ReleaseAsset

Releases an asset lease, decrementing its reference count.

**Implementation**: [internal/service/service.go:256](../../internal/service/service.go:256)

```protobuf
rpc ReleaseAsset(ReleaseAssetRequest) returns (ReleaseAssetResponse);
```

#### Request
```go
type ReleaseAssetRequest struct {
    LeaseId string // Lease ID from AcquireAsset
}
```

#### Response
```go
type ReleaseAssetResponse struct {
    // Empty response
}
```

#### Example
```go
_, err := client.ReleaseAsset(ctx, &assetv1.ReleaseAssetRequest{
    LeaseId: "lease-abc123",
})
```

### DeleteAsset

Deletes an asset from the system. Fails if the asset has active references unless forced.

**Implementation**: [internal/service/service.go:275](../../internal/service/service.go:275)

```protobuf
rpc DeleteAsset(DeleteAssetRequest) returns (DeleteAssetResponse);
```

#### Request
```go
type DeleteAssetRequest struct {
    AssetId string // Asset to delete
    Force   bool   // Force deletion even with active references
}
```

#### Response
```go
type DeleteAssetResponse struct {
    // Empty response
}
```

#### Example
```go
_, err := client.DeleteAsset(ctx, &assetv1.DeleteAssetRequest{
    AssetId: "old-kernel-123",
    Force: false, // Fail if in use
})
```

### GarbageCollect

Triggers garbage collection to clean up unreferenced assets older than the threshold.

**Implementation**: [internal/service/service.go:309](../../internal/service/service.go:309)

```protobuf
rpc GarbageCollect(GarbageCollectRequest) returns (GarbageCollectResponse);
```

#### Request
```go
type GarbageCollectRequest struct {
    AgeThresholdSeconds int64 // Assets older than this are eligible
    DryRun              bool  // Preview without deleting
}
```

#### Response
```go
type GarbageCollectResponse struct {
    DeletedAssets []*Asset // Assets that were/would be deleted
}
```

#### Example
```go
resp, err := client.GarbageCollect(ctx, &assetv1.GarbageCollectRequest{
    AgeThresholdSeconds: 7 * 24 * 3600, // 7 days
    DryRun: true, // Preview first
})
fmt.Printf("Would delete %d assets\n", len(resp.DeletedAssets))
```

### PrepareAssets

Stages assets in specified locations, typically for VM jailer preparation. Primary RPC used by metald.

**Implementation**: [internal/service/service.go:343](../../internal/service/service.go:343)

```protobuf
rpc PrepareAssets(PrepareAssetsRequest) returns (PrepareAssetsResponse);
```

#### Request
```go
type PrepareAssetsRequest struct {
    Assets []*AssetReference // Assets to prepare
}

type AssetReference struct {
    AssetId    string // Asset UUID
    TargetPath string // Where to stage the asset
}
```

#### Response
```go
type PrepareAssetsResponse struct {
    // Empty - success means all assets prepared
}
```

#### Example
```go
// Metald preparing assets for a new VM
resp, err := client.PrepareAssets(ctx, &assetv1.PrepareAssetsRequest{
    Assets: []*assetv1.AssetReference{
        {
            AssetId: "kernel-550e8400",
            TargetPath: "/srv/jailer/firecracker/vm-123/root/kernel",
        },
        {
            AssetId: "rootfs-660f9400", 
            TargetPath: "/srv/jailer/firecracker/vm-123/root/rootfs.ext4",
        },
    },
})
```

#### Downstream Calls
- Checks asset availability in local storage
- Creates hard links if on same filesystem (optimization)
- Falls back to copying if cross-filesystem
- Downloads from remote storage if needed

## Data Types

### AssetType
```protobuf
enum AssetType {
    ASSET_TYPE_UNSPECIFIED = 0;
    ASSET_TYPE_KERNEL = 1;      // Linux kernel image
    ASSET_TYPE_ROOTFS = 2;      // Root filesystem image
    ASSET_TYPE_INITRD = 3;      // Initial ramdisk
    ASSET_TYPE_DISK_IMAGE = 4;  // Persistent disk image
}
```

### AssetStatus
```protobuf
enum AssetStatus {
    ASSET_STATUS_UNSPECIFIED = 0;
    ASSET_STATUS_UPLOADING = 1;   // Being uploaded/registered
    ASSET_STATUS_AVAILABLE = 2;   // Ready for use
    ASSET_STATUS_DELETING = 3;    // Being deleted
    ASSET_STATUS_ERROR = 4;       // Error state
}
```

### StorageBackend
```protobuf
enum StorageBackend {
    STORAGE_BACKEND_UNSPECIFIED = 0;
    STORAGE_BACKEND_LOCAL = 1;    // Local filesystem
    STORAGE_BACKEND_S3 = 2;       // S3/Object storage
    STORAGE_BACKEND_HTTP = 3;     // HTTP/HTTPS URL
    STORAGE_BACKEND_NFS = 4;      // Network filesystem
}
```

### Asset
```protobuf
message Asset {
    string id = 1;                      // UUID
    string name = 2;                    // Human-readable name
    AssetType type = 3;                 // Asset type
    AssetStatus status = 4;             // Current status
    StorageBackend backend = 5;         // Storage backend
    string location = 6;                // Backend-specific location
    int64 size_bytes = 7;               // Size in bytes
    string checksum = 8;                // SHA256 checksum
    map<string, string> labels = 9;    // Metadata labels
    string created_by = 10;             // Creator identity
    google.protobuf.Timestamp created_at = 11;
    google.protobuf.Timestamp last_accessed_at = 12;
    int64 reference_count = 13;         // Active references
    string build_id = 14;               // Optional builderd ID
    string source_image = 15;           // Optional source reference
}
```

## Error Handling

All RPCs return standard gRPC status codes:

- `NOT_FOUND` - Asset or lease not found
- `ALREADY_EXISTS` - Asset with same checksum already exists
- `FAILED_PRECONDITION` - Operation not allowed (e.g., delete with active refs)
- `INVALID_ARGUMENT` - Invalid request parameters
- `INTERNAL` - Internal service error

Example error handling:
```go
resp, err := client.DeleteAsset(ctx, req)
if err != nil {
    if status.Code(err) == codes.FailedPrecondition {
        // Asset has active references
        log.Printf("Cannot delete asset: %v", err)
    }
    return err
}
```

## Authentication

All requests must include SPIFFE mTLS credentials:

```go
// Client setup with SPIFFE
source, err := workloadapi.NewX509Source(ctx)
tlsConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())

httpClient := &http.Client{
    Transport: &http2.Transport{
        TLSClientConfig: tlsConfig,
    },
}

client := assetv1connect.NewAssetServiceClient(
    httpClient,
    "https://assetmanagerd:8083",
)
```

## Rate Limiting

Currently, no rate limiting is implemented. Future versions may add per-client rate limits.

## Metrics

All RPCs export Prometheus metrics via the observability interceptor:

- `assetmanagerd_rpc_duration_seconds` - RPC latency histogram
- `assetmanagerd_rpc_requests_total` - RPC request counter
- `assetmanagerd_rpc_errors_total` - RPC error counter

See [Operations Guide](../operations/) for complete metrics reference.