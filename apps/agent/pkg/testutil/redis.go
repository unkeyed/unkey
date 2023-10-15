package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func CreateRedis(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	container, err := redis.RunContainer(
		ctx,
		testcontainers.WithImage("redis:6.2"),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})

	url, err := container.ConnectionString(ctx)
	require.NoError(t, err)
	return url
}
