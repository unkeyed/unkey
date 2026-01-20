package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/prefixedapikey"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	migrateHandler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_migrate_keys"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestKeyVerificationWithMigration(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	migrateRoute := &migrateHandler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		ApiCache:  h.Caches.LiveApiByID,
	}

	verifyRoute := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Logger:     h.Logger,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(verifyRoute)
	h.Register(migrateRoute)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key", "api.*.create_key")

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("verifies key with migration ID", func(t *testing.T) {
		// Create a migration
		migrationID := uid.New("migration")

		// Insert migration directly to database
		err := db.Query.InsertKeyMigration(ctx, h.DB.RW(), db.InsertKeyMigrationParams{
			ID:          migrationID,
			WorkspaceID: workspace.ID,
			Algorithm:   db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey,
		})
		require.NoError(t, err, "Failed to insert migration")

		resendKey, err := prefixedapikey.GenerateAPIKey(&prefixedapikey.GenerateAPIKeyOptions{
			KeyPrefix: "re",
		})
		require.NoError(t, err)

		migrateReq := migrateHandler.Request{
			ApiId:       api.ID,
			MigrationId: migrationID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash:    resendKey.LongTokenHash,
					Enabled: ptr.P(true),
				},
			},
		}

		migrateRes := testutil.CallRoute[migrateHandler.Request, migrateHandler.Response](h, migrateRoute, headers, migrateReq)
		require.Equal(t, 200, migrateRes.Status, "expected 200, received: %#v", migrateRes)
		require.Len(t, migrateRes.Body.Data.Failed, 0, "No keys should fail migration")
		require.Len(t, migrateRes.Body.Data.Migrated, 1, "One key should be migrated")
		keyID := migrateRes.Body.Data.Migrated[0].KeyId

		req := handler.Request{
			Key:         resendKey.Token,
			MigrationId: ptr.P(migrationID),
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, verifyRoute, headers, req)

		require.Equal(t, 200, res1.Status, "expected 200, received: %#v", res1)
		require.NotNil(t, res1.Body)
		require.Equal(t, openapi.VALID, res1.Body.Data.Code, "Key should be valid but got %s", res1.Body.Data.Code)
		require.True(t, res1.Body.Data.Valid, "Key should be valid but got %t", res1.Body.Data.Valid)

		// Now we should be able to verify the key without the migration ID
		req = handler.Request{
			Key: resendKey.Token,
		}

		res2 := testutil.CallRoute[handler.Request, handler.Response](h, verifyRoute, headers, req)
		require.Equal(t, 200, res2.Status, "expected 200, received: %#v", res2)
		require.NotNil(t, res2.Body)
		require.Equal(t, openapi.VALID, res2.Body.Data.Code, "Key should be valid but got %s", res2.Body.Data.Code)
		require.True(t, res2.Body.Data.Valid, "Key should be valid but got %t", res2.Body.Data.Valid)

		// The migration ID should be removed from the key and the hash updated
		key, err := db.Query.FindKeyByID(ctx, h.DB.RW(), keyID)
		require.NoError(t, err)
		require.False(t, key.PendingMigrationID.Valid)
		require.Empty(t, key.PendingMigrationID.String)
		require.NotEqual(t, resendKey.LongTokenHash, key.Hash, "Hash should be different after migration")
		require.Equal(t, hash.Sha256(resendKey.Token), key.Hash)
		require.Equal(t, resendKey.Token[:7], key.Start, "start should match first 6 chars of raw key after migration")
	})
}
