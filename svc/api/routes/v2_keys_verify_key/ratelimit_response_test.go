package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestRatelimitResponse(t *testing.T) {
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

	t.Run("rate limit response fields validation", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "test-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Minute.Milliseconds(),
					Limit:       5,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid, "Key should be valid but got %t", res.Body.Data.Valid)

		// Validate rate limit response fields
		require.NotNil(t, res.Body.Data.Ratelimits, "Rate limits should be present")
		ratelimits := res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 1, "Should have one rate limit")

		rl := ratelimits[0]
		require.Equal(t, "test-limit", rl.Name, "Rate limit name should match")
		require.Equal(t, int64(5), rl.Limit, "Rate limit limit should match")
		require.Equal(t, time.Minute.Milliseconds(), rl.Duration, "Rate limit duration should match")
		require.True(t, rl.AutoApply, "Rate limit should be auto-applied")
		require.False(t, rl.Exceeded, "Rate limit should not be exceeded")
		require.Equal(t, int64(4), rl.Remaining, "Should have 4 remaining requests")
		require.Greater(t, rl.Reset, time.Now().UnixMilli(), "Reset time should be in the future")
	})

	t.Run("rate limit exceeded fields", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "strict-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Minute.Milliseconds(),
					Limit:       1,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		// First request should pass
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, openapi.VALID, res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid)

		// Second request should be rate limited
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid)

		// Validate rate limit response fields for exceeded limit
		require.NotNil(t, res.Body.Data.Ratelimits, "Rate limits should be present")
		ratelimits := res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 1, "Should have one rate limit")

		rl := ratelimits[0]
		require.True(t, rl.Exceeded, "Rate limit should be exceeded")
		require.Equal(t, int64(0), rl.Remaining, "Should have 0 remaining requests")
	})

	t.Run("custom rate limit with cost", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
		})

		req := handler.Request{
			Key: key.Key,
			Ratelimits: &[]openapi.KeysVerifyKeyRatelimit{{
				Name:     "custom",
				Cost:     ptr.P(3),
				Duration: ptr.P(int(time.Minute.Milliseconds())),
				Limit:    ptr.P(10),
			}},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, openapi.VALID, res.Body.Data.Code, "Key should be valid but got %s", res.Body.Data.Code)

		// Validate custom rate limit response
		require.NotNil(t, res.Body.Data.Ratelimits, "Rate limits should be present")
		ratelimits := res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 1, "Should have one rate limit")

		rl := ratelimits[0]
		require.Equal(t, "custom", rl.Name, "Rate limit name should match")
		require.Equal(t, int64(10), rl.Limit, "Rate limit limit should match")
		require.Equal(t, int64(7), rl.Remaining, "Should have 7 remaining (10-3)")
		require.False(t, rl.AutoApply, "Custom rate limit should not be auto-applied")
	})

	t.Run("multiple rate limits with accurate remaining counters", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "fast-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Minute.Milliseconds(),
					Limit:       3,
				},
				{
					Name:        "slow-limit",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Hour.Milliseconds(),
					Limit:       10,
				},
			},
		})

		req := handler.Request{
			Key: key.Key,
		}

		// Helper function to find rate limit by name
		findRatelimit := func(ratelimits []openapi.VerifyKeyRatelimitData, name string) *openapi.VerifyKeyRatelimitData {
			for _, rl := range ratelimits {
				if rl.Name == name {
					return &rl
				}
			}
			return nil
		}

		// Request 1: Both limits should decrement
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.Equal(t, openapi.VALID, res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid)

		ratelimits := res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 2, "Should have two rate limits")

		fastLimit := findRatelimit(ratelimits, "fast-limit")
		slowLimit := findRatelimit(ratelimits, "slow-limit")
		require.NotNil(t, fastLimit, "fast-limit should be present")
		require.NotNil(t, slowLimit, "slow-limit should be present")

		require.Equal(t, int64(2), fastLimit.Remaining, "fast-limit: expected remaining=2 after 1st request")
		require.Equal(t, int64(9), slowLimit.Remaining, "slow-limit: expected remaining=9 after 1st request")
		require.False(t, fastLimit.Exceeded)
		require.False(t, slowLimit.Exceeded)

		// Request 2: Both limits should decrement again
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.Equal(t, openapi.VALID, res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid)

		ratelimits = res.Body.Data.Ratelimits
		fastLimit = findRatelimit(ratelimits, "fast-limit")
		slowLimit = findRatelimit(ratelimits, "slow-limit")

		require.Equal(t, int64(1), fastLimit.Remaining, "fast-limit: expected remaining=1 after 2nd request")
		require.Equal(t, int64(8), slowLimit.Remaining, "slow-limit: expected remaining=8 after 2nd request")
		require.False(t, fastLimit.Exceeded)
		require.False(t, slowLimit.Exceeded)

		// Request 3: Both limits should decrement, fast-limit reaches 0
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.Equal(t, openapi.VALID, res.Body.Data.Code)
		require.True(t, res.Body.Data.Valid)

		ratelimits = res.Body.Data.Ratelimits
		fastLimit = findRatelimit(ratelimits, "fast-limit")
		slowLimit = findRatelimit(ratelimits, "slow-limit")

		require.Equal(t, int64(0), fastLimit.Remaining, "fast-limit: expected remaining=0 after 3rd request")
		require.Equal(t, int64(7), slowLimit.Remaining, "slow-limit: expected remaining=7 after 3rd request")
		require.False(t, fastLimit.Exceeded)
		require.False(t, slowLimit.Exceeded)

		// Request 4: fast-limit should be exceeded, slow-limit continues
		res = testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code)
		require.False(t, res.Body.Data.Valid, "Key should be rate limited")

		ratelimits = res.Body.Data.Ratelimits
		fastLimit = findRatelimit(ratelimits, "fast-limit")
		slowLimit = findRatelimit(ratelimits, "slow-limit")

		require.Equal(t, int64(0), fastLimit.Remaining, "fast-limit: expected remaining=0 when exceeded")
		require.True(t, fastLimit.Exceeded, "fast-limit should be exceeded")
		// slow-limit should NOT increment since the request was denied
		require.Equal(t, int64(7), slowLimit.Remaining, "slow-limit: should not decrement when request is denied")
		require.False(t, slowLimit.Exceeded, "slow-limit should not be exceeded")
	})

	t.Run("identity rate limits with same config but different names are isolated", func(t *testing.T) {
		// Create an identity with two rate limits that have identical duration and limit but different names
		// This tests that the rate limit name is included in the identifier to prevent shared counters
		identity := h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspace.ID,
			ExternalID:  "user_with_multiple_limits",
			Ratelimits: []seed.CreateRatelimitRequest{
				{
					Name:        "api_requests",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Minute.Milliseconds(),
					Limit:       5,
				},
				{
					Name:        "data_access",
					WorkspaceID: workspace.ID,
					AutoApply:   true,
					Duration:    time.Minute.Milliseconds(), // Same duration as api_requests
					Limit:       5,                          // Same limit as api_requests
				},
			},
		})

		// Create a key for this identity
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			IdentityID:  ptr.P(identity.ID),
		})

		req := handler.Request{
			Key: key.Key,
		}

		// Helper function to find rate limit by name
		findRatelimit := func(ratelimits []openapi.VerifyKeyRatelimitData, name string) *openapi.VerifyKeyRatelimitData {
			for _, rl := range ratelimits {
				if rl.Name == name {
					return &rl
				}
			}
			return nil
		}

		// Make 5 requests - should use up api_requests limit
		for i := 0; i < 5; i++ {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, 200, res.Status)
			require.Equal(t, openapi.VALID, res.Body.Data.Code, "Request %d should be valid", i+1)
			require.True(t, res.Body.Data.Valid)

			ratelimits := res.Body.Data.Ratelimits
			require.Len(t, ratelimits, 2, "Should have two identity rate limits")

			apiLimit := findRatelimit(ratelimits, "api_requests")
			dataLimit := findRatelimit(ratelimits, "data_access")
			require.NotNil(t, apiLimit, "api_requests rate limit should be present")
			require.NotNil(t, dataLimit, "data_access rate limit should be present")

			// Both limits should decrement independently
			require.Equal(t, int64(5-i-1), apiLimit.Remaining, "api_requests: expected remaining=%d after request %d", 5-i-1, i+1)
			require.Equal(t, int64(5-i-1), dataLimit.Remaining, "data_access: expected remaining=%d after request %d", 5-i-1, i+1)
			require.False(t, apiLimit.Exceeded, "api_requests should not be exceeded yet")
			require.False(t, dataLimit.Exceeded, "data_access should not be exceeded yet")
		}

		// 6th request should be rate limited
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.Equal(t, openapi.RATELIMITED, res.Body.Data.Code, "6th request should be rate limited")
		require.False(t, res.Body.Data.Valid)

		ratelimits := res.Body.Data.Ratelimits
		apiLimit := findRatelimit(ratelimits, "api_requests")
		dataLimit := findRatelimit(ratelimits, "data_access")

		// Both limits should be exceeded (since they have the same config and both were checked)
		require.True(t, apiLimit.Exceeded, "api_requests should be exceeded")
		require.True(t, dataLimit.Exceeded, "data_access should be exceeded")
		require.Equal(t, int64(0), apiLimit.Remaining, "api_requests: should have 0 remaining")
		require.Equal(t, int64(0), dataLimit.Remaining, "data_access: should have 0 remaining")
	})
}
