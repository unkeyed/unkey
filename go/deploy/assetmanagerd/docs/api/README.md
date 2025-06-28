# AssetManagerd API Documentation

The AssetManagerd service exposes a ConnectRPC API for managing VM assets. All API endpoints use Protocol Buffers for message serialization and support both gRPC and HTTP/JSON protocols.

## Service Definition

**Proto Definition**: [`asset/v1/asset.proto`](../../proto/asset/v1/asset.proto)  
**Service Implementation**: [`internal/service/service.go`](../../internal/service/service.go)

```protobuf
service AssetManagerService {
  rpc RegisterAsset(RegisterAssetRequest) returns (RegisterAssetResponse);
  rpc GetAsset(GetAssetRequest) returns (GetAssetResponse);
  rpc ListAssets(ListAssetsRequest) returns (ListAssetsResponse);
  rpc AcquireAsset(AcquireAssetRequest) returns (AcquireAssetResponse);
  rpc ReleaseAsset(ReleaseAssetRequest) returns (ReleaseAssetResponse);
  rpc PrepareAssets(PrepareAssetsRequest) returns (PrepareAssetsResponse);
  rpc DeleteAsset(DeleteAssetRequest) returns (DeleteAssetResponse);
  rpc GarbageCollect(GarbageCollectRequest) returns (GarbageCollectResponse);
  rpc QueryAssets(QueryAssetsRequest) returns (QueryAssetsResponse);
}
```

## RPC Methods

### RegisterAsset

Registers a new asset in the system. The asset file must already exist in storage before registration. This is typically called by builderd after uploading an asset.

**Implementation**: [`internal/service/service.go:46`](../../internal/service/service.go:46)

#### Request

```protobuf
message RegisterAssetRequest {
  string name = 1;
  AssetType type = 2;
  StorageBackend backend = 3;
  string location = 4;
  int64 size_bytes = 5;
  string checksum = 6;
  map<string, string> labels = 7;
  string created_by = 8;
  string build_id = 9;
  string source_image = 10;
  string id = 11;  // Optional: specific asset ID to use
}
```

#### Response

```protobuf
message RegisterAssetResponse {
  Asset asset = 1;  // Complete asset with generated metadata
}
```

#### Example

```bash
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/RegisterAsset \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ubuntu 22.04 Base",
    "type": "ASSET_TYPE_ROOTFS",
    "backend": "STORAGE_BACKEND_LOCAL",
    "location": "e3/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "size_bytes": 2147483648,
    "checksum": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "labels": {
      "os": "ubuntu",
      "arch": "x86_64",
      "docker_image": "docker.io/library/ubuntu:22.04"
    },
    "build_id": "build-123",
    "source_image": "docker.io/library/ubuntu:22.04",
    "created_by": "builderd"
  }'
```

#### Downstream Calls

- Validates asset doesn't already exist in SQLite database
- Verifies file exists at specified path in storage backend
- Stores metadata in database with `ACTIVE` status
- Updates asset registry for quick lookups

### GetAsset

Retrieves information about a specific asset and optionally ensures it's available locally.

**Implementation**: [`internal/service/service.go:143`](../../internal/service/service.go:143)

#### Request

```protobuf
message GetAssetRequest {
  string id = 1;         // Asset ID
  bool ensure_local = 2; // Download from remote storage if needed
}
```

#### Response

```protobuf
message GetAssetResponse {
  Asset asset = 1;
  string local_path = 2;  // Local path if ensure_local was true
}
```

#### Example

```bash
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/GetAsset \
  -H "Content-Type: application/json" \
  -d '{
    "id": "01HQXYZABC123456789DEFGHJ",
    "ensure_local": true
  }'
```

#### Downstream Behavior

- Reads from SQLite database
- If `ensure_local=true` and asset not in local cache:
  - Downloads from configured storage backend (S3, HTTP, etc.)
  - Verifies checksum
  - Updates local cache

### ListAssets

