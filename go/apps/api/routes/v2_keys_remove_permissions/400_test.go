package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_remove_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestValidationErrors(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "api.*.update_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a valid API and key for testing using testutil helper
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
	validKeyID := keyResponse.KeyID

	t.Run("missing keyId", func(t *testing.T) {
		req := map[string]interface{}{
			"permissions": []map[string]interface{}{
				{"id": "perm_123"},
			},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	t.Run("empty keyId", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "",
			"permissions": []map[string]interface{}{
				{"id": "perm_123"},
			},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	t.Run("missing permissions", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": validKeyID,
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	t.Run("empty permissions array", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId":       validKeyID,
			"permissions": []map[string]interface{}{},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	t.Run("invalid keyId format", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "ab", // too short
			"permissions": []map[string]interface{}{
				{"id": "perm_123"},
			},
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	t.Run("permission missing both id and slug", func(t *testing.T) {
		// Create a permission for valid structure
		permissionID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.remove.validation",
			Slug:         "documents.read.remove.validation",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		req := handler.Request{
			KeyId: validKeyID,
			Permissions: []struct {
				Id   *string `json:"id,omitempty"`
				Slug *string `json:"slug,omitempty"`
			}{
				{}, // Neither id nor slug provided
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("permission not found by id", func(t *testing.T) {
		nonExistentPermissionID := uid.New(uid.TestPrefix)

		req := handler.Request{
			KeyId: validKeyID,
			Permissions: []struct {
				Id   *string `json:"id,omitempty"`
				Slug *string `json:"slug,omitempty"`
			}{
				{Id: &nonExistentPermissionID},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("permission not found by name", func(t *testing.T) {
		nonExistentPermissionName := "nonexistent.permission.remove"

		req := handler.Request{
			KeyId: validKeyID,
			Permissions: []struct {
				Id   *string `json:"id,omitempty"`
				Slug *string `json:"slug,omitempty"`
			}{
				{Slug: &nonExistentPermissionName},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "was not found")
	})

	t.Run("key not found", func(t *testing.T) {
		// Create a permission that exists
		permissionID := uid.New(uid.TestPrefix)
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         "documents.read.remove.keynotfound",
			Slug:         "documents.read.remove.keynotfound",
			Description:  sql.NullString{Valid: true, String: "Read documents permission"},
		})
		require.NoError(t, err)

		nonExistentKeyID := uid.New(uid.KeyPrefix)

		req := handler.Request{
			KeyId: nonExistentKeyID,
			Permissions: []struct {
				Id   *string `json:"id,omitempty"`
				Slug *string `json:"slug,omitempty"`
			}{
				{Id: &permissionID},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "key was not found")
	})
}
