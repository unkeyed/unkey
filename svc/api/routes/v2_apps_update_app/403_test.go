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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_update_app"
)

func TestUpdateAppForbidden(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

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
		Name:          "Forbidden App",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard app permission", permissions: []string{"app.*.update_app"}, shouldPass: true},
		{name: "specific app permission", permissions: []string{fmt.Sprintf("app.%s.update_app", app.ID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "app.*.update_app"}, shouldPass: true},
		{name: "project scoped update does not match", permissions: []string{fmt.Sprintf("project.%s.update_app", project.ID)}, shouldPass: false},
		{name: "wrong action", permissions: []string{"app.*.read_app"}, shouldPass: false},
		{name: "create does not match update", permissions: []string{"app.*.create_app"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.create_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			newName := "Updated"
			req := handler.Request{Project: project.ID, App: app.ID, Name: &newName}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
			} else {
				require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
			}
		})
	}
}
