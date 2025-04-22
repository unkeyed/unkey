package containers

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

// RunClickHouse starts a ClickHouse container and returns a configured ClickHouse connection.
//
// The method starts a containerized ClickHouse instance, waits until it's ready to accept
// connections, runs the provided schema migrations, and returns a properly configured
// ClickHouse connection that can be used for testing.
//
// Thread safety:
//   - This method is not thread-safe and should be called from a single goroutine.
//   - The underlying ClickHouse connection is not shared between tests.
//
// Performance characteristics:
//   - Starting the container typically takes 5-10 seconds depending on the system.
//   - Container and database resources are cleaned up automatically after the test.
//
// Side effects:
//   - Creates a Docker container that will persist until test cleanup.
//   - Creates database schema by running the provided migrations.
//   - Registers cleanup functions with the test to remove resources after test completion.
//
// Returns:
//   - A configured ClickHouse connection (driver.Conn) ready to use for testing
//   - A DSN string that can be used to create additional connections
//
// The method will automatically register cleanup functions with the test to ensure
// that the container is stopped and removed when the test completes, regardless of success
// or failure.
//
// Example usage:
//
//	func TestClickHouseOperations(t *testing.T) {
//	    containers := containers.NewContainers(t)
//	    conn, dsn := containers.RunClickHouse()
//
//	    // Use the connection for testing
//	    ctx := context.Background()
//	    rows, err := conn.Query(ctx, "SELECT 1")
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer rows.Close()
//
//	    // Or create a new connection using the DSN
//	    db, err := sql.Open("clickhouse", dsn)
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer db.Close()
//
//	    // No need to clean up - it happens automatically when the test finishes
//	}
//
// Note: This function requires Docker to be installed and running on the system
// where tests are executed. It will fail if Docker is not available.
//
// See also:
//   - [RunMySQL] for starting a MySQL container.
//   - [RunRedis] for starting a Redis container.
func (c *Containers) RunClickHouse() (hostDsn, dockerDsn string) {
	c.t.Helper()
	defer func(start time.Time) {
		c.t.Logf("starting ClickHouse took %s", time.Since(start))
	}(time.Now())
	// Start ClickHouse container
	resource, err := c.pool.Run("bitnami/clickhouse", "latest", []string{
		"CLICKHOUSE_ADMIN_USER=default",
		"CLICKHOUSE_ADMIN_PASSWORD=password",
	})
	require.NoError(c.t, err)

	err = resource.ConnectToNetwork(c.network)
	require.NoError(c.t, err)
	c.t.Cleanup(func() {
		if !c.t.Failed() {
			require.NoError(c.t, c.pool.Purge(resource))
		}
	})

	// Construct DSN
	port := resource.GetPort("9000/tcp")
	hostDsn = fmt.Sprintf("clickhouse://default:password@localhost:%s?secure=false&skip_verify=true&dial_timeout=10s", port)
	dockerDsn = fmt.Sprintf("clickhouse://default:password@%s:9000?secure=false&skip_verify=true&dial_timeout=10s", resource.GetIPInNetwork(c.network))

	// Configure ClickHouse connection
	var conn driver.Conn
	require.NoError(c.t, c.pool.Retry(func() error {
		var connErr error
		conn, connErr = clickhouse.Open(&clickhouse.Options{
			Addr: []string{fmt.Sprintf("localhost:%s", port)},
			Auth: clickhouse.Auth{
				Username: "default",
				Password: "password",
			},
			DialTimeout: 5 * time.Second,
			Settings: map[string]interface{}{
				"max_execution_time": 60,
			},
		})
		if connErr != nil {
			return connErr
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return conn.Ping(ctx)
	}))

	require.NoError(c.t, conn.Close())

	err = runClickHouseMigrations(conn)
	require.NoError(c.t, err)

	return hostDsn, dockerDsn
}

// runClickHouseMigrations executes SQL migration files
//
// The function reads all .sql files from the given directory, in lexicographical order,
// and executes them against the provided ClickHouse connection.
//
// Parameters:
//   - conn: A ClickHouse connection to use for migrations
//
// Returns:
//   - An error if any part of the migration process fails
func runClickHouseMigrations(conn driver.Conn) error {

	return fs.WalkDir(schema.Migrations, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		f, err := schema.Migrations.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		content, err := io.ReadAll(f)
		if err != nil {
			return err
		}

		queries := strings.Split(string(content), ";")

		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}

			err = conn.Exec(context.Background(), fmt.Sprintf("%s;", query))
			if err != nil {
				return err
			}
		}

		return nil
	})

}
