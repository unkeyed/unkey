package github

import "github.com/unkeyed/unkey/pkg/fault"

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
