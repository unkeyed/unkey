package proxy

import (
	"crypto/tls"
	"errors"
	"os"
	"sync/atomic"
	"time"
)

// spareConn is an idle TLS connection to a peer frontline. Until it is
// claimed, a watcher reads from it to notice when the peer closes it.
type spareConn struct {
	*tls.Conn
	watched chan struct{}
	dead    atomic.Bool
}

func newSpareConn(conn *tls.Conn) *spareConn {
	return &spareConn{Conn: conn, watched: make(chan struct{}), dead: atomic.Bool{}}
}

// watch blocks in Read until the peer closes the connection or claim
// interrupts it. Reading through the TLS layer consumes post-handshake
// messages such as session tickets without losing record state.
func (c *spareConn) watch(onDead func()) {
	defer close(c.watched)

	var b [1]byte
	_, err := c.Read(b[:])
	if errors.Is(err, os.ErrDeadlineExceeded) {
		return
	}
	c.dead.Store(true)
	onDead()
}

// claim stops the watcher and reports whether the connection is still usable.
// A TLS read timeout is not sticky, so the connection stays usable after it.
func (c *spareConn) claim() bool {
	_ = c.SetReadDeadline(time.Unix(1, 0))
	<-c.watched
	if c.dead.Load() {
		return false
	}
	_ = c.SetReadDeadline(time.Time{})
	return true
}
