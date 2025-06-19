# builderd API Reference

This document provides a complete reference for the builderd gRPC/ConnectRPC API.

## Service Definition

**Service**: [`builder.v1.BuilderService`](../../proto/builder/v1/builder.proto:10-35)

The BuilderService provides multi-tenant build execution for various source types, transforming them into optimized artifacts for deployment.

## Table of Contents

1. [RPC Methods](#rpc-methods)
   - [CreateBuild](#createbuild) - Start a new build job
   - [GetBuild](#getbuild) - Retrieve build status
   - [ListBuilds](#listbuilds) - List builds with filtering
   - [CancelBuild](#cancelbuild) - Cancel running build
   - [DeleteBuild](#deletebuild) - Delete build and artifacts
   - [StreamBuildLogs](#streambuildlogs) - Real-time log streaming
   - [GetTenantQuotas](#gettenantquotas) - Quota information
   - [GetBuildStats](#getbuildstats) - Build statistics
2. [Data Types](#data-types)
3. [Error Handling](#error-handling)
4. [Authentication](#authentication)

## RPC Methods

### CreateBuild

Creates a new build job for the specified tenant and configuration.

**Implementation**: [`internal/service/builder.go:47-111`](../../internal/service/builder.go:47-111)

#### Request
```protobuf
message CreateBuildRequest {
  BuildConfig config = 1;
}
```

#### Response
```protobuf
message CreateBuildResponse {
  string build_id = 1;
  BuildState state = 2;
  google.protobuf.Timestamp created_at = 3;
  string rootfs_path = 4;  // Path to generated rootfs for VM creation
}
```

#### Example
```go
req := &builderv1.CreateBuildRequest{
    Config: &builderv1.BuildConfig{
        Tenant: &builderv1.TenantContext{
            TenantId:   "tenant-123",
            CustomerId: "customer-456",
            Tier:       builderv1.TenantTier_TENANT_TIER_PRO,
        },
        Source: &builderv1.BuildSource{
            SourceType: &builderv1.BuildSource_DockerImage{
                DockerImage: &builderv1.DockerImageSource{
                    ImageUri: "ghcr.io/unkeyed/api:latest",
                    Auth: &builderv1.DockerAuth{
                        Token: "ghp_xxxxxxxxxxxx",
                    },
                },
            },
        },
        Target: &builderv1.BuildTarget{
            TargetType: &builderv1.BuildTarget_MicrovmRootfs{
                MicrovmRootfs: &builderv1.MicroVMRootfs{
                    InitStrategy: builderv1.InitStrategy_INIT_STRATEGY_TINI,
                    RuntimeConfig: &builderv1.RuntimeConfig{
                        Command: []string{"/app/server"},
                        Environment: map[string]string{
                            "PORT": "8080",
                        },
                    },
                    Optimization: &builderv1.OptimizationSettings{
                        StripDebugSymbols: true,
                        RemoveDocs: true,
                        RemoveCache: true,
                    },
                },
            },
        },
        Strategy: &builderv1.BuildStrategy{
            StrategyType: &builderv1.BuildStrategy_DockerExtract{
                DockerExtract: &builderv1.DockerExtractStrategy{
                    FlattenFilesystem: true,
                    ExcludePatterns: []string{
                        "*.log",
                        "/tmp/*",
                    },
                },
            },
        },
        Limits: &builderv1.TenantResourceLimits{
            MaxMemoryBytes: 4 << 30,  // 4GB
            MaxCpuCores: 2,
            TimeoutSeconds: 900,      // 15 minutes
        },
    },
}

resp, err := client.CreateBuild(ctx, connect.NewRequest(req))
```

#### Downstream Calls
- **Docker Registry**: Pulls specified images using Docker client
- **Storage Backend**: Writes rootfs artifacts to configured storage
- **Metrics Service**: Records build metrics via OpenTelemetry

### GetBuild

Retrieves the current status and details of a build job.

**Implementation**: [`internal/service/builder.go:114-149`](../../internal/service/builder.go:114-149)

#### Request
```protobuf
message GetBuildRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
}
```

#### Response
```protobuf
message GetBuildResponse {
  BuildJob build = 1;
}
```

#### Example
```go
req := &builderv1.GetBuildRequest{
    BuildId:  "build-abc123",
    TenantId: "tenant-123",
}

resp, err := client.GetBuild(ctx, connect.NewRequest(req))
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Build state: %s\n", resp.Msg.Build.State)
fmt.Printf("Progress: %d%%\n", resp.Msg.Build.ProgressPercent)
fmt.Printf("Current step: %s\n", resp.Msg.Build.CurrentStep)
```

### ListBuilds

Lists builds for a tenant with optional filtering and pagination.

**Implementation**: [`internal/service/builder.go:152-173`](../../internal/service/builder.go:152-173)

#### Request
```protobuf
message ListBuildsRequest {
  string tenant_id = 1;              // Required for filtering
  repeated BuildState state_filter = 2;
  int32 page_size = 3;
  string page_token = 4;
}
```

#### Response
```protobuf
message ListBuildsResponse {
  repeated BuildJob builds = 1;
  string next_page_token = 2;
  int32 total_count = 3;
}
```

#### Example
```go
req := &builderv1.ListBuildsRequest{
    TenantId: "tenant-123",
    StateFilter: []builderv1.BuildState{
        builderv1.BuildState_BUILD_STATE_COMPLETED,
        builderv1.BuildState_BUILD_STATE_FAILED,
    },
    PageSize: 50,
}

resp, err := client.ListBuilds(ctx, connect.NewRequest(req))
for _, build := range resp.Msg.Builds {
    fmt.Printf("Build %s: %s\n", build.BuildId, build.State)
}
```

### CancelBuild

Cancels a running build job.

**Implementation**: [`internal/service/builder.go:176-200`](../../internal/service/builder.go:176-200)

#### Request
```protobuf
message CancelBuildRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
}
```

#### Response
```protobuf
message CancelBuildResponse {
  bool success = 1;
  BuildState state = 2;
}
```

#### Example
```go
req := &builderv1.CancelBuildRequest{
    BuildId:  "build-abc123",
    TenantId: "tenant-123",
}

resp, err := client.CancelBuild(ctx, connect.NewRequest(req))
if resp.Msg.Success {
    fmt.Println("Build cancelled successfully")
}
```

### DeleteBuild

Deletes a build and its associated artifacts.

**Implementation**: [`internal/service/builder.go:203-223`](../../internal/service/builder.go:203-223)

#### Request
```protobuf
message DeleteBuildRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
  bool force = 3;        // Delete even if running
}
```

#### Response
```protobuf
message DeleteBuildResponse {
  bool success = 1;
}
```

#### Example
```go
req := &builderv1.DeleteBuildRequest{
    BuildId:  "build-abc123",
    TenantId: "tenant-123",
    Force:    false,
}

resp, err := client.DeleteBuild(ctx, connect.NewRequest(req))
```

### StreamBuildLogs

Streams build logs in real-time or retrieves historical logs.

**Implementation**: [`internal/service/builder.go:226-255`](../../internal/service/builder.go:226-255)

#### Request
```protobuf
message StreamBuildLogsRequest {
  string build_id = 1;
  string tenant_id = 2;  // For authorization
  bool follow = 3;       // Continue streaming new logs
}
```

#### Response (Stream)
```protobuf
message StreamBuildLogsResponse {
  google.protobuf.Timestamp timestamp = 1;
  string level = 2;      // "info", "warn", "error", "debug"
  string message = 3;
  string component = 4;  // "puller", "extractor", "builder"
  map<string, string> metadata = 5;
}
```

#### Example
```go
req := &builderv1.StreamBuildLogsRequest{
    BuildId:  "build-abc123",
    TenantId: "tenant-123",
    Follow:   true,
}

stream, err := client.StreamBuildLogs(ctx, connect.NewRequest(req))
if err != nil {
    log.Fatal(err)
}

for stream.Receive() {
    log := stream.Msg()
    fmt.Printf("[%s] %s: %s\n", 
        log.Timestamp.AsTime().Format(time.RFC3339),
        log.Level,
        log.Message,
    )
}
```

### GetTenantQuotas

Retrieves tenant resource quotas and current usage.

**Implementation**: [`internal/service/builder.go:258-296`](../../internal/service/builder.go:258-296)

#### Request
```protobuf
message GetTenantQuotasRequest {
  string tenant_id = 1;
}
```

#### Response
```protobuf
message GetTenantQuotasResponse {
  TenantResourceLimits current_limits = 1;
  TenantUsageStats current_usage = 2;
  repeated QuotaViolation violations = 3;
}
```

#### Example
```go
req := &builderv1.GetTenantQuotasRequest{
    TenantId: "tenant-123",
}

resp, err := client.GetTenantQuotas(ctx, connect.NewRequest(req))
fmt.Printf("Daily builds: %d/%d\n", 
    resp.Msg.CurrentUsage.DailyBuildsUsed,
    resp.Msg.CurrentLimits.MaxDailyBuilds,
)
fmt.Printf("Storage used: %d GB\n", 
    resp.Msg.CurrentUsage.StorageBytesUsed >> 30,
)
```

### GetBuildStats

Retrieves build statistics for a tenant within a time range.

**Implementation**: [`internal/service/builder.go:299-320`](../../internal/service/builder.go:299-320)

#### Request
```protobuf
message GetBuildStatsRequest {
  string tenant_id = 1;
  google.protobuf.Timestamp start_time = 2;
  google.protobuf.Timestamp end_time = 3;
}
```

#### Response
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

#### Example
```go
req := &builderv1.GetBuildStatsRequest{
    TenantId:  "tenant-123",
    StartTime: timestamppb.New(time.Now().AddDate(0, -1, 0)), // Last month
    EndTime:   timestamppb.Now(),
}

resp, err := client.GetBuildStats(ctx, connect.NewRequest(req))
fmt.Printf("Success rate: %.2f%%\n", 
    float64(resp.Msg.SuccessfulBuilds) / float64(resp.Msg.TotalBuilds) * 100,
)
```

## Data Types

### Core Enumerations

#### BuildState
[`proto/builder/v1/builder.proto:38-49`](../../proto/builder/v1/builder.proto:38-49)
```protobuf
enum BuildState {
  BUILD_STATE_UNSPECIFIED = 0;
  BUILD_STATE_PENDING = 1;     // Job queued
  BUILD_STATE_PULLING = 2;     // Pulling Docker image or source
  BUILD_STATE_EXTRACTING = 3;  // Extracting/preparing source
  BUILD_STATE_BUILDING = 4;    // Building rootfs
  BUILD_STATE_OPTIMIZING = 5;  // Applying optimizations
  BUILD_STATE_COMPLETED = 6;   // Build successful
  BUILD_STATE_FAILED = 7;      // Build failed
  BUILD_STATE_CANCELLED = 8;   // Build cancelled
  BUILD_STATE_CLEANING = 9;    // Cleaning up resources
}
```

#### TenantTier
[`proto/builder/v1/builder.proto:52-58`](../../proto/builder/v1/builder.proto:52-58)
```protobuf
enum TenantTier {
  TENANT_TIER_UNSPECIFIED = 0;
  TENANT_TIER_FREE = 1;        // Limited resources
  TENANT_TIER_PRO = 2;         // Standard resources
  TENANT_TIER_ENTERPRISE = 3;  // Higher limits + isolation
  TENANT_TIER_DEDICATED = 4;   // Dedicated infrastructure
}
```

### Core Messages

#### BuildConfig
[`proto/builder/v1/builder.proto:232-251`](../../proto/builder/v1/builder.proto:232-251)
The main build configuration containing tenant context, source, target, strategy, and resource limits.

#### BuildJob
[`proto/builder/v1/builder.proto:295-322`](../../proto/builder/v1/builder.proto:295-322)
Complete build job information including configuration, state, timestamps, results, and metrics.

#### TenantContext
[`proto/builder/v1/builder.proto:69-76`](../../proto/builder/v1/builder.proto:69-76)
Multi-tenant context with tenant ID, customer ID, tier, and permissions.

#### BuildSource
[`proto/builder/v1/builder.proto:79-86`](../../proto/builder/v1/builder.proto:79-86)
Extensible source types including Docker images, Git repositories, and archives.

#### BuildTarget
[`proto/builder/v1/builder.proto:125-131`](../../proto/builder/v1/builder.proto:125-131)
Target artifact types including microVM rootfs and container images.

## Error Handling

builderd uses standard gRPC/Connect error codes with additional context:

### Common Error Codes

- `InvalidArgument` - Invalid build configuration or parameters
- `NotFound` - Build ID not found or tenant doesn't have access
- `PermissionDenied` - Tenant lacks permission for requested operation
- `ResourceExhausted` - Quota exceeded or system resources unavailable
- `FailedPrecondition` - Build in wrong state for operation
- `Cancelled` - Build was cancelled by user
- `DeadlineExceeded` - Build timeout exceeded
- `Internal` - Internal service error

### Error Example
```go
resp, err := client.CreateBuild(ctx, req)
if err != nil {
    if connectErr, ok := err.(*connect.Error); ok {
        switch connectErr.Code() {
        case connect.CodeInvalidArgument:
            fmt.Printf("Invalid config: %s\n", connectErr.Message())
        case connect.CodeResourceExhausted:
            fmt.Printf("Quota exceeded: %s\n", connectErr.Message())
        default:
            fmt.Printf("Build failed: %s\n", connectErr.Message())
        }
    }
}
```

## Authentication

builderd uses multi-tenant authentication enforced by interceptors:

1. **Tenant Context**: All requests must include valid tenant information
2. **SPIFFE/mTLS**: Service-to-service communication uses SPIFFE identities
3. **Authorization**: Tenant access is validated for all resource operations

### Authentication Flow

1. Client includes tenant context in request
2. [`TenantAuthInterceptor`](../../internal/observability/interceptor.go) validates tenant
3. Request is authorized based on tenant permissions
4. Operations are scoped to tenant's resources

### Example with Authentication
```go
// Client setup with SPIFFE
tlsConfig := tlspkg.Config{
    Mode: tlspkg.ModeSPIFFE,
    SPIFFESocketPath: "/run/spire/sockets/agent.sock",
}

httpClient := &http.Client{
    Transport: &http2.Transport{
        TLSClientConfig: tlsConfig,
    },
}

client := builderv1connect.NewBuilderServiceClient(
    httpClient,
    "https://builderd:8082",
)
```

## Metrics and Observability

builderd exports comprehensive metrics via OpenTelemetry:

### Key Metrics

- `builderd_builds_total` - Total builds by tenant, source, target, and state
- `builderd_build_duration_seconds` - Build execution time histogram
- `builderd_concurrent_builds` - Current number of active builds
- `builderd_resource_usage` - CPU, memory, and disk usage per build
- `builderd_quota_usage` - Tenant quota utilization

### Tracing

All RPC methods are traced with OpenTelemetry spans including:
- Tenant context
- Build configuration
- Execution steps
- Error details

AIDEV-NOTE: builderd is designed for future integration with other Unkey Deploy services but currently operates independently. The API is structured to support these integrations when implemented.