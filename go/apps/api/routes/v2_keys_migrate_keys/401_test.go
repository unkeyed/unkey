package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_migrate_keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestMigrateKeysUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		ApiCache:  h.Caches.LiveApiByID,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	migrationID := uid.New("")
	err := db.Query.InsertKeyMigration(ctx, h.DB.RW(), db.InsertKeyMigrationParams{
		ID:          migrationID,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Algorithm:   db.KeyMigrationsAlgorithmGithubcomSeamapiPrefixedApiKey,
	})
	require.NoError(t, err)

	// Basic request body
	req := handler.Request{
		ApiId:       api.ID,
		MigrationId: migrationID,
		Keys: []openapi.V2KeysMigrateKeyData{
			{
				Hash: uid.New(""),
			},
		},
	}

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_key_12345"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("nonexistent key", func(t *testing.T) {
		nonexistentKey := uid.New(uid.KeyPrefix)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", nonexistentKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("bearer with extra spaces", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer   invalid_key_with_spaces   "},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})
}
