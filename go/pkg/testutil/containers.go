package testutil

import (
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

type Containers struct {
	t    *testing.T
	pool *dockertest.Pool
}

func NewContainers(t *testing.T) *Containers {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	err = pool.Client.Ping()
	require.NoError(t, err)

	c := &Containers{
		t:    t,
		pool: pool,
	}

	return c
}

func (c *Containers) RunMySQL() string {
	resource, err := c.pool.Run("mysql", "latest", []string{
		"MYSQL_ROOT_PASSWORD=root",
		"MYSQL_DATABASE=unkey",
		"MYSQL_USER=unkey",
		"MYSQL_PASSWORD=password",
	})
	require.NoError(c.t, err)

	c.t.Cleanup(func() {
		require.NoError(c.t, c.pool.Purge(resource))
	})

	addr := fmt.Sprintf("unkey:password@(localhost:%s)/unkey", resource.GetPort("3306/tcp"))

	require.NoError(c.t, c.pool.Retry(func() error {
		db, err := sql.Open("mysql", addr)
		if err != nil {
			return fmt.Errorf("unable to open mysql conenction: %w", err)
		}
		err = db.Ping()
		if err != nil {
			return fmt.Errorf("unable to ping mysql: %w", err)
		}
		return nil
	}))

	return addr
}
