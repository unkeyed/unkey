# Builderd Configuration Guide

## Environment Variables

All builderd configuration uses the `UNKEY_BUILDERD_` prefix for consistency with other Unkey services.

### Core Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_PORT` | `8082` | HTTP server port | Use reverse proxy in production |
| `UNKEY_BUILDERD_HOST` | `localhost` | Bind address | Set to `0.0.0.0` for containers |
| `UNKEY_BUILDERD_LOG_LEVEL` | `info` | Logging level (debug, info, warn, error) | Use `warn` in production |
| `UNKEY_BUILDERD_LOG_FORMAT` | `json` | Log format (json, text) | Always use `json` for production |

### Storage Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_SCRATCH_DIR` | `/tmp/builderd/scratch` | Temporary build workspace | Use fast SSD storage |
| `UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR` | `/var/lib/builderd/rootfs` | Rootfs output directory | Persistent storage required |
| `UNKEY_BUILDERD_WORKSPACE_DIR` | `/var/lib/builderd/workspace` | Build workspace directory | Fast storage recommended |
| `UNKEY_BUILDERD_CACHE_DIR` | `/var/lib/builderd/cache` | Docker layer cache | Large storage capacity needed |
| `UNKEY_BUILDERD_MAX_STORAGE_SIZE` | `100GB` | Maximum total storage usage | Adjust based on capacity |

### Database Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_DATABASE_TYPE` | `sqlite` | Database type (sqlite, postgres) | SQLite3 recommended for all environments |
| `UNKEY_BUILDERD_DATABASE_URL` | `file:/var/lib/builderd/builderd.db` | Database connection string | Include connection pooling |
| `UNKEY_BUILDERD_DATABASE_DATA_DIR` | `/var/lib/builderd/data` | SQLite data directory | Use persistent storage |
| `UNKEY_BUILDERD_DATABASE_MAX_CONNECTIONS` | `25` | Max database connections | Tune based on load |
| `UNKEY_BUILDERD_DATABASE_IDLE_TIMEOUT` | `5m` | Connection idle timeout | Balance performance vs resources |

### Docker Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_DOCKER_SOCKET` | `/var/run/docker.sock` | Docker daemon socket | Ensure proper permissions |
| `UNKEY_BUILDERD_DOCKER_TIMEOUT` | `300s` | Docker operation timeout | Adjust for large images |
| `UNKEY_BUILDERD_DOCKER_PULL_POLICY` | `missing` | Image pull policy (always, missing, never) | Use `always` for production |
| `UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR` | - | Registry mirror URL | Use local mirror for performance |

### OpenTelemetry Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_OTEL_ENABLED` | `true` | Enable OpenTelemetry | Always enable in production |
| `UNKEY_BUILDERD_OTEL_ENDPOINT` | `http://localhost:4317` | OTLP collector endpoint | Use production collector |
| `UNKEY_BUILDERD_OTEL_SERVICE_NAME` | `builderd` | Service name for tracing | Keep consistent across deployments |
| `UNKEY_BUILDERD_OTEL_SAMPLE_RATE` | `1.0` | Trace sampling rate (0.0-1.0) | Reduce for high traffic (e.g., 0.1) |
| `UNKEY_BUILDERD_OTEL_INSECURE` | `true` | Use insecure OTLP connection | Set to `false` with TLS in production |

### Security Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_ENABLE_ISOLATION` | `true` | Enable tenant isolation | Always enable in production |
| `UNKEY_BUILDERD_ENABLE_ENCRYPTION` | `true` | Enable storage encryption | Required for Enterprise+ tiers |
| `UNKEY_BUILDERD_BUILD_USER_ID` | `1000` | Build process user ID | Use dedicated builderd user |
| `UNKEY_BUILDERD_BUILD_GROUP_ID` | `1000` | Build process group ID | Use dedicated builderd group |
| `UNKEY_BUILDERD_ENABLE_SECCOMP` | `true` | Enable seccomp profiles | Provides additional security |
| `UNKEY_BUILDERD_ENABLE_CGROUPS` | `true` | Enable cgroups v2 resource limits | Required for resource isolation |

