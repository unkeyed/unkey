package server

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"github.com/stretchr/testify/require"
)

// TestUDPShutdownBeforeStart guarantees that a shutdown racing startup neither
// panics inside the miekg server nor blocks past its context.
func TestUDPShutdownBeforeStart(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := conn.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Error(err)
		}
	})
	server := NewUDP(conn, rejectingHandler(t))
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	t.Cleanup(cancel)
	require.ErrorIs(t, server.Shutdown(ctx), context.DeadlineExceeded)
}

// TestProbeChecksBothTransports guarantees that readiness probes succeed
// against running servers without reaching the handler, and fail once a
// server has shut down.
func TestProbeChecksBothTransports(t *testing.T) {
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	udp := NewUDP(conn, rejectingHandler(t))
	served := make(chan error, 1)
	go func() { served <- udp.Serve(t.Context()) }()
	require.Eventually(t, func() bool {
		return Probe(t.Context(), "udp", conn.LocalAddr().String()) == nil
	}, 5*time.Second, 10*time.Millisecond)

	tcp, _ := startTCPTest(t, rejectingHandler(t))
	require.NoError(t, Probe(t.Context(), "tcp", tcp.listener.Addr().String()))

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	t.Cleanup(cancel)
	require.NoError(t, udp.Shutdown(ctx))
	require.NoError(t, <-served)
	require.NoError(t, udp.Shutdown(ctx), "a repeated shutdown after Serve returned must not block")
	require.Error(t, Probe(t.Context(), "udp", conn.LocalAddr().String()))
}

func rejectingHandler(t *testing.T) dnswire.Handler {
	t.Helper()
	return dnswire.HandlerFunc(func(context.Context, dnswire.ResponseWriter, *dnswire.Msg) {
		t.Error("probe reached the handler")
	})
}
