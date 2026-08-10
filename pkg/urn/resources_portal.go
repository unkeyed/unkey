package urn

import "fmt"

// Portal builds portal resource paths.
//
// Hierarchy:
//
//	workspace
//	└── portals/{portal_id}
type Portal struct {
	workspaceID string
	portalID    string
}

// String returns this portal resource path.
func (p Portal) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: fmt.Sprintf("portals/%s", p.portalID)}.String()
}

// SessionToken returns a portal session token resource path.
func (p Portal) SessionToken(tokenID string) V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("portals/%s/session_tokens/%s", p.portalID, tokenID),
	}
}

// Session returns a portal session resource path.
func (p Portal) Session(sessionID string) V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("portals/%s/sessions/%s", p.portalID, sessionID),
	}
}

// Branding returns a portal branding resource path.
func (p Portal) Branding() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("portals/%s/branding", p.portalID),
	}
}

// Any returns a descendant pattern below this portal.
func (p Portal) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("portals/%s/**", p.portalID),
	}
}
