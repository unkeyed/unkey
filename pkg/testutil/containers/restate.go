package containers

import (
	"testing"
	"time"
)

const (
	restateImage     = "restatedev/restate:1.6.0"
	restatePort      = "8080/tcp"
	restateAdminPort = "9070/tcp"
)

// RestateConfig holds connection information for a Restate container.
type RestateConfig struct {
	// IngressURL is the Restate ingress endpoint URL.
	IngressURL string
	// AdminURL is the Restate admin endpoint URL.
	AdminURL string
}

// Restate starts a Restate container and returns ingress/admin URLs.
//
// The container is reused by stable Docker name across Bazel test processes.
// This function blocks until the admin health endpoint responds (up to 30s).
// Fails the test if Docker is unavailable or the container fails to start.
func Restate(t *testing.T, opts ...Opt) RestateConfig {
	t.Helper()

	cfg := containerConfig{
		Image:        restateImage,
		ExposedPorts: []string{restatePort, restateAdminPort},
		WaitStrategy: NewHTTPWait(restateAdminPort, "/health"),
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

	return RestateConfig{
		IngressURL: ctr.HostURL("http", restatePort),
		AdminURL:   ctr.HostURL("http", restateAdminPort),
	}
}
