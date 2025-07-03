# Builderd Operations Manual

This guide covers production deployment, monitoring, and operational management of the builderd service.

## Installation and Configuration

### System Requirements

**Minimum Requirements**:
- **OS**: Linux with Docker runtime support
- **CPU**: 4+ cores recommended for concurrent builds
- **Memory**: 8GB+ for running multiple builds simultaneously  
- **Storage**: SSD with 100GB+ free space for build artifacts
- **Docker**: Docker Engine 20.10+ for image extraction

### Installation

```bash
# Build from source
cd builderd
make build

# Install with systemd integration  
sudo make install

# Verify installation
systemctl status builderd
```

**Binary Location**: `/usr/local/bin/builderd`
**Systemd Unit**: `/etc/systemd/system/builderd.service`

**Systemd Configuration**: [contrib/systemd/builderd.service](../../contrib/systemd/builderd.service)

### Core Configuration

**Environment File**: `/etc/builderd/environment`

```bash
# Server Configuration
UNKEY_BUILDERD_PORT=8082
UNKEY_BUILDERD_ADDRESS=0.0.0.0
UNKEY_BUILDERD_SHUTDOWN_TIMEOUT=15s
UNKEY_BUILDERD_RATE_LIMIT=100

# Build Configuration  
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=5
UNKEY_BUILDERD_BUILD_TIMEOUT=15m
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/opt/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/opt/builderd/workspace
UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd

# Storage Configuration
UNKEY_BUILDERD_STORAGE_BACKEND=local
UNKEY_BUILDERD_STORAGE_RETENTION_DAYS=30
UNKEY_BUILDERD_STORAGE_MAX_SIZE_GB=100

# Docker Configuration
UNKEY_BUILDERD_DOCKER_MAX_IMAGE_SIZE_GB=5
UNKEY_BUILDERD_DOCKER_PULL_TIMEOUT=10m
UNKEY_BUILDERD_DOCKER_REGISTRY_AUTH=true

# Tenant Configuration
UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED=true
UNKEY_BUILDERD_TENANT_DEFAULT_TIER=free
```

