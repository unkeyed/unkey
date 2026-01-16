package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	t.Run("project not found", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       uid.New(uid.ProjectPrefix), // Non-existent project ID
			Branch:          "main",
			EnvironmentSlug: "production",
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})

		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusInternalServerError, res.Body.Error.Status)
		require.Equal(t, "The requested project does not exist or has been deleted.", res.Body.Error.Detail)
	})

	t.Run("environment not found", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       setup.Project.ID,
			Branch:          "main",
			EnvironmentSlug: "nonexistent-env", // Non-existent environment
		}

		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusInternalServerError, res.Body.Error.Status)
		require.Equal(t, "Project not found.", res.Body.Error.Detail)
	})
}
