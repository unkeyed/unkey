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

func TestDeleteAppForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:         h.DB,
		CtrlClient: &testutil.MockAppClient{},
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name        string
		permissions func(projectID, appID string) []string
		shouldPass  bool
	}{
		{name: "wildcard app permission", permissions: func(_, _ string) []string { return []string{"app.*.delete_app"} }, shouldPass: true},
		{name: "specific app permission", permissions: func(_, a string) []string { return []string{fmt.Sprintf("app.%s.delete_app", a)} }, shouldPass: true},
		{name: "wildcard and more", permissions: func(_, _ string) []string { return []string{"some.other.permission", "app.*.delete_app"} }, shouldPass: true},
		{name: "project scoped delete does not match", permissions: func(p, _ string) []string { return []string{fmt.Sprintf("project.%s.delete_app", p)} }, shouldPass: false},
		{name: "wrong action", permissions: func(_, _ string) []string { return []string{"app.*.read_app"} }, shouldPass: false},
		{name: "create does not match delete", permissions: func(_, _ string) []string { return []string{"app.*.create_app"} }, shouldPass: false},
		{name: "unrelated permission", permissions: func(_, _ string) []string { return []string{"api.*.create_api"} }, shouldPass: false},
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
				ID:            uid.New(uid.AppPrefix),
				WorkspaceID:   workspace.ID,
				ProjectID:     project.ID,
				Name:          "Forbidden Test",
				Slug:          appSlug,
				DefaultBranch: "main",
			})

			rootKey := h.CreateRootKey(workspace.ID, tc.permissions(project.ID, app.ID)...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.ID, App: app.ID})
			if tc.shouldPass {
				require.Equal(t, 202, res.Status, "expected 202 for %v, got: %s", tc.name, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.name, res.RawBody)
			}
		})
	}
}
