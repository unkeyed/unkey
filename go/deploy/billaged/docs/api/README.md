# Billaged API Documentation

The Billaged service exposes a ConnectRPC API for receiving and processing VM usage metrics. All API endpoints use Protocol Buffers for message serialization and support both gRPC and HTTP/JSON protocols.

## Service Definition

**Proto Definition**: [`billing/v1/billing.proto`](../../proto/billing/v1/billing.proto:1-15)

```protobuf
service BillingService {
  rpc SendMetricsBatch(SendMetricsBatchRequest) returns (SendMetricsBatchResponse);
  rpc SendHeartbeat(SendHeartbeatRequest) returns (SendHeartbeatResponse);
  rpc NotifyVmStarted(NotifyVmStartedRequest) returns (NotifyVmStartedResponse);
  rpc NotifyVmStopped(NotifyVmStoppedRequest) returns (NotifyVmStoppedResponse);
  rpc NotifyPossibleGap(NotifyPossibleGapRequest) returns (NotifyPossibleGapResponse);
}
```

## RPC Methods

### SendMetricsBatch

Processes a batch of VM usage metrics from metald instances. This is the primary data ingestion endpoint.

**Implementation**: [`internal/service/billing.go:33-84`](../../internal/service/billing.go:33-84)

#### Request

```protobuf
message SendMetricsBatchRequest {
  string vm_id = 1;
  string customer_id = 2;
  repeated VMMetrics metrics = 3;
}

message VMMetrics {
  google.protobuf.Timestamp timestamp = 1;
  int64 cpu_time_nanos = 2;
  int64 memory_usage_bytes = 3;
  int64 disk_read_bytes = 4;
  int64 disk_write_bytes = 5;
  int64 network_rx_bytes = 6;
  int64 network_tx_bytes = 7;
}
```

#### Response

```protobuf
message SendMetricsBatchResponse {
  bool success = 1;
  string message = 2;
}
```

#### Example

```bash
curl -X POST http://localhost:8081/billing.v1.BillingService/SendMetricsBatch \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "vm-123",
    "customer_id": "customer-456",
    "metrics": [
      {
        "timestamp": "2024-01-01T12:00:00Z",
        "cpu_time_nanos": 1000000000,
        "memory_usage_bytes": 1073741824,
        "disk_read_bytes": 10485760,
        "disk_write_bytes": 5242880,
        "network_rx_bytes": 1048576,
        "network_tx_bytes": 524288
      }
    ]
  }'
```

#### Downstream Processing

1. **Metrics Validation**: Checks for non-empty metrics batch
2. **Aggregator Processing**: [`internal/aggregator/aggregator.go:97-133`](../../internal/aggregator/aggregator.go:97-133)
   - Calculates deltas for cumulative metrics
   - Updates VM usage tracking data
   - Handles counter resets gracefully
3. **Metrics Recording**: Updates OpenTelemetry metrics if enabled

### SendHeartbeat

Receives periodic heartbeats from metald instances with their active VM lists.

**Implementation**: [`internal/service/billing.go:87-106`](../../internal/service/billing.go:87-106)

#### Request

```protobuf
message SendHeartbeatRequest {
  string instance_id = 1;
  repeated string active_vms = 2;
}
```

#### Response

```protobuf
message SendHeartbeatResponse {
  bool success = 1;
}
```

#### Example

```bash
curl -X POST http://localhost:8081/billing.v1.BillingService/SendHeartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "metald-node-1",
    "active_vms": ["vm-123", "vm-456", "vm-789"]
  }'
```

### NotifyVmStarted

Handles VM lifecycle start events. Initializes tracking for new VMs.

**Implementation**: [`internal/service/billing.go:109-128`](../../internal/service/billing.go:109-128)

#### Request

```protobuf
message NotifyVmStartedRequest {
  string vm_id = 1;
  string customer_id = 2;
  int64 start_time = 3;  // Unix timestamp in nanoseconds
}
```

#### Response

```protobuf
message NotifyVmStartedResponse {
  bool success = 1;
}
```

#### Example