**Configuration Reference**: [internal/config/config.go:147](../../internal/config/config.go#L147)

### Security Configuration

**SPIFFE/SPIRE Integration** (Required for Production):
```bash
# TLS Configuration
UNKEY_BUILDERD_TLS_MODE=spiffe
UNKEY_BUILDERD_SPIFFE_SOCKET=/var/lib/spire/agent/agent.sock

# Alternative: File-based TLS
UNKEY_BUILDERD_TLS_MODE=file
UNKEY_BUILDERD_TLS_CERT_FILE=/etc/builderd/certs/server.crt
UNKEY_BUILDERD_TLS_KEY_FILE=/etc/builderd/certs/server.key
UNKEY_BUILDERD_TLS_CA_FILE=/etc/builderd/certs/ca.crt
```

**SPIFFE Configuration**: [internal/config/config.go:224](../../internal/config/config.go#L224)

### Database Configuration

```bash
# SQLite (Recommended for single-instance)
UNKEY_BUILDERD_DATABASE_TYPE=sqlite
UNKEY_BUILDERD_DATABASE_DATA_DIR=/opt/builderd/data

# PostgreSQL (For multi-instance deployments)
UNKEY_BUILDERD_DATABASE_TYPE=postgres
UNKEY_BUILDERD_DATABASE_HOST=postgres.internal
UNKEY_BUILDERD_DATABASE_PORT=5432
UNKEY_BUILDERD_DATABASE_NAME=builderd
UNKEY_BUILDERD_DATABASE_USERNAME=builderd
UNKEY_BUILDERD_DATABASE_PASSWORD=<secure-password>
UNKEY_BUILDERD_DATABASE_SSL_MODE=require
```

### Assetmanagerd Integration

```bash
# Asset Management Configuration
UNKEY_BUILDERD_ASSETMANAGER_ENABLED=true
UNKEY_BUILDERD_ASSETMANAGER_ENDPOINT=https://assetmanagerd:8083
```

**Integration Details**: [internal/assetmanager/client.go:28](../../internal/assetmanager/client.go#L28)

## Monitoring and Observability

### OpenTelemetry Configuration

```bash
# OpenTelemetry Configuration
UNKEY_BUILDERD_OTEL_ENABLED=true
UNKEY_BUILDERD_OTEL_SERVICE_NAME=builderd
UNKEY_BUILDERD_OTEL_SERVICE_VERSION=0.2.0
UNKEY_BUILDERD_OTEL_SAMPLING_RATE=1.0
UNKEY_BUILDERD_OTEL_ENDPOINT=http://otel-collector:4318

# Prometheus Integration
UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED=true
UNKEY_BUILDERD_OTEL_PROMETHEUS_PORT=9466
UNKEY_BUILDERD_OTEL_PROMETHEUS_INTERFACE=127.0.0.1

# High-Cardinality Metrics (Use with caution)
UNKEY_BUILDERD_OTEL_HIGH_CARDINALITY_ENABLED=false
```

**Observability Implementation**: [internal/observability/otel.go](../../internal/observability/otel.go)

### Key Metrics to Monitor

**Build Metrics**:
- `builderd_builds_total` - Total builds started (by tenant, type, status)
- `builderd_build_duration_seconds` - Build completion time histogram
- `builderd_build_errors_total` - Build failure counter (by error type)
- `builderd_active_builds` - Currently running builds gauge
- `builderd_build_cancellations_total` - Build cancellation counter

**Pipeline Metrics**:
- `builderd_pull_duration_seconds` - Image/source pull time
- `builderd_extract_duration_seconds` - Filesystem extraction time  
- `builderd_optimize_duration_seconds` - Rootfs optimization time

**Resource Metrics**:
- `builderd_image_size_bytes` - Source image sizes
- `builderd_rootfs_size_bytes` - Generated rootfs sizes
- `builderd_compression_ratio` - Size reduction achieved
- `builderd_build_memory_usage_bytes` - Peak memory usage
- `builderd_build_disk_usage_bytes` - Peak disk usage

**Tenant Metrics** (High-cardinality):
- `builderd_tenant_builds_total` - Per-tenant build counters
- `builderd_tenant_quota_violations_total` - Quota violation events

**Metrics Implementation**: [internal/observability/metrics.go](../../internal/observability/metrics.go)

### Prometheus Configuration

**Scrape Configuration**:
```yaml
- job_name: 'builderd'
  static_configs:
    - targets: ['builderd:9466']
  scrape_interval: 30s
  metrics_path: /metrics
```

**Dashboard Queries**:
```promql
# Build success rate
rate(builderd_builds_total{status="success"}[5m]) / 
rate(builderd_builds_total[5m])

# Average build duration
rate(builderd_build_duration_seconds_sum[5m]) / 
rate(builderd_build_duration_seconds_count[5m])

# Active builds by tenant
builderd_active_builds

# Error rate by type
rate(builderd_build_errors_total[5m])
```

### Health Checks

**Health Endpoint**: `GET /health`

**Response Format**:
```json
{
  "status": "healthy",
  "version": "0.2.0",
  "uptime": "24h30m15s",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Health Check Implementation**: Uses unified health package with rate limiting.

**Service Validation**: [cmd/builderd/main.go:497](../../cmd/builderd/main.go#L497)

### Logging Configuration

**Structured Logging**: JSON format with contextual fields

**Log Levels**:
- `INFO` - Normal operation events
- `WARN` - Non-fatal issues and degraded performance
- `ERROR` - Build failures and service errors
- `DEBUG` - Detailed debugging information

**Key Log Fields**:
- `tenant_id` - Tenant identifier for all operations
- `build_id` - Build job identifier
- `source_type` - Build source type (docker, git, archive)
- `duration` - Operation timing

### Tracing

**Trace Hierarchy**:
```
builderd.build_execution
├── builderd.docker.pull_image
├── builderd.docker.create_container
├── builderd.docker.extract_filesystem
├── builderd.docker.optimize_rootfs
└── builderd.assetmanager.register_artifact
```

**Trace Attributes**:
- `tenant.id` - Tenant identifier
- `build.type` - Build type (docker, git, archive)
- `source.image` - Docker image URI
- `build.duration` - Total build time

## Performance Tuning

### Build Concurrency

**Concurrent Build Limits**:
```bash
# System-wide concurrent builds
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=10

# Per-tenant limits (configured per tier)
# Free: 3, Pro: 5, Enterprise: 10, Dedicated: 20
```

**Resource Considerations**:
- **Memory**: ~2GB per concurrent Docker build
- **CPU**: 1-2 cores per build for optimal performance
- **Disk I/O**: SSD recommended for workspace and output directories

### Docker Optimization

**Docker Configuration**:
```bash
# Image size limits
UNKEY_BUILDERD_DOCKER_MAX_IMAGE_SIZE_GB=5

# Pull timeout for large images
UNKEY_BUILDERD_DOCKER_PULL_TIMEOUT=10m

# Registry mirrors for faster pulls
UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR=https://mirror.gcr.io
```

### Storage Optimization

**Storage Backend Tuning**:
```bash
# Local storage configuration
UNKEY_BUILDERD_STORAGE_CACHE_ENABLED=true
UNKEY_BUILDERD_STORAGE_CACHE_MAX_SIZE_GB=50

# Retention policies
UNKEY_BUILDERD_STORAGE_RETENTION_DAYS=30
UNKEY_BUILDERD_CLEANUP_INTERVAL=1h
```

**Directory Structure Optimization**:
- Use separate partitions for workspace and output directories
- Configure appropriate filesystem (ext4 recommended)
- Enable noatime mount option for performance

### Network Optimization

**Registry Access**:
- Configure registry mirrors for commonly used images
- Use dedicated registry credentials for faster authentication
- Consider private registry for base images

## Security Operations

### Access Control

**SPIFFE Identity Validation**: All service communications require valid SPIFFE identities.

**Tenant Isolation**:
- Build workspaces isolated per tenant
- Resource quotas enforced at multiple levels
- Storage paths scoped by tenant identifier

### Security Monitoring

**Security Events to Monitor**:
- Authentication failures from invalid SPIFFE identities
- Quota violations and potential abuse
- Unusual build patterns or resource usage
- Failed registry authentication attempts

**Log Monitoring Queries**:
```bash
# Authentication failures
journalctl -u builderd | grep "authentication failed"

# Quota violations  
journalctl -u builderd | grep "quota exceeded"

# Build failures
journalctl -u builderd | grep "build execution failed"
```

### Vulnerability Management

**Container Security**:
- Regularly update Docker runtime and base images
- Monitor CVE databases for image vulnerabilities
- Implement image scanning in build pipeline (planned)

**Build Isolation**:
- Builds execute in temporary isolated directories
- No network access except to approved registries
- Resource limits prevent resource exhaustion attacks

## Backup and Recovery

### Data Backup

**Critical Data**:
- Build job database (SQLite/PostgreSQL)
- Configuration files (`/etc/builderd/`)
- Build artifacts (`/opt/builderd/rootfs/`)
- Base assets (`/opt/builderd/rootfs/base/`)

**Backup Strategy**:
```bash
# Database backup (SQLite)
sqlite3 /opt/builderd/data/builderd.db ".backup /backup/builderd-$(date +%Y%m%d).db"

# Configuration backup
tar -czf /backup/builderd-config-$(date +%Y%m%d).tar.gz /etc/builderd/

# Artifacts backup (selective)
rsync -av --exclude='workspace/' /opt/builderd/ /backup/builderd-data/
```

### Disaster Recovery

**Recovery Procedure**:
1. Restore configuration files to `/etc/builderd/`
2. Restore database from backup
3. Restore base assets to enable new builds
4. Restart builderd service
5. Verify health endpoints and basic functionality

**Recovery Testing**: Regularly test recovery procedures in non-production environments.

## Troubleshooting

### Common Issues

**Build Failures**:
```bash
# Check Docker daemon status
systemctl status docker

# Verify Docker access
docker info

# Check disk space
df -h /opt/builderd/

# Review build logs
journalctl -u builderd -f
```

**SPIFFE Authentication Issues**:
```bash
# Check SPIRE agent status
systemctl status spire-agent

# Verify socket permissions
ls -la /var/lib/spire/agent/agent.sock

# Test SPIFFE identity
spire-agent api fetch -socketPath /var/lib/spire/agent/agent.sock
```

**Performance Issues**:
```bash
# Check active builds
curl -s http://localhost:9466/metrics | grep builderd_active_builds

# Review resource usage
top -p $(pgrep builderd)

# Check Docker resource usage
docker stats
```

### Debug Configuration

**Debug Mode**:
```bash
# Enable debug logging
UNKEY_BUILDERD_LOG_LEVEL=debug

# Enable request tracing
UNKEY_BUILDERD_OTEL_SAMPLING_RATE=1.0
```

**Debug Endpoints**:
- `/health` - Service health with detailed status
- `/metrics` - Prometheus metrics for debugging
- OpenTelemetry traces via configured collector

### Log Analysis

**Important Log Patterns**:
```bash
# Build lifecycle events
grep "build execution" /var/log/builderd.log

# Tenant operations
grep "tenant_id.*build" /var/log/builderd.log

# Error patterns
grep "ERROR" /var/log/builderd.log | tail -20

# Performance metrics
grep "duration" /var/log/builderd.log
```

## Capacity Planning

### Resource Requirements

**Per-Build Resources**:
- **Memory**: 1-2GB per concurrent build
- **CPU**: 1-2 cores per build for optimal performance
- **Temporary Disk**: 2-5GB per build (varies by image size)
- **Network**: Registry pull bandwidth (varies by image size)

**System Scaling Guidelines**:
- **Small**: 5 concurrent builds, 4 cores, 16GB RAM
- **Medium**: 10 concurrent builds, 8 cores, 32GB RAM  
- **Large**: 20 concurrent builds, 16 cores, 64GB RAM

### Growth Planning

**Metrics to Track**:
- Build frequency and duration trends
- Storage usage growth patterns
- Resource utilization per tenant tier
- Peak usage patterns and seasonal variations

**Scaling Indicators**:
- Build queue times increasing
- Resource utilization > 80% sustained
- Frequent quota violations
- Storage approaching configured limits