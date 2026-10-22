package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_update_namespace"
)

func TestUpdateNamespaceAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	ns := seedNamespace(t, h, workspace.ID)
	other := seedNamespace(t, h, workspace.ID)
	namespaceURN := func(projectID, namespaceID, action string) string {
		return fmt.Sprintf("unkey:v1:%s:projects/%s/ratelimits/namespaces/%s#%s", workspace.ID, projectID, namespaceID, action)
	}

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "urn write on every namespace", permissions: []string{writeAnyNamespace(workspace.ID)}, shouldPass: true},
		{name: "urn write on this namespace", permissions: []string{namespaceURN(ns.ProjectID, ns.ID, "write")}, shouldPass: true},
		{name: "urn write alongside unrelated", permissions: []string{"api.*.read_api", namespaceURN(ns.ProjectID, ns.ID, "write")}, shouldPass: true},
		{name: "legacy update namespace", permissions: []string{"ratelimit.*.update_namespace"}, shouldPass: false},
		{name: "urn read on this namespace", permissions: []string{namespaceURN(ns.ProjectID, ns.ID, "read")}, shouldPass: false},
		{name: "urn write on another namespace", permissions: []string{namespaceURN(ns.ProjectID, other.ID, "write")}, shouldPass: false},
		{name: "urn write on another project", permissions: []string{namespaceURN(uid.New(uid.ProjectPrefix), "*", "write")}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(workspace.ID, tc.permissions...))
			name := newName()
			req := handler.Request{Namespace: ns.ID, Name: name}

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				require.Equal(t, name, findNamespace(t, h, ns).Name)
				return
			}

			if strings.Contains(tc.name, "read") {
				res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
				require.Contains(t, res.Body.Error.Detail, "#write")
				require.Contains(t, res.Body.Error.Detail, "Log in again")
				require.NotEqual(t, name, findNamespace(t, h, ns).Name)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, ns.ID)
			require.NotEqual(t, name, findNamespace(t, h, ns).Name)
		})
	}
}

func TestUpdateNamespaceExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	ns := seedNamespace(t, h, workspace.ID)
	headers := authHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/ratelimits/namespaces/*#read", workspace.ID)))

	realRes := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{Namespace: ns.ID, Name: newName()})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Namespace: uid.New(uid.RatelimitNamespacePrefix), Name: newName()})

	require.Equal(t, http.StatusForbidden, realRes.Status, "expected 403, received: %s", realRes.RawBody)
	require.Contains(t, realRes.Body.Error.Detail, "#write")
	require.Contains(t, realRes.Body.Error.Detail, "Log in again")
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotEqual(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
}
