package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestProjectNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "test-deployment-id"}, nil
			},
		},
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	req := handler.Request{
		Project:         "nonexistent-project-slug",
		App:             "default",
		Branch:          "main",
		EnvironmentSlug: "production",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
	require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
	require.Equal(t, "The requested project or app does not exist.", res.Body.Error.Detail)
}

func TestCrossWorkspaceProjectIsolation(t *testing.T) {
	h := testutil.NewHarness(t)

	// Both workspaces use the same project slug "shared-slug" and app slug "default".
	// The attacker should only be able to reach their own workspace's project.
	sharedSlug := "shared-slug"

	attackerSetup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		ProjectSlug: sharedSlug,
		Permissions: []string{"project.*.create_deployment"},
	})
	// Victim has a project with the exact same slug in a different workspace.
	_ = h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		ProjectSlug: sharedSlug,
		Permissions: []string{"project.*.create_deployment"},
	})

	var capturedProjectID string
	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				capturedProjectID = req.ProjectId
				return &ctrlv1.CreateDeploymentResponse{DeploymentId: "test-deployment-id"}, nil
			},
		},
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", attackerSetup.RootKey)},
	}

	// Attacker uses the shared slug — should resolve to *their own* project, not the victim's.
	req := handler.Request{
		Project:         sharedSlug,
		App:             "default",
		Branch:          "main",
		EnvironmentSlug: "production",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
	require.Equal(t, http.StatusCreated, res.Status, "expected 201, received: %s", res.RawBody)

	// The resolved project ID must be the attacker's, not the victim's.
	require.Equal(t, attackerSetup.Project.ID, capturedProjectID,
		"slug resolved to wrong workspace's project")
}

func TestEnvironmentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
		CtrlClient: &testutil.MockDeploymentClient{
			CreateDeploymentFunc: func(ctx context.Context, req *ctrlv1.CreateDeploymentRequest) (*ctrlv1.CreateDeploymentResponse, error) {
				return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("environment not found"))
			},
		},
	}
	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	req := handler.Request{
		Project:         setup.Project.Slug,
		App:             "default",
		Branch:          "main",
		EnvironmentSlug: "nonexistent-env",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
