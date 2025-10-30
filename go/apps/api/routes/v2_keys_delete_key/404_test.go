package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_delete_key"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestKeyDeleteNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		LiveKeyCache: h.Caches.LiveKeyByID,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.delete_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create a workspace and user
	workspace := h.Resources().UserWorkspace

	// Create a keyAuth (keyring) for the API
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   workspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false},
		DefaultBytes:  sql.NullInt32{Valid: false},
	})
	require.NoError(t, err)

	// Create a test API
	apiID := uid.New("api")
	err = db.Query.InsertApi(ctx, h.DB.RW(), db.InsertApiParams{
		ID:          apiID,
		Name:        "Test API",
		WorkspaceID: workspace.ID,
		AuthType:    db.NullApisAuthType{Valid: true, ApisAuthType: db.ApisAuthTypeKey},
		KeyAuthID:   sql.NullString{Valid: true, String: keyAuthID},
		CreatedAtM:  time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	keyID := uid.New(uid.KeyPrefix)
	key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "test",
		ByteLength: 16,
	})

	insertParams := db.InsertKeyParams{
		ID:                keyID,
		KeyringID:         keyAuthID,
		Hash:              key.Hash,
		Start:             key.Start,
		WorkspaceID:       workspace.ID,
		ForWorkspaceID:    sql.NullString{Valid: false},
		Name:              sql.NullString{Valid: true, String: "test-key"},
		Expires:           sql.NullTime{Valid: false},
		CreatedAtM:        time.Now().UnixMilli(),
		Enabled:           true,
		IdentityID:        sql.NullString{Valid: false, String: ""},
		RemainingRequests: sql.NullInt32{Int32: 100, Valid: true},
	}

	err = db.Query.InsertKey(ctx, h.DB.RW(), insertParams)
	require.NoError(t, err)

	err = db.Query.SoftDeleteKeyByID(ctx, h.DB.RW(), db.SoftDeleteKeyByIDParams{
		Now: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		ID:  keyID,
	})
	require.NoError(t, err)

	t.Run("nonexistent keyId", func(t *testing.T) {
		req := handler.Request{
			KeyId: uid.New(uid.KeyPrefix),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not find the requested key")
	})

	t.Run("can't delete soft deleted key", func(t *testing.T) {
		req := handler.Request{
			KeyId: keyID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not find the requested key")
	})
}
