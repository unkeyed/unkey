# Billaged Configuration Guide

Complete guide for configuring the billaged service for development and production environments.

## Configuration Overview

Billaged uses environment variables for configuration, making it easy to deploy in containerized environments. All configuration options have sensible defaults for development, with production-specific recommendations noted.

## Environment Variables

### Server Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `BILLAGED_PORT` | `8081` | Port for ConnectRPC API | Use `8081` to avoid conflicts with metald |
| `BILLAGED_ADDRESS` | `0.0.0.0` | Bind address | Use specific IP in production |

### Aggregation Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `BILLAGED_AGGREGATION_INTERVAL` | `60s` | Usage summary aggregation interval | Keep at `60s` for standard billing |

### OpenTelemetry Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `BILLAGED_OTEL_ENABLED` | `false` | Enable OpenTelemetry | Set to `true` in production |
| `BILLAGED_OTEL_SERVICE_NAME` | `billaged` | Service name for traces/metrics | Include environment suffix |
| `BILLAGED_OTEL_SERVICE_VERSION` | `0.0.1` | Service version | Match deployment version |
| `BILLAGED_OTEL_SAMPLING_RATE` | `1.0` | Trace sampling rate (0.0-1.0) | Use `0.1` for high volume |
| `BILLAGED_OTEL_ENDPOINT` | `localhost:4318` | OTLP endpoint | Point to collector |
| `BILLAGED_OTEL_PROMETHEUS_ENABLED` | `true` | Enable Prometheus metrics | Keep enabled |
| `BILLAGED_OTEL_PROMETHEUS_PORT` | `9465` | Prometheus metrics port | Ensure unique port |
| `BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED` | `false` | Enable high-cardinality labels | **Keep false in production** |

## Configuration Examples

### Development Configuration

```bash
# Basic development setup
export BILLAGED_PORT=8081
export BILLAGED_ADDRESS=0.0.0.0
export BILLAGED_AGGREGATION_INTERVAL=60s

# Optional: Enable local observability
export BILLAGED_OTEL_ENABLED=true
export BILLAGED_OTEL_ENDPOINT=localhost:4318
export BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=true  # OK for dev

./billaged
```

### Production Configuration

```bash
# Production setup with observability
export BILLAGED_PORT=8081
export BILLAGED_ADDRESS=10.0.1.50
export BILLAGED_AGGREGATION_INTERVAL=60s

# OpenTelemetry for production monitoring
export BILLAGED_OTEL_ENABLED=true
export BILLAGED_OTEL_SERVICE_NAME=billaged-prod
export BILLAGED_OTEL_SERVICE_VERSION=1.2.3
export BILLAGED_OTEL_SAMPLING_RATE=0.1
export BILLAGED_OTEL_ENDPOINT=otel-collector.monitoring.svc.cluster.local:4318
export BILLAGED_OTEL_PROMETHEUS_ENABLED=true
export BILLAGED_OTEL_PROMETHEUS_PORT=9465
export BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false  # Critical for production

./billaged
```

### Docker Configuration

```dockerfile
FROM golang:1.21-alpine AS builder
# ... build steps ...

FROM alpine:3.19
COPY --from=builder /app/billaged /usr/local/bin/

# Set production defaults
ENV BILLAGED_PORT=8081 \
    BILLAGED_ADDRESS=0.0.0.0 \
    BILLAGED_AGGREGATION_INTERVAL=60s \
    BILLAGED_OTEL_ENABLED=true \
    BILLAGED_OTEL_SERVICE_NAME=billaged \
    BILLAGED_OTEL_PROMETHEUS_ENABLED=true \
    BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false

EXPOSE 8081 9465
CMD ["billaged"]
```

### Kubernetes Configuration

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: billaged-config
data:
  BILLAGED_AGGREGATION_INTERVAL: "60s"
  BILLAGED_OTEL_ENABLED: "true"
  BILLAGED_OTEL_SERVICE_NAME: "billaged-prod"
  BILLAGED_OTEL_ENDPOINT: "otel-collector.monitoring.svc.cluster.local:4318"
  BILLAGED_OTEL_PROMETHEUS_ENABLED: "true"
  BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED: "false"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: billaged
