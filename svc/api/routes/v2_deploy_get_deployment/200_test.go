package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

func TestGetDeploymentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	t.Run("get existing deployment successfully", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			Permissions: []string{"project.*.create_deployment", "project.*.read_deployment"},
		})

		deploymentID := createTestDeployment(t, h.CtrlDeploymentClient, setup.Project.ID, setup.RootKey)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
		}

		req := handler.Request{
			DeploymentId: deploymentID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status, "expected 200, received: %#v", res)
		require.NotNil(t, res.Body)
		require.Equal(t, deploymentID, res.Body.Data.Id)
		require.NotEmpty(t, res.Body.Data.Status)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}

func TestGetDeploymentWithWildcardPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	// Create setup with create_deployment permission to create a test deployment
	setupCreate := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	deploymentID := createTestDeployment(t, h.CtrlDeploymentClient, setupCreate.Project.ID, setupCreate.RootKey)

	// Now create a separate key with wildcard read_deployment permission for the actual test
	rootKey := h.CreateRootKey(setupCreate.Workspace.ID, "project.*.read_deployment")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		DeploymentId: deploymentID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
}

func TestGetDeploymentWithSpecificProjectPermission(t *testing.T) {
	t.Parallel()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	// Create setup with create_deployment permission to create a test deployment
	setupCreate := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	deploymentID := createTestDeployment(t, h.CtrlDeploymentClient, setupCreate.Project.ID, setupCreate.RootKey)

	// Now create a separate key with project-specific read_deployment permission for the actual test
	rootKey := h.CreateRootKey(setupCreate.Workspace.ID, fmt.Sprintf("project.%s.read_deployment", setupCreate.Project.ID))

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	req := handler.Request{
		DeploymentId: deploymentID,
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
	require.NotNil(t, res.Body)
}

func createTestDeployment(t *testing.T, client ctrlv1connect.DeploymentServiceClient, projectID, rootKey string) string {
	t.Helper()

	req := &ctrlv1.CreateDeploymentRequest{
		ProjectId:       projectID,
		Branch:          "main",
		EnvironmentSlug: "production",
		Source: &ctrlv1.CreateDeploymentRequest_DockerImage{
			DockerImage: "nginx:latest",
		},
		GitCommit: &ctrlv1.GitCommitInfo{
			CommitSha: "abc123",
		},
	}

	connectReq := connect.NewRequest(req)
	connectReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", rootKey))

	resp, err := client.CreateDeployment(context.Background(), connectReq)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.GetDeploymentId())

	return resp.Msg.GetDeploymentId()
}
