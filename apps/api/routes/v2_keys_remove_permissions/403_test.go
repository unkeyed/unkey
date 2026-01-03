package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_keys_remove_permissions"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
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

	// Create test data using testutil helper
	defaultPrefix := "test"
	defaultBytes := int32(16)
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   workspace.ID,
		DefaultPrefix: &defaultPrefix,
		DefaultBytes:  &defaultBytes,
	})

	keyName := "Test Key"
	permissionDescription := "Read documents permission"
	keyResponse := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
		Name:        &keyName,
		Permissions: []seed.CreatePermissionRequest{
			{
				WorkspaceID: workspace.ID,
				Name:        "documents.read.remove.auth403",
				Slug:        "documents.read.remove.auth403",
				Description: &permissionDescription,
			},
		},
	})
	keyID := keyResponse.KeyID
	permissionID := keyResponse.PermissionIds[0]

	req := handler.Request{
		KeyId:       keyID,
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
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      "Other Workspace",
			Slug:      uid.New("slug"),
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create API and key in other workspace using testutil helper
		otherApiName := "Other Workspace API"
		otherApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   otherWorkspaceID,
			Name:          &otherApiName,
			DefaultPrefix: &defaultPrefix,
			DefaultBytes:  &defaultBytes,
		})

		otherKeyName := "Other Workspace Key"
		otherKeyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: otherWorkspaceID,
			KeySpaceID:  otherApi.KeyAuthID.String,
			Name:        &otherKeyName,
		})
		otherKeyID := otherKeyResponse.KeyID

		// Create root key for original workspace (authorized for workspace.ID, not otherWorkspaceID)
		authorizedRootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

		reqWithOtherKey := handler.Request{
			KeyId:       otherKeyID,
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
