package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_permissions_create_permission"
	"github.com/unkeyed/unkey/pkg/testutil"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for missing required name
	t.Run("missing name", func(t *testing.T) {

		req := handler.Request{
			// Name is missing but slug is provided
			Slug: "test-slug",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for missing required slug
	t.Run("missing slug", func(t *testing.T) {

		req := handler.Request{
			Name: "test.permission",
			// Slug is missing
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for empty name
	t.Run("empty name", func(t *testing.T) {

		req := handler.Request{
			Name: "", // Empty string is invalid
			Slug: "test-slug",
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for empty slug
	t.Run("empty slug", func(t *testing.T) {

		req := handler.Request{
			Name: "test.permission",
			Slug: "", // Empty string is invalid
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for malformed JSON body
	t.Run("malformed JSON body", func(t *testing.T) {
		// Send malformed JSON that will fail to parse
		invalidJSON := `{"name": "test.permission", "invalid json": }`

		// Make a direct HTTP request with invalid JSON payload
		req, err := http.NewRequest(http.MethodPost, "/v2/permissions.createPermission", strings.NewReader(invalidJSON))
		require.NoError(t, err)

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

		// Use the test harness to execute the request
		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)

		// Check response status
		require.Equal(t, http.StatusBadRequest, res.Status)
	})

	// Test case for invalid slug pattern - spaces
	t.Run("invalid slug with spaces", func(t *testing.T) {
		req := handler.Request{
			Name: "test.permission",
			Slug: "test slug with spaces", // Contains spaces which are not allowed
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for invalid slug pattern - special characters
	t.Run("invalid slug with special characters", func(t *testing.T) {
		req := handler.Request{
			Name: "test.permission",
			Slug: "test@slug#with$special%chars", // Contains special chars not allowed
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for slug too long
	t.Run("slug too long", func(t *testing.T) {
		// Create a slug longer than 128 characters
		veryLongSlug := ""
		for i := 0; i < 130; i++ { // 130 characters, over the 128 limit
			veryLongSlug += "a"
		}

		req := handler.Request{
			Name: "test.permission",
			Slug: veryLongSlug,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "validate schema")
	})

	// Test case for valid slug patterns
	t.Run("valid slug patterns", func(t *testing.T) {
		validSlugs := []string{
			"simple",
			"with-dashes",
			"with_underscores",
			"with.periods",
			"Mixed123Case",
			"all-valid_chars.123",
		}

		for i, slug := range validSlugs {
			req := handler.Request{
				Name: fmt.Sprintf("test.permission.%d", i),
				Slug: slug,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](
				h,
				route,
				headers,
				req,
			)

			require.Equal(t, http.StatusOK, res.Status, "Valid slug '%s' should be accepted", slug)
		}
	})

	// Test for very long description
	t.Run("very long description", func(t *testing.T) {
		// Create a very long description (more than would be reasonable)
		veryLongDesc := ""
		for i := 0; i < 2000; i++ { // Just over 1000 character limit
			veryLongDesc += "a"
		}

		req := handler.Request{
			Name:        "test.permission",
			Slug:        "test-permission",
			Description: &veryLongDesc,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusBadRequest, res.Status, "Expected status code to be %d, got: %s", http.StatusBadRequest, res.RawBody)
	})
}
