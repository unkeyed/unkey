package sinks

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeSink is a deterministic Sink that returns a scripted sequence of
// errors, one per Send call. Anything past the script length returns nil
// so the test can assert on the exact attempt count.
type fakeSink struct {
	errs    []error
	calls   atomic.Int32
	gotData [][]Record
}

func (f *fakeSink) Send(_ context.Context, batch []Record) error {
	idx := int(f.calls.Add(1)) - 1
	f.gotData = append(f.gotData, batch)
	if idx >= len(f.errs) {
		return nil
	}
	return f.errs[idx]
}

func (f *fakeSink) HealthCheck(_ context.Context) error { return nil }

func TestSendWithRetry_NonRetryableReturnsImmediately(t *testing.T) {
	t.Parallel()

	// 401 Unauthorized is fatal; the helper must not retry an auth
	// failure — the credential isn't going to fix itself in 250ms.
	sink := &fakeSink{
		errs: []error{
			&SendError{
				Err:    errors.New("axiom returned 401"),
				Status: http.StatusUnauthorized,
			},
		},
	}

	err := SendWithRetry(context.Background(), sink, []Record{{}})
	require.Error(t, err)
	require.EqualValues(t, 1, sink.calls.Load(),
		"non-retryable error must short-circuit on the first attempt")
}

func TestSendWithRetry_RetriesTransientThenSucceeds(t *testing.T) {
	t.Parallel()

	// First two attempts fail with retryable errors (5xx, then transport),
	// third attempt succeeds. The helper must observe success and stop.
	sink := &fakeSink{
		errs: []error{
			&SendError{Err: errors.New("503"), Status: http.StatusServiceUnavailable},
			&SendError{Err: errors.New("transport"), Status: 0},
		},
	}

	err := SendWithRetry(context.Background(), sink, []Record{{}})
	require.NoError(t, err)
	require.EqualValues(t, 3, sink.calls.Load(),
		"helper must keep retrying until either success or attempts exhausted")
}

func TestSendWithRetry_ExhaustsAllAttempts(t *testing.T) {
	t.Parallel()

	// All four attempts (initial + 3 retries) fail; the helper must
	// surface the last error rather than swallowing it.
	allFail := &SendError{Err: errors.New("502"), Status: http.StatusBadGateway}
	sink := &fakeSink{errs: []error{allFail, allFail, allFail, allFail}}

	err := SendWithRetry(context.Background(), sink, []Record{{}})
	require.Error(t, err)
	require.EqualValues(t, 1+len(sendBackoffs), sink.calls.Load(),
		"final attempt count must equal 1 + len(sendBackoffs)")
}

func TestSendWithRetry_PlainErrorIsTreatedAsRetryable(t *testing.T) {
	t.Parallel()

	// A sink that hasn't been migrated to SendError still returns plain
	// errors. The helper must treat those as retryable transport errors
	// rather than dropping them on the floor.
	sink := &fakeSink{
		errs: []error{errors.New("dial tcp: i/o timeout")},
	}

	err := SendWithRetry(context.Background(), sink, []Record{{}})
	require.NoError(t, err, "plain error treated as retryable; succeeds on retry")
	require.EqualValues(t, 2, sink.calls.Load())
}

func TestSendWithRetry_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Cancelling the context mid-retry must abort, not silently complete
	// the schedule. We use an already-cancelled context after the first
	// failure to keep the test deterministic.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sink := &fakeSink{
		errs: []error{
			&SendError{Err: errors.New("503"), Status: http.StatusServiceUnavailable},
		},
	}
	err := SendWithRetry(ctx, sink, []Record{{}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSendError_RetryableMatrix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		status int
		want   bool
	}{
		{0, true},                              // transport error
		{http.StatusOK, false},                 // not an error in practice
		{http.StatusBadRequest, false},         // 400
		{http.StatusUnauthorized, false},       // 401
		{http.StatusForbidden, false},          // 403
		{http.StatusNotFound, false},           // 404
		{http.StatusRequestTimeout, true},      // 408
		{http.StatusTooManyRequests, true},     // 429
		{http.StatusInternalServerError, true}, // 500
		{http.StatusBadGateway, true},          // 502
		{http.StatusServiceUnavailable, true},  // 503
		{http.StatusGatewayTimeout, true},      // 504
	}
	for _, tc := range cases {
		se := &SendError{Err: errors.New("x"), Status: tc.status}
		require.Equal(t, tc.want, se.Retryable(),
			"status %d retryable mismatch", tc.status)
	}
}

func TestJitter(t *testing.T) {
	t.Parallel()

	// Sample many times; full jitter must never exceed the base delay,
	// must never be negative, and must not be a constant.
	const base = 100 * time.Millisecond
	seen := make(map[time.Duration]struct{})
	for range 1000 {
		d := jitter(base)
		require.GreaterOrEqual(t, d, time.Duration(0))
		require.Less(t, d, base)
		seen[d] = struct{}{}
	}
	require.Greater(t, len(seen), 1, "jitter should not be a constant")
}
