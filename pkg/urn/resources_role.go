package urn

import "fmt"

const rolePathFormat = "projects/%s/rbac/roles/%s"

var rolePattern = compileResourcePattern(rolePathFormat)

// Role builds RBAC role resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
//	        └── roles/{role_id}
type Role struct {
	WorkspaceID string
	ProjectID   string
	RoleID      string
}

// String returns the complete URN for this RBAC role.
func (r Role) String() string {
	return V1{
		WorkspaceID: r.WorkspaceID,
		Resource:    fmt.Sprintf(rolePathFormat, r.ProjectID, r.RoleID),
	}.String()
}

// ParseRole parses:
//
//	unkey:v1:ws_123:projects/proj_123/rbac/roles/role_123
//
// into:
//
//	Role{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		RoleID:      "role_123",
//	}
//
// Resource ID positions may contain "*".
func ParseRole(urn string) (Role, error) {
	matches := rolePattern.FindStringSubmatch(urn)
	if matches == nil {
		return Role{}, fmt.Errorf("%w: resource does not match role", ErrInvalidResourceName)
	}
	return Role{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		RoleID:      matches[3],
	}, nil
}
