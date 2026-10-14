package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_ratelimit_list_namespaces"
)

func TestListNamespacesPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	namespace := seedNamespace(t, h, workspace.ID, uid.New("KEBAP"))
	namespaces := urn.New().Workspace(workspace.ID).Project("*").RatelimitNamespace("*")
	otherWorkspaceNamespaces := urn.New().Workspace(h.CreateWorkspace().ID).Project("*").RatelimitNamespace("*")
	otherProject := urn.New().Workspace(workspace.ID).Project(uid.New(uid.ProjectPrefix))

	testCases := []struct {
		name        string
		permissions []string
		visible     bool
	}{
		{name: "every namespace", permissions: []string{fmt.Sprintf("%s#read", namespaces)}, visible: true},
		{name: "every namespace alongside unrelated", permissions: []string{"api.*.read_api", fmt.Sprintf("%s#read", namespaces)}, visible: true},
		{name: "global admin", permissions: []string{fmt.Sprintf("unkey:v1:%s:**#*", workspace.ID)}, visible: true},
		{name: "project namespaces", permissions: []string{fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(namespace.projectID).RatelimitNamespace("*"))}, visible: true},
		{name: "specific namespace", permissions: []string{fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(namespace.projectID).RatelimitNamespace(namespace.id))}, visible: true},
		{name: "no permissions", permissions: []string{}, visible: false},
		{name: "legacy override read", permissions: []string{"ratelimit.*.read_override"}, visible: false},
		{name: "legacy create namespace", permissions: []string{"ratelimit.*.create_namespace"}, visible: false},
		{name: "legacy wildcard", permissions: []string{"ratelimit.*.read_namespace"}, visible: false},
		{name: "legacy specific namespace", permissions: []string{fmt.Sprintf("ratelimit.%s.read_namespace", namespace.id)}, visible: false},
		{name: "write", permissions: []string{fmt.Sprintf("%s#write", namespaces)}, visible: false},
		{name: "override read", permissions: []string{fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project("*").RatelimitNamespace("*").Override("*"))}, visible: false},
		{name: "other workspace", permissions: []string{fmt.Sprintf("%s#read", otherWorkspaceNamespaces)}, visible: false},
		{name: "other project namespaces", permissions: []string{fmt.Sprintf("%s#read", otherProject.RatelimitNamespace("*"))}, visible: false},
		{name: "everything in other project", permissions: []string{fmt.Sprintf("%s/**#read", otherProject)}, visible: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := authHeaders(h.CreateRootKey(workspace.ID, tc.permissions...))
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			if !tc.visible {
				require.Empty(t, res.Body.Data, res.RawBody)
				return
			}
			require.Len(t, res.Body.Data, 1, res.RawBody)
			require.Equal(t, namespace.id, res.Body.Data[0].Id)
		})
	}
}
