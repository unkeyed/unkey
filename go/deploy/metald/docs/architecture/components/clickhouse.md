# ClickHouse Analytics Architecture

> High-performance columnar database for billing analytics and time-series data

## Table of Contents
- [Overview](#overview)
- [Architecture](#architecture)
- [Schema Design](#schema-design)
- [Data Ingestion](#data-ingestion)
- [Query Optimization](#query-optimization)
- [Cluster Architecture](#cluster-architecture)
- [Data Lifecycle Management](#data-lifecycle-management)
- [Performance Characteristics](#performance-characteristics)
- [Operational Considerations](#operational-considerations)
- [Cross-References](#cross-references)

---

## Overview

ClickHouse serves as the primary analytics database for the Unkey platform, providing high-performance storage and querying capabilities for billing data, usage analytics, and operational metrics.

### Key Responsibilities
- **Time-Series Storage**: High-performance storage for billing metrics and usage data
- **Real-Time Analytics**: Fast analytical queries for dashboards and reporting
- **Data Aggregation**: Pre-computed aggregations for performance optimization
- **Historical Analysis**: Long-term data retention for trend analysis
- **Customer Analytics**: Usage patterns and billing insights
- **Operational Metrics**: System performance and health monitoring

### Technology Stack
- **Database**: ClickHouse 23.x+ (Columnar OLAP database)
- **Cluster Management**: ClickHouse Keeper for coordination
- **Data Format**: Native ClickHouse format with compression
- **Replication**: Multi-master replication for high availability
- **Backup**: Incremental backups with point-in-time recovery

---

## Architecture

### High-Level ClickHouse Architecture

```mermaid
graph TB
    subgraph "Data Sources"
        BillingService[Billing Service<br/>- Real-time Metrics<br/>- Batch Processing<br/>- JSON Format]
        BillingAggregator[Billing Aggregator<br/>- Data Processing<br/>- Business Logic<br/>- Batch Optimization]
        MetricsExporter[Metrics Exporter<br/>- System Metrics<br/>- Performance Data<br/>- Health Monitoring]
    end
    
    subgraph "Ingestion Layer"
        DataLoader[Data Loader<br/>- Batch Processing<br/>- Format Conversion<br/>- Validation]
        StreamProcessor[Stream Processor<br/>- Real-time Ingestion<br/>- Kafka Integration<br/>- Buffer Management]
    end
    
    subgraph "ClickHouse Cluster"
        subgraph "Shard 1"
            CH1_1[ClickHouse Node 1.1<br/>- Primary Replica<br/>- Write Operations<br/>- Local Storage]
            CH1_2[ClickHouse Node 1.2<br/>- Secondary Replica<br/>- Read Operations<br/>- Backup Storage]
        end
        
        subgraph "Shard 2"
            CH2_1[ClickHouse Node 2.1<br/>- Primary Replica<br/>- Write Operations<br/>- Local Storage]
            CH2_2[ClickHouse Node 2.2<br/>- Secondary Replica<br/>- Read Operations<br/>- Backup Storage]
        end
        
        subgraph "Shard N"
            CHN_1[ClickHouse Node N.1<br/>- Primary Replica<br/>- Write Operations<br/>- Local Storage]
            CHN_2[ClickHouse Node N.2<br/>- Secondary Replica<br/>- Read Operations<br/>- Backup Storage]
        end
        
        Keeper[ClickHouse Keeper<br/>- Cluster Coordination<br/>- Metadata Management<br/>- Leader Election]
    end
    
    subgraph "Query Layer"
        QueryRouter[Query Router<br/>- Query Planning<br/>- Shard Selection<br/>- Load Balancing]
        ResultAggregator[Result Aggregator<br/>- Cross-shard Queries<br/>- Result Merging<br/>- Performance Optimization]
    end
    
    subgraph "Application Layer"
        Dashboard[Analytics Dashboard<br/>- Real-time Charts<br/>- Usage Reports<br/>- Performance Metrics]
        ReportingService[Reporting Service<br/>- Billing Reports<br/>- Usage Analysis<br/>- Cost Analytics]
        AlertingSystem[Alerting System<br/>- Threshold Monitoring<br/>- Anomaly Detection<br/>- Notification Delivery]
    end
    
    BillingService --> DataLoader
    BillingAggregator --> DataLoader
    MetricsExporter --> StreamProcessor
    
    DataLoader --> CH1_1
    DataLoader --> CH2_1
    StreamProcessor --> CHN_1
    
    CH1_1 --> CH1_2
    CH2_1 --> CH2_2
    CHN_1 --> CHN_2
    
    CH1_1 --> Keeper
    CH2_1 --> Keeper
    CHN_1 --> Keeper
    
    QueryRouter --> CH1_1
    QueryRouter --> CH2_1
    QueryRouter --> CHN_1
    
    QueryRouter --> ResultAggregator
    ResultAggregator --> Dashboard
    ResultAggregator --> ReportingService
    ResultAggregator --> AlertingSystem
```

### Data Flow Architecture

```mermaid
sequenceDiagram
    participant Billing as Billing Service
    participant Aggregator as Billing Aggregator
    participant Loader as Data Loader
    participant ClickHouse as ClickHouse
    participant Dashboard as Analytics Dashboard
    
    Note over Billing,Dashboard: Real-time Data Flow
    Billing->>Aggregator: Metrics Batch
    Aggregator->>Aggregator: Process & Validate
    Aggregator->>Loader: Processed Data
    Loader->>Loader: Format Conversion
    Loader->>ClickHouse: INSERT Batch
    ClickHouse->>ClickHouse: Distribute to Shards
    ClickHouse->>ClickHouse: Replicate Data
    
    Note over Billing,Dashboard: Query Processing
    Dashboard->>ClickHouse: Analytics Query
    ClickHouse->>ClickHouse: Query Planning
    ClickHouse->>ClickHouse: Cross-shard Execution
    ClickHouse->>ClickHouse: Result Aggregation
    ClickHouse->>Dashboard: Query Results
    
    Note over Billing,Dashboard: Background Processing
    loop Every Hour
        ClickHouse->>ClickHouse: Merge Parts
        ClickHouse->>ClickHouse: Optimize Tables
        ClickHouse->>ClickHouse: Update Materialized Views
    end
```

---

## Schema Design

### Core Tables Structure

```sql
-- Billing Metrics Table (Main fact table)
CREATE TABLE billing_metrics (
    timestamp DateTime64(3) CODEC(Delta, ZSTD),
    vm_id String CODEC(ZSTD),
    customer_id String CODEC(ZSTD),
    
    -- Resource Usage Metrics
    cpu_usage_percent Float32 CODEC(ZSTD),
    cpu_cores UInt8 CODEC(ZSTD),
    memory_used_bytes UInt64 CODEC(Delta, ZSTD),
    memory_total_bytes UInt64 CODEC(ZSTD),
    
    -- Network Metrics
    network_rx_bytes UInt64 CODEC(Delta, ZSTD),
    network_tx_bytes UInt64 CODEC(Delta, ZSTD),
    network_rx_packets UInt64 CODEC(Delta, ZSTD),
    network_tx_packets UInt64 CODEC(Delta, ZSTD),
    
    -- Disk Metrics
    disk_read_bytes UInt64 CODEC(Delta, ZSTD),
    disk_write_bytes UInt64 CODEC(Delta, ZSTD),
    disk_read_ops UInt64 CODEC(Delta, ZSTD),
    disk_write_ops UInt64 CODEC(Delta, ZSTD),
    
    -- Billing Context
    rate_id String CODEC(ZSTD),
    region String CODEC(ZSTD),
    tier String CODEC(ZSTD),
    
    -- Calculated Fields
    cpu_cost Float64 CODEC(ZSTD),
    memory_cost Float64 CODEC(ZSTD),
    network_cost Float64 CODEC(ZSTD),
    disk_cost Float64 CODEC(ZSTD),
    total_cost Float64 CODEC(ZSTD)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/billing_metrics', '{replica}')
PARTITION BY toYYYYMM(timestamp)
ORDER BY (customer_id, vm_id, timestamp)
TTL timestamp + INTERVAL 2 YEAR
SETTINGS index_granularity = 8192;

-- Customer Dimension Table
CREATE TABLE customers (
    customer_id String,
    customer_name String,
    tier String,
    region String,
    created_at DateTime,
    updated_at DateTime,
    
    -- Billing Information
    billing_email String,
    payment_method String,
    currency String,
    
    -- Usage Limits
    max_vms UInt32,
    max_cpu_hours UInt64,
    max_memory_gb_hours UInt64,
    
    -- Status
    status Enum8('active' = 1, 'suspended' = 2, 'deleted' = 3)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/customers', '{replica}')
ORDER BY customer_id
SETTINGS index_granularity = 8192;

-- VM Dimension Table
CREATE TABLE vms (
    vm_id String,
    customer_id String,
    vm_name String,
    
    -- Configuration
    cpu_cores UInt8,
    memory_gb UInt32,
    disk_gb UInt32,
    
    -- Metadata
    backend String,  -- firecracker, cloudhypervisor
    region String,
    availability_zone String,
    
    -- Lifecycle
    created_at DateTime,
    updated_at DateTime,
    deleted_at Nullable(DateTime),
    status Enum8('creating' = 1, 'running' = 2, 'stopped' = 3, 'deleted' = 4)
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/vms', '{replica}')
ORDER BY (customer_id, vm_id)
SETTINGS index_granularity = 8192;

-- Billing Rates Table
CREATE TABLE billing_rates (
    rate_id String,
    tier String,
    region String,
    
    -- Resource Rates (per hour)
    cpu_rate_per_core Float64,
    memory_rate_per_gb Float64,
    network_rate_per_gb Float64,
    disk_rate_per_gb Float64,
    
    -- Metadata
    currency String,
    effective_from DateTime,
    effective_to Nullable(DateTime),
    created_at DateTime
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/billing_rates', '{replica}')
ORDER BY (tier, region, effective_from)
SETTINGS index_granularity = 8192;
```

### Materialized Views for Performance

```sql
-- Hourly Aggregated Metrics
CREATE MATERIALIZED VIEW billing_metrics_hourly
ENGINE = ReplicatedSummingMergeTree('/clickhouse/tables/{shard}/billing_metrics_hourly', '{replica}')
PARTITION BY toYYYYMM(hour)
ORDER BY (customer_id, vm_id, hour)
AS SELECT
    toStartOfHour(timestamp) AS hour,
    customer_id,
    vm_id,
    
    -- Usage Aggregations
    avg(cpu_usage_percent) AS avg_cpu_usage,
    max(cpu_usage_percent) AS max_cpu_usage,
    avg(memory_used_bytes) AS avg_memory_used,
    max(memory_used_bytes) AS max_memory_used,
    
    -- Network Aggregations
    sum(network_rx_bytes) AS total_network_rx,
    sum(network_tx_bytes) AS total_network_tx,
    sum(network_rx_packets) AS total_network_rx_packets,
    sum(network_tx_packets) AS total_network_tx_packets,
    
    -- Disk Aggregations
    sum(disk_read_bytes) AS total_disk_read,
    sum(disk_write_bytes) AS total_disk_write,
    sum(disk_read_ops) AS total_disk_read_ops,
    sum(disk_write_ops) AS total_disk_write_ops,
    
    -- Cost Aggregations
    sum(cpu_cost) AS total_cpu_cost,
    sum(memory_cost) AS total_memory_cost,
    sum(network_cost) AS total_network_cost,
    sum(disk_cost) AS total_disk_cost,
    sum(total_cost) AS total_cost,
    
    -- Metadata
    rate_id,
    region,
    tier,
    count() AS sample_count
FROM billing_metrics
GROUP BY
    hour,
    customer_id,
    vm_id,
    rate_id,
    region,
    tier;

-- Daily Customer Summary
CREATE MATERIALIZED VIEW customer_daily_summary
ENGINE = ReplicatedSummingMergeTree('/clickhouse/tables/{shard}/customer_daily_summary', '{replica}')
PARTITION BY toYYYYMM(date)
ORDER BY (customer_id, date)
AS SELECT
    toDate(timestamp) AS date,
    customer_id,
    
    -- VM Statistics
    uniq(vm_id) AS unique_vms,
    count() / uniq(vm_id) AS avg_samples_per_vm,
    
    -- Resource Usage
    avg(cpu_usage_percent) AS avg_cpu_usage,
    avg(memory_used_bytes) AS avg_memory_usage,
    sum(network_rx_bytes + network_tx_bytes) AS total_network_traffic,
    sum(disk_read_bytes + disk_write_bytes) AS total_disk_io,
    
    -- Cost Summary
    sum(total_cost) AS daily_cost,
    
    -- Performance Metrics
    quantile(0.95)(cpu_usage_percent) AS p95_cpu_usage,
    quantile(0.95)(memory_used_bytes) AS p95_memory_usage
FROM billing_metrics
GROUP BY
    date,
    customer_id;
```

### Partitioning Strategy

```mermaid
graph TB
    subgraph "Temporal Partitioning"
        YearMonth[Year-Month Partitioning<br/>- PARTITION BY toYYYYMM(timestamp)<br/>- Monthly Granularity<br/>- Optimized for Time-based Queries]
        Daily[Daily Sub-partitioning<br/>- Secondary Partitioning<br/>- Faster Range Queries<br/>- Efficient TTL Application]
    end
    
    subgraph "Customer Sharding"
        CustomerHash[Customer Hash Sharding<br/>- SHARD BY cityHash64(customer_id)<br/>- Even Distribution<br/>- Customer Affinity]
        VMHash[VM Hash Sharding<br/>- Secondary Sharding<br/>- Load Distribution<br/>- Query Optimization]
    end
    
    subgraph "Index Strategy"
        PrimaryIndex[Primary Index<br/>- (customer_id, vm_id, timestamp)<br/>- Customer-first Queries<br/>- Range Scan Optimization]
        SecondaryIndex[Secondary Indexes<br/>- Skip Indexes<br/>- MinMax Indexes<br/>- Set Indexes]
    end
    
    YearMonth --> CustomerHash
    Daily --> VMHash
    CustomerHash --> PrimaryIndex
    VMHash --> SecondaryIndex
```

---

## Data Ingestion

### Batch Ingestion Pipeline

```mermaid
graph TB
    subgraph "Data Sources"
        BillingAggregator[Billing Aggregator<br/>- Processed Metrics<br/>- JSON Format<br/>- Batch Ready]
    end
    
    subgraph "Ingestion Pipeline"
        DataValidator[Data Validator<br/>- Schema Validation<br/>- Data Quality Checks<br/>- Error Handling]
        FormatConverter[Format Converter<br/>- JSON to ClickHouse<br/>- Type Conversion<br/>- Optimization]
        Deduplicator[Deduplicator<br/>- Duplicate Detection<br/>- Data Integrity<br/>- Consistency Checks]
    end
    
    subgraph "Load Balancer"
        ShardSelector[Shard Selector<br/>- Customer-based Sharding<br/>- Load Distribution<br/>- Health Checking]
        BatchOptimizer[Batch Optimizer<br/>- Size Optimization<br/>- Compression<br/>- Performance Tuning]
    end
    
    subgraph "ClickHouse Insertion"
        AsyncInsert[Async INSERT<br/>- Non-blocking Writes<br/>- Buffer Management<br/>- Performance Optimization]
        ReplicationManager[Replication Manager<br/>- Multi-replica Writes<br/>- Consistency Guarantees<br/>- Failure Handling]
    end
    
    BillingAggregator --> DataValidator
    DataValidator --> FormatConverter
    FormatConverter --> Deduplicator
    Deduplicator --> ShardSelector
    ShardSelector --> BatchOptimizer
    BatchOptimizer --> AsyncInsert
    AsyncInsert --> ReplicationManager
```

### Real-time Streaming Ingestion

```mermaid
sequenceDiagram
    participant Source as Data Source
    participant Kafka
    participant Consumer as ClickHouse Consumer
    participant Buffer as Buffer Manager
    participant ClickHouse
    
    Note over Source,ClickHouse: Stream Setup
    Source->>Kafka: Publish Metrics
    Consumer->>Kafka: Subscribe to Topic
    Consumer->>Buffer: Initialize Buffer
    
    loop Continuous Streaming
        Kafka->>Consumer: Deliver Message
        Consumer->>Consumer: Parse & Validate
        Consumer->>Buffer: Add to Buffer
        
        alt Buffer Full or Timeout
            Buffer->>ClickHouse: INSERT Batch
            ClickHouse->>Buffer: Acknowledge
            Buffer->>Buffer: Reset Buffer
        end
    end
    
    Note over Source,ClickHouse: Error Handling
    alt Processing Error
        Consumer->>Kafka: Acknowledge Message
        Consumer->>Consumer: Log Error
        Consumer->>Consumer: Continue Processing
    end
```

### Data Quality Assurance

```mermaid
graph LR
    subgraph "Validation Rules"
        SchemaValidation[Schema Validation<br/>- Required Fields<br/>- Data Types<br/>- Format Checks]
        BusinessRules[Business Rules<br/>- Value Ranges<br/>- Consistency Checks<br/>- Relationship Validation]
        DataIntegrity[Data Integrity<br/>- Duplicate Detection<br/>- Completeness Checks<br/>- Sequence Validation]
    end
    
    subgraph "Quality Metrics"
        ValidationRate[Validation Rate<br/>- Success Percentage<br/>- Error Types<br/>- Quality Trends]
        DataCompleteness[Data Completeness<br/>- Missing Data<br/>- Gap Detection<br/>- Coverage Analysis]
        ConsistencyChecks[Consistency Checks<br/>- Cross-table Validation<br/>- Referential Integrity<br/>- Temporal Consistency]
    end
    
    subgraph "Remediation Actions"
        ErrorReporting[Error Reporting<br/>- Detailed Error Logs<br/>- Alert Generation<br/>- Dashboard Updates]
        DataCorrection[Data Correction<br/>- Automated Fixes<br/>- Manual Review<br/>- Reprocessing]
        QualityImprovement[Quality Improvement<br/>- Root Cause Analysis<br/>- Process Enhancement<br/>- Source Optimization]
    end
    
    SchemaValidation --> ValidationRate
    BusinessRules --> DataCompleteness
    DataIntegrity --> ConsistencyChecks
    
    ValidationRate --> ErrorReporting
    DataCompleteness --> DataCorrection
    ConsistencyChecks --> QualityImprovement
```

---

## Query Optimization

### Query Performance Patterns

```sql
-- Optimized Query Examples

-- 1. Customer Usage Report (Optimized)
SELECT 
    customer_id,
    toStartOfMonth(timestamp) AS month,
    sum(total_cost) AS monthly_cost,
    avg(cpu_usage_percent) AS avg_cpu,
    avg(memory_used_bytes) / 1024^3 AS avg_memory_gb
FROM billing_metrics_hourly  -- Use materialized view
WHERE 
    customer_id = 'customer-123'  -- Primary key filter
    AND hour >= '2025-01-01'      -- Partition pruning
    AND hour < '2025-07-01'
GROUP BY customer_id, month
ORDER BY month;

-- 2. Top Customers by Cost (Optimized)
SELECT 
    customer_id,
    sum(total_cost) AS total_cost,
    uniq(vm_id) AS vm_count
FROM customer_daily_summary  -- Use pre-aggregated data
WHERE 
    date >= today() - 30         -- Recent data only
GROUP BY customer_id
ORDER BY total_cost DESC
LIMIT 10;

-- 3. Resource Utilization Analysis (Optimized)
SELECT 
    vm_id,
    customer_id,
    quantile(0.5)(cpu_usage_percent) AS median_cpu,
    quantile(0.95)(cpu_usage_percent) AS p95_cpu,
    quantile(0.5)(memory_used_bytes) AS median_memory,
    quantile(0.95)(memory_used_bytes) AS p95_memory
FROM billing_metrics
WHERE 
    customer_id IN ('customer-1', 'customer-2')  -- IN clause optimization
    AND timestamp >= now() - INTERVAL 7 DAY      -- Recent data
GROUP BY vm_id, customer_id
HAVING median_cpu > 50  -- Filter after aggregation
ORDER BY p95_cpu DESC;
```

### Index Optimization Strategy

```mermaid
graph TB
    subgraph "Primary Indexes"
        CustomerFirst[Customer-First Index<br/>- ORDER BY (customer_id, vm_id, timestamp)<br/>- Customer Queries<br/>- Range Scans]
        TimeFirst[Time-First Index<br/>- ORDER BY (timestamp, customer_id, vm_id)<br/>- Time-series Queries<br/>- Recent Data Access]
    end
    
    subgraph "Skip Indexes"
        MinMaxIndex[MinMax Skip Index<br/>- Numeric Columns<br/>- Range Queries<br/>- Partition Pruning]
        SetIndex[Set Skip Index<br/>- Categorical Columns<br/>- Equality Queries<br/>- Enumeration Filtering]
        BloomFilterIndex[Bloom Filter Index<br/>- String Columns<br/>- Existence Queries<br/>- Memory Efficient]
    end
    
    subgraph "Specialized Indexes"
        DataSkippingIndex[Data Skipping Index<br/>- Custom Functions<br/>- Complex Conditions<br/>- Performance Optimization]
        ProjectionIndex[Projection Index<br/>- Pre-computed Views<br/>- Query Acceleration<br/>- Storage Efficiency]
    end
    
    CustomerFirst --> MinMaxIndex
    TimeFirst --> SetIndex
    MinMaxIndex --> BloomFilterIndex
    SetIndex --> DataSkippingIndex
    BloomFilterIndex --> ProjectionIndex
```

### Query Execution Optimization

```mermaid
sequenceDiagram
    participant Client
    participant QueryRouter
    participant Planner as Query Planner
    participant Executor as Query Executor
    participant Storage
    
    Client->>QueryRouter: Submit Query
    QueryRouter->>Planner: Parse & Analyze
    Planner->>Planner: Cost-based Optimization
    Planner->>Planner: Shard Pruning
    Planner->>Planner: Index Selection
    Planner->>Executor: Execution Plan
    
    Executor->>Storage: Parallel Execution
    Storage->>Storage: Index Scans
    Storage->>Storage: Data Filtering
    Storage->>Storage: Aggregation
    Storage->>Executor: Partial Results
    
    Executor->>Executor: Result Merging
    Executor->>QueryRouter: Final Results
    QueryRouter->>Client: Query Response
```

---

## Cluster Architecture

### Sharding Strategy

```mermaid
graph TB
    subgraph "Shard Distribution"
        Shard1[Shard 1<br/>customer_id hash 0-333]
        Shard2[Shard 2<br/>customer_id hash 334-666]
        Shard3[Shard 3<br/>customer_id hash 667-999]
    end
    
    subgraph "Replication Setup"
        subgraph "Shard 1 Replicas"
            S1R1[Primary Replica<br/>Write Operations]
            S1R2[Secondary Replica<br/>Read Operations]
            S1R3[Backup Replica<br/>Disaster Recovery]
        end
        
        subgraph "Shard 2 Replicas"
            S2R1[Primary Replica<br/>Write Operations]
            S2R2[Secondary Replica<br/>Read Operations]
            S2R3[Backup Replica<br/>Disaster Recovery]
        end
        
        subgraph "Shard 3 Replicas"
            S3R1[Primary Replica<br/>Write Operations]
            S3R2[Secondary Replica<br/>Read Operations]
            S3R3[Backup Replica<br/>Disaster Recovery]
        end
    end
    
    subgraph "Coordination Layer"
        Keeper1[ClickHouse Keeper 1]
        Keeper2[ClickHouse Keeper 2]
        Keeper3[ClickHouse Keeper 3]
    end
    
    Shard1 --> S1R1
    Shard1 --> S1R2
    Shard1 --> S1R3
    
    Shard2 --> S2R1
    Shard2 --> S2R2
    Shard2 --> S2R3
    
    Shard3 --> S3R1
    Shard3 --> S3R2
    Shard3 --> S3R3
    
    S1R1 --> Keeper1
    S2R1 --> Keeper2
    S3R1 --> Keeper3
```

### High Availability Configuration

```xml
<!-- ClickHouse Cluster Configuration -->
<clickhouse>
    <remote_servers>
        <unkey_billing_cluster>
            <shard>
                <replica>
                    <host>ch-shard1-replica1.internal</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>ch-shard1-replica2.internal</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>ch-shard1-replica3.internal</host>
                    <port>9000</port>
                </replica>
            </shard>
            <shard>
                <replica>
                    <host>ch-shard2-replica1.internal</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>ch-shard2-replica2.internal</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>ch-shard2-replica3.internal</host>
                    <port>9000</port>
                </replica>
            </shard>
            <shard>
                <replica>
                    <host>ch-shard3-replica1.internal</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>ch-shard3-replica2.internal</host>
                    <port>9000</port>
                </replica>
                <replica>
                    <host>ch-shard3-replica3.internal</host>
                    <port>9000</port>
                </replica>
            </shard>
        </unkey_billing_cluster>
    </remote_servers>
    
    <keeper_server>
        <tcp_port>9181</tcp_port>
        <server_id from_env="KEEPER_SERVER_ID"/>
        <log_storage_path>/var/lib/clickhouse/coordination/log</log_storage_path>
        <snapshot_storage_path>/var/lib/clickhouse/coordination/snapshots</snapshot_storage_path>
        
        <coordination_settings>
            <operation_timeout_ms>10000</operation_timeout_ms>
            <session_timeout_ms>30000</session_timeout_ms>
            <raft_logs_level>information</raft_logs_level>
        </coordination_settings>
        
        <raft_configuration>
            <server>
                <id>1</id>
                <hostname>keeper1.internal</hostname>
                <port>9234</port>
            </server>
            <server>
                <id>2</id>
                <hostname>keeper2.internal</hostname>
                <port>9234</port>
            </server>
            <server>
                <id>3</id>
                <hostname>keeper3.internal</hostname>
                <port>9234</port>
            </server>
        </raft_configuration>
    </keeper_server>
</clickhouse>
```

---

## Data Lifecycle Management

### Data Retention Strategy

```mermaid
graph TB
    subgraph "Data Tiers"
        HotData[Hot Data<br/>Last 30 Days<br/>- Full Resolution<br/>- SSD Storage<br/>- Fast Queries]
        WarmData[Warm Data<br/>31-365 Days<br/>- Hourly Aggregation<br/>- Standard Storage<br/>- Moderate Performance]
        ColdData[Cold Data<br/>1-2 Years<br/>- Daily Aggregation<br/>- Cold Storage<br/>- Archival Queries]
        ArchivedData[Archived Data<br/>2+ Years<br/>- Monthly Summary<br/>- Object Storage<br/>- Compliance Only]
    end
    
    subgraph "Lifecycle Transitions"
        HotToCold[Hot → Warm<br/>- Automatic TTL<br/>- Data Aggregation<br/>- Storage Migration]
        WarmToCold[Warm → Cold<br/>- Further Aggregation<br/>- Compression<br/>- Access Optimization]
        ColdToArchive[Cold → Archive<br/>- Export to S3<br/>- Local Deletion<br/>- Compliance Retention]
    end
    
    subgraph "Management Policies"
        TTLPolicies[TTL Policies<br/>- Table-level TTL<br/>- Column-level TTL<br/>- Conditional TTL]
        CompressionPolicies[Compression Policies<br/>- Age-based Compression<br/>- Algorithm Selection<br/>- Performance Balance]
        BackupPolicies[Backup Policies<br/>- Incremental Backups<br/>- Point-in-time Recovery<br/>- Cross-region Replication]
    end
    
    HotData --> HotToCold
    WarmData --> WarmToCold
    ColdData --> ColdToArchive
    
    HotToCold --> TTLPolicies
    WarmToCold --> CompressionPolicies
    ColdToArchive --> BackupPolicies
```

### TTL Configuration

```sql
-- Table with TTL for data lifecycle management
CREATE TABLE billing_metrics_with_ttl (
    timestamp DateTime64(3),
    customer_id String,
    vm_id String,
    -- ... other columns
    
    -- Raw data columns (deleted after 30 days)
    raw_cpu_usage Float32 TTL timestamp + INTERVAL 30 DAY,
    raw_memory_usage UInt64 TTL timestamp + INTERVAL 30 DAY,
    
    -- Aggregated data (kept longer)
    hourly_avg_cpu Float32,
    hourly_max_memory UInt64
)
ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/billing_metrics_ttl', '{replica}')
PARTITION BY toYYYYMM(timestamp)
ORDER BY (customer_id, vm_id, timestamp)
TTL 
    timestamp + INTERVAL 30 DAY TO DISK 'cold_storage',  -- Move to cold storage
    timestamp + INTERVAL 1 YEAR TO VOLUME 'archive',     -- Move to archive volume
    timestamp + INTERVAL 2 YEAR DELETE                   -- Delete completely
SETTINGS 
    index_granularity = 8192,
    ttl_only_drop_parts = 1;  -- Drop entire parts when TTL expires

-- Materialized view with different retention
CREATE MATERIALIZED VIEW billing_metrics_daily_summary
ENGINE = ReplicatedSummingMergeTree('/clickhouse/tables/{shard}/billing_daily', '{replica}')
PARTITION BY toYYYYMM(date)
ORDER BY (customer_id, date)
TTL date + INTERVAL 5 YEAR DELETE  -- Keep daily summaries longer
AS SELECT
    toDate(timestamp) AS date,
    customer_id,
    sum(total_cost) AS daily_cost,
    avg(cpu_usage_percent) AS avg_cpu,
    max(memory_used_bytes) AS peak_memory
FROM billing_metrics_with_ttl
GROUP BY date, customer_id;
```

---

## Performance Characteristics

### Query Performance Benchmarks

| Query Type | Dataset Size | Performance Target | Actual Performance |
|------------|--------------|-------------------|-------------------|
| **Real-time Dashboard** | Last 24 hours | < 100ms | 50-80ms |
| **Customer Monthly Report** | 1 month, 1 customer | < 500ms | 200-400ms |
| **Top Customers Analysis** | 30 days, all customers | < 2s | 800ms-1.5s |
| **Resource Utilization** | 7 days, 1000 VMs | < 1s | 400-800ms |
| **Historical Trend Analysis** | 1 year, aggregated | < 5s | 2-4s |
| **Cross-customer Analytics** | 90 days, all customers | < 10s | 5-8s |

### Throughput Specifications

```mermaid
graph TB
    subgraph "Ingestion Performance"
        BatchIngestion[Batch Ingestion<br/>1M records/minute<br/>Per Shard]
        StreamIngestion[Stream Ingestion<br/>100K records/second<br/>Real-time Pipeline]
        BulkLoad[Bulk Loading<br/>10M records/minute<br/>Historical Data Import]
    end
    
    subgraph "Query Performance"
        SimpleQueries[Simple Queries<br/>1K queries/second<br/>Key-value Lookups]
        AnalyticalQueries[Analytical Queries<br/>100 queries/second<br/>Aggregation Heavy]
        ReportingQueries[Reporting Queries<br/>10 queries/second<br/>Complex Analytics]
    end
    
    subgraph "Resource Utilization"
        CPUUtilization[CPU Utilization<br/>< 70% Average<br/>< 90% Peak]
        MemoryUtilization[Memory Utilization<br/>< 80% Average<br/>< 95% Peak]
        DiskIOPS[Disk IOPS<br/>10K IOPS/node<br/>Read-optimized]
    end
    
    BatchIngestion --> SimpleQueries
    StreamIngestion --> AnalyticalQueries
    BulkLoad --> ReportingQueries
    
    SimpleQueries --> CPUUtilization
    AnalyticalQueries --> MemoryUtilization
    ReportingQueries --> DiskIOPS
```

### Scaling Characteristics

```mermaid
graph LR
    subgraph "Data Growth"
        DataVolume[Data Volume<br/>100GB → 10TB<br/>Linear Growth]
        RecordCount[Record Count<br/>1B → 100B records<br/>Time-based Growth]
        CustomerCount[Customer Count<br/>1K → 100K customers<br/>Business Growth]
    end
    
    subgraph "Infrastructure Scaling"
        ShardCount[Shard Count<br/>3 → 30 shards<br/>Horizontal Scaling]
        ReplicaCount[Replica Count<br/>2 → 3 per shard<br/>Availability Scaling]
        NodeCount[Node Count<br/>6 → 90 nodes<br/>Total Infrastructure]
    end
    
    subgraph "Performance Scaling"
        QueryThroughput[Query Throughput<br/>Linear with Shards<br/>Parallel Processing]
        IngestionRate[Ingestion Rate<br/>Linear with Shards<br/>Distributed Writes]
        StorageCapacity[Storage Capacity<br/>Linear with Nodes<br/>Distributed Storage]
    end
    
    DataVolume --> ShardCount
    RecordCount --> ReplicaCount
    CustomerCount --> NodeCount
    
    ShardCount --> QueryThroughput
    ReplicaCount --> IngestionRate
    NodeCount --> StorageCapacity
```

---

## Operational Considerations

### Monitoring & Alerting

```yaml
# ClickHouse Monitoring Configuration
monitoring:
  metrics:
    # Query Performance Metrics
    - name: clickhouse_query_duration_seconds
      description: "Query execution time"
      labels: [query_type, shard, replica]
      thresholds:
        warning: 5s
        critical: 30s
    
    - name: clickhouse_queries_per_second
      description: "Query rate per second"
      labels: [query_type, result]
      thresholds:
        warning: 100
        critical: 10
    
    # Ingestion Metrics
    - name: clickhouse_insert_rows_per_second
      description: "Row insertion rate"
      labels: [table, shard]
      thresholds:
        warning: 10000
        critical: 1000
    
    - name: clickhouse_insert_failures_total
      description: "Failed insert operations"
      labels: [table, error_type]
      thresholds:
        warning: 10/hour
        critical: 100/hour
    
    # Resource Metrics
    - name: clickhouse_memory_usage_bytes
      description: "Memory utilization"
      labels: [node, type]
      thresholds:
        warning: 80%
        critical: 95%
    
    - name: clickhouse_disk_space_usage_percent
      description: "Disk space utilization"
      labels: [node, mount]
      thresholds:
        warning: 80%
        critical: 90%
    
    # Replication Metrics
    - name: clickhouse_replication_lag_seconds
      description: "Replication lag between replicas"
      labels: [shard, replica]
      thresholds:
        warning: 60s
        critical: 300s
    
    - name: clickhouse_replica_health
      description: "Replica health status"
      labels: [shard, replica]
      thresholds:
        critical: 0  # Any unhealthy replica

  alerts:
    # Performance Alerts
    - name: ClickHouseSlowQueries
      condition: clickhouse_query_duration_seconds > 30
      for: 2m
      severity: warning
      description: "ClickHouse queries are running slower than expected"
    
    - name: ClickHouseLowQueryRate
      condition: rate(clickhouse_queries_per_second[5m]) < 10
      for: 5m
      severity: critical
      description: "ClickHouse query rate is critically low"
    
    # Data Ingestion Alerts
    - name: ClickHouseIngestionFailures
      condition: rate(clickhouse_insert_failures_total[5m]) > 0.1
      for: 2m
      severity: warning
      description: "High rate of ClickHouse ingestion failures"
    
    # Resource Alerts
    - name: ClickHouseHighMemoryUsage
      condition: clickhouse_memory_usage_bytes / clickhouse_memory_total_bytes > 0.9
      for: 5m
      severity: warning
      description: "ClickHouse memory usage is high"
    
    - name: ClickHouseDiskSpaceCritical
      condition: clickhouse_disk_space_usage_percent > 90
      for: 1m
      severity: critical
      description: "ClickHouse disk space is critically low"
    
    # Replication Alerts
    - name: ClickHouseReplicationLag
      condition: clickhouse_replication_lag_seconds > 300
      for: 5m
      severity: critical
      description: "ClickHouse replication lag is high"
    
    - name: ClickHouseReplicaDown
      condition: clickhouse_replica_health == 0
      for: 1m
      severity: critical
      description: "ClickHouse replica is unhealthy"
```

### Backup & Disaster Recovery

```mermaid
graph TB
    subgraph "Backup Strategy"
        FullBackup[Full Backup<br/>- Weekly Schedule<br/>- Complete Cluster<br/>- Point-in-time Snapshot]
        IncrementalBackup[Incremental Backup<br/>- Daily Schedule<br/>- Changed Data Only<br/>- Efficient Storage]
        ContinuousBackup[Continuous Backup<br/>- Real-time Replication<br/>- Cross-region Copy<br/>- Zero Data Loss]
    end
    
    subgraph "Storage Locations"
        LocalBackup[Local Backup<br/>- Fast Recovery<br/>- Same Region<br/>- Quick Access]
        RemoteBackup[Remote Backup<br/>- Disaster Recovery<br/>- Different Region<br/>- Compliance]
        CloudBackup[Cloud Backup<br/>- Long-term Storage<br/>- Cost Effective<br/>- Unlimited Capacity]
    end
    
    subgraph "Recovery Procedures"
        PointInTimeRecovery[Point-in-time Recovery<br/>- Specific Timestamp<br/>- Data Consistency<br/>- Partial Restore]
        CompleteRestore[Complete Restore<br/>- Full Cluster Rebuild<br/>- Disaster Scenario<br/>- Business Continuity]
        SelectiveRestore[Selective Restore<br/>- Table-level Recovery<br/>- Data Corruption<br/>- Minimal Impact]
    end
    
    FullBackup --> LocalBackup
    IncrementalBackup --> RemoteBackup
    ContinuousBackup --> CloudBackup
    
    LocalBackup --> PointInTimeRecovery
    RemoteBackup --> CompleteRestore
    CloudBackup --> SelectiveRestore
```

### Capacity Planning

```yaml
# Capacity Planning Configuration
capacity_planning:
  data_growth:
    # Current metrics
    current_data_size: "500GB"
    current_record_count: "5B records"
    current_ingestion_rate: "1M records/hour"
    
    # Growth projections
    monthly_growth_rate: 0.15  # 15% monthly growth
    yearly_projection: "2TB"
    customer_growth_factor: 2.0  # Doubling customers annually
    
  infrastructure_requirements:
    # Current cluster
    current_nodes: 9
    current_shards: 3
    current_replicas: 3
    
    # Scaling triggers
    cpu_threshold: 70%
    memory_threshold: 80%
    disk_threshold: 80%
    query_latency_threshold: "1s"
    
    # Scaling actions
    shard_split_threshold: "100GB per shard"
    node_addition_threshold: "CPU > 80% for 1 week"
    replica_addition_threshold: "Query latency > 2s"
  
  performance_targets:
    # Query performance
    dashboard_queries: "< 100ms"
    report_queries: "< 5s"
    analytical_queries: "< 30s"
    
    # Ingestion performance
    real_time_latency: "< 1s"
    batch_throughput: "1M records/minute"
    
    # Availability targets
    uptime_sla: 99.9%
    recovery_time_objective: "15 minutes"
    recovery_point_objective: "5 minutes"
```

---

## Cross-References

### Architecture Documentation
- **[System Architecture Overview](../overview.md)** - Complete system design
- **[Billing Architecture](billing.md)** - Data source integration
- **[Data Flow Diagrams](../data-flow.md)** - End-to-end data flows

### Operational Documentation
- **[Production Deployment](../../deployment/production.md)** - Deployment procedures
- **[Monitoring Setup](../../deployment/monitoring-setup.md)** - Observability setup
- **[Reliability Guide](../../operations/reliability.md)** - Operational procedures

### Development Documentation
- **[Testing Guide](../../development/testing/stress-testing.md)** - Load testing procedures
- **[Contribution Guide](../../development/contribution-guide.md)** - Development setup

### Reference Documentation
- **[Metrics Reference](../../reference/metrics-reference.md)** - Complete metrics documentation
- **[Error Codes](../../reference/error-codes.md)** - Error handling reference

---

*Last updated: 2025-06-12 | Next review: ClickHouse Architecture Review*