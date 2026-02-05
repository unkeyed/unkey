package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_keys_update_credits"
)

func TestKeyUpdateCreditsForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:           h.DB,
		Keys:         h.Keys,
		Auditlogs:    h.Auditlogs,
		KeyCache:     h.Caches.VerificationKeyByHash,
		UsageLimiter: h.UsageLimiter,
	}

	h.Register(route)

	// Create API for testing using testutil helper
	apiName := "test-api"
	api := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        &apiName,
	})

	diffApi := h.CreateApi(seed.CreateApiRequest{
		WorkspaceID: h.Resources().UserWorkspace.ID,
		Name:        &apiName,
	})

	key := h.CreateKey(seed.CreateKeyRequest{
		WorkspaceID: api.WorkspaceID,
		KeySpaceID:  api.KeyAuthID.String,
		Remaining:   ptr.P(int32(100)),
	})

	req := handler.Request{
		KeyId:     key.KeyID,
		Operation: openapi.Increment,
		Value:     nullable.NewNullableWithValue(int64(10)),
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

	t.Run("wrong permission - has create but not update", func(t *testing.T) {
		// Create root key with read permission instead of create
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("cross workspace access", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Create a root key for the different workspace with full permissions
		rootKey := h.CreateRootKey(differentWorkspace.ID, "api.*.update_key")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("cross api access", func(t *testing.T) {
		// Create root key with read permission for a single api
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("api.%s.update_key", diffApi.ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
	})
}
