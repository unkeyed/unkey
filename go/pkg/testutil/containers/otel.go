package containers

import (
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/stretchr/testify/require"
)

func (c *Containers) RunOtel(preventPurge ...bool) {
	c.t.Helper()
	defer func(start time.Time) {
		c.t.Logf("starting Otel took %s", time.Since(start))
	}(time.Now())

	resource, ok := c.pool.ContainerByName("otel")
	if ok {
		err := resource.ConnectToNetwork(c.network)
		require.NoError(c.t, err)
		return
	}
	// nolint:exhaustruct
	resource, err := c.pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "otel",
		Hostname:   "otel",
		Repository: "grafana/otel-lgtm",
		Tag:        "latest",
		ExposedPorts: []string{
			"3000",
			"4317",
			"4318",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"3000": []docker.PortBinding{{
				HostIP:   "127.0.0.1",
				HostPort: "3000",
			}},
		},
		Networks: []*dockertest.Network{
			c.network,
		},
	})
	require.NoError(c.t, err)

	if len(preventPurge) == 0 || !preventPurge[0] {
		c.t.Cleanup(func() {
			require.NoError(c.t, c.pool.Purge(resource))
		})
	}

}
