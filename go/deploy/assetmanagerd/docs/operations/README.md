# Assetmanagerd Operations Manual

This document provides comprehensive guidance for deploying, configuring, monitoring, and troubleshooting assetmanagerd in production environments.

## Installation & Deployment

### Binary Installation

Build and install using the service Makefile:

```bash
# Build the binary
cd assetmanagerd
make build

# Install with systemd unit
sudo make install
```

**Build Configuration**: [Makefile](../../Makefile)

### Systemd Service

Service unit provides robust process management:

**Unit File**: [assetmanagerd.service](../../contrib/systemd/assetmanagerd.service)

```bash
# Service management
sudo systemctl enable assetmanagerd
sudo systemctl start assetmanagerd
sudo systemctl status assetmanagerd

# Log monitoring
sudo journalctl -fu assetmanagerd
```

### Container Deployment

Docker deployment with proper volume mounts:

```dockerfile
FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY assetmanagerd /usr/local/bin/
EXPOSE 8083 9467
USER 1000:1000
ENTRYPOINT ["/usr/local/bin/assetmanagerd"]
```

Volume requirements:
- `/opt/vm-assets`: Asset storage directory
- `/opt/assetmanagerd`: Database and cache directory
- `/var/lib/spire/agent/agent.sock`: SPIFFE agent socket

## Configuration

### Environment Variables

Complete configuration reference with defaults and validation rules.

**Config Source**: [config.go](../../internal/config/config.go)

#### Service Configuration

```bash
# Network configuration
UNKEY_ASSETMANAGERD_PORT=8083                    # Service port (1-65535)
UNKEY_ASSETMANAGERD_ADDRESS=0.0.0.0              # Bind address

# Storage backend selection
UNKEY_ASSETMANAGERD_STORAGE_BACKEND=local        # local|s3|nfs
UNKEY_ASSETMANAGERD_LOCAL_STORAGE_PATH=/opt/vm-assets
UNKEY_ASSETMANAGERD_DATABASE_PATH=/opt/assetmanagerd/assets.db
UNKEY_ASSETMANAGERD_CACHE_DIR=/opt/assetmanagerd/cache
```

#### Storage Limits & Performance

```bash
# Asset size limits
UNKEY_ASSETMANAGERD_MAX_ASSET_SIZE=10737418240    # 10GB max per asset
UNKEY_ASSETMANAGERD_MAX_CACHE_SIZE=107374182400   # 100GB total cache

# Performance tuning
UNKEY_ASSETMANAGERD_DOWNLOAD_CONCURRENCY=4       # Concurrent downloads
UNKEY_ASSETMANAGERD_DOWNLOAD_TIMEOUT=30m         # Download timeout
```

#### Garbage Collection

```bash
# GC scheduling and thresholds
UNKEY_ASSETMANAGERD_GC_ENABLED=true              # Enable background GC
UNKEY_ASSETMANAGERD_GC_INTERVAL=1h               # GC run frequency
UNKEY_ASSETMANAGERD_GC_MAX_AGE=168h              # 7 days retention
UNKEY_ASSETMANAGERD_GC_MIN_REFERENCES=0          # Min refs to preserve
```

#### Builderd Integration

```bash
# Auto-build configuration
UNKEY_ASSETMANAGERD_BUILDERD_ENABLED=true                    # Enable integration
UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT=https://localhost:8082 # Builderd URL
UNKEY_ASSETMANAGERD_BUILDERD_TIMEOUT=30m                     # Build timeout
UNKEY_ASSETMANAGERD_BUILDERD_AUTO_REGISTER=true              # Auto-register builds
UNKEY_ASSETMANAGERD_BUILDERD_MAX_RETRIES=3                   # Retry attempts
UNKEY_ASSETMANAGERD_BUILDERD_RETRY_DELAY=5s                  # Retry delay
```

#### TLS/SPIFFE Configuration

```bash
# SPIFFE/SPIRE integration (required)
UNKEY_ASSETMANAGERD_TLS_MODE=spiffe                          # spiffe|file|disabled
UNKEY_ASSETMANAGERD_SPIFFE_SOCKET=/var/lib/spire/agent/agent.sock

# File-based TLS (alternative)
UNKEY_ASSETMANAGERD_TLS_CERT_FILE=/path/to/cert.pem
UNKEY_ASSETMANAGERD_TLS_KEY_FILE=/path/to/key.pem
UNKEY_ASSETMANAGERD_TLS_CA_FILE=/path/to/ca.pem
```