### Rate Limiting & Quotas

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_DEFAULT_TENANT_TIER` | `free` | Default tier for new tenants | Use appropriate default |
| `UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS` | `10` | Global concurrent build limit | Adjust based on capacity |
| `UNKEY_BUILDERD_BUILD_TIMEOUT` | `900s` | Default build timeout | Tier-specific limits apply |
| `UNKEY_BUILDERD_RATE_LIMIT_ENABLED` | `true` | Enable rate limiting | Always enable in production |
| `UNKEY_BUILDERD_RATE_LIMIT_REQUESTS` | `100` | Requests per minute per tenant | Adjust based on tier |

### Monitoring Configuration

| Variable | Default | Description | Production Notes |
|----------|---------|-------------|------------------|
| `UNKEY_BUILDERD_METRICS_ENABLED` | `true` | Enable Prometheus metrics | Always enable in production |
| `UNKEY_BUILDERD_METRICS_PORT` | `9090` | Metrics endpoint port | Secure access to metrics |
| `UNKEY_BUILDERD_HEALTH_CHECK_ENABLED` | `true` | Enable health check endpoint | Required for load balancers |
| `UNKEY_BUILDERD_HEALTH_CHECK_INTERVAL` | `30s` | Health check interval | Tune for responsiveness |

## Configuration Examples

### Development Configuration

```bash
# .env.development
UNKEY_BUILDERD_LOG_LEVEL=debug
UNKEY_BUILDERD_LOG_FORMAT=text
UNKEY_BUILDERD_OTEL_ENABLED=false
UNKEY_BUILDERD_DATABASE_TYPE=sqlite
UNKEY_BUILDERD_DATABASE_URL=file:./builderd.db
UNKEY_BUILDERD_ENABLE_ISOLATION=false
UNKEY_BUILDERD_ENABLE_ENCRYPTION=false
UNKEY_BUILDERD_SCRATCH_DIR=./tmp/scratch
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=./tmp/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=./tmp/workspace
```

### Production Configuration

```bash
# /etc/builderd/builderd.env
UNKEY_BUILDERD_HOST=0.0.0.0
UNKEY_BUILDERD_PORT=8082
UNKEY_BUILDERD_LOG_LEVEL=warn
UNKEY_BUILDERD_LOG_FORMAT=json

# Storage
UNKEY_BUILDERD_SCRATCH_DIR=/var/lib/builderd/scratch
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/var/lib/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/var/lib/builderd/workspace
UNKEY_BUILDERD_CACHE_DIR=/var/lib/builderd/cache

# Database
UNKEY_BUILDERD_DATABASE_TYPE=sqlite
UNKEY_BUILDERD_DATABASE_DATA_DIR=/var/lib/builderd/data

# Security
UNKEY_BUILDERD_ENABLE_ISOLATION=true
UNKEY_BUILDERD_ENABLE_ENCRYPTION=true
UNKEY_BUILDERD_BUILD_USER_ID=1000
UNKEY_BUILDERD_BUILD_GROUP_ID=1000

# OpenTelemetry
UNKEY_BUILDERD_OTEL_ENABLED=true
UNKEY_BUILDERD_OTEL_ENDPOINT=https://otel-collector.example.com:4317
UNKEY_BUILDERD_OTEL_INSECURE=false
UNKEY_BUILDERD_OTEL_SAMPLE_RATE=0.1

# Docker
UNKEY_BUILDERD_DOCKER_PULL_POLICY=always
UNKEY_BUILDERD_DOCKER_REGISTRY_MIRROR=https://registry-mirror.example.com

# Limits
UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=50
UNKEY_BUILDERD_BUILD_TIMEOUT=1800s
```

### Docker Configuration

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
      UNKEY_BUILDERD_DATABASE_TYPE: "sqlite"
      UNKEY_BUILDERD_DATABASE_DATA_DIR: "/var/lib/builderd/data"
      UNKEY_BUILDERD_OTEL_ENDPOINT: "http://jaeger:14268/api/traces"
    volumes:
      - builderd_data:/var/lib/builderd
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      - jaeger

volumes:
  builderd_data:
```

### Kubernetes Configuration

```yaml
# kubernetes/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: builderd-config
data:
  UNKEY_BUILDERD_HOST: "0.0.0.0"
  UNKEY_BUILDERD_PORT: "8082"
  UNKEY_BUILDERD_LOG_LEVEL: "info"
  UNKEY_BUILDERD_LOG_FORMAT: "json"
  UNKEY_BUILDERD_DATABASE_TYPE: "sqlite"
  UNKEY_BUILDERD_DATABASE_DATA_DIR: "/var/lib/builderd/data"
  UNKEY_BUILDERD_OTEL_ENABLED: "true"
  UNKEY_BUILDERD_OTEL_ENDPOINT: "http://jaeger-collector:14268/api/traces"
  UNKEY_BUILDERD_ENABLE_ISOLATION: "true"
  UNKEY_BUILDERD_ENABLE_ENCRYPTION: "true"
```

## Advanced Configuration

### Database Configuration

SQLite3 is the recommended database for builderd due to its simplicity, reliability, and excellent performance for build workloads:

```bash
# Production SQLite configuration
UNKEY_BUILDERD_DATABASE_TYPE=sqlite
UNKEY_BUILDERD_DATABASE_DATA_DIR=/var/lib/builderd/data

# Advanced SQLite tuning
UNKEY_BUILDERD_DATABASE_PRAGMA_JOURNAL_MODE=WAL
UNKEY_BUILDERD_DATABASE_PRAGMA_SYNCHRONOUS=NORMAL
UNKEY_BUILDERD_DATABASE_PRAGMA_CACHE_SIZE=10000
UNKEY_BUILDERD_DATABASE_PRAGMA_TEMP_STORE=MEMORY
```

