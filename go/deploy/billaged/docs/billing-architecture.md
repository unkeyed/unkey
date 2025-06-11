# Billing Service Architecture

## Overview

The billing service provides high-precision usage accounting for VM resources across multiple regions. It collects metrics from metald instances and aggregates them for accurate customer billing with nanosecond CPU precision and byte-level I/O tracking.

## Architecture Diagram

```mermaid
graph TB
    subgraph "Region A"
        MA[metald A] --> BCA[Buffer/Cache A]
    end
    
    subgraph "Region B"
        MB[metald B] --> BCB[Buffer/Cache B]
    end
    
    subgraph "Region N"
        MN[metald N] --> BCN[Buffer/Cache N]
    end
    
    BCA --> |1min batches| BS[Billing Service]
    BCB --> |1min batches| BS
    BCN --> |1min batches| BS
    
    BS --> CH[ClickHouse]
    BS --> AGG[Aggregator]
    
    AGG --> |Hourly Rollups| CH
    CH --> CALC[Billing Calculator]
    CALC --> INV[Invoice Generator]
    
    subgraph "Billing Service Components"
        BS
        CH
        AGG
        CALC
        INV
    end
    
    subgraph "External Systems"
        EXT[Deploy Control Plane]
        CUST[Customer Portal]
    end
    
    INV --> EXT
    CALC --> CUST
```

## Component Architecture

```mermaid
graph LR
    subgraph "Billing Service"
        API[ConnectRPC API]
        COL[Collector]
        AGG[Aggregator]
        STOR[Storage Layer]
        BILL[Billing Engine]
        
        API --> COL
        COL --> STOR
        STOR --> AGG
        AGG --> STOR
        STOR --> BILL
    end
    
    subgraph "Data Flow"
        METR[Metrics Ingestion] --> COL
        COL --> DEDUP[Deduplication]
        DEDUP --> VAL[Validation]
        VAL --> STOR
    end
    
    subgraph "Storage"
        STOR --> RAW[Raw Metrics]
        STOR --> ROLL[Rollup Tables]
        STOR --> BILL_DATA[Billing Data]
    end
```

## Data Flow Architecture

### Metrics Collection Flow

```mermaid
sequenceDiagram
    participant VM as VM Instance
    participant M as metald
    participant BC as Buffer/Cache
    participant BS as Billing Service
    participant TSDB as Time Series DB
    
    Note over VM,TSDB: 100ms Collection Cycle
    
    loop Every 100ms
        VM->>M: cgroups metrics
        Note right of VM: cpu_time_nanos<br/>memory_bytes<br/>disk_io_bytes<br/>network_bytes
        M->>BC: Store metrics
    end
    
    Note over M,BS: 1-minute Batch Window
    
    loop Every 1 minute
        BC->>BS: SendMetricsBatch
        Note right of BC: Aggregated 600 samples<br/>(100ms * 600 = 1min)
        BS->>BS: Validate & Deduplicate
        BS->>TSDB: Store raw metrics
    end
    
    Note over BS,TSDB: Hourly Aggregation
    
    loop Every hour
        BS->>TSDB: Query raw metrics
        BS->>BS: Calculate rollups
        BS->>TSDB: Store hourly aggregates
    end
```

### Billing Calculation Flow

```mermaid
sequenceDiagram
    participant TSDB as Time Series DB
    participant AGG as Aggregator
    participant CALC as Calculator
    participant RATES as Pricing Rates
    participant INV as Invoice Generator
    
    Note over TSDB,INV: Monthly Billing Cycle
    
    INV->>TSDB: Query customer usage
    Note right of INV: Get all VM hours<br/>for billing period
    
    TSDB->>AGG: Raw usage data
    AGG->>AGG: Multi-region consolidation
    AGG->>CALC: Aggregated usage
    
    CALC->>RATES: Get pricing rates
    RATES->>CALC: Current rates
    
    Note over CALC: Calculate costs:<br/>CPU: nanos → core-hours<br/>Memory: byte-seconds → GB-hours<br/>I/O: bytes → GB transferred<br/>Network: bytes → GB transferred
    
    CALC->>INV: Usage costs
    INV->>INV: Generate invoice
    INV->>INV: Handle prorated periods
```

## Precision Requirements

### Resource Tracking Precision

| Resource | Collection Precision | Storage Precision | Billing Precision |
|----------|---------------------|-------------------|------------------|
| CPU Time | nanoseconds | nanoseconds | milliseconds |
| Memory | bytes | bytes | KB (rounded up) |
| Disk I/O | bytes | bytes | KB (rounded up) |
| Network | bytes | bytes | KB (rounded up) |

### Time Windows

