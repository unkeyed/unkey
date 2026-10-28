package spool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/unkeyed/unkey/pkg/fault"
)

const chunkBytes = 32 << 10

// ValidateFunc validates one body. It must read body only on the calling
// goroutine and must not use body after it returns. It returns nil only after
// it read body to io.EOF and checked the complete document; the pool trusts
// that result because a validator can buffer input internally. It returns the
// first failure, and must return an error whenever a Read returns an error
// other than io.EOF.
type ValidateFunc func(ctx context.Context, body io.Reader) error

// Config sets the storage location and limits of a [Pool].
type Config struct {
	// Dir is an absolute path to a directory on a disk-backed filesystem.
	// [NewPool] rejects tmpfs and ramfs. There is no default, because the
	// usual temporary directory is often memory-backed or read-only.
	Dir string
	// BodyBytesMax is the largest body one admission accepts.
	BodyBytesMax int64
	// TotalBytesMax bounds the bytes reserved by all admissions and open
	// bodies of the pool. It must be at least BodyBytesMax.
	TotalBytesMax int64
	// ConcurrentMax bounds admissions in progress plus open bodies.
	ConcurrentMax int64
}

// Pool admits bodies within shared byte and concurrency limits. It is safe
// for concurrent use and owns no goroutines or files, so it needs no Close.
type Pool struct {
	cfg           Config
	mu            sync.Mutex
	bodiesOpen    int64
	bytesReserved int64
}

// NewPool checks cfg and Dir and returns a pool. Dir must exist, be writable,
// and not be tmpfs or ramfs. NewPool fails on systems other than Linux.
func NewPool(cfg Config) (*Pool, error) {
	switch {
	case !filepath.IsAbs(cfg.Dir):
		return nil, fault.New("spool: Dir must be an absolute path")
	case cfg.BodyBytesMax <= 0:
		return nil, fault.New("spool: BodyBytesMax must be positive")
	case cfg.TotalBytesMax < cfg.BodyBytesMax:
		return nil, fault.New("spool: TotalBytesMax must be at least BodyBytesMax")
	case cfg.ConcurrentMax <= 0:
		return nil, fault.New("spool: ConcurrentMax must be positive")
	}
	if err := checkDir(cfg.Dir); err != nil {
		return nil, fault.Wrap(err, fault.Internal("spool: Dir is not usable"))
	}
	return &Pool{cfg: cfg, mu: sync.Mutex{}, bodiesOpen: 0, bytesReserved: 0}, nil
}

// Admit reads body through validate and stores the same bytes in a temporary
// file. It returns a [Body] only if validate returned nil, the input was at
// EOF, the size matches contentLength, and ctx was not done at the commit
// point. Otherwise it returns an [*Error] and no Body.
//
// Admit owns body. It closes body exactly once before it returns, on every
// path, and reports a close error. A nil body is rejected; pass
// [net/http.NoBody] for an empty body. Admit does not drain body after a
// rejection. body.Close must be safe to call while a Read is blocked, because
// a canceled ctx closes body to unblock that Read. For a net/http server body,
// Close waits for a blocked Read to return, so bound reads with a read deadline.
//
// contentLength is the declared size, or -1 if unknown. A known length
// reserves exactly that many bytes and is a hard bound: a longer or shorter
// input fails with [FailureSourceRead]. An unknown length reserves
// [Config.BodyBytesMax] and returns the unused bytes to the pool on success.
// Any length above BodyBytesMax fails with [FailureTooLarge].
//
// The commit point follows input EOF, the input close, and the stop of the
// cancellation callback. A ctx canceled after the commit point does not affect
// the returned Body; the caller owns it and must call [Body.Close].
func (p *Pool) Admit(ctx context.Context, body io.ReadCloser, contentLength int64, validate ValidateFunc) (*Body, error) {
	if body == nil {
		return nil, &Error{Kind: FailureInternal, Err: fault.New("body is nil; pass http.NoBody for an empty body")}
	}
	closeInput := sync.OnceValue(func() error {
		if err := body.Close(); err != nil {
			return fault.Wrap(err, fault.Internal("closing input failed"))
		}
		return nil
	})

	var rejection *Error
	switch {
	case ctx == nil:
		rejection = &Error{Kind: FailureInternal, Err: fault.New("ctx is nil")}
	case validate == nil:
		rejection = &Error{Kind: FailureInternal, Err: fault.New("validate is nil")}
	case contentLength < -1:
		rejection = &Error{Kind: FailureInternal, Err: fault.New(fmt.Sprintf("content length %d is invalid; use -1 for unknown", contentLength))}
	case ctx.Err() != nil:
		rejection = &Error{Kind: FailureCanceled, Err: contextErr(ctx)}
	case contentLength > p.cfg.BodyBytesMax:
		rejection = &Error{Kind: FailureTooLarge, Err: fault.New(fmt.Sprintf("content length %d exceeds the limit of %d bytes", contentLength, p.cfg.BodyBytesMax))}
	}
	if rejection != nil {
		return nil, rejection.withCleanup(closeInput())
	}

	limit := p.cfg.BodyBytesMax
	if contentLength >= 0 {
		limit = contentLength
	}
	if err := p.reserve(limit); err != nil {
		return nil, (&Error{Kind: FailureCapacity, Err: err}).withCleanup(closeInput())
	}
	file, err := openTempFile(p.cfg.Dir)
	if err != nil {
		return nil, (&Error{Kind: FailureStorage, Err: err}).withCleanup(closeInput(), p.release(limit))
	}
	return p.spool(ctx, body, closeInput, file, limit, contentLength >= 0, validate)
}

