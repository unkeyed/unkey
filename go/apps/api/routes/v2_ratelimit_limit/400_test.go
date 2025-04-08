package v2RatelimitLimit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_ratelimit_limit"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:                            h.DB,
		Keys:                          h.Keys,
		Logger:                        h.Logger,
		Permissions:                   h.Permissions,
		Ratelimit:                     h.Ratelimit,
		RatelimitNamespaceByNameCache: h.Caches.RatelimitNamespaceByName,
		RatelimitOverrideMatchesCache: h.Caches.RatelimitOverridesMatch,
	})

	h.Register(route)

	t.Run("negative cost", func(t *testing.T) {
		req := openapi.V2RatelimitLimitRequestBody{
			Namespace:  uid.New("test"),
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
			Cost:       ptr.P[int64](-5),
		}

		namespace := db.InsertRatelimitNamespaceParams{
			ID:          uid.New(uid.TestPrefix),
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        req.Namespace,
			CreatedAt:   time.Now().UnixMilli(),
		}

		if namespace.Name != "" {
			err := db.Query.InsertRatelimitNamespace(context.Background(), h.DB.RW(), namespace)
			require.NoError(t, err)
		}

		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, fmt.Sprintf("ratelimit.%s.limit", namespace.ID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.limit' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	// Uncomment and adapt these tests if needed
	/*
		t.Run("missing namespace", func(t *testing.T) {
			req := openapi.V2RatelimitLimitRequestBody{
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			}

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.limit"))},
			}

			res := testutil.CallRoute[handler.Request, openapi.BadRequestError](h, route, headers, req)

			require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
			require.NotNil(t, res.Body)

			require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Type)
			require.Equal(t, "POST request body for '/v2/ratelimit.limit' failed to validate schema", res.Body.Detail)
			require.Equal(t, http.StatusBadRequest, res.Body.Status)
			require.Equal(t, "Bad Request", res.Body.Title)
			require.NotEmpty(t, res.Body.RequestId)
			require.Greater(t, len(res.Body.Errors), 0)
			require.Nil(t, res.Body.Instance)
		})
	*/
}

func TestMissingAuthorizationHeader(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Ratelimit:   h.Ratelimit,
	})

	h.Register(route)

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{
			Namespace:  "test_namespace",
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{
			Namespace:  "test_namespace",
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{
			Namespace:  "test_namespace",
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "Got %s", res.RawBody)
		require.NotNil(t, res.Body)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{
			Namespace:  "test_namespace",
			Identifier: "user_123",
			Limit:      100,
			Duration:   60000,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})
}
