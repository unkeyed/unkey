# Service Port Mapping

This document lists all network ports used by services in the Unkey deployment stack.

## Main Service Ports

| Service | Port | Protocol | Purpose | Interface |
|---------|------|----------|---------|-----------|
| metald | 8080 | HTTP/2 | ConnectRPC API - VM lifecycle management | 0.0.0.0 |
| billaged | 8081 | HTTP/2 | ConnectRPC API - Billing metrics collection | 0.0.0.0 |
| builderd | 8082 | HTTP/2 | ConnectRPC API - Container to rootfs conversion | 0.0.0.0 |
| assetmanagerd | 8083 | HTTP/2 | ConnectRPC API - VM asset registry | 0.0.0.0 |

## Prometheus Metrics Ports

| Service | Port | Paths | Interface | Purpose |
|---------|------|-------|-----------|---------|
| metald | 9464 | /metrics, /health | 127.0.0.1 | Service metrics & health |
| billaged | 9465 | /metrics, /health | 127.0.0.1 | Billing metrics & health |
| builderd | 9466 | /metrics, /health | 127.0.0.1 | Build metrics & health |
| assetmanagerd | 9467 | /metrics, /health | 127.0.0.1 | Asset metrics & health |

## SPIRE Infrastructure Ports

| Component | Port | Purpose | Interface |
|-----------|------|---------|-----------|
| SPIRE Server | 8085 | Server API | 0.0.0.0 |
| SPIRE Server | 9991 | Health checks (/live, /ready) | 127.0.0.1 |
| SPIRE Server | 9988 | Prometheus metrics | 127.0.0.1 |
| SPIRE Agent | 9990 | Health checks (/live, /ready) | 127.0.0.1 |
| SPIRE Agent | 9989 | Prometheus metrics | 127.0.0.1 |

## Health Check Endpoints

All services expose health checks on their metrics ports (non-TLS):

| Service | URL | Response Format |
|---------|-----|-----------------|
| metald | http://localhost:9464/health | JSON: `{"status":"ok","service":"metald","version":"0.1.0","uptime_seconds":123.45}` |
| billaged | http://localhost:9465/health | JSON: `{"status":"ok","service":"billaged","version":"0.1.0","uptime_seconds":123.45}` |
| builderd | http://localhost:9466/health | JSON: `{"status":"ok","service":"builderd","version":"0.1.0","uptime_seconds":123.45}` |
| assetmanagerd | http://localhost:9467/health | JSON: `{"status":"ok","service":"assetmanagerd","version":"0.1.0","uptime_seconds":123.45}` |

## Unix Domain Sockets

These are not network ports but local communication endpoints:

| Service | Socket Path | Purpose |
|---------|-------------|---------|
| SPIRE Server | /run/spire/server.sock | Local admin API |
| SPIRE Agent | /run/spire/sockets/agent.sock | Workload API for SVID delivery |
| Firecracker | /tmp/firecracker.sock | VM management API |
| Cloud Hypervisor | /tmp/ch.sock | VM management API |

## External Service Endpoints

These are configured endpoints for external services:

| Service | Default Endpoint | Purpose |
|---------|------------------|---------|
| OpenTelemetry Collector | localhost:4318 | OTLP trace/metric export |

## Port Conflicts

✅ **Previously Resolved Conflicts:**
- **SPIRE Server** was on port 8081 conflicting with **billaged (8081)**
  - Resolution: SPIRE Server moved to port 8085
- **SPIRE Server health** was on port 8080 conflicting with **metald (8080)**
  - Resolution: SPIRE Server health moved to port 9991
- **SPIRE Agent health** was on port 8082 conflicting with **builderd (8082)**
  - Resolution: SPIRE Agent health moved to port 9990

## Network Security Recommendations

1. **API Ports (8080-8083)**: Should be behind a load balancer/API gateway with proper authentication
2. **Metrics Ports (9464-9466, 9988-9989)**: Restrict to monitoring systems only (Prometheus scraper)
3. **Health Check Ports**: Can be exposed to load balancers for health monitoring
4. **SPIRE Ports**: Server API (8081) should be protected; agent socket requires local access only
5. **All Services**: When TLS mode is enabled, inter-service communication uses mTLS via SPIFFE

## Environment Variables for Port Configuration

| Service | Environment Variable | Default | Description |
|---------|---------------------|---------|-------------|
| metald | UNKEY_METALD_PORT | 8080 | Main API port |
| metald | UNKEY_METALD_OTEL_PROMETHEUS_PORT | 9464 | Metrics port |
| billaged | UNKEY_BILLAGED_PORT | 8081 | Main API port |
| billaged | UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT | 9465 | Metrics port |
| builderd | UNKEY_BUILDERD_PORT | 8082 | Main API port |
| builderd | UNKEY_BUILDERD_OTEL_PROMETHEUS_PORT | 9466 | Metrics port |
| assetmanagerd | UNKEY_ASSETMANAGERD_PORT | 8083 | Main API port |
| assetmanagerd | UNKEY_ASSETMANAGERD_OTEL_PROMETHEUS_PORT | 9467 | Metrics port |

## Quick Reference

```bash
# Check all listening ports
sudo ss -tlnp | grep -E "(8080|8081|8082|8083|8085|9464|9465|9466|9467|9988|9989|9990|9991)"

# Test service health (on metrics ports)
curl http://localhost:9464/health  # metald
curl http://localhost:9465/health  # billaged
curl http://localhost:9466/health  # builderd
curl http://localhost:9467/health  # assetmanagerd

# Test SPIRE health
curl http://localhost:9991/live   # SPIRE server
curl http://localhost:9990/live   # SPIRE agent
```

## AIDEV-NOTE

- All ports are configurable via environment variables
- The unified health endpoint format was implemented across all services
- SPIRE Agent port was changed from 8082 to 8084 to avoid collision with builderd
- Health endpoints moved from main TLS ports to metrics ports (non-TLS) for monitoring
- Fixed port conflict: assetmanagerd moved from 9466 to 9467 (was conflicting with builderd)
- Consider using a service mesh or API gateway to manage port allocation in production