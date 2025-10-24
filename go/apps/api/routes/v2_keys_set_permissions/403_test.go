package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestForbidden(t *testing.T) {
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
		Disabled:     false,
		WorkspaceID:  workspace.ID,
		KeySpaceID:   api.KeyAuthID.String,
		Remaining:    nil,
		IdentityID:   nil,
		Meta:         nil,
		Expires:      nil,
		Name:         nil,
		Deleted:      false,
		RefillAmount: nil,
		RefillDay:    nil,
		Permissions:  nil,
		Roles:        nil,
		Ratelimits:   nil,
	})

	permissionID := uid.New(uid.PermissionPrefix)
	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         "documents.read.forbidden",
		Slug:         "documents.read.forbidden",
		Description:  dbtype.NullString{Valid: true, String: "Read documents permission"},
	})
	require.NoError(t, err)

	req := handler.Request{
		KeyId:       key.KeyID,
		Permissions: []string{permissionID},
	}

	t.Run("missing update_key permission", func(t *testing.T) {
		// Create root key without the required permission
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key") // Wrong permission

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("no permissions at all", func(t *testing.T) {
		// Create root key with no permissions
		rootKey := h.CreateRootKey(workspace.ID) // No permissions specified

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("insufficient permissions", func(t *testing.T) {
		// Create root key with related but insufficient permissions
		rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.create_key") // Missing update_key

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("permission for different resource", func(t *testing.T) {
		// Create root key with permission for different resource
		rootKey := h.CreateRootKey(workspace.ID, "identity.*.update_identity") // Wrong resource type

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})
}
