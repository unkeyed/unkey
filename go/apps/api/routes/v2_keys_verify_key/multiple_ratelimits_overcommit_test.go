package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

// TestMultipleRatelimitsCounterLeakBug tests the critical bug where multiple rate limits
// cause incorrect counter decrements when one limit is triggered.
//
// Bug description:
// When a key has multiple rate limits (e.g., 12/minute and 200/month), and the first
// limit gets checked and decremented, then the second limit is triggered, the first
// limit's counter remains decremented even though the request was rejected.
//
// This causes the monthly limit to be depleted faster than it should be, leading to
// users being rate limited at 164 requests instead of the configured 200.
//
// Expected behavior:
// When a rate limit check fails and the request is rejected, NO rate limit counters
// should be decremented. All counters should only be decremented when the request
// is actually allowed through.
func TestMultipleRatelimitsCounterLeakBug(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
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

	t.Run("monthly counter should not leak when minute limit is hit", func(t *testing.T) {
		clk := clock.NewTestClock()

		// Create an identity with a monthly rate limit (200 requests per month)
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  "user-with-monthly-limit",
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "requests-per-month",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2592000000, // 30 days in milliseconds
					Limit:       200,
				},
			},
		})

		// Create a key with a per-minute rate limit (12 requests per minute)
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			IdentityID:  ptr.P(identity),
			Ratelimits: []seed.CreateRatelimitRequest{
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

		// Set initial time
		headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))

		// Make 12 valid requests (hitting the per-minute limit exactly)
		for i := 0; i < 12; i++ {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Request %d should be valid", i+1)
			require.True(t, res.Body.Data.Valid, "Request %d should be valid", i+1)
		}

		// Allow brief time for Redis propagation
		time.Sleep(100 * time.Millisecond)

		// The 13th request should be rate limited by the per-minute limit
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "Request 13 should be rate limited")
		require.False(t, res.Body.Data.Valid, "Request 13 should be rate limited")

		// Verify the rate limit message indicates it was the per-minute limit
		ratelimits := *res.Body.Data.Ratelimits
		minuteLimitExceeded := false
		monthLimitExceeded := false
		var monthLimitRemaining int64

		for _, rl := range ratelimits {
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

		// Advance time by 61 seconds to reset the minute window
		clk.Tick(61 * time.Second)
		headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))
		time.Sleep(100 * time.Millisecond) // Allow for Redis propagation

		// Make another 12 requests, which should all be valid
		for i := 0; i < 12; i++ {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.NotNil(t, res.Body)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Request %d in second batch should be valid", i+1)
			require.True(t, res.Body.Data.Valid, "Request %d in second batch should be valid", i+1)
		}

		// Allow for Redis propagation
		time.Sleep(100 * time.Millisecond)

		// Now the monthly limit should have 176 remaining (200 - 24)
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		ratelimits = *res.Body.Data.Ratelimits

		for _, rl := range ratelimits {
			if rl.Name == "requests-per-month" {
				monthLimitRemaining = rl.Remaining
				break
			}
		}

		require.Equal(t, int64(175), monthLimitRemaining,
			"After 25 requests (24 valid + 1 that triggered per-minute limit), "+
				"monthly limit should have 175 remaining. If this is lower, the bug caused counter leakage.")
	})

	t.Run("sequential ratelimit checks should not leak counters", func(t *testing.T) {
		// This test simulates the exact scenario from the customer report:
		// After many requests that hit the per-minute limit, the monthly limit
		// should still be accurate and allow the full 200 requests.

		clk := clock.NewTestClock()
		headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))

		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  "sequential-test-user",
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "test-monthly",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    2592000000, // 30 days
					Limit:       50,         // Using smaller number for faster test
				},
			},
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			IdentityID:  ptr.P(identity),
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "test-minute",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    60000,
					Limit:       5, // Very restrictive per-minute limit
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
			for i := 0; i < 5; i++ {
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

			// Advance time by 61 seconds to reset the minute window
			clk.Tick(61 * time.Second)
			headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))
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
		ratelimits := *res.Body.Data.Ratelimits
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

	t.Run("both key and identity limits should not leak counters", func(t *testing.T) {
		// Test that counter leakage doesn't happen with the key/identity override precedence

		clk := clock.NewTestClock()
		headers.Set("X-Test-Time", fmt.Sprintf("%d", clk.Now().UnixMilli()))

		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  "precedence-leak-test",
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "shared-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    600000, // 10 minutes
					Limit:       100,
				},
				{
					Name:        "identity-only-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    60000, // 1 minute
					Limit:       3,
				},
			},
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
			IdentityID:  ptr.P(identity),
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "shared-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    60000,
					Limit:       5,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		// Make 3 valid requests
		for i := 0; i < 3; i++ {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, openapi.VALID, res.Body.Data.Code)
		}

		// Allow for Redis propagation
		time.Sleep(100 * time.Millisecond)

		// The 4th request should be rate limited by identity-only-limit
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)

		// Check that the shared-limit counter wasn't decremented for the 4th request
		ratelimits := *res.Body.Data.Ratelimits
		var sharedLimitRemaining int64

		for _, rl := range ratelimits {
			if rl.Name == "shared-limit" {
				sharedLimitRemaining = rl.Remaining
				break
			}
		}

		// The shared-limit should have 2 remaining (5 - 3), not 1
		require.Equal(t, int64(2), sharedLimitRemaining,
			"Shared limit should have 2 remaining (5 - 3 valid requests). "+
				"If this is 1, the bug caused the counter to be decremented "+
				"even though the request was rejected by identity-only-limit.")
	})
}
