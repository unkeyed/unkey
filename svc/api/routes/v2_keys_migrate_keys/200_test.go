package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/prefixedapikey"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_migrate_keys"
)

func TestMigrateKeysSuccess(t *testing.T) {

	h := testutil.NewHarness(t)
	ctx := context.Background()

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		ApiCache:  h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create API using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	migrationID := uid.New(uid.TestPrefix)
	err := db.Query.InsertKeyMigration(ctx, h.DB.RW(), db.InsertKeyMigrationParams{
		ID:          migrationID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Algorithm:   db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	generatedKey, err := prefixedapikey.GenerateAPIKey(&prefixedapikey.GenerateAPIKeyOptions{
		KeyPrefix: "unkeyed",
	})
	require.NoError(t, err)

	keyToMigrate := openapi.V2KeysMigrateKeyData{
		Hash: generatedKey.LongTokenHash,
		Credits: &openapi.KeyCreditsData{
			Remaining: nullable.Nullable[int64]{},
		},
		Enabled:    ptr.P(false),
		Expires:    nil,
		ExternalId: ptr.P("ext_123"),
		Meta: ptr.P(map[string]interface{}{
			"key": "value",
		}),
		Name:        ptr.P("Migration-Key"),
		Permissions: ptr.P([]string{"test"}),
		Ratelimits: &[]openapi.RatelimitRequest{
			{
				AutoApply: true,
				Duration:  time.Hour.Milliseconds(),
				Limit:     100,
				Name:      "default",
			},
		},
		Roles: ptr.P([]string{"admin"}),
	}

	t.Run("basic migration", func(t *testing.T) {
		req := handler.Request{
			ApiId:       api.ID,
			MigrationId: migrationID,
			Keys:        []openapi.V2KeysMigrateKeyData{keyToMigrate},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Empty(t, res.Body.Data.Failed)
		require.NotEmpty(t, res.Body.Data.Migrated)
		require.Equal(t, res.Body.Data.Migrated[0].Hash, generatedKey.LongTokenHash)

		// Verify key was created in database
		key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), res.Body.Data.Migrated[0].KeyId)
		require.NoError(t, err)

		keydata := db.ToKeyData(key)

		require.Equal(t, res.Body.Data.Migrated[0].KeyId, key.ID)
		require.Equal(t, generatedKey.LongTokenHash, key.Hash)
		require.Empty(t, keydata.Key.Start)
		require.False(t, keydata.Key.Enabled)
		require.NotNil(t, keydata.Identity)
		require.NotEmpty(t, keydata.Identity.ID)
		require.NotEmpty(t, keydata.Key.Name.String)
		require.NotEmpty(t, keydata.Key.Meta.String)
		require.Len(t, keydata.Permissions, 1)
		require.Len(t, keydata.Roles, 1)
		require.Len(t, keydata.RolePermissions, 0)
		require.Len(t, keydata.Ratelimits, 1)
	})

	t.Run("Finds the correct ids and doesn't double insert", func(t *testing.T) {
		// Generate a new key hash for this test
		otherGeneratedKey, err := prefixedapikey.GenerateAPIKey(&prefixedapikey.GenerateAPIKeyOptions{
			KeyPrefix: "unkeyed",
		})
		require.NoError(t, err)

		// Create a new key with the same identity, permissions, and roles
		keyToMigrate2 := openapi.V2KeysMigrateKeyData{
			Hash:        otherGeneratedKey.LongTokenHash,
			Credits:     keyToMigrate.Credits,
			Enabled:     keyToMigrate.Enabled,
			Expires:     keyToMigrate.Expires,
			ExternalId:  keyToMigrate.ExternalId, // Same external ID
			Meta:        keyToMigrate.Meta,
			Name:        ptr.P("Migration-Key-2"),
			Permissions: keyToMigrate.Permissions, // Same permissions
			Ratelimits:  keyToMigrate.Ratelimits,
			Roles:       keyToMigrate.Roles, // Same roles
		}

		req := handler.Request{
			ApiId:       api.ID,
			MigrationId: migrationID,
			Keys:        []openapi.V2KeysMigrateKeyData{keyToMigrate2},
		}

		// First, verify the identity, permission, and role exist from the first migration
		identity, err := db.Query.FindIdentitiesByExternalId(ctx, h.DB.RO(), db.FindIdentitiesByExternalIdParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalIds: []string{"ext_123"},
		})
		require.NoError(t, err, "Identity should exist from first migration")
		require.Len(t, identity, 1, "Identity should exist from first migration")

		permissions, err := db.Query.FindPermissionsBySlugs(ctx, h.DB.RO(), db.FindPermissionsBySlugsParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Slugs:       []string{"test"},
		})
		require.NoError(t, err)
		require.Len(t, permissions, 1, "Permission should exist from first migration")

		roles, err := db.Query.FindRolesByNames(ctx, h.DB.RO(), db.FindRolesByNamesParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Names:       []string{"admin"},
		})
		require.NoError(t, err)
		require.Len(t, roles, 1, "Role should exist from first migration")

		// Perform the second migration
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.Empty(t, res.Body.Data.Failed)
		require.NotEmpty(t, res.Body.Data.Migrated)
		require.Equal(t, res.Body.Data.Migrated[0].Hash, otherGeneratedKey.LongTokenHash)

		// Verify the new key was created with the same identity
		key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), res.Body.Data.Migrated[0].KeyId)
		require.NoError(t, err)
		keydata := db.ToKeyData(key)

		require.NotNil(t, keydata.Identity)
		require.Equal(t, identity[0].ID, keydata.Identity.ID, "Should reuse existing identity")
		require.Equal(t, "ext_123", keydata.Identity.ExternalID)

		// Verify no duplicate identities were created
		identities, err := db.Query.FindIdentitiesByExternalId(ctx, h.DB.RO(), db.FindIdentitiesByExternalIdParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalIds: []string{"ext_123"},
			Deleted:     false,
		})
		require.NoError(t, err)
		require.Len(t, identities, 1, "Should not create duplicate identities")

		// Verify no duplicate permissions were created
		allPermissions, err := db.Query.FindPermissionsBySlugs(ctx, h.DB.RO(), db.FindPermissionsBySlugsParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Slugs:       []string{"test"},
		})
		require.NoError(t, err)
		require.Len(t, allPermissions, 1, "Should not create duplicate permissions")

		// Verify no duplicate roles were created
		allRoles, err := db.Query.FindRolesByNames(ctx, h.DB.RO(), db.FindRolesByNamesParams{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Names:       []string{"admin"},
		})
		require.NoError(t, err)
		require.Len(t, allRoles, 1, "Should not create duplicate roles")

		// Verify the key has the correct permission and role associations
		require.Len(t, keydata.Permissions, 1, "Key should have one permission")
		require.Equal(t, permissions[0].ID, keydata.Permissions[0].ID)
		require.Len(t, keydata.Roles, 1, "Key should have one role")
		require.Equal(t, roles[0].ID, keydata.Roles[0].ID)
	})

	t.Run("Fail duplicate hashes", func(t *testing.T) {
		req := handler.Request{
			ApiId:       api.ID,
			MigrationId: migrationID,
			Keys:        []openapi.V2KeysMigrateKeyData{keyToMigrate},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.Failed)
		require.Contains(t, res.Body.Data.Failed, keyToMigrate.Hash, "Hash has to be in failed array")
		require.Empty(t, res.Body.Data.Migrated)
	})
}
