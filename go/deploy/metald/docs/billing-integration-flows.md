# Billing Integration Flows

## Overview

This document describes the integration flows between metald metrics collection and the billaged service, showing how the two systems work together to provide accurate usage accounting.

**Cross-Reference**: This document integrates with:
- [`billaged/docs/billing-architecture.md`](../../billaged/docs/billing-architecture.md) - Overall billing system design
- [`billaged/docs/billing-flows.md`](../../billaged/docs/billing-flows.md) - End-to-end billing workflows
- [`billing-metrics-architecture.md`](./billing-metrics-architecture.md) - metald metrics collection design

## Integration Architecture

```mermaid
graph TB
    subgraph "metald Service"
        VMS[VM Service] --> MC[Metrics Collector]
        MC --> WAL[Write-Ahead Log]
        MC --> HB[Heartbeat Service]
    end
    
    subgraph "billaged Service"
        MI[Metrics Ingestion] --> DV[Data Validation]
        DV --> DD[Duplicate Detection]
        DD --> TS[Time Series Storage]
        
        HM[Heartbeat Monitor] --> TO[Timeout Detection]
        TO --> GH[Gap Handling]
    end
    
    subgraph "Data Flow"
        MC -->|ConnectRPC Batches| MI
        HB -->|Heartbeat| HM
        WAL -->|Recovery| MI
    end
    
    subgraph "Storage Layer"
        TS --> CH[(ClickHouse)]
        GH --> CH
    end
```

## Normal Operation Flow

### VM Lifecycle with Billing Integration

```mermaid
sequenceDiagram
    participant CP as Control Plane
    participant VMS as VM Service
    participant MC as Metrics Collector
    participant VM as VM Instance
    participant BS as billaged Service
    participant DB as ClickHouse
    
    Note over CP,DB: VM Creation with Billing Lifecycle
    
    CP->>VMS: CreateVm(customer_id, vm_spec)
    VMS->>VM: Create VM instance
    VM->>VMS: VM created
    VMS->>CP: CreateVmResponse(vm_id)
    
    CP->>VMS: BootVm(vm_id)
    VMS->>VM: Boot VM
    VM->>VMS: VM booted
    
    Note over VMS,MC: Start billing collection
    VMS->>MC: StartCollection(vm_id, customer_id)
    MC->>BS: NotifyVmStarted(vm_id, customer_id, start_time)
    BS->>DB: Create billing session
    
    Note over MC,DB: Metrics Collection Loop (Every 100ms)
    
    loop Every 100ms
        MC->>VM: GetVMMetrics()
        VM->>MC: Resource usage data
        MC->>MC: Add to batch buffer
        
        alt Batch full (600 samples = 1 minute)
            MC->>BS: SendMetricsBatch(vm_id, metrics[])
            BS->>DB: Store raw metrics
            BS->>MC: BatchAck
        end
    end
    
    Note over CP,DB: VM Termination
    
    CP->>VMS: ShutdownVm(vm_id)
    VMS->>MC: StopCollection(vm_id)
    MC->>BS: SendFinalBatch(vm_id, final_metrics[])
    MC->>BS: NotifyVmStopped(vm_id, stop_time)
    
    VMS->>VM: Shutdown VM
    VM->>VMS: VM stopped
    VMS->>CP: ShutdownVmResponse
    
    BS->>DB: Finalize billing session
```

### Heartbeat and Health Monitoring

```mermaid
sequenceDiagram
    participant MC as Metrics Collector
    participant HM as Heartbeat Monitor
    participant TO as Timeout Detector
    participant DB as ClickHouse
    
    Note over MC,DB: Continuous Health Monitoring
    
    loop Every 30 seconds
        MC->>HM: SendHeartbeat(instance_id, active_vms[])
        HM->>MC: HeartbeatAck
        HM->>HM: Update last_seen timestamp
    end
    
    Note over TO,DB: Timeout Detection (Every 2 minutes)
    
    loop Every 2 minutes
        TO->>HM: CheckStaleInstances()
        HM->>TO: Instances without heartbeat > 2min
        
        alt Stale instance found
            TO->>DB: Mark VMs stopped at last_heartbeat
            TO->>HM: Remove stale instance
            Note over TO: Log potential data loss
        end
    end
```

## Failure Recovery Flows

### metald Restart Recovery

