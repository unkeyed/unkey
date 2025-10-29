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
			ID:          "test-key-id",
			Hash:        "test-hash",
			WorkspaceID: "test-workspace",
			Enabled:     true,
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.Equal(t, "test-key-id", result.Key.ID)
		require.Equal(t, "test-hash", result.Key.Hash)
		require.Equal(t, "test-workspace", result.Key.WorkspaceID)
		require.True(t, result.Key.Enabled)
	})

	t.Run("FindLiveKeyByIDRow pointer", func(t *testing.T) {
		row := FindLiveKeyByIDRow{
			ID:          "test-key-id-ptr",
			Hash:        "test-hash-ptr",
			WorkspaceID: "test-workspace-ptr",
			Enabled:     false,
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.Equal(t, "test-key-id-ptr", result.Key.ID)
		require.Equal(t, "test-hash-ptr", result.Key.Hash)
		require.Equal(t, "test-workspace-ptr", result.Key.WorkspaceID)
		require.False(t, result.Key.Enabled)
	})

	t.Run("FindLiveKeyByHashRow value", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			ID:          "hash-key-id",
			Hash:        "hash-test",
			WorkspaceID: "hash-workspace",
			Enabled:     true,
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.Equal(t, "hash-key-id", result.Key.ID)
		require.Equal(t, "hash-test", result.Key.Hash)
		require.Equal(t, "hash-workspace", result.Key.WorkspaceID)
		require.True(t, result.Key.Enabled)
	})

	t.Run("FindLiveKeyByHashRow pointer", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			ID:          "hash-key-ptr",
			Hash:        "hash-ptr",
			WorkspaceID: "hash-workspace-ptr",
			Enabled:     false,
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.Equal(t, "hash-key-ptr", result.Key.ID)
		require.Equal(t, "hash-ptr", result.Key.Hash)
		require.Equal(t, "hash-workspace-ptr", result.Key.WorkspaceID)
		require.False(t, result.Key.Enabled)
	})
}

func TestToKeyData_EmptyValues(t *testing.T) {
	t.Run("zero value FindLiveKeyByIDRow", func(t *testing.T) {
		row := FindLiveKeyByIDRow{} // All zero values

		result := ToKeyData(row, nil)

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

		result := ToKeyData(row, nil)

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
		row := FindLiveKeyByHashRow{
			ID:                 "key-with-identity",
			WorkspaceID:        "workspace-123",
			IdentityTableID:    sql.NullString{String: "identity-123", Valid: true},
			IdentityExternalID: sql.NullString{String: "user-456", Valid: true},
			IdentityMeta:       []byte(`{"role": "admin"}`),
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.NotNil(t, result.Identity)
		require.Equal(t, "identity-123", result.Identity.ID)
		require.Equal(t, "user-456", result.Identity.ExternalID)
		require.Equal(t, "workspace-123", result.Identity.WorkspaceID)
		require.Equal(t, []byte(`{"role": "admin"}`), result.Identity.Meta)
	})

	t.Run("without identity data", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			ID:              "key-no-identity",
			WorkspaceID:     "workspace-123",
			IdentityTableID: sql.NullString{Valid: false}, // No identity
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.Nil(t, result.Identity)
	})
}

func TestToKeyData_WithJSONFields(t *testing.T) {
	t.Run("with valid JSON arrays", func(t *testing.T) {
		roles := []RoleInfo{{Name: "admin"}, {Name: "user"}}
		rolesJSON, _ := json.Marshal(roles)

		permissions := []PermissionInfo{{Slug: "read"}, {Slug: "write"}}
		permissionsJSON, _ := json.Marshal(permissions)

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
		ratelimitsJSON, _ := json.Marshal(ratelimits)

		row := FindLiveKeyByHashRow{
			ID:              "key-with-json",
			Roles:           rolesJSON,
			Permissions:     permissionsJSON,
			RolePermissions: permissionsJSON,
			Ratelimits:      ratelimitsJSON,
		}

		result := ToKeyData(row, nil)

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
		require.Equal(t, int64(3600), result.Ratelimits[0].Duration)
		require.Equal(t, int32(100), result.Ratelimits[0].Limit)
		require.Equal(t, "hourly-limit", result.Ratelimits[0].Name)
		require.True(t, result.Ratelimits[0].AutoApply)
	})

	t.Run("with invalid JSON - should ignore errors", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			ID:          "key-bad-json",
			Roles:       []byte(`{invalid json}`),      // Bad JSON
			Permissions: []byte(`not json at all`),     // Bad JSON
			Ratelimits:  []byte(`{"incomplete": true`), // Bad JSON
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		// Should default to empty arrays when JSON unmarshaling fails
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
		require.Empty(t, result.Ratelimits)
	})

	t.Run("with nil JSON fields", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			ID:              "key-nil-json",
			Roles:           nil,
			Permissions:     nil,
			RolePermissions: nil,
			Ratelimits:      nil,
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
		require.Empty(t, result.Ratelimits)
	})

	t.Run("with non-byte slice fields", func(t *testing.T) {
		row := FindLiveKeyByHashRow{
			ID:              "key-wrong-type",
			Roles:           "not a byte slice", // Wrong type
			Permissions:     123,                // Wrong type
			RolePermissions: struct{}{},         // Wrong type
		}

		result := ToKeyData(row, nil)

		require.NotNil(t, result)
		// Should default to empty arrays when type assertion fails
		require.Empty(t, result.Roles)
		require.Empty(t, result.Permissions)
		require.Empty(t, result.RolePermissions)
	})
}
