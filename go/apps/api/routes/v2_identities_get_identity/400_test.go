package handler_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity/handler"
	"github.com/unkeyed/unkey/go/internal/testutil"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	t.Run("missing both identityId and externalId", func(t *testing.T) {
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "missing required field", res.Body.Error.Detail)
		require.Equal(t, 400, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("empty identityId", func(t *testing.T) {
		emptyStr := ""
		req := handler.Request{
			identityID: &emptyStr,
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
		emptyStr := ""
		req := handler.Request{
			externalID: &emptyStr,
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
}
