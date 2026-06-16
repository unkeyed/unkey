package urn

import "fmt"

// portal builds portal resource paths.
//
// Hierarchy:
//
//	workspace
//	└── portals/{portal_id}
type portal struct {
	workspaceID string
	path        string
}

// SessionToken returns a portal session token resource path.
//
// Subresource:
//
//	portals/{portal_id}
//	└── session_tokens/{token_id}
func (p portal) SessionToken(tokenID string) V1 {
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
func (p portal) Session(sessionID string) V1 {
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
func (p portal) Branding() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/branding",
	}
}

// Any returns a descendant pattern below this portal.
func (p portal) Any() V1 {
	return V1{
		WorkspaceID: p.workspaceID,
		Resource:    p.path + "/**",
	}
}
