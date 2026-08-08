package urn

// Workspace builds resource paths inside one workspace. It is also the
// concrete target for operations that affect the workspace itself.
//
// Hierarchy:
//
//	workspace
//
// Children with their own descendants return another typed builder. Leaf
// resources return V1 directly.
type Workspace struct {
	workspaceID string

	// Team builds team resource paths in this workspace.
	Team team
}

// String returns the workspace resource path.
func (w Workspace) String() string {
	return V1{WorkspaceID: w.workspaceID, Resource: "workspace"}.String()
}

// Project returns builders for project resource paths.
func (w Workspace) Project(projectID string) Project {
	return Project{workspaceID: w.workspaceID, projectID: projectID}
}

// Portal returns builders for portal resource paths.
func (w Workspace) Portal(portalID string) Portal {
	return Portal{workspaceID: w.workspaceID, portalID: portalID}
}
