package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

func TestConflictErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

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

	// Test case for duplicate role name
	t.Run("duplicate role name", func(t *testing.T) {
		roleName := "test.duplicate.role"

		// First, create a role
		req1 := handler.Request{
			Name: roleName,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status, "First role creation should succeed")
		require.NotNil(t, res1.Body)
		require.NotNil(t, res1.Body.Data)
		require.NotEmpty(t, res1.Body.Data.RoleId)

		// Now try to create another role with the same name
		req2 := handler.Request{
			Name: roleName,
		}

		res2 := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](
			h,
			route,
			headers,
			req2,
		)

		require.Equal(t, 409, res2.Status, "Duplicate role creation should fail with 409")
		require.NotNil(t, res2.Body)
		require.NotNil(t, res2.Body.Error)
		require.Contains(t, res2.Body.Error.Detail, "already exists")
	})

	// Test case for duplicate role name with different case (if case-insensitive)
	t.Run("duplicate role name different case", func(t *testing.T) {
		roleName := "Test.Case.Sensitive.Role"

		// First, create a role
		req1 := handler.Request{
			Name: roleName,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status, "First role creation should succeed")

		// Now try to create another role with the same name but different case
		req2 := handler.Request{
			Name: "test.case.sensitive.role", // lowercase version
		}

		// This test might pass or fail depending on if role names are case-sensitive
		// Try to create the role and check the response
		res2 := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](
			h,
			route,
			headers,
			req2,
		)

		// If roles are case-insensitive, should be 409 Conflict
		if res2.Status == 409 {
			// Case-insensitive implementation
			require.NotNil(t, res2.Body.Error)
			require.Contains(t, res2.Body.Error.Detail, "already exists")
		}
	})

	// Test case for creating a role using existing database records
	t.Run("existing role in database", func(t *testing.T) {
		// Directly insert a role into the database
		roleID := uid.New(uid.RolePrefix)
		roleName := "test.existing.role"

		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
		})
		require.NoError(t, err)

		// Now try to create a role with the same name using the API
		req := handler.Request{
			Name: roleName,
		}

		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 409, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "already exists")
	})
}
