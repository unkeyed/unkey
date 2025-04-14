//nolint:exhaustruct
package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_create_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity")
	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing external id", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("empty external id", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{ExternalId: ""}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("external id too short", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{ExternalId: "12"}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("metadata exceeds maximum size limit", func(t *testing.T) {
		metaData := make(map[string]interface{})
		entriesNeeded := (handler.MAX_META_LENGTH_MB * 1024 * 1024) / 15
		for i := 0; i < entriesNeeded+1000; i++ {
			var data interface{} = fmt.Sprintf("some_%d", i)
			metaData[fmt.Sprintf("key_%d", i)] = &data
		}

		rawMeta, _ := json.Marshal(metaData)

		req := openapi.V2IdentitiesCreateIdentityRequestBody{ExternalId: uid.New("test"), Meta: &metaData}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/invalid_input", res.Body.Error.Type)
		require.Equal(t, fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", handler.MAX_META_LENGTH_MB, float64(len(rawMeta))/1024/1024), res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("invalid ratelimit", func(t *testing.T) {

		req := openapi.V2IdentitiesCreateIdentityRequestBody{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.V2Ratelimit{
				{
					Duration: 1,
					Limit:    1,
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{ExternalId: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})
}
