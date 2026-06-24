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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		CtrlClient: &testutil.MockAppClient{},
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.delete_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("unknown app id returns 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: uid.New(uid.ProjectPrefix), App: uid.New(uid.AppPrefix)})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("app in another workspace returns 404", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		otherProjectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		otherProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "Theirs",
			Slug:        otherProjectSlug,
		})
		otherAppSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		otherApp := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   otherWorkspace.ID,
			ProjectID:     otherProject.ID,
			Name:          "Theirs",
			Slug:          otherAppSlug,
			DefaultBranch: "main",
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: otherProject.ID, App: otherApp.ID})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace app, received: %s", res.RawBody)
	})
}
