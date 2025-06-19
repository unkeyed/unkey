# Builderd Operations Manual

## Installation and Configuration

### System Requirements

- **OS**: Linux (Ubuntu 20.04+, RHEL 8+, or compatible)
- **CPU**: Minimum 2 cores, recommended 4+ cores
- **Memory**: Minimum 4GB, recommended 8GB+
- **Storage**: 50GB+ SSD for build workspace and cache
- **Docker**: Version 20.10+ installed and running
- **Network**: Outbound HTTPS for registry access

### Installation

#### Binary Installation

```bash
cd builderd
make build    # Build the binary
make install  # Install with systemd unit
```

#### Container Deployment

```bash
docker run -d \
  --name builderd \
  -p 8082:8082 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /opt/builderd:/opt/builderd \
  -e UNKEY_BUILDERD_TLS_MODE=spiffe \
  -e UNKEY_BUILDERD_SPIFFE_SOCKET=/run/spire/sockets/agent.sock \
  -v /run/spire/sockets:/run/spire/sockets:ro \
  ghcr.io/unkeyed/builderd:latest
```

### Configuration

All configuration is done via environment variables following the `UNKEY_BUILDERD_*` pattern.

#### Core Configuration

```bash
# Server settings
UNKEY_BUILDERD_PORT=8082                    # Service port
UNKEY_BUILDERD_ADDRESS=0.0.0.0              # Bind address
UNKEY_BUILDERD_SHUTDOWN_TIMEOUT=15s         # Graceful shutdown timeout
UNKEY_BUILDERD_RATE_LIMIT=100               # Health endpoint rate limit

# Build settings
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=5      # Parallel build limit
UNKEY_BUILDERD_BUILD_TIMEOUT=15m            # Build timeout
UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd    # Temp directory
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/opt/builderd/rootfs  # Output directory
UNKEY_BUILDERD_WORKSPACE_DIR=/opt/builderd/workspace   # Build workspace
```

#### Storage Configuration

```bash
# Storage backend
UNKEY_BUILDERD_STORAGE_BACKEND=local        # local, s3, gcs
UNKEY_BUILDERD_STORAGE_RETENTION_DAYS=30    # Artifact retention
UNKEY_BUILDERD_STORAGE_MAX_SIZE_GB=100      # Max storage size
UNKEY_BUILDERD_STORAGE_CACHE_ENABLED=true   # Enable caching
UNKEY_BUILDERD_STORAGE_CACHE_MAX_SIZE_GB=50 # Cache size limit

# S3 configuration (if backend=s3)
UNKEY_BUILDERD_STORAGE_S3_BUCKET=builderd-artifacts
UNKEY_BUILDERD_STORAGE_S3_REGION=us-east-1
UNKEY_BUILDERD_STORAGE_S3_ACCESS_KEY=AKIAXXXXXXXX
UNKEY_BUILDERD_STORAGE_S3_SECRET_KEY=xxxxxxxxxx
```

#### Security Configuration

```bash
# TLS/mTLS settings
UNKEY_BUILDERD_TLS_MODE=spiffe              # disabled, file, spiffe
UNKEY_BUILDERD_SPIFFE_SOCKET=/run/spire/sockets/agent.sock

# Docker settings
UNKEY_BUILDERD_DOCKER_REGISTRY_AUTH=true    # Enable registry auth
UNKEY_BUILDERD_DOCKER_MAX_IMAGE_SIZE_GB=5   # Max image size
UNKEY_BUILDERD_DOCKER_PULL_TIMEOUT=10m      # Pull timeout
```

#### Tenant Configuration

```bash
# Default limits
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_MEMORY_BYTES=2147483648  # 2GB
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_CPU_CORES=2
UNKEY_BUILDERD_TENANT_DEFAULT_MAX_DAILY_BUILDS=100
UNKEY_BUILDERD_TENANT_DEFAULT_TIMEOUT_SECONDS=900
UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED=true
```

## Monitoring and Metrics

### Prometheus Metrics

When OpenTelemetry is enabled, builderd exports metrics on the configured port (default 9466).

#### Key Metrics

**Build Metrics**:
- `builderd_builds_total{state,source,target,tenant}` - Total builds by status
- `builderd_build_duration_seconds` - Build execution time histogram
- `builderd_concurrent_builds` - Current running builds gauge
- `builderd_build_size_bytes` - Rootfs size distribution

**Resource Metrics**:
- `builderd_cpu_usage_cores{tenant}` - CPU usage per tenant
- `builderd_memory_usage_bytes{tenant}` - Memory usage per tenant
- `builderd_disk_usage_bytes{tenant}` - Disk usage per tenant

**Dependency Metrics**:
- `builderd_docker_operations_total{operation,status}` - Docker operation counts
- `builderd_assetmanager_calls_total{method,status}` - AssetManagerd calls
- `builderd_storage_operations_duration_seconds` - Storage operation latency

#### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'builderd'
    static_configs:
      - targets: ['localhost:9466']
    scrape_interval: 15s
