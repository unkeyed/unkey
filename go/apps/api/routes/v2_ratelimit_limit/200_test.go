package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestLimitSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:                            h.DB,
		Keys:                          h.Keys,
		Logger:                        h.Logger,
		ClickHouse:                    h.ClickHouse,
		Ratelimit:                     h.Ratelimit,
		RatelimitNamespaceByNameCache: h.Caches.RatelimitNamespaceByName,
	}

	h.Register(route)

	// Test basic rate limiting
	t.Run("basic rate limiting", func(t *testing.T) {

		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000, // 1 minute in ms
		}

		// First request should succeed
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Success, "Rate limit should not be exceeded on first request")
		require.Equal(t, int64(100), res.Body.Data.Limit)
		require.Equal(t, int64(99), res.Body.Data.Remaining)
		require.Greater(t, res.Body.Data.Reset, int64(0))
		require.Nil(t, res.Body.Data.OverrideId, "No override should be applied")
	})

	// Test basic rate limiting
	t.Run("the event is flushed to clickhouse", func(t *testing.T) {
		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: uid.New("test"),
			Limit:      100,
			Duration:   60000, // 1 minute in ms
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)

		row := schema.RatelimitRequestV1{}
		require.Eventually(t, func() bool {

			data, err := clickhouse.Select[schema.RatelimitRequestV1](
				ctx,
				h.ClickHouse.Conn(),
				"SELECT * FROM ratelimits.raw_ratelimits_v1 WHERE workspace_id = {workspace_id:String} AND namespace_id = {namespace_id:String}",
				map[string]string{
					"workspace_id": h.Resources().UserWorkspace.ID,
					"namespace_id": namespaceID,
				},
			)
			require.NoError(t, err)
			if len(data) != 1 {
				return false
			}
			row = data[0]
			return true

		}, 15*time.Second, 100*time.Millisecond)

		require.Equal(t, req.Identifier, row.Identifier)
		require.Equal(t, res.Body.Data.Success, row.Passed)
		require.Equal(t, res.Body.Meta.RequestId, row.RequestID)

	})

	// Test with custom cost
	t.Run("custom cost", func(t *testing.T) {
		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		cost := int64(5)
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: "user_456",
			Limit:      100,
			Duration:   60000, // 1 minute in ms
			Cost:       &cost,
		}

		// Request with custom cost should reduce remaining by that amount
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, int64(100), res.Body.Data.Limit)
		require.Equal(t, int64(95), res.Body.Data.Remaining) // 100 - 5
	})

	// Test with rate limit override
	t.Run("with override", func(t *testing.T) {
		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		// Create an override
		identifier := "user_789"
		overrideID := uid.New(uid.RatelimitOverridePrefix)
		limit := int32(200)
		duration := int32(120000) // 2 minutes

		err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: namespaceID,
			Identifier:  identifier,
			Limit:       limit,
			Duration:    duration,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: identifier,
			Limit:      100,   // Different from override
			Duration:   60000, // Different from override
		}

		// The override should take precedence over the request values
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, int64(limit), res.Body.Data.Limit) // Should use override limit
		require.Equal(t, int64(199), res.Body.Data.Remaining)
		require.NotNil(t, res.Body.Data.OverrideId)
		require.Equal(t, overrideID, *res.Body.Data.OverrideId)
	})

	// Test with rate limit override
	t.Run("with wildcard override", func(t *testing.T) {

		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		// Create an override

		identifier := uid.New("prefix")

		override := strings.Replace(identifier, "prefix", "p*", 1)
		overrideID := uid.New(uid.RatelimitOverridePrefix)
		limit := int32(200)
		duration := int32(120000) // 2 minutes

		err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: namespaceID,
			Identifier:  override,
			Limit:       limit,
			Duration:    duration,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: identifier,
			Limit:      100,   // Different from override
			Duration:   60000, // Different from override
		}

		// The override should take precedence over the request values
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data.OverrideId)
		require.Equal(t, overrideID, *res.Body.Data.OverrideId)
		require.True(t, res.Body.Data.Success)
		require.Equal(t, int64(limit), res.Body.Data.Limit) // Should use override limit
		require.Equal(t, int64(199), res.Body.Data.Remaining)
		require.NotNil(t, res.Body.Data.OverrideId)
	})
	// Test rate limit exceeded
	t.Run("rate limit exceeded", func(t *testing.T) {
		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		// Create a small limit
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: uid.New("test"),
			Limit:      1, // Only 1 request allowed
			Duration:   60000,
		}

		// First request should succeed
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)
		require.True(t, res1.Body.Data.Success)
		require.Equal(t, int64(0), res1.Body.Data.Remaining)

		// Second request should fail (rate limited)
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status) // Still returns 200 OK
		require.NotNil(t, res2.Body)
		require.False(t, res2.Body.Data.Success, "Rate limit should be exceeded")
		require.Equal(t, int64(1), res2.Body.Data.Limit)
		require.Equal(t, int64(0), res2.Body.Data.Remaining)
	})
	t.Run("rate limiting with active override", func(t *testing.T) {
		namespaceID, namespaceName := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespaceID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		// Create an override with tight limits
		identifier := "override_user"
		overrideLimit := int32(3)        // Only allow 3 requests
		overrideDuration := int32(60000) // 1 minute window

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: namespaceID,
			Identifier:  identifier,
			Limit:       overrideLimit,
			Duration:    overrideDuration,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Make a rate limit request with more permissive values that should be overridden
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: identifier,
			Limit:      100,    // Higher than override
			Duration:   120000, // Higher than override
		}

		// First request - should succeed and use override values
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)
		require.True(t, res1.Body.Data.Success)
		require.Equal(t, int64(overrideLimit), res1.Body.Data.Limit) // Should use override limit
		require.Equal(t, int64(2), res1.Body.Data.Remaining)         // 3-1=2 remaining
		require.NotNil(t, res1.Body.Data.OverrideId)
		require.Equal(t, overrideID, *res1.Body.Data.OverrideId)

		// Second request - should succeed
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)
		require.True(t, res2.Body.Data.Success)
		require.Equal(t, int64(1), res2.Body.Data.Remaining) // 2-1=1 remaining

		// Third request - should succeed but use up last remaining quota
		res3 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res3.Status)
		require.True(t, res3.Body.Data.Success)
		require.Equal(t, int64(0), res3.Body.Data.Remaining) // No more remaining

		// Fourth request - should be rate limited
		res4 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res4.Status)
		require.False(t, res4.Body.Data.Success, "Request should be rate limited")
		require.Equal(t, int64(0), res4.Body.Data.Remaining)
		require.NotNil(t, res4.Body.Data.OverrideId)
		require.Equal(t, overrideID, *res4.Body.Data.OverrideId)
	})
}

func createNamespace(t *testing.T, h *testutil.Harness) (id, name string) {
	// Create a namespace
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	namespaceName := uid.New("test")
	err := db.Query.InsertRatelimitNamespace(context.Background(), h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	return namespaceID, namespaceName
}