```mermaid
sequenceDiagram
    participant START as Startup Process
    participant WAL as Write-Ahead Log
    participant VMM as VMM Backend
    participant MC as Metrics Collector
    participant BS as billaged Service
    
    Note over START,BS: metald Restart Sequence
    
    START->>WAL: ReadUnsentEntries()
    WAL->>START: Unsent metrics list
    
    par Recover unsent metrics
        START->>BS: SendRecoveredBatch(metrics[])
        BS->>START: RecoveryAck
    and Discover running VMs
        START->>VMM: ListActiveVMs()
        VMM->>START: Running VM list
    and Query expected billing state
        START->>BS: GetActiveBillingSessions(instance_id)
        BS->>START: Expected VM list
    end
    
    Note over START: Cross-reference VM states
    
    loop For each running VM
        alt VM in billing but not in VMM
            START->>BS: NotifyVmStopped(vm_id, restart_time)
            Note over START: VM stopped during downtime
        else VM in VMM but not in billing
            START->>MC: StartCollection(vm_id, customer_id)
            START->>BS: NotifyVmStarted(vm_id, customer_id, restart_time)
            Note over START: New VM or missed start
        else VM in both systems
            START->>MC: ResumeCollection(vm_id, customer_id)
            START->>BS: NotifyPossibleGap(vm_id, last_sent, restart_time)
            Note over START: Resume with gap notification
        end
    end
    
    START->>MC: StartHeartbeat()
    MC->>BS: SendHeartbeat(instance_id, active_vms[])
```

### Network Partition Recovery

```mermaid
sequenceDiagram
    participant MC as Metrics Collector
    participant LQ as Local Queue
    participant DS as Disk Spill
    participant BS as billaged Service
    
    Note over MC,BS: Network Partition Scenario
    
    MC->>BS: SendMetricsBatch() [Network Error]
    BS--xMC: Connection failed
    
    MC->>LQ: Queue batch for retry
    Note over MC: Continue collecting to local storage
    
    loop Collection continues
        MC->>MC: Collect metrics
        MC->>LQ: Add to pending queue
        
        alt Queue exceeds memory limit
            LQ->>DS: Spill batches to disk
            DS->>LQ: Disk spill complete
        end
    end
    
    Note over MC,BS: Network restored
    
    MC->>BS: SendHeartbeat() [Success]
    BS->>MC: HeartbeatAck
    
    Note over MC: Start batch replay
    
    loop Replay pending batches
        MC->>LQ: Get next pending batch
        LQ->>MC: Batch data
        
        MC->>BS: SendMetricsBatch(batch)
        alt Send successful
            BS->>MC: BatchAck
            MC->>LQ: Remove from queue
        else Send failed
            MC->>MC: Exponential backoff
            Note over MC: Retry later
        end
    end
    
    opt Load spilled batches
        MC->>DS: Load spilled batches
        DS->>MC: Historical batch data
        MC->>BS: SendMetricsBatch(historical)
        BS->>MC: BatchAck
    end
```

### billaged Service Recovery

```mermaid
sequenceDiagram
    participant MC as Metrics Collector
    participant CB as Circuit Breaker
    participant LQ as Local Queue
    participant BS as billaged Service
    participant DB as ClickHouse
    
    Note over MC,DB: billaged Service Down
    
    MC->>BS: SendMetricsBatch() [Service Down]
    BS--xMC: Service unavailable
    
    MC->>CB: Record failure
    CB->>MC: Circuit state: Degraded
    
    Note over MC: Switch to local queuing mode
    
    loop Service down period
        MC->>MC: Continue metrics collection
        MC->>LQ: Queue all batches locally
        
        alt Circuit breaker threshold reached
            CB->>MC: Circuit state: Open
            Note over MC: Stop attempting sends for 5 minutes
        end
    end
    
    Note over BS,DB: billaged service restored
    
    BS->>DB: Service startup complete
    
    Note over CB: Circuit breaker half-open timeout
    
    CB->>MC: Circuit state: Half-Open
    MC->>BS: Test connection with heartbeat
    BS->>MC: HeartbeatAck
    
    CB->>MC: Circuit state: Closed
    Note over MC: Resume normal operation
    
    loop Replay queued batches
        MC->>LQ: Get pending batch
        LQ->>MC: Batch data
        MC->>BS: SendMetricsBatch(batch)
        BS->>MC: BatchAck
        
        alt Replay successful
            MC->>LQ: Remove from queue
        else Replay failed
            CB->>MC: Circuit state: Degraded
            Note over MC: Back to local queuing
        end
    end
```

## Data Consistency Flows

### Duplicate Detection and Deduplication

