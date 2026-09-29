package urn

// workspace builds resource paths inside one workspace.
//
// Hierarchy:
//
//	workspace
//	├── github/apps/{github_app_id}
//	└── projects/{project_id}
type workspace struct {
	workspaceID string
}

// GitHubApp returns a GitHub app resource path.
//
// Subresource:
//
//	workspace
//	└── github/apps/{github_app_id}
func (w workspace) GitHubApp(githubAppID string) GitHubApp {
	return GitHubApp{
		WorkspaceID: w.workspaceID,
		GitHubAppID: githubAppID,
	}
}

// Project returns builders for project resource paths.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
func (w workspace) Project(projectID string) Project {
	return Project{
		WorkspaceID: w.workspaceID,
		ProjectID:   projectID,
	}
}
