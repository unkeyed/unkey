package db

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToKeyData_ValidCases(t *testing.T) {
	t.Run("FindLiveKeyByIDRow value", func(t *testing.T) {
		row := FindLiveKeyByIDRow{
			KeyID:                     "test-key-id",
			KeyHash:                   "test-key-hash",
			KeyPrefix:                 "prod_sk",
			KeyStart:                  "abcd",
			KeyEnd:                    "wxyz",
			KeyWorkspaceID:            "test-workspace",
			KeyForWorkspaceID:         sql.NullString{String: "root-workspace", Valid: true},
			KeyEnabled:                true,
			ApiID:                     "api-id",
			ApiName:                   "api-name",
			KeyAuthID:                 "key-auth-id",
			KeyAuthStoreEncryptedKeys: true,
			KeyAuthDefaultPrefix:      sql.NullString{String: "prefix", Valid: true},
			KeyAuthDefaultBytes:       sql.NullInt32{Int32: 16, Valid: true},
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Equal(t, "test-key-id", result.Key.ID)
		require.Equal(t, "test-key-hash", result.Key.Hash)
		require.Equal(t, "prod_sk", result.Key.Prefix)
		require.Equal(t, "abcd", result.Key.Start)
		require.Equal(t, "wxyz", result.Key.End)
		require.Equal(t, "test-workspace", result.Key.WorkspaceID)
		require.Equal(t, "root-workspace", result.Key.ForWorkspaceID.String)
		require.True(t, result.Key.ForWorkspaceID.Valid)
		require.True(t, result.Key.Enabled)
		require.Equal(t, "api-id", result.Api.ID)
		require.Equal(t, "api-name", result.Api.Name)
		require.Equal(t, "key-auth-id", result.KeyAuth.ID)
		require.True(t, result.KeyAuth.StoreEncryptedKeys)
		require.Equal(t, "prefix", result.KeyAuth.DefaultPrefix.String)
		require.Equal(t, int32(16), result.KeyAuth.DefaultBytes.Int32)
		require.Empty(t, result.Workspace)
	})

	t.Run("FindLiveKeyByIDRow pointer", func(t *testing.T) {
		row := FindLiveKeyByIDRow{
			KeyID:          "test-key-id-ptr",
			KeyWorkspaceID: "test-workspace-ptr",
			KeyEnabled:     false,
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Equal(t, "test-key-id-ptr", result.Key.ID)
		require.Empty(t, result.Key.Hash)
		require.Equal(t, "test-workspace-ptr", result.Key.WorkspaceID)
		require.False(t, result.Key.Enabled)
	})

	t.Run("FindLiveKeyByHashRow value", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			KeyID:                     "hash-key-id",
			KeyPrefix:                 "prod_sk",
			KeyStart:                  "abcd",
			KeyEnd:                    "wxyz",
			KeyWorkspaceID:            "hash-workspace",
			KeyEnabled:                true,
			ApiID:                     "hash-api-id",
			KeyAuthStoreEncryptedKeys: true,
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Equal(t, "hash-key-id", result.Key.ID)
		require.Empty(t, result.Key.Hash)
		require.Equal(t, "prod_sk", result.Key.Prefix)
		require.Equal(t, "abcd", result.Key.Start)
		require.Equal(t, "wxyz", result.Key.End)
		require.Equal(t, "hash-workspace", result.Key.WorkspaceID)
		require.True(t, result.Key.Enabled)
		require.Equal(t, "hash-api-id", result.Api.ID)
		require.Empty(t, result.Api.Name)
		require.True(t, result.KeyAuth.StoreEncryptedKeys)
		require.Empty(t, result.KeyAuth.ID)
		require.Empty(t, result.Workspace)
	})

	t.Run("FindLiveKeyByHashRow pointer", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			KeyID:          "hash-key-ptr",
			KeyWorkspaceID: "hash-workspace-ptr",
			KeyEnabled:     false,
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Equal(t, "hash-key-ptr", result.Key.ID)
		require.Empty(t, result.Key.Hash)
		require.Equal(t, "hash-workspace-ptr", result.Key.WorkspaceID)
		require.False(t, result.Key.Enabled)
	})
}

func TestToKeyData_EmptyValues(t *testing.T) {
	t.Run("zero value FindLiveKeyByIDRow", func(t *testing.T) {
		row := FindLiveKeyByIDRow{} // All zero values

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Equal(t, "", result.Key.ID)
		require.Equal(t, "", result.Key.Hash)
		require.Equal(t, "", result.Key.WorkspaceID)
		require.False(t, result.Key.Enabled) // bool zero value
		require.Nil(t, result.Identity)      // No identity data
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
		require.Empty(t, result.Ratelimits)
	})

	t.Run("zero value FindLiveKeyByHashRow", func(t *testing.T) {
		row := FindLiveKeyByHashRow{} // All zero values

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Equal(t, "", result.Key.ID)
		require.Equal(t, "", result.Key.Hash)
		require.Equal(t, "", result.Key.WorkspaceID)
		require.False(t, result.Key.Enabled)
		require.Nil(t, result.Identity)
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
		require.Empty(t, result.Ratelimits)
	})
}

