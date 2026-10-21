package peer

import (
	"errors"
	"io"
	"sync"
)

// DefaultReplayLimit bounds how many request body bytes stay buffered until
// the relay acknowledges its dial.
const DefaultReplayLimit = 64 << 10

var (
	errAttemptOver = errors.New("peer: request body attempt is over")
	errAcked       = errors.New("peer: request body was delivered and cannot be replayed")
)

// ReplayBody makes a request body replayable across backhaul attempts until
// one attempt is acknowledged. It records what it reads from the source. Once
// the record reaches the limit, reads wait for the acknowledgement, so the
// record always holds every byte any attempt has sent.
type ReplayBody struct {
	mu      sync.Mutex
	cond    *sync.Cond
	src     io.Reader
	limit   int
	prefix  []byte
	srcErr  error
	acked   bool
	reading bool
	attempt int
}

// NewReplayBody wraps src. Call [ReplayBody.Attempt] for each forward.
func NewReplayBody(src io.Reader, limit int) *ReplayBody {
	b := &ReplayBody{
		mu:      sync.Mutex{},
		cond:    nil,
		src:     src,
		limit:   limit,
		prefix:  nil,
		srcErr:  nil,
		acked:   false,
		reading: false,
		attempt: 0,
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

// Attempt returns a reader for a new attempt that starts at the first byte.
// Earlier attempts stop returning data. It fails once an attempt was
// acknowledged.
func (b *ReplayBody) Attempt() (io.ReadCloser, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.acked {
		return nil, errAcked
	}
	b.attempt++
	b.cond.Broadcast()
	return &replayAttempt{body: b, id: b.attempt, off: 0, closed: false}, nil
}

func (b *ReplayBody) ack(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if id == b.attempt {
		b.acked = true
		b.cond.Broadcast()
	}
}

type replayAttempt struct {
	body   *ReplayBody
	id     int
	off    int
	closed bool
}

// Ack marks this attempt as delivered: the record stops growing and no new
// attempts can start.
func (a *replayAttempt) Ack() { a.body.ack(a.id) }

func (a *replayAttempt) Read(p []byte) (int, error) {
	b := a.body
	b.mu.Lock()
	defer b.mu.Unlock()
	for {
		if a.closed || a.id != b.attempt {
			return 0, errAttemptOver
		}
		if a.off < len(b.prefix) {
			n := copy(p, b.prefix[a.off:])
			a.off += n
			return n, nil
		}
		if b.srcErr != nil {
			return 0, b.srcErr
		}
		if len(p) == 0 {
			return 0, nil
		}
		recording := !b.acked
		if b.reading || (recording && len(b.prefix) >= b.limit) {
			b.cond.Wait()
			continue
		}

		buf := p
		if recording && len(buf) > b.limit-len(b.prefix) {
			buf = buf[:b.limit-len(b.prefix)]
		}
		b.reading = true
		b.mu.Unlock()
		n, err := b.src.Read(buf)
		b.mu.Lock()
		b.reading = false
		b.cond.Broadcast()

		if recording {
			b.prefix = append(b.prefix, buf[:n]...)
		}
		if err != nil {
			b.srcErr = err
		}
		if a.closed || a.id != b.attempt {
			return 0, errAttemptOver
		}
		if recording {
			a.off += n
		}
		return n, err
	}
}

func (a *replayAttempt) Close() error {
	a.body.mu.Lock()
	defer a.body.mu.Unlock()
	a.closed = true
	a.body.cond.Broadcast()
	return nil
}
