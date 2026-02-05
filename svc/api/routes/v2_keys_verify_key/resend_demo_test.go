package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/prefixedapikey"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_verify_key"
)

func TestResendDemo(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		Keys:       h.Keys,
		Auditlogs:  h.Auditlogs,
		ClickHouse: h.ClickHouse,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.verify_key")

	api := h.CreateApi(seed.CreateApiRequest{WorkspaceID: workspace.ID})

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("verifies key with migration ID", func(t *testing.T) {

		// 1. Create a migration
		// This will be done by us, no need to think about it.

		// Insert migration directly to database
		migrationID := uid.New(uid.TestPrefix)
		err := db.Query.InsertKeyMigration(ctx, h.DB.RW(), db.InsertKeyMigrationParams{
			ID:          migrationID,
			WorkspaceID: workspace.ID,
			Algorithm:   db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey,
		})
		require.NoError(t, err, "Failed to insert migration")

		// 2. Get an existing key.
		//
		// In the future you'd use unkey to issue new keys, but for your existing ones,
		// we'll create one using your library.
		//
		//
		// ```js
		// import { generateAPIKey } from "prefixed-api-key"
		//
		// const key = await generateAPIKey({ keyPrefix: 'resend' })
		//
		// console.log(key)
		// /*
		// {
		//   shortToken: "2aGwhSYz",
		//   longToken: "GEbTboUygK1ixefLDTUM5wf7",
		//   longTokenHash: "c4fbfe7c69a067cb0841dea343346a750a69908a08ea9656d2a8c19fb0823c64",
		//   token: "resend_2aGwhSYz_GEbTboUygK1ixefLDTUM5wf7",
		// }
		// */
		// ```

		// When migrating keys to unkey, you just need to give us the longTokenHash
		// and optional user id etc to link them together so you can later query all
		// keys for a specific user.
		// longTokenHash := "f8d7af831e76a886cb225e56d0750a54efab6f89c036e01b2ca1f52203425c72"

		// Unkey doesn't store this token, we just use it below to run a demo
		// verification.
		// token := "re_QgLu9m3D_FMbosT9oDBP3D8RkTu6p24wT"
		resendKey, err := prefixedapikey.GenerateAPIKey(&prefixedapikey.GenerateAPIKeyOptions{
			KeyPrefix: "re",
		})
		require.NoError(t, err)

		// 3. Migrate existing keys to unkey
		//
		// We'll give you an api endpoint to send your existing hashes to.

		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:                 uid.New(uid.KeyPrefix),
			KeySpaceID:         api.KeyAuthID.String,
			WorkspaceID:        workspace.ID,
			CreatedAtM:         time.Now().UnixMilli(),
			Hash:               resendKey.LongTokenHash,
			Enabled:            true,
			PendingMigrationID: sql.NullString{Valid: true, String: migrationID},
		})
		require.NoError(t, err)

		// 4. Now we're ready to verify keys.
		// You'll grab the key from the request against your api and then make a call to unkey
		//
		// You need to send the key and the preshared constant migration ID,

		res1 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Key:         resendKey.Token,
			MigrationId: ptr.P(migrationID),
		})

		require.Equal(t, 200, res1.Status)
		require.True(t, res1.Body.Data.Valid)

		// During the first verification, we look up the key using the algorithm from
		// your library and then rehash it to use unkey's default algorithm.
		// Now this key is fully migrated and just like any other unkey key.
		// Sending the migration ID along for this key is no longer necessary, but doesn't hurt either.
		// Since you don't know before hand if the key is migrated or not, you can always send the migration ID along with the key and we will handle it accordingly.
		res2 := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Key: resendKey.Token,
		})

		require.Equal(t, 200, res2.Status)
		require.True(t, res2.Body.Data.Valid)
	})
}
