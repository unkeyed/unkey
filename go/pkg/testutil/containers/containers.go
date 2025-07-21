package containers

import (
	"testing"

	mysql "github.com/go-sql-driver/mysql"
)

// S3Config holds S3/MinIO connection configuration for testing.
//
// This configuration provides both host and docker URLs to support different
// connection scenarios in integration tests. The host URL is used when connecting
// from test runners, while the docker URL is used for services running inside
// the docker-compose network.
//
// The access credentials are configured to match the default MinIO setup
// in the docker-compose configuration.
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

// OTELConfig holds OpenTelemetry service configuration for testing.
//
// Provides endpoints for both HTTP and gRPC OTEL collectors, plus the Grafana
// dashboard URL for observability during testing. All endpoints use localhost
// with mapped ports for access from test runners.
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

// StartAllServices is a no-op placeholder for service initialization.
//
// This function exists for compatibility with testing frameworks that expect
// a service startup function, but does not actually start any services.
// Services are expected to be running via docker-compose before tests execute.
//
// In a typical workflow:
//  1. Start services: docker-compose up -d
//  2. Run tests: go test ./...
//  3. Stop services: docker-compose down
//
// The function is safe to call multiple times and from multiple test functions.
func StartAllServices(t *testing.T) {
	// Services are managed externally via docker-compose.
	// This is intentionally a no-op.
}

// MySQL returns MySQL database configuration for integration testing.
//
// Returns a configuration for connecting from test runners to localhost:3306.
// Uses standard credentials (unkey/password) with parse time enabled and
// logging disabled to reduce test output noise.
//
// Database name is intentionally left empty - tests should create and use
// specific database names to avoid conflicts between test runs.
//
// Example usage:
//
//	cfg := containers.MySQL(t)
//	cfg.DBName = "unkey"
//	db, err := sql.Open("mysql", cfg.FormatDSN())
//	require.NoError(t, err)
//	defer db.Close()
func MySQL(t *testing.T) *mysql.Config {
	cfg := mysql.NewConfig()
	cfg.User = "unkey"
	cfg.Passwd = "password"
	cfg.Net = "tcp"
	cfg.Addr = "localhost:3306"
	cfg.DBName = ""
	cfg.ParseTime = true
	cfg.Logger = &mysql.NopLogger{}

	return cfg
}

// Redis returns Redis connection URL for integration testing.
//
// Returns a Redis URL configured for connecting from test runners to localhost:6379.
//
// Example usage:
//
//	redisURL := containers.Redis(t)
//	// Use redisURL with your Redis client
func Redis(t *testing.T) string {
	return "redis://localhost:6379"
}

// ClickHouse returns ClickHouse database connection string for integration testing.
//
// Returns a Data Source Name (DSN) configured for connecting from test runners
// to localhost:9000 with:
//   - User: default (ClickHouse default user)
//   - Password: password (matches docker-compose configuration)
//   - Security disabled for testing (secure=false, skip_verify=true)
//   - Extended timeout for slower test environments (dial_timeout=10s)
//
// Example usage:
//
//	dsn := containers.ClickHouse(t)
//	conn, err := clickhouse.Open(&clickhouse.Options{
//	    Addr: []string{dsn},
//	})
//	require.NoError(t, err)
//	defer conn.Close()
func ClickHouse(t *testing.T) string {
	return "clickhouse://default:password@localhost:9000?secure=false&skip_verify=true&dial_timeout=10s"
}

// S3 returns S3/MinIO configuration for integration testing.
//
// Returns a complete [S3Config] with endpoints and credentials configured
// for the MinIO service running in docker-compose. The configuration includes
// both host and docker URLs to support different connection scenarios.
//
// Credentials are set to the default MinIO root user configuration:
//   - Access Key: minio_root_user
//   - Secret Key: minio_root_password
//
// These credentials must match the MINIO_ROOT_USER and MINIO_ROOT_PASSWORD
// environment variables in your docker-compose.yaml.
//
// Example usage:
//
//	s3Config := containers.S3(t)
//	client := minio.New(s3Config.HostURL, &minio.Options{
//	    Creds: credentials.NewStaticV4(s3Config.AccessKeyID, s3Config.AccessKeySecret, ""),
//	})
func S3(t *testing.T) S3Config {
	return S3Config{
		HostURL:         "http://localhost:3902",
		DockerURL:       "http://s3:3902",
		AccessKeyID:     "minio_root_user",
		AccessKeySecret: "minio_root_password",
	}
}

// OTEL returns OpenTelemetry service configuration for integration testing.
//
// Returns an [OTELConfig] with all OTEL-related endpoints configured for
// localhost access. This includes both the OTEL collector endpoints (HTTP and gRPC)
// and the Grafana dashboard URL for observability during testing.
//
// The configuration supports different OTEL export scenarios:
//   - HTTP endpoint: For OTLP over HTTP (port 4318)
//   - gRPC endpoint: For OTLP over gRPC (port 4317)
//   - Grafana URL: For viewing traces and metrics (port 3000)
//
// Example usage:
//
//	otelConfig := containers.OTEL(t)
//	exporter, err := otlptracehttp.New(ctx,
//	    otlptracehttp.WithEndpoint(otelConfig.HTTPEndpoint),
//	    otlptracehttp.WithInsecure(),
//	)
//	require.NoError(t, err)
func OTEL(t *testing.T) OTELConfig {
	return OTELConfig{
		HTTPEndpoint: "http://localhost:4318",
		GRPCEndpoint: "http://localhost:4317",
		GrafanaURL:   "http://localhost:3000",
	}
}

// Kafka returns Kafka broker addresses for integration testing.
//
// Returns broker addresses for connecting to the Kafka service running
// in docker-compose. Kafka is used for distributed cache invalidation
// and other event streaming scenarios.
//
// The function returns host addresses (for connecting from test runners) that
// point to localhost:9092, matching the port mapping in docker-compose.yaml.
//
// Example usage:
//
//	brokers := containers.Kafka(t)
//	// Use brokers with your Kafka client
func Kafka(t *testing.T) []string {
	return []string{"127.0.0.1:9092"}
}
