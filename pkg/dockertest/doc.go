// Package dockertest provides lightweight Docker container management for integration tests.
//
// This package uses the Docker SDK directly to spin up containers dynamically,
// avoiding the overhead of external tools like docker-compose or testcontainers.
// Containers are automatically cleaned up when tests complete.
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
// # Requirements
//
// Tests using this package require Docker to be running and accessible.
// If Docker is unavailable, tests will be skipped with t.Skip().
//
// # Design
//
// Containers are created per-test for isolation. Each container:
//   - Uses a random host port to avoid conflicts
//   - Waits for the service to be ready before returning
//   - Is automatically removed via t.Cleanup()
//
// # Available Services
//
// Currently supported:
//   - [Redis]: Redis 8.0 container
//
// # Comparison with pkg/testutil/containers
//
// The [pkg/testutil/containers] package returns hardcoded localhost URLs and
// expects services to be running via docker-compose. This package dynamically
// starts containers, making tests self-contained and runnable without external
// setup. Both packages can coexist during migration.
package dockertest
