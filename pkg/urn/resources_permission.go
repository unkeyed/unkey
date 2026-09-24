package urn

import "fmt"

const permissionPathFormat = "projects/%s/rbac/permissions/%s"

var permissionPattern = compileResourcePattern(permissionPathFormat)

// Permission builds RBAC permission resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── rbac
//	        └── permissions/{permission_id}
type Permission struct {
	WorkspaceID  string
	ProjectID    string
	PermissionID string
}

// String returns the complete URN for this RBAC permission.
func (p Permission) String() string {
	return V1{
		WorkspaceID: p.WorkspaceID,
		Resource:    fmt.Sprintf(permissionPathFormat, p.ProjectID, p.PermissionID),
	}.String()
}

// ParsePermission parses:
//
//	unkey:v1:ws_123:projects/proj_123/rbac/permissions/permission_123
//
// into:
//
//	Permission{
//		WorkspaceID:  "ws_123",
//		ProjectID:    "proj_123",
//		PermissionID: "permission_123",
//	}
//
// Resource ID positions may contain "*".
func ParsePermission(urn string) (Permission, error) {
	matches := permissionPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Permission{}, fmt.Errorf("%w: resource does not match permission", ErrInvalidResourceName)
	}
	return Permission{
		WorkspaceID:  matches[1],
		ProjectID:    matches[2],
		PermissionID: matches[3],
	}, nil
}
