# AssetManagerd Operations Guide

This guide covers deployment, monitoring, and operational procedures for AssetManagerd in production environments.

## Deployment

### System Requirements

- **OS**: Linux (Ubuntu 22.04+ or compatible)
- **Memory**: 512MB minimum, 2GB recommended
- **Storage**: 10GB minimum for metadata + asset storage needs
- **Network**: Low latency to metald and builderd services

### Installation

AssetManagerd is distributed as a single Go binary and systemd service:

```bash
# Build and install (from project root)
cd assetmanagerd
make install

# This will:
# 1. Build the binary to ./build/assetmanagerd
# 2. Install to /usr/local/bin/assetmanagerd
# 3. Install systemd unit from contrib/systemd/
# 4. Create required directories
```

### Directory Structure

```
/opt/assetmanagerd/
├── assets.db           # SQLite metadata database
└── logs/              # Service logs (if not using journald)

/opt/vm-assets/        # Asset storage (configurable)
├── ab/                # Sharded directories
├── cd/
└── ...

/etc/assetmanagerd/    # Configuration (optional)
└── config.yaml        # Not currently used
```

### Systemd Service

**Unit file**: [contrib/systemd/assetmanagerd.service](../../contrib/systemd/assetmanagerd.service)

```bash
# Service management
systemctl start assetmanagerd
systemctl status assetmanagerd
systemctl enable assetmanagerd  # Auto-start on boot

# View logs
journalctl -u assetmanagerd -f
```

### Configuration

All configuration via environment variables:

```bash
# /etc/assetmanagerd/environment
UNKEY_ASSETMANAGERD_PORT=8083
UNKEY_ASSETMANAGERD_METRICS_PORT=9467
UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local
UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/opt/vm-assets
UNKEY_ASSETMANAGERD_DATABASE_PATH=/opt/assetmanagerd/assets.db
UNKEY_ASSETMANAGERD_GC_ENABLED=true
UNKEY_ASSETMANAGERD_GC_INTERVAL=3600s
UNKEY_ASSETMANAGERD_GC_AGE_THRESHOLD=604800  # 7 days in seconds
UNKEY_ASSETMANAGERD_LOG_LEVEL=info
UNKEY_ASSETMANAGERD_TLS_MODE=spiffe
SPIFFE_ENDPOINT_SOCKET=unix:///tmp/spire-agent/public/api.sock
```

## Monitoring

### Health Checks

AssetManagerd exposes a gRPC health check endpoint:

```bash
# Using grpc-health-probe
grpc-health-probe -addr=localhost:8083 -tls -tls-no-verify

# Response codes:
# - SERVING: Healthy and ready
# - NOT_SERVING: Unhealthy
# - UNKNOWN: Health check not implemented
```

**Implementation**: Standard gRPC health service ([cmd/assetmanagerd/main.go](../../cmd/assetmanagerd/main.go))

### Metrics

Prometheus metrics exposed on port 9467:

#### RPC Metrics
- `assetmanagerd_rpc_requests_total{method,status}` - Total RPC requests
- `assetmanagerd_rpc_duration_seconds{method,status}` - RPC latency histogram
- `assetmanagerd_rpc_requests_in_flight` - Current concurrent requests

#### Storage Metrics  
- `assetmanagerd_storage_operations_total{op,status}` - Storage operations
- `assetmanagerd_storage_duration_seconds{op}` - Storage operation latency
- `assetmanagerd_storage_bytes_total{op}` - Bytes read/written

#### Asset Metrics
- `assetmanagerd_assets_total{type,status}` - Total assets by type/status
- `assetmanagerd_assets_size_bytes_total{type}` - Total storage by asset type
- `assetmanagerd_assets_references_total` - Total active references
- `assetmanagerd_assets_leases_active` - Active lease count

#### GC Metrics
- `assetmanagerd_gc_runs_total{status}` - GC execution count
- `assetmanagerd_gc_duration_seconds` - GC execution time
- `assetmanagerd_gc_assets_deleted_total` - Assets cleaned up
- `assetmanagerd_gc_bytes_reclaimed_total` - Storage reclaimed

### Prometheus Configuration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'assetmanagerd'
    static_configs:
      - targets: ['assetmanagerd:9467']
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'go_.*'
        action: drop  # Drop Go runtime metrics if not needed
