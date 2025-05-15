package handler_test

import (
	"net/http"
	"testing"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_role"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestBadRequest(t *testing.T) {
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {`Bearer ` + rootKey},
	}

	// Test case for missing roleId
	t.Run("missing roleId", func(t *testing.T) {
		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			map[string]interface{}{},
		)

		testutil.RequireBadRequest(t, res)
	})

	// Test case for empty roleId
	t.Run("empty roleId", func(t *testing.T) {
		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](
			h,
			route,
			headers,
			map[string]interface{}{
				"roleId": "",
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
			[]byte(`{"roleId": malformed}`),
		)

		testutil.RequireBadRequest(t, res)
	})
}
