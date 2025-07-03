# Billaged Operations Manual

This guide covers production deployment, monitoring, and operational procedures for the billaged service.

## Installation & Deployment

### System Requirements

#### Hardware Requirements
- **CPU**: 2+ cores for high-throughput metric processing
- **Memory**: 4GB+ RAM for large-scale VM aggregation (1MB per 1000 active VMs)
- **Storage**: 10GB+ for logs, configuration, and temporary data
- **Network**: Low-latency connection to metald instances (<10ms preferred)

#### Software Dependencies
- **OS**: Linux with systemd support (RHEL 8+, Ubuntu 20.04+, Fedora 35+)
- **Go Runtime**: Go 1.24.4+ for building from source
- **SPIRE Agent**: Running SPIRE workload API for service authentication
- **systemd**: Service management and process supervision

### Building from Source

**Build Location**: [Makefile](../../Makefile)

```bash
# Clone repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/billaged

# Build binary
make build
# Creates: build/billaged

# Install with systemd integration
sudo make install
# Installs: binary, systemd unit, environment template
```

### Installation Locations

**Systemd Integration**: [contrib/systemd/](../../contrib/systemd/)

```
/usr/local/bin/billaged              # Service binary
/etc/systemd/system/billaged.service # Systemd unit file
/etc/billaged/environment            # Environment configuration
/var/log/billaged/                   # Log directory
/var/lib/billaged/                   # Runtime data directory
```

## Configuration Management

### Environment Variables

**Configuration Reference**: [config/config.go](../../internal/config/config.go)

#### Core Server Configuration

```bash
# Server binding configuration
export UNKEY_BILLAGED_PORT=8081                    # Service port
export UNKEY_BILLAGED_ADDRESS=0.0.0.0              # Bind address

# Billing aggregation settings  
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s     # Summary interval
```

#### Security Configuration

```bash
# TLS and authentication
export UNKEY_BILLAGED_TLS_MODE=spiffe              # TLS mode: spiffe, file, disabled
export UNKEY_BILLAGED_SPIFFE_SOCKET=/var/lib/spire/agent/agent.sock # SPIFFE socket

# File-based TLS (alternative to SPIFFE)
export UNKEY_BILLAGED_TLS_CERT_FILE=/etc/billaged/tls/cert.pem
export UNKEY_BILLAGED_TLS_KEY_FILE=/etc/billaged/tls/key.pem  
export UNKEY_BILLAGED_TLS_CA_FILE=/etc/billaged/tls/ca.pem
```

#### OpenTelemetry Configuration

```bash
# Observability settings
export UNKEY_BILLAGED_OTEL_ENABLED=true                        # Enable OTEL
export UNKEY_BILLAGED_OTEL_SERVICE_NAME=billaged               # Service name
export UNKEY_BILLAGED_OTEL_SERVICE_VERSION=0.1.0              # Version tag
export UNKEY_BILLAGED_OTEL_SAMPLING_RATE=1.0                  # Trace sampling
export UNKEY_BILLAGED_OTEL_ENDPOINT=localhost:4318            # OTLP endpoint

# Prometheus metrics
export UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED=true            # Enable Prometheus
export UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT=9465               # Metrics port  
export UNKEY_BILLAGED_OTEL_PROMETHEUS_INTERFACE=127.0.0.1     # Bind interface
export UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false     # Limit cardinality
```

### Production Configuration Template

**Environment File**: [contrib/systemd/billaged.env.example](../../contrib/systemd/billaged.env.example)

```bash
# Production environment template
UNKEY_BILLAGED_PORT=8081
UNKEY_BILLAGED_ADDRESS=0.0.0.0
UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s

# Security (SPIFFE required in production)
UNKEY_BILLAGED_TLS_MODE=spiffe
UNKEY_BILLAGED_SPIFFE_SOCKET=/var/lib/spire/agent/agent.sock

# Observability (full monitoring enabled)
UNKEY_BILLAGED_OTEL_ENABLED=true
UNKEY_BILLAGED_OTEL_SERVICE_NAME=billaged
UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED=true
UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT=9465
UNKEY_BILLAGED_OTEL_PROMETHEUS_INTERFACE=127.0.0.1
UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED=false
```

### Configuration Validation

