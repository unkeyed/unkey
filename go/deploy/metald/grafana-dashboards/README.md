# Metald Grafana Dashboards

This directory contains pre-built Grafana dashboards for comprehensive metald monitoring.

## Dashboards Overview

### 1. VM Operations Dashboard (`vm-operations.json`)
- **VM Lifecycle Metrics**: Create, boot, shutdown, delete operations
- **Success Rates**: Real-time success/failure rates for all operations
- **Process Management**: Firecracker process creation and management
- **Jailer Integration**: AWS production jailer operations
- **Backend Support**: Firecracker and Cloud Hypervisor metrics

**Key Metrics:**
- `unkey_metald_vm_*_requests_total` - Operation request counts
- `unkey_metald_vm_*_success_total` - Successful operations
- `unkey_metald_vm_*_failures_total` - Failed operations
- `unkey_metald_process_*_total` - Process management metrics
- `unkey_metald_jailer_*_total` - Jailer operations

### 2. Billing & Metrics Dashboard (`billing-metrics.json`)
- **Metrics Collection**: Real-time VM metrics collection (100ms precision)
- **Billing Batches**: Batch transmission to billing service
- **Collection Performance**: Duration and throughput metrics
- **Per-VM Analytics**: Individual VM billing breakdown
- **Heartbeat Monitoring**: Billing service connectivity

**Key Metrics:**
- `metald_metrics_collected_total` - Metrics collection counts
- `metald_billing_batches_sent_total` - Billing batch transmission
- `metald_heartbeat_sent_total` - Heartbeat counts
- `metald_*_duration_seconds` - Performance metrics

### 3. System Health Dashboard (`system-health.json`)
- **Service Status**: Overall metald health and uptime
- **Resource Usage**: CPU, memory, and Go runtime metrics
- **HTTP Performance**: Request rates and response times
- **Go Runtime**: GC, goroutines, and memory statistics
- **Log Analysis**: Error and warning log trends

**Key Metrics:**
- `up{job="metald"}` - Service availability
- `process_*` - System resource usage
- `go_*` - Go runtime statistics
- `http_*` - HTTP server performance

## Quick Start

### 1. Start the LGTM Stack
```bash
# Start Grafana LGTM stack (Loki, Grafana, Tempo, Mimir)
make o11y

# Verify Grafana is running
curl http://localhost:3000/api/health
```

### 2. Import Dashboards
```bash
# Automated import
./scripts/import-dashboards.sh

# Manual import via Grafana UI
# 1. Open http://localhost:3000 (admin/admin)
# 2. Go to Dashboards > Import
# 3. Upload each .json file
```

### 3. Start Metald with Telemetry
```bash
# Enable OpenTelemetry and start metald
UNKEY_METALD_OTEL_ENABLED=true \
UNKEY_METALD_OTEL_PROMETHEUS_ENABLED=true \
./build/metald
```

### 4. Access Dashboards
- **Grafana UI**: http://localhost:3000 (admin/admin)
- **VM Operations**: http://localhost:3000/d/metald-vm-ops
- **Billing & Metrics**: http://localhost:3000/d/metald-billing
- **System Health**: http://localhost:3000/d/metald-system-health

## Configuration

### Environment Variables
```bash
# Required for telemetry
export UNKEY_METALD_OTEL_ENABLED=true
export UNKEY_METALD_OTEL_PROMETHEUS_ENABLED=true

# Optional configuration
export UNKEY_METALD_OTEL_SAMPLING_RATE=1.0
export UNKEY_METALD_OTEL_ENDPOINT=localhost:4318
export UNKEY_METALD_OTEL_PROMETHEUS_PORT=9464
```

### Prometheus Configuration
The LGTM stack automatically scrapes metrics from metald. For custom Prometheus setup:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'metald'
    static_configs:
      - targets: ['localhost:9464']
    scrape_interval: 10s
    metrics_path: /metrics
```

## Dashboard Features

### Variables and Templating
- **Backend Filter**: Filter by Firecracker/Cloud Hypervisor
- **VM ID Filter**: Focus on specific VMs
- **Customer ID Filter**: Billing metrics by customer

### Alerting Ready
All dashboards include threshold configurations suitable for Grafana alerting:
- VM operation failure rates > 5%
- High memory usage > 500MB
- Billing batch failures
- Service downtime

### Time Range Controls
- Default: Last 15 minutes with 5-second refresh
- Customizable time ranges
- Real-time monitoring support

## Troubleshooting

### Common Issues

**Dashboard shows "No data":**
1. Verify metald is running with telemetry enabled
2. Check Prometheus datasource configuration
3. Ensure metrics endpoint is accessible: `curl http://localhost:9464/metrics`

**Import script fails:**
1. Check Grafana is running: `curl http://localhost:3000/api/health`
2. Verify jq is installed: `sudo apt install jq` (Ubuntu/Debian)
3. Check Grafana credentials (default: admin/admin)

**Missing metrics:**
1. Confirm OpenTelemetry is enabled in metald config
2. Check for backend-specific metrics (Firecracker vs Cloud Hypervisor)
3. Verify billing service integration for billing metrics

### Manual Verification
```bash
# Check service health
curl http://localhost:8080/_/health

# View raw metrics
curl http://localhost:9464/metrics | grep unkey_metald

# Test VM operations
curl -X POST http://localhost:8080/vmprovisioner.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -d '{"config":{"cpu":{"vcpu_count":1},"memory":{"size_bytes":134217728}}}'
```

## Customization

### Adding Custom Panels
1. Use Grafana UI to create new panels
2. Export dashboard JSON
3. Save to this directory
4. Update import script if needed

### Metric Queries
All dashboards use standard PromQL queries. Common patterns:
- Rate calculations: `rate(metric_total[5m])`
- Success rates: `rate(success_total[5m]) / rate(requests_total[5m]) * 100`
- Percentiles: `histogram_quantile(0.95, rate(metric_bucket[5m]))`

### Integration with LGTM Stack
The dashboards are designed to work seamlessly with the included LGTM stack:
- **Loki**: Log aggregation and querying
- **Grafana**: Visualization and dashboards
- **Tempo**: Distributed tracing
- **Mimir**: Long-term metrics storage

For production deployments, consider configuring persistent storage and retention policies.