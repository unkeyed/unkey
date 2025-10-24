package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_add_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
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
		permissionID := uid.New(uid.PermissionPrefix)
		permissionSlug := "documents.read.404keynotfound"
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         permissionSlug,
			Slug:         permissionSlug,
			Description:  dbtype.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		// Use a non-existent key ID
		nonExistentKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId:       nonExistentKeyID,
			Permissions: []string{permissionSlug},
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

	t.Run("key from different workspace", func(t *testing.T) {
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
			KeySpaceID:  otherApi.KeyAuthID.String,
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
			Description:  dbtype.NullString{Valid: true, String: "Read documents permission"},
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
