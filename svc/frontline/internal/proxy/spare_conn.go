package proxy

import (
	"crypto/tls"
	"errors"
	"os"
	"time"
)

// spareConn is an idle TLS connection to a peer frontline. Until it is
// claimed, a watcher reads from it to notice when the peer closes it.
type spareConn struct {
	*tls.Conn
	born    time.Time
	watched chan struct{}
	// dead is written by the watcher before it closes watched and read only
	// after watched is closed.
	dead bool
}

func newSpareConn(conn *tls.Conn, born time.Time) *spareConn {
	return &spareConn{Conn: conn, born: born, watched: make(chan struct{}), dead: false}
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
	c.dead = true
	onDead()
}

// claim stops the watcher and reports whether the connection is still usable.
// A TLS read timeout is not sticky, so the connection stays usable after it.
func (c *spareConn) claim() bool {
	_ = c.SetReadDeadline(time.Unix(1, 0))
	<-c.watched
	if c.dead {
		return false
	}
	if err := c.SetReadDeadline(time.Time{}); err != nil {
		_ = c.Close()
		return false
	}
	return true
}
