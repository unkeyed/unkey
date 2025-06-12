# Billaged API Reference

ConnectRPC-based API for high-precision VM usage billing and metrics aggregation.

## Overview

Billaged provides a high-performance billing service that collects, aggregates, and stores VM resource usage metrics with nanosecond precision. It receives metrics from multiple metald instances and provides accurate usage accounting for customer billing.

## Base URL
- HTTP: `http://localhost:8081`
- gRPC: `localhost:8081`

## Service Endpoints

### SendMetricsBatch
Processes a batch of VM metrics from metald instances.

**Endpoint**: `POST /billing.v1.BillingService/SendMetricsBatch`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef",
  "customer_id": "customer-456",
  "metrics": [
    {
      "timestamp": "2024-01-15T10:30:00.100Z",
      "cpu_time_nanos": 1234567890123,
      "memory_usage_bytes": 536870912,
      "disk_read_bytes": 1048576,
      "disk_write_bytes": 2097152,
      "network_rx_bytes": 4194304,
      "network_tx_bytes": 8388608
    }
  ]
}
```

**Response**:
```json
{
  "success": true,
  "message": "processed 600 metrics"
}
```

**Field Descriptions**:

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `vm_id` | `string` | Unique VM identifier | `"ud-1234567890abcdef"` |
| `customer_id` | `string` | Customer identifier for billing | `"customer-456"` |
| `metrics` | `array[VMMetrics]` | Array of metrics samples | See VMMetrics below |

### VMMetrics Structure

| Field | Type | Description | Precision |
|-------|------|-------------|-----------|
| `timestamp` | `google.protobuf.Timestamp` | Collection time | Nanosecond |
| `cpu_time_nanos` | `int64` | Cumulative CPU time | Nanoseconds |
| `memory_usage_bytes` | `int64` | Current memory usage | Bytes |
| `disk_read_bytes` | `int64` | Cumulative disk reads | Bytes |
| `disk_write_bytes` | `int64` | Cumulative disk writes | Bytes |
| `network_rx_bytes` | `int64` | Cumulative network received | Bytes |
| `network_tx_bytes` | `int64` | Cumulative network transmitted | Bytes |

### SendHeartbeat
Processes heartbeat from metald with list of active VMs.

**Endpoint**: `POST /billing.v1.BillingService/SendHeartbeat`

**Request**:
```json
{
  "instance_id": "metald-us-east-1a",
  "active_vms": [
    "ud-1234567890abcdef",
    "ud-abcdef1234567890",
    "ud-fedcba0987654321"
  ]
}
```

**Response**:
```json
{
  "success": true
}
```

**Purpose**:
- Health monitoring of metald instances
- Active VM tracking for reconciliation
- Gap detection in metrics collection
- Distributed system coordination

### NotifyVmStarted
Notifies billing service when a VM starts.

**Endpoint**: `POST /billing.v1.BillingService/NotifyVmStarted`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef",
  "customer_id": "customer-456",
  "start_time": 1705317000000000000
}
```

**Response**:
```json
{
  "success": true
}
```

**Notes**:
- `start_time` is Unix timestamp in nanoseconds
- Creates billing session for the VM
- Initiates usage tracking

### NotifyVmStopped
Notifies billing service when a VM stops.

**Endpoint**: `POST /billing.v1.BillingService/NotifyVmStopped`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef",
  "stop_time": 1705320600000000000
}
```

**Response**:
```json
{
  "success": true
}
```

**Notes**:
- `stop_time` is Unix timestamp in nanoseconds
- Finalizes billing session
- Triggers final usage calculation

### NotifyPossibleGap
Reports potential data gaps in metrics collection.

**Endpoint**: `POST /billing.v1.BillingService/NotifyPossibleGap`

**Request**:
```json
{
  "vm_id": "ud-1234567890abcdef",
  "last_sent": 1705317000000000000,
  "resume_time": 1705317600000000000
}
```

**Response**:
```json
{
  "success": true
}
```

**Purpose**:
- Data integrity monitoring
- Gap detection and reporting
- Reconciliation triggers
- Alerting for incomplete data

## Integration Patterns

### Typical Metrics Collection Flow

```bash
# 1. VM starts - notify billaged
curl -X POST http://localhost:8081/billing.v1.BillingService/NotifyVmStarted \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "ud-1234567890abcdef",
    "customer_id": "customer-456",
    "start_time": 1705317000000000000
  }'

# 2. Send metrics batches every minute (600 samples at 100ms intervals)
curl -X POST http://localhost:8081/billing.v1.BillingService/SendMetricsBatch \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "ud-1234567890abcdef",
    "customer_id": "customer-456",
    "metrics": [...]
  }'

# 3. Send periodic heartbeats (every 30 seconds)
curl -X POST http://localhost:8081/billing.v1.BillingService/SendHeartbeat \
  -H "Content-Type: application/json" \
  -d '{
    "instance_id": "metald-us-east-1a",
    "active_vms": ["ud-1234567890abcdef"]
  }'

# 4. VM stops - notify billaged
curl -X POST http://localhost:8081/billing.v1.BillingService/NotifyVmStopped \
  -H "Content-Type: application/json" \
  -d '{
    "vm_id": "ud-1234567890abcdef",
    "stop_time": 1705320600000000000
  }'
```

## Error Handling

### Standard Error Responses

```json
{
  "code": "invalid_argument",
  "message": "vm_id is required"
}
```

### Error Codes

| Code | Description | Example Scenario |
|------|-------------|------------------|
| `invalid_argument` | Invalid request parameters | Missing vm_id |
| `resource_exhausted` | Rate limit exceeded | Too many requests |
| `internal` | Internal server error | Database unavailable |
| `unavailable` | Service temporarily unavailable | During maintenance |

## Performance Considerations

### Batch Size Recommendations

| Metric | Recommended | Maximum | Notes |
|--------|-------------|---------|-------|
| Metrics per batch | 600 | 1000 | 1 minute at 100ms intervals |
| Batch frequency | 60 seconds | 120 seconds | Balance latency vs efficiency |
| Concurrent connections | 100 | 1000 | Per metald instance |
| Request timeout | 10 seconds | 30 seconds | Network latency buffer |

### Throughput Targets

- **Metrics ingestion**: 50,000 metrics/second sustained
- **Peak capacity**: 200,000 metrics/second burst
- **Latency**: <10ms p95 for batch processing
- **Storage efficiency**: ~100 bytes per metric after compression

## Monitoring Endpoints

### Health Check
```bash
curl http://localhost:8081/health
```

### Metrics (Prometheus)
```bash
curl http://localhost:9465/metrics
```

Key metrics to monitor:
- `billaged_metrics_processed_total` - Total metrics processed
- `billaged_batch_size_histogram` - Distribution of batch sizes
- `billaged_processing_duration_seconds` - Processing latency
- `billaged_active_vms` - Currently tracked VMs

## Best Practices

1. **Batch Optimization**
   - Send full 600-sample batches when possible
   - Use compression for large deployments
   - Implement client-side buffering

2. **Error Recovery**
   - Implement exponential backoff on failures
   - Use NotifyPossibleGap for extended outages
   - Maintain local WAL for critical metrics

3. **Time Synchronization**
   - Ensure NTP synchronization across all nodes
   - Use monotonic clocks for intervals
   - Include timezone information in timestamps

4. **Security**
   - Use TLS for production deployments
   - Implement authentication tokens
   - Rate limit by customer_id

This API is designed for high-throughput, low-latency metrics collection with strong consistency guarantees for accurate billing.