```

### Grafana Dashboard

Import the dashboard from [contrib/grafana-dashboards/assetmanagerd.json](../../contrib/grafana-dashboards/) (when available).

Key panels:
- RPC request rate and latency
- Storage operations and failures
- Asset inventory by type
- Reference count trends
- GC effectiveness

### Alerts

Example Prometheus alert rules:

```yaml
groups:
  - name: assetmanagerd
    rules:
      - alert: AssetManagerdDown
        expr: up{job="assetmanagerd"} == 0
        for: 5m
        annotations:
          summary: "AssetManagerd is down"
          
      - alert: AssetManagerdHighErrorRate
        expr: rate(assetmanagerd_rpc_requests_total{status="error"}[5m]) > 0.05
        for: 10m
        annotations:
          summary: "High RPC error rate"
          
      - alert: AssetManagerdStorageFull
        expr: disk_usage_percent{path="/opt/vm-assets"} > 90
        for: 5m
        annotations:
          summary: "Asset storage near capacity"
          
      - alert: AssetManagerdGCFailing
        expr: rate(assetmanagerd_gc_runs_total{status="error"}[1h]) > 0
        annotations:
          summary: "Garbage collection failures"
```

## Logging

AssetManagerd uses structured logging with slog:

```json
{
  "time": "2024-01-20T10:30:45Z",
  "level": "INFO",
  "msg": "registered asset",
  "asset_id": "550e8400-e29b-41d4-a716",
  "type": "KERNEL",
  "size_bytes": 12345678,
  "trace_id": "abc123"
}
```

### Log Levels

Set via `UNKEY_ASSETMANAGERD_LOG_LEVEL`:
- `debug` - Verbose debugging information
- `info` - Normal operations (default)
- `warn` - Warning conditions
- `error` - Error conditions

### Log Aggregation

For production, aggregate logs to a central system:

```bash
# Vector configuration example
[sources.assetmanagerd]
type = "journald"
include_units = ["assetmanagerd"]

[transforms.parse_assetmanagerd]
type = "remap"
inputs = ["assetmanagerd"]
source = '''
.service = "assetmanagerd"
.level = .MESSAGE.level
.asset_id = .MESSAGE.asset_id
'''

[sinks.elasticsearch]
type = "elasticsearch"
inputs = ["parse_assetmanagerd"]
endpoints = ["http://elasticsearch:9200"]
```

## Operational Procedures

### Backup and Recovery

#### Database Backup

The SQLite database contains critical metadata:

```bash
# Online backup (safe while running)
sqlite3 /opt/assetmanagerd/assets.db ".backup /backup/assets.db"

# Scheduled backup script
#!/bin/bash
BACKUP_DIR="/backup/assetmanagerd/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"
sqlite3 /opt/assetmanagerd/assets.db ".backup $BACKUP_DIR/assets.db"
find /backup/assetmanagerd -mtime +7 -delete  # Keep 7 days
```

#### Asset Recovery

Assets can be rebuilt from metadata if files are lost:

```bash
# List all assets in database
sqlite3 /opt/assetmanagerd/assets.db \
  "SELECT id, location, checksum FROM assets WHERE status = 2"

# Verify asset integrity
find /opt/vm-assets -type f -exec sha256sum {} \; > checksums.txt
# Compare with database checksums
```

### Capacity Planning

#### Storage Requirements

Calculate based on asset types and retention:

```
Daily storage = (kernels × 50MB) + (rootfs × 500MB) + (disk_images × size)
Total storage = Daily storage × Retention days × 1.2 (20% overhead)
```

Example for 100 VMs/day, 7-day retention:
```
Kernels: 100 × 50MB × 7 = 35GB
RootFS: 100 × 500MB × 7 = 350GB  
Total: (35 + 350) × 1.2 = 462GB
```

#### Database Growth

SQLite database grows with asset count:
- ~1KB per asset record
- ~0.5KB per lease record
- Negligible for millions of assets

### Manual Garbage Collection

Trigger GC manually when needed:

```bash
# Dry run to see what would be deleted
grpcurl -plaintext localhost:8083 \
  asset.v1.AssetService/GarbageCollect \
  -d '{"dry_run": true, "age_threshold_seconds": 604800}'

