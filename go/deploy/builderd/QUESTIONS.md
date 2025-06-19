# builderd Questions

Q: What is the primary purpose of builderd in the Unkey Deploy ecosystem?
A: builderd transforms various source types (Docker images, Git repositories, archives) into optimized rootfs images specifically designed for Firecracker microVM execution, providing multi-tenant build isolation and resource management.

Q: How does builderd handle multi-tenant isolation during builds?
A: builderd uses Linux namespaces (PID, network, mount) for process and filesystem isolation, cgroups for resource limits, and tenant-specific storage buckets to ensure complete isolation between different tenants' build processes.

Q: What build sources are currently supported vs planned?
A: Currently implemented: Docker image extraction with registry authentication. Planned but not implemented: Git repository builds and archive extraction (tar.gz, zip).

Q: How does builderd optimize rootfs images for microVM usage?
A: builderd can strip debug symbols, compress binaries, remove documentation and package caches, flatten Docker layers, and apply custom optimization settings to minimize the rootfs size while preserving functionality.

Q: What are the tenant tier levels and their implications?
A: Four tiers exist: FREE (limited resources), PRO (standard resources), ENTERPRISE (higher limits + enhanced isolation), and DEDICATED (dedicated infrastructure). Each tier has different resource quotas for CPU, memory, disk, concurrent builds, and daily build limits.

Q: How does builderd integrate with other Unkey Deploy services?
A: Currently builderd operates standalone. Future integrations planned: metald will consume rootfs outputs for VM provisioning, assetmanagerd will provide centralized artifact management, and billaged will receive resource usage metrics for billing.

Q: What storage backends are supported for build artifacts?
A: Three backends are supported: local filesystem (for development), S3/S3-compatible storage (for cloud deployments), and Google Cloud Storage (for GCP environments). All backends support multi-tenant isolation and retention policies.

Q: How is build progress tracked and monitored?
A: builderd provides real-time log streaming via StreamBuildLogs RPC, detailed build states (PENDING, PULLING, EXTRACTING, BUILDING, OPTIMIZING, COMPLETED, FAILED), progress percentages, and comprehensive OpenTelemetry metrics for monitoring.

Q: What security measures are in place for build execution?
A: Security includes SPIFFE/mTLS for service communication, unprivileged build processes, read-only root filesystems during builds, no network access during builds, registry allowlists, and tenant-specific authentication validation.

Q: How does builderd handle resource quotas and limits?
A: Each tenant has configurable limits for memory (bytes), CPU cores, disk space, build timeout, concurrent builds, daily build count, and total storage. Quotas are enforced per tenant tier and checked before build execution.

Q: What happens when a build fails or times out?
A: Failed builds transition to BUILD_STATE_FAILED with error details logged. Timed-out builds are automatically cancelled and cleaned up. Resources are released, and metrics are recorded for monitoring and debugging.

Q: How does the build caching system work?
A: builderd implements layer-based caching for Docker builds using the cache key format {tenant_id}/{image_digest}/{layer_digest}. Cache uses LRU eviction with configurable size limits to improve performance for repeated builds.

Q: What is the difference between synchronous and asynchronous build execution?
A: Currently builderd executes builds synchronously in the request handler for simplicity. Future plans include implementing async queue-based execution for better scalability and resource utilization.

Q: How are build artifacts stored and retrieved?
A: Artifacts are stored in the configured storage backend with paths like {tenant_id}/builds/{build_id}/rootfs.tar. Each tenant has isolated storage with configurable retention periods and size limits.

Q: What metrics are exposed for monitoring builderd?
A: Key metrics include builderd_builds_total (by state/tenant), builderd_build_duration_seconds, builderd_concurrent_builds, builderd_resource_usage (CPU/memory/disk), and builderd_tenant_quota_usage for comprehensive observability.