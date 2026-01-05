package v2_ratelimit_multi_limit_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_multi_limit"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Ratelimit:               h.Ratelimit,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:               h.Auditlogs,
	}

	h.Register(route)

	t.Run("empty request array", func(t *testing.T) {
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.limit")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Empty array request
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 for empty array, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("negative cost", func(t *testing.T) {
		namespaceName := uid.New("test")
		req := handler.Request{
			{
				Namespace:  namespaceName,
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
				Cost:       ptr.P[int64](-5),
			},
		}

		namespace := db.InsertRatelimitNamespaceParams{
			ID:          uid.New(uid.RatelimitNamespacePrefix),
			WorkspaceID: h.Resources().UserWorkspace.ID,
			Name:        namespaceName,
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

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.multiLimit' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("missing namespace in request", func(t *testing.T) {
		// Create a root key with wildcard permission for any namespace
		rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "ratelimit.*.limit")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		// Request with empty namespace
		req := handler.Request{
			{
				// namespace missing (empty string)
				Namespace:  "",
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			},
		}

		// Should return an error for missing namespace
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received body: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/ratelimit.multiLimit' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})
}

func TestMissingAuthorizationHeader(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:                      h.DB,
		Keys:                    h.Keys,
		Logger:                  h.Logger,
		Ratelimit:               h.Ratelimit,
		RatelimitNamespaceCache: h.Caches.RatelimitNamespace,
		Auditlogs:               h.Auditlogs,
	}

	h.Register(route)

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{
			{
				Namespace:  "test_namespace",
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{
			{
				Namespace:  "test_namespace",
				Identifier: "user_123",
				Limit:      100,
				Duration:   60000,
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/malformed", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
