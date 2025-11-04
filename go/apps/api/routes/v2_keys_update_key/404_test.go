package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_keys_update_key"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
)

func TestUpdateKeyNotFound(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Logger:       h.Logger,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.update_key")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("when the key does not exist", func(t *testing.T) {
		t.Parallel()
		req := handler.Request{
			KeyId:   "nonexistent_key",
			Enabled: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not find the requested key")
	})

	t.Run("when the key has been deleted", func(t *testing.T) {
		t.Parallel()
		// Create API using helper
		api := h.CreateApi(seed.CreateApiRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
		})

		// Create key using helper then delete it
		keyResponse := h.CreateKey(seed.CreateKeyRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			KeySpaceID:  api.KeyAuthID.String,
			Name:        ptr.P("test"),
			Deleted:     true, // This will mark the key as deleted
		})

		req := handler.Request{
			KeyId:   keyResponse.KeyID,
			Enabled: ptr.P(false),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "We could not find the requested key")
	})
}
