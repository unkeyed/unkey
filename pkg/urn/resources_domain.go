package urn

import "fmt"

const domainPathFormat = "projects/%s/apps/%s/environments/%s/domains/%s"

var domainPattern = compileResourcePattern(domainPathFormat)

// Domain builds custom domain resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── domains/{domain_id}
type Domain struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	DomainID      string
}

// String returns the complete URN for this domain.
//
// Subresource:
//
//	environments/{environment_id}
//	└── domains/{domain_id}
func (d Domain) String() string {
	return V1{
		WorkspaceID: d.WorkspaceID,
		Resource:    fmt.Sprintf(domainPathFormat, d.ProjectID, d.AppID, d.EnvironmentID, d.DomainID),
	}.String()
}

// ParseDomain parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/domains/domain_123
//
// into:
//
//	Domain{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//		DomainID:      "domain_123",
//	}
//
// Resource ID positions may contain "*".
func ParseDomain(urn string) (Domain, error) {
	matches := domainPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Domain{}, fmt.Errorf("%w: resource does not match domain", ErrInvalidResourceName)
	}
	return Domain{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
		DomainID:      matches[5],
	}, nil
}
