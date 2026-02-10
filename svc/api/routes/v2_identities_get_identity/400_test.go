package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_get_identity"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing identity", func(t *testing.T) {
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.GreaterOrEqual(t, len(res.Body.Error.Errors), 1)
		require.Contains(t, res.Body.Error.Errors[0].Location, "identity")
		require.Equal(t, 400, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty identity", func(t *testing.T) {
		req := handler.Request{
			Identity: "",
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.Equal(t, 400, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		req := handler.Request{
			Identity: uid.New(uid.TestPrefix),
		}

		// Call without auth header
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
			"Content-Type": {"application/json"},
		}, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Error.Type)
		require.NotEmpty(t, res.Body.Error.Detail)
	})

	t.Run("malformed Authorization header", func(t *testing.T) {
		req := handler.Request{
			Identity: uid.New(uid.TestPrefix),
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"InvalidFormat xyz"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/malformed", res.Body.Error.Type)
	})
}
