# Builderd API Documentation

This document provides comprehensive API reference for the Builderd service ConnectRPC/gRPC interface.

## Service Definition

The BuilderService provides multi-tenant build execution for various source types. All operations are tenant-scoped and subject to resource quotas.

**Proto Definition**: [`proto/builder/v1/builder.proto`](../../proto/builder/v1/builder.proto)

## Base URL

- **gRPC/ConnectRPC**: `https://localhost:8082/builder.v1.BuilderService/`
- **Health Check**: `https://localhost:8082/health`

## Authentication

All API calls require tenant authentication via SPIFFE/mTLS or custom tenant context headers.

## API Documentation

### BuilderService

Service definition: [builder.proto:10](../../../proto/builder/v1/builder.proto:10)

The BuilderService provides multi-tenant build execution for various source types including Docker images, Git repositories, and archives.

### RPCs

#### CreateBuild

**Method**: `rpc CreateBuild(CreateBuildRequest) returns (CreateBuildResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:12)

Creates and executes a new build job. Currently executes synchronously but will support async execution in future.

**Request Schema** ([CreateBuildRequest](../../../proto/builder/v1/builder.proto:352)):
```protobuf
message CreateBuildRequest {
  BuildConfig config = 1;  // Complete build configuration
}
```

**Response Schema** ([CreateBuildResponse](../../../proto/builder/v1/builder.proto:354)):
```protobuf
message CreateBuildResponse {
  string build_id = 1;              // Unique build identifier
  BuildState state = 2;             // Current build state
  google.protobuf.Timestamp created_at = 3;
  string rootfs_path = 4;           // Path to generated rootfs
}
```

**Downstream Calls**: 
- Executes Docker commands locally for image extraction
- Registers artifacts with assetmanagerd upon successful completion ([client.go:63](../../../internal/assetmanager/client.go:63))

**Error Handling**:
- `INVALID_ARGUMENT`: Invalid build configuration
- `RESOURCE_EXHAUSTED`: Quota limits exceeded
- `INTERNAL`: Build execution failures

**Example**:
```json
{
  "config": {
    "tenant": {
      "tenant_id": "tenant-123",
      "customer_id": "cust-456",
      "tier": "TENANT_TIER_PRO"
    },
    "source": {
      "docker_image": {
        "image_uri": "ghcr.io/unkeyed/api:latest"
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
    }
  }
}
```

#### GetBuild

**Method**: `rpc GetBuild(GetBuildRequest) returns (GetBuildResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:15)

Retrieves build status and detailed information.

**Request Schema** ([GetBuildRequest](../../../proto/builder/v1/builder.proto:361)):
```protobuf
message GetBuildRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
}
```

**Response Schema** ([GetBuildResponse](../../../proto/builder/v1/builder.proto:366)):
```protobuf
message GetBuildResponse {
  BuildJob build = 1;  // Complete build information
}
```

**Error Handling**:
- `NOT_FOUND`: Build not found
- `PERMISSION_DENIED`: Tenant lacks access to build

#### ListBuilds

**Method**: `rpc ListBuilds(ListBuildsRequest) returns (ListBuildsResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:18)

Lists builds for a tenant with optional filtering.

**Request Schema** ([ListBuildsRequest](../../../proto/builder/v1/builder.proto:368)):
```protobuf
message ListBuildsRequest {
  string tenant_id = 1;                    // Required for filtering
  repeated BuildState state_filter = 2;    // Optional state filters
  int32 page_size = 3;                     // Pagination size
  string page_token = 4;                   // Pagination token
}
```

**Response Schema** ([ListBuildsResponse](../../../proto/builder/v1/builder.proto:375)):
```protobuf
message ListBuildsResponse {
  repeated BuildJob builds = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

#### CancelBuild

**Method**: `rpc CancelBuild(CancelBuildRequest) returns (CancelBuildResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:21)

Cancels a running build job.

**Request Schema** ([CancelBuildRequest](../../../proto/builder/v1/builder.proto:381)):
```protobuf
message CancelBuildRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
}
```

**Response Schema** ([CancelBuildResponse](../../../proto/builder/v1/builder.proto:386)):
```protobuf
message CancelBuildResponse {
  bool success = 1;
  BuildState state = 2;
}
```

#### DeleteBuild

**Method**: `rpc DeleteBuild(DeleteBuildRequest) returns (DeleteBuildResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:24)

Deletes a build and its artifacts.

**Request Schema** ([DeleteBuildRequest](../../../proto/builder/v1/builder.proto:391)):
```protobuf
message DeleteBuildRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
  bool force = 3;        // Delete even if running
}
```

**Response Schema** ([DeleteBuildResponse](../../../proto/builder/v1/builder.proto:397)):
```protobuf
message DeleteBuildResponse {
  bool success = 1;
}
```

#### StreamBuildLogs

**Method**: `rpc StreamBuildLogs(StreamBuildLogsRequest) returns (stream StreamBuildLogsResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:27)

Streams build logs in real-time.

**Request Schema** ([StreamBuildLogsRequest](../../../proto/builder/v1/builder.proto:399)):
```protobuf
message StreamBuildLogsRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
  bool follow = 3;       // Continue streaming new logs
}
```

**Response Schema** ([StreamBuildLogsResponse](../../../proto/builder/v1/builder.proto:325)):
```protobuf
message StreamBuildLogsResponse {
  google.protobuf.Timestamp timestamp = 1;
  string level = 2;      // "info", "warn", "error", "debug"
  string message = 3;
  string component = 4;  // "puller", "extractor", "builder"
  map<string, string> metadata = 5;
}
```

#### GetTenantQuotas

**Method**: `rpc GetTenantQuotas(GetTenantQuotasRequest) returns (GetTenantQuotasResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:31)

