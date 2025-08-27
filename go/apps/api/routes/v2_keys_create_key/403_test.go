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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_create_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateKeyForbidden(t *testing.T) {

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
		ApiCache:  h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create API for testing
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
	})
	require.NoError(t, err)

	err = db.Query.UpdateKeyringKeyEncryption(ctx, h.DB.RW(), db.UpdateKeyringKeyEncryptionParams{
		ID:                 keyAuthID,
		StoreEncryptedKeys: true,
	})
	require.NoError(t, err)

	apiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "test-api",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Create another API for cross-API testing
	otherKeyAuthID := uid.New(uid.KeyAuthPrefix)
	err = db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            otherKeyAuthID,
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
	})
	require.NoError(t, err)

	otherApiID := uid.New(uid.APIPrefix)
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          otherApiID,
		Name:        "other-api",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: otherKeyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	req := handler.Request{
		ApiId: apiID,
	}

	t.Run("no permissions", func(t *testing.T) {
		// Create root key with no permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("wrong permission - has read but not create", func(t *testing.T) {
		// Create root key with read permission instead of create
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("permission for different API", func(t *testing.T) {
		// Create root key with create permission for other API
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.create_key", otherApiID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("permission for specific API but requesting different API", func(t *testing.T) {
		// Create root key with create permission for specific API
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.create_key", otherApiID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Try to create key for different API
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("unrelated permission", func(t *testing.T) {
		// Create root key with completely unrelated permission
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "workspace.read")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("partial permission match", func(t *testing.T) {
		// Create root key with permission that partially matches but isn't sufficient
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.create")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("create recoverable key without perms", func(t *testing.T) {
		// Create root key with permission that partially matches but isn't sufficient
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

		req := handler.Request{
			ApiId:       apiID,
			Recoverable: ptr.P(true),
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})
}
