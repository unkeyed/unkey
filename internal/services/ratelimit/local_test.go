package ratelimit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestNewLocal_EnforcesSlidingWindowWithoutExternalServices(t *testing.T) {
	t.Parallel()

	start := time.Unix(1_800_000_000, 0)
	clk := clock.NewTestClock(start)
	svc := ratelimit.NewLocal(clk)
	t.Cleanup(func() { require.NoError(t, svc.Close()) })
	var limiter ratelimit.Service = svc
	req := ratelimit.RatelimitRequest{
		WorkspaceID: uid.New(uid.WorkspacePrefix),
		Namespace:   "orders",
		Identifier:  "user",
		Limit:       4,
		Duration:    time.Minute,
		Cost:        3,
		Time:        time.Time{},
	}

	for _, step := range []struct {
		advance            time.Duration
		cost               int64
		success            bool
		current, remaining int64
		resetAfterStart    time.Duration
	}{
		{0, 3, true, 3, 1, time.Minute},
		{0, 2, false, 5, 0, time.Minute},
		{0, 1, true, 4, 0, time.Minute},
		{time.Minute, 1, false, 5, 0, 2 * time.Minute},
		{30 * time.Second, 1, true, 3, 1, 2 * time.Minute},
		{90 * time.Second, 4, true, 4, 0, 4 * time.Minute},
	} {
		clk.Tick(step.advance)
		req.Cost = step.cost
		response, err := limiter.Ratelimit(t.Context(), req)
		require.NoError(t, err)
		require.Equal(t, step.success, response.Success)
		require.Equal(t, step.current, response.Current)
		require.Equal(t, step.remaining, response.Remaining)
		require.Equal(t, int64(4), response.Limit)
		require.Equal(t, start.Add(step.resetAfterStart), response.Reset)
	}
}

func TestNewLocal_BatchRollbackAndInstanceIsolation(t *testing.T) {
	t.Parallel()

	clk := clock.NewTestClock(time.Unix(1_800_000_000, 0))
	svc := ratelimit.NewLocal(clk)
	t.Cleanup(func() { require.NoError(t, svc.Close()) })
	workspaceID := uid.New(uid.WorkspacePrefix)
	requests := []ratelimit.RatelimitRequest{
		{WorkspaceID: workspaceID, Namespace: "orders", Identifier: "user", Limit: 7, Duration: time.Minute, Cost: 3, Time: clk.Now()},
		{WorkspaceID: workspaceID, Namespace: "emails", Identifier: "user", Limit: 5, Duration: time.Minute, Cost: 4, Time: clk.Now()},
	}
	responses, err := svc.RatelimitMany(t.Context(), requests)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.True(t, responses[0].Success)
	require.True(t, responses[1].Success)
	require.Equal(t, int64(4), responses[0].Remaining)
	require.Equal(t, int64(1), responses[1].Remaining)

	requests[0].Cost = 2
	requests[1].Cost = 2
	responses, err = svc.RatelimitMany(t.Context(), requests)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.True(t, responses[0].Success)
	require.False(t, responses[1].Success)
	require.Equal(t, int64(4), responses[0].Remaining)

	requests[0].Cost = 0
	requests[1].Cost = 0
	responses, err = svc.RatelimitMany(t.Context(), requests)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.True(t, responses[0].Success)
	require.True(t, responses[1].Success)
	require.Equal(t, int64(4), responses[0].Remaining)
	require.Equal(t, int64(1), responses[1].Remaining)

	other := ratelimit.NewLocal(nil)
	t.Cleanup(func() { require.NoError(t, other.Close()) })
	responses, err = other.RatelimitMany(t.Context(), requests)
	require.NoError(t, err)
	require.Len(t, responses, 2)
	require.True(t, responses[0].Success)
	require.True(t, responses[1].Success)
	require.Equal(t, int64(7), responses[0].Remaining)
	require.Equal(t, int64(5), responses[1].Remaining)
}
