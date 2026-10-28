//go:build linux

package spool_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
	"golang.org/x/sys/unix"
)

// newTestPool returns a pool over a fresh directory and checks, after the
// test, that no temporary file kept a directory entry.
func newTestPool(t *testing.T, cfg spool.Config) *spool.Pool {
	t.Helper()
	dir := newTestDir(t)
	cfg.Dir = dir
	pool, err := spool.NewPool(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		entries, err := os.ReadDir(dir)
		require.NoError(t, err)
		require.Empty(t, entries, "temporary files must not keep a directory entry")
	})
	return pool
}

// newTestDir returns a fresh directory and skips the test when the test
// temporary directory is memory-backed, which the pool rejects.
func newTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	var stat unix.Statfs_t
	require.NoError(t, unix.Statfs(dir, &stat))
	if int64(stat.Type) == unix.TMPFS_MAGIC || int64(stat.Type) == unix.RAMFS_MAGIC {
		t.Skip("the test temporary directory is memory-backed; spool requires a disk-backed Dir")
	}
	return dir
}

// requireExactCapacity proves that every slot and byte was returned to the
// pool, and no more: it fills the pool with full-size bodies, checks that one
// more admission fails, and closes the bodies. cfg.TotalBytesMax must equal
// cfg.ConcurrentMax * cfg.BodyBytesMax.
func requireExactCapacity(t *testing.T, pool *spool.Pool, cfg spool.Config) {
	t.Helper()
	require.Equal(t, cfg.ConcurrentMax*cfg.BodyBytesMax, cfg.TotalBytesMax)
	bodies := make([]*spool.Body, 0, cfg.ConcurrentMax)
	for range cfg.ConcurrentMax {
		body := admitBytes(t, pool, bytes.Repeat([]byte("a"), int(cfg.BodyBytesMax)))
		bodies = append(bodies, body)
	}
	_, err := pool.Admit(context.Background(), newInput(bytes.NewReader([]byte("a"))), 1, readAll)
	requireKind(t, err, spool.FailureCapacity)
	for _, body := range bodies {
		require.NoError(t, body.Close())
	}
}

func admitBytes(t *testing.T, pool *spool.Pool, data []byte) *spool.Body {
	t.Helper()
	body, err := pool.Admit(context.Background(), newInput(bytes.NewReader(data)), int64(len(data)), readAll)
	require.NoError(t, err)
	return body
}

func requireKind(t *testing.T, err error, kind spool.FailureKind) {
	t.Helper()
	failure, ok := errors.AsType[*spool.Error](err)
	require.True(t, ok, "error %v is not a *spool.Error", err)
	require.Equal(t, kind, failure.Kind, "error: %v", err)
}

func requireBodyBytes(t *testing.T, body *spool.Body, want []byte) {
	t.Helper()
	got, err := io.ReadAll(body.Reader())
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.Equal(t, int64(len(want)), body.Size())
}

// readAll is a validator that accepts any body after reading it to EOF.
func readAll(_ context.Context, body io.Reader) error {
	_, err := io.Copy(io.Discard, body)
	return err
}

// input is an io.ReadCloser that counts reads and closes.
type input struct {
	reader   io.Reader
	closeErr error
	reads    atomic.Int64
	closes   atomic.Int64
}

func newInput(reader io.Reader) *input {
	return &input{reader: reader, closeErr: nil, reads: atomic.Int64{}, closes: atomic.Int64{}}
}

func (in *input) Read(p []byte) (int, error) {
	in.reads.Add(1)
	return in.reader.Read(p)
}

// Close also closes the wrapped reader when it is an io.Closer, such as the
// read side of an io.Pipe.
func (in *input) Close() error {
	in.closes.Add(1)
	if closer, ok := in.reader.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return in.closeErr
}

// dataErrReader returns all of its data together with err in one Read.
type dataErrReader struct {
	data []byte
	err  error
}

func (r *dataErrReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
}

// stutterReader returns (0, nil) stutters times before each read of reader.
type stutterReader struct {
	reader   io.Reader
	stutters int
	count    int
}

func (r *stutterReader) Read(p []byte) (int, error) {
	if r.count < r.stutters {
		r.count++
		return 0, nil
	}
	r.count = 0
	return r.reader.Read(p)
}

func patternBytes(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i * 7 % 251)
	}
	return data
}
