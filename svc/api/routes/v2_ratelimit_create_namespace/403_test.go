package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_create_namespace"
)

func TestCreateNamespaceAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)

	workspace := h.Resources().UserWorkspace
	defaultProjectID, err := projects.EnsureDefaultProject(t.Context(), h.DB.RW(), workspace.ID)
	require.NoError(t, err)
	namespaceURN := func(projectID, namespaceID, action string) string {
		return fmt.Sprintf("unkey:v1:%s:projects/%s/ratelimits/namespaces/%s#%s", workspace.ID, projectID, namespaceID, action)
	}

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "urn write on every project", permissions: []string{writeAnyNamespace(workspace.ID)}, shouldPass: true},
		{name: "urn write on the default project", permissions: []string{namespaceURN(defaultProjectID, "*", "write")}, shouldPass: true},
		{name: "urn write alongside unrelated", permissions: []string{"api.*.read_api", writeAnyNamespace(workspace.ID)}, shouldPass: true},
		{name: "legacy create namespace", permissions: []string{"ratelimit.*.create_namespace"}, shouldPass: false},
		{name: "urn read on every namespace", permissions: []string{namespaceURN("*", "*", "read")}, shouldPass: false},
		{name: "urn write on one namespace", permissions: []string{namespaceURN(defaultProjectID, uid.New(uid.RatelimitNamespacePrefix), "write")}, shouldPass: false},
		{name: "urn write on another project", permissions: []string{namespaceURN(uid.New(uid.ProjectPrefix), "*", "write")}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(workspace.ID, tc.permissions...))
			name := newName()
			req := handler.Request{Name: name}

			if tc.shouldPass {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				_, found := findNamespace(t, h, workspace.ID, name)
				require.True(t, found)
				return
			}

			res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
			require.Contains(t, res.Body.Error.Detail, "ratelimits/namespaces/*#write")
			_, found := findNamespace(t, h, workspace.ID, name)
			require.False(t, found)
		})
	}
}
