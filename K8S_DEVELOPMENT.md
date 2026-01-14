# Kubernetes Development Setup

Run Unkey services locally using Kubernetes instead of Docker Compose.

## Prerequisites

- Docker Desktop with Kubernetes enabled OR OrbStack with Kubernetes enabled
- kubectl
- [just](https://github.com/casey/just) - Install with `brew install just`

## Quick Start

Start everything with Tilt:
```bash
just dev
```

This will:
1. Create a minikube cluster
2. Enable metrics-server addon
3. Start Tilt with all services

## Docker Compose Alternative

If you prefer Docker Compose over Kubernetes:

```bash
# Start infrastructure services
just up

# Stop and clean up
just clean
```

## Management

```bash
# Stop Tilt (keeps cluster)
just down

# Delete minikube cluster entirely
just nuke

# View services
kubectl get pods -n unkey
kubectl get services -n unkey
```

## Tilt (Advanced)

Start specific services manually:
```bash
tilt up -f ./dev/Tiltfile -- --services=mysql --services=clickhouse
tilt up -f ./dev/Tiltfile -- --services=api --services=gw --services=ctrl
```

## Seeding Local Environment

After services are running:
```bash
just unkey dev seed local --slug myproject
```
