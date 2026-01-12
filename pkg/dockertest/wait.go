package dockertest

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// WaitStrategy defines how to wait for a container to become ready.
//
// Implementations should poll the container until it is ready to accept
// connections or perform its intended function. If readiness cannot be
// established within the timeout, the implementation should fail the test.
type WaitStrategy interface {
	// Wait blocks until the container is ready or the timeout expires.
	// Fails the test if the container does not become ready in time.
	Wait(t *testing.T, c *Container, timeout time.Duration)
}

// TCPWait waits for a TCP port to accept connections.
//
// This is the simplest readiness check and works for most services that
// accept TCP connections (Redis, MySQL, PostgreSQL, etc.). For services
// that need application-level health checks, implement a custom [WaitStrategy].
type TCPWait struct {
	// Port is the container port to wait for (e.g., "6379/tcp").
	Port string

	// PollInterval is how often to attempt connection. Defaults to 100ms if zero.
	PollInterval time.Duration
}

// Wait polls the TCP port until it accepts connections or the timeout expires.
// Fails the test if the port is not mapped or does not become ready in time.
func (w *TCPWait) Wait(t *testing.T, c *Container, timeout time.Duration) {
	t.Helper()

	hostPort := c.Port(w.Port)
	require.NotEmpty(t, hostPort, "port %s not mapped", w.Port)

	address := net.JoinHostPort(c.Host, hostPort)

	pollInterval := w.PollInterval
	if pollInterval == 0 {
		pollInterval = 100 * time.Millisecond
	}

	require.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", address, pollInterval)
		if err != nil {
			return false
		}
		conn.Close()
		return true
	}, timeout, pollInterval, "container port %s did not become ready", address)
}

// NewTCPWait creates a [TCPWait] strategy for the given container port.
// The port should be in the format "port/protocol" (e.g., "6379/tcp").
func NewTCPWait(port string) *TCPWait {
	return &TCPWait{
		Port:         port,
		PollInterval: 100 * time.Millisecond,
	}
}
