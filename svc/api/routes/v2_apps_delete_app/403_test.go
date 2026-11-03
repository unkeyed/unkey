package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_delete_app"
)

func TestDeleteAppForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	restate, deletes := newRecordingRestate(t)

	route := &handler.Handler{
		DB:      h.DB,
		Restate: restate,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name        string
		permissions func(projectID, appID string) []string
		shouldPass  bool
	}{
		{name: "exact app URN", permissions: func(p, a string) []string {
			return []string{rbac.U(urn.New().Workspace(workspace.ID).Project(p).App(a), permissions.Delete).Value}
		}, shouldPass: true},
		{name: "wildcard app URN", permissions: func(_, _ string) []string {
			return []string{rbac.U(urn.New().Workspace(workspace.ID).Project("*").App("*"), permissions.Delete).Value}
		}, shouldPass: true},
		{name: "wrong workspace URN", permissions: func(p, a string) []string {
			return []string{rbac.U(urn.New().Workspace(uid.New(uid.WorkspacePrefix)).Project(p).App(a), permissions.Delete).Value}
		}, shouldPass: false},
		{name: "wrong project URN", permissions: func(_, a string) []string {
			return []string{rbac.U(urn.New().Workspace(workspace.ID).Project(uid.New(uid.ProjectPrefix)).App(a), permissions.Delete).Value}
		}, shouldPass: false},
		{name: "wrong app URN", permissions: func(p, _ string) []string {
			return []string{rbac.U(urn.New().Workspace(workspace.ID).Project(p).App(uid.New(uid.AppPrefix)), permissions.Delete).Value}
		}, shouldPass: false},
		{name: "read URN does not allow deletion", permissions: func(p, a string) []string {
			return []string{rbac.U(urn.New().Workspace(workspace.ID).Project(p).App(a), permissions.Read).Value}
		}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
			project := h.CreateProject(seed.CreateProjectRequest{
				ID:          uid.New(uid.ProjectPrefix),
				WorkspaceID: workspace.ID,
				Name:        "Forbidden Test",
				Slug:        projectSlug,
			})
			appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
			app := h.CreateApp(seed.CreateAppRequest{
				ID:          uid.New(uid.AppPrefix),
				WorkspaceID: workspace.ID,
				ProjectID:   project.ID,
				Name:        "Forbidden Test",
				Slug:        appSlug,
			})

			rootKey := h.CreateRootKey(workspace.ID, tc.permissions(project.ID, app.ID)...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.ID, App: app.ID})
			if tc.shouldPass {
				require.Equal(t, 202, res.Status, "expected 202 for %v, got: %s", tc.name, res.RawBody)
				testutil.Receive(t, deletes, 10*time.Second)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.name, res.RawBody)
				testutil.RequireNoReceive(t, deletes, time.Second)
			}
		})
	}
}
