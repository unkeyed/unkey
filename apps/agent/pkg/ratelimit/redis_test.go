package ratelimit_test

import (
	"testing"
	"time"
	// "time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/testutil"
)

func TestRedisRateLimiter(t *testing.T) {

	redisUrl := testutil.CreateRedis(t)

	config := ratelimit.RedisConfig{
		RedisUrl: redisUrl,
		Logger:   logging.New(nil),
	}

	rl, err := ratelimit.NewRedis(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Test that the rate limiter works as expected
	identifier := "test-identifier"
	maxTokens := int32(10)
	interval := int32(5000)
	refillRate := int32(4)
	// now := time.Now().UnixMilli()

	req := ratelimit.RatelimitRequest{
		Identifier:     identifier,
		Max:            maxTokens,
		RefillRate:     refillRate,
		RefillInterval: interval,
	}
	// First 10 requests should succeed

	for i := maxTokens; i > 0; i-- {
		res := rl.Take(req)
		require.True(t, res.Pass)
		require.Equal(t, maxTokens, res.Limit)
		require.Equal(t, int32(i-1), res.Remaining)
	}

	// Next request should fail
	res := rl.Take(req)
	require.False(t, res.Pass)
	require.Equal(t, maxTokens, res.Limit)
	require.Equal(t, int32(0), res.Remaining)

	// wait until the reset
	time.Sleep(time.Until(time.UnixMilli(res.Reset).Add(time.Millisecond * 500)))

	for i := refillRate; i > 0; i-- {
		res := rl.Take(req)
		t.Log(i, res)
		require.True(t, res.Pass)
		require.Equal(t, maxTokens, res.Limit)
		require.Equal(t, int32(i-1), res.Remaining)
	}

	// Next request should fail
	res2 := rl.Take(req)
	require.False(t, res2.Pass)
	require.Equal(t, maxTokens, res2.Limit)
	require.Equal(t, int32(0), res2.Remaining)

}
