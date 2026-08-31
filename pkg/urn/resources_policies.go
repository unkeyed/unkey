package urn

// GatewayPolicy builds gateway policy resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway/policies/{policy_id}
type GatewayPolicy struct {
	workspaceID string
	path        string
}

// String returns this gateway policy resource path.
//
// Subresource:
//
//	gateway
//	└── policies/{policy_id}
func (g GatewayPolicy) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
}

// Any returns a descendant pattern below this gateway policy.
func (g GatewayPolicy) Any() V1 {
	return V1{
		WorkspaceID: g.workspaceID,
		Resource:    g.path + "/**",
	}
}
