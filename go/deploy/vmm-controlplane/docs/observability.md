# Observability Guide

This guide covers the OpenTelemetry implementation in the Cloud Hypervisor Control Plane.

## Overview

CHCP includes comprehensive observability features powered by OpenTelemetry (OTEL):
- **Distributed tracing** for all RPC and HTTP requests
- **Metrics collection** with dual export (OTLP push + Prometheus pull)
- **Parent-based sampling** with configurable rates
- **Always-on error capture** regardless of sampling rate
- **LAN-accessible** observability stack

## Quick Start

### 1. Start the Observability Stack

```bash
# Start Grafana LGTM stack (Loki, Grafana, Tempo, Mimir)
make o11y

# Stack will be available at:
# - Grafana UI: http://0.0.0.0:3000 (admin/admin)
# - OTLP HTTP: 0.0.0.0:4318
# - OTLP gRPC: 0.0.0.0:4317
```

### 2. Run CHCP with Telemetry

```bash
# Enable OpenTelemetry
export UNKEY_CHCP_OTEL_ENABLED=true
export UNKEY_CHCP_OTEL_SAMPLING_RATE=1.0
export UNKEY_CHCP_OTEL_ENDPOINT=localhost:4318

# Start the API
./build/chcp-api
```

### 3. View Telemetry Data

1. Open Grafana at http://localhost:3000
2. Navigate to Explore → Tempo for traces
3. Navigate to Explore → Prometheus for metrics

## Configuration Reference

### Environment Variables

| Variable | Description | Default | Valid Values |
|----------|-------------|---------|--------------|
| `UNKEY_CHCP_OTEL_ENABLED` | Enable/disable OpenTelemetry | `false` | `true`, `false` |
| `UNKEY_CHCP_OTEL_SERVICE_NAME` | Service name in telemetry | `cloud-hypervisor-controlplane` | Any string |
| `UNKEY_CHCP_OTEL_SERVICE_VERSION` | Service version | `0.0.1` | Any string |
| `UNKEY_CHCP_OTEL_SAMPLING_RATE` | Trace sampling ratio | `1.0` | `0.0` to `1.0` |
| `UNKEY_CHCP_OTEL_ENDPOINT` | OTLP HTTP endpoint | `localhost:4318` | Host:port |
| `UNKEY_CHCP_OTEL_PROMETHEUS_ENABLED` | Enable Prometheus endpoint | `true` | `true`, `false` |
| `UNKEY_CHCP_OTEL_PROMETHEUS_PORT` | Prometheus metrics port | `9464` | Any port |

### Sampling Configuration

CHCP uses a sophisticated sampling strategy:

1. **Parent-Based Sampling**: If a request has an existing trace context (from an upstream service), we honor its sampling decision
2. **Local Sampling Rate**: For new traces (no parent), we use `UNKEY_CHCP_OTEL_SAMPLING_RATE`
3. **Always-On for Errors**: Any span with an error is resampled and exported, even if the original sampling decision was "don't sample"

Example sampling scenarios:
- `SAMPLING_RATE=1.0`: Sample all traces (100%)
- `SAMPLING_RATE=0.1`: Sample 10% of new traces, but honor parent decisions
- `SAMPLING_RATE=0.0`: Only sample errors and traces with sampled parents

## Metrics

### Available Metrics

#### RPC Metrics
- `rpc_server_requests_total`: Counter of total RPC requests
  - Labels: `rpc.method`, `rpc.status`
- `rpc_server_request_duration_seconds`: Histogram of request durations
  - Labels: `rpc.method`
- `rpc_server_active_requests`: Gauge of currently active requests
  - Labels: `rpc.method`

#### System Metrics (via health endpoint)
- CPU usage
- Memory usage
- System uptime
- Backend health status

### Prometheus Scraping

Metrics are exposed for Prometheus scraping at:
```
http://0.0.0.0:9464/metrics
```

Example Prometheus configuration:
```yaml
scrape_configs:
  - job_name: 'chcp'
    static_configs:
      - targets: ['your-server-ip:9464']
```

## Distributed Tracing

### Trace Propagation

CHCP supports W3C Trace Context propagation:
- Incoming requests: Trace context is extracted from headers
- Outgoing requests: Trace context is injected into headers
- Backend calls: All Cloud Hypervisor API calls include trace context

### Trace Attributes

