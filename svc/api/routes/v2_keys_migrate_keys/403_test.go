package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_migrate_keys"
)

func TestMigrateKeysForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		ApiCache:  h.Caches.LiveApiByID,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		EncryptedKeys: true,
	})

	otherAPI := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		EncryptedKeys: true,
	})

	req := handler.Request{
		ApiId: api.ID,
		Keys: []openapi.V2KeysMigrateKeyData{
			{
				Hash: uid.New(""),
			},
		},
		MigrationId: uid.New(""),
	}

	t.Run("no permissions", func(t *testing.T) {
		// Create root key with no permissions
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("wrong permission - has read but not create", func(t *testing.T) {
		// Create root key with read permission instead of create
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("permission for different API", func(t *testing.T) {
		// Create root key with create permission for other API
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.create_key", otherAPI.ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("permission for specific API but requesting different API", func(t *testing.T) {
		// Create root key with create permission for specific API
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.create_key", otherAPI.ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Try to create key for different API
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("unrelated permission", func(t *testing.T) {
		// Create root key with completely unrelated permission
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "workspace.read")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("partial permission match", func(t *testing.T) {
		// Create root key with permission that partially matches but isn't sufficient
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.create")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})
}
