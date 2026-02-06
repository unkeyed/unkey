---
title: containers
description: "provides testing utilities for integration tests with docker-compose services"
---

Package containers provides testing utilities for integration tests with docker-compose services.

This package simplifies integration testing by providing pre-configured connections to services managed by docker-compose. Instead of dynamically discovering service ports (which is slow), it uses hardcoded port mappings that match the docker-compose configuration for consistent and fast test execution.

The package was designed for scenarios where tests need to connect to real external services like MySQL, Redis, ClickHouse, S3/MinIO, and OTEL collectors. It provides both host configurations (for test runners connecting from outside containers) and docker configurations (for services running inside the docker-compose network).

### Key Design Decisions

We chose hardcoded ports over dynamic discovery because dynamic port discovery using docker-compose commands added significant overhead to test execution (hundreds of milliseconds per service). Since our docker-compose configuration uses fixed port mappings, hardcoding them provides the same functionality with zero runtime overhead.

### Key Types

The main entry points are the service configuration functions: \[MySQL], \[Redis], \[ClickHouse], \[S3], and \[OTEL]. Each returns appropriate configuration objects or clients for connecting to the respective services.

Configuration objects include \[S3Config] for S3/MinIO settings and \[OTELConfig] for OpenTelemetry endpoint configuration.

### Usage

Basic setup in integration tests:

	func TestDatabaseOperations(t *testing.T) {
		containers.StartAllServices(t) // No-op, services managed externally

		hostCfg, dockerCfg := dockertest.MySQL(t)
		db, err := sql.Open("mysql", hostCfg.FormatDSN())
		require.NoError(t, err)
		defer db.Close()

		// Run your database tests...
	}

Multiple services example:

	func TestFullIntegration(t *testing.T) {
		// Get MySQL connection
		hostCfg, _ := dockertest.MySQL(t)
		db, err := sql.Open("mysql", hostCfg.FormatDSN())
		require.NoError(t, err)
		defer db.Close()

		// Get Redis client
		redisClient, hostAddr, _ := containers.Redis(t)
		defer redisClient.Close()

		// Get S3 configuration
		s3Config := containers.S3(t)

		// Run integration tests with all services...
	}

### Service Port Configuration

The package uses these hardcoded port mappings that must match your docker-compose.yaml:

  - MySQL: 3306
  - Redis: 6379
  - ClickHouse: 9000
  - S3/MinIO: 3902
  - OTEL HTTP: 4318
  - OTEL gRPC: 4317
  - Grafana: 3000

### Prerequisites

Tests using this package require:

  - docker-compose services running before test execution
  - Port mappings in docker-compose.yaml matching the hardcoded constants
  - Network connectivity from test runner to localhost on the specified ports

### Host vs Docker Configurations

Most service functions return two configurations:

  - Host configuration: For connecting from the test runner (uses localhost:port)
  - Docker configuration: For services running inside docker-compose network (uses service:port)

Use host configuration in your tests, and docker configuration when configuring services that run inside the docker-compose network and need to connect to each other.

## Functions

### func ClickHouse

```go
func ClickHouse(t *testing.T) string
```

ClickHouse returns ClickHouse database connection string for integration testing.

Returns a Data Source Name (DSN) configured for connecting from test runners to localhost:9000 with:

  - User: default (ClickHouse default user)
  - Password: password (matches docker-compose configuration)
  - Security disabled for testing (secure=false, skip\_verify=true)
  - Extended timeout for slower test environments (dial\_timeout=10s)

Example usage:

	dsn := containers.ClickHouse(t)
	conn, err := clickhouse.Open(&clickhouse.Options{
	    Addr: []string{dsn},
	})
	require.NoError(t, err)
	defer conn.Close()

### func ControlPlane

```go
func ControlPlane(t *testing.T) (string, string)
```

### func Kafka

```go
func Kafka(t *testing.T) []string
```

Kafka returns Kafka broker addresses for integration testing.

Returns broker addresses for connecting to the Kafka service running in docker-compose. Kafka is used for distributed cache invalidation and other event streaming scenarios.

The function returns host addresses (for connecting from test runners) that point to localhost:9092, matching the port mapping in docker-compose.yaml.

Example usage:

	brokers := containers.Kafka(t)
	// Use brokers with your Kafka client

### func MySQL

```go
func MySQL(t *testing.T) *mysql.Config
```

MySQL returns MySQL database configuration for integration testing.

