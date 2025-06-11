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
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity")
	route := &handler.Handler{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	}

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

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
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

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
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

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	t.Run("metadata exceeds maximum size limit", func(t *testing.T) {
		metaData := make(map[string]any)
		entriesNeeded := (handler.MAX_META_LENGTH_MB * 1024 * 1024) / 15
		for i := range entriesNeeded + 1000 {
			metaData[fmt.Sprintf("key_%d", i)] = ptr.P(fmt.Sprintf("some_%d", i))
		}

		rawMeta, _ := json.Marshal(metaData)

		req := openapi.V2IdentitiesCreateIdentityRequestBody{ExternalId: uid.New("test"), Meta: &metaData}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, fmt.Sprintf("Metadata is too large, it must be less than %dMB, got: %.2f", handler.MAX_META_LENGTH_MB, float64(len(rawMeta))/1024/1024), res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	// Test for missing name in rate limit
	t.Run("missing rate limit name", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.Ratelimit{
				{
					Duration: 1000,
					Limit:    1,
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Nil(t, res.Body.Error.Instance)
	})

	// Test for negative limit value
	t.Run("negative rate limit value", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.Ratelimit{
				{
					Name:     "test_limit",
					Duration: 1000, // valid duration
					Limit:    -10,  // negative limit
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Rate limit value must be greater than zero.", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	// Test for zero limit value
	t.Run("zero rate limit value", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.Ratelimit{
				{
					Name:     "test_limit",
					Duration: 1000, // valid duration
					Limit:    0,    // zero limit
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Rate limit value must be greater than zero.", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	// Test for duration less than 1000ms
	t.Run("duration less than 1000ms", func(t *testing.T) {
		req := openapi.V2IdentitiesCreateIdentityRequestBody{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.Ratelimit{
				{
					Name:     "test_limit",
					Duration: 999, // less than 1000ms
					Limit:    100, // valid limit
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "Rate limit duration must be at least 1000ms (1 second).", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Nil(t, res.Body.Error.Instance)
	})

	// Note: Testing invalid JSON directly is difficult because the Go JSON unmarshaller
	// would reject it before our handler even sees it. This is handled at the framework level.

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

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"malformed_header"},
		}

		req := handler.Request{ExternalId: uid.New("test")}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
	})
}
