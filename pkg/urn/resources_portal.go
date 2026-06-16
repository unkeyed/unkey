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
	path        string
}

// String returns this portal resource path.
//
// Subresource:
//
//	workspace
//	└── portals/{portal_id}
func (p Portal) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}

// SessionToken returns a portal session token resource path.
//
// Subresource:
//
//	portals/{portal_id}
//	└── session_tokens/{portal_session_token_id}
func (p Portal) SessionToken(tokenID string) V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("%s/session_tokens/%s", p.path, tokenID),
	}
}

// Session returns a portal session resource path.
//
// Subresource:
//
//	portals/{portal_id}
//	└── sessions/{session_id}
func (p Portal) Session(sessionID string) V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    fmt.Sprintf("%s/sessions/%s", p.path, sessionID),
	}
}

// Branding returns a portal branding resource path.
//
// Subresource:
//
//	portals/{portal_id}
//	└── branding
func (p Portal) Branding() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/branding",
	}
}

// Any returns a descendant pattern below this portal.
func (p Portal) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/**",
	}
}
