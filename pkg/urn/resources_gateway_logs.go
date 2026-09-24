package urn

import "fmt"

const gatewayLogsPathFormat = "projects/%s/apps/%s/environments/%s/gateway/logs"

var gatewayLogsPattern = compileResourcePattern(gatewayLogsPathFormat)

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
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
}

// String returns the complete URN for these gateway logs.
func (g GatewayLogs) String() string {
	return V1{
		WorkspaceID: g.WorkspaceID,
		Resource:    fmt.Sprintf(gatewayLogsPathFormat, g.ProjectID, g.AppID, g.EnvironmentID),
	}.String()
}

// ParseGatewayLogs parses:
//
//	unkey:v1:ws_123:projects/proj_123/apps/app_123/environments/env_123/gateway/logs
//
// into:
//
//	GatewayLogs{
//		WorkspaceID:   "ws_123",
//		ProjectID:     "proj_123",
//		AppID:         "app_123",
//		EnvironmentID: "env_123",
//	}
//
// Resource ID positions may contain "*".
func ParseGatewayLogs(urn string) (GatewayLogs, error) {
	matches := gatewayLogsPattern.FindStringSubmatch(urn)
	if matches == nil {
		return GatewayLogs{}, fmt.Errorf("%w: resource does not match gateway logs", ErrInvalidResourceName)
	}
	return GatewayLogs{
		WorkspaceID:   matches[1],
		ProjectID:     matches[2],
		AppID:         matches[3],
		EnvironmentID: matches[4],
	}, nil
}
