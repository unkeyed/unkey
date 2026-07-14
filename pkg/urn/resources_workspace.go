package urn

import "fmt"

// workspace builds resource paths inside one workspace.
//
// Hierarchy:
//
//	workspace
//	├── team
//	├── billing
//	├── keyspaces/{keyspace_id}
//	├── identities/{identity_id}
//	├── ratelimits/namespaces/{namespace_id}
//	├── rbac
//	├── projects/{project_id}
//	└── portals/{portal_id}
//
// Children with their own descendants return another typed builder. Leaf
// resources return V1 directly.
type workspace struct {
	workspaceID string

	// Team builds team resource paths in this workspace.
	Team team

	// RBAC builds RBAC resource paths in this workspace.
	RBAC rbac
}

// Billing returns builders for billing resource paths.
//
// Subresource:
//
//	workspace
//	└── billing
func (w workspace) Billing() billing {
	return billing{workspaceID: w.workspaceID, path: "billing"}
}

// Keyspace returns builders for keyspace resource paths.
//
// Subresource:
//
//	workspace
//	└── keyspaces/{keyspace_id}
func (w workspace) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{workspaceID: w.workspaceID, path: fmt.Sprintf("keyspaces/%s", keyspaceID)}
}

// Identity returns an identity resource path.
func (w workspace) Identity(identityID string) V1 {
	return w.v1(fmt.Sprintf("identities/%s", identityID))
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
//
// Subresource:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
func (w workspace) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{workspaceID: w.workspaceID, path: fmt.Sprintf("ratelimits/namespaces/%s", namespaceID)}
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

// Portal returns builders for portal resource paths.
//
// Subresource:
//
//	workspace
//	└── portals/{portal_id}
func (w workspace) Portal(portalID string) Portal {
	return Portal{workspaceID: w.workspaceID, path: fmt.Sprintf("portals/%s", portalID)}
}

// v1 wraps a resource path in a [V1] for this workspace.
func (w workspace) v1(path string) V1 {
	return V1{
		WorkspaceID: w.workspaceID,
		Resource:    path,
	}
}
