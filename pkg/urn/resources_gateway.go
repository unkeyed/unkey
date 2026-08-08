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
type Gateway struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

// String returns this gateway resource path.
func (g Gateway) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/gateway", g.projectID, g.appID, g.environmentID)}.String()
}

// GatewayPolicy is a gateway policy resource path.
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
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	policyID      string
}

// String returns this gateway policy resource path.
func (p GatewayPolicy) String() string {
	return V1{WorkspaceID: p.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/gateway/policies/%s", p.projectID, p.appID, p.environmentID, p.policyID)}.String()
}

// GatewayRoute is a gateway route resource path.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
//	                └── routes/{route_id}
type GatewayRoute struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	routeID       string
}

// String returns this gateway route resource path.
func (r GatewayRoute) String() string {
	return V1{WorkspaceID: r.workspaceID, Resource: fmt.Sprintf("projects/%s/apps/%s/environments/%s/gateway/routes/%s", r.projectID, r.appID, r.environmentID, r.routeID)}.String()
}

// Policy returns a policy beneath this gateway.
func (g Gateway) Policy(policyID string) GatewayPolicy {
	return GatewayPolicy{workspaceID: g.workspaceID, projectID: g.projectID, appID: g.appID, environmentID: g.environmentID, policyID: policyID}
}

// Route returns a route beneath this gateway.
func (g Gateway) Route(routeID string) GatewayRoute {
	return GatewayRoute{workspaceID: g.workspaceID, projectID: g.projectID, appID: g.appID, environmentID: g.environmentID, routeID: routeID}
}
