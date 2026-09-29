package urn

import "fmt"

const githubAppPathFormat = "github/apps/%s"

var githubAppPattern = compileResourcePattern(githubAppPathFormat)

// GitHubApp builds GitHub app resource paths.
//
// Hierarchy:
//
//	workspace
//	└── github/apps/{github_app_id}
//
// A GitHub app is the only public resource that does not belong to a project.
type GitHubApp struct {
	WorkspaceID string
	GitHubAppID string
}

// String returns the complete URN for this GitHub app.
func (g GitHubApp) String() string {
	return V1{
		WorkspaceID: g.WorkspaceID,
		Resource:    fmt.Sprintf(githubAppPathFormat, g.GitHubAppID),
	}.String()
}

// ParseGitHubApp parses:
//
//	unkey:v1:ws_123:github/apps/github_123
//
// into:
//
//	GitHubApp{
//		WorkspaceID: "ws_123",
//		GitHubAppID: "github_123",
//	}
//
// Resource ID positions may contain "*".
func ParseGitHubApp(urn string) (GitHubApp, error) {
	matches := githubAppPattern.FindStringSubmatch(urn)
	if matches == nil {
		return GitHubApp{}, fmt.Errorf("%w: resource does not match GitHub app", ErrInvalidResourceName)
	}
	return GitHubApp{
		WorkspaceID: matches[1],
		GitHubAppID: matches[2],
	}, nil
}
