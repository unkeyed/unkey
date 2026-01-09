package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	// Get CTRL service URL and token
	ctrlURL, ctrlToken := containers.ControlPlane(t)

	ctrlClient := ctrlv1connect.NewDeploymentServiceClient(
		http.DefaultClient,
		ctrlURL,
	)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: ctrlClient,
		CtrlToken:  ctrlToken,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("project not found", func(t *testing.T) {
		req := handler.Request{
			ProjectId:       uid.New(uid.ProjectPrefix), // Non-existent project ID
			Branch:          "main",
			EnvironmentSlug: "production",
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusInternalServerError, res.Body.Error.Status)
		require.Equal(t, "Project or environment not found.", res.Body.Error.Detail)
	})

	t.Run("environment not found", func(t *testing.T) {
		projectID := uid.New(uid.ProjectPrefix)
		projectName := "test-project"

		h.CreateProject(seed.CreateProjectRequest{
			WorkspaceID: workspace.ID,
			Name:        projectName,
			ID:          projectID,
			Slug:        "production",
		})

		req := handler.Request{
			ProjectId:       projectID,
			Branch:          "main",
			EnvironmentSlug: "nonexistent-env", // Non-existent environment
			DockerImage:     ptr.P("nginx:latest"),
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected , received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
		require.Equal(t, http.StatusInternalServerError, res.Body.Error.Status)
		require.Equal(t, "Project or environment not found.", res.Body.Error.Detail)
	})
}
