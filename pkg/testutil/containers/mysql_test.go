package containers_test

import (
	"database/sql"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestMySQL_ReusesContainerAndSchema(t *testing.T) {
	cfg1 := containers.MySQL(t)
	cfg2 := containers.MySQL(t)

	require.Equal(t, cfg1.DSN, cfg2.DSN)

	db, err := sql.Open("mysql", cfg2.DSN)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	var tableName string
	err = db.QueryRow("SHOW TABLES LIKE 'workspaces'").Scan(&tableName)
	require.NoError(t, err)
	require.Equal(t, "workspaces", tableName)
}

func TestMySQLIsolated_DoesNotShareRows(t *testing.T) {
	databases := make([]*sql.DB, 2)
	for i := range databases {
		cfg := containers.MySQLIsolated(t)
		db, err := sql.Open("mysql", cfg.DSN)
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		_, err = db.Exec("CREATE TABLE IF NOT EXISTS mysql_isolation_probe (id INT PRIMARY KEY)")
		require.NoError(t, err)
		t.Cleanup(func() {
			_, err := db.Exec("DROP TABLE IF EXISTS mysql_isolation_probe")
			require.NoError(t, err)
		})
		databases[i] = db
	}

	_, err := databases[0].Exec("INSERT INTO mysql_isolation_probe (id) VALUES (1)")
	require.NoError(t, err)
	var count int
	require.NoError(t, databases[1].QueryRow("SELECT COUNT(*) FROM mysql_isolation_probe").Scan(&count))
	require.Zero(t, count, "an isolated instance must not contain another test's rows")
}
