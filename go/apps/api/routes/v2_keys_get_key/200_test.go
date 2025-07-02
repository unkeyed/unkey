package handler_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_get_key"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func Test_GetKey_ByKeyID(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
		Vault:       h.Vault,
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

	// Create test identities
	identityID := uid.New("identity")
	identityExternalID := "test_user"
	err = db.Query.InsertIdentity(ctx, h.DB.RW(), db.InsertIdentityParams{
		ID:          identityID,
		ExternalID:  identityExternalID,
		WorkspaceID: workspace.ID,
		Environment: "",
		CreatedAt:   time.Now().UnixMilli(),
		Meta:        []byte(`{"role": "admin"}`),
	})
	require.NoError(t, err)

	ratelimitID := uid.New(uid.RatelimitPrefix)
	err = db.Query.InsertIdentityRatelimit(ctx, h.DB.RW(), db.InsertIdentityRatelimitParams{
		ID:          ratelimitID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		IdentityID:  sql.NullString{String: identityID, Valid: true},
		Name:        "api_calls",
		Limit:       100,
		Duration:    60000, // 1 minute
		CreatedAt:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	keyID := uid.New(uid.KeyPrefix)
	key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "test",
		ByteLength: 16,
	})

	metaBytes, _ := json.Marshal([]byte("123"))
	insertParams := db.InsertKeyParams{
		ID:             keyID,
		KeyringID:      keyAuthID,
		Hash:           key.Hash,
		Start:          key.Start,
		WorkspaceID:    workspace.ID,
		ForWorkspaceID: sql.NullString{Valid: false},
		Name:           sql.NullString{Valid: true, String: "test-key"},
		Meta:           sql.NullString{Valid: true, String: string(metaBytes)},
		Expires:        sql.NullTime{Valid: false},
		CreatedAtM:     time.Now().UnixMilli(),
		Enabled:        true,
		IdentityID:     sql.NullString{Valid: true, String: identityID},
	}

	err = db.Query.InsertKey(ctx, h.DB.RW(), insertParams)
	require.NoError(t, err)

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.decrypt_key")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This also tests that we have the correct data for the key.
	// E.g roles, permissions, ratelimits etc.
	t.Run("get key by keyId without decrypting", func(t *testing.T) {
		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("get key by keyId with decrypting", func(t *testing.T) {
		req := handler.Request{
			KeyId:   ptr.P(keyID),
			Decrypt: ptr.P(true),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Plaintext, key.Key)
	})

	t.Run("get key by plaintext key", func(t *testing.T) {
		req := handler.Request{
			Key: ptr.P(key.Key),
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, res.Body.Data.Plaintext, key.Key)
	})
}
