---
title: dockertest
description: "provides lightweight Docker container management for integration tests"
---

Package dockertest provides lightweight Docker container management for integration tests.

This package uses the Docker SDK directly to spin up containers dynamically, avoiding the overhead of external tools like docker-compose or testcontainers. Containers are automatically cleaned up when tests complete via t.Cleanup.

### Requirements

Tests using this package MUST be run via Bazel:

	bazel test //pkg/dockertest:dockertest_test

Running via \`go test\` directly is not supported and will fail with an error.

### Key Types

The main entry point is through service functions like \[Redis], which start containers and return connection information. For lower-level access, \[Container] provides metadata about running containers including port mappings.

Container readiness is determined by \[WaitStrategy] implementations. The package provides \[TCPWait] for TCP port-based health checks, created via \[NewTCPWait].

### Usage

Each service function starts a container and returns connection information:

	func TestRedisIntegration(t *testing.T) {
	    redisURL := dockertest.Redis(t)
	    // redisURL is "redis://localhost:{randomPort}"
	    // Container is automatically removed when test completes
	}

### Design

Containers are created per-test for isolation. Each container:

  - Uses a random host port to avoid conflicts
  - Waits for the service to be ready before returning
  - Is automatically removed via t.Cleanup

### Available Services

Currently supported:

  - \[MySQL]: MySQL with dev schema preloaded
  - \[Redis]: Redis 8.0 container
  - \[S3]: MinIO S3-compatible object storage
  - \[Restate]: Restate server (ingress + admin)

## Constants

```go
const (
	// Use same image as docker-compose for compatibility
	clickhouseImage    = "bitnamilegacy/clickhouse:25.6.4"
	clickhousePort     = "9000/tcp"
	clickhouseHTTPPort = "8123/tcp"
	clickhouseUser     = "default"
	clickhousePassword = "password"
)
```

```go
const (
	mysqlImage    = "mysql:9.4.0"
	mysqlPort     = "3306/tcp"
	mysqlUser     = "unkey"
	mysqlPassword = "password"
	mysqlDatabase = "unkey"
)
```

```go
const (
	redisImage = "redis:8.0"
	redisPort  = "6379/tcp"
)
```

```go
const (
	restateImage     = "restatedev/restate:1.6.0"
	restatePort      = "8080/tcp"
	restateAdminPort = "9070/tcp"
)
```

```go
const (
	minioImage = "quay.io/minio/minio:latest"
	minioPort  = "9000/tcp"

	// Default MinIO credentials used for test containers.
	minioAccessKey = "minioadmin"
	minioSecretKey = "minioadmin"
)
```


## Variables

```go
var (
	dockerClient     *client.Client
	dockerClientOnce sync.Once
)
```


## Functions

### func Redis

```go
func Redis(t *testing.T) string
```

Redis starts a Redis 8.0 container and returns the connection URL.

The returned URL is in the format "redis://localhost:{port}" and can be used directly with most Redis client libraries. The container is automatically removed when the test completes via t.Cleanup.

This function blocks until Redis is accepting TCP connections (up to 30s). Fails the test if Docker is unavailable or the container fails to start.


## Types

### type ClickHouseConfig

```go
type ClickHouseConfig struct {
	// DSN is the connection string for connecting from the test runner.
	DSN string
}
```

ClickHouseConfig holds connection information for a ClickHouse test container.

#### func ClickHouse

```go
func ClickHouse(t *testing.T) ClickHouseConfig
```

ClickHouse starts a ClickHouse container and applies the schema.

The container is started with a random available port and the schema is loaded from pkg/clickhouse/schema/\*.sql files in order. This function blocks until ClickHouse is accepting connections and the schema has been applied. Fails the test if Docker is unavailable or the container fails to start.

### type Container

```go
type Container struct {
	// ID is the Docker container ID (64-character hex string).
	ID string

	// Host is the hostname to connect to (typically "localhost").
	Host string

	// Ports maps container ports to host ports.
	// Key is the container port (e.g., "6379/tcp"), value is the host port.
	Ports map[string]string
}
```

Container holds metadata about a running Docker container.

Use the \[Container.Port] method to retrieve the host port mapped to a specific container port.

#### func (Container) HostURL

```go
func (c *Container) HostURL(scheme, containerPort string) string
```

HostURL returns a URL for the container using the provided scheme and port. The containerPort should be in the format "port/protocol" (e.g., "8080/tcp").

#### func (Container) Port

```go
func (c *Container) Port(containerPort string) string
```

Port returns the mapped host port for a given container port. The containerPort should be in the format "port/protocol" (e.g., "6379/tcp"). Returns an empty string if the port is not mapped.

### type HTTPWait

```go
type HTTPWait struct {
	// Port is the container port to connect to (e.g., "9000/tcp").
	Port string

	// Path is the HTTP path to request (e.g., "/minio/health/live").
	Path string

	// ExpectedStatus is the HTTP status code that indicates readiness.
	// Defaults to 200 if zero.
	ExpectedStatus int

	// PollInterval is how often to attempt the request. Defaults to 100ms if zero.
	PollInterval time.Duration
}
```

HTTPWait waits for an HTTP endpoint to return an expected status code.

This strategy is useful for services that expose health check endpoints, such as MinIO's /minio/health/live endpoint. Unlike \[TCPWait], this verifies that the application is actually responding to HTTP requests, not just accepting TCP connections.

#### func NewHTTPWait

```go
func NewHTTPWait(port, path string) *HTTPWait
```

NewHTTPWait creates an \[HTTPWait] strategy for the given port and path. The port should be in the format "port/protocol" (e.g., "9000/tcp"). The path should include the leading slash (e.g., "/health").

#### func (HTTPWait) Wait

```go
func (w *HTTPWait) Wait(t *testing.T, c *Container, timeout time.Duration)
```

Wait polls the HTTP endpoint until it returns the expected status code or the timeout expires. Fails the test if the endpoint does not become ready.

### type MySQLConfig

```go
type MySQLConfig struct {
	// DSN is the host DSN for connecting from the test runner.
	DSN string
	// DockerDSN is the DSN for connecting from containers on the docker network.
	DockerDSN string
}
```

MySQLConfig holds connection information for a MySQL test container.

#### func MySQL

```go
func MySQL(t *testing.T) MySQLConfig
```

MySQL starts the local MySQL test container and returns DSNs.

The container is based on the local dev image with preloaded schema. This function blocks until the MySQL port is accepting TCP connections (up to 60s). Fails the test if Docker is unavailable or the container fails to start.

### type RestateConfig

```go
type RestateConfig struct {
	// IngressURL is the Restate ingress endpoint URL.
	IngressURL string
	// AdminURL is the Restate admin endpoint URL.
	AdminURL string
}
```

RestateConfig holds connection information for a Restate container.

#### func Restate

```go
func Restate(t *testing.T) RestateConfig
```

Restate starts a Restate container and returns ingress/admin URLs.

The container is automatically removed when the test completes via t.Cleanup. This function blocks until the admin health endpoint responds (up to 30s). Fails the test if Docker is unavailable or the container fails to start.

### type S3Config

```go
type S3Config struct {
	// URL is the S3 endpoint URL (e.g., "http://localhost:54321").
	URL string

	// AccessKeyID is the access key for authentication.
	AccessKeyID string

	// SecretAccessKey is the secret key for authentication.
	SecretAccessKey string
}
```

S3Config holds connection information for an S3-compatible container.

The returned configuration can be used directly with AWS SDK, MinIO client, or any S3-compatible client library. Credentials are set to MinIO defaults.

#### func S3

```go
func S3(t *testing.T) S3Config
```

S3 starts a MinIO container and returns the connection configuration.

MinIO is an S3-compatible object storage server. The container is configured with default credentials (minioadmin/minioadmin) and a single server instance suitable for testing.

The container is automatically removed when the test completes via t.Cleanup. This function blocks until MinIO's health endpoint responds (up to 30s). Fails the test if Docker is unavailable or the container fails to start.

Example usage:

	func TestS3Integration(t *testing.T) {
	    s3 := dockertest.S3(t)
	    client, err := minio.New(s3.URL, &minio.Options{
	        Creds: credentials.NewStaticV4(s3.AccessKeyID, s3.SecretAccessKey, ""),
	    })
	    require.NoError(t, err)
	    // Use client...
	}

### type TCPWait

```go
type TCPWait struct {
	// Port is the container port to wait for (e.g., "6379/tcp").
	Port string

	// PollInterval is how often to attempt connection. Defaults to 100ms if zero.
	PollInterval time.Duration
}
```

TCPWait waits for a TCP port to accept connections.

This is the simplest readiness check and works for most services that accept TCP connections (Redis, MySQL, PostgreSQL, etc.). For services that need application-level health checks, implement a custom \[WaitStrategy].

#### func NewTCPWait

```go
func NewTCPWait(port string) *TCPWait
```

NewTCPWait creates a \[TCPWait] strategy for the given container port. The port should be in the format "port/protocol" (e.g., "6379/tcp").

#### func (TCPWait) Wait

```go
func (w *TCPWait) Wait(t *testing.T, c *Container, timeout time.Duration)
```

Wait polls the TCP port until it accepts connections or the timeout expires. Fails the test if the port is not mapped or does not become ready in time.

### type WaitStrategy

```go
type WaitStrategy interface {
	// Wait blocks until the container is ready or the timeout expires.
	// Fails the test if the container does not become ready in time.
	Wait(t *testing.T, c *Container, timeout time.Duration)
}
```

WaitStrategy defines how to wait for a container to become ready.

Implementations should poll the container until it is ready to accept connections or perform its intended function. If readiness cannot be established within the timeout, the implementation should fail the test.

