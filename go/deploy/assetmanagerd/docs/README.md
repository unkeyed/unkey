# AssetManagerd - VM Asset Management Service

AssetManagerd is a ConnectRPC service that manages VM assets (kernels, rootfs images) across the infrastructure. It provides asset registration, reference counting, garbage collection, and support for multiple storage backends.

## Architecture

AssetManagerd acts as a centralized registry for VM assets, tracking:
- Asset metadata (type, size, checksum, labels)
- Storage location (local filesystem, S3, etc.)
- Reference counting for safe garbage collection
- Build provenance (which builderd build created it)

## Key Features

1. **Asset Registration**: Track assets created by builderd or uploaded manually
2. **Reference Counting**: Prevent deletion of in-use assets
3. **Automatic Garbage Collection**: Clean up unreferenced assets after configurable TTL
4. **Multiple Storage Backends**: Local filesystem, S3 (future), NFS (future)
5. **Asset Leasing**: Acquire/release references with optional TTL
6. **Local Caching**: Cache remote assets locally for performance

## API Overview

### RegisterAsset
Register a new asset after it's been uploaded to storage:
```go
resp, err := client.RegisterAsset(ctx, &assetv1.RegisterAssetRequest{
    Name:        "ubuntu-22.04-rootfs",
    Type:        assetv1.AssetType_ASSET_TYPE_ROOTFS,
    Backend:     assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
    Location:    "ab/abcd1234...", // Relative path in storage
    SizeBytes:   536870912,
    Checksum:    "sha256:...",
    CreatedBy:   "builderd",
    BuildId:     "build-12345",
    SourceImage: "ubuntu:22.04",
    Labels: map[string]string{
        "os":      "ubuntu",
        "version": "22.04",
    },
})
```

### GetAsset
Retrieve asset information and ensure it's available locally:
```go
resp, err := client.GetAsset(ctx, &assetv1.GetAssetRequest{
    Id:          "01JQKP3X5V2Q8Z9R1N4M7BHCFD",
    EnsureLocal: true, // Download if remote
})
// resp.LocalPath contains the local filesystem path
```

### AcquireAsset / ReleaseAsset
Manage asset references for VMs:
```go
// Acquire reference when creating VM
acquireResp, err := client.AcquireAsset(ctx, &assetv1.AcquireAssetRequest{
    AssetId:    "01JQKP3X5V2Q8Z9R1N4M7BHCFD",
    AcquiredBy: "vm-123",
    TtlSeconds: 3600, // Auto-release after 1 hour
})

// Release when VM is deleted
_, err = client.ReleaseAsset(ctx, &assetv1.ReleaseAssetRequest{
    LeaseId: acquireResp.LeaseId,
})
```

### ListAssets
Query assets with filters:
```go
resp, err := client.ListAssets(ctx, &assetv1.ListAssetsRequest{
    Type:   assetv1.AssetType_ASSET_TYPE_KERNEL,
    Status: assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
    LabelSelector: map[string]string{
        "arch": "amd64",
    },
})
```

### GarbageCollect
Manually trigger garbage collection:
```go
resp, err := client.GarbageCollect(ctx, &assetv1.GarbageCollectRequest{
    MaxAgeSeconds:      86400, // 24 hours
    DeleteUnreferenced: true,
    DryRun:             false,
})
// resp.DeletedAssets contains what was cleaned up
// resp.BytesFreed shows reclaimed space
```

## Integration with Other Services

### Builderd Integration
After creating an image, builderd registers it:
```go
// 1. Builderd extracts rootfs to /opt/vm-assets/
location := storage.Store(buildID, rootfsReader)

// 2. Register with assetmanagerd
asset, err := assetClient.RegisterAsset(ctx, &assetv1.RegisterAssetRequest{
    Name:        fmt.Sprintf("%s-rootfs", imageName),
    Type:        assetv1.AssetType_ASSET_TYPE_ROOTFS,
    Location:    location,
    BuildId:     buildID,
    SourceImage: dockerImage,
})
```

### Metald Integration
When creating a VM, metald:
1. Queries available assets
2. Acquires references to needed assets
3. Gets local paths for jailer preparation
4. Releases references when VM is deleted

```go
// Get available kernels
kernels, _ := assetClient.ListAssets(ctx, &assetv1.ListAssetsRequest{
    Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
})

// Acquire assets for VM
kernelLease, _ := assetClient.AcquireAsset(ctx, &assetv1.AcquireAssetRequest{
    AssetId:    kernelID,
    AcquiredBy: vmID,
})

rootfsLease, _ := assetClient.AcquireAsset(ctx, &assetv1.AcquireAssetRequest{
    AssetId:    rootfsID,
    AcquiredBy: vmID,
})

// Get local paths
kernel, _ := assetClient.GetAsset(ctx, &assetv1.GetAssetRequest{
    Id:          kernelID,
    EnsureLocal: true,
})

// Use kernel.LocalPath for VM configuration
```

## Configuration

See `internal/config/config.go` for all environment variables.

Key settings:
- `UNKEY_ASSETMANAGERD_STORAGE_BACKEND`: Storage backend (local, s3)
- `UNKEY_ASSETMANAGERD_GC_ENABLED`: Enable automatic garbage collection
- `UNKEY_ASSETMANAGERD_GC_INTERVAL`: How often to run GC
- `UNKEY_ASSETMANAGERD_GC_MAX_AGE`: Delete unreferenced assets older than this

## Storage Layout

Local storage uses sharded directories for performance:
```
/opt/vm-assets/
├── 01/
│   └── 01JQKP3X5V2Q8Z9R1N4M7BHCFD  # Asset file
├── 02/
│   └── 02HRMN8Y4P6K1Z3Q5V7W9XBCDG
└── ab/
    └── abcd1234567890...
```

Assets are stored by ID with the first 2 characters used as the directory name.

## Database Schema

SQLite database tracks metadata:
- `assets`: Core asset information
- `asset_labels`: Key-value labels for filtering
- `asset_leases`: Active references with optional TTL

## Future Enhancements

1. **S3 Backend**: Store assets in S3-compatible object storage
2. **Asset Replication**: Replicate popular assets across regions
3. **Compression**: Transparent compression for storage efficiency
4. **Deduplication**: Content-addressable storage for identical assets
5. **Pre-staging**: Proactively stage assets on compute nodes
6. **Metrics**: Asset access patterns, cache hit rates, etc.