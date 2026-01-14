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
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
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
		workspace := h.CreateWorkspace()
		rootKey := h.CreateRootKey(workspace.ID)

		projectID := uid.New(uid.ProjectPrefix)
		project := h.CreateProject(seed.CreateProjectRequest{
			WorkspaceID: workspace.ID,
			Name:        "test-project",
			ID:          projectID,
			Slug:        "production",
		})

		h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:               uid.New(uid.EnvironmentPrefix),
			WorkspaceID:      workspace.ID,
			ProjectID:        project.ID,
			Slug:             "production",
			Description:      "Production environment",
			DeleteProtection: false,
		})

		deploymentID := createTestDeployment(t, h.CtrlDeploymentClient, project.ID, rootKey)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
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
