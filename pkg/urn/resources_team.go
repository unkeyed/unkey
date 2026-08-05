package urn

import "fmt"

// team builds team resource paths.
//
// Hierarchy:
//
//	workspace
//	└── team
type team struct {
	workspaceID string
	path        string
}

// Membership returns a team membership resource path.
//
// Subresource:
//
//	workspace
//	└── team
//	    └── memberships/{membership_id}
func (t team) Membership(membershipID string) V1 {
	return V1{
		WorkspaceID: t.workspaceID,
		Resource:    fmt.Sprintf("%s/memberships/%s", t.path, membershipID),
	}
}

// Invitation returns a team invitation resource path.
//
// Subresource:
//
//	workspace
//	└── team
//	    └── invitations/{invitation_id}
func (t team) Invitation(invitationID string) V1 {
	return V1{
		WorkspaceID: t.workspaceID,
		Resource:    fmt.Sprintf("%s/invitations/%s", t.path, invitationID),
	}
}
