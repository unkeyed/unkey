package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_migrate_keys"
)

func TestMigrateKeysBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		ApiCache:  h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create API using testutil helper
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing everything", func(t *testing.T) {
		req := handler.Request{
			// Missing everything
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty apiId", func(t *testing.T) {
		req := handler.Request{
			ApiId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty migrationId", func(t *testing.T) {
		req := handler.Request{
			ApiId:       api.ID,
			MigrationId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("no keys", func(t *testing.T) {
		req := handler.Request{
			ApiId:       api.ID,
			MigrationId: uid.New(""),
			Keys:        []openapi.V2KeysMigrateKeyData{},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty hash", func(t *testing.T) {
		req := handler.Request{
			ApiId:       api.ID,
			MigrationId: uid.New(""),
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash: "",
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("negative expires timestamp", func(t *testing.T) {
		invalidExpires := int64(-1)
		req := handler.Request{
			ApiId: api.ID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash:    uid.New("prefix Prefix"),
					Expires: &invalidExpires,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty permission in list", func(t *testing.T) {
		emptyPermissions := []string{""}
		req := handler.Request{
			ApiId: api.ID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash:        uid.New("prefix Prefix"),
					Permissions: &emptyPermissions,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty role in list", func(t *testing.T) {
		emptyRoles := []string{""}
		req := handler.Request{
			ApiId: api.ID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash:  uid.New("prefix Prefix"),
					Roles: &emptyRoles,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("permission too long", func(t *testing.T) {
		// Create a permission string that's longer than 512 characters
		longPermission := strings.Repeat("a", 513)
		longPermissions := []string{longPermission}
		req := handler.Request{
			ApiId: api.ID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash:        uid.New("prefix Prefix"),
					Permissions: &longPermissions,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("role too long", func(t *testing.T) {
		// Create a role string that's longer than 512 characters
		req := handler.Request{
			ApiId: api.ID,
			Keys: []openapi.V2KeysMigrateKeyData{
				{
					Hash:  uid.New("prefix Prefix"),
					Roles: ptr.P([]string{strings.Repeat("a", 513)}),
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}
