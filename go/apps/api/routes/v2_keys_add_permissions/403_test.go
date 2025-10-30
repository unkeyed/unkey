package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
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

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Credits:     nil,
		IdentityID:  nil,
		Meta:        nil,
		Expires:     nil,
		Name:        nil,
		Deleted:     false,
		Permissions: nil,
		Roles:       nil,
		Ratelimits:  nil,
	})

	permissionID := uid.New(uid.TestPrefix)
	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         "documents.read.auth403",
		Slug:         "documents.read.auth403",
		Description:  dbtype.NullString{Valid: true, String: "Read documents permission"},
	})
	require.NoError(t, err)

	req := handler.Request{
		KeyId:       key.KeyID,
		Permissions: []string{permissionID},
	}

	t.Run("root key without required permissions", func(t *testing.T) {
		// Create root key without the required permission
		insufficientRootKey := h.CreateRootKey(workspace.ID, "some.other.permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", insufficientRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("root key with partial permissions", func(t *testing.T) {
		// Create root key with related but insufficient permission
		partialRootKey := h.CreateRootKey(workspace.ID, "api.read.update_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", partialRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("key belongs to different workspace", func(t *testing.T) {
		diffWorkspace := h.CreateWorkspace()

		diffApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   diffWorkspace.ID,
			IpWhitelist:   "",
			EncryptedKeys: false,
			Name:          nil,
			CreatedAt:     nil,
			DefaultPrefix: nil,
			DefaultBytes:  nil,
		})

		diffKey := h.CreateKey(seed.CreateKeyRequest{
			Disabled:    false,
			WorkspaceID: diffWorkspace.ID,
			KeySpaceID:  diffApi.KeyAuthID.String,
			Credits:     nil,
			IdentityID:  nil,
			Meta:        nil,
			Expires:     nil,
			Name:        nil,
			Deleted:     false,
			Permissions: nil,
			Roles:       nil,
			Ratelimits:  nil,
		})

		// Create root key for original workspace (authorized for workspace.ID, not otherWorkspaceID)
		authorizedRootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_permission_to_key")

		reqWithOtherKey := handler.Request{
			KeyId:       diffKey.KeyID,
			Permissions: []string{permissionID},
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", authorizedRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			reqWithOtherKey,
		)

		require.Equal(t, 404, res.Status) // Key not found (because it belongs to different workspace)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})

	t.Run("root key with no permissions", func(t *testing.T) {
		// Create root key with no permissions
		noPermissionsRootKey := h.CreateRootKey(workspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", noPermissionsRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})
}
