package containers

import (
	"fmt"
	"testing"
	"time"
)

const (
	redisImage = "redis:8.0"
	redisPort  = "6379/tcp"
)

// Redis starts a Redis container and returns the connection URL.
//
// The container is owned by t and removed automatically with t.Cleanup.
func Redis(t testing.TB) string {
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
	return fmt.Sprintf("redis://127.0.0.1:%s", port)
}
