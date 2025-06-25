package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestNamespaceNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with non-existent namespace ID
	t.Run("namespace id not found", func(t *testing.T) {
		nonExistentNamespaceId := "ns_nonexistent"
		req := handler.Request{
			NamespaceId: &nonExistentNamespaceId,
			Identifier:  "some_identifier",
			Limit:       10,
			Duration:    1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})

	// Test with non-existent namespace name
	t.Run("namespace name not found", func(t *testing.T) {
		nonExistentNamespaceName := "nonexistent_namespace"
		req := handler.Request{
			NamespaceName: &nonExistentNamespaceName,
			Identifier:    "some_identifier",
			Limit:         10,
			Duration:      1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/ratelimit_namespace_not_found", res.Body.Error.Type)
	})
}
