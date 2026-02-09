package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestCreateDeploymentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{DeploymentId: "test-deployment-id"}), nil
			},
		},
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
			DockerImage:     "nginx:latest",
		}

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
			DockerImage:     "nginx:latest",
			GitCommit: &openapi.V2DeployGitCommit{
				AuthorAvatarUrl: ptr.P("https://avatar.example.com/johndoe.jpg"),
				AuthorHandle:    ptr.P("johndoe"),
				CommitMessage:   ptr.P("feat: add new feature"),
				CommitSha:       ptr.P("abc123def456"),
				Timestamp:       ptr.P(int64(1704067200000)),
			},
		}

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

func TestCreateDeploymentWithWildcardPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{DeploymentId: "test-deployment-id"}), nil
			},
		},
	}
	h.Register(route)

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
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "Expected 201, got: %d", res.Status)
	require.NotNil(t, res.Body)
}

func TestCreateDeploymentWithSpecificProjectPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
				return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{DeploymentId: "test-deployment-id"}), nil
			},
		},
	}
	h.Register(route)

	// First create the project/environment setup
	setup := h.CreateTestDeploymentSetup()

	// Now create a root key with project-specific permission
	rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf("project.%s.create_deployment", setup.Project.ID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		ProjectId:       setup.Project.ID,
		Branch:          "main",
		EnvironmentSlug: "production",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "Expected 201, got: %d", res.Status)
	require.NotNil(t, res.Body)
}
