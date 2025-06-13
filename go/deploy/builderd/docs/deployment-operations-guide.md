# Builderd Deployment & Operations Guide

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Installation Methods](#installation-methods)
3. [Configuration](#configuration)
4. [Production Deployment](#production-deployment)
5. [Operations & Maintenance](#operations--maintenance)
6. [Monitoring & Alerting](#monitoring--alerting)
7. [Backup & Recovery](#backup--recovery)
8. [Scaling](#scaling)
9. [Security](#security)
10. [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements

| Component | Requirement | Notes |
|-----------|-------------|-------|
| **OS** | Linux kernel 5.4+ | Required for cgroups v2 |
| **CPU** | 2 cores | Additional cores per concurrent build |
| **Memory** | 4GB RAM | Base + (2GB × concurrent builds) |
| **Disk** | 50GB SSD | Fast storage for Docker layers |
| **Docker** | 20.10+ | Docker Engine with containerd |

### Recommended Production

| Component | Requirement | Notes |
|-----------|-------------|-------|
| **OS** | Ubuntu 22.04 LTS / RHEL 9 | Tested distributions |
| **CPU** | 8+ cores | For multi-tenant workloads |
| **Memory** | 16+ GB RAM | Buffer for tenant isolation |
| **Disk** | 500GB+ NVMe SSD | High IOPS for Docker operations |
| **Network** | 1Gbps+ | For registry pulls |

### Storage Requirements

```bash
# Disk usage planning
/var/lib/builderd/
├── rootfs/          # 10-100MB per build artifact
├── cache/           # 1-5GB per tenant (Docker layers)
├── workspace/       # 1-10GB per active build
├── scratch/         # 1-5GB temporary space
└── logs/           # 100MB-1GB logs
```

### Network Requirements

| Service | Port | Protocol | Purpose |
|---------|------|----------|---------|
| Builderd API | 8082 | HTTP/2 | Main API endpoint |
| Metrics | 9090 | HTTP | Prometheus metrics |
| Health Check | 8082/health | HTTP | Load balancer health |

**Outbound Access**:
- Docker registries (docker.io, ghcr.io, etc.)
- Git repositories (github.com, gitlab.com, etc.)
- OpenTelemetry collector endpoints

## Installation Methods

### Method 1: Systemd Service (Recommended)

```bash
# 1. Create builderd user
sudo useradd --system --home /var/lib/builderd --shell /bin/false builderd
sudo usermod -aG docker builderd

# 2. Create directories
sudo mkdir -p /var/lib/builderd/{data,scratch,rootfs,workspace,cache,logs}
sudo mkdir -p /etc/builderd
sudo chown -R builderd:builderd /var/lib/builderd

# 3. Install binary
sudo cp build/builderd /usr/local/bin/
sudo chmod +x /usr/local/bin/builderd

# 4. Install systemd service
sudo cp contrib/systemd/builderd.service /etc/systemd/system/
sudo cp contrib/systemd/builderd.env.example /etc/builderd/builderd.env

# 5. Configure service
sudo systemctl daemon-reload
sudo systemctl enable builderd
sudo systemctl start builderd
```

### Method 2: Docker Container

```yaml
# docker-compose.yml
version: '3.8'
services:
  builderd:
    image: builderd:latest
    ports:
      - "8082:8082"
      - "9090:9090"
    environment:
      UNKEY_BUILDERD_HOST: "0.0.0.0"
      UNKEY_BUILDERD_DATABASE_TYPE: "postgres"
      UNKEY_BUILDERD_DATABASE_URL: "postgres://builderd:password@postgres:5432/builderd"
    volumes:
      - builderd_data:/var/lib/builderd
      - /var/run/docker.sock:/var/run/docker.sock
    restart: unless-stopped
    depends_on:
      - postgres

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: builderd
      POSTGRES_USER: builderd
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

volumes:
  builderd_data:
  postgres_data:
```

### Method 3: Kubernetes Deployment

```yaml
# kubernetes/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: builderd

---
# kubernetes/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: builderd-config
  namespace: builderd
data:
  UNKEY_BUILDERD_HOST: "0.0.0.0"
  UNKEY_BUILDERD_PORT: "8082"
  UNKEY_BUILDERD_LOG_LEVEL: "info"
  UNKEY_BUILDERD_DATABASE_TYPE: "postgres"
  UNKEY_BUILDERD_OTEL_ENABLED: "true"

---
# kubernetes/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: builderd-secrets
  namespace: builderd
type: Opaque
stringData:
  UNKEY_BUILDERD_DATABASE_URL: "postgres://builderd:password@postgres:5432/builderd"

---
# kubernetes/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: builderd
  namespace: builderd
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
        image: builderd:latest
        ports:
        - containerPort: 8082
        - containerPort: 9090
        envFrom:
        - configMapRef:
            name: builderd-config
        - secretRef:
            name: builderd-secrets
        volumeMounts:
        - name: builderd-data
          mountPath: /var/lib/builderd
        - name: docker-socket
          mountPath: /var/run/docker.sock
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "4Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /health
            port: 8082
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8082
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: builderd-data
        persistentVolumeClaim:
          claimName: builderd-data
      - name: docker-socket
        hostPath:
          path: /var/run/docker.sock

---
# kubernetes/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: builderd
  namespace: builderd
spec:
  selector:
    app: builderd
  ports:
  - name: api
    port: 8082
    targetPort: 8082
  - name: metrics
    port: 9090
    targetPort: 9090
```

## Configuration

### Production Environment File

```bash
# /etc/builderd/builderd.env

# Server Configuration
UNKEY_BUILDERD_HOST=0.0.0.0
UNKEY_BUILDERD_PORT=8082
UNKEY_BUILDERD_LOG_LEVEL=warn
UNKEY_BUILDERD_LOG_FORMAT=json

# Storage Configuration
UNKEY_BUILDERD_SCRATCH_DIR=/var/lib/builderd/scratch
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/var/lib/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/var/lib/builderd/workspace
UNKEY_BUILDERD_CACHE_DIR=/var/lib/builderd/cache

# Database Configuration
UNKEY_BUILDERD_DATABASE_TYPE=postgres
UNKEY_BUILDERD_DATABASE_URL=postgres://builderd:secure_password@postgres.example.com:5432/builderd?sslmode=require

# Security Configuration
UNKEY_BUILDERD_ENABLE_ISOLATION=true
UNKEY_BUILDERD_ENABLE_ENCRYPTION=true
UNKEY_BUILDERD_BUILD_USER_ID=1000
UNKEY_BUILDERD_BUILD_GROUP_ID=1000

# OpenTelemetry Configuration
UNKEY_BUILDERD_OTEL_ENABLED=true
UNKEY_BUILDERD_OTEL_ENDPOINT=https://otel-collector.example.com:4317
UNKEY_BUILDERD_OTEL_INSECURE=false

# Docker Configuration
UNKEY_BUILDERD_DOCKER_PULL_POLICY=always
UNKEY_BUILDERD_DOCKER_TIMEOUT=600s

# Performance Configuration
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=20
UNKEY_BUILDERD_BUILD_TIMEOUT=1800s
```

### Database Setup

#### PostgreSQL Setup

```sql
-- Create database and user
CREATE USER builderd WITH PASSWORD 'secure_password';
CREATE DATABASE builderd OWNER builderd;
GRANT ALL PRIVILEGES ON DATABASE builderd TO builderd;

-- Connect to builderd database
\c builderd

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Grant schema permissions
GRANT ALL ON SCHEMA public TO builderd;
```

#### Database Migration

```bash
# Run database migrations (if applicable)
/usr/local/bin/builderd migrate --config /etc/builderd/builderd.env

# Verify database schema
/usr/local/bin/builderd validate-db --config /etc/builderd/builderd.env
```

## Production Deployment

### Pre-Deployment Checklist

- [ ] **System Requirements**: Verify OS, Docker, and hardware requirements
- [ ] **User Account**: Create dedicated `builderd` user with Docker access
- [ ] **Storage**: Configure persistent storage with appropriate permissions
- [ ] **Database**: Set up PostgreSQL with proper credentials and networking
- [ ] **Network**: Configure firewall rules and load balancer
- [ ] **Monitoring**: Set up Prometheus, Grafana, and alerting
- [ ] **Security**: Review isolation settings and access controls
- [ ] **Backup**: Configure automated backup procedures

### Load Balancer Configuration

#### Nginx Configuration

```nginx
upstream builderd {
    server builderd-1.example.com:8082;
    server builderd-2.example.com:8082;
    server builderd-3.example.com:8082;
}

server {
    listen 443 ssl http2;
    server_name builderd.example.com;

    ssl_certificate /etc/ssl/certs/builderd.pem;
    ssl_certificate_key /etc/ssl/private/builderd.key;

    location / {
        proxy_pass http://builderd;
        proxy_http_version 2.0;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # For streaming builds
        proxy_buffering off;
        proxy_cache off;
    }

    location /health {
        access_log off;
        proxy_pass http://builderd;
    }
}
```

#### HAProxy Configuration

```
global
    daemon
    log stdout local0

defaults
    mode http
    timeout connect 5000ms
    timeout client 50000ms
    timeout server 50000ms

frontend builderd_frontend
    bind *:443 ssl crt /etc/ssl/builderd.pem
    redirect scheme https if !{ ssl_fc }
    default_backend builderd_backend

backend builderd_backend
    balance roundrobin
    option httpchk GET /health
    server builderd-1 builderd-1.example.com:8082 check
    server builderd-2 builderd-2.example.com:8082 check
    server builderd-3 builderd-3.example.com:8082 check
```

### SSL/TLS Configuration

```bash
# Generate SSL certificate (production should use CA-signed certs)
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout /etc/ssl/private/builderd.key \
    -out /etc/ssl/certs/builderd.pem \
    -subj "/C=US/ST=State/L=City/O=Organization/CN=builderd.example.com"

# Set proper permissions
chmod 600 /etc/ssl/private/builderd.key
chmod 644 /etc/ssl/certs/builderd.pem
```

## Operations & Maintenance

### Service Management

```bash
# Start/Stop/Restart service
sudo systemctl start builderd
sudo systemctl stop builderd
sudo systemctl restart builderd

# Check service status
sudo systemctl status builderd

# View logs
journalctl -u builderd -f
journalctl -u builderd --since "1 hour ago"

# Reload configuration
sudo systemctl reload builderd
```

### Log Management

```bash
# Configure log rotation
sudo tee /etc/logrotate.d/builderd << EOF
/var/log/builderd/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        systemctl reload builderd
    endscript
}
EOF

# Manual log rotation
sudo logrotate -f /etc/logrotate.d/builderd
```

### Storage Management

```bash
# Monitor disk usage
df -h /var/lib/builderd
du -sh /var/lib/builderd/*

# Clean up old builds (configure retention policy)
find /var/lib/builderd/rootfs -type d -mtime +30 -exec rm -rf {} \;
find /var/lib/builderd/cache -type f -mtime +7 -delete

# Monitor Docker disk usage
docker system df
docker system prune -f --volumes
```

### Database Maintenance

```sql
-- Vacuum and analyze database
VACUUM ANALYZE;

-- Update table statistics
ANALYZE;

-- Check database size
SELECT pg_size_pretty(pg_database_size('builderd'));

-- Monitor active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'builderd';
```

## Monitoring & Alerting

### Prometheus Metrics

Key metrics exposed at `/metrics`:

```
# Build metrics
builderd_builds_total{status="completed|failed|cancelled"}
builderd_builds_duration_seconds
builderd_builds_queue_size

# Resource metrics
builderd_memory_usage_bytes
builderd_cpu_usage_percent
builderd_disk_usage_bytes

# Tenant metrics
builderd_tenant_builds_active{tenant_id}
builderd_tenant_quota_usage{tenant_id,quota_type}

# Docker metrics
builderd_docker_pulls_total{registry}
builderd_docker_pull_duration_seconds
```

### Grafana Dashboard

```json
{
  "dashboard": {
    "title": "Builderd Operations",
    "panels": [
      {
        "title": "Build Success Rate",
        "type": "stat",
        "targets": [
          {
            "expr": "rate(builderd_builds_total{status=\"completed\"}[5m]) / rate(builderd_builds_total[5m]) * 100",
            "legendFormat": "Success Rate %"
          }
        ]
      },
      {
        "title": "Active Builds",
        "type": "graph",
        "targets": [
          {
            "expr": "builderd_builds_queue_size",
            "legendFormat": "Queued"
          },
          {
            "expr": "sum(builderd_tenant_builds_active)",
            "legendFormat": "Running"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
# prometheus/alerts.yml
groups:
- name: builderd
  rules:
  - alert: BuilderdServiceDown
    expr: up{job="builderd"} == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Builderd service is down"

  - alert: BuilderdHighFailureRate
    expr: rate(builderd_builds_total{status="failed"}[5m]) > 0.1
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High build failure rate: {{ $value }}"

  - alert: BuilderdDiskSpaceLow
    expr: disk_free_bytes{mountpoint="/var/lib/builderd"} / disk_total_bytes * 100 < 10
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Builderd disk space low: {{ $value }}% remaining"

  - alert: BuilderdQuotaViolation
    expr: increase(builderd_quota_violations_total[5m]) > 0
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "Quota violation detected for tenant {{ $labels.tenant_id }}"
```

## Backup & Recovery

### Database Backup

```bash
#!/bin/bash
# /usr/local/bin/backup-builderd-db.sh

BACKUP_DIR="/var/backups/builderd"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/builderd_$DATE.sql.gz"

mkdir -p "$BACKUP_DIR"

# Create compressed backup
pg_dump -h postgres.example.com -U builderd builderd | gzip > "$BACKUP_FILE"

# Keep last 30 days of backups
find "$BACKUP_DIR" -name "builderd_*.sql.gz" -mtime +30 -delete

# Verify backup
if [ $? -eq 0 ]; then
    echo "Backup completed successfully: $BACKUP_FILE"
else
    echo "Backup failed!" >&2
    exit 1
fi
```

### Storage Backup

```bash
#!/bin/bash
# /usr/local/bin/backup-builderd-storage.sh

BACKUP_DIR="/var/backups/builderd-storage"
DATE=$(date +%Y%m%d_%H%M%S)
SOURCE_DIR="/var/lib/builderd"

mkdir -p "$BACKUP_DIR"

# Backup critical data (exclude scratch and temp)
tar -czf "$BACKUP_DIR/builderd-storage_$DATE.tar.gz" \
    --exclude="$SOURCE_DIR/scratch" \
    --exclude="$SOURCE_DIR/workspace" \
    "$SOURCE_DIR"

# Keep last 7 days of storage backups
find "$BACKUP_DIR" -name "builderd-storage_*.tar.gz" -mtime +7 -delete
```

### Recovery Procedures

```bash
# Database recovery
gunzip -c /var/backups/builderd/builderd_20240115_120000.sql.gz | \
    psql -h postgres.example.com -U builderd builderd

# Storage recovery
tar -xzf /var/backups/builderd-storage/builderd-storage_20240115_120000.tar.gz \
    -C /var/lib/

# Verify service after recovery
sudo systemctl restart builderd
curl -f http://localhost:8082/health
```

## Scaling

### Horizontal Scaling

#### Load Balancer Setup

```bash
# Add new builderd instance
# 1. Provision new server with same configuration
# 2. Add to load balancer pool
# 3. Update monitoring configuration

# Example with HAProxy
echo "server builderd-4 builderd-4.example.com:8082 check" >> /etc/haproxy/haproxy.cfg
systemctl reload haproxy
```

#### Database Connection Pooling

```bash
# Increase connection pool size for multiple instances
UNKEY_BUILDERD_DATABASE_URL="postgres://builderd:password@postgres:5432/builderd?pool_max_conns=100"
```

### Vertical Scaling

```bash
# Update systemd service resource limits
sudo tee /etc/systemd/system/builderd.service.d/override.conf << EOF
[Service]
LimitNOFILE=65536
LimitNPROC=32768
MemoryMax=16G
CPUQuota=800%
EOF

sudo systemctl daemon-reload
sudo systemctl restart builderd
```

### Auto-Scaling (Kubernetes)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: builderd-hpa
  namespace: builderd
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: builderd
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Security

### Network Security

```bash
# Configure firewall (UFW example)
sudo ufw allow 22/tcp
sudo ufw allow from 10.0.0.0/8 to any port 8082
sudo ufw allow from 10.0.0.0/8 to any port 9090
sudo ufw deny 8082
sudo ufw deny 9090
sudo ufw enable
```

### File System Security

```bash
# Set proper ownership and permissions
sudo chown -R builderd:builderd /var/lib/builderd
sudo chmod 750 /var/lib/builderd
sudo chmod 640 /etc/builderd/builderd.env

# SELinux contexts (if enabled)
sudo setsebool -P container_manage_cgroup on
sudo semanage fcontext -a -t container_file_t "/var/lib/builderd(/.*)?"
sudo restorecon -R /var/lib/builderd
```

### Container Security

```bash
# Docker daemon security
sudo tee /etc/docker/daemon.json << EOF
{
  "userns-remap": "default",
  "live-restore": true,
  "userland-proxy": false,
  "no-new-privileges": true,
  "seccomp-profile": "/etc/docker/seccomp.json"
}
EOF

sudo systemctl restart docker
```

## Troubleshooting

### Common Issues

#### Build Failures

```bash
# Check build logs
curl -H "X-Tenant-ID: tenant-123" \
    http://localhost:8082/builder.v1.BuilderService/GetBuildLogs/BUILD_ID

# Check Docker daemon
sudo systemctl status docker
docker info

# Check storage space
df -h /var/lib/builderd
```

#### Performance Issues

```bash
# Check resource usage
top -p $(pgrep builderd)
iostat -x 1 5
vmstat 1 5

# Check database performance
sudo -u postgres psql -d builderd -c "
SELECT query, mean_exec_time, calls 
FROM pg_stat_statements 
ORDER BY mean_exec_time DESC 
LIMIT 10;"
```

#### Network Issues

```bash
# Test Docker registry connectivity
docker pull alpine:latest

# Test database connectivity
pg_isready -h postgres.example.com -p 5432 -U builderd

# Check network policies
iptables -L -n | grep 8082
```

### Diagnostic Commands

```bash
# Service health check
curl -f http://localhost:8082/health

# Detailed service status
curl http://localhost:8082/debug/status

# Configuration validation
/usr/local/bin/builderd --validate-config

# Database connectivity test
/usr/local/bin/builderd --test-database
```

### Log Analysis

```bash
# Recent errors
journalctl -u builderd --since "1 hour ago" | grep ERROR

# Build failures
journalctl -u builderd | grep "build_status.*failed"

# Quota violations
journalctl -u builderd | grep "quota.*exceeded"

# Performance issues
journalctl -u builderd | grep "timeout\|slow\|high"
```

For additional troubleshooting and support, refer to the main [README](README.md) documentation.