package dockertest

import (
	"fmt"
	"testing"
	"time"
)

const (
	// redisImage is the Docker image used for Redis containers.
	redisImage = "redis:8.0"

	// redisPort is the default Redis port.
	redisPort = "6379/tcp"
)

// Redis starts a Redis container and returns the connection URL.
//
// The container uses Redis 8.0 and is automatically removed when the test
// completes. If Docker is unavailable, the test is skipped.
//
// The returned URL is in the format "redis://localhost:{port}" and can be
// used directly with most Redis client libraries.
//
// Example:
//
//	func TestWithRedis(t *testing.T) {
//	    url := dockertest.Redis(t)
//	    // Use url with your Redis client
//	}
func Redis(t *testing.T) string {
	t.Helper()

	ctr := startContainer(t, containerConfig{
		Image:        redisImage,
		ExposedPorts: []string{redisPort},
		WaitStrategy: NewTCPWait(redisPort),
		WaitTimeout:  30 * time.Second,
	})

	port := ctr.Port(redisPort)
	return fmt.Sprintf("redis://localhost:%s", port)
}
