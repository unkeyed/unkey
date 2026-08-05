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
//	    ├── ratelimits/namespaces/{namespace_id}
//	    └── rbac
//
// Projects are owned by a workspace.
type Project struct {
	workspaceID string
	path        string
}

// String returns this project resource path.
func (p Project) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}

// App returns builders for app resource paths.
func (p Project) App(appID string) App {
	return App{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/apps/%s", p.path, appID)}
}

// Identity returns an identity resource path.
func (p Project) Identity(identityID string) Identity {
	return Identity{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/identities/%s", p.path, identityID)}
}

// Keyspace returns builders for project-owned keyspace resource paths.
func (p Project) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/keyspaces/%s", p.path, keyspaceID)}
}

// RatelimitNamespace returns builders for project-owned rate limit namespace resource paths.
func (p Project) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/ratelimits/namespaces/%s", p.path, namespaceID)}
}

// RBAC returns builders for project-owned role and permission definition paths.
func (p Project) RBAC() RBAC {
	return RBAC{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/rbac", p.path)}
}

// Any returns a descendant pattern below this project.
func (p Project) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("%s/**", p.path),
	}
}
