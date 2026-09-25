package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

// deploymentAuthorizationFixture provides one owned deployment with no read permission.
type deploymentAuthorizationFixture struct {
	h            *testutil.Harness
	route        *handler.Handler
	setup        testutil.DeploymentTestSetup
	deploymentID string
}

// TestGetDeploymentWithURNPermission guarantees that concrete and wildcard
// URN permissions propagate through root key authentication to this legacy route.
func TestGetDeploymentWithURNPermission(t *testing.T) {
	t.Parallel()
	fixture := newDeploymentAuthorizationFixture(t)

	testCases := []struct {
		name       string
		permission string
	}{
		{
			name: "concrete",
			permission: deploymentPermission(
				fixture.setup.Workspace.ID,
				fixture.setup.Project.ID,
				fixture.setup.App.ID,
				fixture.setup.Environment.ID,
				fixture.deploymentID,
				"read",
			),
		},
		{
			name: "wildcard",
			permission: deploymentPermission(
				fixture.setup.Workspace.ID,
				"*",
				"*",
				"*",
				"*",
				"read",
			),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := fixture.h.CreateRootKey(fixture.setup.Workspace.ID, tc.permission)
			res := fixture.call(rootKey)

			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, fixture.deploymentID, res.Body.Data.Id)
		})
	}
}

// TestGetDeploymentRejectsURNPermissionWithWrongAncestry guarantees that
// every segment of the deployment's stored ownership path constrains access.
func TestGetDeploymentRejectsURNPermissionWithWrongAncestry(t *testing.T) {
	t.Parallel()
	fixture := newDeploymentAuthorizationFixture(t)

	testCases := []struct {
		name          string
		projectID     string
		appID         string
		environmentID string
		deploymentID  string
	}{
		{
			name:          "project",
			projectID:     uid.New(uid.ProjectPrefix),
			appID:         fixture.setup.App.ID,
			environmentID: fixture.setup.Environment.ID,
			deploymentID:  fixture.deploymentID,
		},
		{
			name:          "app",
			projectID:     fixture.setup.Project.ID,
			appID:         uid.New(uid.AppPrefix),
			environmentID: fixture.setup.Environment.ID,
			deploymentID:  fixture.deploymentID,
		},
		{
			name:          "environment",
			projectID:     fixture.setup.Project.ID,
			appID:         fixture.setup.App.ID,
			environmentID: uid.New(uid.EnvironmentPrefix),
			deploymentID:  fixture.deploymentID,
		},
		{
			name:          "deployment",
			projectID:     fixture.setup.Project.ID,
			appID:         fixture.setup.App.ID,
			environmentID: fixture.setup.Environment.ID,
			deploymentID:  uid.New(uid.DeploymentPrefix),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			permission := deploymentPermission(
				fixture.setup.Workspace.ID,
				tc.projectID,
				tc.appID,
				tc.environmentID,
				tc.deploymentID,
				"read",
			)
			rootKey := fixture.h.CreateRootKey(fixture.setup.Workspace.ID, permission)
			res := fixture.call(rootKey)

			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}

// TestGetDeploymentRejectsURNWritePermission guarantees that an action on
// the correct resource cannot authorize a different operation.
func TestGetDeploymentRejectsURNWritePermission(t *testing.T) {
	t.Parallel()
	fixture := newDeploymentAuthorizationFixture(t)

	permission := deploymentPermission(
		fixture.setup.Workspace.ID,
		fixture.setup.Project.ID,
		fixture.setup.App.ID,
		fixture.setup.Environment.ID,
		fixture.deploymentID,
		"write",
	)
	rootKey := fixture.h.CreateRootKey(fixture.setup.Workspace.ID, permission)
	res := fixture.call(rootKey)

	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}

// TestGetDeploymentRejectsURNPermissionFromAnotherWorkspace guarantees
// that URN workspace scope is enforced independently of resource ancestry.
func TestGetDeploymentRejectsURNPermissionFromAnotherWorkspace(t *testing.T) {
	t.Parallel()
	fixture := newDeploymentAuthorizationFixture(t)

	permission := deploymentPermission(
		uid.New(uid.WorkspacePrefix),
		fixture.setup.Project.ID,
		fixture.setup.App.ID,
		fixture.setup.Environment.ID,
		fixture.deploymentID,
		"read",
	)
	rootKey := fixture.h.CreateRootKey(fixture.setup.Workspace.ID, permission)
	res := fixture.call(rootKey)

	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}

// TestGetDeploymentMasksURNCrossWorkspaceAccess guarantees that an exact
// foreign permission cannot bypass the workspace guard or reveal deployment metadata.
func TestGetDeploymentMasksURNCrossWorkspaceAccess(t *testing.T) {
	t.Parallel()
	fixture := newDeploymentAuthorizationFixture(t)

	foreign := fixture.h.CreateTestDeploymentSetup()
	foreignDeploymentID := uid.New(uid.DeploymentPrefix)
	fixture.h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            foreignDeploymentID,
		WorkspaceID:   foreign.Workspace.ID,
		ProjectID:     foreign.Project.ID,
		AppID:         foreign.App.ID,
		EnvironmentID: foreign.Environment.ID,
		GitBranch:     "main",
	})

	permission := deploymentPermission(
		foreign.Workspace.ID,
		foreign.Project.ID,
		foreign.App.ID,
		foreign.Environment.ID,
		foreignDeploymentID,
		"read",
	)
	rootKey := fixture.h.CreateRootKey(fixture.setup.Workspace.ID, permission)
	res := fixture.callDeployment(rootKey, foreignDeploymentID)

	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotContains(t, res.RawBody, foreign.Workspace.ID)
	require.NotContains(t, res.RawBody, foreignDeploymentID)
}

// newDeploymentAuthorizationFixture creates the owned deployment used by
// URN permission tests and registers the route under test.
func newDeploymentAuthorizationFixture(t *testing.T) deploymentAuthorizationFixture {
	t.Helper()
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup()
	deploymentID := uid.New(uid.DeploymentPrefix)
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            deploymentID,
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		GitBranch:     "main",
	})

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	return deploymentAuthorizationFixture{
		h:            h,
		route:        route,
		setup:        setup,
		deploymentID: deploymentID,
	}
}

// call invokes the route for the fixture's owned deployment.
func (f deploymentAuthorizationFixture) call(rootKey string) testutil.TestResponse[handler.Response] {
	return f.callDeployment(rootKey, f.deploymentID)
}

// callDeployment invokes the route for an explicit deployment ID.
func (f deploymentAuthorizationFixture) callDeployment(rootKey, deploymentID string) testutil.TestResponse[handler.Response] {
	return testutil.CallRoute[handler.Request, handler.Response](f.h, f.route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{DeploymentId: deploymentID})
}

// deploymentPermission formats an independently expected deployment permission.
func deploymentPermission(workspaceID, projectID, appID, environmentID, deploymentID, action string) string {
	return fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/%s#%s",
		workspaceID,
		projectID,
		appID,
		environmentID,
		deploymentID,
		action,
	)
}
