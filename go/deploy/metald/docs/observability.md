# Observability

metald includes OpenTelemetry integration for comprehensive monitoring.

## Quick Setup

### 1. Enable Telemetry
```bash
export UNKEY_METALD_OTEL_ENABLED=true
export UNKEY_METALD_OTEL_SAMPLING_RATE=1.0
./build/metald
```

### 2. Access Metrics
```bash
# Prometheus metrics
curl http://localhost:9464/metrics

# Key metrics:
# - metald_vm_operations_total
# - metald_metrics_collected_total
# - metald_billing_batches_sent_total
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `UNKEY_METALD_OTEL_ENABLED` | Enable OpenTelemetry | `false` |
| `UNKEY_METALD_OTEL_SAMPLING_RATE` | Trace sampling (0.0-1.0) | `1.0` |
| `UNKEY_METALD_OTEL_ENDPOINT` | OTLP endpoint | `localhost:4318` |
| `UNKEY_METALD_OTEL_PROMETHEUS_PORT` | Metrics port | `9464` |

## Key Features

- **Distributed Tracing**: Full request tracing across VM operations
- **Metrics Export**: Dual OTLP push + Prometheus pull
- **Parent-based Sampling**: Honors upstream trace decisions
- **Error Capture**: Always captures errors regardless of sampling rate

## Available Metrics

### VM Operations
- `metald_vm_operations_total{operation, status}` - VM operation counts
- `metald_vm_operation_duration_seconds{operation}` - Operation timing

### Billing Metrics
- `metald_metrics_collected_total{vm_id}` - Metrics collection counts
- `metald_billing_batches_sent_total{vm_id}` - Billing batch counts
- `metald_heartbeat_sent_total` - Heartbeat counts

### System Metrics
- `go_*` - Standard Go runtime metrics
- HTTP and gRPC metrics via OpenTelemetry auto-instrumentation

## Monitoring Setup

For production monitoring, integrate with your existing Prometheus/Grafana stack:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'metald'
    static_configs:
      - targets: ['localhost:9464']
    scrape_interval: 10s
```

## Tracing

When connected to an OTLP-compatible backend (Jaeger, Tempo), metald provides detailed traces for:
- VM lifecycle operations (create, boot, shutdown, delete)
- Billing metrics collection and streaming
- Backend communication (Firecracker, Cloud Hypervisor)
- Health check operations
