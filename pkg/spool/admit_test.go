//go:build linux

package spool_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
)

// TestAdmit_StoresExactlyTheBytesTheValidatorRead guarantees that the replayed
// body is byte-for-byte the input the validator accepted, however the input
// splits its reads. A mismatch would forward bytes that were never validated.
func TestAdmit_StoresExactlyTheBytesTheValidatorRead(t *testing.T) {
	t.Parallel()
	data := patternBytes(3*32<<10 + 7)
	readers := []struct {
		name string
		wrap func(io.Reader) io.Reader
	}{
		{name: "whole reads", wrap: func(r io.Reader) io.Reader { return r }},
		{name: "one byte per read", wrap: iotest.OneByteReader},
		{name: "half reads", wrap: iotest.HalfReader},
		{name: "last data returned with EOF", wrap: iotest.DataErrReader},
		{name: "repeated empty reads", wrap: func(r io.Reader) io.Reader { return &stutterReader{reader: r, stutters: 99, count: 0} }},
	}
	for _, reader := range readers {
		for _, contentLength := range []int64{int64(len(data)), -1} {
			t.Run(reader.name, func(t *testing.T) {
				t.Parallel()
				pool := newTestPool(t, spool.Config{Dir: "", BodyBytesMax: 1 << 20, TotalBytesMax: 1 << 20, ConcurrentMax: 1})
				in := newInput(reader.wrap(bytes.NewReader(data)))
				var seen bytes.Buffer
				body, err := pool.Admit(context.Background(), in, contentLength, func(_ context.Context, r io.Reader) error {
					_, copyErr := io.Copy(&seen, r)
					return copyErr
				})
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, body.Close()) })

				require.Equal(t, data, seen.Bytes())
				requireBodyBytes(t, body, data)
				require.Equal(t, int64(1), in.closes.Load())
			})
		}
	}
}

