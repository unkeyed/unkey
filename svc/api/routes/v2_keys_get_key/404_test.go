package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_get_key"
)

func TestGetKeyNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
		Vault:     h.Vault,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("nonexistent keyId", func(t *testing.T) {
		nonexistentKeyID := uid.New(uid.KeyPrefix)
		req := handler.Request{
			KeyId:   nonexistentKeyID,
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not find the requested key")
	})

	t.Run("key from different workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace()

		// Create API and key in the other workspace
		apiName := "other-workspace-api"
		otherAPI := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: otherWorkspace.ID,
			Name:        &apiName,
		})

		otherKey := h.CreateKey(seed.CreateKeyRequest{
			KeySpaceID:  otherAPI.KeyAuthID.String,
			WorkspaceID: otherWorkspace.ID,
		})

		// Try to access the key from different workspace using our root key
		req := handler.Request{
			KeyId:   otherKey.KeyID,
			Decrypt: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "specified key was not found")
	})
}
