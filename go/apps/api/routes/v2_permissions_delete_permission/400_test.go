package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_permission"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for missing required permissionId
	t.Run("missing permissionId", func(t *testing.T) {
		req := handler.Request{
			// PermissionId is missing
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "invalid")
	})

	// Test case for empty permissionId
	t.Run("empty permissionId", func(t *testing.T) {
		req := handler.Request{
			PermissionId: "", // Empty string is invalid
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "invalid")
	})

	// Test case for malformed JSON body
	t.Run("malformed JSON body", func(t *testing.T) {
		res, err := h.Client.Post(
			"/v2/permissions.deletePermission",
			"application/json",
			[]byte("{malformed json"),
			headers,
		)

		require.NoError(t, err)
		require.Equal(t, 400, res.StatusCode)
	})

	// Test case for invalid permissionId format
	t.Run("invalid permissionId format", func(t *testing.T) {
		req := handler.Request{
			PermissionId: "not_a_valid_permission_id_format",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		// Note: This might pass if there's no format validation for permissionId
	})
}
