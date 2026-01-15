package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

func TestGetDeploymentInsufficientPermissions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	// Create setup with insufficient permissions (wrong action)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"deploy.*.create_deployment"},
	})

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", setup.RootKey)},
	}

	req := handler.Request{
		ProjectId:    setup.Project.ID,
		DeploymentId: "d_123abc",
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, res.Status)
	require.NotNil(t, res.Body)
}

func TestGetDeploymentCorrectPermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		roles []string
	}{
		{
			name:  "wildcard deploy",
			roles: []string{"deploy.*.read_deployment"},
		},
		{
			name:  "specific project",
			roles: []string{}, // Will be filled in with specific project ID
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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
			setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
				Permissions: []string{"deploy.*.create_deployment"},
			})

			deploymentID := createTestDeployment(t, h.CtrlDeploymentClient, setup.Project.ID, setup.RootKey)

			// Now create a separate key with read_deployment permissions for the actual test
			var roles []string
			if tc.name == "wildcard deploy" {
				roles = tc.roles
			} else {
				roles = []string{fmt.Sprintf("deploy.%s.read_deployment", setup.Project.ID)}
			}

			rootKey := h.CreateRootKey(setup.Workspace.ID, roles...)

			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			req := handler.Request{
				ProjectId:    setup.Project.ID,
				DeploymentId: deploymentID,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
			require.NotNil(t, res.Body)
		})
	}
}