**Validation Logic**: [config/config.go:183-198](../../internal/config/config.go#L183-198)

The service validates configuration on startup:

- **Sampling Rate**: Must be between 0.0 and 1.0 for tracing
- **OTLP Endpoint**: Required when OpenTelemetry is enabled
- **Service Name**: Required for proper observability tagging

## Service Management

### Systemd Integration

**Unit File**: [contrib/systemd/billaged.service](../../contrib/systemd/billaged.service)

#### Service Control

```bash
# Start service
sudo systemctl start billaged

# Enable automatic startup
sudo systemctl enable billaged

# Check service status
sudo systemctl status billaged

# View logs
sudo journalctl -u billaged -f

# Restart service
sudo systemctl restart billaged

# Stop service
sudo systemctl stop billaged
```

#### Service Status Monitoring

```bash
# Service health check
curl http://localhost:8081/health

# Service statistics
curl http://localhost:8081/stats

# Prometheus metrics (if enabled)
curl http://localhost:9465/metrics
```

### Process Management

#### Resource Limits

**Systemd Configuration**:
```ini
[Service]
LimitNOFILE=65536        # File descriptor limit
LimitNPROC=4096          # Process limit  
MemoryMax=4G             # Memory limit
CPUQuota=200%            # CPU limit (2 cores)
```

#### Health Monitoring

**Health Handler**: [pkg/health/health.go](../../../pkg/health/health.go)

```bash
# Health endpoint provides:
# - Service version and uptime
# - Basic connectivity status
# - Resource usage summary

curl http://localhost:8081/health
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime": "2h30m15s",
  "active_vms": 45
}
```

## Monitoring & Observability

### Key Metrics

**Metrics Implementation**: [observability/metrics.go](../../internal/observability/metrics.go)

#### Core Business Metrics

```prometheus
# Usage processing metrics
billaged_usage_records_processed_total{vm_id,customer_id}  # Total records processed
billaged_aggregation_duration_seconds                      # Processing latency histogram
billaged_active_vms                                        # Current VM tracking count
billaged_billing_errors_total{error_type}                 # Processing error count
```

#### System Performance Metrics

```prometheus
# HTTP request metrics (via OTEL)
http_request_duration_seconds{method,status}              # Request latency
http_requests_total{method,status}                        # Request count
http_request_size_bytes                                   # Request size distribution
http_response_size_bytes                                  # Response size distribution

# Go runtime metrics
go_memstats_alloc_bytes                                   # Memory allocation
go_memstats_gc_duration_seconds                           # GC pause time
go_goroutines                                            # Goroutine count
```

### Prometheus Configuration

#### Scrape Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'billaged'
    static_configs:
      - targets: ['billaged-host:9465']
    scrape_interval: 30s
    metrics_path: /metrics
    
    # Optional: SPIFFE mTLS authentication
    tls_config:
      cert_file: /etc/prometheus/spiffe-cert.pem
      key_file: /etc/prometheus/spiffe-key.pem
      ca_file: /etc/prometheus/spiffe-ca.pem
```

#### Alert Rules

```yaml
# billaged-alerts.yml
groups:
  - name: billaged
    rules:
      # High error rate
      - alert: BillagedHighErrorRate
        expr: rate(billaged_billing_errors_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "Billaged error rate is high"
          description: "Error rate {{ $value }} errors/sec for 2 minutes"

      # Processing latency
      - alert: BillagedHighLatency  
        expr: histogram_quantile(0.95, rate(billaged_aggregation_duration_seconds_bucket[5m])) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Billaged processing latency is high"
          description: "95th percentile latency is {{ $value }}s"

      # Memory usage
      - alert: BillagedHighMemoryUsage
        expr: process_resident_memory_bytes{job="billaged"} > 4e9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Billaged memory usage is high"
          description: "Memory usage {{ $value | humanizeBytes }} above 4GB"
```

### Grafana Dashboards

**Dashboard Location**: [contrib/grafana-dashboards/](../../contrib/grafana-dashboards/)

#### Key Dashboard Panels

1. **Usage Processing Rate**: `rate(billaged_usage_records_processed_total[5m])`
2. **Active VM Count**: `billaged_active_vms`
3. **Processing Latency**: `histogram_quantile(0.95, rate(billaged_aggregation_duration_seconds_bucket[5m]))`
4. **Error Rate**: `rate(billaged_billing_errors_total[5m])`
5. **Memory Usage**: `process_resident_memory_bytes`

## Log Management

### Log Configuration

**Logging Setup**: [cmd/billaged/main.go:89-92](../../cmd/billaged/main.go#L89-92)

#### Structured JSON Logging

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO", 
  "msg": "received metrics batch",
  "vm_id": "vm-firecracker-123",
  "customer_id": "customer-abc",
  "metrics_count": 5,
  "component": "billing_service"
}
```

#### Log Levels

- **DEBUG**: Detailed metric processing information and delta calculations
- **INFO**: Service lifecycle, usage summaries, and normal operations
- **WARN**: Gap detection, potential data issues, and configuration warnings  
- **ERROR**: Processing failures, authentication errors, and system issues

### Log Aggregation

#### Centralized Logging

```bash
# Systemd journal integration
sudo journalctl -u billaged -o json | tee /var/log/billaged/billaged.log

# Log rotation configuration
sudo logrotate -d /etc/logrotate.d/billaged

# Real-time log monitoring
sudo tail -f /var/log/billaged/billaged.log | jq .
```

#### Usage Summary Logging

**Summary Format**: [cmd/billaged/main.go:392-437](../../cmd/billaged/main.go#L392-437)

```json
{
  "time": "2024-01-15T10:31:00Z",
  "level": "INFO",
  "msg": "=== BILLAGED USAGE SUMMARY ===",
  "vm_id": "vm-firecracker-123", 
  "customer_id": "customer-abc",
  "period": "60s",
  "cpu_time_used_ms": 1500,
  "avg_memory_usage_mb": 512,
  "disk_read_mb": 1,
  "disk_write_mb": 0.5,
  "network_rx_mb": 0.002,
  "network_tx_mb": 0.001,
  "resource_score": 2.8505,
  "sample_count": 60
}
```

## Performance Tuning

### Memory Optimization

#### VM Tracking Optimization

**Data Structures**: [aggregator.go:68-79](../../internal/aggregator/aggregator.go#L68-79)

```bash
# Monitor memory usage per VM
# Typical: ~1KB per active VM
# High load: 1000 VMs = ~1MB memory usage

# Optimize for large deployments
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=30s  # More frequent cleanup
```

#### Garbage Collection Tuning

```bash
# Go GC tuning for high-throughput
export GOGC=100                    # Default GC percentage
export GOMEMLIMIT=3GiB            # Memory limit hint
export GODEBUG=gctrace=1          # GC debugging (development only)
```

### Aggregation Performance

#### Interval Tuning

**Configuration**: [config/config.go:152-153](../../internal/config/config.go#L152-153)

```bash
# High-precision billing (more CPU, more accurate)
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=30s

# Balanced performance (recommended)  
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=60s

# High-throughput (less CPU, less precision)
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=120s
```

#### Batch Size Optimization

- **Small Batches** (1-10 metrics): Lower latency, higher overhead
- **Medium Batches** (10-100 metrics): Optimal balance (recommended)
- **Large Batches** (100+ metrics): Higher latency, better throughput

### Network Performance

#### ConnectRPC Optimization

```bash
# Connection pooling and HTTP/2
# Handled automatically by ConnectRPC library

# Monitor connection metrics
curl http://localhost:9465/metrics | grep http_request
```

#### SPIFFE Performance

```bash
# Certificate caching configuration
export UNKEY_BILLAGED_SPIFFE_CERT_CACHE_TTL=5s

# Monitor certificate renewal
sudo journalctl -u spire-agent -f | grep billaged
```

## Security Operations

### SPIFFE/SPIRE Configuration

#### Workload Registration

**Registration Script**: [../../spire/scripts/register-services.sh](../../spire/scripts/register-services.sh)

```bash
# Register billaged workload with SPIRE
spire-server entry create \
  -spiffeID spiffe://unkey.dev/billaged \
  -parentID spiffe://unkey.dev/agent \
  -selector unix:uid:billaged \
  -selector unix:gid:billaged
```

#### Certificate Monitoring

```bash
# Check SPIFFE identity
spire-agent api fetch -socketPath /var/lib/spire/agent/agent.sock

# Monitor certificate expiration
sudo journalctl -u spire-agent -f | grep billaged
```

### Network Security

#### Firewall Configuration

```bash
# iptables rules for billaged
sudo iptables -A INPUT -p tcp --dport 8081 -s 10.0.0.0/8 -j ACCEPT     # Service port
sudo iptables -A INPUT -p tcp --dport 9465 -s 127.0.0.1 -j ACCEPT      # Metrics port (localhost only)
sudo iptables -A INPUT -p tcp --dport 8081 -j DROP                      # Block external access
```

#### TLS Configuration

```bash
# Verify TLS setup
openssl s_client -connect localhost:8081 -servername billaged

# Check certificate chain
openssl x509 -in /var/lib/spire/agent/svid.pem -text -noout
```

## Troubleshooting

### Common Issues

#### 1. SPIFFE Authentication Failures

**Symptoms**: `Unauthenticated` errors in logs, connection failures
**Diagnosis**:
```bash
# Check SPIRE agent status
sudo systemctl status spire-agent

# Verify workload registration
spire-server entry show -spiffeID spiffe://unkey.dev/billaged

# Check socket permissions
ls -la /var/lib/spire/agent/agent.sock
```

**Resolution**:
```bash
# Restart SPIRE agent
sudo systemctl restart spire-agent

# Re-register workload if needed
./register-services.sh
```

#### 2. High Memory Usage

**Symptoms**: Memory usage above 4GB, slow aggregation
**Diagnosis**:
```bash
# Check VM count
curl http://localhost:8081/stats

# Monitor memory growth
watch 'ps aux | grep billaged'
```

**Resolution**:
```bash
# Reduce aggregation interval for more frequent cleanup
export UNKEY_BILLAGED_AGGREGATION_INTERVAL=30s

# Restart service to clear memory
sudo systemctl restart billaged
```

#### 3. Processing Latency Issues

**Symptoms**: High `billaged_aggregation_duration_seconds` metrics
**Diagnosis**:
```bash
# Check current processing load
curl http://localhost:9465/metrics | grep billaged_aggregation_duration

# Monitor CPU usage
top -p $(pgrep billaged)
```

**Resolution**:
```bash
# Increase CPU allocation
# Edit /etc/systemd/system/billaged.service
CPUQuota=400%  # 4 cores

sudo systemctl daemon-reload
sudo systemctl restart billaged
```

#### 4. Missing Metrics

**Symptoms**: No usage summaries in logs, zero active VMs
**Diagnosis**:
```bash
# Check metald connectivity
curl http://metald:8080/health

# Verify service registration
spire-server entry show
```

**Resolution**:
```bash
# Check network connectivity
telnet metald 8080

# Verify SPIFFE configuration
sudo journalctl -u spire-agent -f
```

### Diagnostic Commands

#### Service Health

```bash
# Overall service health
curl http://localhost:8081/health

# Current statistics  
curl http://localhost:8081/stats

# Process information
ps aux | grep billaged
pmap $(pgrep billaged)
```

#### Performance Analysis

```bash
# Go profiling (if enabled)
go tool pprof http://localhost:8081/debug/pprof/profile

# Memory profile
go tool pprof http://localhost:8081/debug/pprof/heap

# Goroutine analysis
go tool pprof http://localhost:8081/debug/pprof/goroutine
```

### Emergency Procedures

#### Service Recovery

```bash
# Immediate restart
sudo systemctl restart billaged

# Clear all state (loses current aggregation data)
sudo systemctl stop billaged
sudo rm -rf /var/lib/billaged/*
sudo systemctl start billaged

# Rollback to previous version
sudo cp /usr/local/bin/billaged.backup /usr/local/bin/billaged
sudo systemctl restart billaged
```

#### Data Reconciliation

If usage data is lost or corrupted:

1. **Review Logs**: Check systemd journal for error patterns
2. **Gap Notifications**: Monitor for `NotifyPossibleGap` calls from metald
3. **Manual Reconciliation**: Use CLI tool to recreate missing lifecycle events
4. **Billing Adjustment**: Coordinate with billing systems for manual adjustments

## Capacity Planning

### Scaling Guidelines

#### Vertical Scaling

- **Memory**: 1MB per 1000 active VMs (linear scaling)
- **CPU**: 1 core per 5000 metrics/second processing
- **Network**: 100Mbps for 10000 concurrent VMs

#### Horizontal Scaling

Multiple billaged instances with client-side load balancing:

```bash
# Deploy multiple instances
billaged-1: port 8081
billaged-2: port 8082  
billaged-3: port 8083

# Client configuration
export UNKEY_BILLAGED_ENDPOINTS=billaged-1:8081,billaged-2:8082,billaged-3:8083
```

### Resource Monitoring

#### Key Capacity Metrics

- **Memory Growth Rate**: Should be linear with VM count
- **Processing Latency**: Should remain under 10ms per batch
- **Error Rate**: Should remain under 0.1% for production
- **Active VM Count**: Monitor for sudden changes indicating issues