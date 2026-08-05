package urn

import "fmt"

// Workspace builds resource paths inside one workspace. It is also the
// concrete target for operations that affect the workspace itself.
//
// Hierarchy:
//
//	workspace
//	├── team
//	├── billing
//	├── keyspaces/{keyspace_id}
//	├── ratelimits/namespaces/{namespace_id}
//	├── rbac
//	├── projects/{project_id}
//	└── portals/{portal_id}
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
//
// Subresource:
//
//	workspace
//	└── billing
func (w Workspace) Billing() billing {
	return billing{workspaceID: w.workspaceID, path: "billing"}
}

// Keyspace returns builders for keyspace resource paths.
//
// Subresource:
//
//	workspace
//	└── keyspaces/{keyspace_id}
func (w Workspace) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{workspaceID: w.workspaceID, path: fmt.Sprintf("keyspaces/%s", keyspaceID)}
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
//
// Subresource:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
func (w Workspace) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{workspaceID: w.workspaceID, path: fmt.Sprintf("ratelimits/namespaces/%s", namespaceID)}
}

// Project returns builders for project resource paths.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
func (w Workspace) Project(projectID string) Project {
	return Project{workspaceID: w.workspaceID, path: fmt.Sprintf("projects/%s", projectID)}
}

// Portal returns builders for portal resource paths.
//
// Subresource:
//
//	workspace
//	└── portals/{portal_id}
func (w Workspace) Portal(portalID string) Portal {
	return Portal{workspaceID: w.workspaceID, path: fmt.Sprintf("portals/%s", portalID)}
}
