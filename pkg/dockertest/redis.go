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

var redisCtr shared

// redisContainerConfig returns the container configuration for Redis.
func redisContainerConfig() containerConfig {
	return containerConfig{
		Image:        redisImage,
		ExposedPorts: []string{redisPort},
		WaitStrategy: NewTCPWait(redisPort),
		WaitTimeout:  30 * time.Second,
		Env:          map[string]string{},
		Cmd:          []string{},
		Tmpfs:        nil,
		SkipCleanup:  false,
	}
}

// Redis starts (or reuses) a shared Redis container and returns the connection URL.
//
// The container starts on the first call in the process and is reused by all
// subsequent calls. Tests should use unique keys (e.g., via uid.New()) to avoid
// cross-test data leakage on the shared instance.
func Redis(t *testing.T) string {
	t.Helper()

	ctr := redisCtr.get(t, redisContainerConfig())
	port := ctr.Port(redisPort)

	return fmt.Sprintf("redis://localhost:%s", port)
}
