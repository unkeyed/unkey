package containers

import (
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Containers represents a container manager for running containerized services during tests.
// It maintains a reference to the current test and Docker pool for launching containers.
type Containers struct {
	t       *testing.T
	pool    *dockertest.Pool
	network *dockertest.Network
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

	network, err := c.getOrCreateNetwork(networkName, true)
	require.NoError(t, err)
	c.network = network

	return c
}

// getOrCreateNetwork safely gets an existing network or creates a new one,
// ignoring "network already exists" errors.
func (c *Containers) getOrCreateNetwork(networkName string, retry bool) (*dockertest.Network, error) {
	networks, err := c.pool.NetworksByName(networkName)
	if err != nil {
		return nil, err
	}

	// Check if network already exists
	for _, found := range networks {
		if found.Network.Name == networkName {
			return &found, nil
		}
	}

	// Network doesn't exist, create it
	network, err := c.pool.CreateNetwork(networkName)
	if err != nil {
		if strings.Contains(err.Error(), "API error (409)") && retry {
			return c.getOrCreateNetwork(networkName, false)
		}

		return nil, err
	}

	return network, nil
}

// getOrCreateContainer safely gets an existing container or creates a new one,
// handling race conditions when multiple tests try to create the same container.
//
// This function protects against the race condition where:
// 1. Test A checks if container exists → false
// 2. Test B checks if container exists → false
// 3. Test A starts creating container
// 4. Test B tries to create container → fails with "already exists"
//
// Returns the container resource and whether it was newly created.
func (c *Containers) getOrCreateContainer(containerName string, runOpts *dockertest.RunOptions) (*dockertest.Resource, bool, error) {

	var err error
	for i := range 10 {
		resource, exists := c.pool.ContainerByName(containerName)
		if exists {
			return resource, false, nil
		}

		resource, err = c.pool.RunWithOptions(runOpts)
		if err == nil {
			return resource, true, nil
		}

		time.Sleep(time.Duration(i) * time.Second)
	}

	return nil, false, fault.Wrap(err, fault.Internal("exceeded retries already"))
}
