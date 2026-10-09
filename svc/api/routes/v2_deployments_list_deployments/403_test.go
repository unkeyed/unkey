package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

// Legacy environment tuples no longer grant read, so a key holding only them
// gets the fixed 403 for every scope, existing or not.
func TestListLegacyTuplesGrantNothing(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	requests := map[string]handler.Request{
		"workspace":       {},
		"environment":     {Project: rid(setup.Project.Slug), App: rid(setup.App.Slug), Environment: rid(setup.Environment.Slug)},
		"missing project": {Project: rid(uid.New(uid.ProjectPrefix))},
	}
	for _, permission := range []string{
		"environment.*.read_deployment",
		"environment." + setup.Environment.ID + ".read_deployment",
	} {
		rootKey := h.CreateRootKey(setup.Workspace.ID, permission)
		for name, req := range requests {
			t.Run(permission+"/"+name, func(t *testing.T) {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
			})
		}
	}
}

// TestListURNGrants covers the grants the dashboard proxy mints. Listing is all
// or nothing: a grant on one environment is a 403.
func TestListURNGrants(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})
	workspace := urn.New().Workspace(setup.Workspace.ID)

	for _, tc := range []struct {
		name    string
		grant   string
		allowed bool
	}{
		{name: "every deployment", grant: rbac.U(workspace.Project("*").App("*").Environment("*").Deployment("*"), permissions.Read).Value, allowed: true},
		{name: "one environment only", grant: rbac.U(workspace.Project(setup.Project.ID).App(setup.App.ID).Environment(setup.Environment.ID).Deployment("*"), permissions.Read).Value, allowed: false},
		{name: "wrong action", grant: rbac.U(workspace.Project("*").App("*").Environment("*").Deployment("*"), permissions.Delete).Value, allowed: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(setup.Workspace.ID, tc.grant)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{})
			if tc.allowed {
				require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}
