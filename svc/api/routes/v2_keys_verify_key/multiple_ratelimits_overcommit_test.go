package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

// TestMultipleRatelimitsCounterLeakBug tests the critical bug where multiple rate limits
// cause incorrect counter decrements when one limit is triggered.
// nolint: godox
// Bug description:
// When a key has multiple rate limits (e.g., 12/minute and 200/month), and the first
// limit gets checked and incremented, then the second limit is triggered, the first
// limit's counter remains incremented even though the request was rejected.
//
// This causes the monthly limit to be depleted faster than it should be, leading to
// users being rate limited too early.
//
// Expected behavior:
// When a rate limit check fails and the request is rejected, NO rate limit counters
// should be incremented. All counters should only be incremented when the request
// is actually allowed through.
func TestMultipleRatelimitsCounterLeakBug(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")
	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("monthly counter should be incremented when minute limit is hit", func(t *testing.T) {
		// Create a key with a per-minute rate limit (12 requests per minute)
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "requests-per-month",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2592000000, // 30 days in milliseconds
					Limit:       200,
				},
				{
					Name:        "requests-per-minute",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    60000, // 1 minute in milliseconds
					Limit:       12,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		// Make 12 valid requests (hitting the per-minute limit exactly)
		for i := range 12 {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Request %d should be valid", i+1)
			require.True(t, res.Body.Data.Valid, "Request %d should be valid", i+1)
		}

		// Allow time for async replay buffer to sync with Redis
		time.Sleep(500 * time.Millisecond)

		// The 13th request should be rate limited by the per-minute limit
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "Request 13 should be rate limited")
		require.False(t, res.Body.Data.Valid, "Request 13 should be rate limited")

		// Verify the rate limit message indicates it was the per-minute limit
		minuteLimitExceeded := false
		monthLimitExceeded := false
		var monthLimitRemaining int64

		for _, rl := range res.Body.Data.Ratelimits {
			if rl.Name == "requests-per-minute" && rl.Exceeded {
				minuteLimitExceeded = true
			}
			if rl.Name == "requests-per-month" {
				if rl.Exceeded {
					monthLimitExceeded = true
				}
				monthLimitRemaining = rl.Remaining
			}
		}

		require.True(t, minuteLimitExceeded, "Per-minute limit should be exceeded")
		require.False(t, monthLimitExceeded, "Per-month limit should NOT be exceeded")

		// CRITICAL TEST: The monthly limit should still have 188 remaining (200 - 12)
		// If the bug exists, the counter will have been decremented for the 13th request
		// even though it was rejected, resulting in 187 remaining instead of 188.
		require.Equal(t, int64(188), monthLimitRemaining,
			"Monthly limit should have exactly 188 remaining (200 - 12 valid requests). "+
				"If this is 187, the bug exists: the monthly counter was decremented "+
				"even though the request was rejected by the per-minute limit.")
	})

	t.Run("sequential ratelimit checks should not leak counters", func(t *testing.T) {
		// After many requests that hit the per-minute limit, the monthly limit
		// should still be accurate and allow the full 200 requests.

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "test-minute",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    60000,
					Limit:       5, // Very restrictive per-minute limit
				},
				{
					Name:        "test-monthly",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2592000000, // 30 days
					Limit:       50,         // Using smaller number for faster test
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		validRequestCount := 0
		rateLimitedRequestCount := 0

		// Make many requests, some will be rate limited by the minute limit
		// but we should eventually be able to make all 50 valid requests
		for cycle := 0; cycle < 10; cycle++ {
			// Try to make requests until we hit the per-minute limit
			// Making 6 requests to ensure we hit the 5/minute limit
			for i := 0; i < 6; i++ {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, 200, res.Status)
				require.NotNil(t, res.Body)

				if res.Body.Data.Valid {
					validRequestCount++
				} else {
					rateLimitedRequestCount++
					break // Hit the minute limit, wait for reset
				}

				if validRequestCount >= 50 {
					break
				}
			}

			if validRequestCount >= 50 {
				break
			}

			// Advance time by 2 minutes to completely flush both current and previous windows
			h.Clock.Tick(2 * time.Minute)
			time.Sleep(100 * time.Millisecond) // Allow for Redis propagation
		}

		// We should have been able to make exactly 50 valid requests
		require.Equal(t, 50, validRequestCount,
			"Should have been able to make exactly 50 valid requests (the monthly limit). "+
				"If this is less than 50, the bug caused the monthly counter to leak during "+
				"rate limited requests.")

		require.Greater(t, rateLimitedRequestCount, 0,
			"We should have hit the per-minute rate limit at least once during this test")

		// The next request should be rate limited by the monthly limit
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid)

		// Verify it's the monthly limit that's exceeded
		ratelimits := res.Body.Data.Ratelimits
		monthLimitExceeded := false

		for _, rl := range ratelimits {
			if rl.Name == "test-monthly" && rl.Exceeded {
				monthLimitExceeded = true
				break
			}
		}

		require.True(t, monthLimitExceeded,
			"The monthly limit should be exceeded after exactly 50 valid requests")
	})
}
