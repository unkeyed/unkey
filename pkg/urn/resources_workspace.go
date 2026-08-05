package urn

import "fmt"

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

	// RBAC builds RBAC resource paths in this workspace.
	RBAC RBAC
}

// String returns the workspace resource path.
func (w Workspace) String() string {
	return V1{WorkspaceID: w.workspaceID, Resource: "workspace"}.String()
}

// Billing returns builders for billing resource paths.
func (w Workspace) Billing() billing {
	return billing{workspaceID: w.workspaceID, path: "billing"}
}

// Keyspace returns builders for keyspace resource paths.
func (w Workspace) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{workspaceID: w.workspaceID, path: fmt.Sprintf("keyspaces/%s", keyspaceID)}
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
func (w Workspace) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{workspaceID: w.workspaceID, path: fmt.Sprintf("ratelimits/namespaces/%s", namespaceID)}
}

// Project returns builders for project resource paths.
func (w Workspace) Project(projectID string) Project {
	return Project{workspaceID: w.workspaceID, path: fmt.Sprintf("projects/%s", projectID)}
}

// Portal returns builders for portal resource paths.
func (w Workspace) Portal(portalID string) Portal {
	return Portal{workspaceID: w.workspaceID, path: fmt.Sprintf("portals/%s", portalID)}
}