```mermaid
gantt
    title Billing Time Windows
    dateFormat X
    axisFormat %s
    
    section Collection
    metald samples     :0, 100
    Buffer window      :0, 60000
    
    section Aggregation  
    Batch to billing   :60000, 120000
    Hourly rollup      :3600000, 7200000
    
    section Billing
    Monthly cycle      :2592000000, 5184000000
```

## Error Handling & Reliability

### Duplicate Handling

```mermaid
flowchart TD
    RECV[Receive Metrics] --> CHECK{Check Timestamp + VM ID}
    CHECK -->|Exists| DUP[Duplicate Detection]
    CHECK -->|New| STORE[Store Metrics]
    
    DUP --> COMP{Compare Values}
    COMP -->|Match| IGNORE[Ignore Duplicate]
    COMP -->|Differ| ALERT[Alert & Manual Review]
    
    STORE --> SUCCESS[Success]
```

### Gap Detection

```mermaid
flowchart TD
    QUERY[Query Time Range] --> ANALYZE[Analyze Timestamps]
    ANALYZE --> GAP{Gap > 2 minutes?}
    
    GAP -->|No| OK[Complete Data]
    GAP -->|Yes| INTERP{Gap < 10 minutes?}
    
    INTERP -->|Yes| LINEAR[Linear Interpolation]
    INTERP -->|No| ZERO[Zero Usage Period]
    
    LINEAR --> STORE[Store Interpolated]
    ZERO --> STORE
```

## Multi-Region Consolidation

### Customer Usage Aggregation

```mermaid
graph TB
    subgraph "Customer ABC Usage"
        subgraph "Region US-East"
            USE1[VM-1: 720h]
            USE2[VM-2: 720h]
        end
        
        subgraph "Region EU-West"
            USE3[VM-3: 432h]
        end
        
        subgraph "Region APAC"
            USE4[VM-4: 168h]
        end
    end
    
    USE1 --> AGG[Regional Aggregator]
    USE2 --> AGG
    USE3 --> AGG
    USE4 --> AGG
    
    AGG --> TOTAL[Total: 2,040 VM-hours]
    TOTAL --> BILL[Monthly Bill]
```

## Storage Schema

### Time Series Tables

```sql
-- Raw metrics table (high frequency)
CREATE TABLE metrics_raw (
    timestamp TIMESTAMPTZ NOT NULL,
    vm_id VARCHAR(64) NOT NULL,
    customer_id VARCHAR(64) NOT NULL,
    region VARCHAR(32) NOT NULL,
    
    -- CPU metrics (nanosecond precision)
    cpu_time_nanos BIGINT NOT NULL,
    cpu_utilization_pct DOUBLE PRECISION,
    cpu_cores_used DOUBLE PRECISION,
    
    -- Memory metrics (byte precision)
    memory_bytes_used BIGINT NOT NULL,
    memory_bytes_peak BIGINT NOT NULL,
    
    -- I/O metrics
    disk_read_bytes BIGINT NOT NULL,
    disk_write_bytes BIGINT NOT NULL,
    network_rx_bytes BIGINT NOT NULL,
    network_tx_bytes BIGINT NOT NULL,
    
    PRIMARY KEY (timestamp, vm_id)
);

-- Hourly rollups (for billing)
CREATE TABLE metrics_hourly (
    hour_start TIMESTAMPTZ NOT NULL,
    vm_id VARCHAR(64) NOT NULL,
    customer_id VARCHAR(64) NOT NULL,
    region VARCHAR(32) NOT NULL,
    
    -- Aggregated values
    cpu_core_hours DOUBLE PRECISION NOT NULL,
    memory_gb_hours DOUBLE PRECISION NOT NULL,
    disk_gb_transferred DOUBLE PRECISION NOT NULL,
    network_gb_transferred DOUBLE PRECISION NOT NULL,
    
    PRIMARY KEY (hour_start, vm_id)
);
```

## Performance Requirements

### Throughput Targets

| Metric | Target | Peak Capacity |
|--------|--------|---------------|
| Metrics Ingestion | 10K/second | 50K/second |
| Query Response | <100ms p95 | <500ms p99 |
| Billing Calculation | <5 seconds | Monthly invoice |
| Data Retention | 13 months | For compliance |

### Scaling Considerations

```mermaid
graph LR
    subgraph "Horizontal Scaling"
        LB[Load Balancer] --> BS1[Billing Service 1]
        LB --> BS2[Billing Service 2]
        LB --> BS3[Billing Service N]
    end
    
    BS1 --> TSDB[Distributed TSDB]
    BS2 --> TSDB
    BS3 --> TSDB
    
    subgraph "Data Partitioning"
        TSDB --> P1[Partition by Customer]
        TSDB --> P2[Partition by Time]
        TSDB --> P3[Partition by Region]
    end
```