package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSetOverrideSuccessfully(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New("test_ns")
	namespaceName := "test_namespace"
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources.UserWorkspace.ID,
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources.UserWorkspace.ID, "ratelimit.*.set_override")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a new override by namespace name
	t.Run("create override using namespace name", func(t *testing.T) {
		req := handler.Request{
			NamespaceName: &namespaceName,
			Identifier:    "user_123",
			Limit:         10,
			Duration:      1000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.OverrideId, "Override ID should not be empty")

		// Verify the override was created correctly
		override, err := db.Query.FindRatelimitOverrideById(ctx, h.DB.RO(), db.FindRatelimitOverrideByIdParams{
			WorkspaceID: h.Resources.UserWorkspace.ID,
			OverrideID:  res.Body.OverrideId,
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
			NamespaceId: &namespaceID,
			Identifier:  "user_456",
			Limit:       20,
			Duration:    2000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %v", res.Body)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.OverrideId, "Override ID should not be empty")

		// Verify the override was created correctly
		override, err := db.Query.FindRatelimitOverrideById(ctx, h.DB.RO(), db.FindRatelimitOverrideByIdParams{
			WorkspaceID: h.Resources.UserWorkspace.ID,
			OverrideID:  res.Body.OverrideId,
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
			NamespaceId: &namespaceID,
			Identifier:  "*", // Wildcard
			Limit:       5,
			Duration:    2000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.OverrideId, "Override ID should not be empty")

		// Verify the override was created correctly
		override, err := db.Query.FindRatelimitOverrideById(ctx, h.DB.RO(), db.FindRatelimitOverrideByIdParams{
			WorkspaceID: h.Resources.UserWorkspace.ID,
			OverrideID:  res.Body.OverrideId,
		})
		require.NoError(t, err)
		require.Equal(t, namespaceID, override.NamespaceID)
		require.Equal(t, "*", override.Identifier)
		require.Equal(t, int32(5), override.Limit)
		require.Equal(t, int32(2000), override.Duration)
	})
}
