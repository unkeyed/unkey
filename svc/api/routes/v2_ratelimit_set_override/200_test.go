package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_set_override"
)

func TestSetOverrideSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New("test_ns")
	namespaceName := uid.New("test")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Auditlogs:               h.Auditlogs,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.set_override", namespaceID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a new override by namespace name
	t.Run("create override using namespace name", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceName,
			Identifier: "user_123",
			Limit:      10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %+v", res.Body)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.OverrideId, "Override ID should not be empty")

		// Verify the override was created correctly
		override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			OverrideID:  res.Body.Data.OverrideId,
		})
		require.NoError(t, err)
		require.Equal(t, namespaceID, override.NamespaceID)
		require.Equal(t, "user_123", override.Identifier)
		require.Equal(t, int32(10), override.Limit)
		require.Equal(t, int32(1000), override.Duration)
	})

	// Create a new override by namespace ID
	t.Run("create override using namespace ID", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: "user_456",
			Limit:      20,
			Duration:   2000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.OverrideId, "Override ID should not be empty")

		// Verify the override was created correctly
		override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			OverrideID:  res.Body.Data.OverrideId,
		})
		require.NoError(t, err)
		require.Equal(t, namespaceID, override.NamespaceID)
		require.Equal(t, "user_456", override.Identifier)
		require.Equal(t, int32(20), override.Limit)
		require.Equal(t, int32(2000), override.Duration)
	})

	// Create an override with a wildcard identifier
	t.Run("create override with wildcard identifier", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: "*", // Wildcard
			Limit:      5,
			Duration:   2000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.OverrideId, "Override ID should not be empty")

		// Verify the override was created correctly
		override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			OverrideID:  res.Body.Data.OverrideId,
		})

		require.NoError(t, err)
		require.Equal(t, namespaceID, override.NamespaceID)
		require.Equal(t, req.Identifier, override.Identifier)
		require.EqualValues(t, req.Limit, override.Limit)
		require.EqualValues(t, req.Duration, override.Duration)
	})

	t.Run("create same override twice should update existing record", func(t *testing.T) {
		req := handler.Request{
			Namespace:  namespaceID,
			Identifier: "*", // Wildcard
			Limit:      5,
			Duration:   2000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.OverrideId, "Override ID should not be empty")

		req2 := handler.Request{
			Namespace:  namespaceID,
			Identifier: "*", // Wildcard
			Limit:      100,
			Duration:   60000,
		}

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req2)
		require.Equal(t, 200, res2.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res2.Body)
		require.Equal(t, res2.Body.Data.OverrideId, res.Body.Data.OverrideId, "Override ID should be the same")
		// Verify the override was updated correctly
		override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			OverrideID:  res.Body.Data.OverrideId,
		})
		require.NoError(t, err)
		require.EqualValues(t, namespaceID, override.NamespaceID)
		require.EqualValues(t, req2.Identifier, override.Identifier)
		require.EqualValues(t, req2.Limit, override.Limit)
		require.EqualValues(t, req2.Duration, override.Duration)
	})
}
