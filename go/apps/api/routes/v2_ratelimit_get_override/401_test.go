package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_get_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestUnauthorizedAccess(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		namespaceName := "test_namespace"
		req := handler.Request{
			NamespaceName: &namespaceName,
			Identifier:    "test_identifier",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		namespaceName := "test_namespace"
		req := handler.Request{
			NamespaceName: &namespaceName,
			Identifier:    "test_identifier",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		namespaceName := "test_namespace"
		req := handler.Request{
			NamespaceName: &namespaceName,
			Identifier:    "test_identifier",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.NotNil(t, res.Body)
	})
}
