//nolint:exhaustruct
package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
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
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("missing external id", func(t *testing.T) {
		req := handler.Request{}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("empty external id", func(t *testing.T) {
		req := handler.Request{ExternalId: ""}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("external id too short", func(t *testing.T) {
		req := handler.Request{ExternalId: "12"}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	t.Run("metadata exceeds maximum size limit", func(t *testing.T) {
		metaData := make(map[string]any)
		entriesNeeded := (handler.MAX_META_LENGTH_MB * 1024 * 1024) / 15
		for i := range entriesNeeded + 1000 {
			metaData[fmt.Sprintf("key_%d", i)] = ptr.P(fmt.Sprintf("some_%d", i))
		}

		req := handler.Request{ExternalId: uid.New("test"), Meta: &metaData}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	// Test for missing name in rate limit
	t.Run("missing rate limit name", func(t *testing.T) {
		req := handler.Request{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.RatelimitRequest{
				{
					Duration:  1000,
					Limit:     1,
					AutoApply: true,
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Greater(t, len(res.Body.Error.Errors), 0)
	})

	// Test for negative limit value
	t.Run("negative rate limit value", func(t *testing.T) {
		req := handler.Request{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.RatelimitRequest{
				{
					Name:      "test_limit",
					Duration:  1000, // valid duration
					Limit:     -10,  // negative limit
					AutoApply: true,
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	// Test for zero limit value
	t.Run("zero rate limit value", func(t *testing.T) {
		req := handler.Request{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.RatelimitRequest{
				{
					Name:      "test_limit",
					Duration:  1000, // valid duration
					Limit:     0,    // zero limit
					AutoApply: true,
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	// Test for duration less than 1000ms
	t.Run("duration less than 1000ms", func(t *testing.T) {
		req := handler.Request{
			ExternalId: uid.New("test"),
			Ratelimits: &[]openapi.RatelimitRequest{
				{
					Name:      "test_limit",
					Duration:  999, // less than 1000ms
					Limit:     100, // valid limit
					AutoApply: true,
				},
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
		require.NotNil(t, res.Body)

		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, "POST request body for '/v2/identities.createIdentity' failed to validate schema", res.Body.Error.Detail)
		require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
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

	t.Run("invalid credit value - too large", func(t *testing.T) {
		req := handler.Request{
			ExternalId: uid.New(""),
			Credits: &openapi.Credits{
				Remaining: nullable.NewNullableWithValue[int64](2147483648), // max int32 + 1
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		// OpenAPI schema validation returns generic message
		require.Contains(t, res.Body.Error.Detail, "failed to validate schema")
	})

	t.Run("invalid credit value - negative", func(t *testing.T) {
		req := handler.Request{
			ExternalId: uid.New(""),
			Credits: &openapi.Credits{
				Remaining: nullable.NewNullableWithValue[int64](-1),
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/identities.createIdentity' failed to validate schema")
	})

	t.Run("invalid refill day - too high", func(t *testing.T) {
		refillDay := 32

		req := handler.Request{
			ExternalId: uid.New(""),
			Credits: &openapi.Credits{
				Remaining: nullable.NewNullableWithValue[int64](100),
				Refill: &openapi.CreditsRefill{
					Amount:    50,
					Interval:  openapi.Monthly,
					RefillDay: &refillDay,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/identities.createIdentity' failed to validate schema")
	})

	t.Run("invalid refill day - zero", func(t *testing.T) {
		refillDay := 0

		req := handler.Request{
			ExternalId: uid.New(""),
			Credits: &openapi.Credits{
				Remaining: nullable.NewNullableWithValue[int64](100),
				Refill: &openapi.CreditsRefill{
					Amount:    50,
					Interval:  openapi.Monthly,
					RefillDay: &refillDay,
				},
			},
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.Contains(t, res.Body.Error.Detail, "POST request body for '/v2/identities.createIdentity' failed to validate schema")
	})
}
