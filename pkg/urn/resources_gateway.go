package urn

import "fmt"

const gatewayPathFormat = "projects/%s/apps/%s/environments/%s/gateway"

var gatewayPattern = compileResourcePattern(gatewayPathFormat)

// Gateway builds gateway resource paths.
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
type Gateway struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
}

// String returns the complete URN for this gateway.
func (g Gateway) String() string {
	return V1{
		WorkspaceID: g.WorkspaceID,
		Resource:    fmt.Sprintf(gatewayPathFormat, g.ProjectID, g.AppID, g.EnvironmentID),
	}.String()
}

// ParseGateway parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway
//
// into:
//
//	Gateway{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//	}
//
// Resource ID positions may contain "*".
func ParseGateway(urn string) (Gateway, error) {
	matches := gatewayPattern.FindStringSubmatch(urn)
	if matches == nil {
		return Gateway{}, fmt.Errorf("%w: resource does not match gateway", ErrInvalidResourceName)
	}
	return Gateway{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
	}, nil
}

// Logs returns the gateway log resource path.
//
// Subresource:
//
//	gateway
//	└── logs
func (g Gateway) Logs() GatewayLogs {
	return GatewayLogs{
		WorkspaceID:   g.WorkspaceID,
		ProjectID:     g.ProjectID,
		AppID:         g.AppID,
		EnvironmentID: g.EnvironmentID,
	}
}

// Policy returns a gateway policy resource path.
//
// Subresource:
//
//	gateway
//	└── policies/{policy_id}
func (g Gateway) Policy(policyID string) GatewayPolicy {
	return GatewayPolicy{
		WorkspaceID:   g.WorkspaceID,
		ProjectID:     g.ProjectID,
		AppID:         g.AppID,
		EnvironmentID: g.EnvironmentID,
		PolicyID:      policyID,
	}
}