```mermaid
sequenceDiagram
    participant MC as Metrics Collector
    participant BS as billaged Service
    participant DD as Duplicate Detector
    participant DB as ClickHouse
    
    Note over MC,DB: Duplicate Handling Flow
    
    MC->>BS: SendMetricsBatch(vm_id, timestamp_range, metrics[])
    BS->>DD: CheckDuplicates(vm_id, timestamp_range)
    
    DD->>DB: Query existing metrics for range
    DB->>DD: Existing timestamp list
    
    alt No duplicates found
        DD->>BS: No duplicates
        BS->>DB: Store all metrics
        BS->>MC: BatchAck(stored_count: all)
    else Partial duplicates found
        DD->>BS: Duplicate timestamps: [list]
        BS->>BS: Filter out duplicates
        BS->>DB: Store non-duplicate metrics
        BS->>MC: BatchAck(stored_count: partial)
        Note over BS: Log duplicate detection event
    else Complete duplicates
        DD->>BS: All timestamps exist
        BS->>MC: BatchAck(stored_count: 0)
        Note over BS: Idempotent operation
    end
```

### Gap Detection and Interpolation

```mermaid
sequenceDiagram
    participant MC as Metrics Collector
    participant BS as billaged Service
    participant GD as Gap Detector
    participant GH as Gap Handler
    participant DB as ClickHouse
    
    Note over MC,DB: Gap Detection Flow
    
    MC->>BS: SendMetricsBatch(vm_id, batch_start, batch_end, metrics[])
    BS->>GD: CheckForGaps(vm_id, batch_start)
    
    GD->>DB: Query last metric timestamp for vm_id
    DB->>GD: Last timestamp: T-N
    
    GD->>GD: Calculate gap: batch_start - last_timestamp
    
    alt Gap ≤ 200ms (normal)
        GD->>BS: No significant gap
        BS->>DB: Store metrics normally
    else Gap 200ms - 10min (short gap)
        GD->>GH: Handle short gap
        GH->>GH: Linear interpolation
        GH->>DB: Store interpolated data
        GH->>DB: Store actual metrics
        Note over GH: Log interpolation event
    else Gap > 10min (long gap)
        GD->>GH: Handle long gap
        GH->>GH: Mark as zero usage period
        GH->>DB: Store zero usage markers
        GH->>DB: Store actual metrics
        Note over GH: Log significant gap event
    end
    
    BS->>MC: BatchAck
```

### Precision and Unit Conversion

```mermaid
flowchart TD
    subgraph "metald Collection (High Precision)"
        CPU_NS[CPU: nanoseconds]
        MEM_B[Memory: bytes]
        IO_B[I/O: bytes] 
        NET_B[Network: bytes]
    end
    
    subgraph "billaged Storage (Raw Precision)"
        CPU_NS --> CPU_NS_STORE[Store: int64 nanoseconds]
        MEM_B --> MEM_B_STORE[Store: int64 bytes]
        IO_B --> IO_B_STORE[Store: int64 bytes]
        NET_B --> NET_B_STORE[Store: int64 bytes]
    end
    
    subgraph "billaged Aggregation (Time Windows)"
        CPU_NS_STORE --> CPU_HOUR[Aggregate: CPU nanoseconds → core-hours]
        MEM_B_STORE --> MEM_HOUR[Aggregate: Memory byte-seconds → GB-hours]
        IO_B_STORE --> IO_TOTAL[Aggregate: I/O bytes → total GB]
        NET_B_STORE --> NET_TOTAL[Aggregate: Network bytes → total GB]
    end
    
    subgraph "billaged Billing (Customer Precision)"
        CPU_HOUR --> CPU_BILL[Bill: millisecond precision]
        MEM_HOUR --> MEM_BILL[Bill: KB precision (rounded up)]
        IO_TOTAL --> IO_BILL[Bill: KB precision (rounded up)]
        NET_TOTAL --> NET_BILL[Bill: KB precision (rounded up)]
    end
```

## Integration Points with billaged

### ConnectRPC Service Definitions

**Metrics Ingestion Service**:
```protobuf
service MetricsIngestionService {
  // Send batch of metrics for a VM
  rpc SendMetricsBatch(SendMetricsBatchRequest) returns (SendMetricsBatchResponse);
  
  // Notify VM lifecycle events
  rpc NotifyVmStarted(NotifyVmStartedRequest) returns (NotifyVmStartedResponse);
  rpc NotifyVmStopped(NotifyVmStoppedRequest) returns (NotifyVmStoppedResponse);
  rpc NotifyPossibleGap(NotifyPossibleGapRequest) returns (NotifyPossibleGapResponse);
  
  // Heartbeat for health monitoring
  rpc SendHeartbeat(SendHeartbeatRequest) returns (SendHeartbeatResponse);
  
  // Recovery operations
  rpc GetActiveBillingSessions(GetActiveBillingSessionsRequest) returns (GetActiveBillingSessionsResponse);
}

message SendMetricsBatchRequest {
  string vm_id = 1;
  string customer_id = 2;
  string metald_instance_id = 3;
  int64 batch_start_timestamp = 4;
  int64 batch_end_timestamp = 5;
  repeated VmMetric metrics = 6;
}

message VmMetric {
  int64 timestamp_nanos = 1;
  int64 cpu_time_nanos = 2;
  int64 memory_usage_bytes = 3;
  int64 disk_read_bytes = 4;
  int64 disk_write_bytes = 5;
  int64 network_rx_bytes = 6;
  int64 network_tx_bytes = 7;
}

message SendHeartbeatRequest {
  string metald_instance_id = 1;
  repeated string active_vm_ids = 2;
  int64 timestamp_nanos = 3;
}
```

