# Billaged Documentation

Welcome to the billaged service documentation. Billaged is the VM usage billing aggregation service that collects and processes virtual machine resource usage metrics for accurate billing and cost tracking across the Unkey Deploy infrastructure.

## Documentation Navigation

### [API Documentation](api/README.md)
Complete reference for all BillingService RPCs:
- Service endpoints and methods with streaming support
- Request/response schemas and examples
- VM metrics collection patterns
- Heartbeat and lifecycle notifications
- Error handling and gap detection

### [Architecture Guide](architecture/README.md)
Deep dive into the service design:
- Real-time metrics aggregation architecture
- Service dependencies and interaction patterns
- Resource score calculation algorithms
- Memory-based storage and periodic processing
- Customer isolation and multi-tenancy

### [Operations Manual](operations/README.md)
Production deployment and management:
- Installation and configuration
- Monitoring and metrics collection
- Health checks and observability setup
- Performance tuning for high-throughput
- Troubleshooting common issues

### [Development Setup](development/README.md)
Getting started with development:
- Build instructions and dependencies
- Local development environment setup
- Testing strategies with mock data
- Integration testing patterns
- Client library usage examples

## Quick Links

- [Service Overview](../) - Main README with key features
- [Billing Proto Definition](../proto/billing/v1/billing.proto) - Protocol buffer definitions
- [Configuration Reference](../internal/config/config.go) - Environment variables and settings

## Service Role

Billaged is one of the four pillar services in Unkey Deploy, responsible for:
- **Usage Metrics Collection** - Real-time aggregation of VM CPU, memory, disk, and network usage
- **Billing Calculations** - Resource score algorithms for accurate cost attribution
- **Customer Isolation** - Multi-tenant usage tracking with secure data separation
- **Lifecycle Management** - VM start/stop notifications and gap detection
- **Performance Aggregation** - Configurable interval summaries for billing systems

## Service Dependencies

### Core Dependencies
- **[metald](../../metald/docs/README.md)** - Primary source of VM usage metrics and lifecycle events
- **SPIFFE/Spire** - mTLS authentication and service authorization
- **OpenTelemetry** - Observability, metrics export, and tracing

