package containers

import (
	"database/sql"
	"fmt"
	"strings"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/unkeyed/unkey/go/pkg/db"

	"github.com/stretchr/testify/require"
)

// RunMySQL starts a MySQL container and returns a database connection string (DSN).
//
// The method starts a containerized MySQL instance, waits until it's ready to accept
// connections, creates the necessary database schema, and then returns a properly
// formatted DSN that can be used to connect to the database.
//
// Thread safety:
//   - This method is not thread-safe and should be called from a single goroutine.
//   - The underlying database connection is not shared between tests.
//
// Performance characteristics:
//   - Starting the container typically takes 5-10 seconds depending on the system.
//   - Container and database resources are cleaned up automatically after the test.
//
// Side effects:
//   - Creates a Docker container that will persist until test cleanup.
//   - Creates a database with the Unkey schema.
//   - Registers cleanup functions with the test to remove resources after test completion.
//
// Returns:
//   - A MySQL DSN string that can be used with sql.Open or similar functions
//
// The method will automatically register cleanup functions with the test to ensure
// that the container is stopped and removed when the test completes, regardless of success
// or failure.
//
// Example usage:
//
//	func TestDatabaseOperations(t *testing.T) {
//	    containers := testutil.NewContainers(t)
//	    dsn := containers.RunMySQL()
//
//	    // Connect to the database using the DSN
//	    db, err := sql.Open("mysql", dsn)
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    // Now use the database for testing
//	    // ...
//
//	    // No need to clean up - it happens automatically when the test finishes
//	}
//
// Note: This function requires Docker to be installed and running on the system
// where tests are executed. It will fail if Docker is not available.
func (c *Containers) RunMySQL() string {
	c.t.Helper()

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

	cfg := mysql.NewConfig()
	cfg.User = "unkey"
	cfg.Passwd = "password"
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("localhost:%s", resource.GetPort("3306/tcp"))
	cfg.DBName = "unkey"
	cfg.ParseTime = true
	cfg.Logger = &mysql.NopLogger{}

	var conn *sql.DB
	require.NoError(c.t, c.pool.Retry(func() error {

		connector, err2 := mysql.NewConnector(cfg)
		if err2 != nil {
			return fmt.Errorf("unable to create mysql connector: %w", err2)
		}

		conn = sql.OpenDB(connector)
		err3 := conn.Ping()
		if err3 != nil {
			return fmt.Errorf("unable to ping mysql: %w", err3)
		}

		return nil
	}))

	c.t.Cleanup(func() {
		require.NoError(c.t, conn.Close())
	})
	// Creating the database tables
	queries := strings.Split(string(db.Schema), ";")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		// Add the semicolon back
		query += ";"

		_, err = conn.Exec(query)
		require.NoError(c.t, err)

	}

	return cfg.FormatDSN()
}
