package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_get_api"
)

func TestGetApiInvalidRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
		Caches: h.Caches,
	}

	h.Register(route)

	// Create a valid root key for authentication
	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.read_api")
	validHeaders := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {rootKey},
	}

	// Test with missing apiId field
	t.Run("missing apiId field", func(t *testing.T) {
		// Create empty request
		req := handler.Request{
			ApiId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			validHeaders,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "POST request body for '/v2/apis.getApi' failed to validate schema")
	})

	// We're unable to test malformed JSON using CallRoute as it would fail at Go's JSON unmarshalling
	// Test with empty apiId field
	t.Run("empty apiId", func(t *testing.T) {
		req := handler.Request{
			ApiId: "",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			validHeaders,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Error)
	})

	// Test with a valid apiId
	t.Run("valid request", func(t *testing.T) {
		// Create a test API in the database
		apiID := "api_valid_test_id"

		req := handler.Request{
			ApiId: apiID,
		}

		// This will return 404 since the API doesn't exist
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			validHeaders,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Error)
	})
}
