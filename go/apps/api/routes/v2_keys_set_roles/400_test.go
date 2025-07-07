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
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_set_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
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

	// Create a test key for valid requests
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

	validKeyID := uid.New(uid.KeyPrefix)
	keyString := "test_" + uid.New("")
	err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
		ID:                validKeyID,
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

	// Test case for missing keyId
	t.Run("missing keyId", func(t *testing.T) {
		req := map[string]interface{}{
			"roles": []map[string]interface{}{
				{"id": "role_123"},
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

	// Test case for missing roles
	t.Run("missing roles", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "key_123",
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

	// Test case for invalid keyId format
	t.Run("invalid keyId format", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "ab", // too short
			"roles": []map[string]interface{}{
				{"id": "role_123"},
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

	// Test case for role with neither id nor name
	t.Run("role with neither id nor name", func(t *testing.T) {
		req := handler.Request{
			KeyId: validKeyID,
			Roles: []struct {
				Id   *string `json:"id,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				{}, // empty role reference
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
		require.Contains(t, res.Body.Error.Detail, "must specify either 'id' or 'name'")
	})

	// Test case for malformed JSON body
	t.Run("malformed JSON body", func(t *testing.T) {
		req := map[string]interface{}{
			"keyId": "key_123",
			"roles": "invalid_not_array",
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

}