Retrieves tenant quota information and current usage.

**Request Schema** ([GetTenantQuotasRequest](../../../proto/builder/v1/builder.proto:405)):
```protobuf
message GetTenantQuotasRequest {
  string tenant_id = 1;
}
```

**Response Schema** ([GetTenantQuotasResponse](../../../proto/builder/v1/builder.proto:407)):
```protobuf
message GetTenantQuotasResponse {
  TenantResourceLimits current_limits = 1;
  TenantUsageStats current_usage = 2;
  repeated QuotaViolation violations = 3;
}
```

#### GetBuildStats

**Method**: `rpc GetBuildStats(GetBuildStatsRequest) returns (GetBuildStatsResponse)`  
[Proto definition](../../../proto/builder/v1/builder.proto:34)

Retrieves build statistics for a tenant.

**Request Schema** ([GetBuildStatsRequest](../../../proto/builder/v1/builder.proto:414)):
```protobuf
message GetBuildStatsRequest {
  string tenant_id = 1;
  google.protobuf.Timestamp start_time = 2;
  google.protobuf.Timestamp end_time = 3;
}
```

**Response Schema** ([GetBuildStatsResponse](../../../proto/builder/v1/builder.proto:420)):
```protobuf
message GetBuildStatsResponse {
  int32 total_builds = 1;
  int32 successful_builds = 2;
  int32 failed_builds = 3;
  int64 avg_build_time_ms = 4;
  int64 total_storage_bytes = 5;
  int64 total_compute_minutes = 6;
  repeated BuildJob recent_builds = 7;
}
```
## Service Interactions

### Outbound Calls

1. **assetmanagerd** ([client implementation](../../../internal/assetmanager/client.go:63))
   - `RegisterAsset`: Called after successful builds to register rootfs artifacts
   - Includes build metadata and tenant labels for tracking

2. **Docker daemon** (local execution)
   - `docker pull`: Fetches images from registries
   - `docker save`: Exports image layers
   - `docker inspect`: Retrieves image metadata

### Inbound Calls

Currently, builderd is called directly by clients. Future integrations:
- **metald** will call CreateBuild when provisioning new VMs
- **API gateway** will proxy build requests from external clients

### Data Flow

1. Client submits build request with source and target configuration
2. Builderd validates tenant quotas and permissions
3. Build executor pulls source (e.g., Docker image)
4. Rootfs is extracted and optimized based on target settings
5. Artifact is stored in configured storage backend
6. Successful builds are registered with assetmanagerd
7. Build metadata and logs are persisted for retrieval

### Authentication

- Service-to-service: SPIFFE/mTLS authentication required
- Tenant context: Extracted from request headers by interceptor ([interceptor.go](../../../internal/observability/interceptor.go))
- Authorization: Tenant ID validated against resource ownership

