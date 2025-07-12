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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
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

	// Create test data
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:                 keyAuthID,
		WorkspaceID:        workspace.ID,
		StoreEncryptedKeys: false,
		DefaultPrefix:      sql.NullString{Valid: true, String: "test"},
		DefaultBytes:       sql.NullInt32{Valid: true, Int32: 16},
		CreatedAtM:         time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	keyID := uid.New(uid.KeyPrefix)
	keyString := "test_" + uid.New("")
	err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
		ID:                keyID,
		KeyringID:         keyAuthID,
		Hash:              hash.Sha256(keyString),
		Start:             keyString[:4],
		WorkspaceID:       workspace.ID,
		ForWorkspaceID:    sql.NullString{Valid: false},
		Name:              sql.NullString{Valid: true, String: "Test Key"},
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		IdentityID:        sql.NullString{Valid: false},
		Meta:              sql.NullString{Valid: false},
		Expires:           sql.NullTime{Valid: false},
		RemainingRequests: sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	permissionID := uid.New(uid.TestPrefix)
	err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         "documents.read.forbidden",
		Slug:         "documents.read.forbidden",
		Description:  sql.NullString{Valid: true, String: "Read documents permission"},
	})
	require.NoError(t, err)

	req := handler.Request{
		KeyId: keyID,
		Permissions: []struct {
			Create *bool   `json:"create,omitempty"`
			Id     *string `json:"id,omitempty"`
			Slug   *string `json:"slug,omitempty"`
		}{
			{Id: &permissionID},
		},
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
