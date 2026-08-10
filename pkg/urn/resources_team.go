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
}

// Membership returns a team membership resource path.
func (t team) Membership(membershipID string) V1 {
	return V1{
		WorkspaceID: t.workspaceID,
		Resource:    fmt.Sprintf("team/memberships/%s", membershipID),
	}
}

// Invitation returns a team invitation resource path.
func (t team) Invitation(invitationID string) V1 {
	return V1{
		WorkspaceID: t.workspaceID,
		Resource:    fmt.Sprintf("team/invitations/%s", invitationID),
	}
}
