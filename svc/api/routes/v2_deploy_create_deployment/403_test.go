package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_create_deployment"
)

func TestCreateDeploymentInsufficientPermissions(t *testing.T) {
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
		Permissions: []string{"project.*.read_deployment"},
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

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, res.Status)
	require.NotNil(t, res.Body)
}

func TestCreateDeploymentCorrectPermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		roles []string
	}{
		{
			name:  "wildcard deploy",
			roles: []string{"project.*.create_deployment"},
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

			// For specific project test, we need to create project first, then root key with project-specific permission
			if tc.name == "specific project" {
				setup := h.CreateTestDeploymentSetup()

				// Now create root key with project-specific permission
				rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf("project.%s.create_deployment", setup.Project.ID))

				headers := http.Header{
					"Content-Type":  {"application/json"},
					"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
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

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusCreated, res.Status, "Expected 201, got: %d", res.Status)
				require.NotNil(t, res.Body)
				return
			}

			// For wildcard test, use Permissions directly
			setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
				Permissions: tc.roles,
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

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusCreated, res.Status, "Expected 201, got: %d", res.Status)
			require.NotNil(t, res.Body)
		})
	}
}
