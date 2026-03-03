package github

import "time"

// GitHubClient defines the interface for GitHub API operations.
type GitHubClient interface {
	// GetInstallationToken retrieves an access token for a specific installation.
	GetInstallationToken(installationID int64) (InstallationToken, error)

	// GetBranchHeadCommit retrieves the HEAD commit of a branch from a GitHub
	// repository using the given installation's credentials.
	GetBranchHeadCommit(installationID int64, repo string, branch string) (CommitInfo, error)
}

// CommitInfo holds metadata about a single Git commit retrieved from the GitHub API.
type CommitInfo struct {
	SHA             string
	Message         string
	AuthorHandle    string
	AuthorAvatarURL string
	Timestamp       time.Time
}

// InstallationToken represents a GitHub installation access token. The token
// provides repository access for a specific App installation and expires after
// 1 hour.
type InstallationToken struct {
	// Token is the installation access token for API requests.
	Token string `json:"token"`

	// ExpiresAt indicates when the token expires, typically 1 hour from issuance.
	ExpiresAt time.Time `json:"expires_at"`
}
