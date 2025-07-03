# Builderd API Documentation

This document provides complete reference for the builderd ConnectRPC API, including all service endpoints, request/response schemas, and integration patterns.

## Service Overview

Builderd exposes a single ConnectRPC service `BuilderService` that handles multi-tenant build operations with comprehensive lifecycle management.

**Service Definition**: [proto/builder/v1/builder.proto](../../proto/builder/v1/builder.proto)

**Generated Client**: [gen/builder/v1/builderv1connect/builder.connect.go](../../gen/builder/v1/builderv1connect/builder.connect.go)

## Authentication

All API calls require SPIFFE/SPIRE mTLS authentication with tenant context:

```go
// Tenant headers required for all requests
X-Tenant-ID: <tenant-id>
X-Customer-ID: <customer-id>  // Optional for enterprise tiers
Authorization: Bearer <jwt-token>
```

Source: [internal/service/builder.go:136](../../internal/service/builder.go#L136)

## API Endpoints

### CreateBuild

Creates a new build job with tenant-scoped resource validation.

**Endpoint**: `/builder.v1.BuilderService/CreateBuild`

**Request Schema**: 
```protobuf
message CreateBuildRequest {
  BuildConfig config = 1;
}
```

**Response Schema**:
```protobuf
message CreateBuildResponse {
  string build_id = 1;
  BuildState state = 2;
  google.protobuf.Timestamp created_at = 3;
  string rootfs_path = 4;
}
```

**Source Types Supported**:
- **Docker Images**: `ghcr.io/unkeyed/unkey:latest` with optional registry authentication
- **Git Repositories**: GitHub/GitLab URLs with branch/tag/commit selection (planned)
- **Archives**: tar.gz/zip files with build context specification (planned)

**Example Request**:
```json
{
  "config": {
    "tenant": {
      "tenant_id": "tenant-123",
      "customer_id": "customer-456",
      "tier": "TENANT_TIER_PRO"
    },
    "source": {
      "docker_image": {
        "image_uri": "ghcr.io/unkeyed/unkey:f4cfee5"
      }
    },
    "target": {
      "microvm_rootfs": {
        "init_strategy": "INIT_STRATEGY_TINI"
      }
    },
    "strategy": {
      "docker_extract": {
        "flatten_filesystem": true
      }
    },
    "suggested_asset_id": "asset-789"
  }
}
```

**Implementation**: [internal/service/builder.go:132](../../internal/service/builder.go#L132)

### GetBuild

Retrieves build status and progress information with tenant authorization.

**Endpoint**: `/builder.v1.BuilderService/GetBuild`

**Request Schema**:
```protobuf
message GetBuildRequest {
  string build_id = 1;
  string tenant_id = 2;
}
```

**Response Schema**:
```protobuf
message GetBuildResponse {
  BuildJob build = 1;
}
```

**Build States**:
- `BUILD_STATE_PENDING` - Job queued
- `BUILD_STATE_PULLING` - Pulling Docker image/source
- `BUILD_STATE_EXTRACTING` - Extracting/preparing source
- `BUILD_STATE_BUILDING` - Building rootfs
- `BUILD_STATE_OPTIMIZING` - Applying optimizations
- `BUILD_STATE_COMPLETED` - Build successful
- `BUILD_STATE_FAILED` - Build failed
- `BUILD_STATE_CANCELLED` - Build cancelled

**Implementation**: [internal/service/builder.go:368](../../internal/service/builder.go#L368)

### ListBuilds

Lists builds for a tenant with filtering and pagination support.

**Endpoint**: `/builder.v1.BuilderService/ListBuilds`

**Request Schema**:
```protobuf
message ListBuildsRequest {
  string tenant_id = 1;
  repeated BuildState state_filter = 2;
  int32 page_size = 3;
  string page_token = 4;
}
```

**Implementation**: [internal/service/builder.go:397](../../internal/service/builder.go#L397)

### CancelBuild

Cancels a running build with proper cleanup and metrics recording.

**Endpoint**: `/builder.v1.BuilderService/CancelBuild`

**Implementation**: [internal/service/builder.go:421](../../internal/service/builder.go#L421)

### DeleteBuild

Deletes a build and its artifacts with optional force flag.

**Endpoint**: `/builder.v1.BuilderService/DeleteBuild`

**Implementation**: [internal/service/builder.go:448](../../internal/service/builder.go#L448)

### StreamBuildLogs

Streams build logs in real-time with optional follow mode.

**Endpoint**: `/builder.v1.BuilderService/StreamBuildLogs`

**Stream Response**:
```protobuf
message StreamBuildLogsResponse {
  google.protobuf.Timestamp timestamp = 1;
  string level = 2;
  string message = 3;
  string component = 4;
  map<string, string> metadata = 5;
}
```

**Implementation**: [internal/service/builder.go:471](../../internal/service/builder.go#L471)

### GetTenantQuotas

Retrieves tenant quota information and current usage statistics.

**Endpoint**: `/builder.v1.BuilderService/GetTenantQuotas`

**Response includes**:
- Current resource limits per tenant tier
- Usage statistics (active builds, storage used, etc.)
- Quota violations and warnings

**Implementation**: [internal/service/builder.go:503](../../internal/service/builder.go#L503)

### GetBuildStats

Retrieves build statistics and analytics for a tenant.

**Endpoint**: `/builder.v1.BuilderService/GetBuildStats`

**Implementation**: [internal/service/builder.go:544](../../internal/service/builder.go#L544)

## Error Handling

The API returns standard ConnectRPC error codes:

- `InvalidArgument` - Malformed request or validation failure
- `NotFound` - Build ID not found or not accessible to tenant
- `PermissionDenied` - Insufficient tenant permissions
- `ResourceExhausted` - Quota limits exceeded
- `Internal` - Server-side processing error

**Error Validation**: [internal/service/builder.go:568](../../internal/service/builder.go#L568)

## Service Interactions

### Downstream Calls to assetmanagerd

After successful builds, builderd automatically registers artifacts:

```go
// Asset registration with tenant context
assetID, err := s.assetClient.RegisterBuildArtifactWithID(
    ctx, 
    buildID, 
    rootfsPath, 
    assetv1.AssetType_ASSET_TYPE_ROOTFS, 
    labels, 
    suggestedAssetID
)
```

**Source**: [internal/service/builder.go:280](../../internal/service/builder.go#L280)

**Client Implementation**: [internal/assetmanager/client.go:72](../../internal/assetmanager/client.go#L72)

## Configuration Schema

### Tenant Resource Limits

```protobuf
message TenantResourceLimits {
  int64 max_memory_bytes = 1;
  int32 max_cpu_cores = 2;
  int64 max_disk_bytes = 3;
  int32 timeout_seconds = 4;
  int32 max_concurrent_builds = 5;
  int32 max_daily_builds = 6;
  int64 max_storage_bytes = 7;
  int32 max_build_time_minutes = 8;
  repeated string allowed_registries = 9;
  repeated string allowed_git_hosts = 10;
  bool allow_external_network = 11;
}
```

### Build Strategies

**Docker Extract Strategy** (Currently Implemented):
```protobuf
message DockerExtractStrategy {
  bool preserve_layers = 1;
  bool flatten_filesystem = 2;
  repeated string exclude_patterns = 3;
}
```

**Future Strategies**:
- `GoApiStrategy` - Go application builds
- `SinatraStrategy` - Ruby/Sinatra applications  
- `NodejsStrategy` - Node.js applications

## Integration Examples

### Go Client Usage

```go
package main

import (
    "context"
    "github.com/unkeyed/unkey/go/deploy/builderd/client"
    builderv1 "github.com/unkeyed/unkey/go/deploy/builderd/gen/builder/v1"
)

func main() {
    // Create client with SPIFFE authentication
    builderClient, err := client.New(ctx, client.Config{
        ServerAddress: "https://builderd:8082",
        TenantID:     "tenant-123",
        UserID:       "user-456",
        TLSMode:      "spiffe",
    })
    if err != nil {
        panic(err)
    }
    defer builderClient.Close()

    // Create build request
    resp, err := builderClient.CreateBuild(ctx, &client.CreateBuildRequest{
        Config: &builderv1.BuildConfig{
            Tenant: &builderv1.TenantContext{
                TenantId:   "tenant-123",
                CustomerId: "customer-456",
                Tier:       builderv1.TenantTier_TENANT_TIER_PRO,
            },
            Source: &builderv1.BuildSource{
                SourceType: &builderv1.BuildSource_DockerImage{
                    DockerImage: &builderv1.DockerImageSource{
                        ImageUri: "ghcr.io/unkeyed/unkey:latest",
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
    })
}
```

**Client Implementation**: [client/client.go](../../client/client.go)

### HTTP/ConnectRPC Integration

```bash
# Create build via HTTP
curl -X POST https://builderd:8082/builder.v1.BuilderService/CreateBuild \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -H "Authorization: Bearer <jwt-token>" \
  -d '{
    "config": {
      "tenant": {"tenant_id": "tenant-123", "tier": "TENANT_TIER_PRO"},
      "source": {"docker_image": {"image_uri": "alpine:latest"}},
      "target": {"microvm_rootfs": {"init_strategy": "INIT_STRATEGY_TINI"}},
      "strategy": {"docker_extract": {"flatten_filesystem": true}}
    }
  }'
```

## Metrics and Observability

The API automatically records metrics for all operations:

- `builderd_builds_total` - Total builds by tenant and type
- `builderd_build_duration_seconds` - Build completion time
- `builderd_build_errors_total` - Build failures by error type
- `builderd_active_builds` - Currently running builds

**Metrics Implementation**: [internal/observability/metrics.go](../../internal/observability/metrics.go)

## Rate Limiting

Health endpoint includes rate limiting:
- Default: 100 requests/second with 10-request burst
- Configurable via `UNKEY_BUILDERD_RATE_LIMIT`

**Implementation**: [cmd/builderd/main.go:474](../../cmd/builderd/main.go#L474)