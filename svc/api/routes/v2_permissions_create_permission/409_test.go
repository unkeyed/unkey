package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_permission"
)

func TestConflictErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for duplicate permission slug
	t.Run("duplicate permission slug", func(t *testing.T) {
		permissionSlug := "test.duplicate.permission"

		// First, create a permission
		req1 := handler.Request{
			Name: permissionSlug + "-1",
			Slug: permissionSlug,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, http.StatusOK, res1.Status, "First permission creation should succeed")
		require.NotNil(t, res1.Body)
		require.NotNil(t, res1.Body.Data)
		require.NotEmpty(t, res1.Body.Data.PermissionId)

		// Now try to create another permission with the same slug
		req2 := handler.Request{
			Name: permissionSlug + "-2",
			Slug: permissionSlug,
		}

		res2 := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](
			h,
			route,
			headers,
			req2,
		)

		require.Equal(t, http.StatusConflict, res2.Status, "Duplicate permission creation should fail with 409")
		require.NotNil(t, res2.Body)
		require.NotNil(t, res2.Body.Error)

		// Our implementation returns just "already exists" as the error detail
		require.Contains(t, res2.Body.Error.Detail, "already exists")
	})
}
