# AssetManagerd Service Implementation

## Overview

AssetManagerd is a dedicated VM asset management service that replaces hardcoded asset lists in metald, providing dynamic asset registration, reference counting, and garbage collection.

## Key Implementation Details

### Port Configuration
- **Service Port**: 8083 (to avoid conflict with builderd on 8082)
- **Prometheus Port**: 9466

### OTEL Configuration
- **CRITICAL**: Use `resource.New()` not `resource.Merge()` to avoid schema conflicts
- **Schema Version**: Uses semconv v1.26.0 for compatibility with other services

### Storage Architecture
- **Sharded Layout**: Assets stored as `/opt/vm-assets/{first-2-chars}/{asset-id}`
- **Example**: Asset `abcd1234...` stored at `/opt/vm-assets/ab/abcd1234...`
- **Rationale**: Prevents filesystem performance issues with too many files in one directory

### Database Schema
- **SQLite**: `/opt/assetmanagerd/assets.db`
- **Tables**:
  - `assets`: Core asset metadata
  - `asset_labels`: Key-value labels for filtering
  - `asset_leases`: Reference counting with optional TTL

### Integration Points

**AIDEV-NOTE**: The hardcoded asset list in `metald/internal/process/manager.go:784` needs to be replaced with dynamic queries to assetmanagerd:
```go
// Instead of:
vmAssets := []string{"vmlinux", "rootfs.ext4", "alpine-builderd.ext4", "busybox-build.ext4", "busybox-builderd.ext4", "build-1749870205104626515.ext4"}

// Use:
assets, err := m.assetClient.ListAssets(ctx, &assetv1.ListAssetsRequest{
    Type: assetv1.AssetType_ASSET_TYPE_ROOTFS,
})
```

### Future Enhancements
1. **S3 Backend**: Already designed for in storage interface
2. **Asset Replication**: For multi-region deployments
3. **Content Deduplication**: Using SHA256 checksums
4. **Remote Sources**: Download assets from HTTP/S3 on demand

### Testing
```bash
# Register an asset
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/RegisterAsset \
  -H "Content-Type: application/json" \
  -d '{
    "name": "test-rootfs.ext4",
    "type": "ASSET_TYPE_ROOTFS",
    "backend": "STORAGE_BACKEND_LOCAL",
    "location": "test-rootfs.ext4",
    "size_bytes": 134217728,
    "created_by": "manual"
  }'

# List assets
curl -X POST http://localhost:8083/asset.v1.AssetManagerService/ListAssets \
  -H "Content-Type: application/json" \
  -d '{"type": "ASSET_TYPE_ROOTFS"}'
```