# Execute GC
grpcurl -plaintext localhost:8083 \
  asset.v1.AssetService/GarbageCollect \
  -d '{"age_threshold_seconds": 604800}'
```

### Performance Tuning

#### SQLite Optimization

```sql
-- Set pragmas for performance (add to startup)
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = -64000;  -- 64MB cache
PRAGMA temp_store = MEMORY;
```

#### Filesystem Tuning

For XFS (recommended for asset storage):
```bash
# Mount options for performance
mount -o noatime,nodiratime,logbufs=8,logbsize=256k /dev/vdb /opt/vm-assets

# Increase directory size for many files
xfs_growfs -d /opt/vm-assets
```

### Troubleshooting

#### Common Issues

**1. Cannot acquire assets - "no such file"**
- Check asset exists: `ls -la /opt/vm-assets/{first-2-chars}/{asset-id}`
- Verify database record: `sqlite3 assets.db "SELECT * FROM assets WHERE id='....'"`
- Check storage permissions

**2. High memory usage**
- Check for memory leaks: `pprof` endpoint (if enabled)
- Review SQLite cache size
- Check for stuck GC operations

**3. Slow asset preparation**
- Verify same filesystem for hard links
- Check disk I/O: `iostat -x 1`
- Monitor network if using remote storage

#### Debug Mode

Enable debug logging for troubleshooting:
```bash
UNKEY_ASSETMANAGERD_LOG_LEVEL=debug systemctl restart assetmanagerd
```

#### Database Corruption

If SQLite corruption suspected:
```bash
# Check integrity
sqlite3 /opt/assetmanagerd/assets.db "PRAGMA integrity_check"

# Recover from backup
systemctl stop assetmanagerd
mv assets.db assets.db.corrupt
cp /backup/assets.db.latest assets.db
systemctl start assetmanagerd
```

## Security Operations

### TLS Certificate Rotation

When using SPIFFE, certificates auto-rotate. Monitor rotation:

```bash
# Check certificate expiry
openssl s_client -connect localhost:8083 -servername assetmanagerd \
  -cert client.crt -key client.key 2>/dev/null | \
  openssl x509 -noout -dates
```

### Audit Logging

Track asset access for compliance:

```sql
-- Query recent asset access
SELECT 
    a.id,
    a.name,
    al.acquired_by,
    datetime(al.acquired_at, 'unixepoch') as acquired_time
FROM assets a
JOIN asset_leases al ON a.id = al.asset_id
WHERE al.acquired_at > strftime('%s', 'now', '-1 day')
ORDER BY al.acquired_at DESC;
```

### Security Scanning

Regular security tasks:
1. Scan asset contents for vulnerabilities
2. Verify checksums haven't changed
3. Review access logs for anomalies
4. Update Go binary for security patches

## Maintenance Windows

### Rolling Updates

For zero-downtime updates with multiple instances:

```bash
# 1. Update one instance at a time
# 2. Verify health before proceeding
# 3. Monitor error rates during rollout

for host in assetmgr-{1..3}; do
    ssh $host "systemctl stop assetmanagerd"
    scp build/assetmanagerd $host:/usr/local/bin/
    ssh $host "systemctl start assetmanagerd"
    sleep 30
    grpc-health-probe -addr=$host:8083 -tls || exit 1
done
```

### Database Maintenance

Periodic optimization:

```bash
# Monthly maintenance script
#!/bin/bash
sqlite3 /opt/assetmanagerd/assets.db <<EOF
VACUUM;
ANALYZE;
PRAGMA optimize;
EOF
```

## Disaster Recovery

### RTO/RPO Targets

- **RTO** (Recovery Time Objective): 15 minutes
- **RPO** (Recovery Point Objective): 1 hour

### Recovery Procedures

1. **Service Failure**: Auto-restart via systemd
2. **Node Failure**: Deploy to new node, restore from backup
3. **Data Loss**: Restore database, re-register missing assets
4. **Complete Loss**: Rebuild from builderd artifacts

### Testing

Regular DR testing schedule:
- Monthly: Backup restoration test
- Quarterly: Full node recovery
- Annually: Complete service rebuild