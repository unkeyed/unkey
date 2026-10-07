package urn

import "fmt"

const rbacPathFormat = "projects/%s/rbac"

var rbacPattern = compileResourcePattern(rbacPathFormat)

// RBAC builds RBAC resource paths.
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
type RBAC struct {
	WorkspaceID string
	ProjectID   string
}

// String returns the complete URN for this RBAC resource.
func (r RBAC) String() string {
	return V1{
		WorkspaceID: r.WorkspaceID,
		Resource:    fmt.Sprintf(rbacPathFormat, r.ProjectID),
	}.String()
}

// ParseRBAC parses:
//
//	unkey:v1:ws_123:projects/proj_123/rbac
//
// into:
//
//	RBAC{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//	}
//
// Resource ID positions may contain "*".
func ParseRBAC(urn string) (RBAC, error) {
	matches := rbacPattern.FindStringSubmatch(urn)
	if matches == nil {
		return RBAC{}, fmt.Errorf("%w: resource does not match RBAC", ErrInvalidResourceName)
	}
	return RBAC{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
	}, nil
}

// Role returns an RBAC role resource path.
//
// Subresource:
//
//	rbac
//	└── roles/{role_id}
func (r RBAC) Role(roleID string) Role {
	return Role{
		WorkspaceID: r.WorkspaceID,
		ProjectID:   r.ProjectID,
		RoleID:      roleID,
	}
}

// Permission returns an RBAC permission resource path.
//
// Subresource:
//
//	rbac
//	└── permissions/{permission_id}
func (r RBAC) Permission(permissionID string) Permission {
	return Permission{
		WorkspaceID:  r.WorkspaceID,
		ProjectID:    r.ProjectID,
		PermissionID: permissionID,
	}
}
