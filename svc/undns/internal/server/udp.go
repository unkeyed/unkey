package server

import (
	"context"
	"net"
	"sync"
	"time"

	dnswire "codeberg.org/miekg/dns"
)

// UDP serves DNS over UDP with the miekg server. Responses are limited to 1232
// bytes, the EDNS size that avoids IP fragmentation.
type UDP struct {
	server   *dnswire.Server
	started  chan struct{}
	stopped  chan struct{}
	shutdown sync.Once
}

// NewUDP returns a server that takes ownership of conn and answers with
// handler.
func NewUDP(conn net.PacketConn, handler dnswire.Handler) *UDP {
	u := &UDP{server: new(dnswire.Server), started: make(chan struct{}), stopped: make(chan struct{}), shutdown: sync.Once{}}
	u.server.PacketConn = conn
	u.server.Handler = handler
	u.server.UDPSize = 1232
	u.server.ReadTimeout = 5 * time.Second
	u.server.NotifyStartedFunc = func(context.Context) { close(u.started) }
	return u
}

// Serve answers queries until [UDP.Shutdown]. It returns nil after Shutdown or
// an error if the server can't start. Call it once.
func (u *UDP) Serve(context.Context) error {
	defer close(u.stopped)
	return u.server.ListenAndServe()
}

// Shutdown stops serving and closes the connection. The miekg server panics
// if it is shut down before it starts or twice, so Shutdown waits for Serve to
// start, returning early if Serve already returned or ctx expires, and only
// the first call shuts the server down.
func (u *UDP) Shutdown(ctx context.Context) error {
	select {
	case <-u.started:
		u.shutdown.Do(func() { u.server.Shutdown(ctx) })
		return nil
	case <-u.stopped:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
