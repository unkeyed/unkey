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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_get_key"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestPreconditionError(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Vault:       h.Vault,
	}

	h.Register(route)

	// Create API manually
	keyAuthID := uid.New(uid.KeyAuthPrefix)
	err := db.Query.InsertKeyring(ctx, h.DB.RW(), db.InsertKeyringParams{
		ID:            keyAuthID,
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		CreatedAtM:    time.Now().UnixMilli(),
		DefaultPrefix: sql.NullString{Valid: false, String: ""},
		DefaultBytes:  sql.NullInt32{Valid: false, Int32: 0},
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

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key", "api.*.encrypt_key")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	keyID := uid.New(uid.KeyPrefix)
	key, _ := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     "test",
		ByteLength: 16,
	})

	insertParams := db.InsertKeyParams{
		ID:             keyID,
		KeyringID:      keyAuthID,
		Hash:           key.Hash,
		Start:          key.Start,
		WorkspaceID:    h.Resources().UserWorkspace.ID,
		ForWorkspaceID: sql.NullString{Valid: false},
		Name:           sql.NullString{Valid: true, String: "test-key"},
		Expires:        sql.NullTime{Valid: false},
		CreatedAtM:     time.Now().UnixMilli(),
		Enabled:        true,
	}

	err = db.Query.InsertKey(ctx, h.DB.RW(), insertParams)
	require.NoError(t, err)

	// Test case for API ID with special characters
	t.Run("Try getting a recoverable key without being opt-in", func(t *testing.T) {
		req := handler.Request{
			Decrypt: ptr.P(true),
			KeyId:   ptr.P(keyID),
		}

		res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 412, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
	})
}
