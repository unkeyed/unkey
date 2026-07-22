package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_github_connection"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		DB:                  h.DB,
		GitHubAppName:       "unkey-test",
		GitHubPrivateKeyPEM: "test-private-key-pem",
	}
}

// stateFrom extracts and JSON-decodes the signed `state` query param from the
// returned install URL so tests can assert what got embedded.
func stateFrom(t *testing.T, installURL string) map[string]any {
	t.Helper()
	parsed, err := url.Parse(installURL)
	require.NoError(t, err)
	require.Equal(t, "github.com", parsed.Host)
	require.Equal(t, "/apps/unkey-test/installations/new", parsed.Path)

	raw := parsed.Query().Get("state")
	require.NotEmpty(t, raw, "install url must carry a state param")

	var state map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &state))
	return state
}

func TestCreateGithubConnectionSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
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

	for _, tc := range []struct {
		name    string
		project string
		app     string
	}{
		{name: "by ids", project: project.ID, app: app.ID},
		{name: "by slugs", project: projectSlug, app: appSlug},
		{name: "project id and app slug", project: project.ID, app: appSlug},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: tc.project,
				App:     tc.app,
			})
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Greater(t, res.Body.Data.ExpiresAt, time.Now().UnixMilli(), "expiresAt must be in the future")

			state := stateFrom(t, res.Body.Data.Url)
			// State carries the resolved ids, not the caller-supplied slugs.
			require.Equal(t, app.ID, state["appId"])
			require.Equal(t, project.ID, state["projectId"])
			require.Equal(t, workspace.ID, state["workspaceId"])
			require.Equal(t, "api", state["source"])
			require.Equal(t, "settings", state["returnTo"])
			require.NotEmpty(t, state["sig"])
			require.NotEmpty(t, state["nonce"])
			// No repository was requested, so it must be absent from the state.
			require.NotContains(t, state, "repository")
		})
	}
}

func TestCreateGithubConnectionNormalizesRepository(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
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

	for _, input := range []string{
		"unkeyed/unkey",
		"https://github.com/unkeyed/unkey",
		"github.com/unkeyed/unkey.git",
	} {
		t.Run(input, func(t *testing.T) {
			repo := input
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project:    project.ID,
				App:        app.ID,
				Repository: &repo,
			})
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			state := stateFrom(t, res.Body.Data.Url)
			require.Equal(t, "unkeyed/unkey", state["repository"], "repository must be normalized to owner/name")
		})
	}
}
