package urn

import "fmt"

const gatewayPolicyPathFormat = "projects/%s/apps/%s/environments/%s/gateway/policies/%s"

var gatewayPolicyPattern = compileResourcePattern(gatewayPolicyPathFormat)

// GatewayPolicy identifies one gateway policy.
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
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
	PolicyID      string
}

// String returns the complete URN for this gateway policy.
//
// Subresource:
//
//	gateway
//	└── policies/{policy_id}
func (g GatewayPolicy) String() string {
	return V1{
		WorkspaceID: g.WorkspaceID,
		Resource:    fmt.Sprintf(gatewayPolicyPathFormat, g.ProjectID, g.AppID, g.EnvironmentID, g.PolicyID),
	}.String()
}

// ParseGatewayPolicy parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/policies/policy_123
//
// into:
//
//	GatewayPolicy{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//		PolicyID:      "policy_123",
//	}
//
// Resource ID positions may contain "*".
func ParseGatewayPolicy(urn string) (GatewayPolicy, error) {
	matches := gatewayPolicyPattern.FindStringSubmatch(urn)
	if matches == nil {
		return GatewayPolicy{}, fmt.Errorf("%w: resource does not match gateway policy", ErrInvalidResourceName)
	}
	return GatewayPolicy{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
		PolicyID:      matches[5],
	}, nil
}
