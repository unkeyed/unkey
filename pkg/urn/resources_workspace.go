package urn

import "fmt"

// workspace builds resource paths inside one workspace.
//
// Hierarchy:
//
//	workspace
//	├── github/apps/{github_app_id}
//	├── rootKeys/{key_id}
//	├── limits
//	├── usage
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

// RootKey returns a root key resource name scoped to its authorized workspace.
func (w workspace) RootKey(keyID string) V1 {
	return V1{WorkspaceID: w.workspaceID, Resource: fmt.Sprintf("rootKeys/%s", keyID)}
}

// Limits returns the workspace limits resource name
func (w workspace) Limits() V1 {
	return V1{WorkspaceID: w.workspaceID, Resource: "limits"}
}

// Usage returns the workspace usage resource name
func (w workspace) Usage() V1 {
	return V1{WorkspaceID: w.workspaceID, Resource: "usage"}
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
