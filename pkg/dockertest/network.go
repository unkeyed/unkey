package dockertest

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Network represents a Docker network created for tests.
type Network struct {
	ID   string
	Name string
}

// CreateNetwork creates a Docker network for tests and returns a cleanup function.
//
// Call the returned cleanup function when the network is no longer needed.
func CreateNetwork(t *testing.T) (Network, func()) {
	t.Helper()

	cli := getClient(t)
	ctx := context.Background()

	name := uid.New(uid.Prefix("dockertest"))
	// nolint:exhaustruct
	resp, err := cli.NetworkCreate(ctx, name, network.CreateOptions{
		Labels: map[string]string{
			"owner": "dockertest",
			"test":  t.Name(),
		},
	})
	require.NoError(t, err, "failed to create network")

	cleanup := func() {
		// nolint:exhaustruct
		inspect, err := cli.NetworkInspect(ctx, resp.ID, network.InspectOptions{})
		if err == nil {
			for containerID := range inspect.Containers {
				disconnectErr := cli.NetworkDisconnect(ctx, resp.ID, containerID, true)
				if disconnectErr != nil {
					t.Logf("failed to disconnect container %s from network %s: %v", containerID, resp.ID, disconnectErr)
				}
			}
		}

		err = cli.NetworkRemove(ctx, resp.ID)
		require.NoError(t, err, "failed to remove network")
	}
	return Network{
		ID:   resp.ID,
		Name: name,
	}, cleanup
}

func networkName(net *Network) string {
	if net == nil {
		return ""
	}
	return net.Name
}
