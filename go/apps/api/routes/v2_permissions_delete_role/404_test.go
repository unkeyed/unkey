package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_role"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/id"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestNotFound(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("nonexistent role", func(t *testing.T) {
		// Generate a role ID that doesn't exist in the database
		nonexistentRoleID := id.NewRole()

		req := handler.Request{
			RoleId: nonexistentRoleID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		testutil.RequireNotFound(t, res)
	})

	t.Run("role from different workspace", func(t *testing.T) {
		// Create another workspace
		anotherWorkspace := h.CreateWorkspace()

		// Create a role in another workspace
		roleID := id.NewRole()
		roleName := "test.role.other.workspace"
		roleDesc := "Test role in another workspace"
		createdAt := time.Now()

		_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			ID:          roleID,
			WorkspaceID: anotherWorkspace.ID,
			Name:        roleName,
			Description: db.NewNullString(roleDesc),
			CreatedAtM:  db.NewNullTime(createdAt),
		})
		if err != nil {
			t.Fatalf("Failed to create test role: %v", err)
		}

		// Try to delete the role from the first workspace
		req := handler.Request{
			RoleId: roleID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		testutil.RequireNotFound(t, res)
	})
}