Each span includes:
- `service.name`: cloud-hypervisor-controlplane
- `service.version`: 0.0.1
- `service.namespace`: unkey
- `rpc.system`: connect_rpc
- `rpc.method`: The RPC method name
- `rpc.service`: The full service path
- `http.method`, `http.url`, etc. for HTTP spans

## Production Deployment

### Resource Considerations

1. **OTLP Endpoint**: Ensure your OTLP collector can handle the load
2. **Sampling Rate**: In production, consider lower sampling rates (0.01-0.1)
3. **Metrics Cardinality**: Monitor label combinations to avoid high cardinality

### Example Production Config

```bash
# Production settings
export UNKEY_CHCP_OTEL_ENABLED=true
export UNKEY_CHCP_OTEL_SERVICE_NAME=chcp-prod
export UNKEY_CHCP_OTEL_SERVICE_VERSION=1.0.0
export UNKEY_CHCP_OTEL_SAMPLING_RATE=0.05  # 5% sampling
export UNKEY_CHCP_OTEL_ENDPOINT=otel-collector.internal:4318
export UNKEY_CHCP_OTEL_PROMETHEUS_ENABLED=true
export UNKEY_CHCP_OTEL_PROMETHEUS_PORT=9464
```

### Security Considerations

1. **Network Access**: Both the API and metrics endpoints bind to `0.0.0.0`
   - Use firewall rules to restrict access
   - Consider TLS termination at a load balancer
2. **Sensitive Data**: Ensure no sensitive data in span attributes or metrics labels
3. **Resource Limits**: Set appropriate limits on your OTLP collector

## Troubleshooting

### No Traces Appearing

1. Check OTEL is enabled: `UNKEY_CHCP_OTEL_ENABLED=true`
2. Verify endpoint is reachable: `curl http://localhost:4318/v1/traces`
3. Check sampling rate isn't 0: `UNKEY_CHCP_OTEL_SAMPLING_RATE > 0`
4. Look for errors in logs about OTLP export failures

### High Memory Usage

1. Reduce sampling rate
2. Check for metric cardinality explosion
3. Ensure OTLP collector is accepting data (backpressure can cause buffering)

### Schema Conflicts

If you see "conflicting Schema URL" errors:
- This is resolved by using semconv v1.24.0 with OTEL v1.36.0
- See the AIDEV-NOTE in `internal/observability/otel.go`

## Development Tips

### Testing Different Sampling Rates

```bash
# Manual testing with different sampling rates
export UNKEY_CHCP_OTEL_ENABLED=true

# Test 0% sampling (only errors)
export UNKEY_CHCP_OTEL_SAMPLING_RATE=0.0
./build/chcp-api

# Test 50% sampling
export UNKEY_CHCP_OTEL_SAMPLING_RATE=0.5
./build/chcp-api

# Test 100% sampling
export UNKEY_CHCP_OTEL_SAMPLING_RATE=1.0
./build/chcp-api
```

**Note**: ✅ **Unit tests implemented** for OTEL configuration in `internal/config/config_test.go`. Future work should include:
- Integration tests with mock OTLP endpoints
- Automated verification of trace/metric collection

### Local Development Setup

```bash
# Terminal 1: Start observability stack
make o11y

# Terminal 2: Run with full sampling
export UNKEY_CHCP_OTEL_ENABLED=true
export UNKEY_CHCP_OTEL_SAMPLING_RATE=1.0
make run

# Terminal 3: Generate test traffic
curl -X POST http://localhost:8080/cloudhypervisor.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{"cpus":{"boot_vcpus":2},"memory":{"size":1024}}'
```

### Viewing Raw Metrics

```bash
# See all available metrics
curl -s http://localhost:9464/metrics

# Filter for specific metrics
curl -s http://localhost:9464/metrics | grep rpc_server
```

## Integration with CI/CD

For CI/CD pipelines, you can disable OTEL to avoid unnecessary overhead:

```yaml
# GitHub Actions example
env:
  UNKEY_CHCP_OTEL_ENABLED: false
```

Or enable with specific settings for integration tests:

```yaml
env:
  UNKEY_CHCP_OTEL_ENABLED: true
  UNKEY_CHCP_OTEL_SAMPLING_RATE: 1.0
  UNKEY_CHCP_OTEL_ENDPOINT: otel-collector:4318
```