package urn

import "fmt"

const portalPathFormat = "projects/%s/portals/%s"

var portalPattern = compileResourcePattern(portalPathFormat)

// Portal builds portal resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── portals/{portal_id}
//	        └── sessions/{session_id}
type Portal struct {
	WorkspaceID string
	ProjectID   string
	PortalID    string
}

// String returns the complete URN for this portal.
func (p Portal) String() string {
	return V1{
		WorkspaceID: p.WorkspaceID,
		Resource:    fmt.Sprintf(portalPathFormat, p.ProjectID, p.PortalID),
	}.String()
}

// ParsePortal parses:
//
//	unkey:v1:ws_123:projects/proj_123/portals/portal_123
//
// into:
//
//	Portal{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		PortalID:    "portal_123",
//	}
//
// Resource ID positions may contain "*".
func ParsePortal(urn string) (Portal, error) {
	matches := portalPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Portal{}, fmt.Errorf("%w: resource does not match portal", ErrInvalidResourceName)
	}
	return Portal{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		PortalID:    matches[3],
	}, nil
}

// Session returns a portal session resource path.
//
// Subresource:
//
//	portals/{portal_id}
//	└── sessions/{session_id}
func (p Portal) Session(sessionID string) PortalSession {
	return PortalSession{
		WorkspaceID: p.WorkspaceID,
		ProjectID:   p.ProjectID,
		PortalID:    p.PortalID,
		SessionID:   sessionID,
	}
}
