package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_update_permission"
)

func TestUpdatePermissionBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	headers := authHeaders(h.CreateRootKey(workspace.ID, writeAnyPermission(workspace.ID)))
	permission := seedPermission(h, workspace.ID)

	emptyName := ""
	longName := strings.Repeat("a", 513)
	slugWithSpace := "documents read"
	emptySlug := ""
	longSlug := strings.Repeat("a", 129)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "missing permission", req: handler.Request{Name: &longName}},
		{name: "permission with space", req: handler.Request{Permission: "perm 123"}},
		{name: "empty name", req: handler.Request{Permission: permission.ID, Name: &emptyName}},
		{name: "name over 512 characters", req: handler.Request{Permission: permission.ID, Name: &longName}},
		{name: "slug with space", req: handler.Request{Permission: permission.ID, Slug: &slugWithSpace}},
		{name: "empty slug", req: handler.Request{Permission: permission.ID, Slug: &emptySlug}},
		{name: "slug over 128 characters", req: handler.Request{Permission: permission.ID, Slug: &longSlug}},
		{name: "description over 512 characters", req: handler.Request{
			Permission:  permission.ID,
			Description: nullable.NewNullableWithValue(strings.Repeat("a", 513)),
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, tc.req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		})
	}

	stored := findPermission(t, h, permission)
	require.Equal(t, permission.Name, stored.Name)
	require.Equal(t, permission.Slug, stored.Slug)
}
