package containers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

// RunRedis starts a Redis container and returns a Redis client configured to connect to it.
//
// The method starts a containerized Redis instance, waits until it's ready to accept
// connections, and then returns a properly configured Redis client that can be used
// for testing.
//
// Thread safety:
//   - This method is not thread-safe and should be called from a single goroutine.
//   - The returned Redis client can be shared between goroutines as the underlying
//     redis/v8 client handles concurrency safely.
//
// Performance characteristics:
//   - Starting the container typically takes 1-3 seconds depending on the system.
//   - Container resources are cleaned up automatically after the test.
//
// Side effects:
//   - Creates a Docker container that will persist until test cleanup.
//   - Registers cleanup functions with the test to remove resources after test completion.
//
// Returns:
//   - A configured redis.Client ready to use for testing
//   - The address of the Redis server in the format "host:port"
//
// The method will automatically register cleanup functions with the test to ensure
// that the container is stopped and removed when the test completes, regardless of success
// or failure.
//
// Example usage:
//
//	func TestRedisOperations(t *testing.T) {
//	    containers := testutil.NewContainers(t)
//	    redisClient, addr := containers.RunRedis()
//
//	    // Use the Redis client for testing
//	    ctx := context.Background()
//	    err := redisClient.Set(ctx, "testKey", "testValue", 0).Err()
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    val, err := redisClient.Get(ctx, "testKey").Result()
//	    if err != nil {
//	        t.Fatal(err)
//	    }
//
//	    if val != "testValue" {
//	        t.Fatalf("expected 'testValue', got '%s'", val)
//	    }
//
//	    // No need to clean up - it happens automatically when the test finishes
//	}
//
// Note: This function requires Docker to be installed and running on the system
// where tests are executed. It will fail if Docker is not available.
//
// See also: [RunMySQL] for starting a MySQL container.
func (c *Containers) RunRedis() (*redis.Client, string) {
	c.t.Helper()

	resource, err := c.pool.Run("redis", "latest", nil)
	require.NoError(c.t, err)

	c.t.Cleanup(func() {
		require.NoError(c.t, c.pool.Purge(resource))
	})

	address := fmt.Sprintf("localhost:%s", resource.GetPort("6379/tcp"))

	// Configure the Redis client
	// nolint:exhaustruct
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Wait for the Redis server to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	require.NoError(c.t, c.pool.Retry(func() error {
		return client.Ping(ctx).Err()
	}))

	c.t.Cleanup(func() {
		require.NoError(c.t, client.Close())
	})

	return client, address
}
