package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_reroll_key"
)

func TestRerollKeyForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: true,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	otherApi := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: true,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		Disabled:    false,
		WorkspaceID: h.Resources().UserWorkspace.ID,
		KeySpaceID:  api.KeyAuthID.String,
	})

	req := handler.Request{
		KeyId:      key.KeyID,
		Expiration: 0,
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
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.create_key", otherApi.ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	encryptedKey := h.CreateKey(seed.CreateKeyRequest{
		Disabled:     false,
		Recoverable:  true,
		WorkspaceID:  h.Resources().UserWorkspace.ID,
		KeySpaceID:   api.KeyAuthID.String,
		Remaining:    nil,
		IdentityID:   nil,
		Meta:         nil,
		Expires:      nil,
		Name:         nil,
		Deleted:      false,
		RefillAmount: nil,
		RefillDay:    nil,
		Permissions:  nil,
		Roles:        nil,
		Ratelimits:   nil,
	})

	t.Run("reroll recoverable key without perms", func(t *testing.T) {
		// Create root key with permission that partially matches but isn't sufficient because no encryption permission
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

		req := handler.Request{
			KeyId:      encryptedKey.KeyID,
			Expiration: 0,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})
}
