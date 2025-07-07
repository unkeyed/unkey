package containers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/stretchr/testify/require"
)

// RunMySQL starts a MySQL container and returns MySQL config structs without database names.
//
// The method starts a containerized MySQL instance, waits until it's ready to accept
// connections, creates the necessary database schemas (unkey and hydra), and then returns
// MySQL config structs that can be used to connect to specific databases.
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
//   - Creates databases for both unkey and hydra schemas.
//   - Registers cleanup functions with the test to remove resources after test completion.
//
// Returns:
//   - MySQL config structs without database names that can be used by setting DBName
//
// The method will automatically register cleanup functions with the test to ensure
// that the container is stopped and removed when the test completes, regardless of success
// or failure.
//
// Example usage:
//
//	func TestDatabaseOperations(t *testing.T) {
//	    containers := testutil.NewContainers(t)
//	    hostCfg, dockerCfg := containers.RunMySQL()
//
//	    // Connect to the unkey database
//	    hostCfg.DBName = "unkey"
//	    db, err := sql.Open("mysql", hostCfg.FormatDSN())
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    // Connect to the hydra database
//	    dockerCfg.DBName = "hydra"
//	    // ...
//
//	    // No need to clean up - it happens automatically when the test finishes
//	}
//
// Note: This function requires Docker to be installed and running on the system
// where tests are executed. It will fail if Docker is not available.
func (c *Containers) RunMySQL() (hostCfg, dockerCfg *mysql.Config) {

	_, err := c.pool.Client.InspectImage(containerNameMySQL)
	if err != nil {
		// Get the path to the current file
		_, currentFilePath, _, _ := runtime.Caller(0)

		// We're going from go/pkg/testutil/containers/ up to unkey/
		projectRoot := filepath.Join(filepath.Dir(currentFilePath), "../../../..")

		t0 := time.Now()
		// nolint:exhaustruct
		err = c.pool.Client.BuildImage(docker.BuildImageOptions{
			Name:         containerNameMySQL,
			Dockerfile:   "deployment/Dockerfile.mysql",
			ContextDir:   projectRoot,
			OutputStream: os.Stdout,
			ErrorStream:  os.Stderr,
		})
		require.NoError(c.t, err)
		c.t.Logf("Building mysql took %s\n", time.Since(t0))
	}
	runOpts := &dockertest.RunOptions{
		Name:       containerNameMySQL,
		Repository: containerNameMySQL,
		Tag:        "latest",
		Env: []string{
			"MYSQL_ALLOW_EMPTY_PASSWORD=yes",
		},
		Networks: []*dockertest.Network{c.network},
	}

	resource, _, err := c.getOrCreateContainer(containerNameMySQL, runOpts)
	require.NoError(c.t, err)

	cfg := mysql.NewConfig()
	cfg.User = "unkey"
	cfg.Passwd = "password"
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("localhost:%s", resource.GetPort("3306/tcp"))
	cfg.DBName = "" // Explicitly no database name in base DSN
	cfg.ParseTime = true
	cfg.Logger = &mysql.NopLogger{}

	var conn *sql.DB
	defer func() {
		if conn != nil {
			require.NoError(c.t, conn.Close())
		}
	}()
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

	hostCfg = cfg

	dockerCfg = mysql.NewConfig()
	dockerCfg.User = cfg.User
	dockerCfg.Passwd = cfg.Passwd
	dockerCfg.Net = cfg.Net
	dockerCfg.Addr = fmt.Sprintf("%s:3306", resource.GetIPInNetwork(c.network))
	dockerCfg.DBName = "" // Explicitly no database name in base config
	dockerCfg.ParseTime = cfg.ParseTime
	dockerCfg.Logger = cfg.Logger

	return hostCfg, dockerCfg
}
