package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_reroll_key"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestCreateKeySuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key", "api.*.encrypt_key")

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		EncryptedKeys: true,
		IpWhitelist:   "",
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	workspace := h.Resources().UserWorkspace

	identityID := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "test_123",
		Meta:        []byte(`{"name": "Test User"}`),
		Ratelimits: []seed.CreateRatelimitRequest{
			{
				Name:        "default-enterprise",
				WorkspaceID: workspace.ID,
				AutoApply:   true,
				Duration:    time.Minute.Milliseconds(),
				Limit:       1500,
				IdentityID:  nil,
				KeyID:       nil, // will be set by the seeder
			},
		},
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID:  workspace.ID,
		Disabled:     false,
		KeyAuthID:    api.KeyAuthID.String,
		Remaining:    ptr.P(int32(16)),
		IdentityID:   ptr.P(identityID),
		Meta:         nil,
		Expires:      nil,
		Name:         ptr.P("Test-Key"),
		Deleted:      false,
		Recoverable:  true,
		RefillAmount: ptr.P(int32(100)),
		RefillDay:    ptr.P(int16(1)),
		Permissions: []seed.CreatePermissionRequest{
			{
				Name:        "Read documents",
				Slug:        "documents.read",
				Description: nil,
				WorkspaceID: workspace.ID,
			},
		},
		Roles: []seed.CreateRoleRequest{
			{
				Name:        "editor",
				WorkspaceID: workspace.ID,
				Description: nil,
				Permissions: []seed.CreatePermissionRequest{
					{
						Name:        "Edit documents",
						Slug:        "documents.edit",
						Description: nil,
						WorkspaceID: workspace.ID,
					},
				},
			},
		},
		Ratelimits: []seed.CreateRatelimitRequest{
			{
				Name:        "default",
				WorkspaceID: workspace.ID,
				AutoApply:   true,
				Duration:    time.Minute.Milliseconds(),
				Limit:       15,
				IdentityID:  nil,
				KeyID:       nil, // will be set by the seeder
			},
		},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test basic key creation
	req := handler.Request{
		KeyId:     key.KeyID,
		Remaining: 0,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	require.NotEmpty(t, res.Body.Data.KeyId)
	require.NotEmpty(t, res.Body.Data.Key)
	require.NotEmpty(t, res.Body.Meta.RequestId)

	createdKeyRow, err := db.Query.FindLiveKeyByID(ctx, h.DB.RW(), key.KeyID)
	require.NoError(t, err)
	require.NotNil(t, createdKeyRow)

	rolledKeyRow, err := db.Query.FindLiveKeyByID(ctx, h.DB.RW(), res.Body.Data.KeyId)
	require.NoError(t, err)
	require.NotNil(t, rolledKeyRow)

	require.NotEqual(t, createdKeyRow.ID, rolledKeyRow.ID)
	require.Equal(t, createdKeyRow.Name.String, rolledKeyRow.Name.String)
	require.Equal(t, createdKeyRow.IdentityID.String, rolledKeyRow.IdentityID.String)
	require.Equal(t, createdKeyRow.Meta, rolledKeyRow.Meta)
	require.Equal(t, createdKeyRow.RefillDay.Int16, rolledKeyRow.RefillDay.Int16)
	require.Equal(t, createdKeyRow.RefillAmount.Int32, rolledKeyRow.RefillAmount.Int32)
	require.Equal(t, createdKeyRow.RemainingRequests.Int32, rolledKeyRow.RemainingRequests.Int32)

	// The first key should expire
	require.True(t, createdKeyRow.Expires.Valid)
	require.True(t, createdKeyRow.EncryptedKey.Valid)
	require.True(t, createdKeyRow.EncryptionKeyID.Valid)

	require.True(t, rolledKeyRow.EncryptedKey.Valid)
	require.True(t, rolledKeyRow.EncryptionKeyID.Valid)

	createdKey := db.ToKeyData(createdKeyRow)
	rolledKey := db.ToKeyData(rolledKeyRow)

	require.Len(t, createdKey.Permissions, len(rolledKey.Permissions))
	require.Len(t, createdKey.Roles, len(rolledKey.Roles))
	require.Len(t, createdKey.Ratelimits, len(rolledKey.Ratelimits))
}
