# Builderd TODO

## High Priority

- [ ] Implement build caching
  - Layer caching across builds
  - Distributed cache support
  - Cache eviction policies

- [ ] Add multi-tenancy support
  - Per-tenant resource quotas
  - Build isolation between tenants
  - Tenant-specific registries

- [ ] Implement build queue management
  - Priority queue for builds
  - Queue persistence across restarts
  - Build scheduling optimizations

## Medium Priority

- [ ] Add support for BuildKit
  - BuildKit backend option
  - Advanced build features
  - Better caching and parallelism

- [ ] Implement build artifacts storage
  - S3-compatible storage backend
  - Artifact retention policies
  - Artifact signing and verification

- [ ] Add build reproducibility features
  - Deterministic builds
  - Build attestation
  - SBOM generation

## Low Priority

- [ ] Add support for other container runtimes
  - Podman support
  - Containerd support
  - Runtime abstraction layer

- [ ] Implement distributed builds
  - Build farm support
  - Work distribution algorithm
  - Cross-node caching

- [ ] Add build analytics
  - Build performance metrics
  - Resource usage tracking
  - Cost analysis per build

## Completed

- [x] Basic service implementation
- [x] Docker integration
- [x] Multi-tenant build support
- [x] ConnectRPC API
- [x] Prometheus metrics
- [x] SPIFFE/mTLS support
- [x] Grafana dashboards
- [x] Container to rootfs conversion
- [x] Unified health endpoint
- [x] Unified Makefile structure