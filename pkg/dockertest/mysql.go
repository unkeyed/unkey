package dockertest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

const (
	mysqlImage    = "mysql:9.4.0"
	mysqlPort     = "3306/tcp"
	mysqlUser     = "unkey"
	mysqlPassword = "password"
	mysqlDatabase = "unkey"
)

// MySQLConfig holds connection information for a MySQL test container.
type MySQLConfig struct {
	// DSN is the host DSN for connecting from the test runner.
	DSN string
	// DockerDSN is the DSN for connecting from containers on the docker network.
	DockerDSN string
}

// MySQLOpt configures the MySQL test container.
type MySQLOpt func(*containerConfig)

// WithDiskStorage disables tmpfs so MySQL writes to real disk.
// Use this for large-scale performance tests that exceed the default 256MB tmpfs.
func WithDiskStorage() MySQLOpt {
	return func(cfg *containerConfig) {
		cfg.Tmpfs = nil
		// Larger buffer pool for big datasets
		for i, arg := range cfg.Cmd {
			if arg == "--innodb-buffer-pool-size=32M" {
				cfg.Cmd[i] = "--innodb-buffer-pool-size=512M"
			}
		}
	}
}

// MySQL starts the local MySQL test container and returns DSNs.
//
// The container is based on the local dev image with preloaded schema.
// This function blocks until the MySQL port is accepting TCP connections
// (up to 60s). Fails the test if Docker is unavailable or the container fails to start.
func MySQL(t *testing.T, opts ...MySQLOpt) MySQLConfig {
	t.Helper()

	containerStart := time.Now()

	cfg := containerConfig{
		Image:        mysqlImage,
		ExposedPorts: []string{mysqlPort},
		WaitStrategy: NewTCPWait(mysqlPort),
		WaitTimeout:  60 * time.Second,
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": mysqlPassword,
			"MYSQL_DATABASE":      mysqlDatabase,
			"MYSQL_USER":          mysqlUser,
			"MYSQL_PASSWORD":      mysqlPassword,
		},
		Cmd: []string{
			// Disable binary logging (not needed for tests)
			"--skip-log-bin",
			"--disable-log-bin",
			// Disable durability for faster writes (crash safety not needed in tests)
			"--innodb-doublewrite=0",
			"--innodb-flush-log-at-trx-commit=0",
			"--innodb-flush-method=nosync",
			// Reduce buffer sizes for faster startup
			"--innodb-buffer-pool-size=32M",
			"--innodb-log-buffer-size=1M",
			// Disable performance schema (overhead not needed for tests)
			"--performance-schema=OFF",
			// Skip name resolution for faster connections
			"--skip-name-resolve",
		},
		Tmpfs: map[string]string{
			"/var/lib/mysql": "rw,noexec,nosuid,size=256m",
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	ctr := startContainer(t, cfg)
	t.Logf("  MySQL container started in %s", time.Since(containerStart))

	port := ctr.Port(mysqlPort)
	addr := fmt.Sprintf("%s:%s", ctr.Host, port)

	hostCfg := mysql.NewConfig()
	hostCfg.User = mysqlUser
	hostCfg.Passwd = mysqlPassword
	hostCfg.Net = "tcp"
	hostCfg.Addr = addr
	hostCfg.DBName = mysqlDatabase
	hostCfg.ParseTime = true
	hostCfg.MultiStatements = true
	hostCfg.Logger = &mysql.NopLogger{}

	dockerCfg := mysql.NewConfig()
	dockerCfg.User = mysqlUser
	dockerCfg.Passwd = mysqlPassword
	dockerCfg.Net = "tcp"
	dockerCfg.Addr = "mysql:3306"
	dockerCfg.DBName = mysqlDatabase
	dockerCfg.ParseTime = true
	dockerCfg.Logger = &mysql.NopLogger{}

	pingStart := time.Now()
	hostDB, err := sql.Open("mysql", hostCfg.FormatDSN())
	require.NoError(t, err)
	defer func() { require.NoError(t, hostDB.Close()) }()
	require.Eventually(t, func() bool {
		pingErr := hostDB.PingContext(context.Background())
		return pingErr == nil
	}, 60*time.Second, 500*time.Millisecond)
	t.Logf("  MySQL ready for connections in %s", time.Since(pingStart))

	schemaStart := time.Now()
	schemaDir := schemaPath()
	entries, err := os.ReadDir(schemaDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(schemaDir, entry.Name()))
		require.NoError(t, readErr)
		_, execErr := hostDB.ExecContext(context.Background(), string(data))
		require.NoError(t, execErr, "failed to apply %s", entry.Name())
	}
	t.Logf("  MySQL schema loaded in %s", time.Since(schemaStart))

	return MySQLConfig{
		DSN:       hostCfg.FormatDSN(),
		DockerDSN: dockerCfg.FormatDSN(),
	}
}

func schemaPath() string {
	if runfiles := os.Getenv("TEST_SRCDIR"); runfiles != "" {
		workspace := os.Getenv("TEST_WORKSPACE")
		if workspace != "" {
			candidate := filepath.Join(runfiles, workspace, "pkg", "mysql", "schema")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		candidate := filepath.Join(runfiles, "_main", "pkg", "mysql", "schema")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Dir(filepath.Dir(currentFile))
	return filepath.Join(root, "mysql", "schema")
}
