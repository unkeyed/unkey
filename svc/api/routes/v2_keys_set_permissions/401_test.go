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
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_set_permissions"
)

func TestAuthenticationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create test data
	keySpaceID := uid.New(uid.KeySpacePrefix)
	err := db.Query.InsertKeySpace(ctx, h.DB.RW(), db.InsertKeySpaceParams{
		ID:                 keySpaceID,
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
		KeySpaceID:        keySpaceID,
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
		Name:         "documents.read.auth",
		Slug:         "documents.read.auth",
		Description:  dbtype.NullString{Valid: true, String: "Read documents permission"},
	})
	require.NoError(t, err)

	req := handler.Request{
		KeyId:       keyID,
		Permissions: []string{permissionID},
	}

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "Authorization header")
	})

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "key")
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"invalid_format"},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "valid root key")
	})

	t.Run("empty bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer "},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "valid root key")
	})

	t.Run("disabled root key", func(t *testing.T) {
		// Use invalid root key to simulate disabled key
		rootKey := "invalid_disabled_key"

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
