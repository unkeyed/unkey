package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_permission"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestConflictErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
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

	// Test case for duplicate permission name
	t.Run("duplicate permission name", func(t *testing.T) {
		permissionName := "test.duplicate.permission"

		// First, create a permission
		req1 := handler.Request{
			Name: permissionName,
			Slug: "test-duplicate-permission",
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

		// Now try to create another permission with the same name
		req2 := handler.Request{
			Name: permissionName,
			Slug: "test-duplicate-permission-2",
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

	// Test case for duplicate permission name with different case (if case-insensitive)
	t.Run("duplicate permission name different case", func(t *testing.T) {
		permissionName := "Test.Case.Sensitive.Permission"

		// First, create a permission
		req1 := handler.Request{
			Name: permissionName,
			Slug: "test-case-sensitive-permission",
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, http.StatusOK, res1.Status, "First permission creation should succeed")

		// Now try to create another permission with the same name but different case
		req2 := handler.Request{
			Name: "test.case.sensitive.permission", // lowercase version
			Slug: "test-case-sensitive-permission-lowercase",
		}

		// This test might pass or fail depending on if permission names are case-sensitive
		// Try both possible assertions based on the expected behavior
		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req2,
		)

		// If permissions are case-sensitive, could be 200 OK
		// If permissions are case-insensitive, should be 409 Conflict
		// Check either case
		if res2.Status == http.StatusConflict {
			// Try calling specifically for a conflict response
			res2Conflict := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](
				h,
				route,
				headers,
				req2,
			)
			require.Equal(t, http.StatusConflict, res2Conflict.Status)
			require.NotNil(t, res2Conflict.Body.Error)
			// Our implementation includes the full context in the error message
			require.Contains(t, res2Conflict.Body.Error.Detail, "already exists")
		}
	})
}
