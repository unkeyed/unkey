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

// Session returns a portal session resource path.
func (p Portal) Session(sessionID string) V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("portals/%s/sessions/%s", p.portalID, sessionID),
	}
}

// Any returns a descendant pattern below this portal.
func (p Portal) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("portals/%s/**", p.portalID),
	}
}
