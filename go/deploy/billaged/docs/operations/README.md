# Billaged Operations Guide

## Configuration

Billaged uses environment variables for configuration. All variables follow the `UNKEY_BILLAGED_*` naming convention.

### Configuration Reference

**Implementation**: [`internal/config/config.go`](../../internal/config/config.go:91-207)

#### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `UNKEY_BILLAGED_PORT` | `8081` | Server port |
| `UNKEY_BILLAGED_ADDRESS` | `0.0.0.0` | Bind address |
| `UNKEY_BILLAGED_AGGREGATION_INTERVAL` | `60s` | Usage summary interval |

#### OpenTelemetry Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `UNKEY_BILLAGED_OTEL_ENABLED` | `false` | Enable OpenTelemetry |
| `UNKEY_BILLAGED_OTEL_SERVICE_NAME` | `billaged` | Service name for traces |
| `UNKEY_BILLAGED_OTEL_SERVICE_VERSION` | `0.1.0` | Service version |
| `UNKEY_BILLAGED_OTEL_SAMPLING_RATE` | `1.0` | Trace sampling rate (0.0-1.0) |
| `UNKEY_BILLAGED_OTEL_ENDPOINT` | `localhost:4318` | OTLP HTTP endpoint |
| `UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED` | `true` | Enable Prometheus metrics |
| `UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT` | `9465` | Prometheus metrics port |
| `UNKEY_BILLAGED_OTEL_PROMETHEUS_INTERFACE` | `127.0.0.1` | Metrics bind interface |
| `UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED` | `false` | Enable high-cardinality labels |

#### TLS Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `UNKEY_BILLAGED_TLS_MODE` | `spiffe` | TLS mode: `disabled`, `file`, `spiffe` |
| `UNKEY_BILLAGED_TLS_CERT_FILE` | - | Certificate file path (file mode) |
| `UNKEY_BILLAGED_TLS_KEY_FILE` | - | Private key file path (file mode) |
| `UNKEY_BILLAGED_TLS_CA_FILE` | - | CA bundle file path (file mode) |
| `UNKEY_BILLAGED_SPIFFE_SOCKET` | `/run/spire/sockets/agent.sock` | SPIFFE workload API socket |

### Production Configuration Example

```bash
# /etc/systemd/system/billaged.service.d/override.conf
[Service]
Environment="UNKEY_BILLAGED_PORT=8081"
Environment="UNKEY_BILLAGED_ADDRESS=0.0.0.0"
Environment="UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s"
Environment="UNKEY_BILLAGED_OTEL_ENABLED=true"
Environment="UNKEY_BILLAGED_OTEL_ENDPOINT=otel-collector.monitoring:4318"
Environment="UNKEY_BILLAGED_OTEL_SAMPLING_RATE=0.1"
Environment="UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false"
Environment="UNKEY_BILLAGED_TLS_MODE=spiffe"
```

## Metrics

### Prometheus Metrics

**Implementation**: [`internal/observability/metrics.go`](../../internal/observability/metrics.go:14-119)

#### Core Metrics

##### billaged_usage_records_processed_total
- **Type**: Counter
- **Labels**: `vm_id`, `customer_id` (when high cardinality enabled)
- **Description**: Total number of usage records processed
- **Use Case**: Monitor ingestion rate and identify per-VM issues

##### billaged_aggregation_duration_seconds
- **Type**: Histogram
- **Labels**: None
- **Description**: Time spent aggregating usage metrics
- **Use Case**: Monitor aggregation performance

##### billaged_active_vms
- **Type**: UpDownCounter
- **Labels**: None
- **Description**: Number of active VMs being tracked
- **Use Case**: Capacity planning and load monitoring

##### billaged_billing_errors_total
- **Type**: Counter
- **Labels**: `error_type`
- **Description**: Total number of billing processing errors
- **Use Case**: Error rate monitoring and alerting

### Grafana Dashboard

Example queries for monitoring:

```promql
# Ingestion rate (per second)
rate(billaged_usage_records_processed_total[5m])

# Active VMs over time
billaged_active_vms

# Aggregation performance (p95)
histogram_quantile(0.95, rate(billaged_aggregation_duration_seconds_bucket[5m]))

# Error rate
rate(billaged_billing_errors_total[5m])
```

### High Cardinality Considerations

When `UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=true`:
- Metrics include `vm_id` and `customer_id` labels
- Significantly increases metric cardinality
- Use only in development or with cardinality limits

Production recommendation: Keep disabled and use logs for detailed tracking.

## Logging

### Log Format

Structured JSON logging with slog:

```json
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "received metrics batch",
  "vm_id": "vm-123",
  "customer_id": "customer-456",
  "metrics_count": 6,
  "component": "billing_service"
}
```

### Log Levels

Configure via slog handler options in [`cmd/billaged/main.go:87-90`](../../cmd/billaged/main.go:87-90)

- **INFO**: Normal operations, lifecycle events
- **WARN**: Recoverable issues, gaps detected
- **ERROR**: Processing failures, invalid data
- **DEBUG**: Detailed metric processing (verbose)

