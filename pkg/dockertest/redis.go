package dockertest

import (
	"fmt"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	redisImage = "redis:8.0"
	redisPort  = "6379/tcp"
)

// RedisConfig holds connection information for a Redis test container.
type RedisConfig struct {
	HostURL   string
	DockerURL string
}

// Redis starts a Redis container and returns connection information.
func (c *Cluster) Redis() RedisConfig {
	c.t.Helper()

	ctr, cleanup, err := startContainer(c.cli, containerConfig{
		ContainerName: "",
		Image:         redisImage,
		ExposedPorts:  []string{redisPort},
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

	wait := NewTCPWait(redisPort)
	wait.Wait(c.t, ctr, 30*time.Second)

	port := ctr.Port(redisPort)
	require.NotEmpty(c.t, port, "redis port not mapped")

	return RedisConfig{
		HostURL:   fmt.Sprintf("redis://%s:%s", ctr.Host, port),
		DockerURL: fmt.Sprintf("redis://%s:%s", ctr.ContainerName, containerPortNumber(redisPort)),
	}
}
