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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_list_overrides"
)

func TestWorkspacePermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	namespaceName := "test_namespace"
	err := db.Query.InsertRatelimitNamespace(ctx, h.DB.RW(), db.InsertRatelimitNamespaceParams{
		ID:          namespaceID,
		WorkspaceID: h.Resources().UserWorkspace.ID, // Use the default workspace
		Name:        namespaceName,
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
		DB:     h.DB,
		Keys:   h.Keys,
		Logger: h.Logger,
	}

	h.Register(route)

	// Create a key for a different workspace
	differentWorkspaceID := "ws_different"
	differentWorkspaceKey := h.CreateRootKey(differentWorkspaceID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
	}

	// Try to access the override with a key from a different workspace
	req := handler.Request{
		Namespace: namespaceID,
	}

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

	// This should return a 404 Not Found (for security reasons we don't reveal if the namespace exists)
	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
