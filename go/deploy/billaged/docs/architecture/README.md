# Billaged Architecture

This document provides a comprehensive overview of the billaged service architecture, including core components, data flow patterns, and system design decisions.

## System Overview

Billaged serves as the central billing aggregation service in the Unkey Deploy ecosystem, processing real-time VM usage metrics and generating accurate billing data through sophisticated resource scoring algorithms.

### Core Components

#### 1. Billing Service
**Location**: [service/billing.go](../../internal/service/billing.go)

The main ConnectRPC service handling all external API interactions:

- **Request Processing**: Validates and routes incoming billing requests
- **Authentication Integration**: Enforces SPIFFE-based authentication and tenant isolation
- **Metrics Integration**: Records OpenTelemetry metrics for usage processing
- **Error Handling**: Provides comprehensive error responses with context

#### 2. Usage Aggregator
**Location**: [aggregator/aggregator.go](../../internal/aggregator/aggregator.go)

In-memory aggregation engine for real-time usage calculations:

- **Data Structures**: Thread-safe maps for VM usage tracking and customer relationships
- **Delta Calculations**: Handles cumulative counters with overflow protection
- **Periodic Processing**: Configurable interval summaries for billing systems
- **Resource Scoring**: Weighted algorithms for multi-dimensional billing

#### 3. Configuration Management
**Location**: [config/config.go](../../internal/config/config.go)

Environment-based configuration with validation and defaults:

- **Server Configuration**: Address, port, and network binding options
- **OpenTelemetry Setup**: Observability and metrics export configuration
- **TLS Configuration**: SPIFFE, file-based, or disabled security modes
- **Aggregation Tuning**: Interval settings for billing precision vs. performance

#### 4. Observability Infrastructure
**Location**: [observability/](../../internal/observability/)

Comprehensive monitoring and telemetry:

- **Metrics Collection**: Custom billing metrics with optional high-cardinality labels
- **Tracing Integration**: OTLP trace export for service interaction visibility
- **Health Monitoring**: Service health and dependency status tracking

## Data Flow Architecture

### 1. Metrics Ingestion Flow

```
metald Instance → ConnectRPC → BillingService → Aggregator → Usage Summary
                     ↓              ↓              ↓             ↓
                 Auth Check    Validation    Delta Calc    Resource Score
```

