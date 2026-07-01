package containers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

const (
	mysqlImage             = "mysql:9.4.0"
	mysqlPort              = "3306/tcp"
	mysqlUser              = "unkey"
	mysqlPassword          = "password"
	mysqlDatabase          = "unkey"
	mysqlSchemaLockName    = "unkey_test_mysql_schema"
	mysqlSchemaMarkerTable = "_unkey_test_schema"
)

type mysqlSchemaDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// MySQLConfig holds connection information for a MySQL test container.
type MySQLConfig struct {
	// DSN is the host DSN for connecting from the test runner.
	DSN string
}

// MySQLOpt configures the MySQL test container.
type MySQLOpt = Opt

// WithDiskStorage disables tmpfs so MySQL writes to real disk.
// Use this for large-scale performance tests that exceed the default 256MB tmpfs.
func WithDiskStorage() MySQLOpt {
	return func(cfg *containerConfig) {
		cfg.Tmpfs = nil
	}
}

// MySQL starts a MySQL container and returns connection info.
//
// The container is reused by stable Docker name across Bazel test processes.
func MySQL(t testing.TB, opts ...MySQLOpt) MySQLConfig {
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
			"--innodb-buffer-pool-size=512M",
			"--innodb-log-buffer-size=1M",
			// Disable performance schema (overhead not needed for tests)
			"--performance-schema=OFF",
			// Skip name resolution for faster connections
			"--skip-name-resolve",
		},
		Tmpfs: map[string]string{
			"/var/lib/mysql": "rw,noexec,nosuid,size=256m",
		},
		Dedicated: false,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	ctr := startContainer(t, cfg)
	t.Logf("  MySQL container started in %s", time.Since(containerStart))

	port := ctr.Port(mysqlPort)
	addr := fmt.Sprintf("%s:%s", ctr.Host, port)

	dsnCfg := mysql.NewConfig()
	dsnCfg.User = mysqlUser
	dsnCfg.Passwd = mysqlPassword
	dsnCfg.Net = "tcp"
	dsnCfg.Addr = addr
	dsnCfg.DBName = mysqlDatabase
	dsnCfg.ParseTime = true
	dsnCfg.MultiStatements = true
	dsnCfg.Logger = &mysql.NopLogger{}

	pingStart := time.Now()
	hostDB, err := sql.Open("mysql", dsnCfg.FormatDSN())
	require.NoError(t, err)
	defer func() { require.NoError(t, hostDB.Close()) }()
	require.Eventually(t, func() bool {
		pingErr := hostDB.PingContext(context.Background())
		return pingErr == nil
	}, 60*time.Second, 500*time.Millisecond)
	t.Logf("  MySQL ready for connections in %s", time.Since(pingStart))

	applyMySQLSchema(t, hostDB)

	return MySQLConfig{
		DSN: dsnCfg.FormatDSN(),
	}
}

// applyMySQLSchema initializes the shared database schema once.
//
// Bazel can run many test processes at the same time. The advisory lock keeps
// non-idempotent CREATE TABLE statements from racing when those processes attach
// to the same MySQL container.
func applyMySQLSchema(t testing.TB, hostDB *sql.DB) {
	t.Helper()

	schemaStart := time.Now()
	ctx := context.Background()

	conn, err := hostDB.Conn(ctx)
	require.NoError(t, err, "failed to pin MySQL schema connection")
	defer func() { require.NoError(t, conn.Close()) }()

	var lockAcquired int
	err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, 60)", mysqlSchemaLockName).Scan(&lockAcquired)
	require.NoError(t, err, "failed to acquire MySQL schema lock")
	require.Equal(t, 1, lockAcquired, "timed out acquiring MySQL schema lock")
	defer func() {
		var lockReleased sql.NullInt64
		releaseErr := conn.QueryRowContext(context.Background(), "SELECT RELEASE_LOCK(?)", mysqlSchemaLockName).Scan(&lockReleased)
		require.NoError(t, releaseErr, "failed to release MySQL schema lock")
		require.True(t, lockReleased.Valid, "MySQL schema lock release returned NULL")
		require.Equal(t, int64(1), lockReleased.Int64, "MySQL schema lock was not held by this connection")
	}()

	if mysqlTableExists(t, ctx, conn, mysqlSchemaMarkerTable) {
		t.Logf("  MySQL schema already loaded in %s", time.Since(schemaStart))
		return
	}

	if mysqlTableExists(t, ctx, conn, "workspaces") {
		markMySQLSchema(t, ctx, conn)
		t.Logf("  MySQL schema marker created for existing schema in %s", time.Since(schemaStart))
		return
	}

	schemaDir := schemaPath()
	entries, err := os.ReadDir(schemaDir)
	require.NoError(t, err)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(schemaDir, entry.Name()))
		require.NoError(t, readErr)
		_, execErr := conn.ExecContext(ctx, string(data))
		require.NoError(t, execErr, "failed to apply %s", entry.Name())
	}
	markMySQLSchema(t, ctx, conn)
	t.Logf("  MySQL schema loaded in %s", time.Since(schemaStart))
}

// mysqlTableExists reports whether a table exists in the current database.
func mysqlTableExists(t testing.TB, ctx context.Context, hostDB mysqlSchemaDB, tableName string) bool {
	t.Helper()

	var count int
	err := hostDB.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.tables
		WHERE table_schema = DATABASE()
			AND table_name = ?
	`, tableName).Scan(&count)
	require.NoError(t, err, "failed to check MySQL table %q", tableName)
	return count > 0
}

// markMySQLSchema records that the shared schema has been applied.
func markMySQLSchema(t testing.TB, ctx context.Context, hostDB mysqlSchemaDB) {
	t.Helper()

	_, err := hostDB.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(64) PRIMARY KEY,
			applied_at BIGINT NOT NULL
		)
	`, mysqlSchemaMarkerTable))
	require.NoError(t, err, "failed to create MySQL schema marker table")

	_, err = hostDB.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (version, applied_at)
		VALUES ('v1', ?)
		ON DUPLICATE KEY UPDATE applied_at = VALUES(applied_at)
	`, mysqlSchemaMarkerTable), time.Now().UnixMilli())
	require.NoError(t, err, "failed to update MySQL schema marker")
}

// schemaPath returns the MySQL schema directory in Bazel runfiles or from the
// source tree.
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

	repoRoot := sourceRepoRoot()
	return filepath.Join(repoRoot, "pkg", "mysql", "schema")
}
