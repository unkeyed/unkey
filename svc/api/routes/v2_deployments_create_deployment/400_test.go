package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	headers := authHeaders(setup.RootKey)

	base := func() handler.Request {
		return handler.Request{
			Project:         setup.Project.Slug,
			App:             setup.App.Slug,
			EnvironmentSlug: setup.Environment.Slug,
		}
	}

	cases := []struct {
		name   string
		mutate func(*handler.Request)
	}{
		{"image missing dockerImage", func(r *handler.Request) {
			r.Source = openapi.DeploymentSourceImage
		}},
		{"git fork without commitSha", func(r *handler.Request) {
			r.Source = openapi.DeploymentSourceGit
			r.ForkRepository = ptr.P("contributor/acme-api")
		}},
		{"git fork bad charset", func(r *handler.Request) {
			r.Source = openapi.DeploymentSourceGit
			r.CommitSha = ptr.P("abc123")
			r.ForkRepository = ptr.P("bad repo!")
		}},
		{"git fork path traversal", func(r *handler.Request) {
			r.Source = openapi.DeploymentSourceGit
			r.CommitSha = ptr.P("abc123")
			r.ForkRepository = ptr.P("../../etc/passwd")
		}},
		{"deployment missing deploymentId", func(r *handler.Request) {
			r.Source = openapi.DeploymentSourceDeployment
		}},
		{"missing source", func(r *handler.Request) {
			r.Source = ""
			r.DockerImage = ptr.P("nginx:latest")
		}},
		{"missing project", func(r *handler.Request) {
			r.Project = ""
			r.Source = openapi.DeploymentSourceImage
			r.DockerImage = ptr.P("nginx:latest")
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture.called = false
			req := base()
			tc.mutate(&req)

			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", req, res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.False(t, capture.called, "ctrl must not be called on a validation failure")
		})
	}
}