## Configuration

### Environment Variables

All configuration follows the `UNKEY_BUILDERD_*` pattern. Key variables:

**Server Configuration**:
- `UNKEY_BUILDERD_PORT`: Service port (default: 8082)
- `UNKEY_BUILDERD_ADDRESS`: Bind address (default: 0.0.0.0)
- `UNKEY_BUILDERD_SHUTDOWN_TIMEOUT`: Graceful shutdown timeout (default: 15s)

**Build Configuration**:
- `UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS`: Max parallel builds (default: 5)
- `UNKEY_BUILDERD_BUILD_TIMEOUT`: Build timeout (default: 15m)
- `UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR`: Output directory (default: /opt/builderd/rootfs)

**Service Endpoints**:
- `UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT`: AssetManagerd URL (default: https://localhost:8083)
- `UNKEY_BUILDERD_ASSETMANAGER_ENABLED`: Enable integration (default: true)

**Tenant Defaults**:
- `UNKEY_BUILDERD_TENANT_DEFAULT_MAX_MEMORY_BYTES`: Memory limit (default: 2GB)
- `UNKEY_BUILDERD_TENANT_DEFAULT_MAX_CPU_CORES`: CPU cores (default: 2)
- `UNKEY_BUILDERD_TENANT_DEFAULT_MAX_DAILY_BUILDS`: Daily quota (default: 100)

See [config.go](../../../internal/config/config.go:154) for complete configuration options.

### Feature Flags

- `UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED`: Enable full tenant isolation (default: true)
- `UNKEY_BUILDERD_STORAGE_CACHE_ENABLED`: Enable build caching (default: true)
- `UNKEY_BUILDERD_OTEL_ENABLED`: Enable OpenTelemetry (default: false)

### Circuit Breakers

Currently not implemented. Future versions will include:
- Registry connection circuit breakers
- Storage backend circuit breakers
- Per-tenant rate limiting

## Operations

### Metrics

When OpenTelemetry is enabled, the following metrics are exported:

**Build Metrics** ([metrics.go](../../../internal/observability/metrics.go)):
- `builderd_builds_total`: Total builds by state, source, and tenant
- `builderd_build_duration_seconds`: Build execution time histogram
- `builderd_concurrent_builds`: Current number of running builds
- `builderd_build_size_bytes`: Rootfs size distribution

**Dependency Health**:
- `builderd_assetmanager_calls_total`: Calls to assetmanagerd
- `builderd_assetmanager_errors_total`: Failed assetmanagerd calls
- `builderd_docker_operations_total`: Docker operation counts

### Health Checks

**Endpoint**: `/health`  
**Implementation**: [main.go:314](../../../cmd/builderd/main.go:314)

Returns service health status including:
- Service uptime
- Build queue depth
- Storage availability
- Dependency connectivity

### Logging

Structured JSON logging via slog with fields:
- `tenant_id`: Tenant identifier
- `build_id`: Build job ID
- `source_type`: Build source (docker, git, etc.)
- `duration`: Operation duration
- `error`: Error details

### Debugging

**Debug Endpoints**: Not currently implemented

**Common Issues**:
1. **Storage Full**: Check `UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR` disk space
2. **Docker Errors**: Verify Docker daemon is running and accessible
3. **Quota Exceeded**: Check tenant limits via GetTenantQuotas
4. **Asset Registration Failed**: Verify assetmanagerd connectivity

## Development

### Build Instructions

```bash
cd builderd
make build  # Build the binary
make install  # Install with systemd unit
```

### Testing

Unit tests: [service/builder_test.go](../../../internal/service/builder_test.go) (to be implemented)
Integration tests require Docker daemon access

### Local Development

1. Start dependencies:
   ```bash
   docker run -d --name spire-agent spiffe/spire-agent:latest
   docker run -d --name assetmanagerd assetmanagerd:latest
   ```

2. Run builderd:
   ```bash
   UNKEY_BUILDERD_TLS_MODE=disabled \
   UNKEY_BUILDERD_ASSETMANAGER_ENABLED=false \
   ./build/builderd
   ```

3. Test with grpcurl:
   ```bash
   grpcurl -plaintext localhost:8082 builder.v1.BuilderService/GetTenantQuotas
   ```