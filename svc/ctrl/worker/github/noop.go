package github

import (
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

// CreateCommitStatus returns an error indicating GitHub is not configured.
func (n *Noop) CreateCommitStatus(_ int64, _ string, _ string, _ string, _ string, _ string, _ string) error {
	return errNotConfigured
}

// ListCommitFiles returns an error indicating GitHub is not configured.
func (n *Noop) ListCommitFiles(_ int64, _ string, _ string) ([]string, error) {
	return nil, errNotConfigured
}

// FindPullRequestForBranch returns an error indicating GitHub is not configured.
func (n *Noop) FindPullRequestForBranch(_ int64, _ string, _ string) (int, error) {
	return 0, errNotConfigured
}

// CreateIssueComment returns an error indicating GitHub is not configured.
func (n *Noop) CreateIssueComment(_ int64, _ string, _ int, _ string) (int64, error) {
	return 0, errNotConfigured
}

// UpdateIssueComment returns an error indicating GitHub is not configured.
func (n *Noop) UpdateIssueComment(_ int64, _ string, _ int64, _ string) error {
	return errNotConfigured
}

// FindBotComment returns an error indicating GitHub is not configured.
func (n *Noop) FindBotComment(_ int64, _ string, _ int, _ string) (int64, string, error) {
	return 0, "", errNotConfigured
}

// GetCommitBySHA returns an error indicating GitHub is not configured.
func (n *Noop) GetCommitBySHA(_ int64, _ string, _ string) (CommitInfo, error) {
	return CommitInfo{}, errNotConfigured
}
