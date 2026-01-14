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
//	    redisURL := dockertest.Redis(t)
//	    // redisURL is "redis://localhost:{randomPort}"
//	    // Container is automatically removed when test completes
//	}
//
// # Design
//
// Containers are created per-test for isolation. Each container:
//   - Uses a random host port to avoid conflicts
//   - Waits for the service to be ready before returning
//   - Is automatically removed via t.Cleanup
//
// # Available Services
//
// Currently supported:
//   - [Redis]: Redis 8.0 container
//   - [S3]: MinIO S3-compatible object storage
package dockertest
