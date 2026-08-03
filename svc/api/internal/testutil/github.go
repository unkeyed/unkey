package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	github "github.com/unkeyed/unkey/pkg/github"
)

// FakeGitHub is a github.GitHubClient stub returning a fixed repository, for
// tests that exercise the app repo-connection paths. Embedding *github.Noop
// supplies the rest of the interface (which those paths never call).
type FakeGitHub struct {
	*github.Noop

	// Repo is returned from GetInstallationRepo. Accessible controls whether the
	// stub reports the repository as reachable by the installation.
	Repo       github.RepoInfo
	Accessible bool
}

// GetInstallationRepo returns the configured repository, or nil when Accessible
// is false.
func (f FakeGitHub) GetInstallationRepo(_ int64, _ string) (*github.RepoInfo, error) {
	if !f.Accessible {
		return nil, nil
	}
	repo := f.Repo
	return &repo, nil
}

// SeedGitHubInstallation inserts a github_app_installations row for a workspace.
// The API never writes this table (the dashboard callback does), so tests seed
// it directly to make an installation resolvable.
func (h *Harness) SeedGitHubInstallation(t *testing.T, workspaceID string, installationID int64) {
	t.Helper()
	_, err := h.DB.RW().ExecContext(
		context.Background(),
		"INSERT INTO github_app_installations (workspace_id, installation_id, created_at) VALUES (?, ?, ?)",
		workspaceID, installationID, time.Now().UnixMilli(),
	)
	require.NoError(t, err)
}
