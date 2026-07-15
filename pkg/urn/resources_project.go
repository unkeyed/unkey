package urn

import "fmt"

// Project builds project resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    ├── apps/{app_id}
//	    ├── identities/{identity_id}
//	    ├── keyspaces/{keyspace_id}
//	    ├── portals/{portal_id}
//	    ├── ratelimits/namespaces/{namespace_id}
//	    └── rbac
//
// Projects are owned by a workspace.
type Project struct {
	workspaceID string
	path        string

	// RBAC builds role and permission resource paths in this project.
	RBAC rbac
}

// String returns this project resource path.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
func (p Project) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}

// App returns builders for app resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── apps/{app_id}
func (p Project) App(appID string) App {
	return App{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/apps/%s", p.path, appID)}
}

// Identity returns an identity resource path.
//
// Subresource:
//
//	projects/{project_id}
//	└── identities/{identity_id}
func (p Project) Identity(identityID string) Identity {
	return Identity{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/identities/%s", p.path, identityID)}
}

// Keyspace returns builders for keyspace resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── keyspaces/{keyspace_id}
func (p Project) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/keyspaces/%s", p.path, keyspaceID)}
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── ratelimits/namespaces/{namespace_id}
func (p Project) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/ratelimits/namespaces/%s", p.path, namespaceID)}
}

// Portal returns builders for portal resource paths.
//
// Subresource:
//
//	projects/{project_id}
//	└── portals/{portal_id}
func (p Project) Portal(portalID string) Portal {
	return Portal{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/portals/%s", p.path, portalID)}
}

// Any returns a descendant pattern below this project.
func (p Project) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/**",
	}
}
