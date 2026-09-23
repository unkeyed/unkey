package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_github_install_app"
)

// TestInstallGithubAuthorization guarantees that installation requires either
// the canonical collection write grant or its legacy equivalent.
func TestInstallGithubAuthorization(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	otherWorkspace := h.CreateWorkspace()

	testCases := []struct {
		name        string
		permissions []string
		shouldPass  bool
	}{
		{name: "workspace install permission", permissions: []string{"workspace.*.install_github"}, shouldPass: true},
		{name: "permission and more", permissions: []string{"some.other.permission", "workspace.*.install_github"}, shouldPass: true},
		{name: "wrong action", permissions: []string{"api.*.read_api"}, shouldPass: false},
		{name: "app-level permission", permissions: []string{"app.*.read_app"}, shouldPass: false},
		{
			name:        "canonical grant for another workspace",
			permissions: []string{fmt.Sprintf("unkey:v1:%s:github/apps/*#write", otherWorkspace.ID)},
			shouldPass:  false,
		},
		{
			name:        "canonical collection grant with wrong action",
			permissions: []string{fmt.Sprintf("unkey:v1:%s:github/apps/*#read", workspace.ID)},
			shouldPass:  false,
		},
		{
			name:        "canonical concrete app grant",
			permissions: []string{fmt.Sprintf("unkey:v1:%s:github/apps/github_app_123#write", workspace.ID)},
			shouldPass:  false,
		},
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
