package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for creating a permission
	t.Run("create permission", func(t *testing.T) {
		description := "Test permission description"
		permissionSlugAndName := "test.permission"
		req := handler.Request{
			Name:        permissionSlugAndName,
			Slug:        "permissionSlugAndNametest-permission",
			Description: &description,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusOK, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)

		// Verify permission was created in database
		perm, err := db.Query.FindPermissionBySlugAndWorkspaceID(ctx, h.DB.RO(), db.FindPermissionBySlugAndWorkspaceIDParams{
			Slug:        permissionSlugAndName,
			WorkspaceID: workspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, req.Name, perm.Name)
		require.Equal(t, req.Slug, perm.Slug)
		require.Equal(t, description, perm.Description.String)
		require.Equal(t, workspace.ID, perm.WorkspaceID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), perm.ID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs, "Audit log for permission creation should exist")

		foundCreateEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "permission.create" {
				foundCreateEvent = true
				break
			}
		}
		require.True(t, foundCreateEvent, "Should find a permission.create audit log event")
	})

	// Test case for creating a permission without description
	t.Run("create permission without description", func(t *testing.T) {
		permissionSlugAndName := "test-permission-no-desc"
		req := handler.Request{
			Name: permissionSlugAndName,
			Slug: permissionSlugAndName,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusOK, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)

		// Verify permission was created in database
		perm, err := db.Query.FindPermissionBySlugAndWorkspaceID(ctx, h.DB.RO(), db.FindPermissionBySlugAndWorkspaceIDParams{
			Slug:        permissionSlugAndName,
			WorkspaceID: workspace.ID,
		})
		require.NoError(t, err)
		require.Equal(t, req.Name, perm.Name)
		require.Equal(t, req.Slug, perm.Slug)
		require.False(t, perm.Description.Valid, "Description should be empty")
		require.Equal(t, workspace.ID, perm.WorkspaceID)
	})
}