#### Observability

```bash
# OpenTelemetry configuration
UNKEY_ASSETMANAGERD_OTEL_ENABLED=true                        # Enable telemetry
UNKEY_ASSETMANAGERD_OTEL_SERVICE_NAME=assetmanagerd
UNKEY_ASSETMANAGERD_OTEL_SERVICE_VERSION=0.2.0
UNKEY_ASSETMANAGERD_OTEL_ENDPOINT=localhost:4318             # OTLP collector
UNKEY_ASSETMANAGERD_OTEL_SAMPLING_RATE=1.0                   # Trace sampling

# Prometheus metrics
UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_ENABLED=true
UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_PORT=9467
UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_INTERFACE=127.0.0.1      # Localhost only
```

### Configuration Validation

The service performs comprehensive validation on startup:

**Validation Logic**: [config.go:88-152](../../internal/config/config.go:88)

Common validation errors:
- Invalid port ranges (1-65535)
- Storage backend misconfiguration
- Size limit inconsistencies (cache < max asset)
- Missing builderd endpoint when enabled
- Invalid TLS mode specification

## Monitoring & Metrics

### Health Checks

Multiple health check endpoints for different monitoring needs:

```bash
# Service health (includes uptime and version)
curl http://localhost:9467/health

# Basic connectivity test
curl https://localhost:8083/health.v1.HealthService/Check
```

**Health Implementation**: [main.go:262](../../cmd/assetmanagerd/main.go:262)

### Prometheus Metrics

Comprehensive metrics exported on `:9467/metrics`:

#### Request Metrics
- `http_request_duration_seconds` - Request duration histogram
- `grpc_server_handling_seconds` - gRPC handling time
- `assetmanagerd_requests_total` - Total request counter
- `assetmanagerd_request_errors_total` - Error counter by type

#### Asset Metrics
- `assetmanagerd_assets_total` - Total assets by type/status
- `assetmanagerd_asset_size_bytes` - Asset size distribution
- `assetmanagerd_storage_usage_bytes` - Storage utilization
- `assetmanagerd_cache_usage_bytes` - Cache utilization

#### Garbage Collection Metrics
- `assetmanagerd_gc_runs_total` - GC execution counter
- `assetmanagerd_gc_duration_seconds` - GC execution time
- `assetmanagerd_gc_assets_deleted_total` - Assets deleted counter
- `assetmanagerd_gc_bytes_freed_total` - Storage freed counter

#### Builderd Integration Metrics
- `assetmanagerd_builds_triggered_total` - Auto-builds triggered
- `assetmanagerd_build_duration_seconds` - Build completion time
- `assetmanagerd_build_failures_total` - Build failure counter

### OpenTelemetry Tracing

Distributed tracing with OTLP export:

**Trace Configuration**: [otel.go](../../internal/observability/otel.go)

#### Key Trace Operations
- `assetmanagerd.service.register_asset` - Asset registration
- `assetmanagerd.service.trigger_build` - Builderd integration
- `assetmanagerd.service.wait_for_build` - Build monitoring
- `assetmanagerd.storage.store` - Storage operations
- `assetmanagerd.registry.create_asset` - Database operations

#### Trace Attributes
- `asset.id`, `asset.type`, `asset.size_bytes`
- `docker.image`, `build.id`, `tenant.id`
- `storage.backend`, `storage.location`

### Logging

Structured JSON logging with contextual information:

**Log Configuration**: [main.go:76-80](../../cmd/assetmanagerd/main.go:76)

#### Log Levels
- `ERROR`: Service failures, build errors, storage issues
- `WARN`: Recoverable errors, expired leases, missing metadata
- `INFO`: Operations, GC runs, build completions
- `DEBUG`: Detailed request tracing (disabled in production)

#### Key Log Fields
```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "registered asset",
  "component": "service",
  "asset_id": "01HN123456789ABCDEF",
  "name": "nginx-rootfs",
  "type": "ASSET_TYPE_ROOTFS",
  "size_bytes": 52428800,
  "build_id": "01HN987654321FEDCBA"
}
```

## Storage Management

### Local Storage Organization

Efficient filesystem organization for performance:

**Storage Implementation**: [local.go](../../internal/storage/local.go)

```
/opt/vm-assets/
├── 01/                    # First 2 chars of asset ID
│   ├── 01HN123456789...   # Asset file
│   └── 01HN234567890...   # Another asset
├── 02/
│   └── 02HN345678901...
└── cache/                 # Downloaded remote assets
    ├── downloads/
    └── temp/
```