spec:
  replicas: 3
  selector:
    matchLabels:
      app: billaged
  template:
    metadata:
      labels:
        app: billaged
    spec:
      containers:
      - name: billaged
        image: unkeyed/billaged:latest
        ports:
        - containerPort: 8081
          name: grpc
        - containerPort: 9465
          name: metrics
        envFrom:
        - configMapRef:
            name: billaged-config
        env:
        - name: BILLAGED_OTEL_SERVICE_VERSION
          value: "1.2.3"
        - name: BILLAGED_OTEL_SAMPLING_RATE
          value: "0.1"
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "2000m"
```

## Advanced Configuration

### High Cardinality Labels

The `BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED` setting controls whether labels like `vm_id` and `customer_id` are included in metrics. 

**Development**: Enable for detailed debugging
```bash
export BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=true
```

**Production**: Disable to prevent cardinality explosion
```bash
export BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false
```

When disabled, metrics are aggregated at service level:
- ✅ `billaged_metrics_processed_total{service="billaged"}`
- ❌ `billaged_metrics_processed_total{service="billaged",vm_id="...",customer_id="..."}`

### Sampling Rate Optimization

Choose sampling rates based on traffic volume:

| Traffic Volume | Recommended Rate | Example |
|----------------|------------------|---------|
| < 1K req/s | 1.0 (100%) | Development |
| 1K-10K req/s | 0.1 (10%) | Small production |
| 10K-100K req/s | 0.01 (1%) | Medium production |
| > 100K req/s | 0.001 (0.1%) | Large production |

### Multi-Region Configuration

For multi-region deployments, use region-specific service names:

```bash
# US East
export BILLAGED_OTEL_SERVICE_NAME=billaged-us-east-1

# EU West
export BILLAGED_OTEL_SERVICE_NAME=billaged-eu-west-1

# APAC
export BILLAGED_OTEL_SERVICE_NAME=billaged-ap-southeast-1
```

## Security Configuration

### TLS Configuration (Future)

While not yet implemented, TLS support is planned:

```bash
# Future TLS configuration
export BILLAGED_TLS_ENABLED=true
export BILLAGED_TLS_CERT_FILE=/path/to/cert.pem
export BILLAGED_TLS_KEY_FILE=/path/to/key.pem
export BILLAGED_TLS_CA_FILE=/path/to/ca.pem
```

### Network Security

1. **Bind Address**: In production, bind to specific interfaces:
   ```bash
   export BILLAGED_ADDRESS=10.0.1.50  # Internal network only
   ```

2. **Firewall Rules**: Restrict access to billaged ports:
   - Port 8081: Only from metald instances
   - Port 9465: Only from monitoring systems

## Performance Tuning

### Resource Allocation

Recommended resources based on load:

| Load Profile | Memory | CPU | Notes |
|--------------|--------|-----|-------|
| Development | 128Mi | 100m | Single instance |
| Small (< 1K VMs) | 256Mi | 500m | 2 replicas |
| Medium (1K-10K VMs) | 1Gi | 2 cores | 3-5 replicas |
| Large (> 10K VMs) | 4Gi | 4 cores | 5-10 replicas |

### Aggregation Interval

The aggregation interval affects memory usage and accuracy:

- **60s** (default): Best balance of accuracy and efficiency
- **30s**: More granular data, 2x memory usage
- **120s**: Less memory, potential accuracy loss

## Monitoring Configuration

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'billaged'
    static_configs:
      - targets: ['billaged:9465']
    scrape_interval: 30s
    metrics_path: /metrics
```

### Key Metrics to Monitor

Configure alerts for these critical metrics:

```yaml
groups:
- name: billaged
  rules:
  - alert: BillagedHighErrorRate
    expr: rate(billaged_errors_total[5m]) > 0.05
    for: 10m
    annotations:
      summary: "High error rate in billaged"
      
  - alert: BillagedHighMemoryUsage
    expr: process_resident_memory_bytes > 3e9
    for: 10m
    annotations:
      summary: "Billaged memory usage exceeds 3GB"
      
  - alert: BillagedSlowProcessing
    expr: histogram_quantile(0.95, billaged_processing_duration_seconds) > 0.1
    for: 10m
    annotations:
      summary: "95th percentile processing time exceeds 100ms"
```

## Validation

After configuration, validate the setup:

```bash
# Check service is running
curl -f http://localhost:8081/health || echo "Health check failed"

# Verify metrics endpoint
curl -s http://localhost:9465/metrics | grep billaged_ || echo "Metrics not found"

# Test API endpoint
curl -X POST http://localhost:8081/billing.v1.BillingService/SendHeartbeat \
  -H "Content-Type: application/json" \
  -d '{"instance_id": "test", "active_vms": []}'
```

## Troubleshooting Configuration

### Common Issues

1. **Port Already in Use**
   ```bash
   Error: listen tcp :8081: bind: address already in use
   Solution: Change BILLAGED_PORT to unused port
   ```

2. **Invalid Sampling Rate**
   ```
   Error: invalid BILLAGED_OTEL_SAMPLING_RATE, using default 1.0
   Solution: Ensure value is between 0.0 and 1.0
   ```

3. **OTLP Connection Failed**
   ```
   Error: failed to export traces: connection refused
   Solution: Verify BILLAGED_OTEL_ENDPOINT is correct
   ```

### Debug Mode

Enable debug logging by setting log level:

```bash
export LOG_LEVEL=debug
./billaged
```

This configuration guide ensures billaged is properly configured for your specific deployment needs, from development to large-scale production.