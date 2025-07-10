package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_credits"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestKeyUpdateCreditsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		KeyCache:  h.Caches.VerificationKeyByHash,
	}

	h.Register(route)

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
		RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
	}

	err = db.Query.InsertKey(ctx, h.DB.RW(), insertParams)
	require.NoError(t, err)

	// Create root key with read permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing keyId", func(t *testing.T) {
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/keys.updateCredits' failed to validate schema")
	})

	t.Run("empty keyId string", func(t *testing.T) {
		req := handler.Request{
			KeyId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("increment with null throws error", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Increment,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("can't increment unlimited key", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Increment,
			Value:     nullable.NewNullableWithValue(int64(1)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("decrement with null throws error", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Decrement,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})

	t.Run("can't decrement unlimited key", func(t *testing.T) {
		req := handler.Request{
			KeyId:     keyID,
			Operation: openapi.Decrement,
			Value:     nullable.NewNullableWithValue(int64(1)),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
