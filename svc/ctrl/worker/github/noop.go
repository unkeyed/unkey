package github

import (
	"encoding/json"
	"fmt"
	"io"
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

// GetInstallationToken returns an error indicating GitHub is not configured.
func (n *Noop) GetInstallationToken(installationID int64) (InstallationToken, error) {
	return InstallationToken{}, fault.New("GitHub client not configured: GitHub App credentials were not provided at startup")
}

// GetBranchHeadCommit returns an error indicating GitHub is not configured.
func (n *Noop) GetBranchHeadCommit(installationID int64, repo string, branch string) (CommitInfo, error) {
	return CommitInfo{}, fault.New("GitHub client not configured: GitHub App credentials were not provided at startup")
}

// GetBranchHeadCommitPublic retrieves the HEAD commit using the public GitHub
// API without authentication. Works for public repositories even when GitHub
// App credentials are not configured.
func (n *Noop) GetBranchHeadCommitPublic(repo string, branch string) (CommitInfo, error) {
	httpClient := &http.Client{Timeout: 30 * time.Second}

	requestURL := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, url.PathEscape(branch))

	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return CommitInfo{}, fault.Wrap(err, fault.Internal("failed to create request"))
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := httpClient.Do(req)
	if err != nil {
		return CommitInfo{}, fault.Wrap(err, fault.Internal("failed to fetch branch head commit"))
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return CommitInfo{}, fault.New(
			"failed to fetch branch head commit (public)",
			fault.Internal(fmt.Sprintf("status %d: %s", resp.StatusCode, string(body))),
		)
	}

	var commit ghCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return CommitInfo{}, fault.Wrap(err, fault.Internal("failed to decode commit response"))
	}

	return CommitInfoFromRaw(
		commit.SHA,
		commit.Commit.Message,
		commit.Author.Login,
		commit.Author.AvatarURL,
		commit.Commit.Author.Date,
	), nil
}
