package spool

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/unkeyed/unkey/pkg/fault"
)

// noProgressReadsMax matches bufio: after this many consecutive (0, nil)
// reads, the input counts as broken.
const noProgressReadsMax = 100

// tee is the reader a validator sees. Each chunk read from input is written to
// file before any of its bytes are returned, so the file always holds exactly
// the bytes taken from input, and never more than limit. The first failure is
// sticky: after it, Read returns only that failure.
type tee struct {
	ctx         context.Context
	input       io.Reader
	file        *os.File
	buf         []byte
	pending     []byte
	written     int64
	limit       int64
	lengthKnown bool
	eof         bool
	failure     *Error
}

func (t *tee) Read(p []byte) (int, error) {
	if t.failure != nil {
		return 0, t.failure
	}
	if len(p) == 0 {
		return 0, nil
	}
	if len(t.pending) == 0 {
		t.fill()
	}
	if t.failure != nil {
		return 0, t.failure
	}
	if len(t.pending) > 0 {
		n := copy(p, t.pending)
		t.pending = t.pending[n:]
		return n, nil
	}
	return 0, io.EOF
}

// verdict decides the result of an admission after validate returned
// validateErr. A failure of the tee wins over the validator's error, because
// the validator only saw the consequence. After a nil validateErr it probes
// for EOF with at most one fill and never drains further input.
func (t *tee) verdict(validateErr error) *Error {
	if t.failure != nil {
		return t.failure
	}
	if t.ctx.Err() != nil {
		return &Error{Kind: FailureCanceled, Err: errors.Join(contextErr(t.ctx), validateErr)}
	}
	if validateErr != nil {
		return &Error{Kind: FailureInvalid, Err: validateErr}
	}
	if len(t.pending) == 0 && !t.eof {
		t.fill()
	}
	if t.failure != nil {
		return t.failure
	}
	if len(t.pending) > 0 || !t.eof {
		return &Error{Kind: FailureValidatorContract, Err: fault.New("validator accepted the body before the input reached EOF")}
	}
	return nil
}

func (t *tee) fill() {
	for range noProgressReadsMax {
		if t.failure != nil || t.eof {
			return
		}
		if t.ctx.Err() != nil {
			t.fail(FailureCanceled, contextErr(t.ctx))
			return
		}
		size := t.readSize()
		n, err := t.input.Read(t.buf[:size])
		if n < 0 || n > size {
			t.fail(FailureSourceRead, fault.New("input returned an invalid read count"))
			return
		}
		if n > 0 {
			t.store(t.buf[:n])
		}
		if err != nil {
			t.readFailed(err)
		}
		if n > 0 || err != nil {
			return
		}
	}
	t.fail(FailureSourceRead, fault.Wrap(io.ErrNoProgress, fault.Internal("input made no progress")))
}

// readSize allows one byte past the limit, so an input that is too long is
// detected without reading a full extra chunk.
func (t *tee) readSize() int {
	room := t.limit - t.written
	if room < int64(len(t.buf)) {
		return int(room) + 1
	}
	return len(t.buf)
}

func (t *tee) store(chunk []byte) {
	if int64(len(chunk)) > t.limit-t.written {
		if t.lengthKnown {
			t.fail(FailureSourceRead, fault.New("input is longer than its declared content length"))
		} else {
			t.fail(FailureTooLarge, fault.New("input exceeds the body size limit"))
		}
		return
	}
	if _, err := t.file.Write(chunk); err != nil {
		t.fail(FailureStorage, fault.Wrap(err, fault.Internal("writing temporary file failed")))
		return
	}
	t.written += int64(len(chunk))
	t.pending = chunk
}

func (t *tee) readFailed(err error) {
	if err == io.EOF {
		t.eof = true
		if t.lengthKnown && t.written != t.limit {
			t.fail(FailureSourceRead, fault.New("input is shorter than its declared content length"))
		}
		return
	}
	if t.ctx.Err() != nil {
		t.fail(FailureCanceled, errors.Join(contextErr(t.ctx), err))
		return
	}
	if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
		t.fail(FailureTooLarge, err)
		return
	}
	t.fail(FailureSourceRead, err)
}

func (t *tee) fail(kind FailureKind, err error) {
	if t.failure == nil {
		t.failure = &Error{Kind: kind, Err: err}
	}
}
