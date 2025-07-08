package containers

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"

	"github.com/docker/go-connections/nat"
	"github.com/go-redis/redis/v8"
	mysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/compose"
)

// S3Config holds S3/MinIO connection configuration
type S3Config struct {
	HostURL         string
	DockerURL       string
	AccessKeyID     string
	AccessKeySecret string
}

// OTELConfig holds OTEL service configuration
type OTELConfig struct {
	HTTPEndpoint string
	GRPCEndpoint string
	GrafanaURL   string
}

var (
	composeInstance *compose.DockerCompose
	composeOnce     sync.Once
)

// getSharedCompose returns a shared docker-compose instance using singleton pattern
func getSharedCompose(t *testing.T) *compose.DockerCompose {
	composeOnce.Do(func() {
		_, currentFilePath, _, _ := runtime.Caller(0)
		
		// We're going from go/pkg/testutil/containers/ up to unkey/deployment/docker-compose.yaml
		composeFilePath := filepath.Join(filepath.Dir(currentFilePath), "../../../../deployment/docker-compose.yaml")
		
		composeStack, err := compose.NewDockerComposeWith(compose.WithStackFiles(composeFilePath), compose.StackIdentifier("go-test"))
		require.NoError(t, err)
		
		composeInstance = composeStack
	})
	return composeInstance
}

// MySQL returns MySQL configuration with dynamically discovered ports
func MySQL(t *testing.T) (hostCfg, dockerCfg *mysql.Config) {
	ctx := context.Background()
	dockerCompose := getSharedCompose(t)
	
	// Start MySQL service
	err := dockerCompose.Up(ctx, compose.Wait(true), compose.RunServices("mysql"))
	require.NoError(t, err)

	// Get the MySQL service container
	mysqlContainer, err := dockerCompose.ServiceContainer(ctx, "mysql")
	require.NoError(t, err)

	// Get the mapped port
	mysqlPort, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	require.NoError(t, err)

	// Get the container IP for docker network access
	containerIP, err := mysqlContainer.ContainerIP(ctx)
	require.NoError(t, err)

	// Host configuration (from test runner)
	hostCfg = mysql.NewConfig()
	hostCfg.User = "unkey"
	hostCfg.Passwd = "password"
	hostCfg.Net = "tcp"
	hostCfg.Addr = fmt.Sprintf("localhost:%s", mysqlPort.Port())
	hostCfg.DBName = ""
	hostCfg.ParseTime = true
	hostCfg.Logger = &mysql.NopLogger{}

	// Docker configuration (from within containers)
	dockerCfg = mysql.NewConfig()
	dockerCfg.User = "unkey"
	dockerCfg.Passwd = "password"
	dockerCfg.Net = "tcp"
	dockerCfg.Addr = fmt.Sprintf("%s:3306", containerIP)
	dockerCfg.DBName = ""
	dockerCfg.ParseTime = true
	dockerCfg.Logger = &mysql.NopLogger{}

	return hostCfg, dockerCfg
}

// Redis returns Redis client and connection strings with dynamically discovered ports
func Redis(t *testing.T) (client *redis.Client, hostAddr, dockerAddr string) {
	ctx := context.Background()
	dockerCompose := getSharedCompose(t)
	
	// Start Redis service
	err := dockerCompose.Up(ctx, compose.Wait(true), compose.RunServices("redis"))
	require.NoError(t, err)

	// Get the Redis service container
	redisContainer, err := dockerCompose.ServiceContainer(ctx, "redis")
	require.NoError(t, err)

	// Get the mapped port
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)

	// Get the container IP
	containerIP, err := redisContainer.ContainerIP(ctx)
	require.NoError(t, err)

	hostAddr = fmt.Sprintf("redis://localhost:%s", redisPort.Port())
	dockerAddr = fmt.Sprintf("redis://%s:6379", containerIP)

	// Create Redis client for host connection
	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("localhost:%s", redisPort.Port()),
	})

	return client, hostAddr, dockerAddr
}

