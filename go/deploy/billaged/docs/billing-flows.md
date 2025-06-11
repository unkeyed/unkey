# Billing Service Flow Documentation

## Core Billing Flows

### 1. VM Lifecycle Billing Flow

```mermaid
sequenceDiagram
    participant CP as Control Plane
    participant M as metald
    participant VM as VM Instance
    participant BS as Billing Service
    participant DB as Time Series DB

    Note over CP,DB: VM Creation & Billing Start
    
    CP->>M: CreateVM(customer_id, vm_spec)
    M->>VM: Start VM
    VM->>M: VM Started (timestamp)
    M->>BS: StartBilling(vm_id, customer_id, start_time)
    BS->>DB: Create billing record
    
    Note over M,DB: Runtime Metrics Collection (Every 100ms)
    
    loop Every 100ms
        VM->>M: Resource metrics
        Note right of VM: cgroups data:<br/>- cpu.usage<br/>- memory.usage<br/>- blkio.io_service_bytes<br/>- network stats
    end
    
    loop Every 1 minute (600 samples)
        M->>BS: SendMetricsBatch(aggregated_metrics)
        BS->>BS: Validate & deduplicate
        BS->>DB: Store metrics
    end
    
    Note over CP,DB: VM Termination & Billing End
    
    CP->>M: DeleteVM(vm_id)
    M->>VM: Stop VM
    VM->>M: VM Stopped (timestamp)
    M->>BS: StopBilling(vm_id, end_time)
    BS->>BS: Calculate final usage
    BS->>DB: Finalize billing record
```

### 2. Multi-Region Usage Aggregation

```mermaid
sequenceDiagram
    participant BSA as Billing Service A
    participant BSB as Billing Service B
    participant BSN as Billing Service N
    participant AGG as Aggregation Service
    participant DB as Central DB
    participant INV as Invoice Service

    Note over BSA,INV: End of Billing Period (Monthly)
    
    par Regional Collection
        BSA->>AGG: Region A usage data
        BSB->>AGG: Region B usage data
        BSN->>AGG: Region N usage data
    end
    
    AGG->>AGG: Consolidate by customer_id
    AGG->>DB: Store consolidated usage
    
    INV->>DB: Query customer usage
    DB->>INV: Aggregated usage data
    
    INV->>INV: Calculate billing amounts
    Note right of INV: Apply pricing rates:<br/>- CPU: $0.10/core-hour<br/>- Memory: $0.05/GB-hour<br/>- I/O: $0.10/GB<br/>- Network: $0.15/GB
    
    INV->>INV: Generate invoice
    INV->>CP: Invoice ready
```

### 3. Real-time Usage Tracking

```mermaid
sequenceDiagram
    participant CLIENT as Customer Portal
    participant API as Billing API
    participant CACHE as Redis Cache
    participant DB as Time Series DB
    participant STREAM as Live Stream

    CLIENT->>API: GetLiveUsage(customer_id)
    
    par Cache Check
        API->>CACHE: Check recent usage
        CACHE-->>API: Cached data (if exists)
    and Database Query
        API->>DB: Query last hour usage
        DB-->>API: Recent usage data
    end
    
    API->>API: Merge cache + DB data
    API->>CLIENT: Current usage response
    
    Note over API,STREAM: Live Updates (Optional)
    
    opt Live streaming enabled
        CLIENT->>API: StreamLiveUsage(customer_id)
        API->>STREAM: Subscribe to updates
        
        loop Every minute
            STREAM->>API: Usage update
            API->>CLIENT: Stream usage update
        end
    end
```

### 4. Billing Calculation Flow

```mermaid
flowchart TD
    START[Monthly Billing Trigger] --> QUERY[Query Usage Data]
    
    QUERY --> VALIDATE{Data Complete?}
    VALIDATE -->|No| GAP[Handle Gaps]
    VALIDATE -->|Yes| AGGREGATE[Aggregate Usage]
    
    GAP --> INTERPOLATE{Gap < 10min?}
    INTERPOLATE -->|Yes| FILL[Linear Interpolation]
    INTERPOLATE -->|No| ZERO[Zero Usage]
    FILL --> AGGREGATE
    ZERO --> AGGREGATE
    
    AGGREGATE --> CONVERT[Convert Units]
    
    subgraph "Unit Conversion"
        CONVERT --> CPU[CPU: nanos → core-hours]
        CONVERT --> MEM[Memory: byte-seconds → GB-hours]
        CONVERT --> IO[I/O: bytes → GB]
        CONVERT --> NET[Network: bytes → GB]
    end
    
    CPU --> RATES[Apply Pricing Rates]
    MEM --> RATES
    IO --> RATES
    NET --> RATES
    
    RATES --> PRORATE{Partial Period?}
    PRORATE -->|Yes| CALC_PRORATE[Calculate Prorated Amount]
    PRORATE -->|No| TOTAL[Calculate Total]
    
    CALC_PRORATE --> TOTAL
    TOTAL --> GENERATE[Generate Invoice]
    GENERATE --> SEND[Send to Customer]
```

### 5. Error Handling & Recovery Flow