// TestAdmit_FailuresReturnNoBodyAndReleaseEverything guarantees that every
// rejected admission closes the input exactly once, frees its slot and bytes,
// leaves no file behind, and reports the failure that caused it rather than a
// consequence seen by the validator.
func TestAdmit_FailuresReturnNoBodyAndReleaseEverything(t *testing.T) {
	t.Parallel()
	const bodyBytesMax = 64
	errRead := errors.New("connection reset")
	errClose := errors.New("close failed")
	errInvalid := errors.New("schema violation")
	rejectAfterRead := func(_ context.Context, r io.Reader) error {
		if _, err := io.Copy(io.Discard, r); err != nil {
			return err
		}
		return errInvalid
	}
	alwaysInvalid := func(_ context.Context, r io.Reader) error {
		_, _ = io.Copy(io.Discard, r)
		return errInvalid
	}
	ignoreReadErrors := func(_ context.Context, r io.Reader) error {
		_, _ = io.Copy(io.Discard, r)
		return nil
	}
	readOneByte := func(_ context.Context, r io.Reader) error {
		_, err := r.Read(make([]byte, 1))
		return err
	}
	readNothing := func(context.Context, io.Reader) error { return nil }

	tests := []struct {
		name          string
		reader        io.Reader
		contentLength int64
		closeErr      error
		canceled      bool
		validate      spool.ValidateFunc
		wantKind      spool.FailureKind
		wantErrs      []error
		// readsMax bounds input reads; -1 means unbounded.
		readsMax int64
		// seenMax bounds the bytes the validator may see; -1 means unbounded.
		seenMax int64
	}{
		{
			name:          "declared length above the limit is rejected before any read",
			reader:        strings.NewReader(strings.Repeat("a", bodyBytesMax+1)),
			contentLength: bodyBytesMax + 1,
			validate:      readAll,
			wantKind:      spool.FailureTooLarge,
			readsMax:      0,
			seenMax:       0,
		},
		{
			name:          "unknown-length input above the limit is rejected before the validator sees extra bytes",
			reader:        strings.NewReader(strings.Repeat("a", bodyBytesMax+1)),
			contentLength: -1,
			validate:      readAll,
			wantKind:      spool.FailureTooLarge,
			readsMax:      -1,
			seenMax:       bodyBytesMax,
		},
		{
			name:          "input longer than its declared length is a false length",
			reader:        strings.NewReader(strings.Repeat("a", 11)),
			contentLength: 10,
			validate:      readAll,
			wantKind:      spool.FailureSourceRead,
			readsMax:      -1,
			seenMax:       10,
		},
		{
			name:          "input shorter than its declared length is a false length",
			reader:        strings.NewReader(strings.Repeat("a", 9)),
			contentLength: 10,
			validate:      readAll,
			wantKind:      spool.FailureSourceRead,
			readsMax:      -1,
			seenMax:       9,
		},
		{
			name:          "read error after data",
			reader:        io.MultiReader(strings.NewReader("abc"), iotest.ErrReader(errRead)),
			contentLength: -1,
			validate:      readAll,
			wantKind:      spool.FailureSourceRead,
			wantErrs:      []error{errRead},
			readsMax:      -1,
			seenMax:       3,
		},
		{
			name:          "data returned together with a read error is rejected",
			reader:        &dataErrReader{data: []byte("abc"), err: errRead},
			contentLength: -1,
			validate:      ignoreReadErrors,
			wantKind:      spool.FailureSourceRead,
			wantErrs:      []error{errRead},
			readsMax:      -1,
			seenMax:       0,
		},
		{
			name:          "EOF joined with a source error cannot accept the body",
			reader:        &dataErrReader{data: []byte("abc"), err: errors.Join(io.EOF, errRead)},
			contentLength: 3,
			validate:      ignoreReadErrors,
			wantKind:      spool.FailureSourceRead,
			wantErrs:      []error{errRead},
			readsMax:      1,
			seenMax:       0,
		},
		{
			name:          "a validator that ignores a read error cannot accept the body",
			reader:        io.MultiReader(strings.NewReader("abc"), iotest.ErrReader(errRead)),
			contentLength: -1,
			validate:      ignoreReadErrors,
			wantKind:      spool.FailureSourceRead,
			wantErrs:      []error{errRead},
			readsMax:      -1,
			seenMax:       -1,
		},
		{
			name:          "upstream MaxBytesReader limit",
			reader:        iotest.ErrReader(&http.MaxBytesError{Limit: 5}),
			contentLength: -1,
			validate:      readAll,
			wantKind:      spool.FailureTooLarge,
			readsMax:      -1,
			seenMax:       -1,
		},
		{
			name:          "input that makes no progress",
			reader:        &stutterReader{reader: strings.NewReader("a"), stutters: 1000, count: 0},
			contentLength: -1,
			validate:      readAll,
			wantKind:      spool.FailureSourceRead,
			wantErrs:      []error{io.ErrNoProgress},
			readsMax:      100,
			seenMax:       0,
		},
		{
			name:          "validator rejection",
			reader:        strings.NewReader("abc"),
			contentLength: 3,
			validate:      rejectAfterRead,
			wantKind:      spool.FailureInvalid,
			wantErrs:      []error{errInvalid},
			readsMax:      -1,
			seenMax:       -1,
		},
		{
			name:          "a limit failure wins over the validator error it caused",
			reader:        strings.NewReader(strings.Repeat("a", bodyBytesMax+1)),
			contentLength: -1,
			validate:      alwaysInvalid,
			wantKind:      spool.FailureTooLarge,
			readsMax:      -1,
			seenMax:       bodyBytesMax,
		},
		{
			name:          "validator accepts with buffered bytes still unread",
			reader:        strings.NewReader("abc"),
			contentLength: 3,
			validate:      readOneByte,
			wantKind:      spool.FailureValidatorContract,
			readsMax:      -1,
			seenMax:       -1,
		},
		{
			name:          "validator accepts without reading, and the probe does not drain the input",
			reader:        iotest.OneByteReader(strings.NewReader("abcdef")),
			contentLength: 6,
			validate:      readNothing,
			wantKind:      spool.FailureValidatorContract,
			readsMax:      1,
			seenMax:       0,
		},
		{
			name:          "context already canceled",
			reader:        strings.NewReader("abc"),
			contentLength: 3,
			canceled:      true,
			validate:      readAll,
			wantKind:      spool.FailureCanceled,
			wantErrs:      []error{context.Canceled},
			readsMax:      0,
			seenMax:       0,
		},
		{
			name:          "input close error after a valid body",
			reader:        strings.NewReader("abc"),
			contentLength: 3,
			closeErr:      errClose,
			validate:      readAll,
			wantKind:      spool.FailureSourceRead,
			wantErrs:      []error{errClose},
			readsMax:      -1,
			seenMax:       -1,
		},
		{
			name:          "input close error joins an earlier failure",
			reader:        strings.NewReader("abc"),
			contentLength: 3,
			closeErr:      errClose,
			validate:      rejectAfterRead,
			wantKind:      spool.FailureInvalid,
			wantErrs:      []error{errInvalid, errClose},
			readsMax:      -1,
			seenMax:       -1,
		},
		{
			name:          "nil validator",
			reader:        strings.NewReader("abc"),
			contentLength: 3,
			validate:      nil,
			wantKind:      spool.FailureInternal,
			readsMax:      0,
			seenMax:       0,
		},
		{
			name:          "content length below -1",
			reader:        strings.NewReader("abc"),
			contentLength: -2,
			validate:      readAll,
			wantKind:      spool.FailureInternal,
			readsMax:      0,
			seenMax:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := spool.Config{Dir: "", BodyBytesMax: bodyBytesMax, TotalBytesMax: bodyBytesMax, ConcurrentMax: 1}
			pool := newTestPool(t, cfg)
			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)
			if tt.canceled {
				cancel()
			}
			in := newInput(tt.reader)
			in.closeErr = tt.closeErr

			var seen int64
			validate := tt.validate
			if validate != nil {
				validate = func(ctx context.Context, r io.Reader) error {
					return tt.validate(ctx, &countingReader{reader: r, count: &seen})
				}
			}
			body, err := pool.Admit(ctx, in, tt.contentLength, validate)

			require.Nil(t, body)
			requireKind(t, err, tt.wantKind)
			for _, want := range tt.wantErrs {
				require.ErrorIs(t, err, want)
			}
			require.Equal(t, int64(1), in.closes.Load(), "input must be closed exactly once")
			if tt.readsMax >= 0 {
				require.LessOrEqual(t, in.reads.Load(), tt.readsMax)
			}
			if tt.seenMax >= 0 {
				require.LessOrEqual(t, seen, tt.seenMax)
			}
			requireExactCapacity(t, pool, cfg)
		})
	}
}

