package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_role"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for missing required name
	t.Run("missing name", func(t *testing.T) {
		req := handler.Request{
			// Name is missing
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

	// Test case for empty name
	t.Run("empty name", func(t *testing.T) {
		req := handler.Request{
			Name: "", // Empty string is invalid
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
			"/v2/permissions.createRole",
			"application/json",
			[]byte("{malformed json"),
			headers,
		)

		require.NoError(t, err)
		require.Equal(t, 400, res.StatusCode)
	})

	// Test case for invalid permission IDs
	t.Run("invalid permission IDs", func(t *testing.T) {
		invalidPermIDs := []string{"perm_does_not_exist"}
		req := handler.Request{
			Name:          "test.role.invalid.perms",
			PermissionIds: &invalidPermIDs,
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
		require.Equal(t, res.Body.Error.Detail, "not found")
	})

	// Test for very long description
	t.Run("very long description", func(t *testing.T) {
		// Create a very long description (more than would be reasonable)
		veryLongDesc := ""
		for i := 0; i < 10000; i++ {
			veryLongDesc += "a"
		}

		req := handler.Request{
			Name:        "test.role",
			Description: &veryLongDesc,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
	})
}
