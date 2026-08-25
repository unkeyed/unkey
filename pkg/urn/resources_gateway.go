package urn

import "fmt"

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

// Policy returns a gateway policy resource path.
//
// Subresource:
//
//	gateway
//	└── policies/{policy_id}
func (g Gateway) Policy(policyID string) GatewayPolicy {
	return GatewayPolicy{workspaceID: g.workspaceID, path: fmt.Sprintf("%s/policies/%s", g.path, policyID)}
}

// Any returns a descendant pattern below this gateway.
func (g Gateway) Any() V1 {
	return V1{
		WorkspaceID: g.workspaceID,
		Resource:    g.path + "/**",
	}
}

// GatewayPolicy is a gateway policy resource path.
type GatewayPolicy struct {
	workspaceID string
	path        string
}

// String returns this gateway policy resource path.
func (p GatewayPolicy) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: p.path}.String()
}
