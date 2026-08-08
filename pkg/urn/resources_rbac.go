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
	projectID   string
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
	projectID   string
	roleID      string
}

// String returns this role resource path.
func (r Role) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("projects/%s/rbac/roles/%s", r.projectID, r.roleID)}.String()
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
	workspaceID  string
	projectID    string
	permissionID string
}

// String returns this permission definition resource path.
func (p PermissionDefinition) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: fmt.Sprintf("projects/%s/rbac/permissions/%s", p.projectID, p.permissionID)}.String()
}

// Role returns an RBAC role resource path.
func (r RBAC) Role(roleID string) Role {
	return Role{workspaceID: r.workspaceID, projectID: r.projectID, roleID: roleID}
}

// Permission returns an RBAC permission resource path.
func (r RBAC) Permission(permissionID string) PermissionDefinition {
	return PermissionDefinition{workspaceID: r.workspaceID, projectID: r.projectID, permissionID: permissionID}
}
