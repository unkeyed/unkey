package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_limit"
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
		Ratelimit:      h.Ratelimit,
		AuditLogBuffer: h.AuditLogBuffer,
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:      h.Auditlogs,
	}

	h.Register(route)

	// Create a key for a different workspace
	differentWorkspace := h.CreateWorkspace()
	differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID,
		rbac.U(urn.New().Workspace(differentWorkspace.ID).Project("*").RatelimitNamespace("*"), permissions.Limit).Value,
	)

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

	// A foreign namespace must look like a missing namespace in the caller's workspace.
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %d, body: %s", res.Status, res.RawBody)
	require.NotNil(t, res.Body)
	require.Contains(t, res.Body.Error.Detail, "#write")
	req.Namespace = uid.New("missing")
	missing := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, missing.Status, res.Status)
	require.Equal(t, missing.Body.Error, res.Body.Error)
	require.NotContains(t, res.Body.Error.Detail, namespaceID)
	require.NotContains(t, res.Body.Error.Detail, h.Resources().UserWorkspace.ID)
}

func TestInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Ratelimit:      h.Ratelimit,
		AuditLogBuffer: h.AuditLogBuffer,
		DB:             h.DB,
		NamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:      h.Auditlogs,
	}

	h.Register(route)

	t.Run("has limit permission but no write permission", func(t *testing.T) {
		// Use a namespace that doesn't exist
		nonExistentNamespace := uid.New("nonexistent")

		// Create a key that can limit any namespace but cannot create namespaces
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, rbac.U(urn.New().Workspace(h.Resources().UserWorkspace.ID).Project("*").RatelimitNamespace("*"), permissions.Limit).Value)

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

		// Namespace creation requires write permission even when limit is granted.
		require.Equal(t, http.StatusForbidden, res.Status, "expected 403, got: %d, body: %s", res.Status, res.RawBody)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "#write")

		// Verify the namespace was NOT created
		ctx := context.Background()
		_, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Namespace:   nonExistentNamespace,
		})
		require.True(t, db.IsNotFound(err), "Namespace should not have been created when user lacks write permission")
	})
}
