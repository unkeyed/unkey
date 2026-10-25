package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_get_environment"
)

func TestGetEnvironmentDeletedAncestors(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"unkey:v1:{workspaceID}:projects/*/apps/*/environments/*#read"},
	})
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + setup.RootKey}}
	for _, table := range []string{"projects", "apps", "environments"} {
		t.Run(table, func(t *testing.T) {
			req := handler.Request{Project: setup.Project.ID, App: setup.App.ID, Environment: setup.Environment.ID}
			live := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			require.Equal(t, http.StatusOK, live.Status, "%s", live.RawBody)
			id := map[string]string{"projects": setup.Project.ID, "apps": setup.App.ID, "environments": setup.Environment.ID}[table]
			_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE "+table+" SET deleted_at = 1 WHERE id = ?", id)
			require.NoError(t, err)
			t.Cleanup(func() {
				_, err := h.DB.RW().ExecContext(context.Background(), "UPDATE "+table+" SET deleted_at = NULL WHERE id = ?", id)
				require.NoError(t, err)
			})
			for _, target := range []handler.Request{req, {Project: setup.Project.Slug, App: setup.App.Slug, Environment: setup.Environment.Slug}} {
				res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, target)
				require.Equal(t, http.StatusNotFound, res.Status, "%s", res.RawBody)
			}
		})
	}
}

func TestGetEnvironmentNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	t.Run("unknown environment id returns 404", func(t *testing.T) {
		req := handler.Request{Project: uid.New(uid.ProjectPrefix), App: uid.New(uid.AppPrefix), Environment: uid.New(uid.EnvironmentPrefix)}
		rootKey := h.CreateRootKey(workspace.ID, rbac.U(urn.New().Workspace(workspace.ID).Project(req.Project).App(req.App).Environment(req.Environment), permissions.Read).Value)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("environment in another workspace returns 404", func(t *testing.T) {
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
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: otherWorkspace.ID,
			ProjectID:   otherProject.ID,
			Name:        "Theirs",
			Slug:        otherAppSlug,
		})
		otherEnvironment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: otherWorkspace.ID,
			ProjectID:   otherProject.ID,
			AppID:       otherApp.ID,
			Slug:        "production",
			Kind:        mysqltype.EnvironmentKindProduction,
			Description: "Theirs",
		})

		rootKey := h.CreateRootKey(workspace.ID, rbac.U(urn.New().Workspace(workspace.ID).Project(otherProject.ID).App(otherApp.ID).Environment(otherEnvironment.ID), permissions.Read).Value)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: otherProject.ID, App: otherApp.ID, Environment: otherEnvironment.ID})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace environment, received: %s", res.RawBody)
	})
}
