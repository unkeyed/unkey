// Package containers provides testing utilities for integration tests with docker-compose services.
//
// This package simplifies integration testing by providing pre-configured connections
// to services managed by docker-compose. Instead of dynamically discovering service
// ports (which is slow), it uses hardcoded port mappings that match the docker-compose
// configuration for consistent and fast test execution.
//
// The package was designed for scenarios where tests need to connect to real external
// services like MySQL, Redis, ClickHouse, S3/MinIO, and OTEL collectors. It provides
// both host configurations (for test runners connecting from outside containers) and
// docker configurations (for services running inside the docker-compose network).
//
// # Key Design Decisions
//
// We chose hardcoded ports over dynamic discovery because dynamic port discovery
// using docker-compose commands added significant overhead to test execution
// (hundreds of milliseconds per service). Since our docker-compose configuration
// uses fixed port mappings, hardcoding them provides the same functionality with
// zero runtime overhead.
//
// # Key Types
//
// The main entry points are the service configuration functions: [MySQL], [Redis],
// [ClickHouse], [S3], and [OTEL]. Each returns appropriate configuration objects
// or clients for connecting to the respective services.
//
// Configuration objects include [S3Config] for S3/MinIO settings and [OTELConfig]
// for OpenTelemetry endpoint configuration.
//
// # Usage
//
// Basic setup in integration tests:
//
//	func TestDatabaseOperations(t *testing.T) {
//		containers.StartAllServices(t) // No-op, services managed externally
//
//		hostCfg, dockerCfg := containers.MySQL(t)
//		db, err := sql.Open("mysql", hostCfg.FormatDSN())
//		require.NoError(t, err)
//		defer db.Close()
//
//		// Run your database tests...
//	}
//
// Multiple services example:
//
//	func TestFullIntegration(t *testing.T) {
//		// Get MySQL connection
//		hostCfg, _ := containers.MySQL(t)
//		db, err := sql.Open("mysql", hostCfg.FormatDSN())
//		require.NoError(t, err)
//		defer db.Close()
//
//		// Get Redis client
//		redisClient, hostAddr, _ := containers.Redis(t)
//		defer redisClient.Close()
//
//		// Get S3 configuration
//		s3Config := containers.S3(t)
//
//		// Run integration tests with all services...
//	}
//
// # Service Port Configuration
//
// The package uses these hardcoded port mappings that must match your docker-compose.yaml:
//
//   - MySQL: 3306
//   - Redis: 6379
//   - ClickHouse: 9000
//   - S3/MinIO: 3902
//   - OTEL HTTP: 4318
//   - OTEL gRPC: 4317
//   - Grafana: 3000
//
// # Prerequisites
//
// Tests using this package require:
//   - docker-compose services running before test execution
//   - Port mappings in docker-compose.yaml matching the hardcoded constants
//   - Network connectivity from test runner to localhost on the specified ports
//
// # Host vs Docker Configurations
//
// Most service functions return two configurations:
//   - Host configuration: For connecting from the test runner (uses localhost:port)
//   - Docker configuration: For services running inside docker-compose network (uses service:port)
//
// Use host configuration in your tests, and docker configuration when configuring
// services that run inside the docker-compose network and need to connect to each other.
package containers
