package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestUnauthorizedAccess(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()

	projectID := uid.New(uid.ProjectPrefix)
	projectName := "test-project"
	projectSlug := "production"

	project := h.CreateProject(seed.CreateProjectRequest{
		WorkspaceID: workspace.ID,
		Name:        projectName,
		ID:          projectID,
		Slug:        projectSlug,
	})

	h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:               uid.New(uid.EnvironmentPrefix),
		WorkspaceID:      workspace.ID,
		ProjectID:        project.ID,
		Slug:             projectSlug,
		Description:      "Production environment",
		DeleteProtection: false,
	})

	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token"},
		}

		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})

		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status, "expected 401, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
	})
}
