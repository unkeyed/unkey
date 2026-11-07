package ratelimit

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/buffer"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/uid"
)

// replayTestCounter wraps an in-memory counter and lets a test script the
// Increment result, so a transient Redis failure can be replayed.
type replayTestCounter struct {
	counter.Counter
	increment func(context.Context, string, int64) (int64, error)
}

func (c *replayTestCounter) Increment(ctx context.Context, key string, value int64, _ ...time.Duration) (int64, error) {
	return c.increment(ctx, key, value)
}

// newReplayTestService builds a service that only exercises the replay path: a
// scripted origin counter and no database or background loops.
func newReplayTestService(t *testing.T, ctr counter.Counter) *service {
	t.Helper()
	svc := &service{ //nolint:exhaustruct // only the replay path is exercised
		clock:                clock.NewTestClock(),
		counters:             sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		strictUntils:         sync.Map{}, //nolint:exhaustruct // sync.Map zero value is ready to use
		origin:               ctr,
		originCircuitBreaker: circuitbreaker.New[int64]("test_replay"),
		replayBuffer:         buffer.New[RatelimitRequest](buffer.Config{Name: "test_replay", Capacity: 16, Drop: false}),
	}
	t.Cleanup(svc.replayBuffer.Close)
	return svc
}

func replayTestRequest() RatelimitRequest {
	return RatelimitRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		Namespace:   "ns",
		Identifier:  "id",
		Limit:       10,
		Duration:    time.Minute,
		Cost:        1,
		Time:        time.Now(),
	}
}

// dialError returns a real dial-phase failure, the only error a replay may
// retry: the connection was never established, so Redis never applied the
// increment and repeating it cannot double count.
func dialError(t *testing.T) error {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	require.NoError(t, listener.Close())

	conn, err := net.Dial("tcp", addr)
	if conn != nil {
		_ = conn.Close()
	}
	require.Error(t, err)
	return err
}

func TestReplayWithRetryRecoversTransientFailure(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	svc := newReplayTestService(t, &replayTestCounter{
		Counter: counter.NewMemory(),
		increment: func(context.Context, string, int64) (int64, error) {
			if calls.Add(1) < 3 {
				return 0, dialError(t)
			}
			return 5, nil
		},
	})

	require.NoError(t, svc.replayWithRetry(context.Background(), replayTestRequest()))
	require.EqualValues(t, 3, calls.Load(), "a transient failure must be retried, not discarded")
}

func TestReplayWithRetryGivesUpAfterBoundedAttempts(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	svc := newReplayTestService(t, &replayTestCounter{
		Counter: counter.NewMemory(),
		increment: func(context.Context, string, int64) (int64, error) {
			calls.Add(1)
			return 0, dialError(t)
		},
	})

	require.Error(t, svc.replayWithRetry(context.Background(), replayTestRequest()))
	require.EqualValues(t, replayMaxAttempts, calls.Load(), "retries must be bounded")
}

func TestReplayRequestsRetriesBeforeDiscarding(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64
	svc := newReplayTestService(t, &replayTestCounter{
		Counter: counter.NewMemory(),
		increment: func(context.Context, string, int64) (int64, error) {
			if calls.Add(1) < 2 {
				return 0, dialError(t)
			}
			return 1, nil
		},
	})

	svc.replayBuffer.Buffer(replayTestRequest())
	svc.replayBuffer.Close()
	svc.replayRequests()

	require.EqualValues(t, 2, calls.Load(), "the buffered increment must reach Redis after a retry")
}

func TestReplayWithRetryDoesNotRetryAmbiguousFailure(t *testing.T) {
	t.Parallel()

	// A read error can follow a successful write, so the increment may already
	// have been applied. Retrying it would double count the quota.
	ambiguous := &net.OpError{Op: "read", Net: "tcp", Err: errors.New("i/o timeout")}

	var calls atomic.Int64
	svc := newReplayTestService(t, &replayTestCounter{
		Counter: counter.NewMemory(),
		increment: func(context.Context, string, int64) (int64, error) {
			calls.Add(1)
			return 0, ambiguous
		},
	})

	require.Error(t, svc.replayWithRetry(context.Background(), replayTestRequest()))
	require.EqualValues(t, 1, calls.Load(), "an ambiguous failure must not be repeated")
}

func TestReplayWithRetryDoesNotRetryOpenCircuit(t *testing.T) {
	t.Parallel()

	// While the breaker is open every call short-circuits, so retrying cannot
	// succeed and would only hold the replay worker.
	var calls atomic.Int64
	svc := newReplayTestService(t, &replayTestCounter{
		Counter: counter.NewMemory(),
		increment: func(context.Context, string, int64) (int64, error) {
			calls.Add(1)
			return 0, circuitbreaker.ErrTripped
		},
	})

	require.Error(t, svc.replayWithRetry(context.Background(), replayTestRequest()))
	require.EqualValues(t, 1, calls.Load(), "an open breaker must not be retried")
}
