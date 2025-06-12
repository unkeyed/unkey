# Billing System Architecture

> Real-time metrics collection and billing aggregation with 100ms precision

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Component Details](#component-details)
- [Metrics Collection](#metrics-collection)
- [Data Processing Pipeline](#data-processing-pipeline)
- [Integration Architecture](#integration-architecture)
- [Reliability & Recovery](#reliability--recovery)
- [Performance Characteristics](#performance-characteristics)
- [Operational Considerations](#operational-considerations)
- [Cross-References](#cross-references)

---

## Overview

The Billing System provides real-time metrics collection from virtual machines with 100ms precision, processing and aggregating billing data for downstream analytics and reporting systems.

### Key Responsibilities
- **Real-time Metrics Collection**: 100ms precision streaming from VM FIFO streams
- **Data Processing**: JSON parsing, validation, and enrichment
- **Batch Aggregation**: Efficient batching for downstream transmission
- **Heartbeat Monitoring**: Health monitoring and failure detection
- **Integration Management**: Communication with billing aggregators and ClickHouse
- **Failure Recovery**: Automatic retry logic and data loss prevention

### Technology Stack
- **Language**: Go 1.21+
- **Data Format**: JSON streaming over FIFO
- **Communication**: HTTP/gRPC for service integration
- **Storage**: In-memory buffering with WAL persistence
- **Observability**: OpenTelemetry tracing and Prometheus metrics

---

## Architecture

### High-Level Billing Architecture

```mermaid
graph TB
    subgraph "VM Runtime Layer"
        VM1[VM Instance 1<br/>- Resource Usage<br/>- Performance Metrics<br/>- Billing Events]
        VM2[VM Instance 2<br/>- Resource Usage<br/>- Performance Metrics<br/>- Billing Events]
        VMN[VM Instance N<br/>- Resource Usage<br/>- Performance Metrics<br/>- Billing Events]
    end
    
    subgraph "Collection Layer"
        FIFO1[FIFO Stream 1<br/>- 100ms Precision<br/>- JSON Format<br/>- Real-time Data]
        FIFO2[FIFO Stream 2<br/>- 100ms Precision<br/>- JSON Format<br/>- Real-time Data]
        FIFON[FIFO Stream N<br/>- 100ms Precision<br/>- JSON Format<br/>- Real-time Data]
    end
    
    subgraph "Billing Service Cluster"
        subgraph "Billing Instance 1"
            Collector1[Metrics Collector<br/>- FIFO Reading<br/>- JSON Parsing<br/>- Data Validation]
            Processor1[Data Processor<br/>- Enrichment<br/>- Aggregation<br/>- Batching]
            Transmitter1[Data Transmitter<br/>- HTTP Client<br/>- Retry Logic<br/>- Error Handling]
        end
        
        subgraph "Billing Instance N"
            CollectorN[Metrics Collector<br/>- FIFO Reading<br/>- JSON Parsing<br/>- Data Validation]
            ProcessorN[Data Processor<br/>- Enrichment<br/>- Aggregation<br/>- Batching]
            TransmitterN[Data Transmitter<br/>- HTTP Client<br/>- Retry Logic<br/>- Error Handling]
        end
    end
    
    subgraph "Downstream Systems"
        Aggregator[Billing Aggregator<br/>- Data Processing<br/>- Business Logic<br/>- Report Generation]
        ClickHouse[(ClickHouse<br/>- Time-series Storage<br/>- Analytics Queries<br/>- Historical Data)]
    end
    
    VM1 --> FIFO1
    VM2 --> FIFO2
    VMN --> FIFON
    
    FIFO1 --> Collector1
    FIFO2 --> Collector1
    FIFON --> CollectorN
    
    Collector1 --> Processor1
    CollectorN --> ProcessorN
    
    Processor1 --> Transmitter1
    ProcessorN --> TransmitterN
    
    Transmitter1 --> Aggregator
    TransmitterN --> Aggregator
    
    Aggregator --> ClickHouse
```

### Data Flow Architecture

```mermaid
sequenceDiagram
    participant VM
    participant FIFO
    participant Collector
    participant Processor
    participant Transmitter
    participant Aggregator
    participant ClickHouse
    
    Note over VM,ClickHouse: Initialization
    VM->>FIFO: Initialize Metrics Stream
    Collector->>FIFO: Start Reading
    Processor->>Transmitter: Initialize Batch
    Transmitter->>Aggregator: Register Billing Client
    
    loop Every 100ms
        VM->>FIFO: Write Metrics JSON
        FIFO->>Collector: Read JSON Data
        Collector->>Collector: Parse & Validate
        Collector->>Processor: Processed Metrics
        Processor->>Processor: Enrich & Aggregate
        
        alt Batch Full or Timeout
            Processor->>Transmitter: Send Batch
            Transmitter->>Aggregator: HTTP POST /metrics
            Aggregator->>Aggregator: Process Batch
            Aggregator->>ClickHouse: Store Metrics
            Aggregator->>Transmitter: Acknowledge
            Transmitter->>Processor: Success
        end
    end
    
    Note over VM,ClickHouse: Error Handling
    alt Collection Error
        Collector->>Processor: Error Event
        Processor->>Transmitter: Error Report
        Transmitter->>Aggregator: POST /errors
    end
    
    Note over VM,ClickHouse: Heartbeat
    loop Every 30s
        Transmitter->>Aggregator: Heartbeat
        Aggregator->>Transmitter: Health Status
    end
```

---

## Component Details

### Metrics Collector

```mermaid
graph TB
    subgraph "FIFO Stream Management"
        FIFOMonitor[FIFO Monitor<br/>- Stream Health<br/>- Data Availability<br/>- Error Detection]
        StreamReader[Stream Reader<br/>- Non-blocking Reads<br/>- Buffer Management<br/>- EOF Handling]
    end
    
    subgraph "JSON Processing"
        JSONParser[JSON Parser<br/>- Streaming Parser<br/>- Error Recovery<br/>- Schema Validation]
        DataValidator[Data Validator<br/>- Required Fields<br/>- Type Checking<br/>- Range Validation]
    end
    
    subgraph "Metrics Enrichment"
        CustomerContext[Customer Context<br/>- Customer ID<br/>- VM Metadata<br/>- Billing Context]
        TimestampNormalization[Timestamp Normalization<br/>- UTC Conversion<br/>- Precision Handling<br/>- Clock Drift Correction]
    end
    
    subgraph "Error Handling"
        ErrorRecovery[Error Recovery<br/>- Parse Errors<br/>- Invalid Data<br/>- Stream Interruption]
        DataLoss[Data Loss Prevention<br/>- Buffer Overflow<br/>- Stream Reconnection<br/>- Checkpoint Recovery]
    end
    
    FIFOMonitor --> StreamReader
    StreamReader --> JSONParser
    JSONParser --> DataValidator
    DataValidator --> CustomerContext
    CustomerContext --> TimestampNormalization
    
    JSONParser --> ErrorRecovery
    StreamReader --> DataLoss
```

### Data Processor

```mermaid
graph TB
    subgraph "Data Aggregation"
        TimeWindowing[Time Windowing<br/>- 100ms Windows<br/>- Sliding Windows<br/>- Alignment Handling]
        MetricsAggregation[Metrics Aggregation<br/>- Sum, Average, Max<br/>- Resource Usage<br/>- Performance Metrics]
    end
    
    subgraph "Batch Management"
        BatchBuilder[Batch Builder<br/>- Size Limits<br/>- Time Limits<br/>- Compression]
        QualityControl[Quality Control<br/>- Data Completeness<br/>- Consistency Checks<br/>- Outlier Detection]
    end
    
    subgraph "Data Enrichment"
        BillingCalculation[Billing Calculation<br/>- Resource Costs<br/>- Usage Multiplication<br/>- Rate Application]
        MetadataInjection[Metadata Injection<br/>- Customer Info<br/>- VM Configuration<br/>- Billing Rules]
    end
    
    subgraph "Performance Optimization"
        MemoryManagement[Memory Management<br/>- Buffer Pooling<br/>- GC Optimization<br/>- Memory Limits]
        Compression[Compression<br/>- Data Compression<br/>- Bandwidth Optimization<br/>- CPU vs Network]
    end
    
    TimeWindowing --> MetricsAggregation
    MetricsAggregation --> BatchBuilder
    BatchBuilder --> QualityControl
    QualityControl --> BillingCalculation
    BillingCalculation --> MetadataInjection
    
    BatchBuilder --> MemoryManagement
    QualityControl --> Compression
```

---

## Metrics Collection

### FIFO Stream Processing

```mermaid
graph LR
    subgraph "VM Process"
        VMRuntime[VM Runtime<br/>- Hypervisor Metrics<br/>- Resource Usage<br/>- Performance Data]
        MetricsExporter[Metrics Exporter<br/>- JSON Serialization<br/>- Timestamp Addition<br/>- Format Validation]
    end
    
    subgraph "FIFO Interface"
        FIFOFile[FIFO File<br/>- Named Pipe<br/>- Binary Stream<br/>- OS Buffer]
        StreamBuffer[Stream Buffer<br/>- Line Buffering<br/>- JSON Boundaries<br/>- Partial Reads]
    end
    
    subgraph "Billing Collector"
        FIFOReader[FIFO Reader<br/>- Non-blocking I/O<br/>- Timeout Handling<br/>- Error Recovery]
        JSONDecoder[JSON Decoder<br/>- Streaming Decoder<br/>- Error Handling<br/>- Performance Optimization]
    end
    
    VMRuntime --> MetricsExporter
    MetricsExporter --> FIFOFile
    FIFOFile --> StreamBuffer
    StreamBuffer --> FIFOReader
    FIFOReader --> JSONDecoder
```

### Metrics Data Format

```json
{
  "timestamp": "2025-06-12T10:30:00.100Z",
  "vm_id": "vm-12345",
  "customer_id": "customer-abc123",
  "metrics": {
    "cpu": {
      "usage_percent": 45.2,
      "cores": 2,
      "cycles": 1234567890
    },
    "memory": {
      "used_bytes": 536870912,
      "total_bytes": 1073741824,
      "cache_bytes": 134217728
    },
    "network": {
      "rx_bytes": 1048576,
      "tx_bytes": 2097152,
      "rx_packets": 1024,
      "tx_packets": 2048
    },
    "disk": {
      "read_bytes": 4194304,
      "write_bytes": 8388608,
      "read_ops": 256,
      "write_ops": 512
    }
  },
  "billing_context": {
    "rate_id": "rate-vm-standard",
    "region": "us-east-1",
    "tier": "standard"
  }
}
```

### Collection Performance Profile

```mermaid
graph TB
    subgraph "Throughput Characteristics"
        CollectionRate[Collection Rate<br/>10 metrics/second<br/>Per VM Instance]
        ProcessingRate[Processing Rate<br/>1000 metrics/second<br/>Per Collector Instance]
        TransmissionRate[Transmission Rate<br/>100 batches/second<br/>Per Transmitter]
    end
    
    subgraph "Latency Profile"
        CollectionLatency[Collection Latency<br/>< 10ms<br/>FIFO to Collector]
        ProcessingLatency[Processing Latency<br/>< 50ms<br/>Parse to Batch]
        TransmissionLatency[Transmission Latency<br/>< 200ms<br/>Batch to Aggregator]
    end
    
    subgraph "Resource Usage"
        CPUUsage[CPU Usage<br/>< 10%<br/>Per 100 VMs]
        MemoryUsage[Memory Usage<br/>< 500MB<br/>Per Collector]
        NetworkUsage[Network Usage<br/>< 10Mbps<br/>Per Transmitter]
    end
    
    CollectionRate --> CollectionLatency
    ProcessingRate --> ProcessingLatency
    TransmissionRate --> TransmissionLatency
    
    CollectionLatency --> CPUUsage
    ProcessingLatency --> MemoryUsage
    TransmissionLatency --> NetworkUsage
```

---

## Data Processing Pipeline

### Stream Processing Flow

```mermaid
graph TB
    subgraph "Input Stage"
        RawFIFO[Raw FIFO Data<br/>- JSON Lines<br/>- Timestamps<br/>- Binary Stream]
        ParsedJSON[Parsed JSON<br/>- Structured Data<br/>- Validated Schema<br/>- Type Safety]
    end
    
    subgraph "Validation Stage"
        SchemaCheck[Schema Validation<br/>- Required Fields<br/>- Data Types<br/>- Format Validation]
        BusinessRules[Business Rules<br/>- Value Ranges<br/>- Consistency Checks<br/>- Data Quality]
    end
    
    subgraph "Enrichment Stage"
        CustomerLookup[Customer Lookup<br/>- Customer Metadata<br/>- Billing Context<br/>- Rate Information]
        BillingCalculation[Billing Calculation<br/>- Usage Calculation<br/>- Rate Application<br/>- Cost Computation]
    end
    
    subgraph "Aggregation Stage"
        TimeWindowing[Time Windowing<br/>- 100ms Windows<br/>- Boundary Alignment<br/>- Late Data Handling]
        MetricsRollup[Metrics Rollup<br/>- Statistical Aggregation<br/>- Resource Summation<br/>- Performance Metrics]
    end
    
    subgraph "Output Stage"
        BatchFormation[Batch Formation<br/>- Size Optimization<br/>- Compression<br/>- Format Conversion]
        QualityAssurance[Quality Assurance<br/>- Completeness Check<br/>- Integrity Validation<br/>- Error Detection]
    end
    
    RawFIFO --> ParsedJSON
    ParsedJSON --> SchemaCheck
    SchemaCheck --> BusinessRules
    BusinessRules --> CustomerLookup
    CustomerLookup --> BillingCalculation
    BillingCalculation --> TimeWindowing
    TimeWindowing --> MetricsRollup
    MetricsRollup --> BatchFormation
    BatchFormation --> QualityAssurance
```

### Batch Processing Strategy

```mermaid
sequenceDiagram
    participant Collector
    participant Processor
    participant Batcher
    participant Transmitter
    
    Note over Collector,Transmitter: Continuous Processing
    loop Metrics Collection
        Collector->>Processor: Raw Metrics
        Processor->>Processor: Validate & Enrich
        Processor->>Batcher: Processed Metrics
        
        alt Batch Size Reached (1000 metrics)
            Batcher->>Transmitter: Send Batch
            Transmitter->>Batcher: Ack Success
            Batcher->>Batcher: Reset Batch
        else Timeout Reached (5 seconds)
            Batcher->>Transmitter: Send Partial Batch
            Transmitter->>Batcher: Ack Success
            Batcher->>Batcher: Reset Batch
        else Error Condition
            Batcher->>Transmitter: Send Error Batch
            Transmitter->>Batcher: Ack Error
            Batcher->>Batcher: Retry Logic
        end
    end
```

---

## Integration Architecture

### Billing Aggregator Integration

```mermaid
graph TB
    subgraph "Billing Service"
        Transmitter[Data Transmitter<br/>- HTTP Client<br/>- Retry Logic<br/>- Circuit Breaker]
        HeartbeatManager[Heartbeat Manager<br/>- Health Monitoring<br/>- Status Reporting<br/>- Connection Management]
    end
    
    subgraph "Billing Aggregator"
        APIGateway[API Gateway<br/>- Authentication<br/>- Rate Limiting<br/>- Load Balancing]
        MetricsEndpoint[/metrics Endpoint<br/>- Batch Processing<br/>- Validation<br/>- Acknowledgment]
        HealthEndpoint[/health Endpoint<br/>- Service Status<br/>- Dependency Health<br/>- Performance Metrics]
    end
    
    subgraph "Downstream Processing"
        DataProcessor[Data Processor<br/>- Business Logic<br/>- Rate Calculation<br/>- Invoice Generation]
        ClickHouseWriter[ClickHouse Writer<br/>- Batch Insertion<br/>- Schema Management<br/>- Performance Optimization]
    end
    
    Transmitter --> APIGateway
    HeartbeatManager --> APIGateway
    
    APIGateway --> MetricsEndpoint
    APIGateway --> HealthEndpoint
    
    MetricsEndpoint --> DataProcessor
    DataProcessor --> ClickHouseWriter
```

### API Contract Specification

```yaml
# Billing Aggregator API Contract
openapi: 3.0.3
info:
  title: Billing Aggregator API
  version: 1.0.0

paths:
  /v1/metrics:
    post:
      summary: Submit billing metrics batch
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                batch_id:
                  type: string
                  format: uuid
                timestamp:
                  type: string
                  format: date-time
                metrics:
                  type: array
                  items:
                    $ref: '#/components/schemas/MetricRecord'
      responses:
        '200':
          description: Batch accepted
          content:
            application/json:
              schema:
                type: object
                properties:
                  status: 
                    type: string
                    enum: [accepted, partial, rejected]
                  batch_id:
                    type: string
                  processed_count:
                    type: integer
                  error_count:
                    type: integer
        '400':
          description: Invalid batch format
        '429':
          description: Rate limit exceeded
        '500':
          description: Internal server error

  /v1/health:
    get:
      summary: Health check
      responses:
        '200':
          description: Service healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    enum: [healthy, degraded, unhealthy]
                  timestamp:
                    type: string
                    format: date-time
                  dependencies:
                    type: object
                    additionalProperties:
                      type: string

components:
  schemas:
    MetricRecord:
      type: object
      required: [vm_id, customer_id, timestamp, metrics]
      properties:
        vm_id:
          type: string
        customer_id:
          type: string
        timestamp:
          type: string
          format: date-time
        metrics:
          type: object
          properties:
            cpu:
              $ref: '#/components/schemas/CPUMetrics'
            memory:
              $ref: '#/components/schemas/MemoryMetrics'
            network:
              $ref: '#/components/schemas/NetworkMetrics'
            disk:
              $ref: '#/components/schemas/DiskMetrics'
```

### Error Handling Strategy

```mermaid
stateDiagram-v2
    [*] --> Normal : Start
    
    Normal --> Retry : HTTP Error (5xx)
    Normal --> Failed : Client Error (4xx)
    Normal --> Circuit_Open : Repeated Failures
    
    Retry --> Normal : Success
    Retry --> Failed : Max Retries Exceeded
    Retry --> Circuit_Open : Pattern Detection
    
    Circuit_Open --> Circuit_Half_Open : Timeout Expired
    Circuit_Half_Open --> Normal : Test Success
    Circuit_Half_Open --> Circuit_Open : Test Failed
    
    Failed --> Normal : Manual Reset
    
    note right of Normal
        Process requests normally
        Monitor success rate
        Track response times
    end note
    
    note right of Retry
        Exponential backoff
        Jitter added
        Limited attempts
    end note
    
    note right of Circuit_Open
        Fail fast responses
        Preserve resources
        Error accumulation
    end note
```

---

## Reliability & Recovery

### Failure Recovery Mechanisms

```mermaid
graph TB
    subgraph "Failure Detection"
        HealthMonitoring[Health Monitoring<br/>- Service Health<br/>- Connection Status<br/>- Response Times]
        ErrorTracking[Error Tracking<br/>- Error Rates<br/>- Error Types<br/>- Pattern Detection]
        ThresholdAlerting[Threshold Alerting<br/>- Error Thresholds<br/>- Latency Thresholds<br/>- Availability Targets]
    end
    
    subgraph "Recovery Actions"
        AutoRetry[Automatic Retry<br/>- Exponential Backoff<br/>- Jitter Addition<br/>- Max Attempts]
        CircuitBreaking[Circuit Breaking<br/>- Failure Isolation<br/>- Resource Protection<br/>- Automatic Recovery]
        DataRecovery[Data Recovery<br/>- Checkpoint Recovery<br/>- Data Replay<br/>- Loss Minimization]
    end
    
    subgraph "Escalation Procedures"
        AlertGeneration[Alert Generation<br/>- Operations Team<br/>- Severity Levels<br/>- Escalation Rules]
        FallbackMode[Fallback Mode<br/>- Degraded Operation<br/>- Essential Functions<br/>- Manual Override]
        IncidentManagement[Incident Management<br/>- Incident Tracking<br/>- Resolution Procedures<br/>- Post-mortem Analysis]
    end
    
    HealthMonitoring --> AutoRetry
    ErrorTracking --> CircuitBreaking
    ThresholdAlerting --> DataRecovery
    
    AutoRetry --> AlertGeneration
    CircuitBreaking --> FallbackMode
    DataRecovery --> IncidentManagement
```

### Data Loss Prevention

```mermaid
sequenceDiagram
    participant VM
    participant FIFO
    participant Collector
    participant WAL as Write-Ahead Log
    participant Processor
    participant Transmitter
    
    Note over VM,Transmitter: Normal Operation
    VM->>FIFO: Write Metrics
    FIFO->>Collector: Read Metrics
    Collector->>WAL: Log Raw Data
    Collector->>Processor: Process Metrics
    Processor->>Transmitter: Send Batch
    Transmitter->>WAL: Mark Transmitted
    
    Note over VM,Transmitter: Failure Scenario
    VM->>FIFO: Write Metrics
    FIFO->>Collector: Read Metrics
    Collector->>WAL: Log Raw Data
    Collector->>Processor: Process Metrics
    Processor->>Transmitter: Send Batch
    Transmitter->>Transmitter: Network Error
    
    Note over VM,Transmitter: Recovery Process
    Transmitter->>WAL: Check Untransmitted
    WAL->>Transmitter: Provide Checkpoint
    Transmitter->>Processor: Replay Data
    Processor->>Transmitter: Rebuild Batch
    Transmitter->>Transmitter: Retry Transmission
```

### Heartbeat Monitoring

```mermaid
graph LR
    subgraph "Heartbeat Generation"
        Timer[Heartbeat Timer<br/>30-second Interval]
        StatusCollector[Status Collector<br/>- Service Health<br/>- Performance Metrics<br/>- Error Counts]
        MessageBuilder[Message Builder<br/>- JSON Format<br/>- Timestamp<br/>- Metadata]
    end
    
    subgraph "Transmission"
        HTTPClient[HTTP Client<br/>- POST /heartbeat<br/>- Timeout Handling<br/>- Retry Logic]
        ResponseHandler[Response Handler<br/>- Success Tracking<br/>- Error Handling<br/>- Status Updates]
    end
    
    subgraph "Failure Handling"
        FailureCounter[Failure Counter<br/>- Consecutive Failures<br/>- Threshold Detection<br/>- Alert Triggering]
        RecoveryProcedure[Recovery Procedure<br/>- Connection Reset<br/>- Service Restart<br/>- Manual Intervention]
    end
    
    Timer --> StatusCollector
    StatusCollector --> MessageBuilder
    MessageBuilder --> HTTPClient
    HTTPClient --> ResponseHandler
    ResponseHandler --> FailureCounter
    FailureCounter --> RecoveryProcedure
```

---

## Performance Characteristics

### Throughput Specifications

| Metric | Specification | Notes |
|--------|---------------|--------|
| **VM Metrics Rate** | 10 metrics/second/VM | 100ms precision |
| **Collector Throughput** | 1,000 metrics/second | Per collector instance |
| **Batch Processing** | 100 batches/second | Per processor instance |
| **Transmission Rate** | 50 batches/second | Per transmitter instance |
| **Memory Usage** | 500MB per 1,000 VMs | Including buffers |
| **CPU Overhead** | 5% per 1,000 VMs | Collection + processing |
| **Network Bandwidth** | 10Mbps per 10,000 VMs | Compressed batches |

### Latency Profile

```mermaid
graph TB
    subgraph "End-to-End Latency"
        VMtoFIFO[VM to FIFO<br/>< 1ms<br/>Local Write]
        FIFOtoCollector[FIFO to Collector<br/>< 10ms<br/>Local Read]
        ProcessingLatency[Processing<br/>< 50ms<br/>Parse + Validate]
        BatchingLatency[Batching<br/>< 100ms<br/>Aggregation]
        TransmissionLatency[Transmission<br/>< 200ms<br/>Network + Processing]
    end
    
    subgraph "Percentile Targets"
        P50[P50 Latency<br/>< 150ms<br/>Typical Case]
        P95[P95 Latency<br/>< 300ms<br/>High Load]
        P99[P99 Latency<br/>< 500ms<br/>Worst Case]
    end
    
    VMtoFIFO --> P50
    FIFOtoCollector --> P50
    ProcessingLatency --> P95
    BatchingLatency --> P95
    TransmissionLatency --> P99
```

### Scalability Characteristics

```mermaid
graph TB
    subgraph "Horizontal Scaling"
        CollectorScaling[Collector Scaling<br/>- 1 Collector per 1,000 VMs<br/>- Stateless Design<br/>- Auto-scaling]
        ProcessorScaling[Processor Scaling<br/>- CPU-bound Scaling<br/>- Memory Management<br/>- Load Balancing]
        TransmitterScaling[Transmitter Scaling<br/>- Network-bound Scaling<br/>- Connection Pooling<br/>- Circuit Breaking]
    end
    
    subgraph "Vertical Scaling"
        CPUScaling[CPU Scaling<br/>- JSON Processing<br/>- Data Validation<br/>- Compression]
        MemoryScaling[Memory Scaling<br/>- Buffer Management<br/>- Batch Accumulation<br/>- Cache Usage]
        NetworkScaling[Network Scaling<br/>- Bandwidth Requirements<br/>- Connection Limits<br/>- Throughput Optimization]
    end
    
    subgraph "Scaling Limits"
        MaxVMs[Maximum VMs<br/>10,000 per Instance]
        MaxThroughput[Maximum Throughput<br/>100,000 metrics/sec]
        MaxMemory[Maximum Memory<br/>8GB per Instance]
    end
    
    CollectorScaling --> CPUScaling
    ProcessorScaling --> MemoryScaling
    TransmitterScaling --> NetworkScaling
    
    CPUScaling --> MaxVMs
    MemoryScaling --> MaxThroughput
    NetworkScaling --> MaxMemory
```

---

## Operational Considerations

### Monitoring & Alerting

```yaml
# Billing System Metrics
metrics:
  # Collection Metrics
  billing_metrics_collected_total:
    type: counter
    description: "Total metrics collected from VMs"
    labels: [vm_id, customer_id, collector_instance]
  
  billing_collection_duration_seconds:
    type: histogram
    description: "Time to collect and process metrics"
    labels: [operation_type, result]
  
  billing_fifo_read_errors_total:
    type: counter
    description: "FIFO read errors"
    labels: [vm_id, error_type]
  
  # Processing Metrics
  billing_batch_size:
    type: histogram
    description: "Size of batches sent to aggregator"
    labels: [batch_type]
  
  billing_processing_lag_seconds:
    type: gauge
    description: "Processing lag behind real-time"
    labels: [processor_instance]
  
  # Transmission Metrics
  billing_batches_sent_total:
    type: counter
    description: "Batches sent to billing aggregator"
    labels: [result, aggregator_endpoint]
  
  billing_transmission_duration_seconds:
    type: histogram
    description: "Time to transmit batch to aggregator"
    labels: [result]
  
  # Heartbeat Metrics
  billing_heartbeat_sent_total:
    type: counter
    description: "Heartbeats sent to aggregator"
    labels: [result]
  
  billing_heartbeat_failures_total:
    type: counter
    description: "Failed heartbeat attempts"
    labels: [failure_reason]

alerts:
  # Collection Alerts
  - alert: BillingCollectionLag
    expr: billing_processing_lag_seconds > 300
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Billing collection lag exceeds 5 minutes"
  
  - alert: BillingFIFOErrors
    expr: rate(billing_fifo_read_errors_total[5m]) > 0.1
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "High rate of FIFO read errors"
  
  # Transmission Alerts
  - alert: BillingTransmissionFailures
    expr: rate(billing_batches_sent_total{result="error"}[5m]) / rate(billing_batches_sent_total[5m]) > 0.05
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High billing transmission failure rate"
  
  - alert: BillingHeartbeatFailures
    expr: rate(billing_heartbeat_failures_total[5m]) > 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Billing service heartbeat failures detected"
```

### Configuration Management

```yaml
# Billing Service Configuration
billing:
  collection:
    fifo_timeout: "1s"
    read_buffer_size: "64KB"
    max_read_size: "1MB"
    poll_interval: "10ms"
    
  processing:
    batch_size: 1000
    batch_timeout: "5s"
    worker_count: 4
    max_memory_mb: 500
    
    validation:
      schema_validation: true
      business_rules: true
      outlier_detection: true
      max_value_ranges:
        cpu_percent: 100
        memory_bytes: "10GB"
        network_mbps: 1000
  
  transmission:
    aggregator_endpoint: "https://billing-aggregator.internal/v1"
    timeout: "30s"
    retry_attempts: 3
    retry_backoff: "1s"
    max_retry_backoff: "30s"
    
    circuit_breaker:
      failure_threshold: 5
      recovery_timeout: "60s"
      half_open_requests: 3
    
    compression:
      enabled: true
      algorithm: "gzip"
      level: 6
  
  heartbeat:
    interval: "30s"
    timeout: "10s"
    endpoint: "/v1/heartbeat"
    failure_threshold: 3
  
  observability:
    metrics_enabled: true
    tracing_enabled: true
    log_level: "info"
    
    sampling:
      trace_ratio: 0.1
      error_traces: 1.0
      slow_requests: 1.0
```

### Capacity Planning

```mermaid
graph TB
    subgraph "Resource Requirements"
        VMCount[VM Count<br/>Target: 10,000 VMs]
        MetricsRate[Metrics Rate<br/>100,000 metrics/sec]
        DataVolume[Data Volume<br/>1GB/hour compressed]
    end
    
    subgraph "Infrastructure Sizing"
        CPUCores[CPU Cores<br/>8 cores per 10k VMs]
        Memory[Memory<br/>8GB per 10k VMs]
        NetworkBandwidth[Network Bandwidth<br/>100Mbps per 10k VMs]
        StorageIOPS[Storage IOPS<br/>1000 IOPS for WAL]
    end
    
    subgraph "Scaling Strategy"
        HorizontalScaling[Horizontal Scaling<br/>Add Instances]
        VerticalScaling[Vertical Scaling<br/>Increase Resources]
        GeographicDistribution[Geographic Distribution<br/>Regional Deployment]
    end
    
    VMCount --> CPUCores
    MetricsRate --> Memory
    DataVolume --> NetworkBandwidth
    DataVolume --> StorageIOPS
    
    CPUCores --> HorizontalScaling
    Memory --> VerticalScaling
    NetworkBandwidth --> GeographicDistribution
```

---

## Cross-References

### Architecture Documentation
- **[System Architecture Overview](../overview.md)** - Complete system design
- **[Metald Architecture](metald.md)** - VM management integration
- **[ClickHouse Architecture](clickhouse.md)** - Data storage integration

### API Documentation
- **[API Reference](../../api/reference.md)** - Complete API documentation
- **[Configuration Guide](../../api/configuration.md)** - Configuration options

### Operational Documentation
- **[Production Deployment](../../deployment/production.md)** - Deployment procedures
- **[Monitoring Setup](../../deployment/monitoring-setup.md)** - Observability setup
- **[Reliability Guide](../../operations/reliability.md)** - Operational procedures

### Development Documentation
- **[Testing Guide](../../development/testing/stress-testing.md)** - Load testing procedures
- **[Contribution Guide](../../development/contribution-guide.md)** - Development setup

---

*Last updated: 2025-06-12 | Next review: Billing Architecture Review*