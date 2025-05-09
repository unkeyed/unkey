package handler_test

import (
	"net/http"
	"testing"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_roles"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequest(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {`Bearer ` + rootKey},
	}

	// Test case for invalid limit (negative)
	t.Run("invalid limit", func(t *testing.T) {
		invalidLimit := int32(-10)

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			map[string]interface{}{
				"limit": invalidLimit,
			},
		)

		testutil.RequireBadRequest(t, res)
	})

	// Test case for invalid cursor format
	t.Run("invalid cursor format", func(t *testing.T) {
		invalidCursor := "not-a-valid-cursor-format"

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			map[string]interface{}{
				"cursor": invalidCursor,
			},
		)

		testutil.RequireBadRequest(t, res)
	})

	// Test case for malformed JSON
	t.Run("malformed json", func(t *testing.T) {
		res := testutil.CallRouteWithRawBody(
			h,
			route,
			headers,
			[]byte(`{"limit": "not-a-number"}`),
		)

		testutil.RequireBadRequest(t, res)
	})
}
