package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/deploy/projectgate"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_get_project"
)

func TestGetDeletedProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"unkey:v1:{workspaceID}:projects/*#read"},
	})
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + setup.RootKey}}
	live := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: setup.Project.ID})
	require.Equal(t, http.StatusOK, live.Status, "%s", live.RawBody)
	_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE projects SET deleted_at = 1 WHERE id = ?", setup.Project.ID)
	require.NoError(t, err)
	for _, identifier := range []string{setup.Project.ID, setup.Project.Slug} {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: identifier})
		require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
	}
}

func TestGetProjectNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/*#read", workspace.ID))
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("unknown id returns 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: uid.New(uid.ProjectPrefix)})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("project in another workspace returns 404", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "Theirs",
			Slug:        slug,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.ID})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace project, received: %s", res.RawBody)
	})

	t.Run("default project returns 404 by id or slug", func(t *testing.T) {
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      workspace.ID,
			Name:             "Default",
			Slug:             projectgate.DefaultSlug,
			DeleteProtection: true,
		})

		for _, identifier := range []string{project.ID, project.Slug} {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: identifier})
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for default project %q, received: %s", identifier, res.RawBody)
		}
	})
}
