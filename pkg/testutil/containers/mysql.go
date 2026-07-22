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
	mysqlPort              = 3306
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

// MySQLConfig holds connection information for the MySQL test container.
type MySQLConfig struct {
	// DSN is the host DSN for connecting from the test runner.
	DSN string
}

// MySQL starts the shared Docker Compose MySQL service and returns connection info.
//
// The container is reused through the worktree's Docker Compose project.
func MySQL(t testing.TB) MySQLConfig {
	t.Helper()

	containerStart := time.Now()

	c := startService(t, "mysql")
	t.Logf("  MySQL container started in %s", time.Since(containerStart))

	dsnCfg := mysql.NewConfig()
	dsnCfg.User = mysqlUser
	dsnCfg.Passwd = mysqlPassword
	dsnCfg.Net = "tcp"
	dsnCfg.Addr = c.Addr(t, mysqlPort)
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
// CI can run many test processes at the same time. The advisory lock keeps
// non-idempotent CREATE TABLE statements from racing when those processes use
// the same MySQL container.
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

// schemaPath returns the MySQL schema directory from test runfiles or from the
// source tree.
func schemaPath() string {
	return dataPath("pkg", "mysql", "schema")
}
