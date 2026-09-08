package urn

import "fmt"

// gateway builds gateway resource paths.
//
// The gateway segment has no ID and is not a permission target. It groups the
// gateway resources in one environment.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
//	                ├── logs
//	                └── policies/{policy_id}
type gateway struct {
	workspaceID string
	path        string
}

// Logs returns the gateway log resource path.
//
// Subresource:
//
//	gateway
//	└── logs
func (g gateway) Logs() GatewayLogs {
	return GatewayLogs{workspaceID: g.workspaceID, path: g.path + "/logs"}
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

// GatewayLogs builds gateway log resource paths.
//
// Hierarchy:
//
//	workspace
//	└── projects/{project_id}
//	    └── apps/{app_id}
//	        └── environments/{environment_id}
//	            └── gateway
//	                └── logs
type GatewayLogs struct {
	workspaceID string
	path        string
}

// String returns this gateway log resource path.
func (g GatewayLogs) String() string {
	return V1{WorkspaceID: g.workspaceID, Resource: g.path}.String()
}
