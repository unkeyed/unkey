package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_migrate_keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/prefixedapikey"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestCreateKeySuccess(t *testing.T) {
	t.Parallel()

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

	err := db.Query.InsertKeyMigration(ctx, h.DB.RW(), db.InsertKeyMigrationParams{
		ID:          "unkeyed",
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Algorithm:   db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey,
	})
	require.NoError(t, err)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	otherApiKey, err := prefixedapikey.GenerateAPIKey(&prefixedapikey.GenerateAPIKeyOptions{
		KeyPrefix: "unkeyed",
	})

	// Test basic migration
	req := handler.Request{
		ApiId:       api.ID,
		MigrationId: "unkeyed",
		Keys: []openapi.V2KeysMigrateKeyData{
			{
				Hash: otherApiKey.LongTokenHash,
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
						Duration:  (time.Minute * 60).Milliseconds(),
						Limit:     100,
						Name:      "default",
					},
				},
				Roles: ptr.P([]string{"admin"}),
			},
		},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, 200, res.Status)
	require.NotNil(t, res.Body)

	require.Empty(t, res.Body.Data.Failed)
	require.NotEmpty(t, res.Body.Data.Migrated)
	require.Equal(t, res.Body.Data.Migrated[0].Hash, otherApiKey.LongTokenHash)
	require.NotEmpty(t, res.Body.Data.Migrated[0].Hash)

	// Verify key was created in database
	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), res.Body.Data.Migrated[0].KeyId)
	require.NoError(t, err)

	keydata := db.ToKeyData(key)

	require.Equal(t, res.Body.Data.Migrated[0].KeyId, key.ID)
	require.Equal(t, otherApiKey.LongTokenHash, key.Hash)
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
}

// func TestCreateKeyWithOptionalFields(t *testing.T) {
// 	t.Parallel()

// 	h := testutil.NewHarness(t)
// 	ctx := context.Background()

// 	route := &handler.Handler{
// 		DB:        h.DB,
// 		Keys:      h.Keys,
// 		Logger:    h.Logger,
// 		Auditlogs: h.Auditlogs,
// 		Vault:     h.Vault,
// 	}

// 	h.Register(route)

// 	// Create API using testutil helper
// 	api := h.CreateApi(seed.CreateApiRequest{
// 		WorkspaceID: h.Resources().UserWorkspace.ID,
// 	})

// 	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

// 	headers := http.Header{
// 		"Content-Type":  {"application/json"},
// 		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
// 	}

// 	// Test key creation with optional fields
// 	name := "Test Key"
// 	prefix := "test"
// 	externalID := "user_123"
// 	byteLength := 24
// 	expires := int64(1704067200000) // Jan 1, 2024
// 	enabled := true

// 	req := handler.Request{
// 		ApiId:      api.ID,
// 		Name:       &name,
// 		Prefix:     &prefix,
// 		ExternalId: &externalID,
// 		ByteLength: &byteLength,
// 		Expires:    &expires,
// 		Enabled:    &enabled,
// 	}

// 	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
// 	require.Equal(t, 200, res.Status)
// 	require.NotNil(t, res.Body)

// 	require.NotEmpty(t, res.Body.Data.KeyId)
// 	require.NotEmpty(t, res.Body.Data.Key)
// 	require.Contains(t, res.Body.Data.Key, prefix+"_")

// 	// Verify key fields in database
// 	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), res.Body.Data.KeyId)
// 	require.NoError(t, err)

// 	require.True(t, key.Name.Valid)
// 	require.Equal(t, name, key.Name.String)
// 	require.True(t, key.Enabled)
// }
