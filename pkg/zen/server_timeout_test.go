package zen_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestServerIdleTimeoutClosesKeepAliveConnectionWhenReadTimeoutIsDisabled(t *testing.T) {
	t.Parallel()

	srv, err := zen.New(zen.Config{ //nolint:exhaustruct
		ReadTimeout: -1,
		IdleTimeout: 100 * time.Millisecond,
	})
	require.NoError(t, err)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		_ = srv.Serve(ctx, ln)
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	_, err = conn.Write([]byte("GET /health HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	require.NoError(t, err)

	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, err = io.ReadAll(conn)
	require.NoError(t, err, "server should close the connection before the client deadline")
}
