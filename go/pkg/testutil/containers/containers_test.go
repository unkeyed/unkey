package containers

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestContainersIntegration validates that testcontainers can start services and they're accessible
func TestContainersIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// No need for container manager anymore

	t.Run("MySQL", func(t *testing.T) {
		// Test MySQL connection
		mysqlCfg, _ := MySQL(t)
		require.NotEmpty(t, mysqlCfg.Addr)

		// Try to connect
		mysqlCfg.DBName = "mysql" // Use built-in mysql database
		dsn := mysqlCfg.FormatDSN()
		db, err := sql.Open("mysql", dsn)
		require.NoError(t, err)
		defer db.Close()

		// Test connection
		err = db.Ping()
		require.NoError(t, err)

		t.Logf("MySQL connected successfully at: %s", mysqlCfg.Addr)
	})

	t.Run("Redis", func(t *testing.T) {
		// Test Redis connection
		client, hostAddr, dockerAddr := Redis(t)
		require.NotEmpty(t, hostAddr)
		require.NotEmpty(t, dockerAddr)
		defer client.Close()

		// Test connection
		ctx := context.Background()
		err = client.Ping(ctx).Err()
		require.NoError(t, err)

		// Test basic operations
		err = client.Set(ctx, "test_key", "test_value", 0).Err()
		require.NoError(t, err)

		val, err := client.Get(ctx, "test_key").Result()
		require.NoError(t, err)
		require.Equal(t, "test_value", val)

		t.Logf("Redis connected successfully at: %s", hostAddr)
	})

	t.Run("ClickHouse", func(t *testing.T) {
		// Test ClickHouse connection
		hostDsn, dockerDsn := ClickHouse(t)
		require.NotEmpty(t, hostDsn)
		require.NotEmpty(t, dockerDsn)

		t.Logf("ClickHouse available at: %s", hostDsn)
		t.Logf("ClickHouse docker DSN: %s", dockerDsn)

		// Note: We could test actual ClickHouse connection here,
		// but it requires the ClickHouse Go driver which might not be
		// available in this simple test context
	})

	t.Run("S3", func(t *testing.T) {
		// Test S3 connection
		s3Config := S3(t)
		require.NotEmpty(t, s3Config.HostURL)
		require.NotEmpty(t, s3Config.DockerURL)
		require.NotEmpty(t, s3Config.AccessKeyID)
		require.NotEmpty(t, s3Config.AccessKeySecret)

		t.Logf("S3 available at: %s", s3Config.HostURL)
		t.Logf("S3 docker URL: %s", s3Config.DockerURL)
	})

	t.Run("OTEL", func(t *testing.T) {
		// Test OTEL connection
		otelConfig := OTEL(t)
		require.NotEmpty(t, otelConfig.HTTPEndpoint)
		require.NotEmpty(t, otelConfig.GRPCEndpoint)
		require.NotEmpty(t, otelConfig.GrafanaURL)

		t.Logf("OTEL HTTP endpoint: %s", otelConfig.HTTPEndpoint)
		t.Logf("OTEL GRPC endpoint: %s", otelConfig.GRPCEndpoint)
		t.Logf("Grafana available at: %s", otelConfig.GrafanaURL)
	})

	t.Run("ServicePortLookup", func(t *testing.T) {
		// Test generic port lookup
		mysqlPort := GetServicePort(t, "mysql", "3306/tcp")
		require.Greater(t, mysqlPort, 0)

		redisPort := GetServicePort(t, "redis", "6379/tcp")
		require.Greater(t, redisPort, 0)

		t.Logf("MySQL port: %d, Redis port: %d", mysqlPort, redisPort)

		// Ports should be different (unless extremely unlucky)
		require.NotEqual(t, mysqlPort, redisPort)
	})
}
