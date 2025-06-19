# Unkey Deploy Operations Guide

## System Health Assessment

### Quick Health Check

Check all services are running and healthy:

```bash
# Service status
for service in metald billaged assetmanagerd builderd; do
  echo "=== $service ==="
  systemctl status $service --no-pager | grep -E "(Active:|Main PID:)"
  curl -s http://localhost:$(( 9464 + $(echo metald billaged builderd assetmanagerd | tr ' ' '\n' | grep -n "^$service$" | cut -d: -f1) - 1 ))/health || echo "UNHEALTHY"
  echo
done
```

### Metrics Endpoints

| Service | Metrics URL | Key Metrics |
|---------|-------------|-------------|
| metald | http://localhost:9464/metrics | `metald_vms_active`, `metald_api_requests_total` |
| billaged | http://localhost:9465/metrics | `billaged_metrics_processed_total`, `billaged_gaps_detected` |
| builderd | http://localhost:9466/metrics | `builderd_builds_total`, `builderd_build_duration_seconds` |
| assetmanagerd | http://localhost:9467/metrics | `assetmanagerd_assets_total`, `assetmanagerd_storage_bytes` |

### Service Dependencies Health

```bash
# Check metald can reach its dependencies
curl -X POST http://localhost:8080/api/v1/health/dependencies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_test"
```

## Deployment Procedures

### Initial Deployment

1. **Install Prerequisites**:
   ```bash
   # Install Go 1.23+
   # Install systemd
   # Install Firecracker (for metald)
   ```

2. **Build and Install Services**:
   ```bash
   make build
   sudo make install
   ```

3. **Configure Environment**:
   ```bash
   # Edit /etc/unkey/deploy/*.env files
   sudo systemctl daemon-reload
   ```

4. **Start Services in Order**:
   ```bash
   # Start infrastructure
   sudo systemctl start spire-server spire-agent
   
   # Start core services
   sudo systemctl start assetmanagerd
   sudo systemctl start billaged
   sudo systemctl start metald
   sudo systemctl start builderd
   ```

### Rolling Updates

Update services with zero downtime:

```bash
# 1. Build new version
SERVICE=metald make build

# 2. Install new binary (doesn't restart)
SERVICE=metald sudo make install-binary

# 3. Reload service
sudo systemctl reload-or-restart metald

# 4. Verify health
curl http://localhost:9464/health
```

### Blue-Green Deployment

For critical updates:

1. **Deploy to staging environment**
2. **Run integration tests**
3. **Switch load balancer to new deployment**
4. **Monitor metrics for anomalies**
5. **Keep old deployment for quick rollback**

## Monitoring and Alerting

### Key Metrics to Monitor

#### System Health
```promql
# Service availability
up{job=~"metald|billaged|assetmanagerd|builderd"} == 0

# High error rates
rate(api_requests_total{status=~"5.."}[5m]) > 0.01

# Memory pressure
process_resident_memory_bytes / process_virtual_memory_bytes > 0.8
```

#### Business Metrics
```promql
# Active VMs
metald_vms_active

# Billing gaps
increase(billaged_gaps_detected_total[1h]) > 0

# Asset storage usage
assetmanagerd_storage_bytes / assetmanagerd_storage_capacity_bytes > 0.9
```

### Recommended Alerts

| Alert | Condition | Severity | Action |
|-------|-----------|----------|--------|
| ServiceDown | `up == 0` for 5m | Critical | Check systemd logs, restart service |
| HighErrorRate | Error rate > 1% | Warning | Check application logs |
| BillingGap | Gaps detected | Warning | Check metald/billaged connectivity |
| StorageFull | > 90% capacity | Critical | Clean old assets, expand storage |
| VmCreateFailure | Failures > 10/min | Critical | Check Firecracker, disk space |

### Grafana Dashboards

Import dashboards from service contrib directories:
```bash
for service in metald billaged assetmanagerd builderd; do
  cp $service/contrib/grafana-dashboards/*.json /var/lib/grafana/dashboards/
done
```

## Incident Response

### Common Issues and Resolution

#### 1. Service Won't Start

**Symptoms**: systemctl status shows failed
**Diagnosis**:
```bash
# Check logs
journalctl -u $SERVICE -n 100 --no-pager

# Common issues:
# - Port already in use
# - Missing config file
# - Permission denied
```

**Resolution**:
- Fix configuration errors
- Check port conflicts: `ss -tlnp | grep :808`
- Verify permissions on config/data directories

#### 2. VM Creation Failures

