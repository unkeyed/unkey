package containers

import (
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	"github.com/stretchr/testify/require"
)

func (c *Containers) RunOtel() {
	c.t.Helper()

	_, _, err := c.getOrCreateContainer(containerNameOtel, &dockertest.RunOptions{

		Name:       containerNameOtel,
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

}
