package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_migrate_keys"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestCreateKeyNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	ctx := t.Context()

	route := &handler.Handler{
		DB:        h.DB,
		Logger:    h.Logger,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
		ApiCache:  h.Caches.LiveApiByID,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	t.Run("nonexistent api", func(t *testing.T) {
		// Use a valid API ID format but one that doesn't exist
		nonexistentApiID := uid.New(uid.APIPrefix)
		req := handler.Request{
			ApiId: nonexistentApiID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash: uid.New(""),
				},
			},
			MigrationId: uid.New(""),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted.")
	})

	t.Run("api with valid format but invalid id", func(t *testing.T) {
		// Create a syntactically valid but non-existent API ID
		fakeApiID := "api_1234567890abcdef"
		req := handler.Request{
			ApiId:       fakeApiID,
			MigrationId: "unkeyed",
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash: uid.New(""),
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted.")
	})

	t.Run("api from different workspace", func(t *testing.T) {
		// Create a different workspace to test cross-workspace isolation
		otherWorkspace := h.CreateWorkspace()

		// Create root key for the other workspace with proper permissions
		otherRootKey := h.CreateRootKey(otherWorkspace.ID, "api.*.create_key")

		otherApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			IpWhitelist:   "",
			EncryptedKeys: false,
			Name:          nil,
			CreatedAt:     nil,
			DefaultPrefix: nil,
			DefaultBytes:  nil,
		})

		req := handler.Request{
			ApiId:       otherApi.ID,
			MigrationId: "unkeyed",
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash: uid.New(""),
				},
			},
		}

		otherHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", otherRootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, otherHeaders, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted.")
	})

	t.Run("api with minimum valid length but nonexistent", func(t *testing.T) {
		// Test with minimum valid API ID length (3 chars as per validation)
		minimalApiID := "api"
		req := handler.Request{
			ApiId: minimalApiID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash: uid.New(""),
				},
			},
			MigrationId: uid.New(""),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted.")
	})

	t.Run("deleted api", func(t *testing.T) {
		deletedApi := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID:   h.Resources().UserWorkspace.ID,
			IpWhitelist:   "",
			EncryptedKeys: false,
			Name:          nil,
			CreatedAt:     nil,
			DefaultPrefix: nil,
			DefaultBytes:  nil,
		})

		db.Query.SoftDeleteApi(ctx, h.DB.RW(), db.SoftDeleteApiParams{
			Now:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ApiID: deletedApi.ID,
		})

		req := handler.Request{
			ApiId: deletedApi.ID,
			Keys: []openapi.V2KeysMigrateKeyData{{
				Hash: uid.New(""),
			}},
			MigrationId: uid.New(""),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "The requested API does not exist or has been deleted.")
	})

	t.Run("migration doesn't exist", func(t *testing.T) {
		req := handler.Request{
			ApiId:       api.ID,
			Keys:        []openapi.V2KeysMigrateKeyData{{Hash: uid.New("")}},
			MigrationId: uid.New("some_migration_id"),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "The requested Migration does not exist or has been deleted.")
	})
}