Lists assets with optional filtering by type, status, and labels.

**Implementation**: [`internal/service/service.go:186`](../../internal/service/service.go:186)

#### Request

```protobuf
message ListAssetsRequest {
  AssetType type = 1;                    // Optional: filter by type
  AssetStatus status = 2;                // Optional: filter by status
  map<string, string> label_selector = 3; // Optional: filter by labels (AND logic)
  int32 page_size = 4;                  // Max results per page (default: 100)
  string page_token = 5;                // Pagination token
}
```

#### Response

```protobuf
message ListAssetsResponse {
  repeated Asset assets = 1;
  string next_page_token = 2;
}
```

#### Example

```bash
# List all active rootfs assets for x86_64
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ASSET_TYPE_ROOTFS",
    "status": "ASSET_STATUS_ACTIVE",
    "label_selector": {"arch": "x86_64"},
    "page_size": 20
  }'
```

### AcquireAsset

Acquires a lease on an asset, preventing it from being garbage collected. Used by metald when creating VMs.

**Implementation**: [`internal/service/service.go:270`](../../internal/service/service.go:270)

#### Request

```protobuf
message AcquireAssetRequest {
  string asset_id = 1;      // Asset to lease
  string acquired_by = 2;   // VM or service acquiring the lease
  int64 ttl_seconds = 3;    // Lease duration (0 = default from config)
}
```

#### Response

```protobuf
message AcquireAssetResponse {
  Asset asset = 1;
  string lease_id = 2;
}
```

#### Example

```bash
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/AcquireAsset \
  -H "Content-Type: application/json" \
  -d '{
    "asset_id": "01HQXYZABC123456789DEFGHJ",
    "acquired_by": "vm-123",
    "ttl_seconds": 3600
  }'
```

#### Downstream Behavior

- Increments asset reference count atomically
- Creates lease record in database
- Sets expiration based on TTL or default (24 hours)

### ReleaseAsset

Releases a previously acquired lease, decrementing the asset's reference count.

**Implementation**: [`internal/service/service.go:318`](../../internal/service/service.go:318)

#### Request

```protobuf
message ReleaseAssetRequest {
  string lease_id = 1;  // Lease ID from AcquireAsset
}
```

#### Response

```protobuf
message ReleaseAssetResponse {
  Asset asset = 1;
}
```

#### Example

```bash
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/ReleaseAsset \
  -H "Content-Type: application/json" \
  -d '{"lease_id": "01HQXYZDEF456789ABCGHIJKL"}'
```

### PrepareAssets

Prepares multiple assets for use by copying or linking them to target locations. This is the primary RPC used by metald to prepare assets in VM jailer chroot paths.

**Implementation**: [`internal/service/service.go:497`](../../internal/service/service.go:497)

#### Request

```protobuf
message PrepareAssetsRequest {
  repeated string asset_ids = 1;   // Assets to prepare
  string target_path = 2;          // Target directory
  string prepared_for = 3;         // VM or service ID
}
```

#### Response

```protobuf
message PrepareAssetsResponse {
  map<string, string> asset_paths = 1;  // asset_id -> local path
}
```

#### Example

```bash
# Metald preparing assets for a new VM
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/PrepareAssets \
  -H "Content-Type: application/json" \
  -d '{
    "asset_ids": ["kernel-123", "rootfs-456"],
    "target_path": "/var/lib/firecracker/vm-789/chroot",
    "prepared_for": "vm-789"
  }'
```

#### Downstream Behavior

- Acquires leases on all requested assets
- Creates target directory if it doesn't exist
- For each asset:
  - Attempts hard link first (same filesystem optimization)
  - Falls back to copy if cross-filesystem
  - Downloads from remote storage if not cached locally
- Returns mapping of asset IDs to prepared paths

### DeleteAsset

Marks an asset for deletion. The asset will be removed during the next garbage collection cycle if it has no active leases.

