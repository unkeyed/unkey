package urn

import "fmt"

const portalSessionPathFormat = "projects/%s/portals/%s/sessions/%s"

var portalSessionPattern = compileResourcePattern(portalSessionPathFormat)

// PortalSession identifies one session issued by a portal.
type PortalSession struct {
	WorkspaceID string
	ProjectID   string
	PortalID    string
	SessionID   string
}

// String returns the complete URN for this portal session.
func (p PortalSession) String() string {
	return V1{
		WorkspaceID: p.WorkspaceID,
		Resource:    fmt.Sprintf(portalSessionPathFormat, p.ProjectID, p.PortalID, p.SessionID),
	}.String()
}

// ParsePortalSession parses:
//
//	unkey:v1:ws_123:projects/proj_123/portals/portal_123/sessions/session_123
//
// into:
//
//	PortalSession{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		PortalID:    "portal_123",
//		SessionID:   "session_123",
//	}
//
// Resource ID positions may contain "*".
func ParsePortalSession(urn string) (PortalSession, error) {
	matches := portalSessionPattern.FindStringSubmatch(urn)
	if matches == nil {
		return PortalSession{}, fmt.Errorf("%w: resource does not match portal session", ErrInvalidResourceName)
	}
	return PortalSession{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		PortalID:    matches[3],
		SessionID:   matches[4],
	}, nil
}
