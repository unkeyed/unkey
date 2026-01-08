package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_remove_roles"
	"golang.org/x/sync/errgroup"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace and root key
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_role_from_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("remove single role", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a role
		roleID := uid.New(uid.TestPrefix)
		roleName := "editor"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Editor role"},
		})
		require.NoError(t, err)

		// Assign role to the key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify key has role initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 1)

		req := handler.Request{
			KeyId: keyID,
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
		require.Len(t, res.Body.Data, 0) // No roles remaining

		// Verify role was removed from key
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 0)
	})

	t.Run("idempotent operation - removing non-assigned role", func(t *testing.T) {
		// Create API and key using testutil helpers
		defaultPrefix := "test"
		defaultBytes := int32(16)
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   workspace.ID,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		keyName := "Test Key"
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Create a role but don't assign it to the key
		roleID := uid.New(uid.TestPrefix)
		roleName := "unassigned_role"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: "Unassigned role"},
		})
		require.NoError(t, err)

		// Verify key has no roles initially
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, currentRoles, 0)

		req := handler.Request{
			KeyId: keyID,
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
		require.Len(t, res.Body.Data, 0) // Still no roles

		// Verify key still has no roles
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, finalRoles, 0)
	})
}

// TestRemoveRolesConcurrent tests that concurrent requests to remove roles
// from the same key don't deadlock. The handler uses SELECT ... FOR UPDATE
// on the key row to serialize concurrent modifications.
func TestRemoveRolesConcurrent(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.remove_role_from_key")

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
		Name:        ptr.P("concurrent-remove-roles-test-key"),
	})

	// Create roles and add them to the key
	numConcurrent := 10
	roles := make([]string, numConcurrent)
	for i := range numConcurrent {
		role := h.CreateRole(seed.CreateRoleRequest{
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("concurrent.remove.role.%d", i),
			Description: ptr.P(fmt.Sprintf("Concurrent role %d", i)),
		})
		roles[i] = role.Name

		// Add role to the key
		err := db.Query.InsertKeyRole(t.Context(), h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyResponse.KeyID,
			RoleID:      role.ID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)
	}

	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			// Each request tries to remove overlapping roles from the same key
			req := handler.Request{
				KeyId: keyResponse.KeyID,
				Roles: []string{roles[i], roles[(i+1)%numConcurrent], roles[(i+2)%numConcurrent]},
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

	// Verify no roles remain (all should have been removed)
	finalRoles, err := db.Query.ListRolesByKeyID(t.Context(), h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.Empty(t, finalRoles)
}
