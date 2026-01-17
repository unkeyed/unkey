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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_set_override"
)

func TestWorkspacePermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	// Create a namespace in the default workspace
	namespaceID := uid.New(uid.RatelimitNamespacePrefix)
	namespaceName := uid.New("name")
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
		Auditlogs:               h.Auditlogs,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	// Create a key for a different workspace
	differentWorkspace := h.CreateWorkspace()
	differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
	}

	// Try to create an override using a namespace from the default workspace
	// but with a key from a different workspace
	req := handler.Request{
		Namespace:  namespaceID,
		Identifier: "test_identifier",
		Limit:      10,
		Duration:   1000,
	}

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

	// This should return a 404 Not Found (for security reasons we don't reveal if the namespace exists)
	require.Equal(t, http.StatusNotFound, res.Status, "got: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
