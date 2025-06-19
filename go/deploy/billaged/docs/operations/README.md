# Billaged Operations Guide

This guide covers production deployment, monitoring, troubleshooting, and operational best practices for the billaged service.

## Table of Contents

- [Deployment](#deployment)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Health Checks](#health-checks)
- [Troubleshooting](#troubleshooting)
- [Performance Tuning](#performance-tuning)
- [Security Operations](#security-operations)

## Deployment

### System Requirements

- **OS**: Linux (any modern distribution)
- **CPU**: 2+ cores recommended
- **Memory**: 2GB minimum, 4GB+ recommended
- **Network**: Low latency to metald instances
- **Disk**: Minimal (logs only)

### Installation Methods

#### 1. Systemd Service

**Installation**: [`Makefile:14-24`](../../Makefile:14-24)

```bash
# Build and install with systemd
cd billaged
make install

# Start the service
sudo systemctl start billaged
sudo systemctl enable billaged

# Check status
sudo systemctl status billaged
```

**Unit File**: [`contrib/systemd/billaged.service`](../../contrib/systemd/billaged.service)

#### 2. Docker Container

```bash
# Build Docker image
docker build -t billaged:latest .

# Run with environment variables
docker run -d \
  --name billaged \
  -p 8081:8081 \
  -p 9465:9465 \
  -e UNKEY_BILLAGED_TLS_MODE=disabled \
  -e UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s \
  billaged:latest
```

#### 3. Kubernetes Deployment

```yaml
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
        image: billaged:latest
        ports:
        - containerPort: 8081
          name: grpc
        - containerPort: 9465
          name: metrics
        env:
        - name: UNKEY_BILLAGED_TLS_MODE
          value: "spiffe"
        - name: UNKEY_BILLAGED_SPIFFE_SOCKET
          value: "/run/spire/sockets/agent.sock"
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
          readOnly: true
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
          type: DirectoryOrCreate
```

## Configuration

Billaged uses environment variables for configuration. All variables follow the `UNKEY_BILLAGED_*` naming convention.

### Configuration Reference

**Implementation**: [`internal/config/config.go:15-34`](../../internal/config/config.go:15-34)

#### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `UNKEY_BILLAGED_PORT` | `8081` | Server port |
| `UNKEY_BILLAGED_ADDRESS` | `0.0.0.0` | Bind address |
| `UNKEY_BILLAGED_AGGREGATION_INTERVAL` | `60s` | Usage summary interval |

#### OpenTelemetry Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `UNKEY_BILLAGED_ENABLE_OTEL` | `false` | Enable OpenTelemetry |
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
| `UNKEY_BILLAGED_SPIFFE_SOCKET` | `/var/lib/spire/agent/agent.sock` | SPIFFE workload API socket |

### Production Configuration Example

```bash
# /etc/systemd/system/billaged.service.d/override.conf
[Service]
Environment="UNKEY_BILLAGED_PORT=8081"
Environment="UNKEY_BILLAGED_ADDRESS=0.0.0.0"
Environment="UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s"
Environment="UNKEY_BILLAGED_ENABLE_OTEL=true"
Environment="UNKEY_BILLAGED_OTEL_ENDPOINT=otel-collector.monitoring:4318"
Environment="UNKEY_BILLAGED_OTEL_SAMPLING_RATE=0.1"
Environment="UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false"
Environment="UNKEY_BILLAGED_TLS_MODE=spiffe"
```

## Monitoring

### Prometheus Metrics

**Endpoint**: `http://localhost:9465/metrics`  
**Implementation**: [`internal/observability/metrics.go:35-55`](../../internal/observability/metrics.go:35-55)

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
- **Type**: Gauge
- **Labels**: None
- **Description**: Number of active VMs being tracked
- **Use Case**: Capacity planning and load monitoring

##### billaged_billing_errors_total
- **Type**: Counter
- **Labels**: `error_type`
- **Description**: Total number of billing processing errors
- **Use Case**: Error rate monitoring and alerting

### Sample Queries

```promql
# VMs per customer
sum by (customer_id) (
  increase(billaged_usage_records_processed_total[5m])
)

# Aggregation latency P99
histogram_quantile(0.99, 
  rate(billaged_aggregation_duration_seconds_bucket[5m])
)

# Error rate
rate(billaged_billing_errors_total[5m])

# Memory usage estimation (1KB per VM)
billaged_active_vms * 1024
```

### Grafana Dashboard

**Location**: [`contrib/grafana-dashboards/billaged-overview.json`](../../contrib/grafana-dashboards/billaged-overview.json)

Dashboard panels:
- Active VMs by customer
- Processing rate and latency
- Error rates by type
- Resource usage trends
- Aggregation interval distribution

### Logging

**Format**: Structured JSON ([`cmd/billaged/main.go:92-104`](../../cmd/billaged/main.go:92-104))

```json
{
  "time": "2024-01-01T12:00:00Z",
  "level": "INFO",
  "msg": "usage summary generated",
  "vm_id": "vm-123",
  "customer_id": "customer-456",
  "cpu_seconds": 45.2,
  "memory_gb_seconds": 30.5,
  "resource_score": 78.5
}
```

#### Log Levels

- **DEBUG**: Detailed processing information
- **INFO**: Normal operations, summaries
- **WARN**: Recoverable issues
- **ERROR**: Processing failures

## Health Checks

### Health Endpoint

**URL**: `http://localhost:8081/health`  
**Implementation**: [`pkg/health`](../../cmd/billaged/main.go:284)

#### Response Format

```json
{
  "status": "healthy",
  "service": "billaged",
  "version": "0.1.0",
  "uptime": "72h15m30s",
  "checks": {
    "memory": "ok",
    "goroutines": 42
  }
}
```

### Readiness vs Liveness

**Readiness**: Service ready to accept traffic
- Check: `/health` returns 200
- Indicates: Service initialized

**Liveness**: Service is running
- Check: `/health` returns any response
- Indicates: Process not deadlocked

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

## Troubleshooting

### Common Issues

#### 1. High Memory Usage

**Symptoms**: Growing memory consumption

**Diagnosis**:
```bash
# Check active VMs
curl http://localhost:8081/stats

# Monitor Prometheus metrics
curl http://localhost:9465/metrics | grep billaged_active_vms
```

**Solutions**:
- Reduce aggregation interval
- Scale horizontally
- Check for VM lifecycle event delivery

#### 2. Missing Metrics

**Symptoms**: No usage summaries generated

**Diagnosis**:
```bash
# Check RPC errors
journalctl -u billaged | grep "rpc error"

# Verify metald connectivity
nc -zv metald-host 8080
```

**Solutions**:
- Verify network connectivity
- Check TLS/mTLS configuration
- Confirm metald is sending metrics

#### 3. SPIFFE Authentication Failures

**Symptoms**: Connection refused errors

**Diagnosis**:
```bash
# Check SPIFFE agent
spire-agent api fetch x509

# Verify socket permissions
ls -la /var/lib/spire/agent/agent.sock
```

**Solutions**:
- Ensure SPIRE agent is running
- Check socket path configuration
- Verify workload registration

### Debug Commands

```bash
# Check active VMs
curl http://localhost:8081/stats | jq

# Monitor metrics ingestion
curl -s http://localhost:9465/metrics | grep billaged_usage_records_processed_total

# Watch aggregation performance
watch -n 5 'curl -s http://localhost:9465/metrics | grep billaged_aggregation_duration'
```

### Debug Mode

Enable detailed logging:

```bash
# Set log level to debug
export UNKEY_BILLAGED_LOG_LEVEL=debug

# Run with verbose output
./billaged 2>&1 | jq '.'
```

## Performance Tuning

### Memory Optimization

1. **Aggregation Interval**: Shorter intervals = less memory
   ```bash
   export UNKEY_BILLAGED_AGGREGATION_INTERVAL=30s
   ```

2. **Batch Size**: Control in metald configuration
   - Smaller batches = more frequent processing
   - Larger batches = better throughput

### CPU Optimization

1. **GOMAXPROCS**: Set based on CPU cores
   ```bash
   export GOMAXPROCS=4
   ```

2. **Concurrency**: Tune based on workload
   - More VMs = higher concurrency needed
   - Monitor goroutine count

### Network Optimization

1. **HTTP/2 Settings**: Already optimized by default
2. **Compression**: Enabled for all RPCs
3. **Keep-alive**: Configured for long-lived connections

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

## Security Operations

### TLS/mTLS Operations

#### Certificate Rotation (File Mode)

```bash
# Update certificates
cp new-cert.pem /etc/billaged/cert.pem
cp new-key.pem /etc/billaged/key.pem

# Reload service
systemctl reload billaged
```

#### SPIFFE Operations

```bash
# List workload entries
spire-server entry list

# Update workload registration
spire-server entry update \
  -entryID <id> \
  -selector unix:uid:1000 \
  -spiffeID spiffe://example.org/billaged
```

### Security Checklist

- [ ] TLS enabled for production
- [ ] SPIFFE workload registered
- [ ] Network policies configured
- [ ] Prometheus endpoint secured
- [ ] Logs sanitized (no sensitive data)
- [ ] Regular security updates

### Incident Response

1. **Metric Data Leak**: No persistent storage, restart clears all
2. **Authentication Bypass**: Check TLS configuration immediately
3. **DoS Attack**: Implement rate limiting at proxy level

## Backup and Recovery

### State Recovery

Billaged is stateless, but consider:

1. **Configuration Backup**: Version control environment files
2. **Metric Continuity**: Metald will retry on recovery
3. **Gap Detection**: Automatic via NotifyPossibleGap

### Disaster Recovery

1. **Multi-region**: Deploy in multiple regions
2. **Load Balancing**: Use anycast or GeoDNS
3. **Failover**: Automatic with health checks

## Capacity Planning

### Sizing Guidelines

| VMs | CPU | Memory | Network |
|-----|-----|--------|---------|
| 1K | 1 core | 512MB | 10 Mbps |
| 10K | 2 cores | 2GB | 100 Mbps |
| 100K | 4 cores | 8GB | 1 Gbps |

### Scaling Triggers

Monitor these metrics for scaling decisions:
- `billaged_active_vms` > 10,000
- `billaged_aggregation_duration_seconds` > 5s
- CPU usage > 80%
- Memory usage > 80%

### Horizontal Scaling

1. **Stateless Design**: Each instance independent
2. **Load Distribution**: Round-robin or least-connections
3. **Session Affinity**: Not required
4. **Shared State**: None (design advantage)

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
          
      - alert: BillagedHighMemory
        expr: billaged_active_vms > 50000
        for: 30m
        annotations:
          summary: "High number of active VMs"
```