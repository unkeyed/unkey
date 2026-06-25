// Package containers provides lightweight Docker container management for integration tests.
//
// This package uses the Docker SDK directly to spin up containers dynamically,
// avoiding the overhead of external tools like docker-compose or testcontainers.
// Service containers are reused by stable Docker names scoped to the current
// worktree, so separate Bazel test processes can share the same backing
// services without colliding with other worktrees.
//
// # Requirements
//
// Tests using this package MUST be run via Bazel:
//
//	bazel test //pkg/testutil/containers:containers_test
//
// Running via `go test` directly is not supported and will fail with an error.
//
// # Key Types
//
// The main entry point is through service functions like [Redis], which start
// containers and return connection information. For lower-level access, [Container]
// provides metadata about running containers including port mappings.
//
// Container readiness is determined by [WaitStrategy] implementations. The package
// provides [TCPWait] for TCP port-based health checks, created via [NewTCPWait].
//
// # Usage
//
// Each service function starts a container and returns connection information:
//
//	func TestRedisIntegration(t *testing.T) {
//	    redisURL := containers.Redis(t)
//	    // redisURL is "redis://localhost:{randomPort}"
//	    // Later tests attach to the same Redis container.
//	}
//
// Pass [WithDedicatedContainer] when a test must own an isolated container.
//
// # Design
//
// Service containers are created on demand and reused by later test requests in
// the same worktree. Each container:
//   - Uses a stable Docker name derived from its worktree scope and image
//   - Uses a random host port to avoid conflicts
//   - Waits for the service to be ready before returning
//
// # Available Services
//
// Currently supported:
//   - [MySQL]: MySQL with dev schema preloaded
//   - [Redis]: Redis 8.0 container
//   - [S3]: MinIO S3-compatible object storage
//   - [Restate]: Restate server (ingress + admin)
package containers
