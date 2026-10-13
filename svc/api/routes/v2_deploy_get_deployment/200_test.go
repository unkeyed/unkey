package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

func TestGetDeploymentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	t.Run("get existing deployment successfully", func(t *testing.T) {
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			Permissions: []string{"unkey:v1:{workspaceID}:**#*"},
		})

		deploymentID := uid.New(uid.DeploymentPrefix)
		h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            deploymentID,
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
			GitBranch:     "main",
		})

		route := &handler.Handler{
			DB: h.DB,
		}
		h.Register(route)

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

	setupCreate := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"unkey:v1:{workspaceID}:**#write"},
	})

	deploymentID := uid.New(uid.DeploymentPrefix)
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            deploymentID,
		WorkspaceID:   setupCreate.Workspace.ID,
		ProjectID:     setupCreate.Project.ID,
		AppID:         setupCreate.App.ID,
		EnvironmentID: setupCreate.Environment.ID,
		GitBranch:     "main",
	})

	route := &handler.Handler{
		DB: h.DB,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(setupCreate.Workspace.ID, deploymentPermission(setupCreate.Workspace.ID, "*", "*", "*", "*", "read"))

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

	setupCreate := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"unkey:v1:{workspaceID}:**#write"},
	})

	deploymentID := uid.New(uid.DeploymentPrefix)
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            deploymentID,
		WorkspaceID:   setupCreate.Workspace.ID,
		ProjectID:     setupCreate.Project.ID,
		AppID:         setupCreate.App.ID,
		EnvironmentID: setupCreate.Environment.ID,
		GitBranch:     "main",
	})

	route := &handler.Handler{
		DB: h.DB,
	}
	h.Register(route)

	rootKey := h.CreateRootKey(setupCreate.Workspace.ID, deploymentPermission(setupCreate.Workspace.ID, setupCreate.Project.ID, "*", "*", "*", "read"))

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
