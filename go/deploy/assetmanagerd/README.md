# AssetManagerd

**Purpose**: Centralized asset management system for VM-related resources across the Unkey Deploy infrastructure.

**Architecture**: Service component providing asset registry, lifecycle management, and distribution for VM kernels, rootfs images, initrd, and disk images.

**Dependencies**: 
- [metald](../docs/metald/) - Primary consumer for VM asset preparation
- [builderd](../docs/builderd/) // AIDEV: builderd documentation needed for complete interaction description
- [SPIFFE/SPIRE](https://spiffe.io/) - mTLS authentication

**Deployment**: Deployed as a systemd service on VM host nodes, uses local SQLite for metadata and pluggable storage backends for assets.

## Quick Links

- [**API Documentation**](docs/api/) - Complete RPC reference with examples
- [**Architecture Guide**](docs/architecture/) - Service design and interactions  
- [**Operations Manual**](docs/operations/) - Metrics, monitoring, and deployment
- [**Development Setup**](docs/development/) - Build instructions and local development

## Overview

AssetManagerd is one of the four pillar services in Unkey Deploy, responsible for managing VM-related assets throughout their lifecycle. It provides:

- **Asset Registry**: Centralized tracking of kernels, rootfs images, initrd, and disk images
- **Lifecycle Management**: Reference counting, leasing, and automated garbage collection
- **Storage Abstraction**: Pluggable backend support (local, S3, HTTP, NFS)
- **Asset Distribution**: Efficient preparation and staging for VM deployment

## Key Features

### Asset Types
- `KERNEL` - Linux kernel images
- `ROOTFS` - Root filesystem images  
- `INITRD` - Initial ramdisk images
- `DISK_IMAGE` - Persistent disk images

### Core Capabilities
- **Reference Counting**: Track asset usage across VMs
- **Lease Management**: Time-based acquisition with automatic expiration
- **Garbage Collection**: Automated cleanup of unused assets
- **Content Deduplication**: SHA256-based duplicate detection
- **Multi-tenant Support**: Label-based asset filtering and isolation

## Service Endpoints

- **ConnectRPC API**: Port 8083 (configurable via `UNKEY_ASSETMANAGERD_PORT`)
- **Metrics**: Port 9467 (Prometheus format)
- **Health Check**: `/grpc.health.v1.Health/Check`

## Configuration

Key environment variables:

```bash
# Service configuration
UNKEY_ASSETMANAGERD_PORT=8083
UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local
UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/opt/vm-assets
UNKEY_ASSETMANAGERD_DATABASE_PATH=/opt/assetmanagerd/assets.db

# Garbage collection
UNKEY_ASSETMANAGERD_GC_ENABLED=true
UNKEY_ASSETMANAGERD_GC_INTERVAL=3600s
UNKEY_ASSETMANAGERD_GC_AGE_THRESHOLD=7d

# Security
UNKEY_ASSETMANAGERD_TLS_MODE=spiffe
```

## Integration Examples

### Registering an Asset (builderd)

```go
client := assetv1connect.NewAssetServiceClient(httpClient, "https://assetmanagerd:8083")

resp, err := client.RegisterAsset(ctx, &assetv1.RegisterAssetRequest{
    Name: "ubuntu-24.04-kernel",
    Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
    Location: "/builds/kernels/ubuntu-24.04.kernel",
    SizeBytes: 12345678,
    Checksum: "sha256:abcdef...",
    Labels: map[string]string{
        "os": "ubuntu",
        "version": "24.04",
    },
})
```

### Preparing Assets for VM (metald)

```go
resp, err := client.PrepareAssets(ctx, &assetv1.PrepareAssetsRequest{
    Assets: []*assetv1.AssetReference{
        {AssetId: "kernel-123", TargetPath: "/jailer/vm-456/kernel"},
        {AssetId: "rootfs-789", TargetPath: "/jailer/vm-456/rootfs.ext4"},
    },
})
```

## Storage Architecture

Assets are stored using a sharded directory structure to prevent filesystem performance degradation:

```
/opt/vm-assets/
├── ab/
│   └── abcdef123456.kernel
├── cd/
│   └── cdef567890ab.rootfs
└── ef/
    └── ef1234567890.initrd
```

## Version

Current version: v0.1.0 ([cmd/assetmanagerd/main.go:21](../assetmanagerd/cmd/assetmanagerd/main.go:21))

## Related Documentation

- [Unkey Deploy Architecture](../docs/architecture-overview.md)
- [Pillar Services Overview](../docs/PILLAR_SERVICES.md)
- [Service Interactions](../docs/service-interactions-detailed.md)