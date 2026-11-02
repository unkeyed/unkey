package urn

import "fmt"

const portalDomainPathFormat = "projects/%s/portals/%s/domains/%s"

var portalDomainPattern = compileResourcePattern(portalDomainPathFormat)

// PortalDomain builds paths for the hostnames a portal is served on.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── portals/{portal_id}
//	        └── domains/{domain_id}
type PortalDomain struct {
	WorkspaceID string
	ProjectID   string
	PortalID    string
	DomainID    string
}

// String returns the complete URN for this portal domain.
func (d PortalDomain) String() string {
	return V1{
		WorkspaceID: d.WorkspaceID,
		Resource:    fmt.Sprintf(portalDomainPathFormat, d.ProjectID, d.PortalID, d.DomainID),
	}.String()
}

// ParsePortalDomain parses:
//
//	unkey:v1:ws_123:projects/proj_123/portals/portal_123/domains/pdom_123
//
// into:
//
//	PortalDomain{
//		WorkspaceID: "ws_123",
//		ProjectID:   "proj_123",
//		PortalID:    "portal_123",
//		DomainID:    "pdom_123",
//	}
//
// Resource ID positions may contain "*".
func ParsePortalDomain(urn string) (PortalDomain, error) {
	matches := portalDomainPattern.FindStringSubmatch(urn)
	if matches == nil {
		return PortalDomain{}, fmt.Errorf("%w: resource does not match portal domain", ErrInvalidResourceName)
	}
	return PortalDomain{
		WorkspaceID: matches[1],
		ProjectID:   matches[2],
		PortalID:    matches[3],
		DomainID:    matches[4],
	}, nil
}
