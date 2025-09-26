package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestWorkspacePermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace in the default workspace
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	namespaceName := uid.New("test")
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID, // Use the default workspace
		Name:        namespaceName,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Ratelimit:               h.Ratelimit,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:               h.Auditlogs,
	}

	h.Register(route)

	// Create a key for a different workspace
	differentWorkspace := h.CreateWorkspace()
	differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
	}

	// Try to access the namespace from the default workspace with a key from a different workspace
	req := handler.Request{
		Namespace:  namespaceName,
		Identifier: "user_123",
		Limit:      100,
		Duration:   60000,
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)

	// This should return a 403 Forbidden - user lacks create_namespace permission
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %d, body: %s", res.Status, res.RawBody)
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Detail, "create_namespace", "Error should mention missing create_namespace permission")
}

func TestInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Ratelimit:               h.Ratelimit,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:               h.Auditlogs,
	}

	h.Register(route)

	t.Run("has limit permission but no create_namespace permission", func(t *testing.T) {
		// Use a namespace that doesn't exist
		nonExistentNamespace := uid.New("nonexistent")

		// Create a key that can limit any namespace but cannot create namespaces
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.limit")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Namespace:  nonExistentNamespace,
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)

		// Should return 403 because user has some permissions but not create_namespace
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %d, body: %s", res.Status, res.RawBody)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "create_namespace", "Error should mention missing create_namespace permission")

		// Verify the namespace was NOT created
		ctx := context.Background()
		_, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Namespace:   nonExistentNamespace,
		})
		require.True(t, db.IsNotFound(err), "Namespace should not have been created when user lacks create_namespace permission")
	})
}
