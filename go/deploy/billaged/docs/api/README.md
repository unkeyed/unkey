# Billaged API Documentation

This document provides complete reference for the billaged ConnectRPC service, including all endpoints, request/response patterns, and integration examples.

## Service Overview

The BillingService handles VM usage metrics collection, aggregation, and lifecycle management through a ConnectRPC interface with comprehensive error handling and tenant isolation.

**Service Definition**: [billing.proto](../../proto/billing/v1/billing.proto)  
**Generated Client**: [billingv1connect/billing.connect.go](../../gen/billing/v1/billingv1connect/billing.connect.go)  
**Service Implementation**: [service/billing.go](../../internal/service/billing.go)

## Authentication & Authorization

All API calls require SPIFFE/mTLS authentication with tenant-scoped authorization:

- **Authentication**: Bearer token format `dev_user_<user_id>` (development) or JWT (production)
- **Tenant Isolation**: `X-Tenant-ID` header for customer data scoping
- **Transport Security**: SPIFFE workload API for certificate management

Source: [client/client.go:216-244](../../client/client.go#L216-244)

## API Endpoints

### SendMetricsBatch

Processes batches of VM usage metrics from metald instances with real-time aggregation.

**RPC Definition**: [billing.proto:10](../../proto/billing/v1/billing.proto#L10)

#### Request Schema

```protobuf
message SendMetricsBatchRequest {
  string vm_id = 1;           // Unique VM identifier
  string customer_id = 2;     // Customer tenant identifier  
  repeated VMMetrics metrics = 3; // Batch of usage metrics
}

message VMMetrics {
  google.protobuf.Timestamp timestamp = 1;    // Metric collection time
  int64 cpu_time_nanos = 2;                   // CPU time used (cumulative)
  int64 memory_usage_bytes = 3;               // Memory usage (point-in-time)
  int64 disk_read_bytes = 4;                  // Disk read bytes (cumulative)
  int64 disk_write_bytes = 5;                 // Disk write bytes (cumulative)  
  int64 network_rx_bytes = 6;                 // Network receive bytes (cumulative)
  int64 network_tx_bytes = 7;                 // Network transmit bytes (cumulative)
}
```

#### Response Schema

```protobuf
message SendMetricsBatchResponse {
  bool success = 1;    // Processing success indicator
  string message = 2;  // Status message or error details
}
```

#### Implementation Details

The service processes metrics through an in-memory aggregator with delta calculations:

- **Delta Processing**: Handles cumulative counters with overflow protection ([aggregator.go:135-173](../../internal/aggregator/aggregator.go#L135-173))
- **Batch Validation**: Ensures non-empty batches and logs first/last metrics for debugging
- **Metrics Recording**: Updates OpenTelemetry counters for usage processed per customer
- **Performance Tracking**: Records aggregation duration for monitoring

#### Request Example

```bash
curl -X POST https://billaged:8081/billing.v1.BillingService/SendMetricsBatch \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_user_metald_1" \
  -H "X-Tenant-ID: customer-abc" \
  -d '{
    "vm_id": "vm-firecracker-123",
    "customer_id": "customer-abc",
    "metrics": [{
      "timestamp": "2024-01-15T10:30:00Z",
      "cpu_time_nanos": 1500000000,
      "memory_usage_bytes": 536870912,
      "disk_read_bytes": 1048576,
      "disk_write_bytes": 524288,
      "network_rx_bytes": 2048,
      "network_tx_bytes": 1024
    }]
  }'
```

#### Response Example

```json
{
  "success": true,
  "message": "processed 1 metrics"
}
```

#### Error Scenarios

- **Empty Batch**: Returns `success: false` with "no metrics provided" message
- **Processing Failure**: Internal aggregation errors logged with context
- **Authentication Failure**: ConnectRPC authentication errors for invalid tokens

---

### SendHeartbeat

Processes heartbeat signals from metald instances with active VM lists for health monitoring and gap detection.

**RPC Definition**: [billing.proto:11](../../proto/billing/v1/billing.proto#L11)

#### Request Schema

```protobuf
message SendHeartbeatRequest {
  string instance_id = 1;       // Metald instance identifier
  repeated string active_vms = 2; // List of currently active VM IDs
}
```

#### Response Schema

```protobuf
message SendHeartbeatResponse {
  bool success = 1;    // Heartbeat acknowledgment
}
```

#### Implementation Details

Currently handles basic heartbeat acknowledgment with plans for enhanced health checking:

- **Health Monitoring**: Logs active VM counts for operational visibility
- **Gap Detection**: Framework for identifying missing metrics (planned enhancement)
- **Instance Tracking**: Associates VM lists with specific metald instances

Source: [service/billing.go:86-106](../../internal/service/billing.go#L86-106)

#### Request Example

```bash
curl -X POST https://billaged:8081/billing.v1.BillingService/SendHeartbeat \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_user_metald_1" \
  -d '{
    "instance_id": "metald-prod-node-01",
    "active_vms": ["vm-web-1", "vm-db-2", "vm-cache-3"]
  }'
```

#### Response Example

```json
{
  "success": true
}
```

---

### NotifyVmStarted

Handles VM startup notifications to initialize billing tracking and lifecycle management.

**RPC Definition**: [billing.proto:12](../../proto/billing/v1/billing.proto#L12)

#### Request Schema

```protobuf
message NotifyVmStartedRequest {
  string vm_id = 1;        // VM identifier
  string customer_id = 2;  // Customer tenant identifier
  int64 start_time = 3;    // Unix timestamp (nanoseconds)
}
```

#### Response Schema

```protobuf
message NotifyVmStartedResponse {
  bool success = 1;    // Notification acknowledgment
}
```

#### Implementation Details

Initializes VM usage tracking and customer mapping:

- **Usage Initialization**: Creates new VMUsageData structure with start time
- **Customer Mapping**: Adds VM to customer's VM list for tenant isolation
- **Duplicate Handling**: Safely handles multiple start notifications for same VM

Source: [aggregator.go:175-206](../../internal/aggregator/aggregator.go#L175-206)

#### Request Example

```bash
curl -X POST https://billaged:8081/billing.v1.BillingService/NotifyVmStarted \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_user_metald_1" \
  -H "X-Tenant-ID: customer-abc" \
  -d '{
    "vm_id": "vm-firecracker-456",
    "customer_id": "customer-abc", 
    "start_time": 1705315800000000000
  }'
```

---

### NotifyVmStopped

Handles VM shutdown notifications to generate final usage summaries and cleanup tracking data.

**RPC Definition**: [billing.proto:13](../../proto/billing/v1/billing.proto#L13)

#### Request Schema

```protobuf
message NotifyVmStoppedRequest {
  string vm_id = 1;     // VM identifier
  int64 stop_time = 2;  // Unix timestamp (nanoseconds)
}
```

#### Response Schema

```protobuf
message NotifyVmStoppedResponse {
  bool success = 1;    // Notification acknowledgment
}
```

#### Implementation Details

Generates final billing summary and cleans up VM tracking:

- **Final Summary**: Creates complete usage summary for billing systems
- **Resource Cleanup**: Removes VM data from in-memory aggregator
- **Customer Cleanup**: Updates customer-to-VM mappings
- **Usage Callback**: Triggers usage summary callback for external processing

Source: [aggregator.go:208-247](../../internal/aggregator/aggregator.go#L208-247)

#### Request Example

```bash
curl -X POST https://billaged:8081/billing.v1.BillingService/NotifyVmStopped \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_user_metald_1" \
  -d '{
    "vm_id": "vm-firecracker-456",
    "stop_time": 1705319400000000000
  }'
```

---

### NotifyPossibleGap

Handles notifications about potential gaps in metrics reporting for data reconciliation.

**RPC Definition**: [billing.proto:14](../../proto/billing/v1/billing.proto#L14)

#### Request Schema

```protobuf
message NotifyPossibleGapRequest {
  string vm_id = 1;       // VM identifier
  int64 last_sent = 2;    // Last successful metrics timestamp (nanoseconds)
  int64 resume_time = 3;  // Metrics reporting resume time (nanoseconds)
}
```

#### Response Schema

```protobuf
message NotifyPossibleGapResponse {
  bool success = 1;    // Notification acknowledgment
}
```

#### Implementation Details

Logs gap information for operational visibility and future reconciliation:

- **Gap Calculation**: Computes gap duration in milliseconds for monitoring
- **Warning Logging**: Logs gap details with severity appropriate for operations
- **Future Enhancement**: Framework for billing period reconciliation and alerting

Source: [service/billing.go:150-176](../../internal/service/billing.go#L150-176)

#### Request Example

```bash
curl -X POST https://billaged:8081/billing.v1.BillingService/NotifyPossibleGap \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_user_metald_1" \
  -d '{
    "vm_id": "vm-firecracker-456",
    "last_sent": 1705315800000000000,
    "resume_time": 1705316100000000000
  }'
```

## Client Libraries

### Go Client

**Package**: [client/](../../client/)  
**Main Types**: [client/types.go](../../client/types.go)

#### Configuration

```go
config := client.Config{
    ServerAddress:    "https://billaged:8081",
    UserID:          "metald-instance-1",
    TenantID:        "customer-abc",
    TLSMode:         "spiffe",
    SPIFFESocketPath: "/var/lib/spire/agent/agent.sock",
    Timeout:         30 * time.Second,
}

client, err := client.New(ctx, config)
defer client.Close()
```

#### Usage Example

```go
// Send metrics batch
response, err := client.SendMetricsBatch(ctx, &client.SendMetricsBatchRequest{
    VmID:       "vm-12345",
    CustomerID: "customer-abc",
    Metrics:    vmMetrics,
})

// Send heartbeat
response, err := client.SendHeartbeat(ctx, &client.SendHeartbeatRequest{
    InstanceID: "metald-prod-1",
    ActiveVMs:  []string{"vm-1", "vm-2"},
})
```

### CLI Tool

**Binary**: [cmd/billaged-cli/](../../cmd/billaged-cli/)

#### Basic Usage

```bash
# Send metrics batch
billaged-cli -server=https://billaged:8081 -tenant=customer-abc send-metrics

# Send heartbeat
billaged-cli -user=metald-1 -tenant=customer-abc heartbeat

# Notify VM lifecycle
billaged-cli notify-started vm-12345
billaged-cli notify-stopped vm-12345
```

#### Environment Configuration

```bash
export UNKEY_BILLAGED_SERVER_ADDRESS=https://billaged:8081
export UNKEY_BILLAGED_USER_ID=metald-instance-1  
export UNKEY_BILLAGED_TENANT_ID=customer-abc
export UNKEY_BILLAGED_TLS_MODE=spiffe
```

## Error Handling

### ConnectRPC Error Codes

| Code | Description | Common Causes |
|------|-------------|---------------|
| `Unauthenticated` | Authentication failure | Invalid bearer token, missing X-Tenant-ID |
| `InvalidArgument` | Request validation failure | Empty VM ID, invalid timestamps |
| `Internal` | Server processing error | Aggregation failures, OTEL errors |
| `Unavailable` | Service unavailable | SPIFFE certificate issues, resource exhaustion |

### Error Response Format

```json
{
  "code": "InvalidArgument",
  "message": "vm_id cannot be empty",
  "details": [
    {
      "type": "BadRequest",
      "field": "vm_id",
      "description": "VM identifier is required for metrics processing"
    }
  ]
}
```

### Retry Strategies

- **Transient Errors**: Exponential backoff for `Unavailable` and `Internal` errors
- **Authentication Errors**: Immediate failure for `Unauthenticated` errors  
- **Validation Errors**: No retry for `InvalidArgument` errors
- **Timeout Handling**: 30-second default timeout with client-configurable override

## Integration Patterns

### Metald Integration

Primary integration pattern with VM provisioning service:

```go
// metald sends metrics periodically
metrics := collectVMMetrics(vmID)
billaging.SendMetricsBatch(ctx, &billaged.SendMetricsBatchRequest{
    VmID:       vmID,
    CustomerID: customer,
    Metrics:    metrics,
})

// metald notifies lifecycle events  
billaging.NotifyVmStarted(ctx, &billaged.NotifyVmStartedRequest{
    VmID:       vmID,
    CustomerID: customer,
    StartTime:  startTime.UnixNano(),
})
```

### Batch Processing

Optimal batching strategies for high-throughput environments:

- **Batch Size**: 10-100 metrics per batch for optimal performance
- **Frequency**: 30-60 second intervals for real-time billing accuracy
- **Ordering**: Chronological ordering by timestamp for accurate delta calculations

### Monitoring Integration

OpenTelemetry metrics for operational visibility:

- **Usage Tracking**: `billaged_usage_records_processed_total{vm_id,customer_id}`
- **Performance**: `billaged_aggregation_duration_seconds` histogram
- **Health**: `billaged_active_vms` gauge for current tracking count

## Rate Limits & Quotas

### Default Limits

- **Request Rate**: 1000 requests/second per instance
- **Batch Size**: 1000 metrics per SendMetricsBatch request
- **Concurrent Connections**: 100 concurrent client connections
- **Memory Usage**: 1GB for VM usage data aggregation

### Production Scaling

For high-scale deployments:

- **Horizontal Scaling**: Deploy multiple billaged instances behind load balancer
- **Resource Monitoring**: Monitor memory usage for large customer bases
- **Aggregation Tuning**: Adjust interval based on billing precision requirements

Configuration: [config/config.go](../../internal/config/config.go)