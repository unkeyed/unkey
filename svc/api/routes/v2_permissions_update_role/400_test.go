package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_role"
)

func TestUpdateRoleBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyRole(workspace.ID)))
	role := seedRole(h, workspace.ID)

	emptyName := ""
	longName := strings.Repeat("a", 129)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing role", req: handler.Request{}},
		{name: "role over 128 characters", req: handler.Request{Role: strings.Repeat("a", 129)}},
		{name: "empty name", req: handler.Request{Role: role.ID, Name: &emptyName}},
		{name: "name over 128 characters", req: handler.Request{Role: role.ID, Name: &longName}},
		{name: "description over 512 characters", req: handler.Request{
			Role:        role.ID,
			Description: nullable.NewNullableWithValue(strings.Repeat("a", 513)),
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	stored := findRole(t, h, role)
	require.Equal(t, role.Name, stored.Name)
	require.Equal(t, role.Description.String, stored.Description.String)
}
