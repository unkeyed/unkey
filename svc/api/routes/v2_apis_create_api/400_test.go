package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
)

// TestCreateApi_BadRequest verifies that API creation requests with invalid
// request bodies, missing required fields, or malformed JSON are properly
// rejected with 400 Bad Request responses. This ensures proper input validation
// and prevents creation of APIs with invalid or incomplete data.
func TestCreateApi_BadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// This test validates the minimum length constraint for API names,
	// ensuring that names below the required character limit are rejected.
	t.Run("name too short", func(t *testing.T) {
		req := handler.Request{
			Name: "ab", // Name should be at least 3 characters
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
	})

	// This test validates that the required 'name' field is properly validated
	// and that requests missing this field are rejected with appropriate error messages.
	t.Run("missing name field", func(t *testing.T) {
		// Using empty struct to simulate missing name
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
	})

	// This test ensures that API names cannot be empty strings, validating
	// the minimum length requirements for API naming.
	t.Run("empty name", func(t *testing.T) {
		req := handler.Request{
			Name: "",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
	})
	// Test detailed error response structure
	t.Run("verify error response structure", func(t *testing.T) {
		req := handler.Request{
			Name: "ab", // Name should be at least 3 characters
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)

		// Verify the error response structure (similar to TypeScript test)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Error)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Title, "Bad Request")

		// Check that the error message contains information about the name length
		require.Equal(t, "POST request body for '/v2/apis.createApi' failed to validate schema", res.Body.Error.Detail)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Equal(t, "/properties/name/minLength", res.Body.Error.Errors[0].Location)
		require.Equal(t, "minLength: got 2, want 3", res.Body.Error.Errors[0].Message)
	})

	// Test with invalid JSON in request
	t.Run("invalid json", func(t *testing.T) {
		// Send malformed JSON that will fail to parse
		invalidJSON := `{"name": "test-api", "invalid json": }`

		// Make a direct HTTP request with invalid JSON payload
		req, err := http.NewRequest(route.Method(), route.Path(), strings.NewReader(invalidJSON))
		require.NoError(t, err)

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

		// Use the test harness to execute the request
		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)

		// Check response status
		require.Equal(t, http.StatusBadRequest, res.Status)

		// Verify error details indicate JSON parsing problem
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.NotEmpty(t, res.Body.Error.Detail)
		require.Equal(t, "Bad Request", res.Body.Error.Title)
	})

	// Test with unexpected fields in request
	t.Run("unexpected fields", func(t *testing.T) {
		// Valid request with additional unexpected fields
		req := struct {
			Name            string `json:"name"`
			UnexpectedField bool   `json:"unexpectedField"`
			AnotherField    int    `json:"anotherField"`
		}{
			Name:            "valid-api-name",
			UnexpectedField: true,
			AnotherField:    42,
		}

		res := testutil.CallRoute[struct {
			Name            string `json:"name"`
			UnexpectedField bool   `json:"unexpectedField"`
			AnotherField    int    `json:"anotherField"`
		}, handler.Response](h, route, headers, req)

		// Depending on the validation configuration, this might be 200 (if extra fields are ignored)
		// or 400 (if the schema is strict and rejects unknown properties)
		if res.Status == http.StatusBadRequest {
			// If 400, verify the error response
			require.Equal(t, http.StatusBadRequest, res.Status)
			require.Contains(t, res.RawBody, "error")
		} else {
			// If 200, verify it worked despite extra fields (API ignores unknown fields)
			require.Equal(t, http.StatusOK, res.Status)
			require.NotEmpty(t, res.Body.Data.ApiId)

			// Verify the API was created with the correct name
			api, err := db.Query.FindApiByID(context.Background(), h.DB.RO(), res.Body.Data.ApiId)
			require.NoError(t, err)
			require.Equal(t, "valid-api-name", api.Name)
		}
	})

	// Test with missing authorization header
	t.Run("missing authorization", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
			// No Authorization header
		}

		req := handler.Request{
			Name: "test-api",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 when authorization header is missing")
	})
}
