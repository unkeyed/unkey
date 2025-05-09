package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_permissions"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestValidationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for negative limit
	t.Run("negative limit", func(t *testing.T) {
		req := handler.Request{
			Limit: -10, // Negative limit is invalid
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
		require.Contains(t, res.Body.Error.Detail, "invalid")
	})

	// Test case for limit above maximum
	t.Run("limit above maximum", func(t *testing.T) {
		req := handler.Request{
			Limit: 1000, // Above maximum of 100
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		// Note: Our implementation may actually cap the limit instead of returning an error
		// If it does, this test might not fail as expected
	})

	// Test case for invalid cursor format
	t.Run("invalid cursor format", func(t *testing.T) {
		invalidCursor := "not_a_valid_cursor_format"
		req := handler.Request{
			Cursor: &invalidCursor,
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 400, res.Status)
		// Note: This might not fail if the cursor format is not validated strictly
	})

	// Test case for malformed JSON body
	t.Run("malformed JSON body", func(t *testing.T) {
		res, err := h.Client.Post(
			"/v2/permissions.listPermissions",
			"application/json",
			[]byte("{malformed json"),
			headers,
		)

		require.NoError(t, err)
		require.Equal(t, 400, res.StatusCode)
	})
}
