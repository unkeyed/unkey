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
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_update_project"
)

func TestUpdateProjectDuplicateSlug(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.update_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	createProject := func(t *testing.T) (string, string) {
		t.Helper()
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Project",
			Slug:        slug,
		})
		return project.ID, slug
	}

	t.Run("changing slug to an existing slug returns 409", func(t *testing.T) {
		_, existingSlug := createProject(t)
		id, _ := createProject(t)

		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{
			Project: id,
			Slug:    &existingSlug,
		})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/project_already_exists", res.Body.Error.Type)
	})
}