### Data Flow Alignment

**Batch Timing Alignment**:
- metald: 100ms collection → 600 samples → 1-minute batches
- billaged: Expects 1-minute batches → Hourly rollups → Monthly billing
- **✓ Aligned**: Batch windows match billaged aggregation periods

**Precision Alignment**:
- metald: Nanosecond CPU, byte-level I/O collection
- billaged: Raw storage preserves precision, billing rounds appropriately
- **✓ Aligned**: No precision loss in collection → storage → billing pipeline

**Gap Handling Alignment**:
- metald: WAL recovery ensures no gaps from process failures
- billaged: Gap detection and interpolation for network failures
- **✓ Aligned**: Comprehensive gap coverage across failure modes

### Error Handling Coordination

**Retry Policy Coordination**:
```
metald Retry Policy:
- Initial retry: 1 minute
- Exponential backoff: 2^attempt minutes (max 60 minutes)
- Drop after: 24 hours

billaged Timeout Policy:
- Heartbeat timeout: 2 minutes
- Instance cleanup: After 2 minutes without heartbeat
- Billing session finalization: Immediate on timeout

✓ Coordination: billaged timeout (2min) < metald first retry (1min)
```

**Circuit Breaker Coordination**:
```
metald Circuit Breaker:
- Degraded: After 3 failures → local queuing
- Open: After 10 failures → stop sending for 5 minutes
- Half-open: Test with heartbeat

billaged Health Endpoint:
- /health/ready: Service can accept requests
- /health/live: Service is running
- Used by metald circuit breaker for state decisions
```

## Monitoring and Observability Integration

### Shared Metrics

**Data Flow Metrics**:
- `metald_metrics_sent_total` ↔ `billaged_metrics_received_total`
- `metald_batch_send_errors_total` ↔ `billaged_batch_receive_errors_total`
- `metald_heartbeat_sent_total` ↔ `billaged_heartbeat_received_total`

**Data Quality Metrics**:
- `billaged_duplicate_metrics_total`: Duplicates detected from metald retries
- `billaged_gap_interpolations_total`: Gaps filled from metald failures
- `billaged_billing_sessions_active`: Active VMs being billed

**SLA Metrics**:
- End-to-end latency: Metric collection → billaged storage
- Data completeness: % of expected metrics received
- Billing accuracy: Variance in expected vs actual billing periods

### Alerting Coordination

**Critical Alerts** (Both systems):
- Metrics collection stopped for any VM > 5 minutes
- Heartbeat failures > 3 consecutive
- WAL recovery failures
- Data gaps > 10 minutes for any VM

**Warning Alerts**:
- Pending batch queue growing > 100 batches
- Circuit breaker state changes
- Significant interpolation events (gaps 2-10 minutes)

## Performance and Scaling

### Throughput Alignment

**metald Capacity**:
- 100ms collection × 1000 VMs = 10,000 metrics/second
- 1-minute batches × 1000 VMs = ~17 batches/second
- Network bandwidth: ~1MB/second for metrics data

**billaged Capacity** (from billaged docs):
- Target: 10K metrics/second ingestion
- Peak: 50K metrics/second capacity
- **✓ Aligned**: metald peak (10K/sec) < billaged target (10K/sec)

### Scaling Considerations

**Horizontal Scaling**:
- Multiple metald instances → Single billaged cluster
- Each metald sends unique instance_id in heartbeats
- billaged partitions data by customer_id and time

**Resource Requirements**:
- metald: ~1MB memory per 1000 VMs (buffering)
- billaged: ClickHouse storage scales with retention period
- **Network**: Steady-state traffic, minimal bursts

## Conclusion

The integration between metald metrics collection and billaged service provides:

1. **Seamless Data Flow**: 100ms precision collection → 1-minute batches → hourly aggregation
2. **Comprehensive Reliability**: WAL recovery + heartbeat monitoring + gap detection
3. **Coordinated Error Handling**: Aligned retry policies and circuit breaker patterns
4. **Consistent Monitoring**: Shared metrics and alerting across service boundaries

This integration ensures accurate billing with sub-second precision while maintaining high availability and data consistency under all failure scenarios.