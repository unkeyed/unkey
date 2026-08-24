package urn

// Gateway builds gateway resource paths.
//
// Gateway is a namespace, not an addressable object: it groups everything the
// gateway owns in an environment, such as policies and routes, so a grant can
// cover the gateway without also covering the environment's build settings,
// variables, or deployments.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
type Gateway struct {
	workspaceID string
	path        string
}

// String returns this gateway resource path.
//
// Subresource:
//
//	environments/{environment_id}
//	└── gateway
func (g Gateway) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
}

// Any returns a descendant pattern below this gateway.
func (g Gateway) Any() V1 {
	return V1{
		WorkspaceID: g.workspaceID,
		Resource:    g.path + "/**",
	}
}