```bash
curl -X POST http://localhost:8081/billing.v1.BillingService/NotifyVmStarted \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "vm-123",
    "customer_id": "customer-456",
    "start_time": 1704110400000000000
  }'
```

### NotifyVmStopped

Handles VM lifecycle stop events. Generates final usage summary.

**Implementation**: [`internal/service/billing.go:131-148`](../../internal/service/billing.go:131-148)

#### Request

```protobuf
message NotifyVmStoppedRequest {
  string vm_id = 1;
  int64 stop_time = 2;  // Unix timestamp in nanoseconds
}
```

#### Response

```protobuf
message NotifyVmStoppedResponse {
  bool success = 1;
}
```

#### Example

```bash
curl -X POST http://localhost:8081/billing.v1.BillingService/NotifyVmStopped \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "vm-123",
    "stop_time": 1704114000000000000
  }'
```

#### Downstream Processing

1. **Final Summary Generation**: [`internal/aggregator/aggregator.go:209-247`](../../internal/aggregator/aggregator.go:209-247)
2. **Callback Execution**: Triggers usage summary callback with final metrics
3. **Cleanup**: Removes VM from active tracking

### NotifyPossibleGap

Handles data gap notifications for billing accuracy.

**Implementation**: [`internal/service/billing.go:151-176`](../../internal/service/billing.go:151-176)

#### Request

```protobuf
message NotifyPossibleGapRequest {
  string vm_id = 1;
  int64 last_sent = 2;   // Last successful metric timestamp
  int64 resume_time = 3; // When metrics resumed
}
```

#### Response

```protobuf
message NotifyPossibleGapResponse {
  bool success = 1;
}
```

#### Example

```bash
curl -X POST http://localhost:8081/billing.v1.BillingService/NotifyPossibleGap \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "vm-123",
    "last_sent": 1704110400000000000,
    "resume_time": 1704111000000000000
  }'
```

## HTTP Endpoints

### GET /stats

Returns current aggregation statistics.

**Implementation**: [`cmd/billaged/main.go:218-229`](../../cmd/billaged/main.go:218-229)

#### Response

```json
{
  "active_vms": 42,
  "aggregation_interval": "60s"
}
```

### GET /metrics

Prometheus metrics endpoint (when OpenTelemetry is enabled).

**Port**: 9465 (configurable)

#### Available Metrics

- `billaged_usage_records_processed_total{vm_id,customer_id}` - Usage records processed
- `billaged_aggregation_duration_seconds` - Aggregation latency histogram
- `billaged_active_vms` - Currently tracked VMs
- `billaged_billing_errors_total{error_type}` - Processing errors

### GET /health

Health check endpoint provided by the health package.

**Implementation**: Uses [`pkg/health`](../../cmd/billaged/main.go:284)

#### Response

```json
{
  "status": "healthy",
  "service": "billaged",
  "version": "0.1.0",
  "uptime": "1h30m45s"
}
```

## Error Handling

All RPC methods return standard ConnectRPC errors:

- `CodeInvalidArgument`: Invalid request parameters
- `CodeInternal`: Internal processing errors
- `CodeUnavailable`: Service temporarily unavailable

Errors are logged with structured fields for debugging:

```go
logger.Error("rpc error",
  slog.String("procedure", req.Spec().Procedure),
  slog.Duration("duration", duration),
  slog.String("error", err.Error()),
)
```

## Authentication

Currently, the service does not implement authentication. In production deployments:

1. Use SPIFFE/mTLS for service-to-service authentication
2. Deploy behind an API gateway for external access
3. Implement customer isolation at the metald level

## Rate Limiting

No built-in rate limiting. Recommended approach:

1. Configure rate limits at the load balancer/proxy level
2. Monitor `billaged_usage_records_processed_total` for anomalies
3. Set appropriate batch size limits in metald

## Protocol Support

The service supports both gRPC and HTTP/JSON through ConnectRPC:

- **gRPC**: Native protocol buffer encoding
- **HTTP/JSON**: Automatic JSON transcoding
- **gRPC-Web**: Browser-compatible protocol

All protocols are served on the same port (8081 by default).