```

### OpenTelemetry Configuration

```bash
# Enable OpenTelemetry
UNKEY_BUILDERD_OTEL_ENABLED=true
UNKEY_BUILDERD_OTEL_SERVICE_NAME=builderd
UNKEY_BUILDERD_OTEL_SERVICE_VERSION=0.1.0
UNKEY_BUILDERD_OTEL_SAMPLING_RATE=1.0
UNKEY_BUILDERD_OTEL_ENDPOINT=localhost:4318

# Prometheus exporter
UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED=true
UNKEY_BUILDERD_OTEL_PROMETHEUS_PORT=9466
UNKEY_BUILDERD_OTEL_PROMETHEUS_INTERFACE=127.0.0.1
```

### Grafana Dashboards

Import the provided dashboard from `builderd/contrib/grafana-dashboards/`:

1. **Build Overview**: Build rates, success rates, duration percentiles
2. **Resource Usage**: CPU, memory, disk usage by tenant
3. **Error Analysis**: Error rates by type and tenant
4. **Dependency Health**: External service status and latencies

## Health Checks and Alerting

### Health Check Endpoint

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime_seconds": 3600,
  "checks": {
    "docker": "healthy",
    "storage": "healthy",
    "assetmanager": "healthy"
  }
}
```

**Implementation**: [main.go:314](../../../cmd/builderd/main.go:314)

### Alerting Rules

Example Prometheus alerting rules:

```yaml
groups:
  - name: builderd
    rules:
      - alert: BuilderdHighFailureRate
        expr: rate(builderd_builds_total{state="failed"}[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: High build failure rate
          
      - alert: BuilderdStorageFull
        expr: builderd_storage_usage_bytes / builderd_storage_limit_bytes > 0.9
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: Storage approaching capacity
          
      - alert: BuilderdUnhealthy
        expr: up{job="builderd"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: Builderd is down
```

## Backup and Recovery

### Data to Backup

1. **Build Artifacts** (if using local storage)
   - Location: `$UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR`
   - Frequency: Daily incremental, weekly full
   - Retention: Match `STORAGE_RETENTION_DAYS`

2. **Database** (when implemented)
   - SQLite: Copy database file
   - PostgreSQL: pg_dump

3. **Configuration**
   - Environment variables
   - TLS certificates (if using file mode)

### Backup Script Example

```bash
#!/bin/bash
# builderd-backup.sh

BACKUP_DIR="/backup/builderd/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# Backup artifacts (local storage only)
if [ "$UNKEY_BUILDERD_STORAGE_BACKEND" = "local" ]; then
  rsync -av --delete \
    "$UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR/" \
    "$BACKUP_DIR/artifacts/"
fi

# Backup database (future)
if [ -f "/opt/builderd/data/builderd.db" ]; then
  cp "/opt/builderd/data/builderd.db" "$BACKUP_DIR/"
fi

# Backup configuration
env | grep ^UNKEY_BUILDERD_ > "$BACKUP_DIR/config.env"
```

### Recovery Procedures

1. **Service Recovery**:
   ```bash
   # Stop service
   systemctl stop builderd
   
   # Restore artifacts
   rsync -av /backup/builderd/latest/artifacts/ $UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR/
   
   # Restore database (if applicable)
   cp /backup/builderd/latest/builderd.db /opt/builderd/data/
   
   # Start service
   systemctl start builderd
   ```

2. **Disaster Recovery**:
   - Deploy new builderd instance
   - Restore configuration from backup
   - Restore artifacts/database
   - Update DNS/load balancer
   - Verify service health

## Performance Tuning

### System Tuning

1. **File Descriptors**:
   ```bash
   # /etc/security/limits.conf
   builderd soft nofile 65536
   builderd hard nofile 65536
   ```

2. **Kernel Parameters**:
   ```bash
   # /etc/sysctl.conf
   vm.max_map_count = 262144
   fs.file-max = 2097152
   net.ipv4.tcp_keepalive_time = 600
   ```

3. **Docker Configuration**:
   ```json
   {
     "storage-driver": "overlay2",
     "log-driver": "json-file",
     "log-opts": {
       "max-size": "10m",
       "max-file": "3"
     }
   }
   ```

### Build Performance

1. **Registry Mirror**:
   ```bash
   UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR=https://mirror.internal
   ```

2. **Concurrent Builds**:
   - Adjust based on CPU/memory
   - Rule of thumb: 1 build per 2 CPU cores
   - Monitor resource usage

3. **Cache Tuning**:
   - Size cache based on working set
   - Monitor cache hit rates
   - Adjust eviction policies

### Storage Optimization

1. **Local Storage**:
   - Use SSD for workspace/output
   - Enable compression for artifacts
   - Regular cleanup of old builds

2. **S3 Storage**:
   - Enable S3 Transfer Acceleration
   - Use appropriate storage class
   - Configure lifecycle policies

## Troubleshooting Guide

### Common Issues