// ClickHouse returns ClickHouse DSN strings with dynamically discovered ports
func ClickHouse(t *testing.T) (hostDsn, dockerDsn string) {
	ctx := context.Background()
	dockerCompose := getSharedCompose(t)
	
	// Start ClickHouse service
	err := dockerCompose.Up(ctx, compose.Wait(true), compose.RunServices("clickhouse"))
	require.NoError(t, err)

	// Get the ClickHouse service container
	chContainer, err := dockerCompose.ServiceContainer(ctx, "clickhouse")
	require.NoError(t, err)

	// Get the mapped port for native protocol (9000)
	clickhousePort, err := chContainer.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)

	// Get the container IP
	containerIP, err := chContainer.ContainerIP(ctx)
	require.NoError(t, err)

	hostDsn = fmt.Sprintf("clickhouse://default:password@localhost:%s?secure=false&skip_verify=true&dial_timeout=10s", clickhousePort.Port())
	dockerDsn = fmt.Sprintf("clickhouse://default:password@%s:9000?secure=false&skip_verify=true&dial_timeout=10s", containerIP)

	return hostDsn, dockerDsn
}

// S3 returns S3/MinIO configuration with dynamically discovered ports
func S3(t *testing.T) S3Config {
	ctx := context.Background()
	dockerCompose := getSharedCompose(t)
	
	// Start S3 service
	err := dockerCompose.Up(ctx, compose.Wait(true), compose.RunServices("s3"))
	require.NoError(t, err)

	// Get the S3 service container
	s3Container, err := dockerCompose.ServiceContainer(ctx, "s3")
	require.NoError(t, err)

	// Get the mapped port for API (3902)
	s3Port, err := s3Container.MappedPort(ctx, "3902/tcp")
	require.NoError(t, err)

	// Get the container IP
	containerIP, err := s3Container.ContainerIP(ctx)
	require.NoError(t, err)

	return S3Config{
		HostURL:         fmt.Sprintf("http://localhost:%s", s3Port.Port()),
		DockerURL:       fmt.Sprintf("http://%s:3902", containerIP),
		AccessKeyID:     "minio_root_user",
		AccessKeySecret: "minio_root_password",
	}
}

// OTEL returns OTEL endpoint configuration with dynamically discovered ports
func OTEL(t *testing.T) OTELConfig {
	ctx := context.Background()
	dockerCompose := getSharedCompose(t)
	
	// Start OTEL service
	err := dockerCompose.Up(ctx, compose.Wait(true), compose.RunServices("otel"))
	require.NoError(t, err)

	// Get the OTEL service container
	otelContainer, err := dockerCompose.ServiceContainer(ctx, "otel")
	require.NoError(t, err)

	// Get the mapped ports
	httpPort, err := otelContainer.MappedPort(ctx, "4318/tcp")
	require.NoError(t, err)

	grpcPort, err := otelContainer.MappedPort(ctx, "4317/tcp")
	require.NoError(t, err)

	grafanaPort, err := otelContainer.MappedPort(ctx, "3000/tcp")
	require.NoError(t, err)

	return OTELConfig{
		HTTPEndpoint: fmt.Sprintf("http://localhost:%s", httpPort.Port()),
		GRPCEndpoint: fmt.Sprintf("http://localhost:%s", grpcPort.Port()),
		GrafanaURL:   fmt.Sprintf("http://localhost:%s", grafanaPort.Port()),
	}
}

// GetServicePort returns the mapped port for any service
func GetServicePort(t *testing.T, serviceName, containerPort string) int {
	ctx := context.Background()
	dockerCompose := getSharedCompose(t)
	
	// Start the specific service
	err := dockerCompose.Up(ctx, compose.Wait(true), compose.RunServices(serviceName))
	require.NoError(t, err)

	container, err := dockerCompose.ServiceContainer(ctx, serviceName)
	require.NoError(t, err)

	mappedPort, err := container.MappedPort(ctx, nat.Port(containerPort))
	require.NoError(t, err)

	port, err := strconv.Atoi(mappedPort.Port())
	require.NoError(t, err)

	return port
}
