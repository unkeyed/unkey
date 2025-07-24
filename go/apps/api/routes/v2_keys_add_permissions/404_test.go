package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFoundErrors(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key", "rbac.*.add_permission_to_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("key not found", func(t *testing.T) {
		// Create a permission that exists
		permissionID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.404keynotfound",
			Slug:         "documents.read.404keynotfound",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		// Use a non-existent key ID
		nonExistentKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId:       nonExistentKeyID,
			Permissions: []string{permissionID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})

	t.Run("permission not found by ID", func(t *testing.T) {
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
			KeyAuthID:   api.KeyAuthID.String,
			Name:        &keyName,
		})
		keyID := keyResponse.KeyID

		// Use a non-existent permission ID
		nonExistentPermissionID := uid.New(uid.PermissionPrefix)

		req := handler.Request{
			KeyId:       keyID,
			Permissions: []string{nonExistentPermissionID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("key from different workspace", func(t *testing.T) {
		// Create another workspace
		otherWorkspaceID := uid.New(uid.WorkspacePrefix)
		err := db.Query.InsertWorkspace(ctx, h.DB.RW(), db.InsertWorkspaceParams{
			ID:        otherWorkspaceID,
			OrgID:     uid.New("test_org"),
			Name:      "Other Workspace",
			CreatedAt: time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create API and key in the other workspace using testutil helpers
		otherDefaultPrefix := "test"
		otherDefaultBytes := int32(16)
		otherApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   otherWorkspaceID,
			DefaultPrefix: &otherDefaultPrefix,
			DefaultBytes:  &otherDefaultBytes,
		})

		otherKeyName := "Other Workspace Key"
		otherKeyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: otherWorkspaceID,
			KeyAuthID:   otherApi.KeyAuthID.String,
			Name:        &otherKeyName,
		})
		otherKeyID := otherKeyResponse.KeyID

		// Create a permission in our workspace
		permissionID := uid.New(uid.TestPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.404keydifferentws",
			Slug:         "documents.read.404keydifferentws",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId:       otherKeyID,
			Permissions: []string{permissionID},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})
}
