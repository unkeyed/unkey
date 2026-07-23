package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspaces_install_github"
)

// Installing the GitHub App is workspace-wide, gated by workspace.*.install_github.
// Any key missing that permission is rejected with a 403.
func TestInstallGithubAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "workspace install permission", permissions: []string{"workspace.*.install_github"}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "workspace.*.install_github"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "app-level connect_github", permissions: []string{"app.*.connect_github"}, shouldPass: false},
		{name: "no permissions", permissions: []string{}, shouldPass: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, tc.permissions...)
			headers := http.Header{
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}
			res := callInstall(h, route, headers)
			if tc.shouldPass {
				require.Equal(t, 200, res.Status, "expected 200 for %v, got: %s", tc.permissions, res.RawBody)
				return
			}
			require.Equal(t, http.StatusForbidden, res.Status, "expected 403 for %v, got: %s", tc.permissions, res.RawBody)
		})
	}
}

func TestInstallGithubNotConfigured(t *testing.T) {
	h := testutil.NewHarness(t)

	// No GitHub app name or signer configured.
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "workspace.*.install_github")
	headers := http.Header{
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := callInstall(h, route, headers)
	require.Equal(t, http.StatusInternalServerError, res.Status, "expected 500 when github is not configured, got: %s", res.RawBody)
}
