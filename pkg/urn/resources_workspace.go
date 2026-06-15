package urn

import "fmt"

// workspace builds resource paths inside one workspace.
//
// Hierarchy:
//
//	workspace
//	├── settings
//	├── memberships/{membership_id}
//	├── invitations/{invitation_id}
//	├── billing
//	├── keyspaces/{keyspace_id}
//	├── identities/{identity_id}
//	├── ratelimits/namespaces/{namespace_id}
//	├── rbac/roles/{role_id}
//	├── rbac/permissions/{permission_id}
//	├── projects/{project_id}
//	└── portals/{portal_id}
//
// Children with their own descendants return another typed builder. Leaf
// resources return V1 directly.
type workspace struct {
	workspaceID string
}

// Settings returns the workspace settings resource path.
func (w workspace) Settings() V1 {
	return w.v1("settings")
}

// Membership returns a workspace membership resource path.
func (w workspace) Membership(membershipID string) V1 {
	return w.v1(fmt.Sprintf("memberships/%s", membershipID))
}

// Invitation returns a workspace invitation resource path.
func (w workspace) Invitation(invitationID string) V1 {
	return w.v1(fmt.Sprintf("invitations/%s", invitationID))
}

// Billing returns builders for billing resource paths.
//
// Subresource:
//
//	workspace
//	└── billing
func (w workspace) Billing() billing {
	return billing{workspaceID: w.workspaceID, path: "billing"}
}

// Keyspace returns builders for keyspace resource paths.
//
// Subresource:
//
//	workspace
//	└── keyspaces/{keyspace_id}
func (w workspace) Keyspace(keyspaceID string) keyspace {
	return keyspace{workspaceID: w.workspaceID, path: fmt.Sprintf("keyspaces/%s", keyspaceID)}
}

// Identity returns an identity resource path.
func (w workspace) Identity(identityID string) V1 {
	return w.v1(fmt.Sprintf("identities/%s", identityID))
}

// RatelimitNamespace returns builders for rate limit namespace resource paths.
//
// Subresource:
//
//	workspace
//	└── ratelimits/namespaces/{namespace_id}
func (w workspace) RatelimitNamespace(namespaceID string) ratelimitNamespace {
	return ratelimitNamespace{workspaceID: w.workspaceID, path: fmt.Sprintf("ratelimits/namespaces/%s", namespaceID)}
}

// Role returns an RBAC role resource path.
func (w workspace) Role(roleID string) V1 {
	return w.v1(fmt.Sprintf("rbac/roles/%s", roleID))
}

// Permission returns an RBAC permission resource path.
func (w workspace) Permission(permissionID string) V1 {
	return w.v1(fmt.Sprintf("rbac/permissions/%s", permissionID))
}

// Project returns builders for project resource paths.
//
// Subresource:
//
//	workspace
//	└── projects/{project_id}
func (w workspace) Project(projectID string) project {
	return project{workspaceID: w.workspaceID, path: fmt.Sprintf("projects/%s", projectID)}
}

// Portal returns builders for portal resource paths.
//
// Subresource:
//
//	workspace
//	└── portals/{portal_id}
func (w workspace) Portal(portalID string) portal {
	return portal{workspaceID: w.workspaceID, path: fmt.Sprintf("portals/%s", portalID)}
}

// v1 wraps a resource path in a [V1] for this workspace.
func (w workspace) v1(path string) V1 {
	return V1{
		WorkspaceID: w.workspaceID,
		Resource:    path,
	}
}
