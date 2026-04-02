package github

import "time"

// GitHubClient defines the interface for GitHub API operations.
type GitHubClient interface {
	// GetInstallationToken retrieves an access token for a specific installation.
	GetInstallationToken(installationID int64) (InstallationToken, error)

	// GetBranchHeadCommit retrieves the HEAD commit of a branch from a GitHub
	// repository using the given installation's credentials.
	GetBranchHeadCommit(installationID int64, repo string, branch string) (CommitInfo, error)

	// GetBranchHeadCommitPublic retrieves the HEAD commit of a branch using
	// the public GitHub API without authentication. Only works for public repos.
	GetBranchHeadCommitPublic(repo string, branch string) (CommitInfo, error)

	// GetCommitBySHA retrieves commit metadata for a specific SHA using
	// the given installation's credentials.
	GetCommitBySHA(installationID int64, repo string, sha string) (CommitInfo, error)

	// CreateDeployment creates a GitHub Deployment on a commit SHA. Returns the
	// GitHub deployment ID for subsequent status updates.
	CreateDeployment(installationID int64, repo string, ref string, environment string, description string, isProduction bool) (int64, error)

	// CreateDeploymentStatus updates the status of a GitHub Deployment.
	// state must be one of: pending, in_progress, success, failure, error.
	// logURL is shown as "View logs" on GitHub; environmentURL as "View deployment".
	CreateDeploymentStatus(installationID int64, repo string, deploymentID int64, state string, environmentURL string, logURL string, description string) error

	// IsCollaborator checks whether a GitHub user is a collaborator on a repository.
	IsCollaborator(installationID int64, repo string, username string) (bool, error)

	// CreateCommitStatus creates a commit status on a SHA. The "Details" link
	// in the PR goes directly to targetURL. context is the label shown (e.g.
	// "Unkey Deploy Authorization"). state: pending|success|error|failure.
	CreateCommitStatus(installationID int64, repo string, sha string, state string, targetURL string, description string, context string) error

	// ListCommitFiles returns the list of filenames changed in a specific commit.
	ListCommitFiles(installationID int64, repo string, sha string) ([]string, error)

	// FindPullRequestForBranch returns the PR number for the given branch,
	// or 0 if no open PR exists.
	FindPullRequestForBranch(installationID int64, repo string, branch string) (int, error)

	// CreateIssueComment posts a new comment on a PR/issue and returns the comment ID.
	CreateIssueComment(installationID int64, repo string, prNumber int, body string) (int64, error)

	// UpdateIssueComment updates an existing PR/issue comment by ID.
	UpdateIssueComment(installationID int64, repo string, commentID int64, body string) error

	// FindBotComment searches PR comments for one containing the given marker string.
	// Returns the comment ID and body, or 0 if not found.
	FindBotComment(installationID int64, repo string, prNumber int, marker string) (int64, string, error)
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
