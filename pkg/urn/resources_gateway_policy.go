package urn

// GatewayPolicy builds gateway policy resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
//	                └── policies/{policy_id}
type GatewayPolicy struct {
	workspaceID string
	path        string
}

// String returns this gateway policy resource path.
func (g GatewayPolicy) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
}