#### Sharding Benefits
- Prevents directory scan performance issues
- Enables parallel filesystem operations
- Improves backup and maintenance efficiency

**Sharding Logic**: [local.go:37-43](../../internal/storage/local.go:37)

### Database Management

SQLite database with WAL mode for performance:

**Database Configuration**: [registry.go:31-39](../../internal/registry/registry.go:31)

```bash
# Database location
/opt/assetmanagerd/assets.db

# Associated files
/opt/assetmanagerd/assets.db-wal    # Write-ahead log
/opt/assetmanagerd/assets.db-shm    # Shared memory
```

#### Backup Strategy

```bash
# Online backup (safe during operation)
sqlite3 /opt/assetmanagerd/assets.db ".backup /backup/assets-$(date +%Y%m%d).db"

# Vacuum for maintenance (offline only)
sudo systemctl stop assetmanagerd
sqlite3 /opt/assetmanagerd/assets.db "VACUUM;"
sudo systemctl start assetmanagerd
```

#### Database Monitoring

```bash
# Check database size and table stats
sqlite3 /opt/assetmanagerd/assets.db "
SELECT 
    name,
    COUNT(*) as rows
FROM sqlite_master 
LEFT JOIN pragma_table_info(name) ON name != 'sqlite_sequence'
GROUP BY name;
"

# Check asset counts by type
sqlite3 /opt/assetmanagerd/assets.db "
SELECT type, COUNT(*) FROM assets GROUP BY type;
"
```

### Storage Cleanup

Manual cleanup operations for maintenance:

```bash
# Force garbage collection
curl -X POST https://localhost:8083/asset.v1.AssetManagerService/GarbageCollect \
  -H "Content-Type: application/json" \
  -d '{"delete_unreferenced": true, "max_age_seconds": 604800}'

# Dry run to see what would be deleted
curl -X POST https://localhost:8083/asset.v1.AssetManagerService/GarbageCollect \
  -H "Content-Type: application/json" \
  -d '{"delete_unreferenced": true, "dry_run": true}'
```

## Performance Tuning

### Database Optimization

SQLite performance tuning:

```sql
-- Current settings (applied automatically)
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 10000;
PRAGMA temp_store = memory;
```

**Connection Pool Settings**: [registry.go:36-39](../../internal/registry/registry.go:36)

### Storage Performance

Filesystem optimization:

```bash
# For ext4 filesystems
sudo tune2fs -o journal_data_writeback /dev/sda1

# For XFS filesystems  
sudo mount -o noatime,nodiratime /dev/sda1 /opt/vm-assets
```

### Network Optimization

gRPC and HTTP/2 tuning:

```bash
# Increase connection limits
echo 'net.core.somaxconn = 1024' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 1024' >> /etc/sysctl.conf
sysctl -p
```

### Memory Management

Go runtime optimization:

```bash
# Set garbage collection target
export GOGC=75

# Limit max goroutines for concurrency control
export GOMAXPROCS=4
```

## Troubleshooting

### Common Issues

#### Service Won't Start

**Symptom**: Service fails to start or exits immediately

**Diagnosis**:
```bash
# Check systemd status
sudo systemctl status assetmanagerd

# View detailed logs
sudo journalctl -u assetmanagerd --no-pager

# Common causes and solutions:
```

1. **Configuration Error**: Check environment variables and validation
2. **SPIFFE Socket Missing**: Ensure spire-agent is running
3. **Database Permissions**: Fix `/opt/assetmanagerd` ownership
4. **Port Conflicts**: Check if port 8083 is already in use

#### Build Integration Failures

**Symptom**: Automatic builds not triggering or failing

**Diagnosis**:
```bash
# Check builderd connectivity
curl -k https://localhost:8082/health

# Verify TLS certificates
openssl s_client -connect localhost:8082 -servername builderd

# Check logs for build errors
sudo journalctl -u assetmanagerd | grep -i build
```

**Common Solutions**:
1. Verify `UNKEY_ASSETMANAGERD_BUILDERD_ENDPOINT` configuration
2. Check SPIFFE/SPIRE service identity registration
3. Increase `UNKEY_ASSETMANAGERD_BUILDERD_TIMEOUT` for slow builds
4. Verify tenant authentication headers

#### Storage Issues

**Symptom**: Asset uploads failing or storage errors

