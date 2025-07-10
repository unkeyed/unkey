package testutil

import (
	"github.com/go-redis/redis/v8"
	mysql "github.com/go-sql-driver/mysql"
)

// TestServices provides static configuration for docker-compose managed test services
// These match the port mappings in deployment/docker-compose.yaml
type TestServices struct{}

// MySQL returns MySQL configuration for connecting to the docker-compose MySQL service
func (TestServices) MySQL() (hostCfg, dockerCfg *mysql.Config) {
	// Host configuration (from test runner)
	hostCfg = mysql.NewConfig()
	hostCfg.User = "unkey"
	hostCfg.Passwd = "password"
	hostCfg.Net = "tcp"
	hostCfg.Addr = "localhost:3306"
	hostCfg.DBName = "" // Explicitly no database name in base config
	hostCfg.ParseTime = true
	hostCfg.Logger = &mysql.NopLogger{}

	// Docker configuration (from within containers)
	dockerCfg = mysql.NewConfig()
	dockerCfg.User = "unkey"
	dockerCfg.Passwd = "password"
	dockerCfg.Net = "tcp"
	dockerCfg.Addr = "mysql:3306"
	dockerCfg.DBName = "" // Explicitly no database name in base config
	dockerCfg.ParseTime = true
	dockerCfg.Logger = &mysql.NopLogger{}

	return hostCfg, dockerCfg
}

// Redis returns Redis client and connection strings for the docker-compose Redis service
func (TestServices) Redis() (client *redis.Client, hostAddr, dockerAddr string) {
	hostAddr = "redis://localhost:6379"
	dockerAddr = "redis://redis:6379"

	// Create Redis client for host connection
	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	return client, hostAddr, dockerAddr
}

// ClickHouse returns ClickHouse DSN strings for the docker-compose ClickHouse service
func (TestServices) ClickHouse() (hostDsn, dockerDsn string) {
	hostDsn = "clickhouse://default:password@localhost:9000?secure=false&skip_verify=true&dial_timeout=10s"
	dockerDsn = "clickhouse://default:password@clickhouse:9000?secure=false&skip_verify=true&dial_timeout=10s"
	return hostDsn, dockerDsn
}

// S3 returns S3/MinIO configuration for the docker-compose MinIO service
func (TestServices) S3() S3Config {
	return S3Config{
		HostURL:         "http://localhost:3902",
		DockerURL:       "http://s3:3902",
		AccessKeyID:     "minio_root_user",
		AccessKeySecret: "minio_root_password",
	}
}

// S3Config holds S3/MinIO connection configuration
type S3Config struct {
	HostURL         string
	DockerURL       string
	AccessKeyID     string
	AccessKeySecret string
}

// OTEL returns OTEL endpoint configuration for the docker-compose OTEL service
func (TestServices) OTEL() OTELConfig {
	return OTELConfig{
		HTTPEndpoint: "http://localhost:4318",
		GRPCEndpoint: "http://localhost:4317",
		GrafanaURL:   "http://localhost:3000",
	}
}

// OTELConfig holds OTEL service configuration
type OTELConfig struct {
	HTTPEndpoint string
	GRPCEndpoint string
	GrafanaURL   string
}

// NewTestServices returns a TestServices instance
// Requires docker-compose services to be running (via 'make start-test-services')
func NewTestServices() *TestServices {
	return &TestServices{}
}
