package urn

import "fmt"

// gateway builds gateway resource paths.
//
// The gateway segment carries no id and is never granted on its own. It groups
// what the gateway owns in an environment, such as policies and routes, so a
// grant can cover them without also covering the environment's build settings,
// variables, or deployments.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
type gateway struct {
	workspaceID string
	path        string
}

// Policy returns a gateway policy resource path.
//
// Subresource:
//
//	gateway
//	└── policies/{policy_id}
func (g gateway) Policy(policyID string) GatewayPolicy {
	return GatewayPolicy{workspaceID: g.workspaceID, path: fmt.Sprintf("%s/policies/%s", g.path, policyID)}
}
