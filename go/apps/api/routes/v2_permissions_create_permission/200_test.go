package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_permission"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestSuccess(t *testing.T) {
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
		require.NotEmpty(t, res.Body.Data.PermissionId)
		require.True(t, len(res.Body.Data.PermissionId) > 0, "PermissionId should not be empty")

		// Verify permission was created in database
		perm, err := h.DB.RO().QueryContext(ctx,
			`SELECT "id", "name", "description", "workspaceId" FROM "permissions" WHERE "id" = $1`,
			res.Body.Data.PermissionId)
		require.NoError(t, err)
		require.True(t, perm.Next(), "Permission should exist in database")

		var id, name, desc, wsID string
		err = perm.Scan(&id, &name, &desc, &wsID)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.PermissionId, id)
		require.Equal(t, req.Name, name)
		require.Equal(t, description, desc)
		require.Equal(t, workspace.ID, wsID)
		perm.Close()

		// Verify audit log was created
		auditLogs, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "auditlogs" WHERE "event" = 'permission.create' AND "resourceId" = $1`,
			res.Body.Data.PermissionId)
		require.NoError(t, err)
		require.True(t, auditLogs.Next(), "Audit log for permission creation should exist")
		auditLogs.Close()
	})

	// Test case for creating a permission without description
	t.Run("create permission without description", func(t *testing.T) {
		req := handler.Request{
			Name: "test.permission.no.desc",
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
		require.NotEmpty(t, res.Body.Data.PermissionId)

		// Verify permission was created in database
		perm, err := h.DB.RO().QueryContext(ctx,
			`SELECT "id", "name", "description", "workspaceId" FROM "permissions" WHERE "id" = $1`,
			res.Body.Data.PermissionId)
		require.NoError(t, err)
		require.True(t, perm.Next(), "Permission should exist in database")

		var id, name, desc, wsID string
		err = perm.Scan(&id, &name, &desc, &wsID)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.PermissionId, id)
		require.Equal(t, req.Name, name)
		require.Empty(t, desc, "Description should be empty")
		require.Equal(t, workspace.ID, wsID)
		perm.Close()
	})
}