**Implementation**: [`internal/service/service.go:350`](../../internal/service/service.go:350)

#### Request

```protobuf
message DeleteAssetRequest {
  string id = 1;     // Asset to delete
  bool force = 2;    // Force deletion even with active leases
}
```

#### Response

```protobuf
message DeleteAssetResponse {
  bool deleted = 1;
  string message = 2;
}
```

#### Example

```bash
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/DeleteAsset \
  -H "Content-Type: application/json" \
  -d '{
    "id": "01HQXYZABC123456789DEFGHJ",
    "force": false
  }'
```

### GarbageCollect

Manually triggers garbage collection to clean up expired leases and deleted assets.

**Implementation**: [`internal/service/service.go:407`](../../internal/service/service.go:407)

#### Request

```protobuf
message GarbageCollectRequest {
  int64 max_age_seconds = 1;       // Delete assets not accessed in this many seconds
  bool delete_unreferenced = 2;    // Delete assets with 0 references
  bool dry_run = 3;               // If true, only report what would be deleted
}
```

#### Response

```protobuf
message GarbageCollectResponse {
  repeated Asset deleted_assets = 1;
  int64 bytes_freed = 2;
}
```

#### Example

```bash
# Dry run to see what would be cleaned up
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/GarbageCollect \
  -H "Content-Type: application/json" \
  -d '{"dry_run": true}'
```

#### Garbage Collection Process

1. Removes expired leases
2. Identifies assets with:
   - Status = DELETED
   - Reference count = 0
   - Last accessed > max_age
3. Removes files from storage
4. Deletes database records

### QueryAssets

Enhanced asset query with automatic build triggering capabilities. This is the primary method used by metald to get assets and automatically trigger builds when assets don't exist.

**Implementation**: [`internal/service/service.go:790`](../../internal/service/service.go:790)

#### Request

```protobuf
message QueryAssetsRequest {
  AssetType type = 1;                    // Optional: filter by type
  AssetStatus status = 2;                // Optional: filter by status
  map<string, string> label_selector = 3; // Optional: filter by labels (AND logic)
  int32 page_size = 4;                  // Max results per page (default: 100)
  string page_token = 5;                // Pagination token
  BuildOptions build_options = 6;       // Build options for automatic asset creation
}

message BuildOptions {
  bool enable_auto_build = 1;           // Enable automatic building if assets don't exist
  bool wait_for_completion = 2;         // Wait for build completion before returning
  int32 build_timeout_seconds = 3;      // Timeout for build operation
  map<string, string> build_labels = 4; // Additional labels for the built asset
  string tenant_id = 5;                 // Tenant context for build authorization
  string suggested_asset_id = 6;        // Suggested asset ID for registration
}
```

#### Response

```protobuf
message QueryAssetsResponse {
  repeated Asset assets = 1;
  string next_page_token = 2;
  repeated BuildInfo triggered_builds = 3;
}

message BuildInfo {
  string build_id = 1;
  string docker_image = 2;
  string status = 3;        // "pending", "building", "completed", "failed"
  string error_message = 4;
  string asset_id = 5;      // Asset ID if build completed
}
```

#### Example

```bash
# Query for nginx rootfs with automatic build
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/QueryAssets \
  -H "Content-Type: application/json" \
  -d '{
    "type": "ASSET_TYPE_ROOTFS",
    "label_selector": {
      "docker_image": "nginx:latest"
    },
    "build_options": {
      "enable_auto_build": true,
      "wait_for_completion": true,
      "build_timeout_seconds": 1800,
      "tenant_id": "tenant-123",
      "suggested_asset_id": "nginx-rootfs-123"
    }
  }'
```

#### Downstream Behavior

- First queries existing assets like ListAssets
- If no assets found and auto-build enabled:
  - Extracts docker_image from label_selector
  - Calls builderd to trigger a new build
  - Optionally waits for completion if wait_for_completion=true
  - Returns build information in triggered_builds field
