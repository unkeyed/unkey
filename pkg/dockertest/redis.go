package dockertest

import (
	"fmt"
	"testing"
	"time"
)

const (
	redisImage = "redis:8.0"
	redisPort  = "6379/tcp"
)

// Redis starts a Redis 8.0 container and returns the connection URL.
//
// The returned URL is in the format "redis://localhost:{port}" and can be
// used directly with most Redis client libraries. The container is
// automatically removed when the test completes via t.Cleanup.
//
// This function blocks until Redis is accepting TCP connections (up to 30s).
// Fails the test if Docker is unavailable or the container fails to start.
func Redis(t *testing.T) string {
	t.Helper()

	ctr := startContainer(t, containerConfig{
		Image:        redisImage,
		ExposedPorts: []string{redisPort},
		WaitStrategy: NewTCPWait(redisPort),
		WaitTimeout:  30 * time.Second,
		Env:          map[string]string{},
		Cmd:          []string{},
		Tmpfs:        nil,
	})

	port := ctr.Port(redisPort)
	return fmt.Sprintf("redis://localhost:%s", port)
}
