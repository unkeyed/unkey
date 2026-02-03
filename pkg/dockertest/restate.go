package dockertest

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
// The container is automatically removed when the test completes via t.Cleanup.
// This function blocks until the admin health endpoint responds (up to 30s).
// Fails the test if Docker is unavailable or the container fails to start.
func Restate(t *testing.T) RestateConfig {
	t.Helper()

	ctr := startContainer(t, containerConfig{
		Image:        restateImage,
		ExposedPorts: []string{restatePort, restateAdminPort},
		WaitStrategy: NewHTTPWait(restateAdminPort, "/health"),
		WaitTimeout:  30 * time.Second,
		Env:          map[string]string{},
		Cmd:          []string{},
	})

	return RestateConfig{
		IngressURL: ctr.HostURL("http", restatePort),
		AdminURL:   ctr.HostURL("http", restateAdminPort),
	}
}
