package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestGitSourceWithoutRepoConnection(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	// No repo connection attached to the app.
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := handler.Request{
		Project:         setup.Project.Slug,
		App:             setup.App.Slug,
		EnvironmentSlug: setup.Environment.Slug,
		Source:          openapi.DeploymentSourceGit,
		Branch:          ptr.P("main"),
	}

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
	require.False(t, capture.called, "ctrl must not be called when the app has no repo connection")
}

// TestControlPlanePreconditionFailure verifies a FailedPrecondition from ctrl is
// surfaced as 412 rather than the 503 that HandleError would otherwise produce,
// and that ctrl's raw message is masked so upstream detail cannot leak or be
// probed through the caller-facing error.
func TestControlPlanePreconditionFailure(t *testing.T) {
	h := testutil.NewHarness(t)
	const ctrlSecret = "status 403: {\"message\":\"rate limit for installation 12345\"}"
	capture := &ctrlCapture{
		err: connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("failed to resolve git commit metadata: %s", ctrlSecret)),
	}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	connectRepo(t, h, setup.Workspace.ID, setup.Project.ID, setup.App.ID)

	req := handler.Request{
		Project:         setup.Project.Slug,
		App:             setup.App.Slug,
		EnvironmentSlug: setup.Environment.Slug,
		Source:          openapi.DeploymentSourceGit,
		Branch:          ptr.P("main"),
	}

	res := testutil.CallRoute[handler.Request, openapi.PreconditionFailedErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusPreconditionFailed, res.Status, "expected 412, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/precondition_failed", res.Body.Error.Type)
	require.True(t, capture.called, "ctrl should have been called")
	require.NotContains(t, res.Body.Error.Detail, ctrlSecret, "ctrl raw message must not leak to the caller")
	require.NotContains(t, res.Body.Error.Detail, "rate limit", "upstream detail must not leak to the caller")
	require.False(t, strings.Contains(res.RawBody, "installation 12345"), "internal identifiers must not leak in the response body")
}
