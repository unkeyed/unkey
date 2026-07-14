package urn

import "fmt"

// Project builds project resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    ├── apps/{app_id}
//	    └── identities/{identity_id}
//
// Projects are owned by a workspace.
type Project struct {
	workspaceID string
	path        string
}

// ProjectIdentity is an identity resource path beneath a project.
type ProjectIdentity struct {
	workspaceID string
	path        string
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
func (p Project) Identity(identityID string) ProjectIdentity {
	return ProjectIdentity{workspaceID: p.workspaceID, path: fmt.Sprintf("%s/identities/%s", p.path, identityID)}
}

// String returns this project identity resource path.
func (i ProjectIdentity) String() string {
	return V1{WorkspaceID: i.workspaceID, Resource: i.path}.String()
}

// Any returns a descendant pattern below this project.
func (p Project) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/**",
	}
}
