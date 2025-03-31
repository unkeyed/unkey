package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.set_override")
	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing all required fields", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Greater(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)
	})

	t.Run("missing identifier", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			NamespaceId: util.Pointer("test_namespace_id"),
			Limit:       10,
			Duration:    1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Greater(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)
	})

	t.Run("empty identifier", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			NamespaceId: util.Pointer("test_namespace_id"),
			Identifier:  "",
			Limit:       10,
			Duration:    1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Greater(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)

	})

	t.Run("missing duration", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			NamespaceId: util.Pointer("test_namespace_id"),
			Identifier:  "user_123",
			Limit:       10,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Greater(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)
	})

	t.Run("invalid limit (negative)", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			NamespaceId: util.Pointer("test_namespace_id"),
			Identifier:  "user_123",
			Limit:       -10,
			Duration:    1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Greater(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)
	})

	t.Run("invalid duration (negative)", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			NamespaceId: util.Pointer("test_namespace_id"),
			Identifier:  "user_123",
			Limit:       10,
			Duration:    -1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Greater(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)
	})

	t.Run("neither namespace ID nor name provided", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			NamespaceId:   nil,
			NamespaceName: nil,
			Identifier:    "user_123",
			Limit:         10,
			Duration:      1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
		require.Equal(t, "You must provide either a namespace ID or name.", res.Body.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Status)
		require.Equal(t, "Bad Request", res.Body.Title)
		require.NotEmpty(t, res.Body.RequestId)
		require.Equal(t, len(res.Body.Errors), 0)
		require.Nil(t, res.Body.Instance)
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
			Limit:         10,
			Duration:      1000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})
}
