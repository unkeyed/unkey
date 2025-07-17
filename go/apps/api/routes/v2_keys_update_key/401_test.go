package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_key"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestUpdateKeyUnauthorized(t *testing.T) {

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID:   h.Resources().UserWorkspace.ID,
		IpWhitelist:   "",
		EncryptedKeys: false,
		Name:          nil,
		CreatedAt:     nil,
		DefaultPrefix: nil,
		DefaultBytes:  nil,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: api.WorkspaceID,
		KeyAuthID:   api.KeyAuthID.String,
	})

	req := handler.Request{KeyId: key.KeyID}

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
		nonexistentKey := uid.New("test_nonexistent")
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
