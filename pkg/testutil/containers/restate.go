package containers

import (
	"fmt"
	"testing"
)

const (
	restatePort      = 8080
	restateAdminPort = 9070
)

// RestateConfig holds connection information for the Restate test container.
type RestateConfig struct {
	// IngressURL is the Restate ingress endpoint URL.
	IngressURL string
	// AdminURL is the Restate admin endpoint URL.
	AdminURL string
}

// Restate starts the shared Docker Compose Restate service and returns ingress/admin URLs.
//
// The container is reused through the worktree's Docker Compose project.
func Restate(t testing.TB) RestateConfig {
	t.Helper()

	c := startService(t, "restate")

	return RestateConfig{
		IngressURL: fmt.Sprintf("http://%s", c.Addr(t, restatePort)),
		AdminURL:   fmt.Sprintf("http://localhost:%d", c.Port(t, restateAdminPort)),
	}
}
