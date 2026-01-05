package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for creating a role without permissions
	t.Run("create role without permissions", func(t *testing.T) {
		roleName := "test.role.no.permissions"
		description := "Test role without permissions"
		req := handler.Request{
			Name:        roleName,
			Description: &description,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotEmpty(t, res.Body.Data.RoleId)
		require.True(t, len(res.Body.Data.RoleId) > 0, "RoleId should not be empty")

		// Verify role was created in database
		role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, role.ID)
		require.Equal(t, req.Name, role.Name)
		require.Equal(t, description, role.Description.String)
		require.Equal(t, workspace.ID, role.WorkspaceID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs, "Audit log for role creation should exist")

		foundCreateEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "role.create" {
				foundCreateEvent = true
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find a role.create audit log event")
	})

	// Test case for creating a role with description
	t.Run("create role with description", func(t *testing.T) {
		roleName := "test.role.with.description"
		description := "Test role with a description field"

		req := handler.Request{
			Name:        roleName,
			Description: &description,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotEmpty(t, res.Body.Data.RoleId)

		// Verify role was created in database
		role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, role.ID)
		require.Equal(t, req.Name, role.Name)
		require.Equal(t, description, role.Description.String)
		require.Equal(t, workspace.ID, role.WorkspaceID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs, "Audit log for role creation should exist")

		foundCreateEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "role.create" {
				foundCreateEvent = true
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find a role.create audit log event")
	})

	// Test case for creating a role without description
	t.Run("create role without description", func(t *testing.T) {
		req := handler.Request{
			Name: "test.role.no.desc",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotEmpty(t, res.Body.Data.RoleId)

		// Verify role was created in database
		role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), res.Body.Data.RoleId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, role.ID)
		require.Equal(t, req.Name, role.Name)
		require.False(t, role.Description.Valid, "Description should be null")
		require.Equal(t, workspace.ID, role.WorkspaceID)
	})
}
