package v2_ratelimit_multi_limit_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
)

func TestLimitSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		ClickHouse:              h.ClickHouse,
		Ratelimit:               h.Ratelimit,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:               h.Auditlogs,
	}

	h.Register(route)

	// Test auto-creation of multiple namespaces with audit logs
	t.Run("auto-create multiple namespaces with audit logs", func(t *testing.T) {
		// Use 3 namespaces that don't exist
		namespaceName1 := uid.New("nonexistent")
		namespaceName2 := uid.New("nonexistent")
		namespaceName3 := uid.New("nonexistent")
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.create_namespace", "ratelimit.*.limit")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		req := handler.Request{
			{
				Namespace:  namespaceName1,
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			},
			{
				Namespace:  namespaceName2,
				Identifier: "user_123",
				Limit:      200,
				Duration:   60000,
			},
			{
				Namespace:  namespaceName3,
				Identifier: "user_123",
				Limit:      300,
				Duration:   60000,
			},
		}

		// First request should succeed and create all namespaces
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true when all limits pass")
		require.Len(t, res.Body.Data.Limits, 3, "Should return 3 results")

		// Verify all responses
		require.True(t, res.Body.Data.Limits[0].Passed, "Rate limit should not be exceeded on first request")
		require.Equal(t, namespaceName1, res.Body.Data.Limits[0].Namespace)
		require.Equal(t, int64(100), res.Body.Data.Limits[0].Limit)
		require.Equal(t, int64(99), res.Body.Data.Limits[0].Remaining)

		require.True(t, res.Body.Data.Limits[1].Passed)
		require.Equal(t, namespaceName2, res.Body.Data.Limits[1].Namespace)
		require.Equal(t, int64(200), res.Body.Data.Limits[1].Limit)
		require.Equal(t, int64(199), res.Body.Data.Limits[1].Remaining)

		require.True(t, res.Body.Data.Limits[2].Passed)
		require.Equal(t, namespaceName3, res.Body.Data.Limits[2].Namespace)
		require.Equal(t, int64(300), res.Body.Data.Limits[2].Limit)
		require.Equal(t, int64(299), res.Body.Data.Limits[2].Remaining)

		// Verify all 3 namespaces were created with audit logs
		for _, nsName := range []string{namespaceName1, namespaceName2, namespaceName3} {
			namespace, err := db.Query.FindRatelimitNamespaceByName(ctx, h.DB.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: h.Resources().UserWorkspace.ID,
				Name:        nsName,
			})
			require.NoError(t, err)
			require.Equal(t, nsName, namespace.Name)
			require.False(t, namespace.DeletedAtM.Valid, "Namespace should not be deleted")

			// Verify audit log was created for the namespace
			auditTargets, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), namespace.ID)
			require.NoError(t, err)
			require.Len(t, auditTargets, 1, "Should have exactly one audit log entry for the namespace")

			auditTarget := auditTargets[0]
			require.Equal(t, string(auditlog.RatelimitNamespaceResourceType), auditTarget.AuditLogTarget.Type)
			require.Equal(t, namespace.ID, auditTarget.AuditLogTarget.ID)
			require.Equal(t, nsName, auditTarget.AuditLogTarget.Name.String)
			require.Equal(t, string(auditlog.RatelimitNamespaceCreateEvent), auditTarget.AuditLog.Event)
			require.Contains(t, auditTarget.AuditLog.Display, nsName, "Audit log should mention the namespace name")
		}
	})

	// Test basic multi rate limiting
	t.Run("basic multi rate limiting", func(t *testing.T) {
		ns1ID, ns1Name := createNamespace(t, h)
		ns2ID, ns2Name := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
			fmt.Sprintf("ratelimit.%s.limit", ns1ID),
			fmt.Sprintf("ratelimit.%s.limit", ns2ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		req := handler.Request{
			{
				Namespace:  ns1Name,
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			},
			{
				Namespace:  ns2Name,
				Identifier: "user_123",
				Limit:      200,
				Duration:   60000,
			},
		}

		// First request should succeed
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, status: %d, body: %s", res.Status, res.RawBody)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true when all limits pass")
		require.Len(t, res.Body.Data.Limits, 2)

		require.True(t, res.Body.Data.Limits[0].Passed, "Rate limit should not be exceeded on first request")
		require.Equal(t, ns1Name, res.Body.Data.Limits[0].Namespace)
		require.Equal(t, int64(100), res.Body.Data.Limits[0].Limit)
		require.Equal(t, int64(99), res.Body.Data.Limits[0].Remaining)
		require.Greater(t, res.Body.Data.Limits[0].Reset, int64(0))
		require.Empty(t, res.Body.Data.Limits[0].OverrideId, "No override should be applied")

		require.True(t, res.Body.Data.Limits[1].Passed)
		require.Equal(t, ns2Name, res.Body.Data.Limits[1].Namespace)
		require.Equal(t, int64(200), res.Body.Data.Limits[1].Limit)
		require.Equal(t, int64(199), res.Body.Data.Limits[1].Remaining)
		require.Empty(t, res.Body.Data.Limits[1].OverrideId)
	})

	// Test multi events are flushed to clickhouse
	t.Run("multiple events are flushed to clickhouse", func(t *testing.T) {
		ns1ID, ns1Name := createNamespace(t, h)
		ns2ID, ns2Name := createNamespace(t, h)
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
			fmt.Sprintf("ratelimit.%s.limit", ns1ID),
			fmt.Sprintf("ratelimit.%s.limit", ns2ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		identifier := uid.New("test")
		req := handler.Request{
			{
				Namespace:  ns1Name,
				Identifier: identifier,
				Limit:      100,
				Duration:   60000,
			},
			{
				Namespace:  ns2Name,
				Identifier: identifier,
				Limit:      200,
				Duration:   60000,
			},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)

		// Check both namespace events are logged
		for i, nsID := range []string{ns1ID, ns2ID} {
			row := schema.Ratelimit{}
			require.Eventually(t, func() bool {
				data, err := clickhouse.Select[schema.Ratelimit](
					ctx,
					h.ClickHouse.Conn(),
					"SELECT * FROM default.ratelimits_raw_v2 WHERE workspace_id = {workspace_id:String} AND namespace_id = {namespace_id:String}",
					map[string]string{
						"workspace_id": h.Resources().UserWorkspace.ID,
						"namespace_id": nsID,
					},
				)
				require.NoError(t, err)
				if len(data) != 1 {
					return false
				}
				row = data[0]
				return true

			}, 15*time.Second, 100*time.Millisecond)

			require.Equal(t, identifier, row.Identifier)
			require.Equal(t, res.Body.Data.Limits[i].Passed, row.Passed)
			require.Equal(t, res.Body.Meta.RequestId, row.RequestID)
		}
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
			{
				Namespace:  namespaceName,
				Identifier: "user_456",
				Limit:      100,
				Duration:   60000,
				Cost:       &cost,
			},
		}

		// Request with custom cost should reduce remaining by that amount
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true")
		require.Len(t, res.Body.Data.Limits, 1)
		require.True(t, res.Body.Data.Limits[0].Passed)
		require.Equal(t, int64(100), res.Body.Data.Limits[0].Limit)
		require.Equal(t, int64(95), res.Body.Data.Limits[0].Remaining) // 100 - 5
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
			{
				Namespace:  namespaceName,
				Identifier: identifier,
				Limit:      100,   // Different from override
				Duration:   60000, // Different from override
			},
		}

		// The override should take precedence over the request values
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true")
		require.Len(t, res.Body.Data.Limits, 1)
		require.True(t, res.Body.Data.Limits[0].Passed)
		require.Equal(t, int64(limit), res.Body.Data.Limits[0].Limit) // Should use override limit
		require.Equal(t, int64(199), res.Body.Data.Limits[0].Remaining)
		require.NotNil(t, res.Body.Data.Limits[0].OverrideId)
		require.Equal(t, overrideID, res.Body.Data.Limits[0].OverrideId)
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
			{
				Namespace:  namespaceName,
				Identifier: identifier,
				Limit:      100,   // Different from override
				Duration:   60000, // Different from override
			},
		}

		// The override should take precedence over the request values
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true")
		require.Len(t, res.Body.Data.Limits, 1)
		require.NotNil(t, res.Body.Data.Limits[0].OverrideId)
		require.Equal(t, overrideID, res.Body.Data.Limits[0].OverrideId)
		require.True(t, res.Body.Data.Limits[0].Passed)
		require.Equal(t, int64(limit), res.Body.Data.Limits[0].Limit) // Should use override limit
		require.Equal(t, int64(199), res.Body.Data.Limits[0].Remaining)
		require.NotNil(t, res.Body.Data.Limits[0].OverrideId)
	})

	// Test rate limit exceeded - multiple limits with some failing
	t.Run("multiple limits - some fail but all results returned", func(t *testing.T) {
		ns1ID, ns1Name := createNamespace(t, h)
		ns2ID, ns2Name := createNamespace(t, h)
		ns3ID, ns3Name := createNamespace(t, h)

		rootKey := h.CreateRootKey(
			h.Resources().UserWorkspace.ID,
			fmt.Sprintf("ratelimit.%s.limit", ns1ID),
			fmt.Sprintf("ratelimit.%s.limit", ns2ID),
			fmt.Sprintf("ratelimit.%s.limit", ns3ID),
		)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		identifier := uid.New("test")
		// Create tight limit for namespace 2
		req := handler.Request{
			{Namespace: ns1Name, Identifier: identifier, Limit: 100, Duration: 60000},
			{Namespace: ns2Name, Identifier: identifier, Limit: 1, Duration: 60000}, // Will fail on second call
			{Namespace: ns3Name, Identifier: identifier, Limit: 300, Duration: 60000},
		}

		// First request - all should succeed
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)
		require.True(t, res1.Body.Data.Passed, "Overall passed should be true when all limits pass")
		require.Len(t, res1.Body.Data.Limits, 3)
		require.True(t, res1.Body.Data.Limits[0].Passed)
		require.True(t, res1.Body.Data.Limits[1].Passed)
		require.True(t, res1.Body.Data.Limits[2].Passed)

		// Second request - ns2 should fail, but ALL results should still be returned
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)
		require.False(t, res2.Body.Data.Passed, "Overall passed should be false when any limit fails")
		require.Len(t, res2.Body.Data.Limits, 3, "Should return all 3 results even when one fails")

		// Verify results match requests (no early exit)
		require.Equal(t, ns1Name, res2.Body.Data.Limits[0].Namespace)
		require.Equal(t, identifier, res2.Body.Data.Limits[0].Identifier)
		require.True(t, res2.Body.Data.Limits[0].Passed, "ns1 should still succeed")

		require.Equal(t, ns2Name, res2.Body.Data.Limits[1].Namespace)
		require.Equal(t, identifier, res2.Body.Data.Limits[1].Identifier)
		require.False(t, res2.Body.Data.Limits[1].Passed, "ns2 should fail (limit exceeded)")
		require.Equal(t, int64(1), res2.Body.Data.Limits[1].Limit)
		require.Equal(t, int64(0), res2.Body.Data.Limits[1].Remaining)

		require.Equal(t, ns3Name, res2.Body.Data.Limits[2].Namespace)
		require.Equal(t, identifier, res2.Body.Data.Limits[2].Identifier)
		require.True(t, res2.Body.Data.Limits[2].Passed, "ns3 should still succeed")
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
			{
				Namespace:  namespaceName,
				Identifier: identifier,
				Limit:      100,    // Higher than override
				Duration:   120000, // Higher than override
			},
		}

		// First request - should succeed and use override values
		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res1.Status)
		require.True(t, res1.Body.Data.Passed, "Overall passed should be true")
		require.Len(t, res1.Body.Data.Limits, 1)
		require.True(t, res1.Body.Data.Limits[0].Passed)
		require.Equal(t, int64(overrideLimit), res1.Body.Data.Limits[0].Limit) // Should use override limit
		require.Equal(t, int64(2), res1.Body.Data.Limits[0].Remaining)         // 3-1=2 remaining
		require.NotNil(t, res1.Body.Data.Limits[0].OverrideId)
		require.Equal(t, overrideID, res1.Body.Data.Limits[0].OverrideId)

		// Second request - should succeed
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res2.Status)
		require.True(t, res2.Body.Data.Passed, "Overall passed should be true")
		require.Len(t, res2.Body.Data.Limits, 1)
		require.True(t, res2.Body.Data.Limits[0].Passed)
		require.Equal(t, int64(1), res2.Body.Data.Limits[0].Remaining) // 2-1=1 remaining

		// Third request - should succeed but use up last remaining quota
		res3 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res3.Status)
		require.True(t, res3.Body.Data.Passed, "Overall passed should be true")
		require.Len(t, res3.Body.Data.Limits, 1)
		require.True(t, res3.Body.Data.Limits[0].Passed)
		require.Equal(t, int64(0), res3.Body.Data.Limits[0].Remaining) // No more remaining

		// Fourth request - should be rate limited
		res4 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res4.Status)
		require.False(t, res4.Body.Data.Passed, "Overall passed should be false when rate limited")
		require.Len(t, res4.Body.Data.Limits, 1)
		require.False(t, res4.Body.Data.Limits[0].Passed, "Request should be rate limited")
		require.Equal(t, int64(0), res4.Body.Data.Limits[0].Remaining)
		require.NotNil(t, res4.Body.Data.Limits[0].OverrideId)
		require.Equal(t, overrideID, res4.Body.Data.Limits[0].OverrideId)
	})

	// Test custom cost with multiple requests
	t.Run("custom cost with multiple requests", func(t *testing.T) {
		ns1ID, ns1Name := createNamespace(t, h)
		ns2ID, ns2Name := createNamespace(t, h)
		ns3ID, ns3Name := createNamespace(t, h)

		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
			fmt.Sprintf("ratelimit.%s.limit", ns1ID),
			fmt.Sprintf("ratelimit.%s.limit", ns2ID),
			fmt.Sprintf("ratelimit.%s.limit", ns3ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		cost2 := int64(3)
		cost3 := int64(10)

		req := handler.Request{
			{
				Namespace:  ns1Name,
				Identifier: "user_multi_cost",
				Limit:      100,
				Duration:   60000,
				// No cost specified - defaults to 1
			},
			{
				Namespace:  ns2Name,
				Identifier: "user_multi_cost",
				Limit:      50,
				Duration:   60000,
				Cost:       &cost2, // Cost of 3
			},
			{
				Namespace:  ns3Name,
				Identifier: "user_multi_cost",
				Limit:      200,
				Duration:   60000,
				Cost:       &cost3, // Cost of 10
			},
		}

		// First request with custom costs
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true when all limits pass")
		require.Len(t, res.Body.Data.Limits, 3)

		// Check ns1 - default cost of 1
		require.True(t, res.Body.Data.Limits[0].Passed)
		require.Equal(t, ns1Name, res.Body.Data.Limits[0].Namespace)
		require.Equal(t, int64(100), res.Body.Data.Limits[0].Limit)
		require.Equal(t, int64(99), res.Body.Data.Limits[0].Remaining) // 100 - 1

		// Check ns2 - cost of 3
		require.True(t, res.Body.Data.Limits[1].Passed)
		require.Equal(t, ns2Name, res.Body.Data.Limits[1].Namespace)
		require.Equal(t, int64(50), res.Body.Data.Limits[1].Limit)
		require.Equal(t, int64(47), res.Body.Data.Limits[1].Remaining) // 50 - 3

		// Check ns3 - cost of 10
		require.True(t, res.Body.Data.Limits[2].Passed)
		require.Equal(t, ns3Name, res.Body.Data.Limits[2].Namespace)
		require.Equal(t, int64(200), res.Body.Data.Limits[2].Limit)
		require.Equal(t, int64(190), res.Body.Data.Limits[2].Remaining) // 200 - 10
	})

	// Test override with multiple requests
	t.Run("override with multiple requests", func(t *testing.T) {
		ns1ID, ns1Name := createNamespace(t, h)
		ns2ID, ns2Name := createNamespace(t, h)

		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID,
			fmt.Sprintf("ratelimit.%s.limit", ns1ID),
			fmt.Sprintf("ratelimit.%s.limit", ns2ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		identifier := "user_multi_override"

		// Create overrides for both namespaces
		override1ID := uid.New(uid.RatelimitOverridePrefix)
		err := db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          override1ID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: ns1ID,
			Identifier:  identifier,
			Limit:       5,
			Duration:    60000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		override2ID := uid.New(uid.RatelimitOverridePrefix)
		err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
			ID:          override2ID,
			WorkspaceID: h.Resources().UserWorkspace.ID,
			NamespaceID: ns2ID,
			Identifier:  identifier,
			Limit:       3,
			Duration:    60000,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		req := handler.Request{
			{
				Namespace:  ns1Name,
				Identifier: identifier,
				Limit:      100, // Base limit (will be overridden to 5)
				Duration:   60000,
			},
			{
				Namespace:  ns2Name,
				Identifier: identifier,
				Limit:      200, // Base limit (will be overridden to 3)
				Duration:   60000,
			},
		}

		// First request - should use overrides
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.True(t, res.Body.Data.Passed, "Overall passed should be true when all limits pass")
		require.Len(t, res.Body.Data.Limits, 2)

		// Check ns1 - override limit of 5
		require.True(t, res.Body.Data.Limits[0].Passed)
		require.Equal(t, ns1Name, res.Body.Data.Limits[0].Namespace)
		require.Equal(t, int64(5), res.Body.Data.Limits[0].Limit) // Override limit
		require.Equal(t, int64(4), res.Body.Data.Limits[0].Remaining)
		require.Equal(t, override1ID, res.Body.Data.Limits[0].OverrideId)

		// Check ns2 - override limit of 3
		require.True(t, res.Body.Data.Limits[1].Passed)
		require.Equal(t, ns2Name, res.Body.Data.Limits[1].Namespace)
		require.Equal(t, int64(3), res.Body.Data.Limits[1].Limit) // Override limit
		require.Equal(t, int64(2), res.Body.Data.Limits[1].Remaining)
		require.Equal(t, override2ID, res.Body.Data.Limits[1].OverrideId)
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
