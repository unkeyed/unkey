package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_set_override"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.set_override")
	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
	}

	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing all required fields", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("missing identifier", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			Namespace: "test_namespace_id",
			Limit:     10,
			Duration:  1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("empty identifier", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			Namespace:  "test_namespace_id",
			Identifier: "",
			Limit:      10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)

	})

	t.Run("missing duration", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			Namespace:  "test_namespace_id",
			Identifier: "user_123",
			Limit:      10,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("invalid limit (negative)", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			Namespace:  "test_namespace_id",
			Identifier: "user_123",
			Limit:      -10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("invalid duration (negative)", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			Namespace:  "test_namespace_id",
			Identifier: "user_123",
			Limit:      10,
			Duration:   -1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("neither namespace ID nor name provided", func(t *testing.T) {
		req := openapi.V2RatelimitSetOverrideRequestBody{
			Namespace:  "",
			Identifier: "user_123",
			Limit:      10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.setOverride' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{
			Namespace:  uid.New("test"),
			Identifier: "test_identifier",
			Limit:      10,
			Duration:   1000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 401, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
	})
}
