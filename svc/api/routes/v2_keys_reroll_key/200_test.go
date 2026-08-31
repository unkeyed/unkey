package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

func TestRerollKeySuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
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

	identityMeta, err := json.Marshal(map[string]string{"name": "Test User"})
	require.NoError(t, err)
	identity := h.CreateIdentity(seed.CreateIdentityRequest{
		WorkspaceID: workspace.ID,
		ExternalID:  "test_123",
		Meta:        identityMeta,
		Ratelimits: []seed.CreateRatelimitRequest{
			{
				Name:        "default-enterprise",
				WorkspaceID: workspace.ID,
				AutoApply:   true,
				Duration:    uint64(time.Minute.Milliseconds()),
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
			Remaining:    ptr.P(int64(16)),
			IdentityID:   ptr.P(identity.ID),
			Meta:         nil,
			Expires:      nil,
			Name:         ptr.P("Test-Key"),
			Deleted:      false,
			Recoverable:  true,
			RefillAmount: ptr.P(int64(100)),
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
					Duration:    uint64(time.Minute.Milliseconds()),
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

		require.NotEqual(t, createdKeyRow.KeyID, rolledKeyRow.KeyID)
		require.Equal(t, createdKeyRow.KeyName.String, rolledKeyRow.KeyName.String)
		require.Equal(t, createdKeyRow.KeyIdentityID.String, rolledKeyRow.KeyIdentityID.String)
		require.Equal(t, createdKeyRow.KeyMeta, rolledKeyRow.KeyMeta)
		require.Equal(t, createdKeyRow.KeyRefillDay.Int16, rolledKeyRow.KeyRefillDay.Int16)
		require.Equal(t, createdKeyRow.KeyRefillAmount.Int64, rolledKeyRow.KeyRefillAmount.Int64)
		require.Equal(t, createdKeyRow.KeyRemainingRequests.Int64, rolledKeyRow.KeyRemainingRequests.Int64)

		// The first key should expire
		require.True(t, createdKeyRow.KeyExpires.Valid)
		require.True(t, createdKeyRow.EncryptedKey.Valid)
		require.True(t, createdKeyRow.EncryptionKeyID.Valid)

		require.False(t, rolledKeyRow.KeyExpires.Valid)
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
			Limit    uint64
			Duration uint64
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
		require.True(t, createdKeyRow.KeyExpires.Valid, "original key should have expiration set")

		expMs := createdKeyRow.KeyExpires.Time.UnixMilli()
		// Account for minute alignment in the handler (tolerate up to 60 seconds)
		require.True(t, expMs >= now && expMs <= now+ttlMs+60000,
			"original key expiration should be between now and now+TTL+1min for rounding (got %d, expected between %d and %d)",
			expMs, now, now+ttlMs+60000)

		// Verify rolled key has no expiration
		rolledKeyRow, err := db.Query.FindLiveKeyByID(ctx, h.DB.RW(), res.Body.Data.KeyId)
		require.NoError(t, err)
		require.False(t, rolledKeyRow.KeyExpires.Valid, "rolled key should not have expiration set but its set to %s %t", rolledKeyRow.KeyExpires.Time.String(), rolledKeyRow.KeyExpires.Valid)
	})
}

func TestRerollKeyWithURNPermission(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	workspace := h.Resources().UserWorkspace
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: workspace.ID,
	})
	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: workspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	createKeyPermission := fmt.Sprintf("unkey:v1:%s:keyspaces/%s#create_key", workspace.ID, api.KeyAuthID.String)
	rootKey := h.CreateRootKey(workspace.ID, createKeyPermission)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		KeyId:      key.KeyID,
		Expiration: 0,
	})
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)
	require.NotEmpty(t, res.Body.Data.KeyId)
	require.NotEmpty(t, res.Body.Data.Key)
}
