package urn

import "fmt"

// rbac builds RBAC resource paths.
//
// Hierarchy:
//
//	workspace
//	└── rbac
type rbac struct {
	workspaceID string
	path        string
}

// Role returns an RBAC role resource path.
//
// Subresource:
//
//	rbac
//	└── roles/{role_id}
func (r rbac) Role(roleID string) V1 {
	return V1{
		WorkspaceID: r.workspaceID,
		Resource:    fmt.Sprintf("%s/roles/%s", r.path, roleID),
	}
}

// Permission returns an RBAC permission resource path.
//
// Subresource:
//
//	rbac
//	└── permissions/{permission_id}
func (r rbac) Permission(permissionID string) V1 {
	return V1{
		WorkspaceID: r.workspaceID,
		Resource:    fmt.Sprintf("%s/permissions/%s", r.path, permissionID),
	}
}

// projectRBAC builds project-scoped RBAC resource paths.
//
// The rbac segment has no ID and is not a permission target. It groups project
// roles and permission definitions.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
//	        ├── roles/{role_id}
//	        └── permissions/{permission_id}
type projectRBAC struct {
	workspaceID string
	path        string
}

// Role returns a project-scoped RBAC role resource path.
func (r projectRBAC) Role(roleID string) Role {
	return Role{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/roles/%s", r.path, roleID)}
}

// Permission returns a project-scoped RBAC permission resource path.
func (r projectRBAC) Permission(permissionID string) Permission {
	return Permission{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/permissions/%s", r.path, permissionID)}
}

// Role builds RBAC role resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
//	        └── roles/{role_id}
type Role struct {
	workspaceID string
	path        string
}

// String returns this RBAC role resource path.
func (r Role) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}.String()
}

// Permission builds RBAC permission resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
//	        └── permissions/{permission_id}
type Permission struct {
	workspaceID string
	path        string
}

// String returns this RBAC permission resource path.
func (p Permission) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}
