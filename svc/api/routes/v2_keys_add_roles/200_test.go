package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_add_roles"
	"golang.org/x/sync/errgroup"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_role_to_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("add single role by name", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		roleName := "editor_single_name"
		key := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
			Roles: []seed.CreateRoleRequest{
				{
					WorkspaceID: workspace.ID,
					Name:        "editor_single_name",
					Description: ptr.P(roleName),
				},
			},
		})

		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{roleName},
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
		require.Len(t, res.Body.Data, 1)
		require.Equal(t, key.RolesIds[0], res.Body.Data[0].Id)
		require.Equal(t, roleName, res.Body.Data[0].Name)

		// Verify role was added to key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 1)
		require.Equal(t, key.RolesIds[0], finalRoles[0].ID)
	})

	t.Run("idempotent behavior - add existing roles", func(t *testing.T) {
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			EncryptedKeys: false,
		})

		key := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  api.KeyAuthID.String,
			WorkspaceID: workspace.ID,
		})

		adminName := "admin_idempotent"
		admin := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        adminName,
			Description: ptr.P("admin_idempotent"),
		})

		editorName := "editor_idempotent"
		h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        editorName,
			Description: ptr.P("editor_idempotent"),
		})

		// First, add admin role to the key
		err := db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       key.KeyID,
			RoleID:      admin.ID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Now try to add both admin (existing) and editor (new) roles
		req := handler.Request{
			KeyId: key.KeyID,
			Roles: []string{adminName, editorName},
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
		require.Len(t, res.Body.Data, 2)

		// Verify both roles are present
		roleNames := []string{res.Body.Data[0].Name, res.Body.Data[1].Name}
		require.Equal(t, []string{"admin_idempotent", "editor_idempotent"}, roleNames)

		// Verify roles in database
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 2)

		// Verify audit logs - should only have one new log for the editor role
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), key.KeyID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs)

		// Count connect events that mention editor (the new role)
		editorConnectEvents := 0
		for _, log := range auditLogs {
			if log.AuditLog.Event == "authorization.connect_role_and_key" &&
				log.AuditLog.Display == fmt.Sprintf("Added role editor_idempotent to key %s", key.KeyID) {
				editorConnectEvents++
			}
		}
		require.Equal(t, 1, editorConnectEvents, "Should find only 1 new role connect event for editor")
	})
}

// TestAddRolesConcurrent tests that concurrent requests to add roles
// to the same key don't deadlock. The handler uses SELECT ... FOR UPDATE
// on the key row to serialize concurrent modifications.
func TestAddRolesConcurrent(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_role_to_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})

	// Create a single key that all concurrent requests will update
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        ptr.P("concurrent-add-roles-test-key"),
	})

	// Create roles that will be added concurrently
	numConcurrent := 10
	roles := make([]string, numConcurrent)
	for i := range numConcurrent {
		role := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("concurrent.add.role.%d", i),
			Description: ptr.P(fmt.Sprintf("Concurrent role %d", i)),
		})
		roles[i] = role.Name
	}

	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			// Each request adds a unique role to the same key
			// This tests deadlock prevention without triggering duplicate entry issues
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Roles: []string{roles[i]},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if res.Status != 200 {
				return fmt.Errorf("request %d: unexpected status %d", i, res.Status)
			}
			return nil
		})
	}

	err := g.Wait()
	require.NoError(t, err, "All concurrent updates should succeed without deadlock")

	// Verify all roles were added
	finalRoles, err := db.Query.ListRolesByKeyID(t.Context(), h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.Len(t, finalRoles, numConcurrent)
}
