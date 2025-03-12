package containers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
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
// Parameters:
//   - migrationsDir: Path to directory containing SQL migration files (pass empty string to skip migrations)
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
//	    conn, dsn := containers.RunClickHouse("./testdata/migrations")
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
func (c *Containers) RunClickHouse(migrationsDir string) (driver.Conn, string) {
	c.t.Helper()
	// Start ClickHouse container
	resource, err := c.pool.Run("bitnami/clickhouse", "latest", []string{
		"CLICKHOUSE_ADMIN_USER=default",
		"CLICKHOUSE_ADMIN_PASSWORD=password",
	})
	require.NoError(c.t, err)

	c.t.Cleanup(func() {
		require.NoError(c.t, c.pool.Purge(resource))
	})

	// Construct DSN
	port := resource.GetPort("9000/tcp")
	dsn := fmt.Sprintf("clickhouse://unkey:password@localhost:%s/unkey?dial_timeout=10s", port)

	// Configure ClickHouse connection
	var conn driver.Conn
	require.NoError(c.t, c.pool.Retry(func() error {
		var err error
		conn, err = clickhouse.Open(&clickhouse.Options{
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
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return conn.Ping(ctx)
	}))

	c.t.Cleanup(func() {
		conn.Close()
	})

	// Run schema migrations if a migrations directory was provided
	if migrationsDir != "" {
		err := runClickHouseMigrations(conn, migrationsDir)
		require.NoError(c.t, err)
	}

	return conn, dsn
}

// runClickHouseMigrations executes SQL migration files from the specified directory.
//
// The function reads all .sql files from the given directory, sorts them alphanumerically,
// and executes them in order against the provided ClickHouse connection.
//
// Parameters:
//   - conn: A ClickHouse connection to use for migrations
//   - migrationsDir: Path to directory containing SQL migration files
//
// Returns:
//   - An error if any part of the migration process fails
func runClickHouseMigrations(conn driver.Conn, migrationsDir string) error {
	// Get all SQL files from migrations directory
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort SQL files
	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(migrationsDir, file.Name()))
		}
	}

	// Sort files alphanumerically to ensure execution order
	sort.Strings(sqlFiles)

	// Execute each migration file
	for _, file := range sqlFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Split file content into individual statements
		queries := strings.Split(string(content), ";")

		ctx := context.Background()
		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" {
				continue
			}

			if err := conn.Exec(ctx, query); err != nil {
				return fmt.Errorf("failed to execute migration from %s: %w\nQuery: %s", file, err, query)
			}
		}
	}

	return nil
}
