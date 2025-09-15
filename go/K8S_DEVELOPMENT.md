# Kubernetes Development Setup

Run Unkey services locally using Kubernetes instead of Docker Compose.

## Prerequisites

- Docker Desktop with Kubernetes enabled OR OrbStack with Kubernetes enabled
- kubectl

Check requirements:
```bash
make k8s-check
```

## Quick Start

Start everything:
```bash
make k8s-up
```

Start with hot reloading (requires Tilt):
```bash
make dev
```

## Individual Services

```bash
make start-mysql
make start-clickhouse
make start-redis
make start-s3
make start-api
make start-gw
make start-ctrl
```

## Management

```bash
# Stop everything
make k8s-down

# Reset environment
make k8s-reset

# View services
kubectl get pods -n unkey
kubectl get services -n unkey
```

## Tilt (Optional)

Start specific services:
```bash
tilt up -- --services=mysql --services=clickhouse
tilt up -- --services=api --services=gw --services=ctrl
tilt up -- --services=all
```

Stop Tilt:
```bash
tilt down
```