package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_get_project"
)

func TestGetProjectSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("returns a never-updated project with populated fields", func(t *testing.T) {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Payments Service",
			Slug:        slug,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ProjectId: project.ID})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Equal(t, project.ID, res.Body.Data.Id)
		require.True(t, strings.HasPrefix(res.Body.Data.Id, "proj_"), "id should have proj_ prefix: %s", res.Body.Data.Id)
		require.Equal(t, slug, res.Body.Data.Slug)
		require.Equal(t, "Payments Service", res.Body.Data.Name)
		require.Greater(t, res.Body.Data.CreatedAt, int64(0))
		require.Zero(t, res.Body.Data.UpdatedAt, "never-updated project should have zero (omitted) updatedAt")
		require.False(t, res.Body.Data.DeleteProtection)
	})

	t.Run("returns a delete-protected project", func(t *testing.T) {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      workspace.ID,
			Name:             "Protected Service",
			Slug:             slug,
			DeleteProtection: true,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ProjectId: project.ID})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.True(t, res.Body.Data.DeleteProtection)
	})
}
