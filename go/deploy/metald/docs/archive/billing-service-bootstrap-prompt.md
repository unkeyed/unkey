# Billing Service Bootstrap Prompt

## Context

I'm implementing a standalone billing service that collects usage metrics from metald (VM orchestration service) across multiple regions and generates accurate customer invoices. This service needs to handle high-precision billing data with nanosecond CPU time tracking and byte-level I/O accounting.

## Architecture Overview

```
┌─────────────────┐    metrics     ┌─────────────────┐
│     metald      │───────────────→│ billing-service │
│ (Multi-region)  │   streaming    │  (Centralized)  │
│                 │                │                 │
│ • VM lifecycle  │   ┌─────────┐  │ • Aggregation   │
│ • Local metrics │   │ Buffer  │  │ • Multi-region  │
│ • 100ms collect │   │ & Retry │  │   consolidation │
│                 │   └─────────┘  │ • Invoice gen   │
└─────────────────┘                └─────────────────┘
```

## Requirements

### Core Features
1. **Metrics Ingestion**: Receive usage data from multiple metald instances
2. **Multi-Region Aggregation**: Consolidate customer usage across regions
3. **Time-Series Storage**: Store metrics with high precision and retention
4. **Billing Calculation**: Generate accurate invoices with prorated periods
5. **ConnectRPC API**: Provide both ingestion and query interfaces

### Precision Requirements
- **CPU Time**: Nanosecond precision via cgroups `cpuacct.usage`
- **Memory**: Page-level (4KB) precision with peak tracking
- **I/O**: Byte-level precision for disk operations
- **Network**: Byte-level precision for ingress/egress

### Billing Model
- **Collection**: 100ms intervals from metald
- **Streaming**: 1-minute batches to billing service
- **Aggregation**: Hourly rollups across regions
- **Invoicing**: Monthly with prorated partial periods

## Technical Stack

- **Language**: Go (matching metald ecosystem)
- **API Framework**: ConnectRPC with Protocol Buffers
- **Time-Series DB**: TBD (InfluxDB, TimescaleDB, or ClickHouse)
- **Message Queue**: Optional for high-throughput scenarios
- **Observability**: OpenTelemetry (matching metald)

## Proto Schema

```protobuf
service BillingCollectorService {
  rpc SendMetrics(SendMetricsRequest) returns (SendMetricsResponse);
  rpc SendMetricsBatch(SendMetricsBatchRequest) returns (SendMetricsBatchResponse);
}

service BillingQueryService {
  rpc GetCustomerUsage(GetCustomerUsageRequest) returns (GetCustomerUsageResponse);
  rpc GetBillingWindow(GetBillingWindowRequest) returns (GetBillingWindowResponse);
  rpc StreamLiveUsage(StreamLiveUsageRequest) returns (stream UsageUpdate);
}

message BillingMetrics {
  string vm_id = 1;
  string customer_id = 2;
  string region = 3;
  google.protobuf.Timestamp timestamp = 4;
  google.protobuf.Timestamp window_start = 5;
  google.protobuf.Timestamp window_end = 6;
  
  // CPU metrics (nanosecond precision)
  uint64 cpu_time_nanos = 7;
  double cpu_utilization_pct = 8;
  double cpu_cores_used = 9;
  
  // Memory metrics (byte precision)
  uint64 memory_bytes_used = 10;
  uint64 memory_bytes_peak = 11;
  uint64 memory_bytes_seconds = 12; // integral for billing
  
  // I/O metrics (byte precision)
  uint64 disk_read_bytes = 13;
  uint64 disk_write_bytes = 14;
  uint64 disk_read_ops = 15;
  uint64 disk_write_ops = 16;
  
  // Network metrics (byte precision)
  uint64 network_rx_bytes = 17;
  uint64 network_tx_bytes = 18;
}

message PricingRates {
  double cpu_per_core_hour = 1;
  double memory_per_gb_hour = 2;
  double disk_io_per_gb = 3;
  double network_per_gb = 4;
}
```

## Billing Example

**Customer runs 2 VMs in 2 regions for 30 days, adds 3rd VM on day 12:**
- Region A: VM1 (720 hours), VM2 (720 hours)
- Region B: VM3 (432 hours, started day 13)
- Total: 1,872 VM-hours
- Invoice: Prorated billing with exact start/stop times

## Project Structure

```
billing-service/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── collector/      # Ingestion from metald
│   ├── aggregator/     # Multi-region consolidation
│   ├── storage/        # Time-series operations
│   ├── billing/        # Cost calculations
│   └── service/        # ConnectRPC handlers
├── proto/
│   └── billing/
│       └── v1/
│           └── billing.proto
├── go.mod
├── go.sum
├── Dockerfile
└── README.md
```

## Implementation Priorities

### Phase 1: Foundation
1. Project setup with Go modules and ConnectRPC
2. Proto definitions and code generation
3. Basic metrics ingestion endpoint
4. In-memory storage for prototyping

### Phase 2: Storage & Aggregation
1. Time-series database integration
2. Multi-region metrics consolidation
3. Hourly/daily/monthly rollup jobs
4. Data retention policies

### Phase 3: Billing Logic
1. Pricing rate configuration
2. Usage cost calculations
3. Prorated billing for partial periods
4. Invoice generation

### Phase 4: Production Features
1. High availability and scaling
2. Comprehensive observability
3. Data backup and recovery
4. Performance optimization

## Error Handling Requirements

- **Duplicate Metrics**: Idempotent ingestion with deduplication
- **Missing Data**: Gap detection and interpolation
- **Backpressure**: Rate limiting and circuit breakers
- **Data Integrity**: Atomic operations and consistency checks

## Security Considerations

- **Authentication**: Service-to-service auth for metald connections
- **Authorization**: Customer data isolation
- **Audit Logging**: All financial operations logged
- **Data Encryption**: At rest and in transit

## Success Metrics

- **Accuracy**: 99.99% billing precision
- **Availability**: 99.9% uptime SLA
- **Latency**: <100ms for query APIs
- **Throughput**: 10K+ metrics/second ingestion
- **Data Loss**: Zero tolerance for billing data

## Questions to Address

1. **Time-Series DB Choice**: Which database best fits our precision and scale requirements?
2. **Message Queue**: Do we need async processing for high-throughput scenarios?
3. **Pricing Model**: How complex should our pricing rules engine be?
4. **Multi-tenancy**: How do we ensure customer data isolation at scale?

## Development Environment

- Use `buf` for Protocol Buffer management
- Include comprehensive unit tests
- Docker for local development
- OpenTelemetry for observability from day one

---

**Task**: Please bootstrap the billing service project structure, implement the basic proto definitions, and create a minimal ConnectRPC server that can receive metrics from metald. Focus on getting the foundation right - we'll iterate on storage and billing logic in subsequent sessions.

**Additional Context**: This billing service will eventually handle thousands of VMs across multiple regions with sub-second billing precision, so design with scale in mind from the beginning.