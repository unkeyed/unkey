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

func TestRerollKeySuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		Logger:       h.Logger,
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		Vault:        h.Vault,
		KeyCache:     h.Caches.VerificationKeyByHash,
		LiveKeyCache: h.Caches.LiveKeyByID,
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

	identity := h.CreateIdentity(seed.CreateIdentityRequest{
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

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("successfully rerolled key with all options", func(t *testing.T) {
		t.Parallel()
		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID:  workspace.ID,
			Disabled:     false,
			KeySpaceID:   api.KeyAuthID.String,
			Remaining:    ptr.P(int32(16)),
			IdentityID:   ptr.P(identity.ID),
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

		req := handler.Request{
			KeyId:      key.KeyID,
			Expiration: 0,
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

		require.False(t, rolledKeyRow.Expires.Valid)
		require.True(t, rolledKeyRow.EncryptedKey.Valid)
		require.True(t, rolledKeyRow.EncryptionKeyID.Valid)

		createdKey := db.ToKeyData(createdKeyRow)
		rolledKey := db.ToKeyData(rolledKeyRow)

		// Compare permissions - build sets of all permission slugs (direct + from roles)
		createdPermSet := make(map[string]struct{})
		for _, perm := range createdKey.Permissions {
			createdPermSet[perm.Slug] = struct{}{}
		}
		for _, perm := range createdKey.RolePermissions {
			createdPermSet[perm.Slug] = struct{}{}
		}

		rolledPermSet := make(map[string]struct{})
		for _, perm := range rolledKey.Permissions {
			rolledPermSet[perm.Slug] = struct{}{}
		}
		for _, perm := range rolledKey.RolePermissions {
			rolledPermSet[perm.Slug] = struct{}{}
		}

		require.Equal(t, createdPermSet, rolledPermSet, "permission sets should be equal")

		// Compare roles by name
		createdRoleSet := make(map[string]struct{})
		for _, role := range createdKey.Roles {
			createdRoleSet[role.Name] = struct{}{}
		}

		rolledRoleSet := make(map[string]struct{})
		for _, role := range rolledKey.Roles {
			rolledRoleSet[role.Name] = struct{}{}
		}

		require.Equal(t, createdRoleSet, rolledRoleSet, "role sets should be equal")

		// Compare ratelimits by name and verify values match
		type ratelimitData struct {
			Limit    int32
			Duration int64
		}

		createdRatelimitMap := make(map[string]ratelimitData)
		for _, rl := range createdKey.Ratelimits {
			createdRatelimitMap[rl.Name] = ratelimitData{
				Limit:    rl.Limit,
				Duration: rl.Duration,
			}
		}

		rolledRatelimitMap := make(map[string]ratelimitData)
		for _, rl := range rolledKey.Ratelimits {
			rolledRatelimitMap[rl.Name] = ratelimitData{
				Limit:    rl.Limit,
				Duration: rl.Duration,
			}
		}

		require.Equal(t, createdRatelimitMap, rolledRatelimitMap, "ratelimit maps should be equal")
	})

	t.Run("reroll sets TTL on original key when expiration is provided", func(t *testing.T) {
		t.Parallel()

		ttlMs := int64(60000) // 60 seconds

		key := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: workspace.ID,
			Disabled:    false,
			KeySpaceID:  api.KeyAuthID.String,
		}) // nolint:exhaustruct

		req := handler.Request{
			KeyId:      key.KeyID,
			Expiration: ttlMs,
		}

		now := time.Now().UnixMilli()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotEmpty(t, res.Body.Data.KeyId)
		require.NotEmpty(t, res.Body.Data.Key)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify original key has expiration set
		createdKeyRow, err := db.Query.FindLiveKeyByID(ctx, h.DB.RW(), key.KeyID)
		require.NoError(t, err)
		require.True(t, createdKeyRow.Expires.Valid, "original key should have expiration set")

		expMs := createdKeyRow.Expires.Time.UnixMilli()
		// Account for minute alignment in the handler (tolerate up to 60 seconds)
		require.True(t, expMs >= now && expMs <= now+ttlMs+60000,
			"original key expiration should be between now and now+TTL+1min for rounding (got %d, expected between %d and %d)",
			expMs, now, now+ttlMs+60000)

		// Verify rolled key has no expiration
		rolledKeyRow, err := db.Query.FindLiveKeyByID(ctx, h.DB.RW(), res.Body.Data.KeyId)
		require.NoError(t, err)
		require.False(t, rolledKeyRow.Expires.Valid, "rolled key should not have expiration set but its set to %s %t", rolledKeyRow.Expires.Time.String(), rolledKeyRow.Expires.Valid)
	})
}
