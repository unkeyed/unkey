package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
	})

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("identity ID does not exist", func(t *testing.T) {
		nonExistentID := uid.New(uid.IdentityPrefix)
		req := handler.Request{
			IdentityId: &nonExistentID,
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, got: %d", res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity/not_found", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "not found")
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("external ID does not exist", func(t *testing.T) {
		nonExistentExternalID := "non_existent_external_id"
		req := handler.Request{
			ExternalId: &nonExistentExternalID,
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, got: %d", res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity/not_found", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Detail, "not found")
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