// TestAdmit_RejectsNilBody guarantees that a nil body fails as a caller bug
// instead of being treated as an unvalidated empty body.
func TestAdmit_RejectsNilBody(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)

	body, err := pool.Admit(context.Background(), nil, 0, readAll)

	require.Nil(t, body)
	requireKind(t, err, spool.FailureInternal)
	requireExactCapacity(t, pool, cfg)
}

func TestAdmit_TemporaryFileHasNoDirectoryEntry(t *testing.T) {
	t.Parallel()
	dir := newTestDir(t)
	pool, err := spool.NewPool(spool.Config{Dir: dir, BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1})
	require.NoError(t, err)

	body, err := pool.Admit(context.Background(), newInput(strings.NewReader("abc")), 3, func(ctx context.Context, r io.Reader) error {
		entries, readDirErr := os.ReadDir(dir)
		require.NoError(t, readDirErr)
		require.Empty(t, entries)
		return readAll(ctx, r)
	})
	require.NoError(t, err)
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, entries)
	require.NoError(t, body.Close())
}

// TestAdmit_ValidatorPanicReleasesEverything guarantees that a panicking
// validator cannot leak a slot, bytes, the input, or the file, and that the
// panic reaches the caller unchanged.
func TestAdmit_ValidatorPanicReleasesEverything(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)
	in := newInput(strings.NewReader("abc"))

	require.PanicsWithValue(t, "validator bug", func() {
		_, _ = pool.Admit(context.Background(), in, 3, func(context.Context, io.Reader) error {
			panic("validator bug")
		})
	})

	require.Equal(t, int64(1), in.closes.Load())
	requireExactCapacity(t, pool, cfg)
}

// countingReader counts the bytes a validator reads.
type countingReader struct {
	reader io.Reader
	count  *int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	*r.count += int64(n)
	return n, err
}
