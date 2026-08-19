package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_github_install_app"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	return &handler.Handler{
		DB:                  h.DB,
		GitHubAppName:       "unkey-test",
		GitHubPrivateKeyPEM: "test-private-key-pem",
	}
}

// callInstall hits the route with an empty body, exactly like `curl -X POST`
// with no `-d`. The endpoint takes no input, so there is nothing to send.
func callInstall(
	h *testutil.Harness,
	route *handler.Handler,
	headers http.Header,
) testutil.TestResponse[handler.Response] {
	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header = headers
	return testutil.CallRaw[handler.Response](h, req)
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

func TestInstallGithubSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "workspace.*.install_github")
	headers := http.Header{
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := callInstall(h, route, headers)
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Meta.RequestId)
	require.Greater(t, res.Body.Data.ExpiresAt, time.Now().UnixMilli(), "expiresAt must be in the future")

	state := stateFrom(t, res.Body.Data.Url)
	require.Equal(t, workspace.ID, state["workspaceId"])
	require.Equal(t, "api", state["flow"])
	require.NotEmpty(t, state["sig"])
	require.NotEmpty(t, state["nonce"])
	// The api flow is workspace-wide: no app, project, repository, returnTo, or
	// user is bound; the callback lands on the caller's workspace settings.
	require.NotContains(t, state, "appId")
	require.NotContains(t, state, "projectId")
	require.NotContains(t, state, "repository")
	require.NotContains(t, state, "returnTo")
	require.NotContains(t, state, "userId")
}
