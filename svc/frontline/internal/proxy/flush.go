package proxy

import (
	"bufio"
	"net"
	"net/http"
	"sync"
	"time"
)

// flushCoalescer rate-limits Flush calls on the response writer.
//
// Why it exists: customer apps that write more than net/http's ~4 KiB output
// buffer produce chunked responses (ContentLength == -1). For those,
// httputil.ReverseProxy unconditionally switches to flush-on-every-write —
// its flushInterval(res) override, not configurable — which costs one TLS
// record + one write syscall per body chunk. For a response whose chunks
// arrive back-to-back from a local upstream, those flushes carry no latency
// benefit; they just multiply syscalls.
//
// The coalescer keeps the first Flush immediate (time-to-first-byte is
// preserved) and then allows at most one flush per interval, arming a timer
// so trailing data is never stranded. Genuine streams (SSE, gRPC) see at
// most `interval` of added latency per event — bounded and small — while
// burst-chunked responses collapse into one or two flushes.
type flushCoalescer struct {
	dst      http.ResponseWriter
	interval time.Duration

	mu        sync.Mutex
	lastFlush time.Time
	timer     *time.Timer
	dirty     bool
}

// flushCoalesceInterval bounds the extra latency a streamed chunk can see.
// 2ms collapses back-to-back chunk flushes (which arrive microseconds apart)
// while staying invisible next to real network RTTs.
const flushCoalesceInterval = 2 * time.Millisecond

func newFlushCoalescer(dst http.ResponseWriter) *flushCoalescer {
	//nolint:exhaustruct
	return &flushCoalescer{dst: dst, interval: flushCoalesceInterval}
}

func (f *flushCoalescer) Header() http.Header { return f.dst.Header() }

func (f *flushCoalescer) WriteHeader(code int) { f.dst.WriteHeader(code) }

func (f *flushCoalescer) Write(p []byte) (int, error) {
	f.mu.Lock()
	f.dirty = true
	f.mu.Unlock()
	return f.dst.Write(p)
}

// Flush implements http.Flusher with rate limiting. The first flush goes
// through immediately; subsequent flushes within the interval arm a timer
// that performs one deferred flush at the interval boundary.
func (f *flushCoalescer) Flush() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	if now.Sub(f.lastFlush) >= f.interval {
		f.flushLocked(now)
		return
	}

	// Within the rate limit: defer to a single timer at the boundary.
	if f.timer == nil {
		delay := f.interval - now.Sub(f.lastFlush)
		f.timer = time.AfterFunc(delay, func() {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.timer = nil
			if f.dirty {
				f.flushLocked(time.Now())
			}
		})
	}
}

func (f *flushCoalescer) flushLocked(now time.Time) {
	if fl, ok := f.dst.(http.Flusher); ok {
		fl.Flush()
	}
	f.lastFlush = now
	f.dirty = false
}

// Close stops the deferred-flush timer and performs a final flush if data is
// pending. Must be called when the proxied request finishes; the server
// flushes on handler return anyway, so this mainly stops the timer.
func (f *flushCoalescer) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.timer != nil {
		f.timer.Stop()
		f.timer = nil
	}
	if f.dirty {
		f.flushLocked(time.Now())
	}
}

// Hijack passes connection takeover through to the underlying writer
// (WebSocket upgrades). After a hijack the coalescer is out of the path.
func (f *flushCoalescer) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := f.dst.(http.Hijacker); ok {
		return hj.Hijack()
	}
	//nolint:exhaustruct
	return nil, nil, http.ErrNotSupported
}

// Unwrap supports http.ResponseController passthrough.
func (f *flushCoalescer) Unwrap() http.ResponseWriter { return f.dst }
