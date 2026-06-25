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
