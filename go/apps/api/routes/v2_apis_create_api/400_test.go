package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_apis_create_api"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestCreateApi_BadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	rootKey := h.CreateRootKey(h.Resources().UserWorkspace.ID, "api.*.create_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test with name too short
	t.Run("name too short", func(t *testing.T) {
		req := handler.Request{
			Name: "ab", // Name should be at least 3 characters
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
	})

	// Test with missing name
	t.Run("missing name", func(t *testing.T) {
		// Using empty struct to simulate missing name
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
	})

	// Test with empty name
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
		require.Equal(t, "https://unkey.com/docs/errors/bad_request", res.Body.Error.Type)
		require.Equal(t, res.Body.Error.Title, "Bad Request")

		// Check that the error message contains information about the name length
		require.Equal(t, "POST request body for '/v2/apis.createApi' failed to validate schema", res.Body.Error.Detail)
		require.Greater(t, len(res.Body.Error.Errors), 0)
		require.Equal(t, "/properties/name/minLength", res.Body.Error.Errors[0].Location)
		require.Equal(t, "minLength: got 2, want 3", res.Body.Error.Errors[0].Message)
	})

}
