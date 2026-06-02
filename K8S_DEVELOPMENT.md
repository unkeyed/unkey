# Kubernetes Development Setup

Run Unkey services locally using Kubernetes instead of Docker Compose.

## Prerequisites

- Docker Desktop with Kubernetes enabled OR OrbStack with Kubernetes enabled
- kubectl

Install pinned project tooling and dependencies:
```bash
./dev/install-mise
mise install --yes --locked
mise run install
```

If GitHub rate-limits attestation checks, retry with `GITHUB_TOKEN` set:

```bash
GITHUB_TOKEN=... mise install --yes --locked
```

## Quick Start

Start with hot reloading:
```bash
mise run dev
```

## Individual Services

```bash
tilt up -f ./dev/Tiltfile -- --services=mysql --services=clickhouse
tilt up -f ./dev/Tiltfile -- --services=api --services=gw --services=ctrl
tilt up -f ./dev/Tiltfile -- --services=all
```

## Management

```bash
# Stop everything
mise run down

# Reset environment
mise run down

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
mise run down
```
