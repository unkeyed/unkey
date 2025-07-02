package handler_test

import (
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing both identityId and externalId", func(t *testing.T) {
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.getIdentity' failed to validate schema", res.Body.Error.Detail)
		require.GreaterOrEqual(t, len(res.Body.Error.Errors), 1)
		log.Printf("%+v", res.Body.Error.Errors)
		require.Equal(t, "/oneOf", res.Body.Error.Errors[0].Location)
		require.Equal(t, "'oneOf' failed, none matched", res.Body.Error.Errors[0].Message)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty identityId", func(t *testing.T) {
		req := handler.Request{
			IdentityId: ptr.P(""),
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.getIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty externalId", func(t *testing.T) {
		req := handler.Request{
			ExternalId: ptr.P(""),
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.getIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("both identityId and externalId provided", func(t *testing.T) {
		identityId := "id_123456789"
		externalId := "external_123"
		req := handler.Request{
			IdentityId: &identityId,
			ExternalId: &externalId,
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.getIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		req := handler.Request{
			IdentityId: strPtr("identity_123"),
		}

		// Call without auth header
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, http.Header{
			"Content-Type": {"application/json"},
		}, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Authorization header for 'bearer' scheme", res.Body.Error.Detail)
	})

	t.Run("malformed Authorization header", func(t *testing.T) {
		req := handler.Request{
			IdentityId: strPtr("identity_123"),
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"InvalidFormat xyz"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/malformed", res.Body.Error.Type)
	})
}
