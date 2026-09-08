package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_delete_override"
)

// TestDeleteOverrideMasksUnauthorizedResources guarantees that callers cannot
// find or delete overrides outside their access scope.
func TestDeleteOverrideMasksUnauthorizedResources(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace in the default workspace
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID, // Use the default workspace
		Name:        uid.New("test"),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create an override in the default workspace
	identifier := "test_identifier"
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       10,
		Duration:    1000,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:             h.DB,
		Auditlogs:      h.Auditlogs,
		NamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)
	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{
		Namespace:  namespaceID,
		Identifier: identifier,
	})
	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/ratelimit_override_not_found", res.Body.Error.Type)
	require.Equal(t, "This override does not exist.", res.Body.Error.Detail)

	// Create a different workspace and key for testing cross-workspace access
	differentWorkspace := h.CreateWorkspace()
	differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
	}

	// Try to delete an override using a namespace from the default workspace
	// but with a key from a different workspace
	req := handler.Request{
		Namespace:  namespaceID,
		Identifier: identifier,
	}

	res = testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)

	// This should return a 404 Not Found (for security reasons we don't reveal if the namespace exists)
	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.NotNil(t, res.Body)

	// Verify the override was NOT deleted
	override, err := db.Query.FindRatelimitOverrideByID(ctx, h.DB.RO(), db.FindRatelimitOverrideByIDParams{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		OverrideID:  overrideID,
	})
	require.NoError(t, err)
	require.False(t, override.DeletedAtM.Valid, "Override should not be deleted")
}