### Data Flow Dependencies
- **metald** sends usage metrics via [`SendMetricsBatch`](../proto/billing/v1/billing.proto#L10)
- **metald** sends heartbeats via [`SendHeartbeat`](../proto/billing/v1/billing.proto#L11) 
- **metald** sends lifecycle events via [`NotifyVmStarted`](../proto/billing/v1/billing.proto#L12) and [`NotifyVmStopped`](../proto/billing/v1/billing.proto#L13)

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     metald      │    │   Other VMs     │    │  Monitoring     │
│   (VM Host)     │    │   (via metald)  │    │   Systems       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │ Metrics/Lifecycle     │ Metrics              │ Prometheus
         │ Events (ConnectRPC)   │ Events               │ Scraping
         │                       │                       │
┌─────────────────────────────────────────────────────────────────┐
│                          billaged                               │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Billing       │  │ Aggregator   │  │ Observability        │ │
│  │ Service       │  │ (In-Memory)  │  │ (OpenTelemetry)      │ │
│  │ (ConnectRPC)  │  │              │  │                      │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
│  ┌───────────────┐  ┌──────────────┐  ┌──────────────────────┐ │
│  │ Config        │  │ Client       │  │ CLI Tool             │ │
│  │ Management    │  │ Library      │  │ (billaged-cli)       │ │
│  └───────────────┘  └──────────────┘  └──────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
         │                       │                       │
         │ Usage Summaries       │ HTTP/gRPC APIs       │ OTLP Export
         │ (Periodic Logs)       │                       │
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Billing Systems │    │ Client Apps     │    │ OTEL Collector │
│ (Log Processing)│    │ (Integration)   │    │ (Jaeger/OTLP)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Getting Started

### Installation

```bash
# Build from source
cd billaged
make build

# Install with systemd
sudo make install
```

### Basic Configuration

```bash
# Minimal configuration for development
export UNKEY_BILLAGED_PORT=8081
export UNKEY_BILLAGED_ADDRESS=0.0.0.0
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s
export UNKEY_BILLAGED_TLS_MODE=disabled
export UNKEY_BILLAGED_OTEL_ENABLED=false

./build/billaged
```

### Send Your First Metrics

```bash
# Using the CLI tool
cd cmd/billaged-cli
go run main.go -server=http://localhost:8081 send-metrics

# Or using the client library
cd client
go run example.go
```

## Key Features

- **Real-time Aggregation**: Sub-second metric processing with configurable aggregation intervals
- **Resource Scoring**: Weighted algorithm combining CPU, memory, and I/O usage for billing accuracy
- **Gap Detection**: Automatic identification and notification of missing metrics data
- **Multi-tenant Isolation**: Customer-scoped data aggregation with secure tenant boundaries
- **Production Observability**: Comprehensive metrics, tracing, and health monitoring
- **High Performance**: In-memory aggregation with optimized data structures for throughput

## Quick Start Examples

### Client Integration

```go
// Create billaged client with SPIFFE authentication
client, err := client.New(ctx, client.Config{
    ServerAddress: "https://billaged:8081",
    UserID:       "metald-instance-1", 
    TenantID:     "customer-abc",
    TLSMode:      "spiffe",
})

// Send VM metrics batch
resp, err := client.SendMetricsBatch(ctx, &client.SendMetricsBatchRequest{
    VmID:       "vm-12345",
    CustomerID: "customer-abc", 
    Metrics:    metricsData,
})
```

### Metrics Monitoring

```bash
# Check service stats
curl http://localhost:8081/stats

# Prometheus metrics (if enabled)
curl http://localhost:9465/metrics

# Health check
curl http://localhost:8081/health
```

## API Highlights

The service exposes a ConnectRPC API with the following main operations:

- [`SendMetricsBatch`](../proto/billing/v1/billing.proto#L10) - Process batches of VM usage metrics
- [`SendHeartbeat`](../proto/billing/v1/billing.proto#L11) - Heartbeat with active VM list for health monitoring  
- [`NotifyVmStarted`](../proto/billing/v1/billing.proto#L12) - VM lifecycle start notifications
- [`NotifyVmStopped`](../proto/billing/v1/billing.proto#L13) - VM lifecycle stop notifications
- [`NotifyPossibleGap`](../proto/billing/v1/billing.proto#L14) - Data gap detection and reconciliation

See [API Documentation](./api/README.md) for complete reference with examples.

## Production Deployment

### System Requirements

- **OS**: Linux with systemd support
- **CPU**: 2+ cores for high-throughput metric processing  
- **Memory**: 4GB+ for large-scale VM metric aggregation
- **Storage**: 10GB+ for logs and temporary metric storage
- **Network**: Low-latency connection to metald instances

### Key Configuration

```bash
# Production environment variables
export UNKEY_BILLAGED_TLS_MODE=spiffe                    # Enable mTLS
export UNKEY_BILLAGED_OTEL_ENABLED=true                  # Enable observability  
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s           # Billing aggregation frequency
export UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED=true       # Metrics export
export UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false # Limit cardinality
```

## Monitoring

Key metrics to monitor in production:

- `billaged_usage_records_processed_total` - Metric processing rate by customer
- `billaged_aggregation_duration_seconds` - Aggregation performance histograms  
- `billaged_active_vms` - Current VM tracking count
- `billaged_billing_errors_total` - Processing failures by error type

Source: [observability/metrics.go](../internal/observability/metrics.go)

See [Operations Guide](./operations/README.md) for complete monitoring setup.

## Development

### Building from Source

```bash
git clone https://github.com/unkeyed/unkey
cd go/deploy/billaged
make build  # Builds to build/billaged
```

### Running Tests

```bash
# Unit tests
make test

# Service integration tests  
go test ./internal/service/...

# Aggregator tests
go test ./internal/aggregator/...
```

See [Development Setup](./development/README.md) for detailed instructions.

## Version

Current version: **v0.1.0** (Initial release with real-time aggregation and resource scoring)

Source: [cmd/billaged/main.go:38](../cmd/billaged/main.go#L38)