**Symptoms**: CreateVm returns errors
**Diagnosis**:
```bash
# Check metald logs
journalctl -u metald -f | grep ERROR

# Check disk space
df -h /var/lib/metald

# Check Firecracker
sudo -u metald firecracker --version
```

**Resolution**:
- Ensure sufficient disk space (>10GB free)
- Verify Firecracker binary permissions
- Check jailer directory permissions

#### 3. Billing Data Gaps

**Symptoms**: billaged reports gaps
**Diagnosis**:
```bash
# Check connectivity
curl -X POST http://localhost:8081/api/v1/debug/gaps

# Check metald billing client
journalctl -u metald | grep "billing.*error"
```

**Resolution**:
- Restart both metald and billaged
- Gap recovery happens automatically
- Monitor `billaged_gaps_recovered_total`

#### 4. Asset Download Failures

**Symptoms**: PrepareAssets fails
**Diagnosis**:
```bash
# Check storage backend
curl http://localhost:9467/metrics | grep storage_errors

# Check disk space
df -h /var/lib/assetmanagerd

# Test storage connectivity (S3/GCS)
```

**Resolution**:
- Verify storage credentials
- Check network connectivity
- Clear corrupted cache: `rm -rf /var/lib/assetmanagerd/cache/*`

### Debugging Tools

#### Service Debug Endpoints

```bash
# metald - List active VMs
curl http://localhost:8080/api/v1/debug/vms

# billaged - Show buffered metrics
curl http://localhost:8081/api/v1/debug/buffer

# assetmanagerd - Asset status
curl http://localhost:8083/api/v1/debug/assets
```

#### Distributed Tracing

Access Jaeger UI (if configured):
```bash
# Default Jaeger URL
http://localhost:16686

# Find slow requests
# Search by service: metald
# Min duration: 1s
```

#### Log Correlation

Find related logs across services:
```bash
# Extract request ID from error
REQUEST_ID="12345-67890"

# Search all services
for service in metald billaged assetmanagerd; do
  echo "=== $service ==="
  journalctl -u $service | grep $REQUEST_ID
done
```

## Backup and Recovery

### Data Backup

#### assetmanagerd SQLite
```bash
# Online backup
sqlite3 /var/lib/assetmanagerd/assets.db ".backup /backup/assets-$(date +%Y%m%d).db"

# Verify backup
sqlite3 /backup/assets-*.db "SELECT COUNT(*) FROM assets;"
```

#### Object Storage
- S3: Use bucket versioning and lifecycle policies
- GCS: Enable object versioning
- Local: Regular rsync to backup location

### Disaster Recovery

#### Service Recovery Order
1. **SPIFFE/SPIRE** - Authentication infrastructure
2. **assetmanagerd** - Required for VM creation
3. **billaged** - Can recover from gaps
4. **metald** - Orchestration layer
5. **builderd** - Non-critical path

#### Data Recovery Priority
1. **Asset metadata** (SQLite databases)
2. **Asset blobs** (object storage)
3. **Configuration** (/etc/unkey/deploy/)
4. **TLS certificates** (if not using SPIFFE)

## Performance Tuning

### Service Tuning

#### metald
```bash
# Increase worker pool for high VM churn
UNKEY_METALD_WORKER_POOL_SIZE=50

# Reduce metrics interval for less load
UNKEY_METALD_METRICS_INTERVAL=30s
```

#### assetmanagerd
```bash
# Increase cache size for frequently used assets
UNKEY_ASSETMANAGERD_CACHE_SIZE_GB=100

# Enable parallel downloads
UNKEY_ASSETMANAGERD_PARALLEL_DOWNLOADS=10
```

### System Tuning

```bash
# Increase file descriptor limits
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# Optimize network settings
cat >> /etc/sysctl.conf <<EOF
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.core.netdev_max_backlog = 65535
EOF
sysctl -p
```

## Maintenance Tasks

### Regular Maintenance

#### Daily
- Check service health endpoints
- Review error logs for anomalies
- Verify backup completion

#### Weekly
- Clean old staged assets: `assetmanagerd cleanup --older-than=7d`
- Review metrics for capacity planning
- Update monitoring thresholds

#### Monthly
- Rotate logs if not automatic
- Review and update documentation
- Test disaster recovery procedures

### Asset Cleanup

```bash
# Remove orphaned staged assets
curl -X POST http://localhost:8083/api/v1/maintenance/cleanup

# Prune old unused assets
curl -X POST http://localhost:8083/api/v1/assets/prune \
  -d '{"older_than_days": 30}'
```

---

For development procedures, see [Development Guide](../development/)
For architecture details, see [Architecture Documentation](../architecture/)