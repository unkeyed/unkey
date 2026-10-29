package validation

import (
	"errors"
	"io"
	"sync"

	"github.com/unkeyed/unkey/pkg/fault"
)

const processingBytes = 64 << 10

var (
	// ErrCapacity means the shared admission or retained-byte budget is full.
	ErrCapacity = errors.New("openapi validation capacity exhausted")
	// ErrBodyTooLarge means the request exceeds its validation path's byte limit.
	ErrBodyTooLarge = errors.New("openapi request body exceeds byte limit")
	// ErrContentLength means the received body differs from its declared length.
	ErrContentLength = errors.New("openapi request body differs from content length")
)

// Limits bounds replay storage, reserved fallback copy headroom, and concurrent
// requests, not decoded allocations or total process memory. BufferedBodyBytesMax
// keeps a separate, conservative cap on requests that require that evaluator.
type Limits struct {
	BodyBytesMax         int64
	BufferedBodyBytesMax int64
	BytesMax             int64
	RequestsMax          int64
}

// Budget is shared across plans. Chunk capacity and a fixed read-buffer allowance
// remain reserved until the accepted body closes. Fallback also reserves temporary
// copy headroom while the existing evaluator runs.
type Budget struct {
	limits Limits
	mu     sync.Mutex
	bytes  int64
	active int64
}

// NewBudget validates limits without allocating request storage.
func NewBudget(limits Limits) (*Budget, error) {
	if limits.BodyBytesMax <= 0 || limits.BufferedBodyBytesMax <= 0 || limits.BufferedBodyBytesMax > limits.BodyBytesMax || limits.BytesMax <= processingBytes || limits.RequestsMax <= 0 {
		return nil, fault.New("invalid openapi validation limits")
	}
	return &Budget{limits: limits, mu: sync.Mutex{}, bytes: 0, active: 0}, nil
}

func (b *Budget) acquire() (*retainedBody, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active >= b.limits.RequestsMax || processingBytes > b.limits.BytesMax-b.bytes {
		return nil, ErrCapacity
	}
	b.active++
	b.bytes += processingBytes
	return &retainedBody{budget: b, mu: sync.Mutex{}, chunks: nil, size: 0, reserved: processingBytes, closed: false, index: 0, offset: 0}, nil
}

type retainedBody struct {
	budget   *Budget
	mu       sync.Mutex
	chunks   [][]byte
	size     int64
	reserved int64
	closed   bool
	index    int
	offset   int
}

func (b *retainedBody) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		if len(b.chunks) == 0 || len(b.chunks[len(b.chunks)-1]) == cap(b.chunks[len(b.chunks)-1]) {
			size := 512
			if len(b.chunks) > 0 {
				size = min(cap(b.chunks[len(b.chunks)-1])*2, 32<<10)
			}
			b.budget.mu.Lock()
			if int64(size) > b.budget.limits.BytesMax-b.budget.bytes {
				b.budget.mu.Unlock()
				return written, ErrCapacity
			}
			b.budget.bytes += int64(size)
			b.budget.mu.Unlock()
			b.reserved += int64(size)
			b.chunks = append(b.chunks, make([]byte, 0, size))
		}
		i := len(b.chunks) - 1
		n := min(len(p), cap(b.chunks[i])-len(b.chunks[i]))
		b.chunks[i] = append(b.chunks[i], p[:n]...)
		b.size += int64(n)
		written += n
		p = p[n:]
	}
	return written, nil
}

func (b *retainedBody) Read(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.read(p, &b.index, &b.offset)
}

func (b *retainedBody) read(p []byte, index, offset *int) (int, error) {
	if b.closed {
		return 0, io.ErrClosedPipe
	}
	if len(p) == 0 {
		return 0, nil
	}
	if *index == len(b.chunks) {
		return 0, io.EOF
	}
	n := copy(p, b.chunks[*index][*offset:])
	*offset += n
	if *offset == len(b.chunks[*index]) {
		*index++
		*offset = 0
	}
	return n, nil
}

func (b *retainedBody) reader() io.ReadCloser {
	return &retainedReader{body: b, index: 0, offset: 0}
}

func (b *retainedBody) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.closed {
		b.closed = true
		b.chunks = nil
		b.budget.mu.Lock()
		b.budget.bytes -= b.reserved
		b.budget.active--
		b.budget.mu.Unlock()
	}
	return nil
}

type retainedReader struct {
	body   *retainedBody
	index  int
	offset int
}

func (r *retainedReader) Read(p []byte) (int, error) {
	r.body.mu.Lock()
	defer r.body.mu.Unlock()
	return r.body.read(p, &r.index, &r.offset)
}

func (r *retainedReader) Close() error { return nil }