SQLite3 advantages for builderd:
- **Single file database**: Easy backup and deployment
- **No external dependencies**: Simplified infrastructure
- **Excellent performance**: Perfect for build metadata storage
- **ACID compliance**: Data integrity guaranteed
- **Concurrent reads**: Multiple readers with WAL mode

### Storage Backend Configuration

#### Local Storage (Development)

```bash
UNKEY_BUILDERD_SCRATCH_DIR=/tmp/builderd/scratch
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/tmp/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/tmp/builderd/workspace
```

#### Network Storage (Production)

```bash
# NFS storage
UNKEY_BUILDERD_SCRATCH_DIR=/mnt/nfs/builderd/scratch
UNKEY_BUILDERD_ROOTFS_OUTPUT_DIR=/mnt/nfs/builderd/rootfs
UNKEY_BUILDERD_WORKSPACE_DIR=/mnt/nfs/builderd/workspace

# Cloud storage integration (future)
UNKEY_BUILDERD_STORAGE_BACKEND=s3
UNKEY_BUILDERD_S3_BUCKET=builderd-artifacts
UNKEY_BUILDERD_S3_REGION=us-east-1
```

### Security Hardening

#### AppArmor/SELinux Integration

```bash
# Enable security profiles
UNKEY_BUILDERD_APPARMOR_PROFILE=builderd-profile
UNKEY_BUILDERD_SELINUX_CONTEXT=builderd_t

# Additional security options
UNKEY_BUILDERD_ENABLE_AUDIT_LOGGING=true
UNKEY_BUILDERD_AUDIT_LOG_FILE=/var/log/builderd/audit.log
```

#### Network Security

```bash
# Network isolation
UNKEY_BUILDERD_NETWORK_MODE=isolated
UNKEY_BUILDERD_ALLOWED_REGISTRIES=docker.io,ghcr.io,registry.example.com
UNKEY_BUILDERD_BLOCKED_DOMAINS=malware.example.com,suspicious.example.com

# TLS configuration
UNKEY_BUILDERD_TLS_ENABLED=true
UNKEY_BUILDERD_TLS_CERT_FILE=/etc/builderd/tls/cert.pem
UNKEY_BUILDERD_TLS_KEY_FILE=/etc/builderd/tls/key.pem
UNKEY_BUILDERD_TLS_CA_FILE=/etc/builderd/tls/ca.pem
```

## Configuration Validation

### Startup Validation

Builderd validates configuration at startup and will fail to start with invalid settings:

```bash
# Test configuration
./builderd --validate-config

# Check configuration with dry-run
./builderd --dry-run
```

### Health Checks

Monitor configuration health:

```bash
# Configuration health endpoint
curl http://localhost:8082/health/config

# Detailed configuration status
curl http://localhost:8082/debug/config
```

### Common Configuration Issues

#### Docker Socket Permissions

```bash
# Fix Docker socket permissions
sudo usermod -aG docker builderd
sudo systemctl restart builderd
```

#### Storage Permissions

```bash
# Fix storage directory permissions
sudo chown -R builderd:builderd /var/lib/builderd
sudo chmod -R 750 /var/lib/builderd
```

#### Database Connection Issues

```bash
# Test SQLite database accessibility
sqlite3 /var/lib/builderd/data/builderd.db "SELECT 1;"

# Check database file permissions
ls -la /var/lib/builderd/data/builderd.db

# Check database status
curl http://localhost:8082/debug/database
```

## Environment-Specific Examples

### CI/CD Integration

```bash
# GitHub Actions environment
UNKEY_BUILDERD_HOST=0.0.0.0
UNKEY_BUILDERD_LOG_LEVEL=debug
UNKEY_BUILDERD_OTEL_ENABLED=false
UNKEY_BUILDERD_DATABASE_URL=file:./test.db
UNKEY_BUILDERD_ENABLE_ISOLATION=false
UNKEY_BUILDERD_DOCKER_PULL_POLICY=always
```

### High-Availability Production

```bash
# Load balancer configuration
UNKEY_BUILDERD_HOST=0.0.0.0
UNKEY_BUILDERD_PORT=8082
UNKEY_BUILDERD_HEALTH_CHECK_ENABLED=true

# Database (SQLite with backup)
UNKEY_BUILDERD_DATABASE_TYPE=sqlite
UNKEY_BUILDERD_DATABASE_DATA_DIR=/var/lib/builderd/data
UNKEY_BUILDERD_DATABASE_BACKUP_ENABLED=true
UNKEY_BUILDERD_DATABASE_BACKUP_INTERVAL=1h

# Distributed storage
UNKEY_BUILDERD_STORAGE_BACKEND=distributed
UNKEY_BUILDERD_STORAGE_NODES=node1.example.com,node2.example.com,node3.example.com
```

For deployment-specific configuration, see the [Deployment & Operations Guide](deployment-operations-guide.md).