func TestToKeyData_WithIdentity(t *testing.T) {
	t.Run("with valid identity data", func(t *testing.T) {
		meta, err := json.Marshal(map[string]string{"role": "admin"})
		require.NoError(t, err)
		row := FindLiveKeyByHashRow{
			KeyID:              "key-with-identity",
			KeyWorkspaceID:     "workspace-123",
			IdentityTableID:    sql.NullString{String: "identity-123", Valid: true},
			IdentityExternalID: sql.NullString{String: "user-456", Valid: true},
			IdentityMeta:       meta,
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.NotNil(t, result.Identity)
		require.Equal(t, "identity-123", result.Identity.ID)
		require.Equal(t, "user-456", result.Identity.ExternalID)
		require.Equal(t, "workspace-123", result.Identity.WorkspaceID)
		require.Equal(t, meta, result.Identity.Meta)
	})

	t.Run("without identity data", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			KeyID:           "key-no-identity",
			KeyWorkspaceID:  "workspace-123",
			IdentityTableID: sql.NullString{Valid: false}, // No identity
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Nil(t, result.Identity)
	})
}

func TestToKeyData_WithJSONFields(t *testing.T) {
	t.Run("with valid JSON arrays", func(t *testing.T) {
		roles := []RoleInfo{{Name: "admin"}, {Name: "user"}}
		rolesJSON, err := json.Marshal(roles)
		require.NoError(t, err)

		permissions := []PermissionInfo{{Slug: "read"}, {Slug: "write"}}
		permissionsJSON, err := json.Marshal(permissions)
		require.NoError(t, err)

		ratelimits := []RatelimitInfo{
			{
				ID:        "rate-1",
				Duration:  3600,
				Limit:     100,
				Name:      "hourly-limit",
				AutoApply: true,
			},
			{
				ID:        "rate-2",
				Duration:  60,
				Limit:     10,
				Name:      "minute-limit",
				AutoApply: false,
			},
		}
		ratelimitsJSON, err := json.Marshal(ratelimits)
		require.NoError(t, err)

		row := FindLiveKeyByHashRow{
			KeyID:           "key-with-json",
			Roles:           rolesJSON,
			Permissions:     permissionsJSON,
			RolePermissions: permissionsJSON,
			Ratelimits:      ratelimitsJSON,
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Len(t, result.Roles, 2)
		require.Equal(t, "admin", result.Roles[0].Name)
		require.Equal(t, "user", result.Roles[1].Name)
		require.Len(t, result.Permissions, 2)
		require.Equal(t, "read", result.Permissions[0].Slug)
		require.Equal(t, "write", result.Permissions[1].Slug)
		require.Len(t, result.RolePermissions, 2)
		require.Len(t, result.Ratelimits, 2)
		require.Equal(t, "rate-1", result.Ratelimits[0].ID)
		require.Equal(t, uint64(3600), result.Ratelimits[0].Duration)
		require.Equal(t, uint64(100), result.Ratelimits[0].Limit)
		require.Equal(t, "hourly-limit", result.Ratelimits[0].Name)
		require.True(t, result.Ratelimits[0].AutoApply)
	})

	t.Run("with invalid JSON - should ignore errors", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			KeyID:       "key-bad-json",
			Roles:       []byte(`{invalid json}`),      // Bad JSON
			Permissions: []byte(`not json at all`),     // Bad JSON
			Ratelimits:  []byte(`{"incomplete": true`), // Bad JSON
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		// Should default to empty arrays when JSON unmarshaling fails
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
		require.Empty(t, result.Ratelimits)
	})

	t.Run("with nil JSON fields", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			KeyID:           "key-nil-json",
			Roles:           nil,
			Permissions:     nil,
			RolePermissions: nil,
			Ratelimits:      nil,
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
		require.Empty(t, result.Ratelimits)
	})

	t.Run("with non-byte slice fields", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			KeyID:           "key-wrong-type",
			Roles:           "not a byte slice", // Wrong type
			Permissions:     123,                // Wrong type
			RolePermissions: struct{}{},         // Wrong type
		}

		result := ToKeyData(row)

		require.NotNil(t, result)
		// Should default to empty arrays when type assertion fails
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
	})
}
