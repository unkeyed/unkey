package dockertest

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	restateImage     = "restatedev/restate:1.6.0"
	restatePort      = "8080/tcp"
	restateAdminPort = "9070/tcp"
)

// RestateConfig holds connection information for a Restate test container.
type RestateConfig struct {
	HostIngressURL   string
	HostAdminURL     string
	DockerIngressURL string
	DockerAdminURL   string
}

// Restate starts a Restate container and returns connection information.
func (c *Cluster) Restate() RestateConfig {
	c.t.Helper()

	ctr, cleanup, err := startContainer(c.cli, containerConfig{
		ContainerName: "",
		Image:         restateImage,
		ExposedPorts:  []string{restatePort, restateAdminPort},
		Env:           map[string]string{},
		Cmd:           []string{},
		Tmpfs:         nil,
		Binds:         nil,
		Keep:          false,
		NetworkName:   c.network.Name,
	}, c.t.Name())
	require.NoError(c.t, err)
	if cleanup != nil {
		c.t.Cleanup(func() { require.NoError(c.t, cleanup()) })
	}

	wait := NewHTTPWait(restateAdminPort, "/health")
	wait.Wait(c.t, ctr, 30*time.Second)

	ingressPort := ctr.Port(restatePort)
	adminPort := ctr.Port(restateAdminPort)
	require.NotEmpty(c.t, ingressPort, "restate ingress port not mapped")
	require.NotEmpty(c.t, adminPort, "restate admin port not mapped")

	return RestateConfig{
		HostIngressURL:   fmt.Sprintf("http://%s:%s", ctr.Host, ingressPort),
		HostAdminURL:     fmt.Sprintf("http://%s:%s", ctr.Host, adminPort),
		DockerIngressURL: fmt.Sprintf("http://%s:%s", ctr.ContainerName, containerPortNumber(restatePort)),
		DockerAdminURL:   fmt.Sprintf("http://%s:%s", ctr.ContainerName, containerPortNumber(restateAdminPort)),
	}
}
