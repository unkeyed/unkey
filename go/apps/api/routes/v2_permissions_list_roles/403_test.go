package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_roles"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestForbidden(t *testing.T) {
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

	t.Run("missing required permission", func(t *testing.T) {
		// Create a root key with a different permission (not rbac.*.read_role)
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{},
		)

		testutil.RequireForbidden(t, res)
	})

	t.Run("no permissions", func(t *testing.T) {
		// Create a root key with no permissions
		rootKey := h.CreateRootKey(workspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{},
		)

		testutil.RequireForbidden(t, res)
	})
}
