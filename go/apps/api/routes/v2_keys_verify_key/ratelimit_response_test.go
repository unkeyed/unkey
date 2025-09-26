package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_verify_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestRatelimitResponse(t *testing.T) {
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

	t.Run("rate limit response fields validation", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
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
		ratelimits := *res.Body.Data.Ratelimits
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
			KeyAuthID:   api.KeyAuthID.String,
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
		ratelimits := *res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 1, "Should have one rate limit")

		rl := ratelimits[0]
		require.True(t, rl.Exceeded, "Rate limit should be exceeded")
		require.Equal(t, int64(0), rl.Remaining, "Should have 0 remaining requests")
	})

	t.Run("custom rate limit with cost", func(t *testing.T) {
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeyAuthID:   api.KeyAuthID.String,
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
		ratelimits := *res.Body.Data.Ratelimits
		require.Len(t, ratelimits, 1, "Should have one rate limit")

		rl := ratelimits[0]
		require.Equal(t, "custom", rl.Name, "Rate limit name should match")
		require.Equal(t, int64(10), rl.Limit, "Rate limit limit should match")
		require.Equal(t, int64(7), rl.Remaining, "Should have 7 remaining (10-3)")
		require.False(t, rl.AutoApply, "Custom rate limit should not be auto-applied")
	})
}
