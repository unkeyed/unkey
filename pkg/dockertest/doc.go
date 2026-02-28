// Package dockertest provides lightweight Docker container management for integration tests.
//
// This package uses the Docker SDK directly to spin up containers dynamically,
// avoiding the overhead of external tools like docker-compose or testcontainers.
// Containers are automatically cleaned up when tests complete via t.Cleanup.
//
// # Requirements
//
// Tests using this package MUST be run via Bazel:
//
//	bazel test //pkg/dockertest:dockertest_test
//
// Running via `go test` directly is not supported and will fail with an error.
//
// # Key Types
//
// The main entry point is [New], which returns a [Cluster] for starting
// containers and retrieving connection information via service methods like
// [Cluster.Redis].
// For lower-level access, [Container] provides metadata about running
// containers including port mappings.
//
// Container readiness is determined by [WaitStrategy] implementations. The package
// provides [TCPWait] for TCP port-based health checks, created via [NewTCPWait].
//
// # Usage
//
// Each service method starts a container and returns connection information:
//
//	func TestRedisIntegration(t *testing.T) {
//	    cluster := dockertest.New(t)
//	    redis := cluster.Redis()
//	    redisURL := redis.HostURL
//	    // redisURL is "redis://localhost:{randomPort}"
//	}
//
// Service configs include host and Docker endpoints, so you can wire
// connections for code running on the host and for other containers in the
// same cluster.
//
// # Design
//
// Containers are created per-test for isolation. Each container:
//   - Uses a random host port to avoid conflicts
//   - Waits for readiness before returning
//   - Is automatically removed via t.Cleanup
//
// # Available Services
//
// Currently supported:
//   - [Cluster.MySQL]: MySQL with dev schema preloaded
//   - [Cluster.Redis]: Redis 8.0 container
//   - [Cluster.ClickHouse]: ClickHouse with schema preloaded
//   - [Cluster.S3]: MinIO S3-compatible object storage
//   - [Cluster.Restate]: Restate server (ingress + admin)
//   - [Cluster.Vault]: Vault service wired to S3
package dockertest
