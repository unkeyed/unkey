# Kubernetes Development Setup

This guide shows how to run Unkey services locally using Docker Desktop or OrbStack Kubernetes instead of Docker Compose.

## Prerequisites

Install the required tools:

- **Docker Desktop with Kubernetes enabled** OR **OrbStack with Kubernetes enabled**
- **kubectl**: https://kubernetes.io/docs/tasks/tools/

Or check if already installed:

```bash
make k8s-check
```

Optional for enhanced development experience:

- **Tilt**: https://docs.tilt.dev/install.html

## Quick Start

### Option 1: Full Environment (Recommended)

```bash
make k8s-up
```

This will:

- Use your current Kubernetes cluster (Docker Desktop/OrbStack)
- Build and deploy all services (MySQL, ClickHouse, S3, Observability, Unkey)
- Wait for all services to be ready
- Show connection info

### Option 2: Individual Services

```bash
# Start only MySQL
make start-mysql

# Start only ctrl (requires cluster to exist)
make start-ctrl

# Start all services individually
make start-all
```

### Option 3: Enhanced Development with Tilt

```bash
make dev
```

If Tilt is installed, this provides:

- Hot reloading for Go code changes
- Unified log viewing in web UI (http://localhost:10350)
- Resource management dashboard
- Automatic rebuilds on file changes
- Selective service startup

## Accessing Services

### Via NodePort (OrbStack)

Services are accessible directly on localhost via NodePort. Check actual port assignments with:

```bash
make k8s-ports
```

This will show the randomly assigned NodePorts, e.g.:

- **Dashboard**: http://localhost:3000
- **API**: http://localhost:32002
- **Gateway**: http://localhost:32003
- **Ctrl**: http://localhost:32004
- **Prometheus**: http://localhost:32005
- **S3 Console**: http://localhost:32006

### Alternative: LoadBalancer domains

OrbStack also supports LoadBalancer services with automatic `*.k8s.orb.local` domains, but NodePort is simpler for development.

### Inside the cluster

- **MySQL**: `mysql.unkey.svc.cluster.local:3306`
- **ClickHouse**: `clickhouse.unkey.svc.cluster.local:8123`
- **S3**: `s3.unkey.svc.cluster.local:9000`
- **Prometheus**: `prometheus.unkey.svc.cluster.local:9090`
- **OTEL Collector**: `otel-collector.unkey.svc.cluster.local:4317`

## Development Workflow

### Making Code Changes

#### With Tilt (Hot Reloading)

1. Start Tilt: `make dev`
2. Edit Go files
3. Changes automatically rebuild and restart services
4. View logs in Tilt UI

#### Without Tilt (Manual)

1. Make code changes
2. Rebuild and redeploy:
   ```bash
   make start-ctrl  # Rebuilds and redeploys ctrl
   ```

### Managing the Environment

```bash
# Check status
make k8s-status

# Reset everything (delete and recreate)
make k8s-reset

# Stop everything
make k8s-down
```

### Debugging

```bash
# View logs
kubectl logs -n unkey -l app=ctrl -f
kubectl logs -n unkey -l app=mysql -f

# Get shell access
kubectl exec -n unkey -it deployment/ctrl -- /bin/sh
kubectl exec -n unkey -it deployment/mysql -- /bin/bash

# Check service health
kubectl get pods -n unkey
kubectl describe pod -n unkey <pod-name>
```

## Configuration

### Environment Variables

The ctrl service uses these key environment variables:

- `UNKEY_DATABASE_PRIMARY`: Connection to partition database
- `UNKEY_DATABASE_HYDRA`: Connection to hydra database
- `UNKEY_HTTP_PORT`: Service port (8084)
- `UNKEY_PLATFORM`: Set to "kubernetes"

### Database Setup

MySQL is automatically configured with:

- `unkey` database for main data
- `hydra` database for OAuth flows
- `partition_001` database for partitioned data
- User `unkey` with password `password`

All schemas are automatically applied on startup.

## Customization

### Selective Services with Tilt

```bash
# Start only database services
tilt up mysql,clickhouse,planetscale

# Start only storage services
tilt up s3

# Start only observability
tilt up observability

# Start only Unkey services
tilt up ctrl
```

### Custom Resource Limits

Edit `k8s/manifests/ctrl.yaml` to adjust:

```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

### Persistent Data

MySQL data persists between restarts via PersistentVolumeClaim. To reset:

```bash
kubectl delete pvc -n unkey mysql-pvc
make start-mysql
```

## Troubleshooting

### Common Issues

#### Cluster Not Available

```bash
# Make sure Kubernetes is enabled in Docker Desktop/OrbStack
# Check current context
kubectl config current-context

# Switch context if needed
kubectl config use-context docker-desktop  # or orbstack
```

#### Services Not Ready

```bash
# Check events
kubectl get events -n unkey --sort-by='.lastTimestamp'

# Check pod logs
kubectl logs -n unkey -l app=mysql
kubectl logs -n unkey -l app=ctrl
```

#### Port Conflicts

If ports 8084 or 3306 are in use:

```bash
# Check what's using the port
lsof -i :8084
lsof -i :3306

# Kill other services or change ports in k8s manifests
```

#### Image Build Issues

```bash
# Manual build
docker build -t unkey/mysql:latest -f ../deployment/Dockerfile.mysql ../
docker build -t unkey:latest .
```

### Performance Tips

- Use `make dev` for fastest development cycle
- Keep cluster running between sessions (don't run `k8s-down`)
- Use selective service startup when working on specific components

## vs Docker Compose

| Feature               | Kubernetes                       | Docker Compose         |
| --------------------- | -------------------------------- | ---------------------- |
| **Prod Similarity**   | ✅ Real Kubernetes               | ❌ Different from prod |
| **Resource Usage**    | ✅ Native containers             | ✅ Native containers   |
| **Setup Complexity**  | ⚠️ Enable K8s in Docker/OrbStack | ✅ Simple setup        |
| **Service Discovery** | ✅ K8s native DNS                | ⚠️ Docker networks     |
| **Scaling**           | ✅ Easy horizontal scaling       | ❌ Limited scaling     |
| **Hot Reloading**     | ✅ With Tilt                     | ⚠️ Manual restarts     |
| **Debugging**         | ✅ Rich tooling                  | ✅ Simple logs         |
| **Architecture**      | ✅ Clean separation              | ✅ Clean separation    |

Choose Kubernetes for true prod-like development, Docker Compose for quick iterations.

## Available Services

All services can be started individually or together:

- `mysql` - MySQL database with schemas
- `clickhouse` - ClickHouse analytics database
- `s3` - MinIO object storage
- `planetscale` - PlanetScale HTTP database proxy
- `observability` - Prometheus + OTEL Collector
- `ctrl` - Main Unkey service (ctrl/api/gw)

Use `make k8s-up` to start everything or selectively with Tilt.
