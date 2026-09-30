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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_remove_roles"
	"golang.org/x/sync/errgroup"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
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
			ProjectID:   api.ProjectID,
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
			ProjectID:   api.ProjectID,
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

func TestRemoveRolesConcurrentSameRole(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
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

	role := h.CreateRole(seed.CreateRoleRequest{
		WorkspaceID: workspace.ID,
		Name:        "kebap.same.role",
	})

	assignRole := func(keyID string) {
		t.Helper()
		err := db.Query.InsertKeyRole(t.Context(), h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      role.ID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)
	}

	warmupKey := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})
	assignRole(warmupKey.KeyID)
	warmup := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		KeyId: warmupKey.KeyID,
		Roles: []string{role.Name},
	})
	require.Equal(t, 200, warmup.Status, "warmup request should succeed")

	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})
	assignRole(keyResponse.KeyID)

	numConcurrent := 10
	g := errgroup.Group{}
	for i := range numConcurrent {
		g.Go(func() error {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				KeyId: keyResponse.KeyID,
				Roles: []string{role.Name},
			})
			if res.Status != 200 {
				return fmt.Errorf("request %d: unexpected status %d", i, res.Status)
			}
			return nil
		})
	}
	require.NoError(t, g.Wait(), "removing the same role concurrently must be idempotent")

	finalRoles, err := db.Query.ListRolesByKeyID(t.Context(), h.DB.RO(), keyResponse.KeyID)
	require.NoError(t, err)
	require.Empty(t, finalRoles)

	disconnectEvents := 0
	for _, ev := range h.FindAuditLogsByTargetID(t.Context(), t, keyResponse.KeyID) {
		if ev.Event == "authorization.disconnect_role_and_key" {
			disconnectEvents++
		}
	}
	require.Equal(t, 1, disconnectEvents)
}