#### 1. Build Failures

**Symptom**: Builds fail with "pull access denied"
**Cause**: Missing registry authentication
**Solution**:
```bash
# Configure Docker auth
docker login ghcr.io
# Restart builderd to pick up credentials
systemctl restart builderd
```

**Symptom**: Builds fail with "no space left on device"
**Cause**: Disk full or quota exceeded
**Solution**:
```bash
# Check disk usage
df -h $UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR
# Clean old artifacts
find $UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR -mtime +30 -delete
# Increase quota limits
UNKEY_BUILDERD_STORAGE_MAX_SIZE_GB=200
```

#### 2. Performance Issues

**Symptom**: Slow build times
**Diagnosis**:
```bash
# Check metrics
curl -s localhost:9466/metrics | grep builderd_build_duration_seconds
# Check Docker daemon
docker system df
docker system events
```
**Solutions**:
- Enable registry mirror
- Increase concurrent builds
- Add more CPU/memory

#### 3. Connection Issues

**Symptom**: "connection refused" errors
**Diagnosis**:
```bash
# Check service status
systemctl status builderd
# Check port binding
ss -tlnp | grep 8082
# Check TLS configuration
openssl s_client -connect localhost:8082
```

### Debug Mode

Enable debug logging:
```bash
UNKEY_BUILDERD_LOG_LEVEL=debug
UNKEY_BUILDERD_LOG_FORMAT=json
```

### Log Analysis

Key log patterns to watch:

```bash
# Build failures
jq 'select(.level=="error" and .build_id)' /var/log/builderd.log

# Slow operations
jq 'select(.duration_ms > 5000)' /var/log/builderd.log

# Tenant issues
jq 'select(.tenant_id=="tenant-123")' /var/log/builderd.log
```

## Maintenance Procedures

### Regular Maintenance

**Daily**:
- Monitor error rates and alerts
- Check disk usage
- Review failed builds

**Weekly**:
- Clean old build artifacts
- Review performance metrics
- Update security patches

**Monthly**:
- Analyze usage trends
- Optimize cache settings
- Review tenant quotas
- Audit access logs

### Upgrade Procedures

1. **Rolling Upgrade** (with multiple instances):
   ```bash
   # For each instance:
   kubectl drain node-X
   kubectl apply -f builderd-new.yaml
   kubectl uncordon node-X
   ```

2. **In-place Upgrade** (single instance):
   ```bash
   # Backup current version
   cp /usr/local/bin/builderd /usr/local/bin/builderd.bak
   
   # Stop service
   systemctl stop builderd
   
   # Install new version
   make install
   
   # Start service
   systemctl start builderd
   
   # Verify health
   curl http://localhost:8082/health
   ```

### Capacity Planning

Monitor these metrics for capacity planning:

1. **Build Queue Depth**: Indicates need for more workers
2. **Resource Utilization**: CPU/memory usage trends
3. **Storage Growth**: Artifact accumulation rate
4. **Error Rates**: May indicate resource constraints

Scaling recommendations:
- **Vertical**: Add CPU/memory when utilization > 80%
- **Horizontal**: Add instances when queue depth > 10
- **Storage**: Expand when usage > 70% of capacity

## Security Operations

### Security Hardening

1. **Network Security**:
   ```bash
   # Firewall rules
   ufw allow from 10.0.0.0/8 to any port 8082
   ufw allow from 172.16.0.0/12 to any port 9466
   ```

2. **Process Isolation**:
   - Run as non-root user
   - Use systemd security features
   - Enable AppArmor/SELinux

3. **TLS Configuration**:
   - Enforce TLS 1.3 minimum
   - Regular certificate rotation
   - Strong cipher suites

### Audit Logging

Enable comprehensive audit logging:

```bash
UNKEY_BUILDERD_AUDIT_ENABLED=true
UNKEY_BUILDERD_AUDIT_LOG_PATH=/var/log/builderd-audit.log
```

Audit log format:
```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "event": "build.create",
  "tenant_id": "tenant-123",
  "user_id": "user-456",
  "resource": "build-abc",
  "action": "create",
  "result": "success"
}
```

### Incident Response

1. **Detection**: Monitor alerts and anomalies
2. **Containment**: Isolate affected tenants
3. **Investigation**: Analyze logs and metrics
4. **Recovery**: Restore service functionality
5. **Post-mortem**: Document and improve

## Disaster Recovery

### RTO/RPO Targets

- **RTO** (Recovery Time Objective): 1 hour
- **RPO** (Recovery Point Objective): 1 hour

### DR Procedures

1. **Primary Site Failure**:
   - Activate DR site
   - Update DNS to DR endpoint
   - Restore from latest backup
   - Verify service health

2. **Data Corruption**:
   - Stop service immediately
   - Identify corruption scope
   - Restore from known-good backup
   - Replay missing builds if needed

3. **Security Breach**:
   - Isolate affected systems
   - Rotate all credentials
   - Audit all access logs
   - Implement additional controls
