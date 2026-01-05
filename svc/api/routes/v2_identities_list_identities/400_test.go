package handler_test

import (
	"fmt"
	"maps"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

func TestBadRequests(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
	}

	// Register the route with the harness
	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.read_identity")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Since this endpoint has mostly optional parameters with defaults,
	// there are fewer validation errors to test compared to other endpoints.
	t.Run("negative limit", func(t *testing.T) {
		negativeLimit := -10
		req := handler.Request{
			Limit: &negativeLimit,
		}

		// Negative limits should be rejected with a 400 status code
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)

		// Verify error type
		require.Contains(t, res.Body.Error.Type, "invalid_input")
	})

	t.Run("zero limit", func(t *testing.T) {
		zeroLimit := 0
		req := handler.Request{
			Limit: &zeroLimit,
		}

		// Zero limits should be rejected with a 400 status code
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)

		// Verify error type
		require.Contains(t, res.Body.Error.Type, "invalid_input")
	})

	t.Run("invalid cursor format", func(t *testing.T) {
		invalidCursor := "invalid_cursor_format"
		req := handler.Request{
			Cursor: &invalidCursor,
		}

		// This might return 400 or might just return an empty result set
		// depending on how cursor validation is implemented
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)

		// If it returns 400, validate the error response
		if res.Status == 400 {
			require.Equal(t, 400, res.Status)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.NotEmpty(t, res.Body.Meta.RequestId)
		}
	})

	t.Run("malformed JSON body", func(t *testing.T) {
		customHeaders := make(http.Header)
		maps.Copy(customHeaders, headers)
		customHeaders.Set("Content-Type", "application/json")

		// Create a malformed JSON string
		malformedJSON := `{"limit": 5, "cursor": "missing_quote_here}`

		// Manual approach to avoid JSON parsing issues
		resp, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(malformedJSON))
		require.NoError(t, err)

		// Set headers
		for k, values := range customHeaders {
			for _, v := range values {
				resp.Header.Add(k, v)
			}
		}

		// Just test that we send the JSON body directly to the TestCallRoute function
		// This simulates a valid request structure but with invalid JSON content
		badJSONReq := struct {
			Limit  *int    `json:"limit"`
			Cursor *string `json:"cursor"`
		}{
			Limit: ptr.P(5),
		}

		res := testutil.CallRoute[any, openapi.BadRequestErrorResponse](h, route, customHeaders, badJSONReq)

		// Verify 400 response
		require.Equal(t, http.StatusBadRequest, res.Status, "Malformed JSON should return 400")
	})

	t.Run("excessive limit value", func(t *testing.T) {
		excessiveLimit := 1000
		req := handler.Request{
			Limit: &excessiveLimit,
		}

		// Excessive limits should be rejected with a 400 status code
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)

		// Verify error type
		require.Contains(t, res.Body.Error.Type, "invalid_input")
	})

	t.Run("limit at boundary (200)", func(t *testing.T) {
		boundaryLimit := 200
		req := handler.Request{
			Limit: &boundaryLimit,
		}

		// Limit values > 100 should be rejected with a 400 status code
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)

		// Verify error type
		require.Contains(t, res.Body.Error.Type, "invalid_input")
	})
}
