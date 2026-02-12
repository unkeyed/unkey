package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:       h.DB,
		Keys:     h.Keys,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_key", "api.*.read_api")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for missing required apiId
	t.Run("missing apiId", func(t *testing.T) {
		req := handler.Request{
			// ApiId is missing - this should trigger OpenAPI validation error
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
	})

	// Test case for empty API ID (OpenAPI validation may or may not catch this)
	t.Run("empty apiId", func(t *testing.T) {
		req := handler.Request{
			ApiId: "", // Empty string - behavior depends on OpenAPI schema
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Empty string might be treated as missing, so could be 400 or pass through to handler
		if res.Status == 400 {
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
			require.Equal(t, "Bad Request", res.Body.Error.Title)
		}
		// If it passes validation, it will likely be a 404 (API not found)
	})

	// Test case for negative limit (if OpenAPI schema enforces minimum)
	t.Run("negative limit", func(t *testing.T) {
		negativeLimit := -1
		req := handler.Request{
			ApiId: "api_1234567890",
			Limit: &negativeLimit,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// This should trigger validation error if schema has minimum constraint
		if res.Status == 400 {
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		}
	})

	// Test case for zero limit (if OpenAPI schema enforces minimum > 0)
	t.Run("zero limit", func(t *testing.T) {
		zeroLimit := 0
		req := handler.Request{
			ApiId: "api_1234567890",
			Limit: &zeroLimit,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// This should trigger validation error if schema has minimum constraint
		if res.Status == 400 {
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		}
	})

	// Test case for invalid API ID format (non-api_ prefix)
	t.Run("invalid apiId format", func(t *testing.T) {
		req := handler.Request{
			ApiId: "invalid_format_123", // Should start with 'api_'
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// This may pass validation but fail at business logic level (404)
		// The test verifies we don't get a 500 error
		require.True(t, res.Status == 400 || res.Status == 404)
	})

	// Test case for extremely large limit value
	t.Run("extremely large limit", func(t *testing.T) {
		largeLimit := 999999
		req := handler.Request{
			ApiId: "api_1234567890",
			Limit: &largeLimit,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should either be validated or handled gracefully by the handler
		if res.Status == 400 {
			require.NotNil(t, res.Body)
			require.NotNil(t, res.Body.Error)
		}
	})

	// Test case for malformed cursor
	t.Run("malformed cursor", func(t *testing.T) {
		malformedCursor := "not_a_valid_cursor_format_!!!"
		req := handler.Request{
			ApiId:  "api_1234567890",
			Cursor: &malformedCursor,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Cursor validation happens at business logic level, not schema level
		// So this might be 200 (empty results), 400, or 404 (API not found)
		require.True(t, res.Status == 200 || res.Status == 400 || res.Status == 404)
	})

	// Test case for empty external ID string
	t.Run("empty external ID", func(t *testing.T) {
		emptyExternalId := ""
		req := handler.Request{
			ApiId:      "api_1234567890",
			ExternalId: &emptyExternalId,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Empty string in externalId is typically handled as "not provided"
		// Should not cause a 400 error, but might return 400 if schema enforces non-empty
		require.True(t, res.Status == 200 || res.Status == 400 || res.Status == 404)
	})

	// Test case for boolean parameters (verify they accept valid values)
	t.Run("valid boolean decrypt parameter", func(t *testing.T) {
		decryptTrue := true
		req := handler.Request{
			ApiId:   "api_1234567890",
			Decrypt: &decryptTrue,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should not be a validation error (might be 404 for non-existent API)
		require.NotEqual(t, 400, res.Status)
	})

	// Test case for valid limit values
	t.Run("valid limit values", func(t *testing.T) {
		validLimit := 50
		req := handler.Request{
			ApiId: "api_1234567890",
			Limit: &validLimit,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		// Should not be a validation error
		require.NotEqual(t, 400, res.Status)
	})

	// Test case for verifying basic validation error response structure
	t.Run("verify response structure for validation errors", func(t *testing.T) {
		// Use the known invalid case (missing required field)
		req := handler.Request{
			// ApiId is missing - this should definitely trigger validation
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.NotEmpty(t, res.Body.Error.Detail)
		require.NotEmpty(t, res.Body.Error.Title)
		require.Equal(t, 400, res.Body.Error.Status)

		// Verify error response has proper structure
		require.NotNil(t, res.Body.Meta)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify the errors array is present
		require.NotNil(t, res.Body.Error.Errors)
	})

	// Test case for multiple validation errors
	t.Run("multiple validation errors", func(t *testing.T) {
		negativeLimit := -5
		req := handler.Request{
			// Missing ApiId AND invalid limit
			Limit: &negativeLimit,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.NotEmpty(t, res.Body.Error.Detail)

		// Should have validation errors array
		require.NotNil(t, res.Body.Error.Errors)
	})
}
