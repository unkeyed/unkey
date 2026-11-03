package containers_test

import (
	"database/sql"
	"testing"

	mysql "github.com/go-sql-driver/mysql"
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

func TestMySQLLogicalIsolated_DoesNotShareRows(t *testing.T) {
	var configs []*mysql.Config
	t.Run("isolated databases", func(t *testing.T) {
		databases := make([]*sql.DB, 2)
		for i := range databases {
			cfg := containers.MySQLLogicalIsolated(t)
			parsed, err := mysql.ParseDSN(cfg.DSN)
			require.NoError(t, err)
			configs = append(configs, parsed)
			db, err := sql.Open("mysql", cfg.DSN)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, db.Close()) })
			var tableName string
			require.NoError(t, db.QueryRow("SHOW TABLES LIKE 'workspaces'").Scan(&tableName))
			require.Equal(t, "workspaces", tableName)
			_, err = db.Exec("CREATE TABLE mysql_isolation_probe (id INT PRIMARY KEY)")
			require.NoError(t, err)
			databases[i] = db
		}
		require.Equal(t, configs[0].Addr, configs[1].Addr)
		require.NotEqual(t, configs[0].DBName, configs[1].DBName)

		_, err := databases[0].Exec("INSERT INTO mysql_isolation_probe (id) VALUES (1)")
		require.NoError(t, err)
		var count int
		require.NoError(t, databases[1].QueryRow("SELECT COUNT(*) FROM mysql_isolation_probe").Scan(&count))
		require.Zero(t, count, "an isolated database must not contain another test's rows")
	})
	for _, cfg := range configs {
		db, err := sql.Open("mysql", cfg.FormatDSN())
		require.NoError(t, err)
		t.Cleanup(func() { require.NoError(t, db.Close()) })
		var mysqlErr *mysql.MySQLError
		require.ErrorAs(t, db.PingContext(t.Context()), &mysqlErr)
		require.EqualValues(t, 1049, mysqlErr.Number, "cleanup must drop the private database")
	}
}
