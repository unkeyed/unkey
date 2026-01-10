package dockertest

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// WaitStrategy defines how to wait for a container to become ready.
type WaitStrategy interface {
	// Wait blocks until the container is ready or fails the test.
	Wait(t *testing.T, c *Container, timeout time.Duration)
}

// TCPWait waits for a TCP port to become available.
type TCPWait struct {
	// Port is the container port to wait for (e.g., "6379/tcp").
	Port string

	// PollInterval is how often to attempt connection. Default is 100ms.
	PollInterval time.Duration
}

// Wait implements WaitStrategy by polling a TCP port until it accepts connections.
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

// NewTCPWait creates a TCPWait strategy for the given container port.
func NewTCPWait(port string) *TCPWait {
	return &TCPWait{
		Port:         port,
		PollInterval: 100 * time.Millisecond,
	}
}
