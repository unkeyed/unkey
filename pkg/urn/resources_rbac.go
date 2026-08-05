package urn

import "fmt"

// RBAC builds role and permission definition resource paths beneath a parent.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
type RBAC struct {
	workspaceID string
	path        string
}

// Role is an RBAC role resource path.
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

// String returns this role resource path.
func (r Role) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}.String()
}

// PermissionDefinition is an RBAC permission definition resource path.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
//	        └── permissions/{permission_id}
type PermissionDefinition struct {
	workspaceID string
	path        string
}

// String returns this permission definition resource path.
func (p PermissionDefinition) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}

// Role returns an RBAC role resource path.
func (r RBAC) Role(roleID string) Role {
	return Role{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/roles/%s", r.path, roleID)}
}

// Permission returns an RBAC permission resource path.
func (r RBAC) Permission(permissionID string) PermissionDefinition {
	return PermissionDefinition{workspaceID: r.workspaceID, path: fmt.Sprintf("%s/permissions/%s", r.path, permissionID)}
}