### Key Log Messages

#### Startup
```json
{
  "msg": "starting billaged service",
  "version": "0.1.0",
  "go_version": "go1.24.3",
  "port": "8081",
  "aggregation_interval": "60s"
}
```

#### Usage Summary
```json
{
  "msg": "=== BILLAGED USAGE SUMMARY ===",
  "vm_id": "vm-123",
  "customer_id": "customer-456",
  "cpu_time_used_seconds": 3600.5,
  "avg_memory_usage_mb": 1024,
  "total_disk_io_mb": 500,
  "resource_score": 3625.15
}
```

#### Errors
```json
{
  "level": "ERROR",
  "msg": "rpc error",
  "procedure": "/billing.v1.BillingService/SendMetricsBatch",
  "duration": "15ms",
  "error": "invalid metrics batch"
}
```

## Health Checks

### /health Endpoint

Provided by [`pkg/health`](../../cmd/billaged/main.go:284)

```bash
curl http://localhost:8081/health
```

Response:
```json
{
  "status": "healthy",
  "service": "billaged",
  "version": "0.1.0",
  "uptime": "1h30m45s"
}
```

### Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 10
  periodSeconds: 30

readinessProbe:
  httpGet:
    path: /health
    port: 8081
  initialDelaySeconds: 5
  periodSeconds: 10
```

## Debugging

### Enable Debug Logging

```go
// Temporary debug logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
```

### Common Issues

#### 1. No Metrics Received
- Check metald connectivity
- Verify TLS/SPIFFE configuration
- Check network policies

#### 2. High Memory Usage
- Monitor active VM count
- Check for VM lifecycle issues
- Verify aggregation interval

#### 3. Missing Usage Summaries
- Check aggregation callback
- Verify timer goroutine
- Look for panic recovery logs

### Debug Commands

```bash
# Check active VMs
curl http://localhost:8081/stats | jq

# Monitor metrics ingestion
curl -s http://localhost:9465/metrics | grep billaged_usage_records_processed_total

# Watch aggregation performance
watch -n 5 'curl -s http://localhost:9465/metrics | grep billaged_aggregation_duration'
```

## Performance Tuning

### Aggregation Interval

Adjust `UNKEY_BILLAGED_AGGREGATION_INTERVAL` based on:
- Number of VMs
- Billing granularity requirements
- System resources

Recommendations:
- Development: `30s` for faster feedback
- Production: `60s` for balance
- High-scale: `300s` to reduce overhead

### Batch Size

Configure metald to send appropriate batch sizes:
- Small batches: Lower latency, higher overhead
- Large batches: Better throughput, higher memory

### Resource Limits

Systemd resource limits:
```ini
[Service]
# Memory limit
MemoryLimit=2G
# CPU quota (200% = 2 cores)
CPUQuota=200%
# Restart on failure
Restart=on-failure
RestartSec=5s
```

## Monitoring Alerts

### Prometheus Alert Rules

```yaml
groups:
  - name: billaged
    rules:
      - alert: BillagedDown
        expr: up{job="billaged"} == 0
        for: 5m
        annotations:
          summary: "Billaged service is down"
          
      - alert: BillagedHighErrorRate
        expr: rate(billaged_billing_errors_total[5m]) > 0.1
        for: 10m
        annotations:
          summary: "High billing error rate"
          
      - alert: BillagedSlowAggregation
        expr: histogram_quantile(0.95, rate(billaged_aggregation_duration_seconds_bucket[5m])) > 5
        for: 15m
        annotations:
          summary: "Slow aggregation performance"
```

## Backup and Recovery

Billaged is stateless and doesn't require backup. However:

1. **Metric Data**: Ensure Prometheus retention
2. **Configuration**: Version control systemd overrides
3. **Logs**: Configure log rotation and archival

## Security Operations

### TLS Certificate Rotation

For file-based TLS:
```bash
# Reload systemd after cert update
systemctl reload billaged
```

For SPIFFE:
- Automatic rotation via workload API
- Monitor SPIFFE agent health

### Access Control

1. **Network Policies**: Restrict ingress to metald instances
2. **Firewall Rules**: Limit Prometheus endpoint access
3. **Service Mesh**: Use Istio/Linkerd policies

## Capacity Planning

### Resource Requirements

Per 1000 active VMs:
- **Memory**: ~100MB
- **CPU**: ~0.5 cores
- **Network**: ~10 Mbps

### Scaling Strategies

1. **Vertical**: Increase resources for single instance
2. **Horizontal**: Multiple instances with LB
3. **Sharding**: Partition by customer/VM range

## Troubleshooting Playbook

### Service Won't Start

1. Check configuration validation
2. Verify TLS/SPIFFE setup
3. Check port availability
4. Review systemd logs: `journalctl -u billaged -f`

### Metrics Not Updating

1. Verify metald connectivity
2. Check for errors in logs
3. Confirm aggregation timer running
4. Test with manual RPC call

### High Resource Usage

1. Check active VM count
2. Monitor metric batch sizes
3. Profile with pprof
4. Review aggregation interval