package urn

import "fmt"

// Gateway builds resources beneath an environment's gateway namespace.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
//	                ├── policies/{policy_id}
//	                └── routes/{route_id}
type Gateway struct {
	workspaceID string
	path        string
}

// String returns this gateway resource path.
func (g Gateway) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
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

// GatewayRoute is a gateway route resource path.
type GatewayRoute struct {
	workspaceID string
	path        string
}

// String returns this gateway route resource path.
func (r GatewayRoute) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: r.path}.String()
}

// Policy returns a policy beneath this gateway.
func (g Gateway) Policy(policyID string) GatewayPolicy {
	return GatewayPolicy{workspaceID: g.workspaceID, path: fmt.Sprintf("%s/policies/%s", g.path, policyID)}
}

// Route returns a route beneath this gateway.
func (g Gateway) Route(routeID string) GatewayRoute {
	return GatewayRoute{workspaceID: g.workspaceID, path: fmt.Sprintf("%s/routes/%s", g.path, routeID)}
}
