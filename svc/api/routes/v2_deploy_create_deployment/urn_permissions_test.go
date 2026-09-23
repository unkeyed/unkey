package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

// TestCreateDeploymentAuthorizesCanonicalURN guarantees an exact environment's
// deployment wildcard can create a deployment without a legacy grant.
func TestCreateDeploymentAuthorizesCanonicalURN(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})
	permission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/*#write",
		setup.Workspace.ID,
		setup.Project.ID,
		setup.App.ID,
		setup.Environment.ID,
	)
	rootKey := h.CreateRootKey(setup.Workspace.ID, permission)

	restate, creates := testutil.RecordingDeployRestate(t)
	route := newRoute(h, restate)
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:         setup.Project.Slug,
		App:             setup.App.Slug,
		Branch:          "main",
		EnvironmentSlug: setup.Environment.Slug,
		DockerImage:     "nginx:latest",
	})

	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID)
	require.Equal(t, setup.Project.ID, observed.Request.GetProjectId())
	require.Equal(t, setup.App.ID, observed.Request.GetAppId())
	require.Equal(t, setup.Environment.ID, observed.Request.GetEnvironmentId())
}

// TestCreateDeploymentRejectsURNForDifferentEnvironment guarantees deployment
// creation cannot cross an environment boundary within the same app.
func TestCreateDeploymentRejectsURNForDifferentEnvironment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})
	otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: setup.Workspace.ID,
		ProjectID:   setup.Project.ID,
		AppID:       setup.App.ID,
		Slug:        "staging",
		Description: "staging environment",
	})
	permission := fmt.Sprintf(
		"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/*#write",
		setup.Workspace.ID,
		setup.Project.ID,
		setup.App.ID,
		setup.Environment.ID,
	)
	rootKey := h.CreateRootKey(setup.Workspace.ID, permission)

	route := newRoute(h, testutil.UncalledDeployRestate(t))
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
		Project:         setup.Project.Slug,
		App:             setup.App.Slug,
		Branch:          "main",
		EnvironmentSlug: otherEnvironment.Slug,
		DockerImage:     "nginx:latest",
	})

	require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
}

// TestCreateDeploymentRejectsNonCoveringURNs guarantees every segment and the
// action in a canonical grant must cover the resolved deployment target.
func TestCreateDeploymentRejectsNonCoveringURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{},
	})

	route := newRoute(h, testutil.UncalledDeployRestate(t))
	h.Register(route)

	otherWorkspaceID := uid.New(uid.WorkspacePrefix)
	otherProjectID := uid.New(uid.ProjectPrefix)
	otherAppID := uid.New(uid.AppPrefix)
	tests := []struct {
		name      string
		workspace string
		project   string
		app       string
		action    string
	}{
		{
			name:      "wrong project ancestry",
			workspace: setup.Workspace.ID,
			project:   otherProjectID,
			app:       setup.App.ID,
			action:    "write",
		},
		{
			name:      "wrong app ancestry",
			workspace: setup.Workspace.ID,
			project:   setup.Project.ID,
			app:       otherAppID,
			action:    "write",
		},
		{
			name:      "wrong workspace",
			workspace: otherWorkspaceID,
			project:   setup.Project.ID,
			app:       setup.App.ID,
			action:    "write",
		},
		{
			name:      "wrong action",
			workspace: setup.Workspace.ID,
			project:   setup.Project.ID,
			app:       setup.App.ID,
			action:    "read",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			permission := fmt.Sprintf(
				"unkey:v1:%s:projects/%s/apps/%s/environments/%s/deployments/*#%s",
				test.workspace,
				test.project,
				test.app,
				setup.Environment.ID,
				test.action,
			)
			rootKey := h.CreateRootKey(setup.Workspace.ID, permission)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(rootKey), handler.Request{
				Project:         setup.Project.Slug,
				App:             setup.App.Slug,
				Branch:          "main",
				EnvironmentSlug: setup.Environment.Slug,
				DockerImage:     "nginx:latest",
			})

			require.Equal(t, http.StatusForbidden, res.Status, "expected 403, received: %s", res.RawBody)
		})
	}
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
}
