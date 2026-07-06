package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_list_deployments"
)

func TestListPagination(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.read_deployment"},
	})

	const total = 5
	created := map[string]bool{}
	for range total {
		dep := h.CreateDeployment(seed.CreateDeploymentRequest{
			ID:            uid.New(uid.DeploymentPrefix),
			WorkspaceID:   setup.Workspace.ID,
			ProjectID:     setup.Project.ID,
			AppID:         setup.App.ID,
			EnvironmentID: setup.Environment.ID,
		})
		created[dep.ID] = true
	}

	seen := map[string]bool{}
	var cursor *string
	pages := 0
	for {
		req := handler.Request{Limit: ptr.P(2), Cursor: cursor}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
		require.LessOrEqual(t, len(res.Body.Data), 2)

		for _, d := range res.Body.Data {
			require.False(t, seen[d.Id], "deployment %s returned on more than one page", d.Id)
			seen[d.Id] = true
		}

		pages++
		require.Less(t, pages, 10, "pagination did not terminate")

		if !res.Body.Pagination.HasMore {
			require.Nil(t, res.Body.Pagination.Cursor)
			break
		}
		require.NotNil(t, res.Body.Pagination.Cursor)
		cursor = res.Body.Pagination.Cursor
	}

	require.Len(t, seen, total)
	for id := range created {
		require.True(t, seen[id], "deployment %s missing from paginated results", id)
	}
}
