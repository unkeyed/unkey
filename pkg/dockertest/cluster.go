package dockertest

import (
	"testing"

	"github.com/docker/docker/client"
)

// Cluster manages a set of Docker containers for a test.
// It owns a dedicated Docker network so containers can reach each other by name.
type Cluster struct {
	t       *testing.T
	cli     *client.Client
	network Network
}

// New creates a new Docker test cluster with its own network.
// Containers start when you call a service method such as [Cluster.Redis].
func New(t *testing.T) *Cluster {
	t.Helper()

	cli := getClient(t)
	network, networkCleanup := createNetwork(t, cli)

	t.Cleanup(networkCleanup)

	return &Cluster{
		t:       t,
		cli:     cli,
		network: network,
	}
}
