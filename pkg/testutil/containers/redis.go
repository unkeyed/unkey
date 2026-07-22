package containers

import (
	"fmt"
	"testing"
)

const redisPort = 6379

// Redis starts the shared Docker Compose Redis service and returns the connection URL.
//
// The container is reused through the worktree's Docker Compose project.
func Redis(t testing.TB) string {
	t.Helper()

	c := startService(t, "redis")
	return fmt.Sprintf("redis://127.0.0.1:%d", c.Port(t, redisPort))
}
