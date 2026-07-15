// Package containers provides Docker Compose-backed containers for integration tests.
//
// This package manages shared Docker services for integration tests.
// Containers are reused through one Docker Compose project per worktree, so
// separate Go test processes share backing services without colliding with
// other worktrees.
//
// # Requirements
//
// Tests using this package can be run through Rask:
//
//	mise exec -- rask ./pkg/testutil/containers
//
// Prefer `mise run test` for full-suite runs. It sets the worktree scope that
// lets Go test processes share containers and removes the scoped Compose project
// when the suite exits.
//
// # Usage
//
// Each service function starts a Compose container and returns connection information:
//
//	func TestRedisIntegration(t *testing.T) {
//	    redisURL := containers.Redis(t)
//	    // redisURL is "redis://localhost:{randomPort}"
//	    // Later tests attach to the same Redis container.
//	}
//
// # Design
//
// Containers are created on demand by pkg/testutil/docker-compose.test.yaml and
// reused by later test requests in the same worktree. Compose assigns random
// host ports to avoid conflicts and waits for container healthchecks before the
// helpers return connection information.
//
// # Available Services
//
// Currently supported:
//   - [MySQL]: MySQL with dev schema preloaded
//   - [Redis]: Redis 8.0
//   - [S3]: MinIO S3-compatible object storage
//   - [Restate]: Restate server (ingress + admin)
//   - [ClickHouse]: ClickHouse with dev schema preloaded
package containers
