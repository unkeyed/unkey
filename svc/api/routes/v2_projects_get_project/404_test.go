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

func TestGetProjectNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("unknown slug returns 404", func(t *testing.T) {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Slug: slug})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("project in another workspace returns 404", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "Theirs",
			Slug:        slug,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Slug: slug})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace project, received: %s", res.RawBody)
	})
}
