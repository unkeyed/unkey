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

	// Create setup with create_deployment permission to create a test deployment
	setupCreate := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	// Create an actual deployment
	deploymentID := createTestDeployment(t, h.CtrlDeploymentClient, setupCreate.Project.ID, setupCreate.RootKey)

	// Now create a key with insufficient permissions (no read_deployment)
	rootKeyWithoutRead := h.CreateRootKey(setupCreate.Workspace.ID, "project.*.create_deployment")

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKeyWithoutRead)},
	}

	req := handler.Request{
		DeploymentId: deploymentID,
	}

	res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
	require.Equal(t, http.StatusForbidden, res.Status)
	require.NotNil(t, res.Body)
}
