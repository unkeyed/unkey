package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/uid"
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
		ProjectId:       uid.New(uid.ProjectPrefix), // Non-existent project ID
		Branch:          "main",
		EnvironmentSlug: "production",
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_not_found", res.Body.Error.Type)
	require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
	require.Equal(t, "The requested project does not exist or has been deleted.", res.Body.Error.Detail)
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
		ProjectId:       setup.Project.ID,
		Branch:          "main",
		EnvironmentSlug: "nonexistent-env", // Non-existent environment
		DockerImage:     "nginx:latest",
	}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	require.NotNil(t, res.Body)
}
