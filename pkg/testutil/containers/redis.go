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
// The container is reused by stable Docker name across Bazel test processes.
func Redis(t testing.TB, opts ...Opt) string {
	t.Helper()

	cfg := containerConfig{
		Image:        redisImage,
		ExposedPorts: []string{redisPort},
		WaitStrategy: NewTCPWait(redisPort),
		WaitTimeout:  30 * time.Second,
		Env:          map[string]string{},
		Cmd:          []string{},
		Tmpfs:        nil,
		Dedicated:    false,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	ctr := startContainer(t, cfg)

	port := ctr.Port(redisPort)
	return fmt.Sprintf("redis://127.0.0.1:%s", port)
}
