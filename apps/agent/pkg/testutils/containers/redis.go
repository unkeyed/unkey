package containers

import (
	"context"
	"fmt"
	"testing"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Redis struct {
	URL    string
	Client *goredis.Client
	Stop   func()
}

// NewRedis runs a Redis container and returns the URL and a client to interact with it.
// The caller is responsible for stopping the container when done.
func NewRedis(t *testing.T) Redis {
	t.Helper()

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		SkipReaper:   true,
		Image:        "redis:latest",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForExposedPort(),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	err = container.Start(ctx)
	require.NoError(t, err)

	host, err := container.Host(ctx)
	require.NoError(t, err)

	port, err := container.MappedPort(ctx, "6379")
	require.NoError(t, err)

	url := fmt.Sprintf("redis://%s:%s", host, port.Port())

	require.NotEmpty(t, url, "connection string is empty")

	opts, err := goredis.ParseURL(url)
	require.NoError(t, err)
	client := goredis.NewClient(opts)

	_, err = client.Ping(ctx).Result()
	require.NoError(t, err)

	return Redis{
		URL:    url,
		Client: client,
		Stop: func() {
			require.NoError(t, client.Close())
			require.NoError(t, container.Terminate(ctx))
		},
	}
}
