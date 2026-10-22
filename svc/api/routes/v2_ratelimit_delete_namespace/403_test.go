package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_delete_namespace"
)

func TestDeleteNamespaceAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	namespaceURN := func(projectID, namespaceID, action string) string {
		return fmt.Sprintf("unkey:v1:%s:projects/%s/ratelimits/namespaces/%s#%s", workspace.ID, projectID, namespaceID, action)
	}

	testCases := []struct {
		name        string
		permissions func(target, other string, projectID string) []string
		shouldPass  bool
	}{
		{name: "urn delete on every namespace", permissions: func(_, _, _ string) []string { return []string{deleteAnyNamespace(workspace.ID)} }, shouldPass: true},
		{name: "urn delete on this namespace", permissions: func(target, _, projectID string) []string {
			return []string{namespaceURN(projectID, target, "delete")}
		}, shouldPass: true},
		{name: "urn delete alongside unrelated", permissions: func(target, _, projectID string) []string {
			return []string{"api.*.read_api", namespaceURN(projectID, target, "delete")}
		}, shouldPass: true},
		{name: "legacy delete namespace", permissions: func(_, _, _ string) []string { return []string{"ratelimit.*.delete_namespace"} }, shouldPass: false},
		{name: "urn write on this namespace", permissions: func(target, _, projectID string) []string {
			return []string{namespaceURN(projectID, target, "write")}
		}, shouldPass: false},
		{name: "urn delete on another namespace", permissions: func(_, other, projectID string) []string {
			return []string{namespaceURN(projectID, other, "delete")}
		}, shouldPass: false},
		{name: "urn delete on another project", permissions: func(_, _, _ string) []string {
			return []string{namespaceURN(uid.New(uid.ProjectPrefix), "*", "delete")}
		}, shouldPass: false},
		{name: "no permissions", permissions: func(_, _, _ string) []string { return []string{} }, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ns := seedNamespace(t, h, workspace.ID)
			other := seedNamespace(t, h, workspace.ID)
			headers := authHeaders(h.CreateRootKey(workspace.ID, tc.permissions(ns.ID, other.ID, ns.ProjectID)...))

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Namespace: ns.ID})
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				require.True(t, isDeleted(t, h, ns))
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Namespace: ns.ID})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
			require.NotContains(t, res.RawBody, ns.ID)
			require.False(t, isDeleted(t, h, ns))
		})
	}
}

func TestDeleteNamespaceExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	ns := seedNamespace(t, h, workspace.ID)
	headers := authHeaders(h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*/ratelimits/namespaces/*#read", workspace.ID)))

	realRes := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, handler.Request{Namespace: ns.ID})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Namespace: uid.New(uid.RatelimitNamespacePrefix)})

	require.Equal(t, http.StatusForbidden, realRes.Status, "expected 403, received: %s", realRes.RawBody)
	require.Contains(t, realRes.Body.Error.Detail, "#delete")
	require.Contains(t, realRes.Body.Error.Detail, "Log in again")
	require.Equal(t, http.StatusNotFound, missingRes.Status, "expected 404, received: %s", missingRes.RawBody)
	require.NotEqual(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail)
}
