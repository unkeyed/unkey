package urn

// GitHubApp builds GitHub app resource paths.
//
// Hierarchy:
//
//	workspace
//	└── github/apps/{github_app_id}
//
// A GitHub app is the only public resource that does not belong to a project.
type GitHubApp struct {
	workspaceID string
	path        string
}

// String returns this GitHub app resource path.
func (g GitHubApp) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
}