**Diagnosis**:
```bash
# Check disk space
df -h /opt/vm-assets

# Check permissions
ls -la /opt/vm-assets

# Check for corrupted assets
sqlite3 /opt/assetmanagerd/assets.db "
SELECT id, name, location FROM assets 
WHERE status = 4;  -- ERROR status
"
```

#### High Memory Usage

**Symptom**: Excessive memory consumption

**Diagnosis**:
```bash
# Check memory usage
sudo systemctl show assetmanagerd --property=MemoryCurrent

# Profile Go application
go tool pprof http://localhost:9467/debug/pprof/heap
```

**Solutions**:
1. Reduce `UNKEY_ASSETMANAGERD_DOWNLOAD_CONCURRENCY`
2. Lower `UNKEY_ASSETMANAGERD_MAX_CACHE_SIZE`
3. Enable more aggressive garbage collection
4. Check for memory leaks in streaming operations

### Debugging Tools

#### Asset Registry Inspection

```bash
# List all assets
sqlite3 /opt/assetmanagerd/assets.db "
SELECT id, name, type, status, size_bytes, reference_count 
FROM assets 
ORDER BY created_at DESC 
LIMIT 10;
"

# Check active leases
sqlite3 /opt/assetmanagerd/assets.db "
SELECT l.id, l.asset_id, l.acquired_by, 
       datetime(l.acquired_at, 'unixepoch') as acquired,
       datetime(l.expires_at, 'unixepoch') as expires
FROM asset_leases l
JOIN assets a ON l.asset_id = a.id;
"

# Find assets by label
sqlite3 /opt/assetmanagerd/assets.db "
SELECT a.id, a.name, al.key, al.value
FROM assets a
JOIN asset_labels al ON a.id = al.asset_id
WHERE al.key = 'docker_image' AND al.value LIKE '%nginx%';
"
```

#### Network Debugging

```bash
# Test gRPC connectivity
grpcurl -insecure localhost:8083 list
grpcurl -insecure localhost:8083 list asset.v1.AssetManagerService

# Test specific RPC
grpcurl -insecure -d '{"type": 1}' localhost:8083 asset.v1.AssetManagerService/ListAssets
```

#### Log Analysis

```bash
# Find recent errors
sudo journalctl -u assetmanagerd --since "1 hour ago" | grep ERROR

# Monitor real-time operations
sudo journalctl -fu assetmanagerd | jq 'select(.level == "INFO")'

# Analyze build patterns
sudo journalctl -u assetmanagerd | grep "build" | jq '.msg, .build_id, .docker_image'
```

### Performance Monitoring

#### Key Performance Indicators

1. **Request Latency**: 95th percentile < 500ms for GetAsset
2. **Build Success Rate**: > 95% auto-build completion
3. **Storage Utilization**: < 80% of max cache size
4. **GC Effectiveness**: Regular cleanup without resource exhaustion

#### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
- name: assetmanagerd
  rules:
  - alert: AssetmanagerdDown
    expr: up{job="assetmanagerd"} == 0
    for: 1m
    labels:
      severity: critical
      
  - alert: HighErrorRate
    expr: rate(assetmanagerd_request_errors_total[5m]) > 0.1
    for: 2m
    labels:
      severity: warning
      
  - alert: StorageNearFull
    expr: assetmanagerd_storage_usage_bytes / assetmanagerd_max_cache_size > 0.9
    for: 5m
    labels:
      severity: warning
      
  - alert: BuildFailures
    expr: rate(assetmanagerd_build_failures_total[10m]) > 0.2
    for: 3m
    labels:
      severity: warning
```

## Security Operations

### Certificate Management

SPIFFE/SPIRE automatic certificate rotation:

```bash
# Check current certificate
spire-agent api fetch x509 -socketPath /var/lib/spire/agent/agent.sock

# Verify service identity
spire-server entry show -spiffeID spiffe://unkey.dev/assetmanagerd
```

### Access Control

Service-level access control via SPIFFE IDs:

```bash
# Register assetmanagerd service identity
spire-server entry create \
  -spiffeID spiffe://unkey.dev/assetmanagerd \
  -parentID spiffe://unkey.dev/node \
  -selector unix:uid:1000
```

### Audit Logging

Comprehensive audit trail in structured logs:

```bash
# Asset access audit
sudo journalctl -u assetmanagerd | jq 'select(.msg | contains("asset")) | {time, msg, asset_id, user}'

# Build trigger audit  
sudo journalctl -u assetmanagerd | jq 'select(.msg | contains("build")) | {time, msg, build_id, docker_image, tenant_id}'
```