// spool runs validate over a tee of input into file and commits or cleans up.
// The deferred cleanup also runs when validate panics or calls runtime.Goexit.
func (p *Pool) spool(
	ctx context.Context,
	input io.Reader,
	closeInput func() error,
	file *os.File,
	limit int64,
	lengthKnown bool,
	validate ValidateFunc,
) (_ *Body, err error) {
	t := &tee{
		ctx:         ctx,
		input:       input,
		file:        file,
		buf:         make([]byte, chunkBytes),
		pending:     nil,
		written:     0,
		limit:       limit,
		lengthKnown: lengthKnown,
		eof:         false,
		failure:     nil,
	}

	callbackDone := make(chan struct{})
	stopAfterFunc := context.AfterFunc(ctx, func() {
		defer close(callbackDone)
		// closeInput keeps the error; the admission path reads it.
		_ = closeInput()
	})
	// stopCallback returns only when the callback can no longer run: stop
	// alone does not wait for a callback that already started. It runs once,
	// because a second stop also returns false when the first one succeeded.
	stopCallback := sync.OnceFunc(func() {
		if !stopAfterFunc() {
			<-callbackDone
		}
	})

	committed := false
	defer func() {
		if committed {
			return
		}
		stopCallback()
		cleanupErr := errors.Join(closeInput(), closeFile(file), p.release(limit))
		if failure, ok := errors.AsType[*Error](err); ok {
			err = failure.withCleanup(cleanupErr)
			return
		}
		// Only a panic or runtime.Goexit in validate reaches this point.
		if cleanupErr != nil {
			panic(errors.Join(fmt.Errorf("spool: admission did not finish: %v", recover()), cleanupErr))
		}
	}()

	if failure := t.verdict(validate(ctx, t)); failure != nil {
		return nil, failure
	}
	if closeInput() != nil {
		// The deferred cleanup joins the close error itself.
		return nil, &Error{Kind: FailureSourceRead, Err: fault.New("input close failed after a valid body")}
	}
	stopCallback()
	if ctx.Err() != nil {
		return nil, &Error{Kind: FailureCanceled, Err: contextErr(ctx)}
	}
	if refundErr := p.refund(limit - t.written); refundErr != nil {
		return nil, &Error{Kind: FailureInternal, Err: refundErr}
	}
	committed = true
	return newBody(p, file, t.written), nil
}

func (p *Pool) reserve(n int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.bodiesOpen >= p.cfg.ConcurrentMax {
		return fault.New(fmt.Sprintf("all %d body slots are in use", p.cfg.ConcurrentMax))
	}
	if n > p.cfg.TotalBytesMax-p.bytesReserved {
		return fault.New(fmt.Sprintf("reserving %d bytes would exceed the pool limit of %d bytes", n, p.cfg.TotalBytesMax))
	}
	p.bodiesOpen++
	p.bytesReserved += n
	return nil
}

func (p *Pool) refund(n int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < 0 || n > p.bytesReserved || p.bodiesOpen < 1 {
		return fault.New(fmt.Sprintf("spool accounting broken: refund %d of %d bytes with %d bodies open", n, p.bytesReserved, p.bodiesOpen))
	}
	p.bytesReserved -= n
	return nil
}

func (p *Pool) release(n int64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if n < 0 || n > p.bytesReserved || p.bodiesOpen < 1 {
		return fault.New(fmt.Sprintf("spool accounting broken: release %d of %d bytes with %d bodies open", n, p.bytesReserved, p.bodiesOpen))
	}
	p.bodiesOpen--
	p.bytesReserved -= n
	return nil
}

func (e *Error) withCleanup(errs ...error) *Error {
	if cleanupErr := errors.Join(errs...); cleanupErr != nil {
		e.Err = errors.Join(e.Err, cleanupErr)
	}
	return e
}

func closeFile(file *os.File) error {
	if err := file.Close(); err != nil {
		return fault.Wrap(err, fault.Internal("closing temporary file failed"))
	}
	return nil
}

func contextErr(ctx context.Context) error {
	err := ctx.Err()
	if cause := context.Cause(ctx); cause != nil && cause != err {
		return errors.Join(err, cause)
	}
	return err
}
