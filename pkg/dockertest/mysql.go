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
	mysqlImage        = "mysql:9.4.0"
	mysqlPort         = "3306/tcp"
	mysqlUser         = "unkey"
	mysqlPassword     = "password"
	mysqlRootPassword = "password"
)

// MySQLConfig holds connection information for a MySQL test container.
type MySQLConfig struct {
	// DSN is the host DSN for connecting from the test runner.
	DSN string
	// DockerDSN is the DSN for connecting from containers on the docker network.
	DockerDSN string
}

var mysqlCtr shared

// mysqlContainerConfig returns the container configuration for MySQL.
func mysqlContainerConfig() containerConfig {
	return containerConfig{
		Image:        mysqlImage,
		ExposedPorts: []string{mysqlPort},
		WaitStrategy: NewTCPWait(mysqlPort),
		WaitTimeout:  60 * time.Second,
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": mysqlRootPassword,
			"MYSQL_USER":          mysqlUser,
			"MYSQL_PASSWORD":      mysqlPassword,
			// No MYSQL_DATABASE — we create ephemeral databases per test.
		},
		Cmd: []string{
			"--skip-log-bin",
			"--disable-log-bin",
			"--innodb-doublewrite=0",
			"--innodb-flush-log-at-trx-commit=0",
			"--innodb-flush-method=nosync",
			"--innodb-buffer-pool-size=32M",
			"--innodb-log-buffer-size=1M",
			"--performance-schema=OFF",
			"--skip-name-resolve",
		},
		Tmpfs: map[string]string{
			"/var/lib/mysql": "rw,noexec,nosuid,size=256m",
		},
		SkipCleanup: false,
	}
}

// MySQL starts (or reuses) a shared MySQL container and returns a fresh ephemeral
// database for this test. The database is dropped when the test completes.
//
// The container starts on the first call in the process and is reused by all
// subsequent calls. Each call creates a unique database with loaded schema,
// ensuring complete isolation between tests.
func MySQL(t *testing.T) MySQLConfig {
	t.Helper()

	containerStart := time.Now()
	ctr := mysqlCtr.get(t, mysqlContainerConfig())
	t.Logf("  MySQL container ready in %s", time.Since(containerStart))

	port := ctr.Port(mysqlPort)
	addr := fmt.Sprintf("%s:%s", ctr.Host, port)

	// Create a unique database name for this test.
	dbName := fmt.Sprintf("test_%d", time.Now().UnixNano())

	// Connect as root to create the ephemeral database and grant privileges.
	rootCfg := mysql.NewConfig()
	rootCfg.User = "root"
	rootCfg.Passwd = mysqlRootPassword
	rootCfg.Net = "tcp"
	rootCfg.Addr = addr
	rootCfg.ParseTime = true
	rootCfg.MultiStatements = true
	rootCfg.Logger = &mysql.NopLogger{}

	rootDB, err := sql.Open("mysql", rootCfg.FormatDSN())
	require.NoError(t, err)
	defer func() { require.NoError(t, rootDB.Close()) }()

	// Wait for MySQL to be ready for connections.
	pingStart := time.Now()
	require.Eventually(t, func() bool {
		return rootDB.PingContext(context.Background()) == nil
	}, 60*time.Second, 500*time.Millisecond)
	t.Logf("  MySQL ready for connections in %s", time.Since(pingStart))

	ctx := context.Background()

	// Create the ephemeral database and grant access.
	_, err = rootDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`", dbName))
	require.NoError(t, err)
	_, err = rootDB.ExecContext(ctx, fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%'", dbName, mysqlUser))
	require.NoError(t, err)

	// Load schema into the new database.
	schemaStart := time.Now()
	schemaCfg := mysql.NewConfig()
	schemaCfg.User = "root"
	schemaCfg.Passwd = mysqlRootPassword
	schemaCfg.Net = "tcp"
	schemaCfg.Addr = addr
	schemaCfg.DBName = dbName
	schemaCfg.ParseTime = true
	schemaCfg.MultiStatements = true
	schemaCfg.Logger = &mysql.NopLogger{}

	schemaDB, err := sql.Open("mysql", schemaCfg.FormatDSN())
	require.NoError(t, err)
	defer func() { require.NoError(t, schemaDB.Close()) }()

	schemaPath := schemaSQLPath()
	schemaBytes, err := os.ReadFile(schemaPath)
	require.NoError(t, err)
	_, err = schemaDB.ExecContext(ctx, string(schemaBytes))
	require.NoError(t, err)
	t.Logf("  MySQL schema loaded in %s", time.Since(schemaStart))

	// Clean up: drop the database when the test finishes.
	t.Cleanup(func() {
		cleanupDB, cleanupErr := sql.Open("mysql", rootCfg.FormatDSN())
		if cleanupErr != nil {
			t.Logf("  MySQL cleanup: failed to connect: %v", cleanupErr)
			return
		}
		defer cleanupDB.Close() //nolint:errcheck
		_, cleanupErr = cleanupDB.ExecContext(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName))
		if cleanupErr != nil {
			t.Logf("  MySQL cleanup: failed to drop database %s: %v", dbName, cleanupErr)
		}
	})

	// Build DSNs for the caller.
	hostCfg := mysql.NewConfig()
	hostCfg.User = mysqlUser
	hostCfg.Passwd = mysqlPassword
	hostCfg.Net = "tcp"
	hostCfg.Addr = addr
	hostCfg.DBName = dbName
	hostCfg.ParseTime = true
	hostCfg.MultiStatements = true
	hostCfg.Logger = &mysql.NopLogger{}

	dockerCfg := mysql.NewConfig()
	dockerCfg.User = mysqlUser
	dockerCfg.Passwd = mysqlPassword
	dockerCfg.Net = "tcp"
	dockerCfg.Addr = "mysql:3306"
	dockerCfg.DBName = dbName
	dockerCfg.ParseTime = true
	dockerCfg.Logger = &mysql.NopLogger{}

	return MySQLConfig{
		DSN:       hostCfg.FormatDSN(),
		DockerDSN: dockerCfg.FormatDSN(),
	}
}

func schemaSQLPath() string {
	if runfiles := os.Getenv("TEST_SRCDIR"); runfiles != "" {
		workspace := os.Getenv("TEST_WORKSPACE")
		if workspace != "" {
			candidate := filepath.Join(runfiles, workspace, "pkg", "db", "schema.sql")
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
		candidate := filepath.Join(runfiles, "_main", "pkg", "db", "schema.sql")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	root := filepath.Dir(filepath.Dir(currentFile))
	return filepath.Join(root, "db", "schema.sql")
}
