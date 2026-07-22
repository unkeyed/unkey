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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_github_connection"
)

// A same-workspace key without the connect_github action must not be able to
// tell a real app apart from a missing one: authorization failure is masked as
// a 404 that never echoes the app ID.
func TestCreateGithubConnectionAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})
	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "wildcard app permission", permissions: []string{"app.*.connect_github"}, shouldPass: true},
		{name: "specific app permission", permissions: []string{fmt.Sprintf("app.%s.connect_github", app.ID)}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "app.*.connect_github"}, shouldPass: true},
		{name: "wrong action read", permissions: []string{"app.*.read_app"}, shouldPass: false},
		{name: "wrong action update", permissions: []string{"app.*.update_app"}, shouldPass: false},
		{name: "unrelated permission", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: project.ID,
				App:     app.ID,
			})
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for %v, got: %s", tc.permissions, res.RawBody)
			require.NotContains(t, res.RawBody, app.ID, "response must not leak the app ID for %v", tc.permissions)
		})
	}
}

func TestCreateGithubConnectionNotConfigured(t *testing.T) {
	h := testutil.NewHarness(t)

	// No GitHub app name or signer configured.
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.connect_github")
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
	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project: project.ID,
		App:     app.ID,
	})
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500 when github is not configured, got: %s", res.RawBody)
}
