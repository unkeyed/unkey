package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateDeploymentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	// Get CTRL service URL and token
	ctrlURL, ctrlToken := containers.ControlPlane(t)

	ctrlClient := ctrlv1connect.NewDeploymentServiceClient(
		http.DefaultClient,
		ctrlURL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", ctrlToken),
		})),
	)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	t.Run("create deployment with docker image", func(t *testing.T) {
		workspace := h.CreateWorkspace()
		rootKey := h.CreateRootKey(workspace.ID)

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

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
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

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
	})

	t.Run("create deployment with build context", func(t *testing.T) {
		workspace := h.CreateWorkspace()
		rootKey := h.CreateRootKey(workspace.ID)

		projectID := uid.New(uid.ProjectPrefix)
		projectName := "test-build-project"
		projectSlug := "staging"

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
			Description:      "Staging environment",
			DeleteProtection: false,
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "develop",
			EnvironmentSlug: "staging",
		}
		err := req.FromV2DeployBuildSource(openapi.V2DeployBuildSource{
			Build: struct {
				Context    string  `json:"context"`
				Dockerfile *string `json:"dockerfile,omitempty"`
			}{
				Context:    "s3://bucket/path/to/context.tar.gz",
				Dockerfile: ptr.P("./Dockerfile"),
			},
		})
		require.NoError(t, err, "failed to set build source")

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
	})

	t.Run("create deployment with git commit info", func(t *testing.T) {
		workspace := h.CreateWorkspace()
		rootKey := h.CreateRootKey(workspace.ID)

		projectID := uid.New(uid.ProjectPrefix)
		projectName := "test-git-project"
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

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			ProjectId:       project.ID,
			Branch:          "main",
			EnvironmentSlug: "production",
			GitCommit: &openapi.V2DeployGitCommit{
				AuthorAvatarUrl: ptr.P("https://avatar.example.com/johndoe.jpg"),
				AuthorHandle:    ptr.P("johndoe"),
				CommitMessage:   ptr.P("feat: add new feature"),
				CommitSha:       ptr.P("abc123def456"),
				Timestamp:       ptr.P(int64(1704067200000)),
			},
		}
		err := req.FromV2DeployImageSource(openapi.V2DeployImageSource{
			Image: "nginx:latest",
		})
		require.NoError(t, err, "failed to set image source")

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
	})
}