Returns a configuration for connecting from test runners to localhost:3306. Uses standard credentials (unkey/password) with parse time enabled and logging disabled to reduce test output noise.

Database name is intentionally left empty - tests should create and use specific database names to avoid conflicts between test runs.

Example usage:

	cfg := containers.MySQL(t)
	cfg.DBName = "unkey"
	db, err := sql.Open("mysql", cfg.FormatDSN())
	require.NoError(t, err)
	defer db.Close()

### func StartAllServices

```go
func StartAllServices(t *testing.T)
```

StartAllServices is a no-op placeholder for service initialization.

This function exists for compatibility with testing frameworks that expect a service startup function, but does not actually start any services. Services are expected to be running via docker-compose before tests execute.

In a typical workflow:

 1. Start services: docker-compose up -d
 2. Run tests: go test ./...
 3. Stop services: docker-compose down

The function is safe to call multiple times and from multiple test functions.


## Types

### type OTELConfig

```go
type OTELConfig struct {
	// HTTPEndpoint is the OTEL collector HTTP endpoint for sending telemetry data.
	// Uses the standard OTEL HTTP port 4318 mapped to localhost.
	HTTPEndpoint string

	// GRPCEndpoint is the OTEL collector gRPC endpoint for sending telemetry data.
	// Uses the standard OTEL gRPC port 4317 mapped to localhost.
	GRPCEndpoint string

	// GrafanaURL is the Grafana dashboard URL for viewing telemetry data.
	// Useful for debugging and monitoring during integration tests.
	GrafanaURL string
}
```

OTELConfig holds OpenTelemetry service configuration for testing.

Provides endpoints for both HTTP and gRPC OTEL collectors, plus the Grafana dashboard URL for observability during testing. All endpoints use localhost with mapped ports for access from test runners.

#### func OTEL

```go
func OTEL(t *testing.T) OTELConfig
```

OTEL returns OpenTelemetry service configuration for integration testing.

Returns an \[OTELConfig] with all OTEL-related endpoints configured for localhost access. This includes both the OTEL collector endpoints (HTTP and gRPC) and the Grafana dashboard URL for observability during testing.

The configuration supports different OTEL export scenarios:

  - HTTP endpoint: For OTLP over HTTP (port 4318)
  - gRPC endpoint: For OTLP over gRPC (port 4317)
  - Grafana URL: For viewing traces and metrics (port 3000)

Example usage:

	otelConfig := containers.OTEL(t)
	exporter, err := otlptracehttp.New(ctx,
	    otlptracehttp.WithEndpoint(otelConfig.HTTPEndpoint),
	    otlptracehttp.WithInsecure(),
	)
	require.NoError(t, err)

### type S3Config

```go
type S3Config struct {
	// HostURL is the S3/MinIO endpoint accessible from the test runner.
	// Uses localhost with the mapped port for external connections.
	HostURL string

	// DockerURL is the S3/MinIO endpoint accessible from within docker-compose network.
	// Uses the service name for internal docker-compose service communication.
	DockerURL string

	// AccessKeyID is the S3 access key for authentication.
	// Matches the MinIO root user configured in docker-compose.
	AccessKeyID string

	// AccessKeySecret is the S3 secret key for authentication.
	// Matches the MinIO root password configured in docker-compose.
	AccessKeySecret string
}
```

S3Config holds S3/MinIO connection configuration for testing.

This configuration provides both host and docker URLs to support different connection scenarios in integration tests. The host URL is used when connecting from test runners, while the docker URL is used for services running inside the docker-compose network.

The access credentials are configured to match the default MinIO setup in the docker-compose configuration.

#### func S3

```go
func S3(t *testing.T) S3Config
```

S3 returns S3/MinIO configuration for integration testing.

Returns a complete \[S3Config] with endpoints and credentials configured for the MinIO service running in docker-compose. The configuration includes both host and docker URLs to support different connection scenarios.

Credentials are set to the default MinIO root user configuration:

  - Access Key: minio\_root\_user
  - Secret Key: minio\_root\_password

These credentials must match the MINIO\_ROOT\_USER and MINIO\_ROOT\_PASSWORD environment variables in your docker-compose.yaml.

Example usage:

	s3Config := containers.S3(t)
	client := minio.New(s3Config.HostURL, &minio.Options{
	    Creds: credentials.NewStaticV4(s3Config.AccessKeyID, s3Config.AccessKeySecret, ""),
	})

