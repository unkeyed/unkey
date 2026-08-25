package urn

import "fmt"

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
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	domainID      string
}

// String returns this domain resource path.
func (d Domain) String() string {
	return V1{WorkspaceID: d.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/domains/%s", d.projectID, d.appID, d.environmentID, d.domainID)}.String()
}

// Any returns a descendant pattern below this domain.
func (d Domain) Any() V1 {
	return V1{
		WorkspaceID: d.workspaceID,
		Resource:    fmt.Sprintf("projects/%s/apps/%s/environments/%s/domains/%s/**", d.projectID, d.appID, d.environmentID, d.domainID),
	}
}
