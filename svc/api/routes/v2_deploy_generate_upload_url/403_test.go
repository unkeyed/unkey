package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_generate_upload_url"
)

func TestGenerateUploadUrlInsufficientPermissions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlBuildClient,
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
		ProjectId: setup.Project.ID,
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, res.Status)
	require.NotNil(t, res.Body)
}

func TestGenerateUploadUrlCorrectPermissions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		roles []string
	}{
		{
			name:  "wildcard deploy",
			roles: []string{"deploy.*.generate_upload_url"},
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
				CtrlClient: h.CtrlBuildClient,
			}
			h.Register(route)

			// For specific project test, we need to create project first, then root key with project-specific permission
			if tc.name == "specific project" {
				setup := h.CreateTestDeploymentSetup()

				// Now create root key with project-specific permission
				rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf("deploy.%s.generate_upload_url", setup.Project.ID))

				headers := http.Header{
					"Content-Type":  {"application/json"},
					"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
				}

				req := handler.Request{
					ProjectId: setup.Project.ID,
				}

				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
				require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
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
				ProjectId: setup.Project.ID,
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, res.Status, "Expected 200, got: %d", res.Status)
			require.NotNil(t, res.Body)
		})
	}
}
