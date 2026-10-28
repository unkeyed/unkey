//go:build linux

package spool_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/spool"
)

// TestAdmit_SlotIsHeldUntilBodyClose guarantees that a body keeps its slot
// while it is forwarded, and that a full pool fails fast without reading the
// new input.
func TestAdmit_SlotIsHeldUntilBodyClose(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 8, TotalBytesMax: 8, ConcurrentMax: 1}
	pool := newTestPool(t, cfg)
	held := admitBytes(t, pool, []byte("abc"))

	in := newInput(strings.NewReader("x"))
	body, err := pool.Admit(context.Background(), in, 1, readAll)

	require.Nil(t, body)
	requireKind(t, err, spool.FailureCapacity)
	require.Equal(t, int64(0), in.reads.Load())
	require.Equal(t, int64(1), in.closes.Load())
	require.NoError(t, held.Close())
	requireExactCapacity(t, pool, cfg)
}

// TestAdmit_UnknownLengthReservesTheLimitUntilCommit guarantees that an input
// of unknown length holds BodyBytesMax of the shared budget while it is read,
// and only its stored size after commit. A known length reserves exactly that
// size.
func TestAdmit_UnknownLengthReservesTheLimitUntilCommit(t *testing.T) {
	t.Parallel()
	pool := newTestPool(t, spool.Config{Dir: "", BodyBytesMax: 100, TotalBytesMax: 150, ConcurrentMax: 4})
	ctx := context.Background()

	small, err := pool.Admit(ctx, newInput(strings.NewReader("0123456789")), -1, func(ctx context.Context, r io.Reader) error {
		_, overErr := pool.Admit(ctx, newInput(bytes.NewReader(make([]byte, 51))), 51, readAll)
		requireKind(t, overErr, spool.FailureCapacity)
		nested := admitBytes(t, pool, make([]byte, 50))
		require.NoError(t, nested.Close())
		return readAll(ctx, r)
	})
	require.NoError(t, err)

	large := admitBytes(t, pool, make([]byte, 100))
	_, err = pool.Admit(ctx, newInput(bytes.NewReader(make([]byte, 41))), 41, readAll)
	requireKind(t, err, spool.FailureCapacity)
	rest := admitBytes(t, pool, make([]byte, 40))

	for _, body := range []*spool.Body{small, large, rest} {
		require.NoError(t, body.Close())
	}
	full := []*spool.Body{admitBytes(t, pool, make([]byte, 100)), admitBytes(t, pool, make([]byte, 50))}
	_, err = pool.Admit(ctx, newInput(strings.NewReader("x")), 1, readAll)
	requireKind(t, err, spool.FailureCapacity)
	for _, body := range full {
		require.NoError(t, body.Close())
	}
}

// TestAdmit_ConcurrentAdmissionsStayWithinLimits guarantees that concurrent
// admissions with mixed outcomes never exceed ConcurrentMax and return every
// slot and byte when they finish.
func TestAdmit_ConcurrentAdmissionsStayWithinLimits(t *testing.T) {
	t.Parallel()
	cfg := spool.Config{Dir: "", BodyBytesMax: 64, TotalBytesMax: 256, ConcurrentMax: 4}
	pool := newTestPool(t, cfg)
	const workers, iterations = 16, 25

	var inUse, inUseMax atomic.Int64
	problems := make(chan error, workers*iterations)
	var wg sync.WaitGroup
	for worker := range workers {
		wg.Go(func() {
			for iteration := range iterations {
				if err := admitOnce(pool, (worker+iteration)%4, &inUse, &inUseMax); err != nil {
					problems <- err
				}
			}
		})
	}
	wg.Wait()
	close(problems)

	for err := range problems {
		require.NoError(t, err)
	}
	require.LessOrEqual(t, inUseMax.Load(), cfg.ConcurrentMax)
	requireExactCapacity(t, pool, cfg)
}

// admitOnce runs one admission whose outcome depends on scenario and returns
// an error if the outcome is not one of the allowed results.
func admitOnce(pool *spool.Pool, scenario int, inUse, inUseMax *atomic.Int64) error {
	data := []byte(strings.Repeat("a", 64))
	contentLength := int64(64)
	wantKind := spool.FailureKind(0)
	errInvalid := fmt.Errorf("scenario %d rejects", scenario)
	switch scenario {
	case 1:
		wantKind = spool.FailureInvalid
	case 2:
		data = append(data, 'a')
		contentLength = -1
		wantKind = spool.FailureTooLarge
	case 3:
		data = data[:10]
		contentLength = -1
	}

	// The count drops while the admission still holds its slot: inside the
	// validator when it rejects, and before Close when a body was admitted.
	body, err := pool.Admit(context.Background(), newInput(bytes.NewReader(data)), contentLength, func(ctx context.Context, r io.Reader) error {
		current := inUse.Add(1)
		for {
			seen := inUseMax.Load()
			if current <= seen || inUseMax.CompareAndSwap(seen, current) {
				break
			}
		}
		err := readAll(ctx, r)
		if err == nil && wantKind == spool.FailureInvalid {
			err = errInvalid
		}
		if err != nil {
			inUse.Add(-1)
		}
		return err
	})
	if err != nil {
		failure, ok := errors.AsType[*spool.Error](err)
		if !ok || (failure.Kind != wantKind && failure.Kind != spool.FailureCapacity) {
			return fmt.Errorf("scenario %d: unexpected error %w", scenario, err)
		}
		return nil
	}
	if wantKind != 0 {
		return fmt.Errorf("scenario %d: admitted a body that should fail with %s", scenario, wantKind)
	}
	got, readErr := io.ReadAll(body.Reader())
	inUse.Add(-1)
	if err := errors.Join(readErr, body.Close()); err != nil {
		return fmt.Errorf("scenario %d: replay failed: %w", scenario, err)
	}
	if !bytes.Equal(got, data) {
		return fmt.Errorf("scenario %d: replayed %d bytes, want %d", scenario, len(got), len(data))
	}
	return nil
}

// TestAdmit_LargeBodyKeepsHeapAllocationsBounded guarantees that a body made
// of one very large JSON string is spooled with bounded heap allocation, so
// body size does not drive application memory. It does not run in parallel
// because it measures process-wide allocations.
func TestAdmit_LargeBodyKeepsHeapAllocationsBounded(t *testing.T) {
	const size = 64 << 20
	pool := newTestPool(t, spool.Config{Dir: "", BodyBytesMax: size, TotalBytesMax: size, ConcurrentMax: 1})
	in := newInput(&jsonStringBody{size: size, offset: 0})

	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	body, err := pool.Admit(context.Background(), in, size, readAll)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, body.Close()) })

	require.Less(t, after.TotalAlloc-before.TotalAlloc, uint64(1<<20))
	stored := sha256.New()
	_, err = io.Copy(stored, body.Reader())
	require.NoError(t, err)
	want := sha256.New()
	_, err = io.Copy(want, &jsonStringBody{size: size, offset: 0})
	require.NoError(t, err)
	require.Equal(t, want.Sum(nil), stored.Sum(nil))
}

// jsonStringBody generates a JSON document that is one string of size bytes,
// quotes included, without holding it in memory.
type jsonStringBody struct {
	size   int64
	offset int64
}

func (b *jsonStringBody) Read(p []byte) (int, error) {
	if b.offset >= b.size {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), b.size-b.offset))
	for i := range n {
		position := b.offset + int64(i)
		p[i] = 'a'
		if position == 0 || position == b.size-1 {
			p[i] = '"'
		}
	}
	b.offset += int64(n)
	return n, nil
}
