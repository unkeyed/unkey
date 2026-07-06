package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

// A key scoped to one environment cannot list across the whole workspace, since
// that would return deployments from environments it may not read.
func TestListWorkspaceWideRequiresWildcard(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+setup.Environment.ID+".read_deployment")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{})
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}

// Listing always requires the wildcard environment.*.read_deployment permission,
// even when filtering down to a single environment: a grant on that one
// environment is not sufficient.
func TestListEnvironmentFilterRequiresWildcard(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup()
	rootKey := h.CreateRootKey(setup.Workspace.ID, "environment."+setup.Environment.ID+".read_deployment")

	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})

	req := handler.Request{
		Project:     rid(setup.Project.Slug),
		App:         rid(setup.App.Slug),
		Environment: rid(setup.Environment.Slug),
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), req)
	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}
