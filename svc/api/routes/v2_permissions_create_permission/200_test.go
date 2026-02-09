package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_permission"
)

func TestSuccess(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for creating a permission
	t.Run("create permission", func(t *testing.T) {
		description := "Test permission description"
		req := handler.Request{
			Name:        "test.permission",
			Slug:        "test-permission",
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
		require.NotEmpty(t, res.Body.Data.PermissionId)
		require.True(t, len(res.Body.Data.PermissionId) > 0, "PermissionId should not be empty")

		// Verify permission was created in database
		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), res.Body.Data.PermissionId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.PermissionId, perm.ID)
		require.Equal(t, req.Name, perm.Name)
		require.Equal(t, req.Slug, perm.Slug)
		require.Equal(t, description, perm.Description.String)
		require.Equal(t, workspace.ID, perm.WorkspaceID)

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), res.Body.Data.PermissionId)
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
		req := handler.Request{
			Name: "test.permission.no.desc",
			Slug: "test-permission-no-desc",
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
		require.NotEmpty(t, res.Body.Data.PermissionId)

		// Verify permission was created in database
		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), res.Body.Data.PermissionId)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.PermissionId, perm.ID)
		require.Equal(t, req.Name, perm.Name)
		require.Equal(t, req.Slug, perm.Slug)
		require.False(t, perm.Description.Valid, "Description should be empty")
		require.Equal(t, workspace.ID, perm.WorkspaceID)
	})
}
