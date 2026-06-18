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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_app"
)

func TestCreateAppProjectNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	ctrlClient := &testutil.MockAppClient{}
	route := &handler.Handler{
		DB:         h.DB,
		CtrlClient: ctrlClient,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.create_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("unknown project slug returns 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ProjectSlug: strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
			Name:        "App",
			Slug:        "app-slug",
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("project in another workspace returns 404", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "Theirs",
			Slug:        projectSlug,
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			ProjectSlug: projectSlug,
			Name:        "App",
			Slug:        "app-slug",
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace project, received: %s", res.RawBody)
	})

	require.Empty(t, ctrlClient.CreateAppCalls, "control plane must not be called for an inaccessible project")
}
