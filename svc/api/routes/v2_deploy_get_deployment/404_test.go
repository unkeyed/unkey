package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/testutil/seed"
	"github.com/unkeyed/unkey/pkg/uid"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deploy_get_deployment"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:     h.Logger,
		DB:         h.DB,
		Keys:       h.Keys,
		CtrlClient: h.CtrlDeploymentClient,
	}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID)

	projectID := uid.New(uid.ProjectPrefix)

	h.CreateProject(seed.CreateProjectRequest{
		WorkspaceID: workspace.ID,
		Name:        "test-project",
		ID:          projectID,
		Slug:        "production",
	})

	t.Run("deployment not found", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			DeploymentId: "d_nonexistent123",
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)

		// CTRL service returns 500 for not found errors, not 404
		require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500, received: %#v", res)
		require.NotNil(t, res.Body)
	})
}
