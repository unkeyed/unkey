package containers_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
)

func TestRedis(t *testing.T) {
	// This test verifies that the Redis container starts correctly
	// and is accessible via the returned URL.
	url := containers.Redis(t)

	// Parse the URL and create a Redis client
	opts, err := redis.ParseURL(url)
	require.NoError(t, err)

	client := redis.NewClient(opts)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify we can ping Redis
	err = client.Ping(ctx).Err()
	require.NoError(t, err)

	// Verify we can set and get a value
	testKey := fmt.Sprintf("test-key-%d", time.Now().UnixNano())
	err = client.Set(ctx, testKey, "test-value", 0).Err()
	require.NoError(t, err)

	val, err := client.Get(ctx, testKey).Result()
	require.NoError(t, err)
	require.Equal(t, "test-value", val)
}

func TestRedis_ReusesContainer(t *testing.T) {
	url1 := containers.Redis(t)
	url2 := containers.Redis(t)

	require.Equal(t, url1, url2)

	opts1, err := redis.ParseURL(url1)
	require.NoError(t, err)
	client1 := redis.NewClient(opts1)
	t.Cleanup(func() { require.NoError(t, client1.Close()) })

	opts2, err := redis.ParseURL(url2)
	require.NoError(t, err)
	client2 := redis.NewClient(opts2)
	t.Cleanup(func() { require.NoError(t, client2.Close()) })

	ctx := context.Background()
	key := fmt.Sprintf("key-%d", time.Now().UnixNano())

	err = client1.Set(ctx, key, "shared-value", 0).Err()
	require.NoError(t, err)

	val2, err := client2.Get(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, "shared-value", val2)
}

func TestRedis_DedicatedContainer(t *testing.T) {
	sharedURL := containers.Redis(t)
	dedicatedURL := containers.Redis(t, containers.WithDedicatedContainer())

	require.NotEqual(t, sharedURL, dedicatedURL)
}
