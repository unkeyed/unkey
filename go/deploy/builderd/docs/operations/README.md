# builderd Operations Guide

This guide covers deployment, monitoring, and operational aspects of builderd.

## Deployment

### System Requirements

- **Operating System**: Linux (kernel 4.18+)
- **CPU**: 2-4 cores minimum (scales with concurrent builds)
- **Memory**: 4-8GB RAM (depends on build workloads)
- **Disk**: 500-1000GB for workspace and artifacts
- **Network**: Outbound HTTPS for registry access

### Installation

#### Using Systemd

1. Build the binary:
```bash
cd builderd
make install  # Builds and installs with systemd unit
```

2. Configure environment:
```bash
# Edit systemd environment file
sudo vim /etc/systemd/system/builderd.service.d/environment.conf

# Add configuration
[Service]
Environment="UNKEY_BUILDERD_PORT=8082"
Environment="UNKEY_BUILDERD_STORAGE_BACKEND=s3"
Environment="UNKEY_BUILDERD_OTEL_ENABLED=true"
# ... additional configuration
```

3. Start the service:
```bash
sudo systemctl daemon-reload
sudo systemctl enable builderd
sudo systemctl start builderd
```

#### Using Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /build
COPY . .
RUN go mod download
RUN go build -o builderd ./cmd/builderd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /build/builderd /usr/local/bin/
EXPOSE 8082 9466
CMD ["builderd"]
```

#### Using Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: builderd
  namespace: unkey-deploy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: builderd
  template:
    metadata:
      labels:
        app: builderd
    spec:
      serviceAccountName: builderd
      containers:
      - name: builderd
        image: ghcr.io/unkeyed/builderd:0.1.0
        ports:
        - containerPort: 8082
          name: grpc
        - containerPort: 9466
          name: metrics
        env:
        - name: UNKEY_BUILDERD_PORT
          value: "8082"
        - name: UNKEY_BUILDERD_STORAGE_BACKEND
          value: "s3"
        - name: UNKEY_BUILDERD_OTEL_ENABLED
          value: "true"
        resources:
          requests:
            cpu: 2
            memory: 4Gi
          limits:
            cpu: 4
            memory: 8Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8082
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8082
          initialDelaySeconds: 5
          periodSeconds: 10
```

### Configuration Management

#### Environment Variables

