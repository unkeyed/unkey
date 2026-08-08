package urn

import "fmt"

// Project builds project resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//
// Projects are owned by a workspace.
type Project struct {
	workspaceID string
	projectID   string
}

// String returns this project resource path.
func (p Project) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: fmt.Sprintf("projects/%s", p.projectID)}.String()
}

// App returns builders for app resource paths.
func (p Project) App(appID string) App {
	return App{workspaceID: p.workspaceID, projectID: p.projectID, appID: appID}
}

// Identity returns an identity resource path.
func (p Project) Identity(identityID string) Identity {
	return Identity{workspaceID: p.workspaceID, projectID: p.projectID, identityID: identityID}
}

// Keyspace returns builders for project-owned keyspace resource paths.
func (p Project) Keyspace(keyspaceID string) Keyspace {
	return Keyspace{workspaceID: p.workspaceID, projectID: p.projectID, keyspaceID: keyspaceID}
}

// RatelimitNamespace returns builders for project-owned rate limit namespace resource paths.
func (p Project) RatelimitNamespace(namespaceID string) RatelimitNamespace {
	return RatelimitNamespace{workspaceID: p.workspaceID, projectID: p.projectID, namespaceID: namespaceID}
}

// RBAC returns builders for project-owned role and permission definition paths.
func (p Project) RBAC() RBAC {
	return RBAC{workspaceID: p.workspaceID, projectID: p.projectID}
}

// Any returns a descendant pattern below this project.
func (p Project) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/**", p.projectID),
	}
}
