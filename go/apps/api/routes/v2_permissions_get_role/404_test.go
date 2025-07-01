package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_role"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFoundErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for non-existent role ID
	t.Run("non-existent role ID", func(t *testing.T) {
		req := handler.Request{
			RoleId: "role_does_not_exist",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "does not exist")
	})

	// Test case for valid-looking but non-existent role ID
	t.Run("valid-looking but non-existent role ID", func(t *testing.T) {
		nonExistentID := uid.New(uid.TestPrefix) // Generate a valid ID format that doesn't exist

		req := handler.Request{
			RoleId: nonExistentID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "does not exist")
	})
}
