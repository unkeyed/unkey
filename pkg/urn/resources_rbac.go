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
