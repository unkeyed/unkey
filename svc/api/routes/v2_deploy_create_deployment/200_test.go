package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestCreateDeploymentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	t.Run("create deployment with docker image", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			Permissions: []string{"project.*.create_deployment"},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			ProjectId:       setup.Project.ID,
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

		require.Equal(t, 201, res.Status, "expected 201, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
	})

	t.Run("create deployment with build context", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			ProjectName:     "test-build-project",
			ProjectSlug:     "staging",
			EnvironmentSlug: "staging",
			Permissions:     []string{"project.*.create_deployment"},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			ProjectId:       setup.Project.ID,
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

		require.Equal(t, 201, res.Status, "expected 201, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
	})

	t.Run("create deployment with git commit info", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			ProjectName: "test-git-project",
			Permissions: []string{"project.*.create_deployment"},
		})

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			ProjectId:       setup.Project.ID,
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

		require.Equal(t, 201, res.Status, "expected 201, received: %#v", res)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Data.DeploymentId, "deployment ID should not be empty")
	})
}
