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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
)

func TestUpdateAppDuplicateSlug(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.update_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})

	createApp := func(t *testing.T) (string, string) {
		t.Helper()
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		app := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          "App",
			Slug:          slug,
			DefaultBranch: "main",
		})
		return app.ID, slug
	}

	t.Run("changing slug to an existing slug in the same project returns 409", func(t *testing.T) {
		_, existingSlug := createApp(t)
		id, _ := createApp(t)

		res := testutil.CallRoute[handler.Request, openapi.ConflictErrorResponse](h, route, headers, handler.Request{
			AppId: id,
			Slug:  &existingSlug,
		})
		require.Equal(t, http.StatusConflict, res.Status, "expected 409, received: %s", res.RawBody)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/app_already_exists", res.Body.Error.Type)
	})
}
