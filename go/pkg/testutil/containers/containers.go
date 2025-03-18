package containers

import (
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

// Containers represents a container manager for running containerized services during tests.
// It maintains a reference to the current test and Docker pool for launching containers.
type Containers struct {
	t    *testing.T
	pool *dockertest.Pool
}

// NewContainers creates a new container manager for the given test.
//
// It initializes a Docker connection pool and verifies connectivity to the Docker daemon.
// If the Docker daemon is not available, the test will fail immediately.
//
// Parameters:
//   - t: The current test context, used for logging and cleanup registration
//
// Returns:
//   - A new Containers instance configured for the test
//
// The returned Containers instance can be used to start various containerized
// services like MySQL databases for integration testing.
//
// Example:
//
//	func TestWithMySQL(t *testing.T) {
//	    containers := testutil.NewContainers(t)
//	    dsn := containers.RunMySQL()
//
//	    // Use the DSN to connect to the database
//	    db, err := sql.Open("mysql", dsn)
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//	    defer db.Close()
//
//	    // Run your tests using the database
//	}
func New(t *testing.T) *Containers {
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