```mermaid
sequenceDiagram
    participant M as metald
    participant BS as Billing Service
    participant DB as Database
    participant ALERT as Alert System
    participant OPS as Operations Team

    Note over M,OPS: Error Scenarios
    
    M->>BS: SendMetricsBatch (network error)
    BS--xM: Connection Failed
    
    Note over M: Retry Logic
    M->>M: Exponential backoff
    M->>BS: Retry SendMetricsBatch
    BS->>BS: Process metrics
    
    alt Duplicate Detection
        BS->>BS: Check timestamp + vm_id
        BS->>BS: Skip duplicate
        BS->>M: Success (idempotent)
    else Data Corruption
        BS->>BS: Validation failed
        BS->>ALERT: Data corruption alert
        ALERT->>OPS: Investigate data issue
        BS->>M: Error response
    else Database Unavailable
        BS->>DB: Store metrics (fails)
        BS->>BS: Buffer to local storage
        BS->>ALERT: Database down alert
        
        loop Retry Logic
            BS->>DB: Retry connection
            DB-->>BS: Connection restored
            BS->>DB: Flush buffered data
        end
    end
```

### 6. Data Aggregation Patterns

```mermaid
graph TD
    subgraph "Raw Data (100ms intervals)"
        R1[Sample 1: T+0ms]
        R2[Sample 2: T+100ms]
        R3[Sample 3: T+200ms]
        RN[Sample N: T+60000ms]
    end
    
    R1 --> A1[1-minute Aggregate]
    R2 --> A1
    R3 --> A1
    RN --> A1
    
    A1 --> H1[Hourly Rollup]
    A2[Aggregate 2] --> H1
    A60[Aggregate 60] --> H1
    
    H1 --> D1[Daily Rollup]
    H2[Hour 2] --> D1
    H24[Hour 24] --> D1
    
    D1 --> M1[Monthly Bill]
    D2[Day 2] --> M1
    D30[Day 30] --> M1
    
    subgraph "Aggregation Functions"
        SUM[SUM: I/O bytes, Network bytes]
        AVG[AVG: CPU utilization, Memory usage]
        MAX[MAX: Peak memory]
        INT[INTEGRAL: Memory-seconds, CPU-seconds]
    end
```

### 7. Precision Handling Flow

```mermaid
flowchart LR
    subgraph "Collection Precision"
        CPU_NS[CPU: nanoseconds]
        MEM_B[Memory: bytes]
        IO_B[I/O: bytes]
        NET_B[Network: bytes]
    end
    
    subgraph "Storage Precision"
        CPU_NS --> CPU_NS_STORE[Store: int64 nanoseconds]
        MEM_B --> MEM_B_STORE[Store: int64 bytes]
        IO_B --> IO_B_STORE[Store: int64 bytes]
        NET_B --> NET_B_STORE[Store: int64 bytes]
    end
    
    subgraph "Billing Precision"
        CPU_NS_STORE --> CPU_MS[Bill: millisecond precision]
        MEM_B_STORE --> MEM_KB[Bill: KB precision (rounded up)]
        IO_B_STORE --> IO_KB[Bill: KB precision (rounded up)]
        NET_B_STORE --> NET_KB[Bill: KB precision (rounded up)]
    end
    
    subgraph "Rate Calculation"
        CPU_MS --> CPU_RATE[Rate: per core-hour]
        MEM_KB --> MEM_RATE[Rate: per GB-hour]
        IO_KB --> IO_RATE[Rate: per GB transferred]
        NET_KB --> NET_RATE[Rate: per GB transferred]
    end
```

## Integration Points

### metald Integration

```mermaid
graph LR
    subgraph "metald Process"
        VM_MON[VM Monitor] --> COLLECT[Metrics Collector]
        COLLECT --> BUFFER[Local Buffer]
        BUFFER --> BATCH[Batch Processor]
    end
    
    BATCH -->|ConnectRPC| BS[Billing Service]
    
    subgraph "Failure Modes"
        BATCH -.->|Network Error| RETRY[Retry Logic]
        BATCH -.->|Service Down| LOCAL[Local Storage]
        RETRY --> BS
        LOCAL -.->|Service Up| FLUSH[Flush to Service]
    end
```

### Customer Portal Integration

```mermaid
sequenceDiagram
    participant PORTAL as Customer Portal
    participant AUTH as Auth Service
    participant API as Billing API
    participant DB as Usage Database

    PORTAL->>AUTH: Authenticate user
    AUTH->>PORTAL: JWT token
    
    PORTAL->>API: GetCurrentUsage(customer_id, auth_token)
    API->>AUTH: Validate token
    AUTH->>API: Token valid
    
    API->>DB: Query recent usage
    DB->>API: Usage data
    
    API->>API: Format for display
    API->>PORTAL: Usage response
    
    Note over PORTAL: Display:<br/>- Current month usage<br/>- Projected costs<br/>- Resource breakdown<br/>- Historical trends
```

## Validation Points

### Data Integrity Checks

1. **Temporal Consistency**: Ensure timestamps are monotonically increasing
2. **Resource Bounds**: Validate metrics are within reasonable limits
3. **Completeness**: Check for missing data gaps
4. **Duplicate Prevention**: Ensure idempotent processing
5. **Cross-validation**: Compare aggregated values with raw data samples

This documentation provides a comprehensive view of how the billing service operates, handles edge cases, and integrates with other system components for accurate usage accounting.