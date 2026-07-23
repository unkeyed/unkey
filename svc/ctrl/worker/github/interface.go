package github

import "time"

// GitHubClient defines the interface for GitHub API operations.
type GitHubClient interface {
	// GetInstallationToken retrieves an access token for a specific installation.
	GetInstallationToken(installationID int64) (InstallationToken, error)

	// GetScopedInstallationToken mints an installation token restricted to a
	// single repository with the given permissions, e.g. {"contents":"read"}.
	// repo is the "owner/repo" full name. Use for untrusted fork PR builds so an
	// exfiltrated token grants only minimal, single-repo access.
	GetScopedInstallationToken(installationID int64, repo string, permissions map[string]string) (InstallationToken, error)

	// IsRepoPublic reports whether repo ("owner/repo") is publicly accessible via
	// an unauthenticated request. A public repo can be cloned without any token.
	// Returns an error (never a silent false) when the probe is inconclusive,
	// e.g. rate limited.
	IsRepoPublic(repo string) (bool, error)

	// GetInstallationRepo fetches repo ("owner/repo") using the installation's
	// credentials. An installation token only sees the repositories the
	// installation was granted, so a nil result (with a nil error) means the
	// installation cannot access the repo: either it was not granted or it does
	// not exist. Use this both to resolve repo metadata and to confirm access
	// before connecting a repository to an app.
	GetInstallationRepo(installationID int64, repo string) (*RepoInfo, error)

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

// RepoInfo is the subset of a GitHub repository's metadata Unkey needs when
// connecting a repository to an app.
type RepoInfo struct {
	// ID is GitHub's numeric repository id. It is stable across repo renames,
	// so it is what we persist as the connection's repository_id.
	ID int64

	// FullName is the canonical "owner/repo" name at the time of the lookup.
	FullName string

	// DefaultBranch is the repository's default branch on GitHub, used as the
	// tracked branch when the caller does not specify one.
	DefaultBranch string
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
