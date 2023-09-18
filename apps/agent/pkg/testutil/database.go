package testutil

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

func createMysqlDatabase(t *testing.T) (string, error) {
	t.Helper()
	ctx := context.Background()
	schemaPath, err := filepath.Abs("../database/schema.sql")
	require.NoError(t, err)

	container, err := mysql.RunContainer(

		ctx,
		mysql.WithDatabase("unkey"),
		mysql.WithUsername("username"),
		mysql.WithPassword("password"),
		mysql.WithScripts(schemaPath),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Stop(ctx, nil))
	})

	return container.ConnectionString(ctx)

}
