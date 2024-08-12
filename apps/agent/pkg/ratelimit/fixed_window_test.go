package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
)

func TestReturnsCorrectRemaining(t *testing.T) {

	rl := ratelimit.NewFixedWindow(logging.NewNoopLogger())

	ctx := context.Background()

	res := rl.Take(ctx, ratelimit.RatelimitRequest{
		Identifier: "test",
		Limit:      10,
		Duration:   time.Minute,
		Cost:       1,
	})
	require.Equal(t, int64(9), res.Remaining)
	require.True(t, res.Pass)
	require.Equal(t, int64(1), res.Current)

}

func TestWithLeaseReturnsCorrectRemaining(t *testing.T) {

	rl := ratelimit.NewFixedWindow(logging.NewNoopLogger())

	ctx := context.Background()

	res := rl.Take(ctx, ratelimit.RatelimitRequest{
		Identifier: "test",
		Limit:      10,
		Duration:   time.Minute,
		Cost:       1,
		Lease: &ratelimit.Lease{
			Cost:      4,
			ExpiresAt: time.Now().Add(time.Second * 30),
		},
	})
	require.Equal(t, int64(5), res.Remaining)
	require.True(t, res.Pass)
	require.Equal(t, int64(5), res.Current)

}

func TestWithLeaseRejects(t *testing.T) {

	rl := ratelimit.NewFixedWindow(logging.NewNoopLogger())

	ctx := context.Background()

	resWithoutlease := rl.Take(ctx, ratelimit.RatelimitRequest{
		Identifier: "test",
		Limit:      10,
		Duration:   time.Minute,
		Cost:       1,
	})
	require.Equal(t, int64(9), resWithoutlease.Remaining)
	require.True(t, resWithoutlease.Pass)
	require.Equal(t, int64(1), resWithoutlease.Current)

	resWithlease := rl.Take(ctx, ratelimit.RatelimitRequest{
		Identifier: "test",
		Limit:      10,
		Duration:   time.Minute,
		Cost:       1,
		Lease: &ratelimit.Lease{
			Cost:      10,
			ExpiresAt: time.Now().Add(time.Second * 30),
		},
	})
	require.Equal(t, int64(9), resWithlease.Remaining)
	require.False(t, resWithlease.Pass)
	require.Equal(t, int64(1), resWithlease.Current)

}
