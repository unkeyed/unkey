package github

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Noop is a no-op implementation of GitHubClient that returns errors for all operations.
// Use this when GitHub credentials are not configured.
type Noop struct{}

// Ensure Noop implements GitHubClient
var _ GitHubClient = (*Noop)(nil)

// NewNoop creates a new no-op GitHub client.
func NewNoop() *Noop {
	return &Noop{}
}

var errNotConfigured = fault.New("GitHub client not configured: GitHub App credentials were not provided at startup")

// GetInstallationToken returns an error indicating GitHub is not configured.
func (n *Noop) GetInstallationToken(_ int64) (InstallationToken, error) {
	return InstallationToken{}, errNotConfigured
}

// GetBranchHeadCommit returns an error indicating GitHub is not configured.
func (n *Noop) GetBranchHeadCommit(_ int64, _ string, _ string) (CommitInfo, error) {
	return CommitInfo{}, errNotConfigured
}

// CreateDeployment returns an error indicating GitHub is not configured.
func (n *Noop) CreateDeployment(_ int64, _ string, _ string, _ string, _ string, _ bool) (int64, error) {
	return 0, errNotConfigured
}

// CreateDeploymentStatus returns an error indicating GitHub is not configured.
func (n *Noop) CreateDeploymentStatus(_ int64, _ string, _ int64, _ string, _ string, _ string, _ string) error {
	return errNotConfigured
}

// IsCollaborator returns an error indicating GitHub is not configured.
func (n *Noop) IsCollaborator(_ int64, _ string, _ string) (bool, error) {
	return false, errNotConfigured
}

// CreateCheckRun returns an error indicating GitHub is not configured.
func (n *Noop) CreateCheckRun(_ int64, _ string, _ string, _ string, _ string, _ string, _ string, _ string, _ string) (int64, error) {
	return 0, errNotConfigured
}

// UpdateCheckRun returns an error indicating GitHub is not configured.
func (n *Noop) UpdateCheckRun(_ int64, _ string, _ int64, _ string, _ string, _ string, _ string) error {
	return errNotConfigured
}

// ListCheckRunsForRef returns an error indicating GitHub is not configured.
func (n *Noop) ListCheckRunsForRef(_ int64, _ string, _ string, _ string) ([]CheckRun, error) {
	return nil, errNotConfigured
}

// GetBranchHeadCommitPublic retrieves the HEAD commit using the public GitHub
// API without authentication. Works for public repositories even when GitHub
// App credentials are not configured.
func (n *Noop) GetBranchHeadCommitPublic(repo string, branch string) (CommitInfo, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, url.PathEscape(branch))

	commit, err := request[ghCommitResponse](httpClient, http.MethodGet, apiURL, githubHeaders(""), nil, http.StatusOK)
	if err != nil {
		return CommitInfo{}, err
	}

	return CommitInfoFromRaw(
		commit.SHA,
		commit.Commit.Message,
		commit.Author.Login,
		commit.Author.AvatarURL,
		commit.Commit.Author.Date,
	), nil
}