- Integration requires builderd client to be configured and enabled

## Data Types

### AssetType Enum

```protobuf
enum AssetType {
  ASSET_TYPE_UNSPECIFIED = 0;
  ASSET_TYPE_KERNEL = 1;      // Linux kernel image
  ASSET_TYPE_ROOTFS = 2;      // Root filesystem
  ASSET_TYPE_INITRD = 3;      // Initial ramdisk
  ASSET_TYPE_DISK_IMAGE = 4;  // Additional disk image
}
```

### AssetStatus Enum

```protobuf
enum AssetStatus {
  ASSET_STATUS_UNSPECIFIED = 0;
  ASSET_STATUS_UPLOADING = 1;   // Being uploaded
  ASSET_STATUS_AVAILABLE = 2;   // Available for use
  ASSET_STATUS_DELETING = 3;    // Being deleted
  ASSET_STATUS_ERROR = 4;       // Error state
}
```

### StorageBackend Enum

```protobuf
enum StorageBackend {
  STORAGE_BACKEND_UNSPECIFIED = 0;
  STORAGE_BACKEND_LOCAL = 1;    // Local filesystem storage
  STORAGE_BACKEND_S3 = 2;       // Amazon S3 or compatible
  STORAGE_BACKEND_HTTP = 3;     // HTTP URL-based storage
  STORAGE_BACKEND_NFS = 4;      // Network File System
}
```

## HTTP Endpoints

### GET /health

Health check endpoint provided by the health package.

**Implementation**: [`cmd/assetmanagerd/main.go`](../../cmd/assetmanagerd/main.go)

#### Response

```json
{
  "status": "healthy",
  "service": "assetmanagerd",
  "version": "0.3.0",
  "uptime": "24h30m15s"
}
```

### GET /metrics

Prometheus metrics endpoint (when OpenTelemetry is enabled).

**Port**: 9467 (configurable)

#### Available Metrics

- `assetmanager_assets_total{type,status}` - Total assets by type and status
- `assetmanager_leases_active{asset_id}` - Active leases per asset
- `assetmanager_storage_bytes_used{type}` - Storage usage by asset type
- `assetmanager_gc_duration_seconds` - Garbage collection duration
- `assetmanager_prepare_duration_seconds` - Asset preparation latency
- `assetmanager_rpc_duration_seconds{method}` - RPC method latencies

## Error Handling

The API uses standard ConnectRPC error codes:

| Code | Name | Usage |
|------|------|-------|
| 3 | InvalidArgument | Invalid request parameters |
| 5 | NotFound | Asset or lease not found |
| 6 | AlreadyExists | Asset ID already registered |
| 7 | PermissionDenied | Authorization failure |
| 9 | FailedPrecondition | Invalid state for operation |
| 13 | Internal | Unexpected server error |
| 16 | Unauthenticated | Missing authentication |

### Error Response Format

```json
{
  "code": "not_found",
  "message": "asset not found: 01HQXYZABC123456789DEFGHJ",
  "details": []
}
```

## Authentication

The service supports three authentication modes via TLS configuration:

1. **SPIFFE** (default): Automatic mTLS via SPIRE agent
2. **File**: Manual certificate management
3. **Disabled**: No authentication (development only)

Production deployments should use SPIFFE mode for service-to-service authentication.

### Client Setup Example

```go
// Client setup with SPIFFE
source, err := workloadapi.NewX509Source(ctx)
tlsConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())

httpClient := &http.Client{
    Transport: &http2.Transport{
        TLSClientConfig: tlsConfig,
    },
}

client := assetv1connect.NewAssetManagerServiceClient(
    httpClient,
    "https://assetmanagerd:8083",
)
```

## Rate Limiting

Currently, no rate limiting is implemented at the API level. Consider implementing rate limits at the proxy/load balancer level for production deployments.

## API Versioning

The API uses protobuf package versioning (`asset.v1`). Breaking changes will result in a new major version (`v2`).