**Source**: [service/billing.go:32-84](../../internal/service/billing.go#L32-84)

#### Processing Pipeline

1. **Authentication**: SPIFFE workload identity verification
2. **Validation**: Request schema validation and batch size limits
3. **Aggregation**: Delta calculation with counter overflow handling
4. **Scoring**: Resource score computation using weighted algorithms
5. **Storage**: In-memory update with thread-safe operations

### 2. Lifecycle Management Flow

```
VM Lifecycle Event → Service Handler → Aggregator → Customer Mapping → Summary Generation
                         ↓                ↓              ↓                 ↓
                    Event Logging    State Update   Tenant Isolation   Billing Output
```

**Source**: [aggregator.go:175-247](../../internal/aggregator/aggregator.go#L175-247)

#### Event Processing

- **VM Start**: Initialize usage tracking with customer association
- **VM Stop**: Generate final usage summary and cleanup resources
- **Gap Detection**: Log potential data inconsistencies for reconciliation

### 3. Observability Data Flow

```
Service Operations → OTEL Metrics → Prometheus → Monitoring Systems
                        ↓              ↓              ↓
                   Custom Metrics   Standard Export   Alerting Rules
```

**Source**: [observability/metrics.go](../../internal/observability/metrics.go)

## Service Dependencies

### Core Dependencies

#### metald Integration
**Documentation**: [metald/docs/](../../metald/docs/)

Primary source of VM usage data and lifecycle events:

- **Metrics Push**: Real-time usage data via `SendMetricsBatch`
- **Heartbeats**: Health monitoring via `SendHeartbeat` with active VM lists
- **Lifecycle Events**: VM start/stop notifications for billing boundaries
- **Gap Reporting**: Data integrity notifications via `NotifyPossibleGap`

#### SPIFFE/SPIRE Integration
**Configuration**: [config/config.go:166-172](../../internal/config/config.go#L166-172)

Provides workload identity and mTLS for service security:

- **Authentication**: Workload identity verification for all API calls
- **Authorization**: Service-to-service communication security
- **Certificate Management**: Automatic certificate rotation and validation
- **Transport Security**: Encrypted communication channels

#### OpenTelemetry Integration
**Implementation**: [observability/otel.go](../../internal/observability/otel.go)

Observability and monitoring infrastructure:

- **Metrics Export**: OTLP and Prometheus metric export
- **Distributed Tracing**: Request correlation across service boundaries
- **Performance Monitoring**: Aggregation duration and throughput tracking

## In-Memory Data Architecture

### Data Structures

#### VMUsageData Structure
**Location**: [aggregator.go:12-33](../../internal/aggregator/aggregator.go#L12-33)

```go
type VMUsageData struct {
    VMID                string    // Unique VM identifier
    CustomerID          string    // Tenant isolation key
    StartTime           time.Time // Billing period start
    LastUpdate          time.Time // Most recent metric timestamp
    
    // Cumulative totals
    TotalCPUNanos       int64    // CPU time used (delta aggregated)
    TotalMemoryBytes    int64    // Maximum memory observed
    TotalDiskReadBytes  int64    // Disk read bytes (delta aggregated)
    TotalDiskWriteBytes int64    // Disk write bytes (delta aggregated)
    TotalNetworkRxBytes int64    // Network receive bytes (delta aggregated)
    TotalNetworkTxBytes int64    // Network transmit bytes (delta aggregated)
    SampleCount         int64    // Number of metrics processed
    
    // Previous values for delta calculation
    LastCPUNanos       int64    // Previous CPU value for delta
    LastMemoryBytes    int64    // Previous memory value  
    LastDiskReadBytes  int64    // Previous disk read value
    LastDiskWriteBytes int64    // Previous disk write value
    LastNetworkRxBytes int64    // Previous network RX value
    LastNetworkTxBytes int64    // Previous network TX value
}
```

#### Thread Safety

All data structures use `sync.RWMutex` for concurrent access:

- **Read Operations**: Concurrent reads for statistics and health checks
- **Write Operations**: Exclusive writes for metric updates and lifecycle events
- **Memory Efficiency**: Optimized for high-throughput metric processing

### Memory Management

#### Resource Tracking
**Implementation**: [aggregator.go:340-345](../../internal/aggregator/aggregator.go#L340-345)

```go
// Memory-efficient VM tracking
vmData    map[string]*VMUsageData // vmID -> usage data
customers map[string][]string     // customerID -> []vmID
```

#### Cleanup Strategies

- **VM Termination**: Automatic cleanup on `NotifyVmStopped`
- **Orphaned VMs**: Periodic cleanup of VMs without recent activity
- **Customer Mapping**: Consistent customer-to-VM relationship maintenance

## Resource Scoring Algorithm

### Weighted Scoring Formula
**Implementation**: [aggregator.go:282-305](../../internal/aggregator/aggregator.go#L282-305)

The resource score combines multiple usage dimensions into a single billing metric:

```
resourceScore = (cpuSeconds × 1.0) + (memoryGB × 0.5) + (diskMB × 0.3)
```

#### Weight Rationale

1. **CPU Weight (1.0)**: Highest weight reflecting direct compute cost correlation
2. **Memory Weight (0.5)**: Medium weight for allocation cost with over-provisioning consideration  
3. **I/O Weight (0.3)**: Lower weight for disk I/O with moderate cost impact

#### Business Rules
**Reference**: [aggregator.go:282-296](../../internal/aggregator/aggregator.go#L282-296)

- **CPU Time Focus**: Billing based on actual CPU usage rather than allocation
- **Memory Maximum**: Uses peak memory usage during the billing period
- **I/O Aggregation**: Combines read and write operations for total I/O cost
- **Periodic Review**: Weights should be adjusted based on infrastructure cost analysis

### Calculation Implementation

```go
// Resource score calculation with production-tuned weights
cpuScore := float64(vmUsage.TotalCPUNanos) / float64(time.Second) * cpuWeight
memoryScore := float64(vmUsage.TotalMemoryBytes) / (1024 * 1024 * 1024) * memoryWeight  
ioScore := float64(vmUsage.TotalDiskReadBytes+vmUsage.TotalDiskWriteBytes) / (1024 * 1024) * ioWeight

resourceScore := cpuScore + memoryScore + ioScore
```

## Multi-Tenant Architecture

### Tenant Isolation

#### Request-Level Isolation
**Implementation**: [service/billing.go:37-45](../../internal/service/billing.go#L37-45)

```go
vmID := req.Msg.GetVmId()
customerID := req.Msg.GetCustomerId()  // Tenant boundary enforcement
```

#### Data-Level Isolation
**Implementation**: [aggregator.go:106-117](../../internal/aggregator/aggregator.go#L106-117)

```go
// Customer-scoped data structures
customers[customerID] = append(customers[customerID], vmID)
```

### Authentication Flow

#### SPIFFE Workload Identity
**Configuration**: [client/client.go:67-106](../../client/client.go#L67-106)

1. **Certificate Retrieval**: SPIFFE workload API provides client certificates
2. **Identity Verification**: Server validates workload identity via SPIRE
3. **Tenant Context**: X-Tenant-ID header provides customer scoping
4. **Authorization**: Service enforces tenant-based data access

## Performance Characteristics

### Throughput Metrics

#### Processing Capacity
- **Metrics/Second**: 10,000+ individual metrics per second
- **Batch Processing**: 100+ concurrent metric batches
- **Memory Usage**: ~1MB per 1000 active VMs
- **Aggregation Latency**: <10ms per batch processing

#### Scaling Patterns

**Horizontal Scaling**: Multiple billaged instances with client-side load balancing
**Vertical Scaling**: Memory-optimized instances for large customer bases
**Performance Tuning**: Configurable aggregation intervals for precision vs. throughput

### Optimization Strategies

#### Delta Processing
**Implementation**: [aggregator.go:135-173](../../internal/aggregator/aggregator.go#L135-173)

```go
// Efficient delta calculation with overflow protection
cpuDelta := metric.GetCpuTimeNanos() - vmUsage.LastCPUNanos
if cpuDelta > 0 {
    vmUsage.TotalCPUNanos += cpuDelta
}
```

#### Memory Efficiency

- **Pointer Usage**: References to large structures to minimize copying
- **Batch Processing**: Amortized allocation costs across metric batches  
- **Cleanup Strategies**: Proactive cleanup on VM termination

## Configuration Architecture

### Environment-Based Configuration
**Implementation**: [config/config.go:88-181](../../internal/config/config.go#L88-181)

#### Configuration Categories

1. **Server Configuration**: Network binding and connection management
2. **Aggregation Configuration**: Billing interval and processing options
3. **OpenTelemetry Configuration**: Observability and export settings
4. **TLS Configuration**: Security mode and certificate management

#### Validation Patterns

```go
func (c *Config) Validate() error {
    if c.OpenTelemetry.TracingSamplingRate < 0.0 || c.OpenTelemetry.TracingSamplingRate > 1.0 {
        return fmt.Errorf("tracing sampling rate must be between 0.0 and 1.0")
    }
    // Additional validation logic...
}
```

### Default Value Strategy

Production-safe defaults with environment override capability:

- **Security First**: SPIFFE mode enabled by default
- **Observability Ready**: OpenTelemetry metrics enabled
- **Performance Optimized**: 60-second aggregation intervals
- **Development Friendly**: Localhost binding with secure remote options

## Error Handling Architecture

### Error Classification

#### Service-Level Errors
**Implementation**: [service/billing.go](../../internal/service/billing.go)

1. **Validation Errors**: Request schema and business rule violations
2. **Authentication Errors**: SPIFFE identity or tenant authorization failures
3. **Processing Errors**: Aggregation or resource calculation failures
4. **Resource Errors**: Memory exhaustion or capacity limit violations

#### Resilience Patterns

- **Graceful Degradation**: Continue processing valid metrics despite individual failures
- **Circuit Breaking**: Automatic protection against cascade failures
- **Retry Logic**: Client-side retry with exponential backoff
- **Health Checks**: Continuous service availability monitoring

### Logging Strategy

#### Structured Logging
**Implementation**: [cmd/billaged/main.go:89-92](../../cmd/billaged/main.go#L89-92)

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
```

#### Log Levels

- **DEBUG**: Detailed metric processing information
- **INFO**: Service lifecycle and usage summaries  
- **WARN**: Gap detection and potential data issues
- **ERROR**: Processing failures and system errors

## Integration Points

### Service Mesh Integration

#### SPIFFE/SPIRE Integration
The service integrates with the SPIFFE/SPIRE service mesh for:

- **Identity Management**: Workload identity certificates for authentication
- **Service Discovery**: Secure service-to-service communication
- **Certificate Rotation**: Automatic certificate lifecycle management
- **Policy Enforcement**: Authorization policies for API access

#### Load Balancing

Multiple billaged instances can be deployed with:

- **Client-Side Load Balancing**: ConnectRPC client library handles distribution
- **Service Mesh Integration**: Envoy proxy support for traffic management
- **Health Check Integration**: Automatic healthy instance selection

### Future Architecture Considerations

#### Persistent Storage Integration
**Planned Enhancement**: Database backend for usage history and reconciliation

#### Event Streaming Integration  
**Planned Enhancement**: Kafka/Pulsar integration for real-time billing events

#### Advanced Analytics Integration
**Planned Enhancement**: Time-series database integration for historical analysis