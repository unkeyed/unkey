package dockertest_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/dockertest"
)

func TestRedis(t *testing.T) {
	// This test verifies that the Redis container starts correctly
	// and is accessible via the returned URL.
	url := dockertest.Redis(t)

	// Parse the URL and create a Redis client
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	defer func() { require.NoError(t, client.Close()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify we can ping Redis
	err = client.Ping(ctx).Err()
	require.NoError(t, err)

	// Verify we can set and get a value
	err = client.Set(ctx, "test-key", "test-value", 0).Err()
	require.NoError(t, err)

	val, err := client.Get(ctx, "test-key").Result()
	require.NoError(t, err)
	require.Equal(t, "test-value", val)
}

func TestRedis_MultipleContainers(t *testing.T) {
	// This test verifies that multiple Redis containers can run in parallel
	// with isolated data.
	url1 := dockertest.Redis(t)
	url2 := dockertest.Redis(t)

	// The URLs should be different (different ports)
	require.NotEqual(t, url1, url2)

	// Create clients for both
	opts1, err := redis.ParseURL(url1)
	require.NoError(t, err)
	client1 := redis.NewClient(opts1)
	defer func() { require.NoError(t, client1.Close()) }()

	opts2, err := redis.ParseURL(url2)
	require.NoError(t, err)
	client2 := redis.NewClient(opts2)
	defer func() { require.NoError(t, client2.Close()) }()

	ctx := context.Background()

	// Set different values in each container
	err = client1.Set(ctx, "key", "value1", 0).Err()
	require.NoError(t, err)

	err = client2.Set(ctx, "key", "value2", 0).Err()
	require.NoError(t, err)

	// Verify isolation - each container has its own value
	val1, err := client1.Get(ctx, "key").Result()
	require.NoError(t, err)
	require.Equal(t, "value1", val1)

	val2, err := client2.Get(ctx, "key").Result()
	require.NoError(t, err)
	require.Equal(t, "value2", val2)
}