See [main README](../../README.md#configuration) for complete list.

#### Storage Configuration

**Local Storage**:
```bash
UNKEY_BUILDERD_STORAGE_BACKEND=local
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/var/lib/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/var/lib/builderd/workspace
```

**S3 Storage**:
```bash
UNKEY_BUILDERD_STORAGE_BACKEND=s3
UNKEY_BUILDERD_STORAGE_S3_BUCKET=unkey-builderd-artifacts
UNKEY_BUILDERD_STORAGE_S3_REGION=us-east-1
UNKEY_BUILDERD_STORAGE_S3_ACCESS_KEY=AKIAXXXXXXXXXXXXXXXX
UNKEY_BUILDERD_STORAGE_S3_SECRET_KEY=XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
```

**GCS Storage**:
```bash
UNKEY_BUILDERD_STORAGE_BACKEND=gcs
UNKEY_BUILDERD_STORAGE_GCS_BUCKET=unkey-builderd-artifacts
UNKEY_BUILDERD_STORAGE_GCS_PROJECT=unkey-deploy
UNKEY_BUILDERD_STORAGE_GCS_CREDENTIALS_PATH=/etc/builderd/gcs-key.json
```

#### TLS/SPIFFE Configuration

**SPIFFE Mode (Recommended)**:
```bash
UNKEY_BUILDERD_TLS_MODE=spiffe
UNKEY_BUILDERD_SPIFFE_SOCKET=/run/spire/sockets/agent.sock
```

**File-based TLS**:
```bash
UNKEY_BUILDERD_TLS_MODE=file
UNKEY_BUILDERD_TLS_CERT_FILE=/etc/builderd/tls/server.crt
UNKEY_BUILDERD_TLS_KEY_FILE=/etc/builderd/tls/server.key
UNKEY_BUILDERD_TLS_CA_FILE=/etc/builderd/tls/ca.crt
```

## Monitoring

### Health Checks

builderd exposes a health endpoint at `/health`:

```bash
curl http://localhost:8082/health
```

Response:
```json
{
  "status": "healthy",
  "version": "0.1.0",
  "uptime_seconds": 3600,
  "build_info": {
    "go_version": "go1.21.0",
    "commit": "abc123",
    "build_time": "2024-01-15T10:00:00Z"
  }
}
```

### Metrics

When OpenTelemetry is enabled, Prometheus metrics are available at `http://localhost:9466/metrics`.

#### Key Metrics

**Build Metrics**:
- `builderd_builds_total{tenant,source,target,state}` - Total builds
- `builderd_build_duration_seconds{tenant,source,target}` - Build duration histogram
- `builderd_build_size_bytes{tenant,type}` - Artifact sizes
- `builderd_concurrent_builds` - Current active builds

**Resource Metrics**:
- `builderd_resource_cpu_seconds{tenant,build_id}` - CPU usage
- `builderd_resource_memory_bytes{tenant,build_id}` - Memory usage
- `builderd_resource_disk_bytes{tenant,build_id}` - Disk usage

**Tenant Metrics**:
- `builderd_tenant_quota_usage{tenant,resource}` - Quota utilization
- `builderd_tenant_builds_queued{tenant}` - Queued builds
- `builderd_tenant_cost_estimate{tenant,resource}` - Cost tracking

#### Grafana Dashboard

Import the dashboard from [`contrib/grafana-dashboards/builderd.json`](../../contrib/grafana-dashboards/builderd.json):

```bash
# Using Grafana API
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @contrib/grafana-dashboards/builderd.json
```

Dashboard includes:
- Build success rates and latencies
- Resource utilization trends
- Tenant quota usage
- Error rates and types
- Cache hit rates

### Logging

builderd uses structured JSON logging:

```json
{
  "time": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "msg": "build completed successfully",
  "service": "builderd",
  "build_id": "build-abc123",
  "tenant_id": "tenant-123",
  "duration_ms": 45000,
  "source_type": "docker_image",
  "rootfs_size_bytes": 524288000
}
```

#### Log Levels

- `DEBUG`: Detailed execution information
- `INFO`: Normal operational events
- `WARN`: Warning conditions
- `ERROR`: Error conditions requiring attention

#### Log Aggregation

For production, aggregate logs using:
- **Elasticsearch/OpenSearch**: Full-text search and analysis
- **Loki**: Lightweight log aggregation
- **CloudWatch/Stackdriver**: Cloud-native solutions

Example Filebeat configuration:
```yaml
filebeat.inputs:
- type: container
  paths:
    - /var/log/containers/builderd-*.log
  processors:
  - decode_json_fields:
      fields: ["message"]
      target: ""
  - add_kubernetes_metadata:
      host: ${NODE_NAME}
      matchers:
      - logs_path:
          logs_path: "/var/log/containers/"

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "builderd-%{+yyyy.MM.dd}"
```

### Alerting

#### Critical Alerts

**High Error Rate**:
```yaml
alert: BuilderdHighErrorRate
expr: rate(builderd_builds_total{state="failed"}[5m]) > 0.1
for: 5m
labels:
  severity: critical
annotations:
  summary: "High build failure rate"
  description: "Build failure rate is {{ $value }} errors/sec"
```

**Quota Exhaustion**:
```yaml
alert: BuilderdTenantQuotaExhausted
expr: builderd_tenant_quota_usage{resource="daily_builds"} >= 0.9
for: 5m
labels:
  severity: warning
annotations:
  summary: "Tenant approaching quota limit"
  description: "Tenant {{ $labels.tenant }} at {{ $value }}% of quota"
```

**Storage Full**:
```yaml
alert: BuilderdStorageFull
expr: builderd_storage_usage_bytes / builderd_storage_limit_bytes > 0.9
for: 10m
labels:
  severity: critical
annotations:
  summary: "Storage approaching capacity"
  description: "Storage at {{ $value }}% capacity"
```

## Maintenance

### Backup and Recovery

#### Database Backup

**SQLite**:
```bash
# Online backup
sqlite3 /opt/builderd/data/builderd.db ".backup /backup/builderd-$(date +%Y%m%d).db"

# Restore
cp /backup/builderd-20240115.db /opt/builderd/data/builderd.db
```

**PostgreSQL**:
```bash
# Backup
pg_dump -h localhost -U builderd -d builderd > builderd-$(date +%Y%m%d).sql

# Restore
psql -h localhost -U builderd -d builderd < builderd-20240115.sql
```

#### Artifact Backup

**S3 Sync**:
```bash
# Backup to another bucket
aws s3 sync s3://unkey-builderd-artifacts s3://unkey-builderd-backup --delete

# Restore
aws s3 sync s3://unkey-builderd-backup s3://unkey-builderd-artifacts
```

### Cleanup and Maintenance

#### Automated Cleanup

builderd automatically cleans up:
- Expired artifacts based on retention policy
- Temporary build workspaces
- Failed build remnants

Manual cleanup if needed:
```bash
# Clean workspace
find /opt/builderd/workspace -type d -mtime +7 -exec rm -rf {} \;

# Clean old artifacts
find /opt/builderd/rootfs -type f -mtime +30 -delete
```

#### Database Maintenance

**SQLite**:
```bash
# Vacuum and analyze
sqlite3 /opt/builderd/data/builderd.db "VACUUM; ANALYZE;"
```

**PostgreSQL**:
```bash
# Vacuum and analyze
psql -h localhost -U builderd -d builderd -c "VACUUM ANALYZE;"
```

### Scaling Operations

#### Horizontal Scaling

1. **Stateless Design**: builderd is stateless, scale by adding instances
2. **Load Balancer**: Use any L4/L7 load balancer
3. **Shared Storage**: Ensure all instances access same storage backend
4. **Database**: Use PostgreSQL for multi-instance deployments

#### Vertical Scaling

Adjust resources based on workload:
```bash
# Update systemd limits
sudo systemctl set-property builderd CPUQuota=400%
sudo systemctl set-property builderd MemoryLimit=16G
```

### Troubleshooting

#### Common Issues

**Build Failures**:
```bash
# Check logs for specific build
journalctl -u builderd --since "1 hour ago" | grep "build-abc123"

# Check disk space
df -h /opt/builderd/workspace

# Check resource limits
systemctl show builderd | grep -E "(CPU|Memory|Tasks)"
```

**Registry Access**:
```bash
# Test registry connectivity
curl -v https://ghcr.io/v2/

# Check auth configuration
echo $UNKEY_BUILDERD_DOCKER_REGISTRY_AUTH
```

**Storage Issues**:
```bash
# Test S3 access
aws s3 ls s3://unkey-builderd-artifacts/

# Check permissions
aws iam get-role-policy --role-name builderd-role --policy-name s3-access
```

#### Debug Mode

Enable debug logging:
```bash
UNKEY_BUILDERD_LOG_LEVEL=debug builderd
```

#### Performance Tuning

**Concurrent Builds**:
```bash
# Increase based on CPU cores
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=10
```

**Cache Tuning**:
```bash
# Increase cache size for better performance
UNKEY_BUILDERD_STORAGE_CACHE_MAX_SIZE_GB=100
```

**Network Optimization**:
```bash
# Use registry mirror for faster pulls
UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR=https://mirror.gcr.io
```

## Security Operations

### Access Control

1. **Service Account**: Run builderd with minimal privileges
2. **File Permissions**: Restrict access to workspace and storage
3. **Network Policies**: Limit egress to required endpoints

### Audit Logging

All operations are logged with tenant context:
```bash
# Audit trail for tenant
journalctl -u builderd -o json | jq 'select(.tenant_id == "tenant-123")'
```

### Incident Response

1. **Isolate**: Stop accepting new builds
2. **Investigate**: Check logs and metrics
3. **Remediate**: Apply fixes or rollback
4. **Document**: Update runbooks

AIDEV-NOTE: Regular monitoring and maintenance ensures optimal performance and reliability of the build service.
