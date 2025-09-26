package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestWorkspacePermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID, // Use the default workspace
		Name:        uid.New("test"),
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create an override
	identifier := "test_identifier"
	overrideID := uid.New(uid.RatelimitOverridePrefix)
	err = db.Query.InsertRatelimitOverride(ctx, h.DB.RW(), db.InsertRatelimitOverrideParams{
		ID:          overrideID,
		WorkspaceID: h.Resources().UserWorkspace.ID, // In default workspace
		NamespaceID: namespaceID,
		Identifier:  identifier,
		Limit:       10,
		Duration:    1000,
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)
	differentWorkspace := h.CreateWorkspace()
	// Create a key for a different workspace
	differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
	}

	// Try to access the override with a key from a different workspace
	req := handler.Request{
		Namespace:  namespaceID,
		Identifier: identifier,
	}

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

	// This should return a 404 Not Found (for security reasons we don't reveal if the namespace exists)
	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
