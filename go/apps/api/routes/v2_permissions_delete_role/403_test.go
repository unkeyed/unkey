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
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestForbidden(t *testing.T) {
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

	// Create a role for testing
	roleID := uid.New(uid.TestPrefix)
	roleName := "test.role.forbidden"
	roleDesc := "Test role for forbidden test"
	createdAt := time.Now()

	_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		ID:          roleID,
		WorkspaceID: workspace.ID,
		Name:        roleName,
		Description: db.NewNullString(roleDesc),
		CreatedAtM:  db.NewNullTime(createdAt),
	})
	if err != nil {
		t.Fatalf("Failed to create test role: %v", err)
	}

	t.Run("missing required permission", func(t *testing.T) {
		// Create a root key with a different permission (not rbac.*.delete_role)
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				RoleId: roleID,
			},
		)

		testutil.RequireForbidden(t, res)
	})

	t.Run("no permissions", func(t *testing.T) {
		// Create a root key with no permissions
		rootKey := h.CreateRootKey(workspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				RoleId: roleID,
			},
		)

		testutil.RequireForbidden(t, res)
	})
}
