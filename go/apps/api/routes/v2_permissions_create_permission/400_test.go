package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_permission"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

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
			// Name is missing
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
		require.Contains(t, "POST request body for '/v2/permissions.createPermission' failed to validate schema", res.Body.Error.Detail)
	})

	// Test case for empty name
	t.Run("empty name", func(t *testing.T) {

		req := handler.Request{
			Name: "", // Empty string is invalid
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
		require.Equal(t, "POST request body for '/v2/permissions.createPermission' failed to validate schema", res.Body.Error.Detail)
	})

	// Test case for malformed JSON body
	t.Run("malformed JSON body", func(t *testing.T) {
		// Send malformed JSON that will fail to parse
		invalidJSON := `{"name": "test.permission", "invalid json": }`

		// Make a direct HTTP request with invalid JSON payload
		req, err := http.NewRequest("POST", "/v2/permissions.createPermission", strings.NewReader(invalidJSON))
		require.NoError(t, err)

		// Add headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

		// Use the test harness to execute the request
		res := testutil.CallRaw[openapi.BadRequestErrorResponse](h, req)

		// Check response status
		require.Equal(t, http.StatusBadRequest, res.Status)
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
