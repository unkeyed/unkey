package urn

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
	workspaceID string
	path        string
}

// String returns this domain resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── domains/{domain_id}
func (d Domain) String() string {
	return V1{WorkspaceID: d.workspaceID, Resource: d.path}.String()
}

// Any returns a descendant pattern below this domain.
func (d Domain) Any() V1 {
	return V1{
		WorkspaceID: d.workspaceID,
		Resource:    d.path + "/**",
	}
}
