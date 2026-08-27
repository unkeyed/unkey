package permissions

import "github.com/unkeyed/unkey/pkg/urn"

// ReadGitHubApp authorizes reading a GitHub app installation.
type ReadGitHubApp struct{}

func (ReadGitHubApp) ActionFor(urn.GitHubApp) {}
func (ReadGitHubApp) String() string          { return "read_github_app" }

// WriteGitHubApp authorizes creating or updating a GitHub app installation.
type WriteGitHubApp struct{}

func (WriteGitHubApp) ActionFor(urn.GitHubApp) {}
func (WriteGitHubApp) String() string          { return "write_github_app" }

// DeleteGitHubApp authorizes deleting a GitHub app installation.
type DeleteGitHubApp struct{}

func (DeleteGitHubApp) ActionFor(urn.GitHubApp) {}
func (DeleteGitHubApp) String() string          { return "delete_github_app" }
