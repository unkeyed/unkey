package urn

import "fmt"

// workspace builds resource paths inside one workspace.
//
// Hierarchy:
//
//	workspace
//	├── github/apps/{github_app_id}
//	├── rootKeys/{key_id}
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
	return GitHubApp{workspaceID: w.workspaceID, path: fmt.Sprintf("github/apps/%s", githubAppID)}
}

// RootKey returns a root key resource name scoped to its authorized workspace.
func (w workspace) RootKey(keyID string) V1 {
	return V1{WorkspaceID: w.workspaceID, Resource: fmt.Sprintf("rootKeys/%s", keyID)}
}

// Project returns builders for project resource paths.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
func (w workspace) Project(projectID string) Project {
	return Project{workspaceID: w.workspaceID, path: fmt.Sprintf("projects/%s", projectID)}
}
