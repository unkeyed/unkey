# Metald Operations Guide

This guide covers monitoring, deployment, and operational aspects of running metald in production.

## Table of Contents

- [Deployment](#deployment)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Health Checks](#health-checks)
- [Logging](#logging)
- [Debugging](#debugging)
- [Performance Tuning](#performance-tuning)
- [Troubleshooting](#troubleshooting)

## Deployment

### System Requirements

- **Operating System**: Linux with KVM support (kernel 4.14+)
- **CPU**: 4+ cores (8+ recommended for production)
- **Memory**: 8GB minimum (16GB+ for running many VMs)
- **Storage**: 
  - 100GB+ SSD for VM images and jailer chroots
  - `/opt/metald/data` for SQLite database
  - `/srv/jailer` for jailer chroot directories
- **Network**: CAP_NET_ADMIN capability for TAP device management

### Installation

#### From Source

```bash
# Clone repository
git clone https://github.com/unkeyed/unkey
cd go/deploy/metald

# Build binary
make build VERSION=0.2.0

# Install systemd service
sudo make install
```

#### Systemd Service

Service definition at [contrib/systemd/metald.service](../../../metald/contrib/systemd/metald.service):

```ini
[Unit]
Description=Metald VM Management Service
After=network.target

[Service]
Type=simple
User=metald
Group=metald
ExecStart=/usr/local/bin/metald
Restart=on-failure
RestartSec=10

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/metald /srv/jailer

# Capabilities
AmbientCapabilities=CAP_NET_ADMIN CAP_SYS_ADMIN
CapabilityBoundingSet=CAP_NET_ADMIN CAP_SYS_ADMIN

[Install]
WantedBy=multi-user.target
```

### Production Checklist

- [ ] Enable jailer with non-root UID/GID
- [ ] Configure TLS/mTLS for API endpoints
- [ ] Set up monitoring and alerting
- [ ] Configure resource quotas
- [ ] Enable structured logging
- [ ] Set up database backups
- [ ] Configure firewall rules
- [ ] Document runbooks

## Configuration

### Environment Variables

Complete configuration via environment variables with `UNKEY_METALD_` prefix:

#### Core Settings

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `UNKEY_METALD_SERVER_PORT` | string | `8080` | API server port |
| `UNKEY_METALD_SERVER_ADDRESS` | string | `0.0.0.0` | Bind address |
| `UNKEY_METALD_BACKEND_TYPE` | string | `firecracker` | Backend type |

#### Jailer Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `UNKEY_METALD_JAILER_UID` | uint32 | `1000` | Jailer process UID |
| `UNKEY_METALD_JAILER_GID` | uint32 | `1000` | Jailer process GID |
| `UNKEY_METALD_JAILER_CHROOT_BASE_DIR` | string | `/srv/jailer` | Base directory for chroots |

#### Service Integration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `UNKEY_METALD_BILLING_ENABLED` | bool | `false` | Enable billaged integration |
| `UNKEY_METALD_BILLING_ENDPOINT` | string | `http://localhost:8081` | Billaged endpoint |
| `UNKEY_METALD_ASSETMANAGER_ENABLED` | bool | `false` | Enable assetmanagerd |
| `UNKEY_METALD_ASSETMANAGER_ENDPOINT` | string | `http://localhost:8083` | AssetManagerd endpoint |

#### Network Configuration

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `UNKEY_METALD_NETWORK_ENABLE_IPV4` | bool | `true` | Enable IPv4 |
| `UNKEY_METALD_NETWORK_BRIDGE_IPV4` | string | `10.100.0.1/16` | Bridge IPv4 |
| `UNKEY_METALD_NETWORK_VM_SUBNET_IPV4` | string | `10.100.0.0/16` | VM subnet |
| `UNKEY_METALD_NETWORK_ENABLE_IPV6` | bool | `true` | Enable IPv6 |
| `UNKEY_METALD_NETWORK_BRIDGE_IPV6` | string | `fd00::1/64` | Bridge IPv6 |

#### Observability

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `UNKEY_METALD_OTEL_ENABLED` | bool | `true` | Enable OpenTelemetry |
| `UNKEY_METALD_OTEL_OTLP_ENDPOINT` | string | `localhost:4318` | OTLP endpoint |
| `UNKEY_METALD_OTEL_PROMETHEUS_PORT` | string | `9090` | Metrics port |
| `UNKEY_METALD_OTEL_HIGH_CARDINALITY_LABELS` | bool | `false` | Enable vm_id labels |

### Production Configuration Example

```bash
# /etc/metald/metald.env
UNKEY_METALD_SERVER_PORT=8080
UNKEY_METALD_BACKEND_TYPE=firecracker

# Security
UNKEY_METALD_JAILER_UID=2000
UNKEY_METALD_JAILER_GID=2000
UNKEY_METALD_TLS_MODE=spiffe
UNKEY_METALD_TLS_SPIFFE_SOCKET=/run/spire/sockets/agent.sock

# Integration
UNKEY_METALD_BILLING_ENABLED=true
UNKEY_METALD_BILLING_ENDPOINT=https://billaged.internal:8081
UNKEY_METALD_ASSETMANAGER_ENABLED=true
UNKEY_METALD_ASSETMANAGER_ENDPOINT=https://assetmanagerd.internal:8083

# Database
UNKEY_METALD_DATABASE_DATA_DIR=/opt/metald/data

# Monitoring
UNKEY_METALD_OTEL_ENABLED=true
UNKEY_METALD_OTEL_OTLP_ENDPOINT=otel-collector.monitoring:4318
UNKEY_METALD_OTEL_PROMETHEUS_ENABLED=true
UNKEY_METALD_OTEL_HIGH_CARDINALITY_LABELS=false
```

## Monitoring

### Prometheus Metrics

Metrics exposed at `http://localhost:9090/metrics`

#### VM Operation Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `vm_create_total` | Counter | VM creation attempts | `backend`, `status` |
| `vm_boot_total` | Counter | VM boot attempts | `backend`, `status` |
| `vm_shutdown_total` | Counter | VM shutdown attempts | `backend`, `status` |
| `vm_delete_total` | Counter | VM deletion attempts | `backend`, `status` |
| `vm_operation_duration_seconds` | Histogram | Operation latency | `operation`, `backend` |

#### Backend Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `firecracker_vm_create_total` | Counter | Firecracker creates | `status` |
| `firecracker_vm_error_total` | Counter | Backend errors | `operation`, `error` |
| `vm_active_count` | Gauge | Currently active VMs | `state` |

#### Billing Integration Metrics

| Metric | Type | Description | Labels |
|--------|------|-------------|--------|
| `billing_metrics_collected_total` | Counter | Metrics collected | `vm_id`* |
| `billing_metrics_sent_total` | Counter | Metrics sent | `status` |
| `billing_metrics_batch_size` | Histogram | Batch sizes | - |
| `billing_errors_total` | Counter | Billing errors | `error_type` |

*High cardinality label, disabled by default

### Grafana Dashboards

Import dashboards from [contrib/grafana-dashboards/](../../../metald/contrib/grafana-dashboards/):

1. **System Overview**: VM counts, operation rates, error rates
2. **Performance**: Operation latencies, queue depths
3. **Billing**: Metrics collection, transmission success
4. **Infrastructure**: CPU, memory, disk usage

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: metald
    rules:
      - alert: HighVMCreationFailureRate
        expr: rate(vm_create_total{status="failure"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High VM creation failure rate
          
      - alert: BillingIntegrationDown
        expr: rate(billing_errors_total[5m]) > 0
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: Billing integration failing
          
      - alert: HighOperationLatency
        expr: histogram_quantile(0.99, rate(vm_operation_duration_seconds_bucket[5m])) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: VM operations taking too long
```

## Health Checks

### Primary Health Endpoint

**GET /health**

Response:
```json
{
  "status": "healthy",
  "version": "0.2.0",
  "uptime_seconds": 3600,
  "checks": {
    "database": "ok",
    "backend": "ok",
    "assetmanager": "ok",
    "billing": "ok"
  }
}
```

Implementation: [handler.go:31-89](../../../metald/internal/health/handler.go#L31-L89)

### Readiness Check

**GET /ready**

Returns 200 when:
- Database is accessible
- Backend is initialized
- Dependencies are reachable

### Liveness Check

**GET /alive**

Simple liveness probe, always returns 200 if service is running.

## Logging

### Log Format

Structured JSON logging to stdout:

```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "creating vm",
  "service": "vm",
  "method": "CreateVm",
  "vm_id": "vm-01HQKP3X5V2Q8Z9R1N4M7BHCFD",
  "customer_id": "cust-123",
  "trace_id": "a1b2c3d4e5f6",
  "span_id": "1234567890"
}
```

### Log Levels

Configure via code (no runtime changes):
- `DEBUG`: Detailed operational info
- `INFO`: Normal operations (default)
- `WARN`: Potential issues
- `ERROR`: Failures requiring attention

### Key Log Fields

- `vm_id`: Virtual machine identifier
- `customer_id`: Customer identifier
- `operation`: Current operation
- `backend`: Backend type
- `error`: Error details
- `duration`: Operation duration
- `trace_id`: OpenTelemetry trace ID

## Debugging

### Debug Endpoints

Available in development mode only:

#### GET /debug/vms
Lists all VMs with internal state:
```json
[
  {
    "vm_id": "vm-01HQKP3X5V2Q8Z9R1N4M7BHCFD",
    "state": "RUNNING",
    "backend_state": "active",
    "created_at": "2024-01-15T10:00:00Z",
    "network": {
      "tap": "tap-vm-01HQKP3X",
      "namespace": "ns-vm-01HQKP3X"
    }
  }
]
```

#### GET /debug/vm/{id}
Detailed VM information including:
- Full configuration
- Resource allocation
- Network details
- Process information

### Tracing

Enable distributed tracing to track requests across services:

1. VM creation request flow
2. Asset preparation calls
3. Billing notifications
4. Backend operations

### Common Issues

#### VM Creation Failures

1. **Asset not found**:
   - Check asset paths exist
   - Verify assetmanagerd integration
   - Check file permissions

2. **Network allocation failure**:
   - Verify CAP_NET_ADMIN capability
   - Check bridge configuration
   - Review IP allocation pool

3. **Jailer errors**:
   - Verify UID/GID configuration
   - Check chroot directory permissions
   - Review cgroup controllers

## Performance Tuning

### Database Optimization

1. **SQLite Tuning**:
   ```sql
   PRAGMA journal_mode = WAL;
   PRAGMA synchronous = NORMAL;
   PRAGMA cache_size = -64000; -- 64MB
   PRAGMA temp_store = MEMORY;
   ```

2. **Index Usage**:
   - Customer queries: `idx_vms_customer_id`
   - State queries: `idx_vms_state`
   - Composite: `idx_vms_customer_state`

### Network Performance

1. **TAP Device Pool** (future):
   - Pre-create TAP devices
   - Reduces creation latency

2. **Bridge Tuning**:
   ```bash
   # Increase bridge forwarding table
   echo 8192 > /sys/class/net/metald0/bridge/hash_max
   
   # Disable bridge netfilter
   echo 0 > /proc/sys/net/bridge/bridge-nf-call-iptables
   ```

3. **Network Namespace**:
   - Cache namespace handles
   - Batch network operations

### Resource Limits

Configure system limits for metald process:

```bash
# /etc/security/limits.d/metald.conf
metald soft nofile 65536
metald hard nofile 65536
metald soft nproc 32768
metald hard nproc 32768
```

## Troubleshooting

### Diagnostic Commands

```bash
# Check service status
systemctl status metald

# View recent logs
journalctl -u metald -n 100

# Check VM processes
ps aux | grep firecracker

# List network namespaces
ip netns list

# Check TAP devices
ip link show type tap

# Database integrity
sqlite3 /opt/metald/data/vms.db "PRAGMA integrity_check;"
```

### Recovery Procedures

#### Stuck VM Cleanup

```bash
# 1. Stop metald
systemctl stop metald

# 2. Clean up processes
pkill -f firecracker

# 3. Clean up network resources
ip netns list | grep vm- | xargs -I {} ip netns delete {}

# 4. Clean up TAP devices
ip link show type tap | grep tap-vm | awk '{print $2}' | sed 's/://' | xargs -I {} ip link delete {}

# 5. Start metald
systemctl start metald
```

#### Database Recovery

```bash
# Backup current database
cp /opt/metald/data/vms.db /opt/metald/data/vms.db.backup

# Check integrity
sqlite3 /opt/metald/data/vms.db "PRAGMA integrity_check;"

# If corrupted, recover from WAL
sqlite3 /opt/metald/data/vms.db "PRAGMA wal_checkpoint(TRUNCATE);"
```

### Performance Analysis

Use pprof for CPU and memory profiling:

```bash
# Enable profiling endpoint (development only)
UNKEY_METALD_ENABLE_PPROF=true ./metald

# CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profile
go tool pprof http://localhost:6060/debug/pprof/heap
```