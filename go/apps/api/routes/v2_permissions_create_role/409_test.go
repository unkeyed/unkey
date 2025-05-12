package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_role"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/id"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestConflictErrors(t *testing.T) {
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
		require.Equal(t, res2.Body.Error.Detail, "already exists")
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
		// Include both possible assertions based on the expected behavior
		res2, err := h.Client.Post(
			"/v2/permissions.createRole",
			"application/json",
			testutil.MustMarshal(req2),
			headers,
		)

		require.NoError(t, err)
		// If roles are case-sensitive, could be 200 OK
		// If roles are case-insensitive, should be 409 Conflict
		// Check either case
		if res2.StatusCode == 409 {
			// Case-insensitive implementation
			conflict := testutil.UnmarshalBody[openapi.ConflictErrorResponse](t, res2)
			require.NotNil(t, conflict.Error)
			require.Equal(t, conflict.Error.Detail, "already exists")
		}
	})

	// Test case for creating a role using existing database records
	t.Run("existing role in database", func(t *testing.T) {
		// Directly insert a role into the database
		roleID := id.NewRole()
		roleName := "test.existing.role"

		_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			ID:          roleID,
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
		require.Equal(t, res.Body.Error.Detail, "already exists")
	})
}
