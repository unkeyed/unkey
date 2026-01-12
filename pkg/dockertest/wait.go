package dockertest

import (
	"fmt"
	"net"
	"net/http"
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

// HTTPWait waits for an HTTP endpoint to return an expected status code.
//
// This strategy is useful for services that expose health check endpoints,
// such as MinIO's /minio/health/live endpoint. Unlike [TCPWait], this verifies
// that the application is actually responding to HTTP requests, not just
// accepting TCP connections.
type HTTPWait struct {
	// Port is the container port to connect to (e.g., "9000/tcp").
	Port string

	// Path is the HTTP path to request (e.g., "/minio/health/live").
	Path string

	// ExpectedStatus is the HTTP status code that indicates readiness.
	// Defaults to 200 if zero.
	ExpectedStatus int

	// PollInterval is how often to attempt the request. Defaults to 100ms if zero.
	PollInterval time.Duration
}

// Wait polls the HTTP endpoint until it returns the expected status code or
// the timeout expires. Fails the test if the endpoint does not become ready.
func (w *HTTPWait) Wait(t *testing.T, c *Container, timeout time.Duration) {
	t.Helper()

	hostPort := c.Port(w.Port)
	require.NotEmpty(t, hostPort, "port %s not mapped", w.Port)

	url := fmt.Sprintf("http://%s:%s%s", c.Host, hostPort, w.Path)

	expectedStatus := w.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}

	pollInterval := w.PollInterval
	if pollInterval == 0 {
		pollInterval = 100 * time.Millisecond
	}

	client := &http.Client{
		Timeout: pollInterval,
	}

	require.Eventually(t, func() bool {
		resp, err := client.Get(url)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == expectedStatus
	}, timeout, pollInterval, "HTTP endpoint %s did not return status %d", url, expectedStatus)
}

// NewHTTPWait creates an [HTTPWait] strategy for the given port and path.
// The port should be in the format "port/protocol" (e.g., "9000/tcp").
// The path should include the leading slash (e.g., "/health").
func NewHTTPWait(port, path string) *HTTPWait {
	return &HTTPWait{
		Port:           port,
		Path:           path,
		ExpectedStatus: http.StatusOK,
		PollInterval:   100 * time.Millisecond,
	}
}
