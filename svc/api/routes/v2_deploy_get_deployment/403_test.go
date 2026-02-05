package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

func TestGetDeploymentInsufficientPermissions(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	setupCreate := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"project.*.create_deployment"},
	})

	deploymentID := uid.New(uid.DeploymentPrefix)
	h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            deploymentID,
		WorkspaceID:   setupCreate.Workspace.ID,
		ProjectID:     setupCreate.Project.ID,
		EnvironmentID: setupCreate.Environment.ID,
		GitBranch:     "main",
	})

	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
	}
	h.Register(route)

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
